package main

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
)

// TestBuildELOModeReport_CoverageGating verifies the M-EVAL-VALIDITY-DISCIPLINE
// coverage gate: a model whose AILANG coverage is below 50% of the max is flagged
// provisional, so a sparse ELO can't be misread as a headline. (This is the exact
// bug where a 6-benchmark local model posted a bogus #1 next to 55-benchmark cloud
// models.)
func TestBuildELOModeReport_CoverageGating(t *testing.T) {
	var results []*eval_analysis.BenchmarkResult
	// full-model runs 40 distinct AILANG benchmarks (the max coverage).
	for i := 0; i < 40; i++ {
		id := fmt.Sprintf("bench%02d", i)
		results = append(results, &eval_analysis.BenchmarkResult{
			ID: id, Lang: "ailang", Model: "full-model", EvalMode: "agent",
			CompileOk: true, RuntimeOk: true, StdoutOk: i%2 == 0, // 50% pass → non-degenerate fit
		})
	}
	// sparse-model runs only 5 benchmarks (< 50% of 40 → provisional).
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("bench%02d", i)
		results = append(results, &eval_analysis.BenchmarkResult{
			ID: id, Lang: "ailang", Model: "sparse-model", EvalMode: "agent",
			CompileOk: true, RuntimeOk: true, StdoutOk: true,
		})
	}
	// mid-model runs exactly at the threshold (20 = 50% of 40 → NOT provisional).
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("bench%02d", i)
		results = append(results, &eval_analysis.BenchmarkResult{
			ID: id, Lang: "ailang", Model: "mid-model", EvalMode: "agent",
			CompileOk: true, RuntimeOk: true, StdoutOk: i%3 == 0,
		})
	}

	rep := buildELOModeReport("agent", results)
	if rep == nil {
		t.Fatal("expected a report")
	}
	if rep.MaxCoverage != 40 {
		t.Fatalf("MaxCoverage want 40, got %d", rep.MaxCoverage)
	}

	byModel := map[string]eloModelRow{}
	for _, m := range rep.Models {
		byModel[m.Model] = m
	}

	if got := byModel["full-model"].Benchmarks; got != 40 {
		t.Fatalf("full-model coverage want 40, got %d", got)
	}
	if byModel["full-model"].Provisional {
		t.Fatal("full-model (40/40) must NOT be provisional")
	}
	if got := byModel["sparse-model"].Benchmarks; got != 5 {
		t.Fatalf("sparse-model coverage want 5, got %d", got)
	}
	if !byModel["sparse-model"].Provisional {
		t.Fatal("sparse-model (5 < 20 threshold) MUST be provisional")
	}
	if byModel["mid-model"].Benchmarks != 20 || byModel["mid-model"].Provisional {
		t.Fatalf("mid-model (20 == threshold) must NOT be provisional; got cov=%d prov=%v",
			byModel["mid-model"].Benchmarks, byModel["mid-model"].Provisional)
	}
}
