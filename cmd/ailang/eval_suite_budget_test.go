package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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
