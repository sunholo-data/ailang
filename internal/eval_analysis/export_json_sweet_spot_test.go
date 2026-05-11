package eval_analysis

import (
	"encoding/json"
	"testing"
)

// TestRenderSweetSpotRow_ShapeStable asserts that the rendered map carries
// every field the dashboard JSX expects. Failure here = JSX consumers break.
//
// This is the M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION M1 schema contract.
func TestRenderSweetSpotRow_ShapeStable(t *testing.T) {
	row := SweetSpotRow{
		Model:              "test-model",
		Harness:            "test-harness",
		TotalRuns:          10,
		PassRate:           0.8,
		MedianTTSMs:        45000,
		MedianTokensPerSec: 120,
		P90CostPerSuccess:  0.05,
		SpeedEfficiency:    0.65,
		DollarsPerPass:     0.04,
		ParetoFrontier:     true,
		CostKilledCount:    1,
		StepExhaustedCount: 0,
		TimeoutCount:       0,
		QuotaCount:         0,
		RateLimitCount:     0,
		APIErrorCount:      2,
		FinishReasons:      map[string]int{"stop": 8, "cost_exhausted": 1, "": 1},
		Buckets: SweetSpotBucket{
			FastPass: 5, SlowPass: 3, BudgetBlocked: 1,
			CapabilityBlocked: 1, ProviderBlocked: 0,
		},
	}

	out := renderSweetSpotRow(row)

	// Top-level required keys
	requiredKeys := []string{
		"model", "harness", "total_runs",
		"pass_rate", "median_tts_ms", "median_tokens_per_sec",
		"p90_cost_per_success", "speed_efficiency",
		"dollars_per_pass", "pareto_frontier",
		"buckets", "error_categories", "finish_reasons",
	}
	for _, k := range requiredKeys {
		if _, ok := out[k]; !ok {
			t.Errorf("rendered row missing required key %q", k)
		}
	}

	// Nested bucket keys
	buckets, ok := out["buckets"].(map[string]int)
	if !ok {
		t.Fatalf("buckets is not map[string]int, got %T", out["buckets"])
	}
	for _, b := range []string{"fast_pass", "slow_pass", "budget_blocked",
		"capability_blocked", "provider_blocked"} {
		if _, ok := buckets[b]; !ok {
			t.Errorf("buckets missing %q", b)
		}
	}

	// Nested error_categories keys
	cats, ok := out["error_categories"].(map[string]int)
	if !ok {
		t.Fatalf("error_categories is not map[string]int, got %T", out["error_categories"])
	}
	for _, c := range []string{"cost_killed", "step_exhausted", "timeout",
		"quota_exhausted", "rate_limit", "api_error"} {
		if _, ok := cats[c]; !ok {
			t.Errorf("error_categories missing %q", c)
		}
	}

	// Spot-check values
	if out["dollars_per_pass"].(float64) != 0.04 {
		t.Errorf("dollars_per_pass = %v, want 0.04", out["dollars_per_pass"])
	}
	if !out["pareto_frontier"].(bool) {
		t.Error("pareto_frontier expected true")
	}
}

// TestRenderSweetSpotRow_JSONRoundtrip asserts the rendered map serializes
// cleanly to JSON and back without losing fields.
func TestRenderSweetSpotRow_JSONRoundtrip(t *testing.T) {
	row := SweetSpotRow{
		Model:          "m",
		PassRate:       1.0,
		MedianTTSMs:    30000,
		DollarsPerPass: 0.02,
		FinishReasons:  map[string]int{"stop": 3},
		Buckets:        SweetSpotBucket{FastPass: 3},
	}
	out := renderSweetSpotRow(row)
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var back map[string]interface{}
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	// dollars_per_pass survives the trip
	if dp, ok := back["dollars_per_pass"].(float64); !ok || dp != 0.02 {
		t.Errorf("dollars_per_pass lost in JSON roundtrip: %v", back["dollars_per_pass"])
	}
	// finish_reasons stop=3
	if fr, ok := back["finish_reasons"].(map[string]interface{}); !ok {
		t.Fatalf("finish_reasons is not a map after roundtrip: %T", back["finish_reasons"])
	} else if n, ok := fr["stop"].(float64); !ok || n != 3 {
		t.Errorf("finish_reasons.stop = %v, want 3", fr["stop"])
	}
}

// TestBuildSweetSpot_PopulatesDollarsPerPassAndFrontier verifies the new
// fields on SweetSpotRow are populated by BuildSweetSpot. This is the
// M1 acceptance gate.
func TestBuildSweetSpot_PopulatesDollarsPerPassAndFrontier(t *testing.T) {
	// 2 models, each runs 1 benchmark:
	// - "cheap"  passes at $0.02 / 50s   → DollarsPerPass=0.02
	// - "fast"   passes at $0.10 / 10s   → DollarsPerPass=0.10
	// Both are Pareto-optimal because each beats the other on one axis.
	results := []*BenchmarkResult{
		{ID: "b1", Model: "cheap", StdoutOk: true, CostUSD: 0.02,
			SuccessAtMs: 50000, DurationMs: 50000, ErrorCategory: "none"},
		{ID: "b1", Model: "fast", StdoutOk: true, CostUSD: 0.10,
			SuccessAtMs: 10000, DurationMs: 10000, ErrorCategory: "none"},
	}
	report := BuildSweetSpot(results, SweetSpotOpts{})
	if len(report.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(report.Rows))
	}
	byModel := map[string]SweetSpotRow{}
	for _, r := range report.Rows {
		byModel[r.Model] = r
	}
	if got := byModel["cheap"].DollarsPerPass; got != 0.02 {
		t.Errorf("cheap DollarsPerPass = %v, want 0.02", got)
	}
	if got := byModel["fast"].DollarsPerPass; got != 0.10 {
		t.Errorf("fast DollarsPerPass = %v, want 0.10", got)
	}
	// Both Pareto-optimal: cheap beats fast on cost, fast beats cheap on TTS.
	if !byModel["cheap"].ParetoFrontier {
		t.Error("cheap should be Pareto-optimal")
	}
	if !byModel["fast"].ParetoFrontier {
		t.Error("fast should be Pareto-optimal")
	}
}

// TestBuildSweetSpot_DominatedModelNotOnFrontier verifies the negative case:
// a model strictly worse on BOTH axes is excluded from the frontier.
func TestBuildSweetSpot_DominatedModelNotOnFrontier(t *testing.T) {
	results := []*BenchmarkResult{
		{ID: "b1", Model: "good", StdoutOk: true, CostUSD: 0.01,
			SuccessAtMs: 10000, DurationMs: 10000, ErrorCategory: "none"},
		{ID: "b1", Model: "bad", StdoutOk: true, CostUSD: 0.10,
			SuccessAtMs: 100000, DurationMs: 100000, ErrorCategory: "none"},
	}
	report := BuildSweetSpot(results, SweetSpotOpts{})
	byModel := map[string]SweetSpotRow{}
	for _, r := range report.Rows {
		byModel[r.Model] = r
	}
	if !byModel["good"].ParetoFrontier {
		t.Error("good should be Pareto-optimal")
	}
	if byModel["bad"].ParetoFrontier {
		t.Error("bad should be dominated (good is cheaper AND faster)")
	}
}

// TestBuildSweetSpot_FinishReasonsTallied verifies the FinishReasons map
// counts all finish_reason values across a model's runs.
func TestBuildSweetSpot_FinishReasonsTallied(t *testing.T) {
	results := []*BenchmarkResult{
		{ID: "b1", Model: "m", StdoutOk: true, CostUSD: 0.01,
			SuccessAtMs: 10000, FinishReason: "stop", ErrorCategory: "none"},
		{ID: "b2", Model: "m", StdoutOk: false, CostUSD: 0.01,
			FinishReason: "cost_exhausted", ErrorCategory: "cost_killed"},
		{ID: "b3", Model: "m", StdoutOk: false, CostUSD: 0,
			FinishReason: "", ErrorCategory: "api_error"},
	}
	report := BuildSweetSpot(results, SweetSpotOpts{})
	if len(report.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(report.Rows))
	}
	fr := report.Rows[0].FinishReasons
	if fr["stop"] != 1 || fr["cost_exhausted"] != 1 || fr[""] != 1 {
		t.Errorf("FinishReasons = %+v, want {stop:1, cost_exhausted:1, '':1}", fr)
	}
}
