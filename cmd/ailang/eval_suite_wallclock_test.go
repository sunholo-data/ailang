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

	// A NEGATIVE cap is a deadline already in the past — deterministically, on
	// every platform. The previous 1ns relied on the clock ticking between the
	// deadline being computed and first read, which holds on Linux and macOS and
	// does NOT on Windows, where the timer granularity is ~15.6ms: `now` and
	// `now+1ns` compared equal, the deadline read as still in the future, and
	// both jobs dispatched (CI, 2026-09-11). A timing-dependent assertion about
	// a timing mechanism is the one place not to rely on luck.
	results := runBenchmarksParallel(context.Background(), jobs, 42, dir,
		time.Second, 1, false, "", nil, "", nil, 0, -time.Second)

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

// A negative cap must mean "already expired", never "unlimited".
//
// `--max-wall-clock=-5m` is a plausible typo for a ceiling, and the old `> 0`
// test read it as no cap at all — silently removing the bound the operator was
// asking for. Unlimited is the one answer a negative ceiling cannot mean, and
// on the single-GPU rig an unbounded suite holds the lock all night.
func TestRunBenchmarksParallel_NegativeWallClockIsNotUnlimited(t *testing.T) {
	dir := t.TempDir()
	jobs := []Job{{Benchmark: "never_runs", Model: "test-model", Language: "ailang", Trial: 1}}

	results := runBenchmarksParallel(context.Background(), jobs, 42, dir,
		time.Second, 1, false, "", nil, "", nil, 0, -5*time.Minute)

	for i, r := range results {
		if r.BenchmarkID != "" {
			t.Errorf("results[%d] populated (%q) — a negative cap was treated as unlimited", i, r.BenchmarkID)
		}
	}
	if _, err := os.ReadFile(filepath.Join(dir, "wallclock_stopped.json")); err != nil {
		t.Errorf("a negative cap must stop dispatch AND leave the sentinel: %v", err)
	}
}
