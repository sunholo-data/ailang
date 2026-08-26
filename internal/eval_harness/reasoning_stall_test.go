package eval_harness

import (
	"errors"
	"fmt"
	"testing"
)

// TestIsReasoningStall pins the signature measured on 2026-08-26 from OpenRouter
// Broadcast traces. The negative cases matter more than the positives here: this
// predicate decides whether a run is attributed to the model's ability to write
// AILANG or to a provider/harness event, and an over-matching version would quietly
// launder real capability failures out of the baseline.
func TestIsReasoningStall(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		outTokens int
		reasoning int
		want      bool
	}{
		// The three cancelled generations in the 08-18..08-22 corpus, verbatim.
		{"deepseek 08-18 StreamLake", "", 7130, 7130, true},
		{"deepseek 08-19 DigitalOcean", "", 1827, 1827, true},
		{"glm-5.2 08-22 OpenCode", "", 14835, 14835, true},

		// Providers disagree on whether output_tokens includes reasoning_tokens.
		// A strict `==` would fail closed on the ones reporting them disjointly.
		{"disjoint token accounting still counts", "", 0, 4096, true},

		// Whitespace-only content is not an answer.
		{"whitespace-only content", "   \n\t ", 900, 900, true},

		// --- negatives: each one is a way an over-eager matcher goes wrong ---

		// THE live counter-example, probed 2026-08-26: output == reasoning == 37,
		// AND the model answered "ok". Token symmetry alone must never be enough.
		{"equal tokens but content present", "ok", 37, 37, false},

		{"normal answer", "module main\nexport func main() -> () { () }", 512, 40, false},
		{"non-reasoning model, empty content", "", 0, 0, false},
		{"refusal-shaped empty with no reasoning", "", 12, 0, false},
		{"content present, reasoning dominant", "x", 9000, 8999, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsReasoningStall(tt.content, tt.outTokens, tt.reasoning); got != tt.want {
				t.Errorf("IsReasoningStall(%q, out=%d, reasoning=%d) = %v, want %v",
					tt.content, tt.outTokens, tt.reasoning, got, tt.want)
			}
		})
	}
}

// TestCategorizeStandardAPIError_ReasoningStall proves the typed error survives
// wrapping and does not collide with the refusal bucket — both are "empty content,
// HTTP 200", which is exactly why they were conflated before.
func TestCategorizeStandardAPIError_ReasoningStall(t *testing.T) {
	wrapped := fmt.Errorf("benchmark execution failed: %w",
		fmt.Errorf("%w (reasoning_tokens=7130 output_tokens=7130 finish_reason=%q)", ErrReasoningStall, ""))

	if got := CategorizeStandardAPIError(wrapped); got != ErrorCategoryReasoningStall {
		t.Errorf("wrapped stall = %q, want %q", got, ErrorCategoryReasoningStall)
	}
	if !errors.Is(wrapped, ErrReasoningStall) {
		t.Error("errors.Is broke through the wrapping — classification depends on it")
	}

	// Controls: the other two buckets must be unaffected, or this test would pass
	// on a classifier that returns reasoning_stall for everything.
	if got := CategorizeStandardAPIError(errors.New("stop_reason=refusal")); got != ErrorCategoryRefused {
		t.Errorf("refusal = %q, want %q", got, ErrorCategoryRefused)
	}
	if got := CategorizeStandardAPIError(errors.New("connection reset by peer")); got != ErrorCategoryAPI {
		t.Errorf("unknown = %q, want %q", got, ErrorCategoryAPI)
	}
}

// TestCheckReasoningStall_WarmupShapeIsNotFlagged guards the reason this check is
// wired per-method instead of inside adapter.generate: GenerateCodeWarmup caps
// output at one token by design, so a warm-up matches the stall signature exactly.
// If someone later moves the check down into the adapter, this is what should fail.
func TestCheckReasoningStall_WarmupShapeIsNotFlagged(t *testing.T) {
	warmup := &GenerateResult{Code: "", OutputTokens: 1, ReasonTokens: 1}

	// The predicate DOES match the warm-up shape — that is the hazard, stated as a
	// measurement rather than an assumption.
	if !IsReasoningStall(warmup.Code, warmup.OutputTokens, warmup.ReasonTokens) {
		t.Fatal("premise broken: warm-up shape no longer matches the stall signature, " +
			"so the per-method wiring may no longer be necessary — re-check before simplifying")
	}

	// ...and GenerateCodeWarmup must therefore never route through checkReasoningStall.
	// Asserted structurally: a warm-up result passed through the checker WOULD error.
	if _, err := checkReasoningStall(warmup, nil); err == nil {
		t.Error("checkReasoningStall did not flag the warm-up shape; the isolation " +
			"argument in GenerateCodeWarmup's comment no longer holds")
	}
}

// TestCheckReasoningStall_PassesThroughRealResults is the no-false-positive control.
func TestCheckReasoningStall_PassesThroughRealResults(t *testing.T) {
	good := &GenerateResult{Code: "module main", OutputTokens: 300, ReasonTokens: 250}
	if _, err := checkReasoningStall(good, nil); err != nil {
		t.Errorf("flagged a real result: %v", err)
	}

	upstream := errors.New("429 too many requests")
	if _, err := checkReasoningStall(nil, upstream); !errors.Is(err, upstream) {
		t.Error("an existing error must pass through untouched")
	}
}
