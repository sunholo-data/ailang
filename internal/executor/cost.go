package executor

import (
	"math"
	"sync/atomic"
)

// CostBudget enforces a hard USD ceiling on a single Task execution.
//
// Each Executor adds incremental token counts at its natural event boundary
// (claude: usage event, opencode: tool_result event, etc.). The kill-on-exceed
// loop lives here so the 5 executors don't duplicate it.
//
// Wall-clock Timeout is a *safety net* for hung connections; cost is the
// primary gate that prevents runaway expensive sessions while letting
// cheap-but-slow models complete fairly.
//
// Zero value behaviour: a *CostBudget with MaxUSD == 0 never reports
// exceeded — used by callers that want speed instrumentation but no cost
// enforcement (legacy back-compat path).
//
// Thread safety: Add/Current/KilledAt are safe under -race when called from
// multiple goroutines. KilledAt() is first-write-wins via atomic CAS.
type CostBudget struct {
	MaxUSD      float64
	InputPer1K  float64
	OutputPer1K float64

	inputTokens  atomic.Int64
	outputTokens atomic.Int64
	killedAt     atomic.Uint64 // bit-cast float64 — 0 == not killed
}

// NewCostBudget constructs a budget with the given ceiling and per-1k pricing.
// Pass MaxUSD=0 to disable cost enforcement (back-compat).
func NewCostBudget(maxUSD, inputPer1K, outputPer1K float64) *CostBudget {
	return &CostBudget{
		MaxUSD:      maxUSD,
		InputPer1K:  inputPer1K,
		OutputPer1K: outputPer1K,
	}
}

// Add records additional input/output tokens and reports the running total.
// If the running total exceeds MaxUSD, exceeded == true and the executor
// should stop reading the stream and return Result{CostKilledAt: current}.
//
// When MaxUSD == 0, exceeded is always false — the budget acts as a
// passive token tally.
func (b *CostBudget) Add(inputDelta, outputDelta int) (current float64, exceeded bool) {
	if b == nil {
		return 0, false
	}
	in := b.inputTokens.Add(int64(inputDelta))
	out := b.outputTokens.Add(int64(outputDelta))
	current = float64(in)/1000.0*b.InputPer1K + float64(out)/1000.0*b.OutputPer1K

	if b.MaxUSD <= 0 {
		return current, false
	}
	if current >= b.MaxUSD {
		// First-write-wins: store current cost only if not already set.
		// uint64 backs the float64 so multiple racing exceedences converge
		// on a single recorded "killed at" value.
		b.killedAt.CompareAndSwap(0, math.Float64bits(current))
		return current, true
	}
	return current, false
}

// Current returns the running cost without modifying counters.
// Safe to call from any goroutine.
func (b *CostBudget) Current() float64 {
	if b == nil {
		return 0
	}
	in := b.inputTokens.Load()
	out := b.outputTokens.Load()
	return float64(in)/1000.0*b.InputPer1K + float64(out)/1000.0*b.OutputPer1K
}

// KilledAt returns the cost at which the budget was first exceeded, or 0
// if not exceeded. Useful to populate Result.CostKilledAt without a second
// Add() call.
func (b *CostBudget) KilledAt() float64 {
	if b == nil {
		return 0
	}
	bits := b.killedAt.Load()
	if bits == 0 {
		return 0
	}
	return math.Float64frombits(bits)
}

// DefaultMaxCostUSD computes the fallback budget used when models.yml
// omits a per-model `budgets:` block.
//
// Formula: min($0.50, input_per_1k × 64 + output_per_1k × 32)
//
// Rationale: a 64K-input × 32K-output benchmark is the upper bound of
// realistic agent loops; cap at $0.50 so flagship models don't get
// generous-by-accident budgets.
func DefaultMaxCostUSD(inputPer1K, outputPer1K float64) float64 {
	formula := inputPer1K*64.0 + outputPer1K*32.0
	const ceilingUSD = 0.50
	if formula < ceilingUSD {
		return formula
	}
	return ceilingUSD
}
