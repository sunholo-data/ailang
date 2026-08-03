package main

import "sync"

// budgetTracker accumulates banked trial cost against an aggregate ceiling
// (M-EVAL-STANDARD-CONFIDENCE-GATING). Safe for concurrent use — the same
// instance is shared across runBenchmarksParallel's worker goroutines via the
// onCost callback threaded into runSingleBenchmark.
type budgetTracker struct {
	mu       sync.Mutex
	limitUSD float64
	spentUSD float64
	exceeded bool
}

// newBudgetTracker returns a tracker for the given ceiling. limitUSD <= 0
// means "no cap" — Add always returns false and Exceeded is always false,
// preserving today's unconstrained behavior when --budget-usd is unset.
func newBudgetTracker(limitUSD float64) *budgetTracker {
	return &budgetTracker{limitUSD: limitUSD}
}

// Add records a trial's banked cost and returns true exactly once — the call
// whose running total first reaches or crosses the limit. Every other call
// (before or after that point) returns false, so callers can distinguish
// "just tripped, act on it" from "already tripped, nothing new to do".
func (b *budgetTracker) Add(costUSD float64) (justExceeded bool) {
	if b.limitUSD <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.spentUSD += costUSD
	if b.spentUSD >= b.limitUSD && !b.exceeded {
		b.exceeded = true
		return true
	}
	return false
}

// Exceeded reports whether the limit has been crossed.
func (b *budgetTracker) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exceeded
}

// Spent reports the running total recorded so far.
func (b *budgetTracker) Spent() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spentUSD
}
