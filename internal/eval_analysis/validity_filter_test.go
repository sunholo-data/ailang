package eval_analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// writeResult drops one result JSON into dir.
func writeResult(t *testing.T, dir, name string, r BenchmarkResult) {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestLoadResults_ExcludesInvalidByDefault is the behaviour that makes the
// measurement contract stick: excluding non-measurements must be what you get
// WITHOUT thinking about it. If callers had to opt in, the next analysis
// written in a hurry would silently include the garbage again.
func TestLoadResults_ExcludesInvalidByDefault(t *testing.T) {
	dir := t.TempDir()
	writeResult(t, dir, "legacy", BenchmarkResult{ID: "legacy", Lang: "ailang", Model: "m", StdoutOk: true})
	writeResult(t, dir, "good", BenchmarkResult{ID: "good", Lang: "ailang", Model: "m", StdoutOk: true, Validity: eval_harness.MarkValid()})
	writeResult(t, dir, "dead", BenchmarkResult{ID: "dead", Lang: "ailang", Model: "m", Validity: eval_harness.MarkInvalid(eval_harness.ReasonCanaryFailed)})

	results, err := LoadResults(dir)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}

	if len(results) != 2 {
		ids := []string{}
		for _, r := range results {
			ids = append(ids, r.ID)
		}
		t.Fatalf("LoadResults returned %d rows %v, want 2 (legacy + good); invalid rows must be excluded by default", len(results), ids)
	}
	for _, r := range results {
		if r.ID == "dead" {
			t.Error("canary-failed row must not appear in a default load")
		}
	}
}

// TestLoadResultsIncludingInvalid_OptsBackIn covers the --include-invalid
// escape hatch: the data is quarantined, never deleted, so it must remain
// reachable for anyone investigating the bug itself.
func TestLoadResultsIncludingInvalid_OptsBackIn(t *testing.T) {
	dir := t.TempDir()
	writeResult(t, dir, "good", BenchmarkResult{ID: "good", Lang: "ailang", Model: "m", StdoutOk: true})
	writeResult(t, dir, "dead", BenchmarkResult{ID: "dead", Lang: "ailang", Model: "m", Validity: eval_harness.MarkInvalid(eval_harness.ReasonZeroPassAll)})

	results, err := LoadResultsIncludingInvalid(dir)
	if err != nil {
		t.Fatalf("LoadResultsIncludingInvalid: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d rows, want both (quarantined data must stay reachable)", len(results))
	}
}

// TestLoadResults_LegacyRowsSurvive re-asserts the back-compat guarantee at the
// LOADER level, not just on the struct: a directory of pre-v0.31.0 results must
// load completely.
func TestLoadResults_LegacyRowsSurvive(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"a", "b", "c"} {
		writeResult(t, dir, id, BenchmarkResult{ID: id, Lang: "ailang", Model: "m", StdoutOk: true})
	}

	results, err := LoadResults(dir)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d legacy rows, want 3 — absent validity must never drop a row", len(results))
	}
}
