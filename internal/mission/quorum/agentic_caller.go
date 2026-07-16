package quorum

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// agenticReviewMaxTokens is unused by the agentic path directly (the executor
// bounds itself via turn/time caps), but the estimateCost pre-check still uses
// the text-tier expectedOutputTokens headroom. Kept as documentation of intent.

// AgenticRun is the minimal, provider-agnostic surface the agentic caller needs
// from the coordinator's executor layer. coordinator.ExecutorProvider.Execute
// satisfies this structurally via the AgenticProvider adapter; tests inject a
// stub. Keeping the surface this small keeps the quorum package free of a
// coordinator import cycle (quorum is imported by the mission-control wiring,
// which owns the coordinator dependency and passes a concrete runner in).
type AgenticRun struct {
	// Output is the agent's final textual output (expected to contain the
	// verdict JSON, same stripCodeFences/ParseReviewResult path as the text tier).
	Output string
	// Success is false when the executor itself errored (unreachable/killed).
	Success bool
	// Err carries the executor error text when Success is false.
	Err string
	// CostUSD is the observed post-hoc cost (= coordinator ExecuteResult.Cost =
	// executor CostUSD). The per-review cap is enforced against THIS value.
	CostUSD float64
}

// AgenticRunner runs a single bounded, read-only agentic review and returns its
// transcript + observed cost. The implementation is expected to:
//   - build a coordinator AnalyzedTask with Kind=="question" (read-only tools),
//   - set opts.Workspace to a read-only worktree,
//   - set opts.Timeout + opts.IdleTimeout for the bounded turn/time cap,
//   - call provider.Execute(ctx, ...),
//   - and map the result into an AgenticRun.
//
// ctx carries cancellation; the caller sets a deadline so a hung run is killed
// and recorded as unreachable (never a silent pass).
type AgenticRunner func(ctx context.Context, systemPrompt, userPrompt string) (*AgenticRun, error)

// agenticCaller adapts an AgenticRunner to the EXISTING JSONCaller interface
// (call.go:35). It is a THIN adapter: it runs the bounded agentic review, then
// hands the raw output back through the same JSON-parse path the text tier
// uses. It records the observed cost so the budget cap can be enforced
// controller-side (against runReviewerWith's post-flight arithmetic and the
// cap check in RunAgenticReviewer).
//
// The verdict CONTRACT is untouched: this caller returns the same
// (rawJSON, *ai.Response, error) triple as handlerCaller, so ParseReviewResult
// / ValidateReviewResult / reviewSchema all apply byte-identically.
type agenticCaller struct {
	run     AgenticRunner
	timeout time.Duration

	// lastCostUSD is the observed cost of the most recent CallJSON, surfaced so
	// RunAgenticReviewer can enforce the per-review cap against the REAL cost
	// (the text tier estimates cost from token counts; the agentic tier observes
	// it post-hoc via the executor, mirroring run.go's post-flight pattern).
	lastCostUSD float64
	// lastErr records an executor-level failure (Success=false) so the outcome
	// is recorded as unreachable, not a silent pass.
	lastErr string
}

// DefaultAgenticTimeout is the bounded wall-clock cap for a single agentic
// review. A run that exceeds it is cancelled and recorded as unreachable
// (N-1 degradation), never a silent pass (bounded-wait discipline).
const DefaultAgenticTimeout = 5 * time.Minute

// CallJSON satisfies JSONCaller. It runs the bounded agentic review and returns
// the raw verdict JSON. The *ai.Response it returns carries zero token counts:
// the agentic tier's cost is OBSERVED (lastCostUSD), not derived from tokens,
// so RunAgenticReviewer uses the observed cost rather than estimateCost.
func (c *agenticCaller) CallJSON(sysPrompt, userPrompt, _ string) (string, *ai.Response, error) {
	timeout := c.timeout
	if timeout <= 0 {
		timeout = DefaultAgenticTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	run, err := c.run(ctx, sysPrompt, userPrompt)
	if err != nil {
		c.lastErr = err.Error()
		return "", nil, err
	}
	if run == nil {
		c.lastErr = "agentic runner returned nil run"
		return "", nil, fmt.Errorf("agentic runner returned nil run")
	}
	c.lastCostUSD = run.CostUSD
	if !run.Success {
		c.lastErr = run.Err
		if c.lastErr == "" {
			c.lastErr = "agentic run reported failure with no error text"
		}
		return "", nil, fmt.Errorf("agentic run failed: %s", c.lastErr)
	}
	return strings.TrimSpace(run.Output), &ai.Response{}, nil
}

// agenticSystemPrompt is the SHIPPED reviewer systemPrompt PLUS the agentic-only
// verification instruction: use the read-only repo tools to actually verify each
// premise-gate-1 claim and cite the check run in `catch`. The verdict schema is
// UNCHANGED — this is a prompt-level ask, not a contract change. The proposed_fix
// wording is added here too (additive-optional, M2): reviewers are ENCOURAGED to
// include a concrete fix on every reject, but a fix-less reject is friction, not
// a validation error.
const agenticSystemPrompt = systemPrompt + `

You have READ-ONLY repository tools (Read, Grep, Glob). Do not just reason about the doc — VERIFY it against the code:
- For each PREMISE VERIFICATION claim (a factual statement about files, APIs, or behavior), open the cited file, grep for the symbol, or run the cited check, and confirm or refute it against the actual code.
- If a claim is confidently stated as verified but the code contradicts it, that is the strongest possible reject: say so and cite the exact check you ran.
- In the "catch" field, name the concrete check you ran (e.g. "grepped internal/foo.go for BarFunc — not present, contradicting the doc's claim").
- On a reject, you are STRONGLY ENCOURAGED to also include a concrete "proposed_fix" (a corrected claim, replacement paragraph, or added verification-log row). It is optional; omit it only if you cannot propose one.`

// BuildAgenticPrompt returns the user prompt for an agentic review. It reuses
// the text tier's BuildPrompt body and, when a contested premise is supplied
// (Tier-2 escalation), focuses the reviewer on verifying that specific premise
// rather than re-reviewing the whole doc.
func BuildAgenticPrompt(docPath, docBody, contestedPremise string) string {
	base := BuildPrompt(docPath, docBody)
	if strings.TrimSpace(contestedPremise) == "" {
		return base
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\nESCALATED VERIFICATION — a Tier-1 text reviewer disputed this specific premise:\n\n> ")
	b.WriteString(strings.TrimSpace(contestedPremise))
	b.WriteString("\n\nVerify THIS premise against the code with your read-only tools and cite the exact check you ran in `catch`.\n")
	return b.String()
}

// RunAgenticReviewer runs one agentic reviewer end-to-end behind the JSONCaller
// seam, with the SAME N-1 degradation contract as RunReviewer: it NEVER returns
// a nil outcome; on any failure it returns Present=false with a named
// AbsentReason. The per-review cost cap is enforced against the OBSERVED cost
// (mirroring run.go's post-flight cost check), degrading over-cap runs to a
// budget absence rather than blocking the loop.
//
// modelLabel is the artifact label for this reviewer (e.g. "gpt5-6-sol@codex").
// runner is the bounded agentic run; timeout bounds the wall clock.
func RunAgenticReviewer(modelLabel, docPath, docBody, contestedPremise string, maxCostUSD float64, timeout time.Duration, runner AgenticRunner) *ReviewerOutcome {
	out := &ReviewerOutcome{Model: modelLabel}
	if maxCostUSD <= 0 {
		maxCostUSD = DefaultMaxCostUSD
	}
	caller := &agenticCaller{run: runner, timeout: timeout}

	raw, _, cerr := caller.CallJSON(agenticSystemPrompt, BuildAgenticPrompt(docPath, docBody, contestedPremise), reviewSchema)

	// Record the observed cost regardless of outcome (audit).
	out.CostUSD = caller.lastCostUSD

	if cerr != nil {
		out.AbsentReason = ReasonUnreachable
		out.Err = cerr.Error()
		return out
	}

	// POST-flight cost cap against the OBSERVED cost (the agentic tier cannot
	// pre-estimate reliably — a multi-turn agent's cost is only known after the
	// run). Over-cap degrades to a budget absence, never blocks (Principle 2 +
	// bounded discipline).
	if out.CostUSD > maxCostUSD {
		out.AbsentReason = ReasonBudget
		out.Err = fmt.Sprintf("observed cost $%.4f exceeds cap $%.4f (agentic run, post-flight)", out.CostUSD, maxCostUSD)
		out.Present = false
		out.Result = nil
		return out
	}

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
