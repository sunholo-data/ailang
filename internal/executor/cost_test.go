package executor

import (
	"sync"
	"testing"
)

// TestCostBudget_AddAndCurrent verifies basic accumulation across multiple
// Add() calls with mixed input+output token deltas.
func TestCostBudget_AddAndCurrent(t *testing.T) {
	tests := []struct {
		name        string
		maxUSD      float64
		inputPer1K  float64
		outputPer1K float64
		adds        [][2]int // [input, output] pairs
		wantFinal   float64
		wantExceed  bool
	}{
		{
			name:        "single add under budget",
			maxUSD:      1.0,
			inputPer1K:  0.001,
			outputPer1K: 0.002,
			adds:        [][2]int{{1000, 500}},
			// 1000/1000 * 0.001 + 500/1000 * 0.002 = 0.001 + 0.001 = 0.002
			wantFinal:  0.002,
			wantExceed: false,
		},
		{
			name:        "accumulating adds reach exactly MaxUSD",
			maxUSD:      0.005,
			inputPer1K:  0.001,
			outputPer1K: 0.002,
			adds:        [][2]int{{1000, 500}, {1000, 500}, {500, 250}},
			// total: 2500/1000 * 0.001 + 1250/1000 * 0.002 = 0.0025 + 0.0025 = 0.005
			wantFinal:  0.005,
			wantExceed: true, // >= MaxUSD triggers exceed
		},
		{
			name:        "MaxUSD=0 disables enforcement",
			maxUSD:      0,
			inputPer1K:  0.001,
			outputPer1K: 0.002,
			adds:        [][2]int{{100000, 50000}},
			// huge cost but no enforcement
			wantFinal:  0.2, // 100 * 0.001 + 50 * 0.002 = 0.1 + 0.1 = 0.2
			wantExceed: false,
		},
		{
			name:        "expensive output dominates",
			maxUSD:      1.0,
			inputPer1K:  0.0006,  // GLM 5 input
			outputPer1K: 0.00208, // GLM 5 output
			adds:        [][2]int{{30000, 10000}},
			// 30 * 0.0006 + 10 * 0.00208 = 0.018 + 0.0208 = 0.0388
			wantFinal:  0.0388,
			wantExceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewCostBudget(tt.maxUSD, tt.inputPer1K, tt.outputPer1K)
			var lastExceed bool
			for _, add := range tt.adds {
				_, lastExceed = b.Add(add[0], add[1])
			}
			got := b.Current()
			if !approxEqual(got, tt.wantFinal, 1e-9) {
				t.Errorf("Current() = %v, want %v", got, tt.wantFinal)
			}
			if lastExceed != tt.wantExceed {
				t.Errorf("last Add().exceeded = %v, want %v", lastExceed, tt.wantExceed)
			}
		})
	}
}

// TestCostBudget_KilledAtFirstWriteWins ensures concurrent exceedences
// converge on a single recorded killed-at value (the first writer wins).
func TestCostBudget_KilledAtFirstWriteWins(t *testing.T) {
	b := NewCostBudget(0.001, 0.001, 0.0)
	// Each Add(1000, 0) costs $0.001 — exactly at the budget.
	// 100 goroutines each calling Add will all see exceeded.

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Add(1000, 0)
		}()
	}
	wg.Wait()

	killed := b.KilledAt()
	if killed == 0 {
		t.Fatal("KilledAt() = 0, expected non-zero after 100 concurrent exceedences")
	}
	// Final cost should be 100 * $0.001 = $0.1, but killedAt locks in the
	// first observation (which could be anywhere from $0.001 to $0.1).
	if killed < 0.001 {
		t.Errorf("KilledAt() = %v, expected ≥ 0.001 (first exceedence)", killed)
	}
	if killed > 0.1+1e-9 {
		t.Errorf("KilledAt() = %v, expected ≤ 0.1 (max possible total)", killed)
	}
	// And it should NOT change on subsequent reads.
	first := killed
	second := b.KilledAt()
	if first != second {
		t.Errorf("KilledAt() not stable: first=%v second=%v", first, second)
	}
}

// TestCostBudget_RaceSafety hammers Add/Current/KilledAt from concurrent
// goroutines. Run with `go test -race` to verify no data races.
func TestCostBudget_RaceSafety(t *testing.T) {
	b := NewCostBudget(10.0, 0.001, 0.002)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = b.Add(10, 5)
				_ = b.Current()
				_ = b.KilledAt()
			}
		}()
	}
	wg.Wait()

	// 50 goroutines × 100 iterations × Add(10,5) = 50000 input + 25000 output
	// cost = 50000/1000 * 0.001 + 25000/1000 * 0.002 = 0.05 + 0.05 = 0.10
	got := b.Current()
	want := 0.10
	if !approxEqual(got, want, 1e-9) {
		t.Errorf("Current() after concurrent adds = %v, want %v", got, want)
	}
}

// TestCostBudget_NilSafe verifies all methods handle nil receivers
// gracefully (used by call sites that conditionally pass a budget).
func TestCostBudget_NilSafe(t *testing.T) {
	var b *CostBudget // nil

	cur, exceeded := b.Add(1000, 500)
	if cur != 0 || exceeded {
		t.Errorf("nil.Add() = (%v, %v), want (0, false)", cur, exceeded)
	}
	if b.Current() != 0 {
		t.Errorf("nil.Current() != 0")
	}
	if b.KilledAt() != 0 {
		t.Errorf("nil.KilledAt() != 0")
	}
}

// TestDefaultMaxCostUSD verifies the per-model budget fallback formula.
func TestDefaultMaxCostUSD(t *testing.T) {
	tests := []struct {
		name        string
		inputPer1K  float64
		outputPer1K float64
		want        float64
	}{
		{
			// or-minimax-m2-7: $0.30 / $1.20 per 1M = $0.0003 / $0.0012 per 1K
			// formula = 0.0003 * 64 + 0.0012 * 32 = 0.0192 + 0.0384 = 0.0576
			// < $0.50 → returns formula
			name:        "cheap model: or-minimax-m2-7",
			inputPer1K:  0.0003,
			outputPer1K: 0.0012,
			want:        0.0576,
		},
		{
			// claude-opus-4-7: $15 / $75 per 1M = $0.015 / $0.075 per 1K
			// formula = 0.015 * 64 + 0.075 * 32 = 0.96 + 2.4 = 3.36
			// > $0.50 → clipped to $0.50
			name:        "priciest model: claude-opus-4-7",
			inputPer1K:  0.015,
			outputPer1K: 0.075,
			want:        0.50,
		},
		{
			// or-glm-5: $0.60 / $2.08 per 1M
			// formula = 0.0006 * 64 + 0.00208 * 32 = 0.0384 + 0.06656 = 0.10496
			// < $0.50 → returns formula
			name:        "mid-tier OS: or-glm-5",
			inputPer1K:  0.0006,
			outputPer1K: 0.00208,
			want:        0.10496,
		},
		{
			name:        "free local model (zero pricing)",
			inputPer1K:  0,
			outputPer1K: 0,
			want:        0, // formula = 0, < ceiling, returns 0
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultMaxCostUSD(tt.inputPer1K, tt.outputPer1K)
			if !approxEqual(got, tt.want, 1e-9) {
				t.Errorf("DefaultMaxCostUSD(%v, %v) = %v, want %v",
					tt.inputPer1K, tt.outputPer1K, got, tt.want)
			}
		})
	}
}

// TestCostBudget_AddReportsCorrectCurrent ensures the (current, exceeded)
// return value matches what Current() reports immediately after.
func TestCostBudget_AddReportsCorrectCurrent(t *testing.T) {
	b := NewCostBudget(1.0, 0.001, 0.002)
	for i := 0; i < 10; i++ {
		fromAdd, _ := b.Add(100, 50)
		fromCurrent := b.Current()
		if !approxEqual(fromAdd, fromCurrent, 1e-9) {
			t.Errorf("iter %d: Add() reported %v, Current() reported %v", i, fromAdd, fromCurrent)
		}
	}
}

func approxEqual(a, b, tol float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < tol
}
