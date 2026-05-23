package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// writeFixtureSummary persists a synthetic summary.json under
// rotationDir/<slot>/summary.json so findCandidates has something to read.
func writeFixtureSummary(t *testing.T, rotationDir, slot string, summaries []eval_harness.BenchmarkSummary) {
	t.Helper()
	slotDir := filepath.Join(rotationDir, slot)
	if err := os.MkdirAll(slotDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	rs := eval_harness.RotationSummary{
		OutputDir:        slotDir,
		TotalResultFiles: len(summaries),
		TrialsPerBench:   3,
		BenchmarkSummary: summaries,
	}
	data, err := json.Marshal(&rs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(slotDir, "summary.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestFindCandidates_PicksPersistentFailures: the canonical use case.
// Three benchmarks at N=3 trials: dense_op fails 3/3 with compile_error
// (clear candidate); adt_option fails 1/3 with logic_error (below
// min-fail threshold); fizzbuzz passes 3/3 (no candidate).
func TestFindCandidates_PicksPersistentFailures(t *testing.T) {
	rotationDir := t.TempDir()
	writeFixtureSummary(t, rotationDir, "0100_gemma4_smoke", []eval_harness.BenchmarkSummary{
		{
			BenchmarkID: "dense_op", Model: "opencode-gemma4-26b", Lang: "ailang",
			Trials: 3, Passed: 0, PassRate: 0.0,
			ErrorCategories: map[string]int{"compile_error": 3},
			TokensFailMean:  37000,
		},
		{
			BenchmarkID: "adt_option", Model: "opencode-gemma4-26b", Lang: "ailang",
			Trials: 3, Passed: 2, PassRate: 2.0 / 3.0,
			ErrorCategories: map[string]int{"logic_error": 1},
			TokensFailMean:  72000,
		},
		{
			BenchmarkID: "fizzbuzz", Model: "opencode-gemma4-26b", Lang: "ailang",
			Trials: 3, Passed: 3, PassRate: 1.0,
		},
	})

	candidates, err := findCandidates(rotationDir, 2, 3, 0.5)
	if err != nil {
		t.Fatalf("findCandidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %+v", len(candidates), candidates)
	}
	c := candidates[0]
	if c.BenchmarkID != "dense_op" {
		t.Errorf("benchmark = %q, want dense_op", c.BenchmarkID)
	}
	if c.ErrorCategory != "compile_error" {
		t.Errorf("error_category = %q, want compile_error", c.ErrorCategory)
	}
	if c.NFail != 3 || c.NTrials != 3 {
		t.Errorf("fails = %d/%d, want 3/3", c.NFail, c.NTrials)
	}
	if c.PassRate != 0.0 {
		t.Errorf("pass_rate = %v, want 0", c.PassRate)
	}
	if c.ExampleTokens != 37000 {
		t.Errorf("example_tokens = %d, want 37000 (from TokensFailMean)", c.ExampleTokens)
	}
}

// TestFindCandidates_AggregatesAcrossSlots: two rotation slots cover the
// same benchmark; counts must aggregate across them and the dominant
// failure category should still surface.
func TestFindCandidates_AggregatesAcrossSlots(t *testing.T) {
	rotationDir := t.TempDir()
	writeFixtureSummary(t, rotationDir, "0100_gemma4_smoke", []eval_harness.BenchmarkSummary{
		{
			BenchmarkID: "dense_op", Model: "m", Lang: "ailang",
			Trials: 3, Passed: 0, PassRate: 0.0,
			ErrorCategories: map[string]int{"compile_error": 3},
			TokensFailMean:  37000,
		},
	})
	writeFixtureSummary(t, rotationDir, "0200_gemma4_smoke", []eval_harness.BenchmarkSummary{
		{
			BenchmarkID: "dense_op", Model: "m", Lang: "ailang",
			Trials: 3, Passed: 0, PassRate: 0.0,
			ErrorCategories: map[string]int{"compile_error": 2, "logic_error": 1},
		},
	})

	candidates, err := findCandidates(rotationDir, 2, 3, 0.5)
	if err != nil {
		t.Fatalf("findCandidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].NTrials != 6 {
		t.Errorf("n_trials = %d, want 6 (aggregated)", candidates[0].NTrials)
	}
	if candidates[0].NFail != 5 {
		t.Errorf("n_fail = %d, want 5 (3+2 compile_error)", candidates[0].NFail)
	}
	if candidates[0].ErrorCategory != "compile_error" {
		t.Errorf("dominant category = %q, want compile_error", candidates[0].ErrorCategory)
	}
}

// TestFindCandidates_RespectsMinTrials: tuples with fewer than min-trials
// total trials are skipped (signal too weak).
func TestFindCandidates_RespectsMinTrials(t *testing.T) {
	rotationDir := t.TempDir()
	writeFixtureSummary(t, rotationDir, "single", []eval_harness.BenchmarkSummary{
		{
			BenchmarkID: "x", Model: "m", Lang: "ailang",
			Trials: 1, Passed: 0, PassRate: 0.0,
			ErrorCategories: map[string]int{"compile_error": 1},
		},
	})
	candidates, err := findCandidates(rotationDir, 1, 3, 0.5)
	if err != nil {
		t.Fatalf("findCandidates: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (trials < min-trials), got %d", len(candidates))
	}
}

// TestFindCandidates_NoSummaries: empty rotation dir reports a clear error
// (caller can show a helpful message about needing --trials N).
func TestFindCandidates_NoSummaries(t *testing.T) {
	rotationDir := t.TempDir()
	if _, err := findCandidates(rotationDir, 2, 3, 0.5); err == nil {
		t.Errorf("expected error for empty rotation dir, got nil")
	}
}

// TestFindCandidates_SortByPassRate: when multiple candidates qualify,
// worst-first (lowest pass rate) is the natural triage order.
func TestFindCandidates_SortByPassRate(t *testing.T) {
	rotationDir := t.TempDir()
	writeFixtureSummary(t, rotationDir, "slot", []eval_harness.BenchmarkSummary{
		{
			BenchmarkID: "bench-b", Model: "m", Lang: "ailang",
			Trials: 4, Passed: 1, PassRate: 0.25,
			ErrorCategories: map[string]int{"compile_error": 3},
		},
		{
			BenchmarkID: "bench-a", Model: "m", Lang: "ailang",
			Trials: 4, Passed: 0, PassRate: 0.0,
			ErrorCategories: map[string]int{"logic_error": 4},
		},
	})
	candidates, err := findCandidates(rotationDir, 2, 3, 0.5)
	if err != nil {
		t.Fatalf("findCandidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].BenchmarkID != "bench-a" {
		t.Errorf("first candidate = %s, want bench-a (lowest pass_rate)", candidates[0].BenchmarkID)
	}
}
