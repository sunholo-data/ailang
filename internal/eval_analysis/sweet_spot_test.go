package eval_analysis

import (
	"encoding/json"
	"strings"
	"testing"
)

// fixture builds a synthetic result with sensible defaults.
func sweetSpotResult(id, model, executor, errCat string, ok bool, costUSD float64, ttsMs int64) *BenchmarkResult {
	return &BenchmarkResult{
		ID:            id,
		Model:         model,
		Executor:      executor,
		ErrorCategory: errCat,
		StdoutOk:      ok,
		CostUSD:       costUSD,
		SuccessAtMs:   ttsMs,
		DurationMs:    ttsMs,
	}
}

func TestBuildSweetSpot_FastVsSlow(t *testing.T) {
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "fast-model", "", "none", true, 0.10, 5_000),
		sweetSpotResult("b2", "fast-model", "", "none", true, 0.12, 8_000),
		sweetSpotResult("b1", "slow-model", "", "none", true, 0.01, 90_000),
		sweetSpotResult("b2", "slow-model", "", "none", true, 0.01, 120_000),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{}) // default slowMs=60s

	if len(r.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(r.Rows))
	}

	got := map[string]SweetSpotBucket{}
	for _, row := range r.Rows {
		got[row.Model] = row.Buckets
	}
	if got["fast-model"].FastPass != 2 || got["fast-model"].SlowPass != 0 {
		t.Errorf("fast-model buckets = %+v, want 2 fast/0 slow", got["fast-model"])
	}
	if got["slow-model"].SlowPass != 2 || got["slow-model"].FastPass != 0 {
		t.Errorf("slow-model buckets = %+v, want 0 fast/2 slow", got["slow-model"])
	}
}

func TestBuildSweetSpot_BudgetBlocked(t *testing.T) {
	// Model A: 1 cost_killed, 1 step_exhausted → both budget_blocked.
	// Model B: 1 logic_error, 1 timeout → both capability_blocked.
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "modelA", "", "cost_killed", false, 0.05, 30_000),
		sweetSpotResult("b2", "modelA", "", "step_exhausted", false, 0.05, 30_000),
		sweetSpotResult("b1", "modelB", "", "logic_error", false, 0.03, 20_000),
		sweetSpotResult("b2", "modelB", "", "timeout", false, 0.03, 20_000),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{})

	by := map[string]SweetSpotRow{}
	for _, row := range r.Rows {
		by[row.Model] = row
	}
	if by["modelA"].Buckets.BudgetBlocked != 2 {
		t.Errorf("modelA budget_blocked = %d, want 2", by["modelA"].Buckets.BudgetBlocked)
	}
	if by["modelA"].CostKilledCount != 1 || by["modelA"].StepExhaustedCount != 1 {
		t.Errorf("modelA typed counts wrong: %+v", by["modelA"])
	}
	if by["modelB"].Buckets.CapabilityBlocked != 2 {
		t.Errorf("modelB capability_blocked = %d, want 2", by["modelB"].Buckets.CapabilityBlocked)
	}
}

func TestBuildSweetSpot_ProviderBlockedExcludedFromPassRate(t *testing.T) {
	// 1 pass + 1 quota_exhausted (provider noise) → pass rate should be
	// 1/1 = 100%, not 1/2 = 50%.
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "m", "", "none", true, 0.05, 30_000),
		sweetSpotResult("b2", "m", "", "quota_exhausted", false, 0, 0),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{})
	if len(r.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(r.Rows))
	}
	if r.Rows[0].PassRate != 1.0 {
		t.Errorf("PassRate = %v, want 1.0 (quota row excluded)", r.Rows[0].PassRate)
	}
	if r.Rows[0].QuotaCount != 1 {
		t.Errorf("QuotaCount = %d, want 1", r.Rows[0].QuotaCount)
	}
}

func TestBuildSweetSpot_AllProviderBlocked(t *testing.T) {
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "m", "", "quota_exhausted", false, 0, 0),
		sweetSpotResult("b2", "m", "", "rate_limit", false, 0, 0),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{})
	if len(r.Rows) != 1 || r.Rows[0].PassRate != 0 {
		t.Errorf("all-provider-blocked row got %+v, want PassRate=0", r.Rows[0])
	}
	if r.Rows[0].Buckets.ProviderBlocked != 2 {
		t.Errorf("ProviderBlocked = %d, want 2", r.Rows[0].Buckets.ProviderBlocked)
	}
}

func TestBuildSweetSpot_Champions(t *testing.T) {
	// b1: 3 models pass; cheapest is "cheap" ($0.01), fastest is "fast" (5s).
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "cheap", "", "none", true, 0.01, 60_000),
		sweetSpotResult("b1", "fast", "", "none", true, 0.20, 5_000),
		sweetSpotResult("b1", "average", "", "none", true, 0.05, 20_000),
		sweetSpotResult("b2", "only-passer", "", "none", true, 0.10, 30_000),
		// A failed run should NOT show up as a champion.
		sweetSpotResult("b2", "broken", "", "logic_error", false, 0.03, 0),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{})

	champs := map[string]BenchmarkChampion{}
	for _, c := range r.Champions {
		champs[c.BenchmarkID] = c
	}
	if got := champs["b1"].CheapestModel; got != "cheap" {
		t.Errorf("b1 cheapest = %q, want cheap", got)
	}
	if got := champs["b1"].FastestModel; got != "fast" {
		t.Errorf("b1 fastest = %q, want fast", got)
	}
	if got := champs["b2"].CheapestModel; got != "only-passer" {
		t.Errorf("b2 cheapest = %q, want only-passer", got)
	}
	if _, ok := champs["b3"]; ok {
		t.Error("b3 should not have a champion (no passing runs)")
	}
}

func TestBuildSweetSpot_HarnessLabel(t *testing.T) {
	// Same model, different executors → two separate rows.
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "claude-haiku-4-5", "claude", "none", true, 0.05, 30_000),
		sweetSpotResult("b1", "claude-haiku-4-5", "motoko", "none", true, 0.05, 30_000),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{})
	if len(r.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (one per harness)", len(r.Rows))
	}
}

func TestBuildSweetSpot_Empty(t *testing.T) {
	r := BuildSweetSpot(nil, SweetSpotOpts{})
	if len(r.Rows) != 0 || len(r.Champions) != 0 || r.TotalRuns != 0 {
		t.Errorf("empty input produced non-empty report: %+v", r)
	}
}

func TestBuildSweetSpot_DurationFallback(t *testing.T) {
	// SuccessAtMs == 0 (executor didn't track it) should fall back to DurationMs.
	r := &BenchmarkResult{
		ID:          "b",
		Model:       "m",
		StdoutOk:    true,
		DurationMs:  45_000,
		SuccessAtMs: 0,
	}
	r.ErrorCategory = "none"
	report := BuildSweetSpot([]*BenchmarkResult{r}, SweetSpotOpts{})
	if report.Rows[0].Buckets.FastPass != 1 {
		t.Errorf("DurationMs fallback failed: %+v", report.Rows[0])
	}
}

func TestFormatSweetSpotJSON_Roundtrip(t *testing.T) {
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "m", "", "none", true, 0.05, 30_000),
	}
	report := BuildSweetSpot(results, SweetSpotOpts{})
	out, err := FormatSweetSpotJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	var back SweetSpotReport
	if err := json.Unmarshal([]byte(out), &back); err != nil {
		t.Fatalf("roundtrip failed: %v\nJSON: %s", err, out)
	}
	if back.TotalRuns != 1 || len(back.Rows) != 1 {
		t.Errorf("roundtripped report mismatched: %+v", back)
	}
}

func TestFormatSweetSpotCSV_HasHeader(t *testing.T) {
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "m", "", "none", true, 0.05, 30_000),
	}
	out, err := FormatSweetSpotCSV(BuildSweetSpot(results, SweetSpotOpts{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "model,harness,total_runs,pass_rate") {
		t.Errorf("CSV header missing: %q", out[:80])
	}
	if !strings.Contains(out, "## Champions") {
		t.Error("CSV should include Champions section")
	}
}

func TestFormatSweetSpotText_RendersAllSections(t *testing.T) {
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "m", "", "none", true, 0.05, 30_000),
		sweetSpotResult("b1", "m", "", "cost_killed", false, 0.01, 0),
	}
	out := FormatSweetSpotText(BuildSweetSpot(results, SweetSpotOpts{}), false)
	for _, want := range []string{
		"Sweet-Spot Report",
		"Pass%", // header
		"Failure Causes",
		"Cheapest / Fastest Pass per Benchmark",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestEfficiencyByModel_GroupsCorrectly(t *testing.T) {
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "a", "", "none", true, 0.05, 30_000),
		sweetSpotResult("b2", "a", "", "none", true, 0.05, 50_000),
		sweetSpotResult("b1", "b", "", "none", true, 0.10, 10_000),
	}
	eff := EfficiencyByModel(results)
	if len(eff) != 2 {
		t.Fatalf("EfficiencyByModel returned %d models, want 2", len(eff))
	}
	if eff["a"].MedianTimeToSuccessMs <= 0 {
		t.Errorf("model a median TTS = %v, want > 0", eff["a"].MedianTimeToSuccessMs)
	}
}
