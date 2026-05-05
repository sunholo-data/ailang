package eval_analysis

import (
	"testing"
)

// TestComputeEfficiency_BasicSuccess verifies the standard happy path:
// 3 successful runs with assorted speed metrics produce sensible medians.
func TestComputeEfficiency_BasicSuccess(t *testing.T) {
	results := []*BenchmarkResult{
		{
			StdoutOk: true, FirstAttemptMs: 1000, SuccessAtMs: 5000,
			DurationMs: 5500, AgentTurns: 2, TokensPerSec: 50.0, CostUSD: 0.01,
		},
		{
			StdoutOk: true, FirstAttemptMs: 2000, SuccessAtMs: 6000,
			DurationMs: 6500, AgentTurns: 3, TokensPerSec: 60.0, CostUSD: 0.02,
		},
		{
			StdoutOk: true, FirstAttemptMs: 3000, SuccessAtMs: 7000,
			DurationMs: 7500, AgentTurns: 4, TokensPerSec: 70.0, CostUSD: 0.03,
		},
	}
	eff := ComputeEfficiency(results)

	if eff.MedianTimeToFirstAttemptMs != 2000 {
		t.Errorf("MedianTimeToFirstAttemptMs = %v, want 2000", eff.MedianTimeToFirstAttemptMs)
	}
	if eff.MedianTimeToSuccessMs != 6000 {
		t.Errorf("MedianTimeToSuccessMs = %v, want 6000", eff.MedianTimeToSuccessMs)
	}
	if eff.MedianTurnsToSuccess != 3 {
		t.Errorf("MedianTurnsToSuccess = %v, want 3", eff.MedianTurnsToSuccess)
	}
	if eff.MedianTokensPerSec != 60.0 {
		t.Errorf("MedianTokensPerSec = %v, want 60", eff.MedianTokensPerSec)
	}
	// success_rate = 1.0, ttsSeconds = 6, score = 1.0 / (1 + 6/60) = 1.0/1.1 ≈ 0.909
	wantScore := 1.0 / (1.0 + 6.0/60.0)
	if !approxEq(eff.SpeedEfficiencyScore, wantScore, 1e-6) {
		t.Errorf("SpeedEfficiencyScore = %v, want %v", eff.SpeedEfficiencyScore, wantScore)
	}
}

// TestComputeEfficiency_FallbackToDuration ensures executors that don't
// populate SuccessAtMs (gemini, etc.) still get a median TTS via DurationMs.
func TestComputeEfficiency_FallbackToDuration(t *testing.T) {
	results := []*BenchmarkResult{
		{StdoutOk: true, SuccessAtMs: 0, DurationMs: 5000, CostUSD: 0.01}, // SuccessAtMs not measured
		{StdoutOk: true, SuccessAtMs: 0, DurationMs: 7000, CostUSD: 0.02},
	}
	eff := ComputeEfficiency(results)
	if eff.MedianTimeToSuccessMs != 6000 {
		t.Errorf("MedianTimeToSuccessMs = %v, want 6000 (fallback to DurationMs)", eff.MedianTimeToSuccessMs)
	}
}

// TestComputeEfficiency_EmptySlice returns zero-value aggregate without
// panicking on empty input.
func TestComputeEfficiency_EmptySlice(t *testing.T) {
	eff := ComputeEfficiency(nil)
	if eff.MedianTimeToSuccessMs != 0 || eff.SpeedEfficiencyScore != 0 {
		t.Errorf("empty slice should produce zero-value: got %+v", eff)
	}
}

// TestComputeEfficiency_AllFailed reports CostKilledCount but zero
// success-derived metrics when no runs passed.
func TestComputeEfficiency_AllFailed(t *testing.T) {
	results := []*BenchmarkResult{
		{StdoutOk: false, CostKilledAt: 0.55, ErrorCategory: "cost_killed"},
		{StdoutOk: false, ErrorCategory: "logic_error"},
		{StdoutOk: false, CostKilledAt: 0.60, ErrorCategory: "cost_killed"},
	}
	eff := ComputeEfficiency(results)
	if eff.CostKilledCount != 2 {
		t.Errorf("CostKilledCount = %v, want 2", eff.CostKilledCount)
	}
	if eff.MedianTimeToSuccessMs != 0 {
		t.Errorf("MedianTimeToSuccessMs = %v, want 0 (no successes)", eff.MedianTimeToSuccessMs)
	}
	if eff.SpeedEfficiencyScore != 0 {
		t.Errorf("SpeedEfficiencyScore = %v, want 0 (no successes)", eff.SpeedEfficiencyScore)
	}
}

// TestComputeEfficiency_P90Cost validates the percentile statistic on
// cost-per-success.
func TestComputeEfficiency_P90Cost(t *testing.T) {
	results := []*BenchmarkResult{}
	// 10 successful runs, costs 0.01..0.10
	for i := 1; i <= 10; i++ {
		results = append(results, &BenchmarkResult{
			StdoutOk: true,
			CostUSD:  float64(i) / 100.0,
		})
	}
	eff := ComputeEfficiency(results)
	// Sorted: [0.01, 0.02, ..., 0.10]; p90 → idx int(10 * 0.9) = 9 → xs[9] = 0.10
	if !approxEq(eff.P90CostPerSuccess, 0.10, 1e-9) {
		t.Errorf("P90CostPerSuccess = %v, want 0.10", eff.P90CostPerSuccess)
	}
}

// TestCountCostKilled counts only runs with CostKilledAt > 0.
func TestCountCostKilled(t *testing.T) {
	results := []*BenchmarkResult{
		{CostKilledAt: 0.5},
		{CostKilledAt: 0},
		{CostKilledAt: 0.3},
		{CostKilledAt: 0},
		{CostKilledAt: 1.0},
	}
	if got := CountCostKilled(results); got != 3 {
		t.Errorf("CountCostKilled = %v, want 3", got)
	}
}

// TestComputeEfficiency_BackCompatPreV016 ensures pre-v0.16.0 result rows
// (no FirstAttemptMs / SuccessAtMs / TokensPerSec) produce sensible
// fallbacks instead of panicking or emitting NaN.
func TestComputeEfficiency_BackCompatPreV016(t *testing.T) {
	results := []*BenchmarkResult{
		{StdoutOk: true, DurationMs: 3000, AgentTurns: 2, CostUSD: 0.01},
		{StdoutOk: true, DurationMs: 5000, AgentTurns: 3, CostUSD: 0.02},
	}
	eff := ComputeEfficiency(results)
	// FirstAttempt + TokensPerSec missing → 0 medians is OK
	if eff.MedianTimeToFirstAttemptMs != 0 {
		t.Errorf("MedianTimeToFirstAttemptMs should be 0 when not measured, got %v", eff.MedianTimeToFirstAttemptMs)
	}
	if eff.MedianTokensPerSec != 0 {
		t.Errorf("MedianTokensPerSec should be 0 when not measured, got %v", eff.MedianTokensPerSec)
	}
	// TTS falls back to DurationMs
	if eff.MedianTimeToSuccessMs != 4000 {
		t.Errorf("MedianTimeToSuccessMs fallback = %v, want 4000", eff.MedianTimeToSuccessMs)
	}
	// SpeedEfficiencyScore should still compute
	if eff.SpeedEfficiencyScore <= 0 {
		t.Errorf("SpeedEfficiencyScore should be > 0 with successes, got %v", eff.SpeedEfficiencyScore)
	}
}

func approxEq(a, b, tol float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tol
}
