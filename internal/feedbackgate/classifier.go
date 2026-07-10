package feedbackgate

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// embeddedPrompt is the checked-in classifier system prompt, embedded so it
// ships with the binary. The canonical copy is prompts/feedback_gate_classifier.md;
// this package-local mirror is kept byte-identical (a test enforces the version
// line is present). Callers that want the prompt at runtime use DefaultPrompt().
//
//go:embed feedback_gate_classifier.md
var embeddedPrompt string

// DefaultPrompt returns the embedded classifier system prompt.
func DefaultPrompt() string { return embeddedPrompt }

// classifierResult is the strict JSON shape the Haiku classifier must return.
// Any deviation (parse error, schema miss) makes the gate fail closed → file.
type classifierResult struct {
	IsGenuineFeedback      bool   `json:"is_genuine_feedback"`
	IsPromptInjection      bool   `json:"is_prompt_injection"`
	BestCategory           string `json:"best_category"`
	EstimatedDispatchValue string `json:"estimated_dispatch_value"`
	Reasoning              string `json:"reasoning"`
}

// classifierResponseSchema is the JSON Schema handed to the provider so it
// enforces structured output natively (ai.Request.ResponseSchema).
const classifierResponseSchema = `{
  "type": "object",
  "properties": {
    "is_genuine_feedback": {"type": "boolean"},
    "is_prompt_injection": {"type": "boolean"},
    "best_category": {"type": "string", "enum": ["bug","feature","docs","limitation","spam"]},
    "estimated_dispatch_value": {"type": "string", "enum": ["high","medium","low","none"]},
    "reasoning": {"type": "string"}
  },
  "required": ["is_genuine_feedback","is_prompt_injection","best_category","estimated_dispatch_value"]
}`

// Classifier is the last-resort JSON pre-screen. It wraps an injected
// ai.Provider (fake in tests, real Anthropic in the cloud wiring) plus the
// checked-in prompt and an optional daily budget store (M5).
type Classifier struct {
	provider ai.Provider
	prompt   string // system prompt text (see prompts/feedback_gate_classifier.md)
	budget   *Budget
}

// NewClassifier builds a classifier from a provider and prompt text. budget
// may be nil (no spend cap). The provider is required; a nil provider makes
// the classifier a no-op that files everything (fail closed).
func NewClassifier(provider ai.Provider, prompt string, budget *Budget) *Classifier {
	return &Classifier{provider: provider, prompt: prompt, budget: budget}
}

// HasProvider reports whether the classifier has a live ai.Provider. A false
// result means the classifier is in its fail-closed posture (nil provider →
// heuristic-flagged messages are filed, never dispatched). The cloud wiring
// uses this only to name the classifier stage in the startup log; it does NOT
// gate any decision (applyClassifier checks the provider directly).
func (c *Classifier) HasProvider() bool {
	return c != nil && c.provider != nil
}

// PromptHash returns a short content hash of the prompt, used as the version
// field for replay (the prompt file carries the same hash in its front matter).
func (c *Classifier) PromptHash() string {
	sum := sha256.Sum256([]byte(c.prompt))
	return hex.EncodeToString(sum[:8])
}

// shouldClassify reports whether a message that passed rules + cooldown should
// still be sent to the classifier. Heuristics (design doc M3): a very long
// code block, a bug with a large body and no snippet, or an auto: message from
// a non-agent sender. agent-* senders are ALWAYS bypassed (adopted decision
// #2) — that check happens in applyClassifier before this is consulted.
func shouldClassify(in Input) bool {
	body := in.Body
	if strings.Count(body, "\n") > 200 {
		return true
	}
	cat := strippedCategory(in.Category)
	if cat == "bug" && len(body) > 4096 && !strings.Contains(strings.ToLower(body), "snippet") {
		return true
	}
	// An auto: message from a non-agent sender is inherently the risky case
	// this gate exists for.
	if hasAutoPrefix(in.Category) && !strings.HasPrefix(in.From, agentSenderPrefix) {
		return true
	}
	return false
}

// applyClassifier runs the M3 stage. Called only when rules + cooldown would
// dispatch AND cfg.Classifier is non-nil. It:
//   - bypasses agent-* senders (never calls the provider),
//   - respects Mode=file-only (classifier disabled),
//   - skips messages no heuristic flagged (they dispatch),
//   - enforces the M5 daily budget (over budget → file),
//   - fails closed on any provider/parse/schema error (→ file).
func applyClassifier(ctx context.Context, in Input, cfg FeedbackGateConfig) (Verdict, error) {
	cl := cfg.Classifier

	// Internal agents already passed coordinator approval — don't double-tax
	// them, and never call the provider for them.
	if strings.HasPrefix(in.From, agentSenderPrefix) {
		return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}, nil
	}

	// Operator kill-switch: file-only mode disables the classifier stage.
	if cfg.Mode == ModeFileOnly {
		return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}, nil
	}

	// Only heuristic-flagged messages reach the LLM.
	if !shouldClassify(in) {
		return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}, nil
	}

	// M5: daily budget. Over budget → file (never dispatch), never a Sonnet
	// flood. Checked BEFORE the provider call so the call itself is skipped.
	if cl.budget != nil {
		ok, err := cl.budget.CheckAndReserve(ctx, cfg.DailyBudgetUSD)
		if err != nil {
			return Verdict{}, err
		}
		if !ok {
			return Verdict{Action: ActionFile, Reason: ReasonBudgetExceeded}, nil
		}
	}

	// A nil provider means we cannot classify — fail closed.
	if cl.provider == nil {
		return Verdict{Action: ActionFile, Reason: ReasonClassifierError}, nil
	}

	req := &ai.Request{
		Model:          cfg.ClassifierModel,
		SystemPrompt:   cl.prompt,
		UserPrompt:     classifierUserPrompt(in),
		ResponseFormat: "json",
		ResponseSchema: classifierResponseSchema,
		MaxTokens:      512,
	}
	resp, err := cl.provider.Generate(ctx, req)
	if err != nil {
		// Provider error is NOT a gate-level error we propagate — a flaky
		// classifier must not open the gate. Fail closed to file.
		return Verdict{Action: ActionFile, Reason: ReasonClassifierError}, nil
	}

	var result classifierResult
	if perr := json.Unmarshal([]byte(strings.TrimSpace(resp.Text)), &result); perr != nil {
		// Malformed JSON → fail closed.
		return Verdict{Action: ActionFile, Reason: ReasonClassifierParseFailed}, nil
	}

	return classifierVerdict(in, result), nil
}

// classifierVerdict maps a parsed classifier result to a Verdict per the M3
// decision matrix. Injection → reject; anything doubtful → file; otherwise
// dispatch.
func classifierVerdict(in Input, r classifierResult) Verdict {
	if r.IsPromptInjection {
		return Verdict{Action: ActionReject, Reason: ReasonClassifierInjection}
	}
	if r.EstimatedDispatchValue == "none" {
		return Verdict{Action: ActionFile, Reason: ReasonClassifierNoValue}
	}
	if !r.IsGenuineFeedback {
		return Verdict{Action: ActionFile, Reason: ReasonClassifierNotGenuine}
	}
	if r.BestCategory != strippedCategory(in.Category) {
		return Verdict{Action: ActionFile, Reason: ReasonClassifierMismatch}
	}
	return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}
}

// classifierUserPrompt renders the message fields into the user turn. Kept
// simple and structured; the classifier returns strict JSON, so the body is
// never echoed into a downstream action.
func classifierUserPrompt(in Input) string {
	return fmt.Sprintf("category: %s\nfrom: %s\ninbox: %s\n\nbody:\n%s",
		strippedCategory(in.Category), in.From, in.Inbox, in.Body)
}
