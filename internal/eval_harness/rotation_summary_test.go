package eval_harness

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixtureMetrics writes a RunMetrics JSON to the eval-suite's standard layout.
// outputDir/agent/<id>[_trialN]_<lang>_<model>_<ts>.json
func fixtureMetrics(t *testing.T, outputDir, id, lang, model string, trial int,
	pass bool, totalTokens int, errCat string,
) {
	t.Helper()
	m := RunMetrics{
		ID:            id,
		Lang:          lang,
		Model:         model,
		EvalMode:      EvalModeAgent,
		Trial:         trial,
		TotalTokens:   totalTokens,
		CompileOk:     pass,
		RuntimeOk:     pass,
		StdoutOk:      pass,
		ErrorCategory: errCat,
		Timestamp:     time.Now().Add(time.Duration(trial) * time.Second),
	}
	logger := NewMetricsLogger(outputDir)
	if err := logger.Log(&m); err != nil {
		t.Fatalf("fixture log: %v", err)
	}
}

// TestSummarizeRotation_PassRate covers the headline aggregation:
// given 3 trials with 2 PASS + 1 FAIL, the summary must report
// pass_rate=2/3 and split token statistics by outcome.
func TestSummarizeRotation_PassRate(t *testing.T) {
	out := t.TempDir()
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "opencode-gemma4-26b", 1, true, 110000, "")
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "opencode-gemma4-26b", 2, true, 120000, "")
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "opencode-gemma4-26b", 3, false, 1800000, "compile_error")

	rs, err := SummarizeRotation(out)
	if err != nil {
		t.Fatalf("SummarizeRotation: %v", err)
	}
	if len(rs.BenchmarkSummary) != 1 {
		t.Fatalf("expected 1 benchmark summary, got %d", len(rs.BenchmarkSummary))
	}
	bs := rs.BenchmarkSummary[0]
	if bs.BenchmarkID != "fizzbuzz" {
		t.Errorf("benchmark_id = %q, want fizzbuzz", bs.BenchmarkID)
	}
	if bs.Trials != 3 {
		t.Errorf("trials = %d, want 3", bs.Trials)
	}
	if bs.Passed != 2 {
		t.Errorf("passed = %d, want 2", bs.Passed)
	}
	if math.Abs(bs.PassRate-2.0/3.0) > 1e-9 {
		t.Errorf("pass_rate = %v, want 2/3", bs.PassRate)
	}
	if math.Abs(bs.TokensPassMean-115000.0) > 0.5 {
		t.Errorf("tokens_pass_mean = %v, want 115000", bs.TokensPassMean)
	}
	if bs.TokensFailMean != 1_800_000 {
		t.Errorf("tokens_fail_mean = %v, want 1800000", bs.TokensFailMean)
	}
	if bs.ErrorCategories["compile_error"] != 1 {
		t.Errorf("error_categories[compile_error] = %d, want 1", bs.ErrorCategories["compile_error"])
	}
	if rs.TrialsPerBench != 3 {
		t.Errorf("rotation TrialsPerBench = %d, want 3", rs.TrialsPerBench)
	}
}

// TestSummarizeRotation_ThrashAbortCount checks Phase-1 integration:
// trials aborted via thrash_aborted contribute to a dedicated ThrashAborts
// counter so candidates triage can flag them as infrastructure-class.
func TestSummarizeRotation_ThrashAbortCount(t *testing.T) {
	out := t.TempDir()
	fixtureMetrics(t, out, "dense_op", "ailang", "model-x", 1, false, 500000, "thrash_aborted")
	fixtureMetrics(t, out, "dense_op", "ailang", "model-x", 2, false, 510000, "thrash_aborted")
	fixtureMetrics(t, out, "dense_op", "ailang", "model-x", 3, true, 200000, "")

	rs, err := SummarizeRotation(out)
	if err != nil {
		t.Fatalf("SummarizeRotation: %v", err)
	}
	if len(rs.BenchmarkSummary) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(rs.BenchmarkSummary))
	}
	bs := rs.BenchmarkSummary[0]
	if bs.ThrashAborts != 2 {
		t.Errorf("thrash_aborts = %d, want 2", bs.ThrashAborts)
	}
	if bs.ErrorCategories["thrash_aborted"] != 2 {
		t.Errorf("error_categories[thrash_aborted] = %d, want 2", bs.ErrorCategories["thrash_aborted"])
	}
	if bs.Passed != 1 {
		t.Errorf("passed = %d, want 1", bs.Passed)
	}
}

// TestSummarizeRotation_MultipleBenchmarks ensures grouping by (id, model, lang)
// works and the output is stably sorted.
func TestSummarizeRotation_MultipleBenchmarks(t *testing.T) {
	out := t.TempDir()
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "model-a", 1, true, 100000, "")
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "model-b", 1, false, 50000, "logic_error")
	fixtureMetrics(t, out, "adt_option", "ailang", "model-a", 1, true, 90000, "")
	fixtureMetrics(t, out, "adt_option", "ailang", "model-b", 1, true, 95000, "")

	rs, err := SummarizeRotation(out)
	if err != nil {
		t.Fatalf("SummarizeRotation: %v", err)
	}
	if len(rs.BenchmarkSummary) != 4 {
		t.Fatalf("expected 4 summaries (2 benchmarks x 2 models), got %d", len(rs.BenchmarkSummary))
	}
	// Stable sort: adt_option first, then fizzbuzz; within each, model-a first.
	if rs.BenchmarkSummary[0].BenchmarkID != "adt_option" || rs.BenchmarkSummary[0].Model != "model-a" {
		t.Errorf("first summary = %+v, want adt_option/model-a", rs.BenchmarkSummary[0])
	}
	if rs.BenchmarkSummary[3].BenchmarkID != "fizzbuzz" || rs.BenchmarkSummary[3].Model != "model-b" {
		t.Errorf("last summary = %+v, want fizzbuzz/model-b", rs.BenchmarkSummary[3])
	}
}

// TestSummarizeRotation_WritesSummaryJSON verifies the on-disk artifact.
func TestSummarizeRotation_WritesSummaryJSON(t *testing.T) {
	out := t.TempDir()
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "m", 1, true, 100, "")
	if _, err := SummarizeRotation(out); err != nil {
		t.Fatalf("SummarizeRotation: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(out, "summary.json"))
	if err != nil {
		t.Fatalf("summary.json not written: %v", err)
	}
	var rs RotationSummary
	if err := json.Unmarshal(data, &rs); err != nil {
		t.Fatalf("malformed summary.json: %v", err)
	}
	if len(rs.BenchmarkSummary) != 1 {
		t.Errorf("on-disk summary has %d benchmarks, want 1", len(rs.BenchmarkSummary))
	}
	if rs.OutputDir != out {
		t.Errorf("output_dir = %q, want %q", rs.OutputDir, out)
	}
}

// TestRunMetrics_FilenameIncludesTrialWhenGreaterThanOne is a back-stop for
// the file-naming change in Log: trial>1 must produce a distinct filename so
// multi-trial runs don't clobber each other.
func TestRunMetrics_FilenameIncludesTrialWhenGreaterThanOne(t *testing.T) {
	out := t.TempDir()
	// Trial 1 (default shape)
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "m", 1, true, 100, "")
	// Trial 3 (trial-shaped filename)
	fixtureMetrics(t, out, "fizzbuzz", "ailang", "m", 3, true, 100, "")

	files, err := filepath.Glob(filepath.Join(out, "agent", "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 result files, got %d", len(files))
	}
	hasTrialN := false
	for _, f := range files {
		if filepath.Base(f) != "" && containsTrialSuffix(f) {
			hasTrialN = true
		}
	}
	if !hasTrialN {
		t.Errorf("no file matched *trial3* pattern; got %v", files)
	}
}

// containsTrialSuffix is a small helper kept inline so this test does not
// depend on production code beyond the public Log behavior.
func containsTrialSuffix(p string) bool {
	base := filepath.Base(p)
	return len(base) > 7 && (base[len("fizzbuzz_trial3_"):][0] == 'a' || true) && // simple substring check below
		(filepath.Ext(base) == ".json") &&
		(stringContains(base, "_trial3_"))
}
func stringContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
