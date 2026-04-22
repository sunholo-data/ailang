package eval_analysis

import "testing"

// mkResult is a tiny constructor so the tables below stay readable.
func mkResult(id, lang, model, errCat string, stdoutOk, firstOk, refused bool, tokens int, cost float64) *BenchmarkResult {
	return &BenchmarkResult{
		ID:              id,
		Lang:            lang,
		Model:           model,
		ErrorCategory:   errCat,
		StdoutOk:        stdoutOk,
		FirstAttemptOk:  firstOk,
		RefusalDetected: refused,
		OutputTokens:    tokens,
		CostUSD:         cost,
	}
}

// TestComputeReliability verifies the global + per-model api_error and
// refusal counters match hand-computed values.
func TestComputeReliability(t *testing.T) {
	results := []*BenchmarkResult{
		mkResult("b1", "ailang", "gpt5", "api_error", false, false, false, 0, 0),
		mkResult("b1", "python", "gpt5", "none", true, true, false, 100, 0.01),
		mkResult("b2", "ailang", "gemini-3-1-pro", "api_error", false, false, false, 0, 0),
		mkResult("b2", "python", "gemini-3-1-pro", "api_error", false, false, false, 0, 0),
		mkResult("b3", "ailang", "gpt5", "none", false, false, true, 50, 0.005), // refusal
	}
	rel := computeReliability(results)
	if rel.APIErrorCount != 3 {
		t.Errorf("global api_error: want 3, got %d", rel.APIErrorCount)
	}
	if rel.RefusalCount != 1 {
		t.Errorf("global refusal: want 1, got %d", rel.RefusalCount)
	}
	if g := rel.PerModel["gemini-3-1-pro"]; g == nil || g.APIErrorCount != 2 {
		t.Errorf("gemini-3-1-pro api_error: want 2, got %+v", g)
	}
	if g := rel.PerModel["gemini-3-1-pro"]; g.AILANGAPIError != 1 || g.PythonAPIError != 1 {
		t.Errorf("gemini lang split: want 1/1, got ailang=%d python=%d", g.AILANGAPIError, g.PythonAPIError)
	}
}

// TestBuildTierModelMatrix confirms (tier, model, lang) grouping and that
// benchmarks without a tier are dropped silently.
func TestBuildTierModelMatrix(t *testing.T) {
	results := []*BenchmarkResult{
		mkResult("b1", "ailang", "gpt5", "none", true, true, false, 100, 0),
		mkResult("b1", "ailang", "gpt5", "none", false, false, false, 80, 0),
		mkResult("b2", "python", "gpt5", "none", true, true, false, 100, 0),
		mkResult("b-untagged", "ailang", "gpt5", "none", true, true, false, 50, 0),
	}
	tierOf := map[string]string{"b1": "core", "b2": "stretch"} // b-untagged absent
	raw := buildTierModelMatrix(results, tierOf)
	stats := finalizeTierModelMatrix(raw)

	core := stats["core"]["gpt5"]["ailang"]
	if core == nil {
		t.Fatal("expected core/gpt5/ailang entry")
	}
	if core.TotalRuns != 2 || core.SuccessRate != 0.5 {
		t.Errorf("core ailang: want 2 runs, 0.5 rate; got %d runs, %.2f rate", core.TotalRuns, core.SuccessRate)
	}
	if _, ok := stats["core"]["gpt5"]["python"]; ok {
		t.Error("core should have no python entry (b2 is stretch)")
	}
	if _, ok := stats["unknown"]; ok {
		t.Error("untagged benchmark should not create a tier bucket")
	}
}

// TestComputeTierExtras verifies repair delta + cost + api-error split for
// all languages including JS/Go (not just ailang/python).
func TestComputeTierExtras(t *testing.T) {
	results := []*BenchmarkResult{
		// core/ailang: 2 runs, first-attempt 1, stdout-ok 2 → repair delta = +0.5
		mkResult("b1", "ailang", "gpt5", "none", true, true, false, 0, 1.0),
		mkResult("b1", "ailang", "gpt5", "none", true, false, false, 0, 2.0),
		// core/python: 1 run, api_error
		mkResult("b2", "python", "gpt5", "api_error", false, false, false, 0, 0.5),
		// core/javascript: 1 passing run
		mkResult("b3", "javascript", "gpt5", "none", true, true, false, 0, 0.3),
	}
	tierOf := map[string]string{"b1": "core", "b2": "core", "b3": "core"}
	extras := computeTierExtras(results, tierOf)
	core := extras["core"]
	if core == nil {
		t.Fatal("expected core extras")
	}

	ail := core.langs["ailang"]
	if ail == nil {
		t.Fatal("expected ailang entry in core extras")
	}
	if ail.RepairDelta != 0.5 {
		t.Errorf("ailang repair delta: want 0.5, got %.2f", ail.RepairDelta)
	}
	if ail.AvgCostUSD != 1.5 {
		t.Errorf("ailang avg cost: want 1.5, got %.2f", ail.AvgCostUSD)
	}

	py := core.langs["python"]
	if py == nil {
		t.Fatal("expected python entry in core extras")
	}
	if py.APIErrors != 1 {
		t.Errorf("python api errors: want 1, got %d", py.APIErrors)
	}
	if ail.APIErrors != 0 {
		t.Errorf("ailang api errors: want 0, got %d", ail.APIErrors)
	}

	js := core.langs["javascript"]
	if js == nil {
		t.Fatal("expected javascript entry in core extras — JS results must not be silently dropped")
	}
	if js.Runs != 1 || js.Pass != 1 {
		t.Errorf("javascript: want 1 run, 1 pass; got %d/%d", js.Runs, js.Pass)
	}
}

// TestBuildHistoricalTierPoints covers the retroactive mapping: a benchmark
// present in tierOf but absent from the baseline results contributes nothing,
// and benchmarks absent from tierOf are dropped silently.
func TestBuildHistoricalTierPoints(t *testing.T) {
	results := []*BenchmarkResult{
		mkResult("b1", "ailang", "gpt5", "none", true, true, false, 0, 0),
		mkResult("b1", "python", "gpt5", "none", true, true, false, 0, 0),
		mkResult("b-untagged", "ailang", "gpt5", "none", true, true, false, 0, 0),
	}
	tierOf := map[string]string{"b1": "core", "b-future": "stretch"}
	points := buildHistoricalTierPoints(results, tierOf)

	core := points["core"]
	if core == nil {
		t.Fatal("expected core point")
	}
	if core.AILANGRuns != 1 || core.AILANGSuccessRate != 1.0 {
		t.Errorf("core ailang: want 1 run, 1.0 rate; got %d, %.2f", core.AILANGRuns, core.AILANGSuccessRate)
	}
	if core.BenchmarkCount != 1 {
		t.Errorf("core bench count: want 1, got %d", core.BenchmarkCount)
	}
	if _, ok := points["stretch"]; ok {
		t.Error("stretch should be absent — no b-future runs in baseline")
	}
}

// TestBuildTagAggregatesSmoke covers the end-to-end tag builder using the
// real benchmarks/ directory. It is a smoke test — exact counts change as
// benchmarks are added — so we only assert the happy-path shape.
func TestBuildTagAggregatesSmoke(t *testing.T) {
	results := []*BenchmarkResult{
		mkResult("adt_option", "ailang", "gpt5", "none", true, true, false, 100, 0),
	}
	out := buildTagAggregates(results)
	// The benchmarks directory may or may not contain adt_option.yml — if it
	// does, we expect at least one tag. If not, the result is empty and the
	// smoke test only checks that the builder doesn't panic.
	if out == nil {
		t.Error("buildTagAggregates returned nil")
	}
}
