package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWriteWallClockStoppedSentinel confirms the sentinel's shape. Its whole
// job is to make a partial run detectable as partial: without it, a night cut
// short by the cap looks identical to a complete one and the regression check
// reads the missing benchmarks as a result rather than an absence.
func TestWriteWallClockStoppedSentinel(t *testing.T) {
	dir := t.TempDir()
	writeWallClockStoppedSentinel(dir, 6*time.Hour, 14, 24)

	path := filepath.Join(dir, "wallclock_stopped.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("sentinel not written: %v", err)
	}
	var got wallClockStoppedSentinel
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("sentinel is not valid JSON: %v\n%s", err, data)
	}
	if got.MaxWallClock != "6h0m0s" {
		t.Errorf("max_wall_clock = %q, want 6h0m0s", got.MaxWallClock)
	}
	if got.JobsCompleted != 14 || got.JobsTotal != 24 {
		t.Errorf("jobs = %d/%d, want 14/24", got.JobsCompleted, got.JobsTotal)
	}
	if got.StoppedAt.IsZero() {
		t.Error("stopped_at must be set — it is how a reader ages the partial run")
	}
}

// An already-expired deadline must stop DISPATCH before any benchmark starts,
// and still leave the sentinel behind. This is the mechanism that bounds a
// night; it must not depend on a benchmark actually running.
func TestRunBenchmarksParallel_WallClockStopsDispatch(t *testing.T) {
	dir := t.TempDir()
	jobs := []Job{
		{Benchmark: "never_runs_a", Model: "test-model", Language: "ailang", Trial: 1},
		{Benchmark: "never_runs_b", Model: "test-model", Language: "ailang", Trial: 1},
	}

	// 1ns is already in the past by the time the dispatch loop reads it.
	results := runBenchmarksParallel(context.Background(), jobs, 42, dir,
		time.Second, 1, false, "", nil, "", nil, 0, time.Nanosecond)

	if len(results) != len(jobs) {
		t.Fatalf("results length = %d, want %d (the slice is pre-sized, not appended)", len(results), len(jobs))
	}
	// Nothing was dispatched, so every slot is still the zero value.
	for i, r := range results {
		if r.BenchmarkID != "" {
			t.Errorf("results[%d] was populated (%q) — a job ran past an expired deadline", i, r.BenchmarkID)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "wallclock_stopped.json"))
	if err != nil {
		t.Fatalf("expected a wallclock_stopped sentinel: %v", err)
	}
	if !strings.Contains(string(data), `"jobs_total": 2`) {
		t.Errorf("sentinel should record the full job count, got:\n%s", data)
	}
}

// 0 means no cap, and must not write a sentinel — every existing caller passes
// 0 and none of them are partial runs.
func TestRunBenchmarksParallel_ZeroWallClockIsNoCap(t *testing.T) {
	dir := t.TempDir()
	results := runBenchmarksParallel(context.Background(), nil, 42, dir,
		time.Second, 1, false, "", nil, "", nil, 0, 0)
	if len(results) != 0 {
		t.Errorf("results = %d, want 0", len(results))
	}
	if _, err := os.Stat(filepath.Join(dir, "wallclock_stopped.json")); !os.IsNotExist(err) {
		t.Error("no cap must not write a stopped sentinel")
	}
}
