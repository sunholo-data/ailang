package eval_harness

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// An idle-killed run is the harness's failure: invalid, so it is retried and
// left out of pass rates, with the spend it really incurred on the row.
func TestMarkIdleKilled(t *testing.T) {
	m := &RunMetrics{ErrorCategory: ErrorCategoryTimeout}
	res := &AgentBenchmarkResult{
		FinishReason: executor.FinishIdleKilled,
		DurationMS:   618000,
		NumTurns:     3,
		Cost:         0.012,
		Usage:        TokenUsage{InputTokens: 1000, OutputTokens: 200, CacheReadInputTokens: 50},
	}
	if !MarkIdleKilled(m, res) {
		t.Fatal("an idle_killed result was not recognised")
	}
	if m.IsValid() || m.InvalidReason() != ReasonHarnessError {
		t.Fatalf("idle kill banked valid=%v reason=%q, want invalid harness_error", m.IsValid(), m.InvalidReason())
	}
	if m.DurationMs != 618000 || m.TotalTokens != 1250 || m.CostUSD != 0.012 || m.AgentTurns != 3 {
		t.Fatalf("spend not carried: dur=%d tokens=%d cost=%v turns=%d", m.DurationMs, m.TotalTokens, m.CostUSD, m.AgentTurns)
	}

	// A model timeout (wall clock) stays a valid failure.
	other := &RunMetrics{ErrorCategory: ErrorCategoryTimeout}
	if MarkIdleKilled(other, &AgentBenchmarkResult{FinishReason: executor.FinishTimeout}) || other.Validity != nil {
		t.Fatal("a wall-clock timeout was treated as an idle kill")
	}
	if got := CategorizeAgentError(errors.New("x"), executor.FinishIdleKilled); got != ErrorCategoryTimeout {
		t.Fatalf("idle_killed categorised as %q, want timeout", got)
	}
}
