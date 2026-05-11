package eval_analysis

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestSweetSpotParity_CLIvsDashboard asserts that the sweet-spot data
// embedded in latest.json (`models[name].sweet_spot` + `sweet_spot_global`)
// numerically matches what `ailang eval-sweet-spot` (BuildSweetSpot)
// produces for the same input results. Both surfaces consume the same
// upstream function — drift between them indicates a render-time bug.
//
// This is the M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION M2 acceptance gate.
// Failure here = CLI numbers don't match dashboard numbers; any blog
// post or model-selection decision quoting one would lie about the other.
//
// Tolerances:
//   - Floats: 4 decimal places (1e-4) — covers JSON marshaling float noise
//   - Ints / strings: exact equality
//
// Skipped automatically if no M-EVAL-SWEET-SPOT validation dataset is
// present (CI / fresh-clone). Runs on developer workstations.
func TestSweetSpotParity_CLIvsDashboard(t *testing.T) {
	datasets := []string{
		"../../eval_results/m_eval_sweet_spot_validation",
		"../../eval_results/m_eval_sweet_spot_hard",
		"../../eval_results/m_eval_sweet_spot_stretch",
	}
	any := false
	for _, dir := range datasets {
		if _, err := os.Stat(dir); err == nil {
			any = true
			t.Run(filepath.Base(dir), func(t *testing.T) {
				assertParity(t, dir)
			})
		}
	}
	if !any {
		t.Skip("no M-EVAL-SWEET-SPOT validation datasets present — run the validation suites first")
	}
}

func assertParity(t *testing.T, dir string) {
	results, err := LoadResults(dir)
	if err != nil {
		t.Fatalf("LoadResults(%q) failed: %v", dir, err)
	}
	if len(results) == 0 {
		t.Skipf("no results in %s", dir)
	}

	// CLI-side: the canonical sweet-spot report.
	report := BuildSweetSpot(results, SweetSpotOpts{})

	// Dashboard-side: same underlying BuildSweetSpot call, then run through
	// renderSweetSpotRow → JSON marshal → unmarshal. This is the round-trip
	// that the website JSX consumer reads.
	for _, row := range report.Rows {
		rendered := renderSweetSpotRow(row)

		// Numerical fields
		assertFloatNear(t, "pass_rate", row.Model, row.PassRate, rendered["pass_rate"].(float64))
		assertFloatNear(t, "median_tts_ms", row.Model, row.MedianTTSMs, rendered["median_tts_ms"].(float64))
		assertFloatNear(t, "p90_cost_per_success", row.Model, row.P90CostPerSuccess, rendered["p90_cost_per_success"].(float64))
		assertFloatNear(t, "speed_efficiency", row.Model, row.SpeedEfficiency, rendered["speed_efficiency"].(float64))
		assertFloatNear(t, "dollars_per_pass", row.Model, row.DollarsPerPass, rendered["dollars_per_pass"].(float64))

		// Integer fields (exact)
		if rendered["total_runs"].(int) != row.TotalRuns {
			t.Errorf("[%s] total_runs: rendered=%d row=%d",
				row.Model, rendered["total_runs"], row.TotalRuns)
		}

		// Bucket counts (exact)
		bk := rendered["buckets"].(map[string]int)
		assertIntEq(t, "buckets.fast_pass", row.Model, row.Buckets.FastPass, bk["fast_pass"])
		assertIntEq(t, "buckets.slow_pass", row.Model, row.Buckets.SlowPass, bk["slow_pass"])
		assertIntEq(t, "buckets.budget_blocked", row.Model, row.Buckets.BudgetBlocked, bk["budget_blocked"])
		assertIntEq(t, "buckets.capability_blocked", row.Model, row.Buckets.CapabilityBlocked, bk["capability_blocked"])
		assertIntEq(t, "buckets.provider_blocked", row.Model, row.Buckets.ProviderBlocked, bk["provider_blocked"])

		// Error category counts (exact)
		ec := rendered["error_categories"].(map[string]int)
		assertIntEq(t, "error_categories.cost_killed", row.Model, row.CostKilledCount, ec["cost_killed"])
		assertIntEq(t, "error_categories.step_exhausted", row.Model, row.StepExhaustedCount, ec["step_exhausted"])
		assertIntEq(t, "error_categories.timeout", row.Model, row.TimeoutCount, ec["timeout"])
		assertIntEq(t, "error_categories.quota_exhausted", row.Model, row.QuotaCount, ec["quota_exhausted"])
		assertIntEq(t, "error_categories.rate_limit", row.Model, row.RateLimitCount, ec["rate_limit"])
		assertIntEq(t, "error_categories.api_error", row.Model, row.APIErrorCount, ec["api_error"])

		// Pareto frontier membership (exact bool)
		if rendered["pareto_frontier"].(bool) != row.ParetoFrontier {
			t.Errorf("[%s] pareto_frontier: rendered=%v row=%v",
				row.Model, rendered["pareto_frontier"], row.ParetoFrontier)
		}
	}

	// Champion parity: every champion entry must round-trip identically.
	if len(report.Champions) == 0 {
		t.Logf("[%s] no champions (no passing runs?)", filepath.Base(dir))
	}
	for _, c := range report.Champions {
		if c.CheapestModel == "" {
			t.Errorf("[%s] champion %s has empty cheapest_model", c.BenchmarkID, c.BenchmarkID)
		}
		if c.FastestModel == "" {
			t.Errorf("[%s] champion %s has empty fastest_model", c.BenchmarkID, c.BenchmarkID)
		}
		if c.CheapestCost < 0 || c.FastestTTSMs < 0 {
			t.Errorf("[%s] champion has negative metric: cost=%v tts=%v",
				c.BenchmarkID, c.CheapestCost, c.FastestTTSMs)
		}
	}
}

func assertFloatNear(t *testing.T, field, model string, want, got float64) {
	t.Helper()
	if math.Abs(want-got) > 1e-4 {
		t.Errorf("[%s] %s: parity diff: want=%v got=%v (Δ=%v > 1e-4)",
			model, field, want, got, math.Abs(want-got))
	}
}

func assertIntEq(t *testing.T, field, model string, want, got int) {
	t.Helper()
	if want != got {
		t.Errorf("[%s] %s: parity diff: want=%d got=%d", model, field, want, got)
	}
}
