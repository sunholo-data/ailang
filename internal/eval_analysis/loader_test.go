package eval_analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeResultFile marshals a BenchmarkResult to <dir>/<name>.json.
func writeResultFile(t *testing.T, dir, name string, r *BenchmarkResult) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newResult(id, lang, model string, ts time.Time) *BenchmarkResult {
	return &BenchmarkResult{
		ID:        id,
		Lang:      lang,
		Model:     model,
		Seed:      1,
		StdoutOk:  true,
		EvalMode:  "agent",
		Timestamp: ts,
	}
}

// TestLoadResultsFromDirs_Merge verifies that eval-report --merge semantics
// hold at the loader level: results from multiple directories are combined,
// the merge is order-independent, and overlapping (identical) result slots
// dedupe to the newest.
func TestLoadResultsFromDirs_Merge(t *testing.T) {
	base := t.TempDir()
	cloudDir := filepath.Join(base, "cloud")
	localDir := filepath.Join(base, "local")

	now := time.Now()

	// Cloud baseline: one cloud model.
	writeResultFile(t, cloudDir, "fizzbuzz_ailang_claude", newResult("fizzbuzz", "ailang", "claude-sonnet-4-6", now))

	// Local rotation: a distinct on-device model, plus an OLDER copy of a
	// slot that also exists (newer) in cloud to exercise dedup.
	writeResultFile(t, localDir, "fizzbuzz_ailang_qwen", newResult("fizzbuzz", "ailang", "pi-qwen3-6-35b-a3b-mxfp8", now))
	writeResultFile(t, localDir, "fizzbuzz_ailang_claude_old", newResult("fizzbuzz", "ailang", "claude-sonnet-4-6", now.Add(-2*time.Hour)))

	// Load cloud first, then local.
	merged, err := LoadResultsFromDirs(cloudDir, localDir)
	if err != nil {
		t.Fatalf("LoadResultsFromDirs: %v", err)
	}

	// Expect exactly 2 results: one claude (newest wins over the old copy),
	// one qwen.
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged results, got %d", len(merged))
	}

	models := map[string]int{}
	for _, r := range merged {
		models[r.Model]++
	}
	if models["claude-sonnet-4-6"] != 1 {
		t.Errorf("expected 1 claude result after dedup, got %d", models["claude-sonnet-4-6"])
	}
	if models["pi-qwen3-6-35b-a3b-mxfp8"] != 1 {
		t.Errorf("expected local qwen model present, got %d", models["pi-qwen3-6-35b-a3b-mxfp8"])
	}

	// The surviving claude result must be the newer one.
	for _, r := range merged {
		if r.Model == "claude-sonnet-4-6" && !r.Timestamp.Equal(now) {
			t.Errorf("dedup kept the older claude result: %v", r.Timestamp)
		}
	}

	// Order independence: swapping the directory order yields the same set.
	mergedRev, err := LoadResultsFromDirs(localDir, cloudDir)
	if err != nil {
		t.Fatalf("LoadResultsFromDirs (reversed): %v", err)
	}
	if len(mergedRev) != len(merged) {
		t.Fatalf("merge is not order-independent: %d vs %d", len(mergedRev), len(merged))
	}
}

// TestLoadResultsFromDirs_OverlappingPaths ensures that passing the same
// directory (or a nested one) twice does not double-count files.
func TestLoadResultsFromDirs_OverlappingPaths(t *testing.T) {
	dir := t.TempDir()
	writeResultFile(t, dir, "a_ailang_m", newResult("a", "ailang", "m", time.Now()))
	writeResultFile(t, dir, "b_ailang_m", newResult("b", "ailang", "m", time.Now()))

	merged, err := LoadResultsFromDirs(dir, dir)
	if err != nil {
		t.Fatalf("LoadResultsFromDirs: %v", err)
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 results after path-dedup, got %d", len(merged))
	}
}

// TestLoadResults_SingleDirDelegates verifies the single-dir wrapper still works.
func TestLoadResults_SingleDirDelegates(t *testing.T) {
	dir := t.TempDir()
	writeResultFile(t, dir, "a_ailang_m", newResult("a", "ailang", "m", time.Now()))

	results, err := LoadResults(dir)
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestLoadResultsFromDirs_MissingDir(t *testing.T) {
	if _, err := LoadResultsFromDirs(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
