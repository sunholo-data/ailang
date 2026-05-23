package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// fixtureSummary builds a synthetic BenchmarkSummary for table tests.
func fixtureSummary(bench, model, lang string, trials, passed int) eval_harness.BenchmarkSummary {
	rate := 0.0
	if trials > 0 {
		rate = float64(passed) / float64(trials)
	}
	return eval_harness.BenchmarkSummary{
		BenchmarkID: bench,
		Model:       model,
		Lang:        lang,
		Trials:      trials,
		Passed:      passed,
		PassRate:    rate,
	}
}

// flatSummaryMap is the in-memory equivalent of loadReleaseBenchmarks output,
// without round-tripping through summary.json on disk.
func flatSummaryMap(items ...eval_harness.BenchmarkSummary) map[string]eval_harness.BenchmarkSummary {
	out := map[string]eval_harness.BenchmarkSummary{}
	for _, bs := range items {
		out[flatKey(bs.BenchmarkID, bs.Model, bs.Lang)] = bs
	}
	return out
}

// TestRenderReleasePage_Matrix: minimum-viable per-benchmark matrix renders
// with frontmatter, model header, and pass-rate cells.
func TestRenderReleasePage_Matrix(t *testing.T) {
	current := flatSummaryMap(
		fixtureSummary("fizzbuzz", "opencode-gemma4-26b", "ailang", 3, 3),
		fixtureSummary("dense_op", "opencode-gemma4-26b", "ailang", 3, 0),
		fixtureSummary("fizzbuzz", "opencode-qwen3-30b", "ailang", 3, 2),
	)

	md := renderReleasePage("v0.23.0", "", current, nil, 10.0)

	// Frontmatter present.
	if !strings.Contains(md, "---\ntitle: v0.23.0 OS-model leaderboard\n---") {
		t.Errorf("frontmatter missing or malformed")
	}
	// Heading present.
	if !strings.Contains(md, "# v0.23.0 OS-model smoke leaderboard") {
		t.Errorf("h1 heading missing")
	}
	// Per-benchmark matrix section.
	if !strings.Contains(md, "## Per-benchmark pass rate") {
		t.Errorf("matrix section missing")
	}
	// Both model columns appear in header.
	if !strings.Contains(md, "opencode-gemma4-26b") || !strings.Contains(md, "opencode-qwen3-30b") {
		t.Errorf("expected both models in header, got:\n%s", md)
	}
	// Pass-rate cells: gemma4 100% for fizzbuzz, 0% for dense_op.
	if !strings.Contains(md, "100% (n=3)") {
		t.Errorf("expected 100%% pass-rate cell, got:\n%s", md)
	}
	if !strings.Contains(md, "0% (n=3)") {
		t.Errorf("expected 0%% pass-rate cell (dense_op), got:\n%s", md)
	}
	// Missing (benchmark, model) tuples render as em-dash.
	if !strings.Contains(md, "—") {
		t.Errorf("expected em-dash for missing tuple, got:\n%s", md)
	}
	// No trend section when prev not supplied.
	if strings.Contains(md, "Trend deltas since") {
		t.Errorf("trend section should be absent without --prev")
	}
}

// TestRenderReleasePage_TrendDeltas: the deltas table only renders rows that
// crossed the min-pp threshold; the dir arrow matches the sign.
func TestRenderReleasePage_TrendDeltas(t *testing.T) {
	prev := flatSummaryMap(
		fixtureSummary("fizzbuzz", "m1", "ailang", 3, 1), // 33%
		fixtureSummary("dense_op", "m1", "ailang", 3, 0), // 0%
		fixtureSummary("noisy", "m1", "ailang", 3, 1),    // 33% — moves only 1pp
	)
	current := flatSummaryMap(
		fixtureSummary("fizzbuzz", "m1", "ailang", 3, 3), // 100% — +67pp ▲
		fixtureSummary("dense_op", "m1", "ailang", 3, 1), // 33% — +33pp ▲
		fixtureSummary("noisy", "m1", "ailang", 30, 10),  // 33% — 0pp, suppressed
	)

	md := renderReleasePage("v0.23.0", "v0.22.0", current, prev, 10.0)

	if !strings.Contains(md, "## Trend deltas since v0.22.0") {
		t.Errorf("trend section header missing")
	}
	if !strings.Contains(md, "fizzbuzz") || !strings.Contains(md, "dense_op") {
		t.Errorf("expected fizzbuzz + dense_op rows in trend table")
	}
	// Split into matrix vs trend sections — noisy belongs in matrix
	// (it has trials) but must NOT appear in the trend table because its
	// pass rate did not move by >= 10pp.
	trendIdx := strings.Index(md, "## Trend deltas since")
	if trendIdx < 0 {
		t.Fatalf("trend section not found")
	}
	trendBody := md[trendIdx:]
	if strings.Contains(trendBody, "| noisy ") {
		t.Errorf("noisy moved 0pp and should be suppressed from trend table; got trend body:\n%s", trendBody)
	}
	if !strings.Contains(md, "▲") {
		t.Errorf("expected up-arrow for positive movement")
	}
	if !strings.Contains(md, "+66.7pp") || !strings.Contains(md, "+33.3pp") {
		t.Errorf("expected +66.7pp and +33.3pp deltas; got:\n%s", md)
	}
}

// TestRenderReleasePage_TrendNegative: regressions render with ▼ and a
// negative pp value.
func TestRenderReleasePage_TrendNegative(t *testing.T) {
	prev := flatSummaryMap(fixtureSummary("regression_bench", "m1", "ailang", 3, 3))    // 100%
	current := flatSummaryMap(fixtureSummary("regression_bench", "m1", "ailang", 3, 0)) // 0%
	md := renderReleasePage("v0.23.0", "v0.22.0", current, prev, 10.0)

	if !strings.Contains(md, "▼") {
		t.Errorf("expected down-arrow ▼ for regression, got:\n%s", md)
	}
	if !strings.Contains(md, "-100.0pp") {
		t.Errorf("expected -100.0pp regression line, got:\n%s", md)
	}
}

// TestRenderReleasePage_TrendNoMovers: when nothing crosses threshold, the
// trend section renders an explicit "(no benchmarks moved by ...)" notice
// rather than an empty table — so the page reads cleanly.
func TestRenderReleasePage_TrendNoMovers(t *testing.T) {
	prev := flatSummaryMap(fixtureSummary("flat", "m1", "ailang", 3, 1))      // 33%
	current := flatSummaryMap(fixtureSummary("flat", "m1", "ailang", 30, 10)) // 33%
	md := renderReleasePage("v0.23.0", "v0.22.0", current, prev, 10.0)

	if !strings.Contains(md, "(no benchmarks moved by >=10.0pp)") {
		t.Errorf("expected 'no movers' notice, got:\n%s", md)
	}
}

// TestUniqueHelpers: the axis helpers de-duplicate and sort, so the per-
// release matrix is stable across runs.
func TestUniqueHelpers(t *testing.T) {
	m := flatSummaryMap(
		fixtureSummary("b2", "mB", "ailang", 1, 1),
		fixtureSummary("b1", "mA", "ailang", 1, 1),
		fixtureSummary("b1", "mB", "ailang", 1, 1),
	)
	benches := uniqueBenchmarks(m)
	if len(benches) != 2 || benches[0] != "b1" || benches[1] != "b2" {
		t.Errorf("uniqueBenchmarks = %v, want [b1 b2]", benches)
	}
	models := uniqueModels(m)
	if len(models) != 2 || models[0] != "mA" || models[1] != "mB" {
		t.Errorf("uniqueModels = %v, want [mA mB]", models)
	}
}

// TestAbsFloat is a 5-line sanity check on the helper.
func TestAbsFloat(t *testing.T) {
	if absFloat(-1.5) != 1.5 {
		t.Errorf("absFloat(-1.5) = %v", absFloat(-1.5))
	}
	if absFloat(2.0) != 2.0 {
		t.Errorf("absFloat(2.0) = %v", absFloat(2.0))
	}
	if absFloat(0) != 0 {
		t.Errorf("absFloat(0) = %v", absFloat(0))
	}
}
