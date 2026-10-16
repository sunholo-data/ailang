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
	if !CarryFailedRunSpend(m, res) || !MarkIdleKilled(m, res) {
		t.Fatal("an idle_killed result was not recognised")
	}
	if m.IsValid() || m.InvalidReason() != ReasonHarnessError {
		t.Fatalf("idle kill banked valid=%v reason=%q, want invalid harness_error", m.IsValid(), m.InvalidReason())
	}
	if m.DurationMs != 618000 || m.TotalTokens != 1250 || m.CostUSD != 0.012 || m.AgentTurns != 3 {
		t.Fatalf("spend not carried: dur=%d tokens=%d cost=%v turns=%d", m.DurationMs, m.TotalTokens, m.CostUSD, m.AgentTurns)
	}
	if got := CategorizeAgentError(errors.New("x"), executor.FinishIdleKilled); got != ErrorCategoryTimeout {
		t.Fatalf("idle_killed categorised as %q, want timeout", got)
	}
}

// A run stopped at its wall-clock bound is a VALID model failure that still
// spent its hour: the 2026-10-16 glm-5.3-flash motoko row banked 0 ms and $0.
func TestCarryFailedRunSpend_WallClockTimeout(t *testing.T) {
	m := &RunMetrics{ErrorCategory: ErrorCategoryTimeout}
	res := &AgentBenchmarkResult{
		DurationMS:     3600000,
		NumTurns:       41,
		ToolCallCount:  40,
		Cost:           0.09,
		CostProvenance: "metered",
		Usage:          TokenUsage{InputTokens: 900, OutputTokens: 100},
	}
	if !CarryFailedRunSpend(m, res) {
		t.Fatal("spend of a timed-out run was not carried")
	}
	if MarkIdleKilled(m, res) || m.Validity != nil {
		t.Fatal("a wall-clock timeout was treated as an idle kill")
	}
	if m.DurationMs != 3600000 || m.CostUSD != 0.09 || m.CostProvenance != "metered" || m.AgentToolCalls != 40 || m.TotalTokens != 1000 {
		t.Fatalf("spend not carried: %+v", m)
	}

	// A setup failure that never ran has nothing to carry.
	if CarryFailedRunSpend(&RunMetrics{}, &AgentBenchmarkResult{}) {
		t.Fatal("an empty result reported spend")
	}
}
