package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestBudgetTracker_StubbedCostStream simulates a run's cost stream crossing
// the cap mid-run and asserts a graceful stop: the trip fires exactly once
// (at the trial that crosses the line, not before and not again after), and
// every trial's cost is still counted toward Spent() — nothing is discarded.
func TestBudgetTracker_StubbedCostStream(t *testing.T) {
	tracker := newBudgetTracker(1.00)

	// Stubbed per-trial costs, as if streamed in from a sequence of banked
	// results: 0.30 + 0.30 + 0.30 = 0.90 (under cap), + 0.30 = 1.20 (crosses
	// the $1.00 cap on this 4th call), + 0.30 = 1.50 (still over, but this
	// call must NOT re-trip).
	costs := []float64{0.30, 0.30, 0.30, 0.30, 0.30}
	var trips int
	for i, c := range costs {
		if tracker.Add(c) {
			trips++
			if i != 3 {
				t.Errorf("Add tripped at call %d, want call 3 (0-indexed) where the running total first reaches $1.00", i)
			}
		}
	}
	if trips != 1 {
		t.Errorf("tracker tripped %d times, want exactly 1 (graceful stop, not repeated aborts)", trips)
	}
	if !tracker.Exceeded() {
		t.Error("Exceeded() = false after the cap was crossed, want true")
	}
	wantSpent := 1.50
	if diff := tracker.Spent() - wantSpent; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("Spent() = %v, want %v — every trial's cost must still be counted, not discarded once tripped", tracker.Spent(), wantSpent)
	}
}

// TestBudgetTracker_NoCapNeverTrips confirms unset (<=0) budgetUSD preserves
// today's unconstrained behavior — the default when --budget-usd is omitted.
func TestBudgetTracker_NoCapNeverTrips(t *testing.T) {
	for _, limit := range []float64{0, -1} {
		tracker := newBudgetTracker(limit)
		for i := range 100 {
			if tracker.Add(1000.0) {
				t.Fatalf("limit=%v: Add tripped on call %d, want never (no cap means no cap)", limit, i)
			}
		}
		if tracker.Exceeded() {
			t.Errorf("limit=%v: Exceeded() = true, want false", limit)
		}
	}
}

// TestBudgetTracker_ConcurrentAddIsRace-safe exercises Add from many
// goroutines at once (mirroring runBenchmarksParallel's worker pool) and
// asserts the trip still fires exactly once despite concurrent callers.
func TestBudgetTracker_ConcurrentAdd(t *testing.T) {
	tracker := newBudgetTracker(5.00)
	var wg sync.WaitGroup
	var mu sync.Mutex
	trips := 0
	for range 50 {
		wg.Go(func() {
			if tracker.Add(0.20) { // 50 * 0.20 = $10.00 total, well past the $5 cap
				mu.Lock()
				trips++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if trips != 1 {
		t.Errorf("concurrent Add tripped %d times, want exactly 1", trips)
	}
	if diff := tracker.Spent() - 10.00; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("Spent() = %v, want 10.00 (all 50 calls counted despite the early trip)", tracker.Spent())
	}
}

// TestWriteBudgetStoppedSentinel confirms the sentinel file's shape — this is
// what run_eval_baseline.sh checks for after a Step 1 invocation to decide
// whether to continue the confidence-gated call sequence and to mark
// baseline.json as a partial run.
func TestWriteBudgetStoppedSentinel(t *testing.T) {
	dir := t.TempDir()
	writeBudgetStoppedSentinel(dir, 150.0, 152.34)

	path := filepath.Join(dir, "budget_stopped.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("sentinel not written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("sentinel file is empty")
	}
	// Loose content check — the exact schema is an implementation detail the
	// shell side doesn't need to parse field-by-field, just detect existence
	// and (optionally) the two numbers for a human-readable log line.
	for _, want := range []string{`"budget_usd": 150`, `"spent_usd": 152.34`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("sentinel content missing %q, got:\n%s", want, data)
		}
	}
}

// The budget must stop DISPATCH in a parallel suite, not just print that it did.
//
// Regression for 2026-11-05: `--budget-usd 8 --parallel 6` over 918 jobs
// tripped at run 609 and then started 309 more. The dispatch loop never blocks,
// so every job passed the pre-semaphore abort check at once and queued on the
// semaphore; after acquiring it, only the wall-clock deadline was re-checked.
// Here each trial banks $10 against an $8 cap, so the FIRST completion trips it.
// At --parallel 2 the second job may already be in flight (allowed: in-flight
// trials finish); a third must never start.
func TestRunBenchmarksParallel_BudgetStopsQueuedJobs(t *testing.T) {
	orig := runSingleBenchmarkFn
	t.Cleanup(func() { runSingleBenchmarkFn = orig })

	var mu sync.Mutex
	started := 0
	runSingleBenchmarkFn = func(_ context.Context, _, _, _, _ string, _ int, _ int64, _ string, _ time.Duration, _ bool, _ string, _ *eval_harness.AgentBenchmarkConfig, _ string, _ *EvalChainContext, onCost func(float64)) (bool, error) {
		mu.Lock()
		started++
		mu.Unlock()
		// Hold the slot long enough for the (non-blocking) dispatch loop to
		// spawn every job, so they are all QUEUED on the semaphore when the
		// budget trips — the shape of the real 918-job run. Without this an
		// instant stub trips the budget before the loop has spawned the rest,
		// the loop's own check stops them, and the test passes with the bug
		// present (verified by reverting the fix).
		time.Sleep(200 * time.Millisecond)
		onCost(10)
		return true, nil
	}

	const n = 10
	jobs := make([]Job, n)
	for i := range jobs {
		jobs[i] = Job{Benchmark: fmt.Sprintf("b%d", i), Model: "test-model", Language: "ailang", Trial: 1}
	}
	dir := t.TempDir()
	runBenchmarksParallel(context.Background(), jobs, 42, dir, time.Second, 2, false, "", nil, "", nil, 8, 0)

	if started > 2 {
		t.Errorf("%d of %d trials started after an $8 cap was crossed by the first $10 trial; want <= 2 (the in-flight pair)", started, n)
	}
	if _, err := os.Stat(filepath.Join(dir, "budget_stopped.json")); err != nil {
		t.Errorf("a tripped budget must leave the budget_stopped sentinel: %v", err)
	}
}
