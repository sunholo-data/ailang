package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// bankRow writes a row the way the suite does — through MetricsLogger.Log, so
// the validity backstop applies exactly as it does in production.
func bankRow(t *testing.T, outputDir string, m RunMetrics) {
	t.Helper()
	if m.Lang == "" {
		m.Lang = "ailang"
	}
	if m.Model == "" {
		m.Model = "fixture-model"
	}
	if m.EvalMode == "" {
		m.EvalMode = EvalModeStandard
	}
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if err := NewMetricsLogger(outputDir).Log(&m); err != nil {
		t.Fatalf("bank row: %v", err)
	}
}

// rawRow writes a row's JSON directly — the pre-v0.31.0 shape with no
// validity field and no backstop.
func rawRow(t *testing.T, path string, m RunMetrics) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPassed_RuntimeFailureWithMatchingStdoutIsNotAPass pins ruling D2: a row
// whose stdout matched but whose program did not run to completion is a
// FAILURE. Standard mode banks exactly this shape (repair.go grades stdout
// without gating on runtime_ok; 30 of the 31 discordant rows in the baselines
// are compile/runtime failures on a benchmark whose expected stdout is empty).
func TestPassed_RuntimeFailureWithMatchingStdoutIsNotAPass(t *testing.T) {
	cases := []struct {
		name                           string
		compile, runtime, stdout, want bool
	}{
		{"all three", true, true, true, true},
		{"stdout matched but runtime failed", true, false, true, false},
		{"stdout matched but compile failed", false, false, true, false},
		{"ran but stdout mismatched", true, true, false, false},
		{"nothing", false, false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := RunMetrics{CompileOk: c.compile, RuntimeOk: c.runtime, StdoutOk: c.stdout}
			if got := m.Passed(); got != c.want {
				t.Errorf("Passed() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestLoadRows_DefaultExcludesInvalid_OptionIncludes: a row that failed for a
// reason the harness could not identify (error_category=api_error) is marked
// invalid at bank time and MUST be excluded by the default loader — that is
// what stops a harness crash being analysed as a model failure — and included
// only when a caller opts in.
func TestLoadRows_DefaultExcludesInvalid_OptionIncludes(t *testing.T) {
	dir := t.TempDir()
	bankRow(t, dir, RunMetrics{ID: "fizzbuzz", CompileOk: true, RuntimeOk: true, StdoutOk: true, ErrorCategory: ErrorCategoryNone})
	bankRow(t, dir, RunMetrics{ID: "json_parse", ErrorCategory: ErrorCategoryAPI, Stderr: "motoko terminated without emitting run_summary"})
	bankRow(t, dir, RunMetrics{ID: "tree_walk", ErrorCategory: ErrorCategoryCompile}) // a real model failure

	rows, stats, err := LoadRows([]string{dir}, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadRows: %v", err)
	}
	if stats.Files != 3 || stats.Loaded != 2 || stats.Invalid != 1 {
		t.Errorf("stats = %+v, want files=3 loaded=2 invalid=1", stats)
	}
	if stats.InvalidBy[ReasonHarnessError] != 1 {
		t.Errorf("InvalidBy = %v, want harness_error=1", stats.InvalidBy)
	}
	for _, r := range rows {
		if r.ID == "json_parse" {
			t.Errorf("default load returned the invalid api_error row")
		}
	}

	all, stats, err := LoadRows([]string{dir}, LoadOptions{IncludeInvalid: true})
	if err != nil {
		t.Fatalf("LoadRows(IncludeInvalid): %v", err)
	}
	if len(all) != 3 || stats.Invalid != 0 {
		t.Errorf("IncludeInvalid: got %d rows (invalid=%d), want 3 (0)", len(all), stats.Invalid)
	}
}

// TestLoadRows_PreValidityRowsStayValid: a raw api_error row WITHOUT a validity
// field (every row banked before v0.31.0) is still loaded. Absent means valid;
// the loader does not re-apply the bank-time backstop to history, or the whole
// pre-v0.31.0 api_error population would vanish from every trend.
func TestLoadRows_PreValidityRowsStayValid(t *testing.T) {
	dir := t.TempDir()
	rawRow(t, filepath.Join(dir, "old.json"), RunMetrics{ID: "b", Lang: "ailang", Model: "m", ErrorCategory: ErrorCategoryAPI, Timestamp: time.Now()})
	rows, stats, err := LoadRows([]string{dir}, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || stats.Invalid != 0 {
		t.Errorf("got %d rows, invalid=%d; a validity-less api_error row must load", len(rows), stats.Invalid)
	}
}

// TestLoadRows_DedupNewestWins_KeepDuplicatesOff: re-runs of one slot collapse
// to the newest row by default; a cost tally opts out with KeepDuplicates.
func TestLoadRows_DedupNewestWins_KeepDuplicatesOff(t *testing.T) {
	dir := t.TempDir()
	older := time.Now().Add(-time.Hour)
	bankRow(t, dir, RunMetrics{ID: "b", Seed: 1, CompileOk: true, RuntimeOk: true, StdoutOk: false, Timestamp: older, CostUSD: 1})
	bankRow(t, dir, RunMetrics{ID: "b", Seed: 1, CompileOk: true, RuntimeOk: true, StdoutOk: true, Timestamp: older.Add(time.Minute), CostUSD: 1})
	// Same benchmark in AGENT mode is a distinct slot and must survive.
	bankRow(t, dir, RunMetrics{ID: "b", Seed: 1, EvalMode: EvalModeAgent, CompileOk: true, RuntimeOk: true, StdoutOk: true, Timestamp: older, CostUSD: 1})

	rows, stats, err := LoadRows([]string{dir}, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || stats.Duplicates != 1 {
		t.Fatalf("got %d rows, dup=%d; want 2 rows (one per mode), 1 duplicate dropped", len(rows), stats.Duplicates)
	}
	if !rows[0].Timestamp.After(rows[1].Timestamp) && !rows[0].Timestamp.Equal(rows[1].Timestamp) {
		t.Errorf("rows not newest-first")
	}
	for _, r := range rows {
		if r.EvalMode == EvalModeStandard && !r.Passed() {
			t.Errorf("standard slot kept the OLDER (failing) row; newest must win")
		}
	}

	all, stats, err := LoadRows([]string{dir}, LoadOptions{KeepDuplicates: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || stats.Duplicates != 0 {
		t.Errorf("KeepDuplicates: got %d rows dup=%d, want 3 / 0 — a paid-for re-run is still spend", len(all), stats.Duplicates)
	}
}

// TestLoadRows_SuiteLayoutStopsAtTheModeDirs: with SuiteLayout, a per-version
// bank dir nested under a root accumulator is NOT pooled in (the 2026-07-20
// misattribution) while a condition dir under a mode dir is; unbounded, the
// whole tree loads.
func TestLoadRows_SuiteLayoutStopsAtTheModeDirs(t *testing.T) {
	root := t.TempDir()
	bankRow(t, root, RunMetrics{ID: "top", CompileOk: true, RuntimeOk: true, StdoutOk: true})                              // root/standard/top.json
	bankRow(t, root, RunMetrics{ID: "cond", Condition: "z3", CompileOk: true, RuntimeOk: true, StdoutOk: true})            // root/standard/z3/cond.json
	bankRow(t, filepath.Join(root, "v0.32.0"), RunMetrics{ID: "nested", CompileOk: true, RuntimeOk: true, StdoutOk: true}) // root/v0.32.0/standard/nested.json

	bounded, _, err := LoadRows([]string{root}, LoadOptions{SuiteLayout: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(bounded) != 2 {
		t.Errorf("SuiteLayout loaded %d rows, want 2 (the version bank dir must not pool in)", len(bounded))
	}
	for _, r := range bounded {
		if r.ID == "nested" {
			t.Errorf("bounded walk reached the nested version dir")
		}
	}
	unbounded, _, err := LoadRows([]string{root}, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(unbounded) != 3 {
		t.Errorf("unbounded walk loaded %d rows, want 3", len(unbounded))
	}
}

// TestLoadRows_SkipsNonRows: summary.json / baseline.json beside the rows, a
// manifest-shaped JSON without id/lang/model, and a corrupt file are never
// counted as runs; they surface in ParseErrors (or are skipped by name) and
// the good rows still load.
func TestLoadRows_SkipsNonRows(t *testing.T) {
	dir := t.TempDir()
	bankRow(t, dir, RunMetrics{ID: "good", CompileOk: true, RuntimeOk: true, StdoutOk: true})
	for _, name := range []string{"summary.json", "baseline.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(`{"version":"x","model":"m"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"ailang_version":"v0.38.8"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "standard", "corrupt.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	rows, stats, err := LoadRows([]string{dir}, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "good" {
		t.Errorf("got %d rows (%v), want just the good one", len(rows), rows)
	}
	if stats.Files != 3 { // good + manifest + corrupt; summary/baseline skipped by name
		t.Errorf("Files = %d, want 3", stats.Files)
	}
	if len(stats.ParseErrors) != 2 {
		t.Errorf("ParseErrors = %v, want the manifest and the corrupt file", stats.ParseErrors)
	}

	if _, _, err := LoadRows([]string{filepath.Join(dir, "does-not-exist")}, LoadOptions{}); err == nil {
		t.Error("a missing directory must be an error, not an empty result")
	}
}
