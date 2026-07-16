package quorum

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// DefaultMaxCostUSD is the per-reviewer budget cap. A design-doc review is a
// few thousand tokens each way; even the priciest reviewer lands in single-
// digit cents. The cap is a guardrail against a runaway (e.g. a reviewer that
// echoes the whole doc back), not an expected-cost estimate.
const DefaultMaxCostUSD = 0.10

// expectedOutputTokens is the pre-flight output estimate for the budget-cap
// PRE-check. The review output is a small structured JSON object; this is a
// generous headroom for reasoning models. Input tokens are estimated from the
// ACTUAL prompt size (see estimateInputTokens) so the cap scales with the doc
// rather than assuming a fixed worst case.
const expectedOutputTokens = 1024

// charsPerToken is a coarse tokens≈chars/4 heuristic for the pre-flight
// estimate only. The POST-check uses the provider's real token counts.
const charsPerToken = 4

// estimateInputTokens estimates prompt tokens from the built prompt + the
// reviewer system prompt, using the chars/4 heuristic with a small floor so a
// trivially short doc still reserves some budget.
func estimateInputTokens(prompt string) int {
	chars := len(prompt) + len(systemPrompt)
	tokens := chars / charsPerToken
	if tokens < 256 {
		tokens = 256
	}
	return tokens
}

// ReviewerOutcome is the full record of one reviewer invocation, including the
// verdict (when present), the cost, and — when the reviewer could not run —
// an explicit absence reason. A missing reviewer is NEVER a silent pass
// (Critical Principle 2): Present=false with a non-empty AbsentReason.
type ReviewerOutcome struct {
	Model   string  `json:"model"`
	CostUSD float64 `json:"cost_usd"`

	// Present is true iff the reviewer ran and returned a valid verdict.
	Present bool `json:"present"`
	// AbsentReason is populated iff Present is false: one of "unreachable",
	// "budget", "auth", "unknown-model", "invalid" (malformed/gate-violating
	// response).
	AbsentReason string `json:"absent_reason,omitempty"`
	// Err carries the underlying error text for an absent reviewer (audit).
	Err string `json:"error,omitempty"`

	// Result is the parsed verdict when Present is true; nil otherwise.
	Result *ReviewResult `json:"result,omitempty"`

	// Landed seeds the catch-rate hook (design doc's "drop a reviewer whose
	// objections never land"): the doc author/evaluator later flips this to
	// true/false once they act (or don't) on the objection. null = not yet
	// adjudicated.
	Landed *bool `json:"landed"`

	// Tier labels which review pass produced this outcome: "" (empty) for the
	// Tier-1 text quorum (preserving the shipped artifact shape byte-for-byte
	// for existing Phase E consumers), "tier2" for an escalated agentic
	// verification. Additive + omitempty — a Tier-1 outcome marshals identically
	// to before.
	Tier string `json:"tier,omitempty"`
}

// TierAgentic is the Tier label for an escalated agentic-verification outcome.
const TierAgentic = "tier2"

// AbsentReason values.
const (
	ReasonUnreachable  = "unreachable"
	ReasonBudget       = "budget"
	ReasonAuth         = "auth"
	ReasonUnknownModel = "unknown-model"
	ReasonInvalid      = "invalid"
)

// RunReviewer runs one reviewer end-to-end with a budget cap. It NEVER returns
// a nil outcome: on any failure it returns an outcome with Present=false and a
// named AbsentReason, so the caller (the orchestrator) can degrade gracefully
// and record the absence. The error return is non-nil only for programmer
// errors (e.g. empty model id), not for provider/auth/budget failures — those
// are data on the outcome.
func RunReviewer(modelID, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome {
	out := &ReviewerOutcome{Model: modelID}
	if maxCostUSD <= 0 {
		maxCostUSD = DefaultMaxCostUSD
	}

	// Resolve provider + auth. A resolve failure is a named absence, not a
	// pass. An unknown model id gets its own semantically correct reason
	// ("unknown-model") rather than being mislabeled as "auth".
	caller, mc, err := ResolveCaller(modelID)
	if err != nil {
		if errors.Is(err, ErrUnknownModel) {
			out.AbsentReason = ReasonUnknownModel
		} else {
			out.AbsentReason = ReasonAuth
		}
		out.Err = err.Error()
		return out
	}
	return runReviewerWith(caller, mc, out, docPath, docBody, maxCostUSD)
}

// runReviewerWith is the resolver-independent core of RunReviewer: given an
// already-built caller + model config, it runs the budget-capped review. Split
// out so unit tests can inject a stub caller + a synthetic ModelConfig without
// touching real providers or auth.
func runReviewerWith(caller JSONCaller, mc *eval_harness.ModelConfig, out *ReviewerOutcome, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome {
	prompt := BuildPrompt(docPath, docBody)

	// PRE-flight budget cap: refuse before spending if the estimate (scaled to
	// the ACTUAL doc size) already exceeds the cap. No silent fallback — a
	// structured refusal, zero spend.
	estCost := estimateCost(mc, estimateInputTokens(prompt), expectedOutputTokens)
	if estCost > maxCostUSD {
		out.AbsentReason = ReasonBudget
		out.Err = fmt.Sprintf("estimated cost $%.4f (doc ~%d input tok) exceeds cap $%.4f (pre-flight refusal, zero spend)", estCost, estimateInputTokens(prompt), maxCostUSD)
		return out
	}

	raw, resp, cerr := caller.CallJSON(systemPrompt, prompt, reviewSchema)
	if cerr != nil {
		out.AbsentReason = ReasonUnreachable
		out.Err = cerr.Error()
		return out
	}

	// POST-flight actual cost from token counts × models.yml pricing.
	out.CostUSD = estimateCost(mc, resp.InputTokens, resp.OutputTokens)

	result, perr := ParseReviewResult(raw)
	if perr != nil {
		out.AbsentReason = ReasonInvalid
		out.Err = perr.Error()
		return out
	}

	out.Present = true
	out.Result = result
	return out
}

// estimateCost computes USD cost from token counts and the model's
// models.yml pricing. Used both for the pre-flight cap and post-flight
// accounting so the two use identical arithmetic.
func estimateCost(mc *eval_harness.ModelConfig, inputTokens, outputTokens int) float64 {
	return float64(inputTokens)/1000.0*mc.Pricing.InputPer1K +
		float64(outputTokens)/1000.0*mc.Pricing.OutputPer1K
}
