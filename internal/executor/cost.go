package executor

import (
	"github.com/sunholo-data/ailang/internal/modelreg"
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

// CostProvenance records whether a Result's CostUSD is money anyone was
// actually charged. Without it, a subscription run and a metered run are
// indistinguishable once both land in the same column — and the v1.0
// cost-per-verified-success KPI, whose numerator is "attributable METERED
// dollars", silently aggregates both.
//
// This is orthogonal to accuracy. A list-price-equivalent figure can be
// perfectly computed and still represent zero spend.
type CostProvenance string

const (
	// CostMetered: the account is genuinely charged per token (API key,
	// OpenRouter credits, Vertex ADC). The only provenance admissible in a
	// metered-dollars KPI.
	CostMetered CostProvenance = "metered"
	// CostListPriceEquivalent: the figure is real arithmetic over real tokens,
	// but the run went through a subscription/OAuth lane and was never billed.
	// Reproducible and comparable; just not spend.
	CostListPriceEquivalent CostProvenance = "list-price-equivalent"
	// CostFreeLocal: an on-device model with zero marginal token cost.
	// Distinct from a $0 metered figure, which would mean "billed nothing".
	CostFreeLocal CostProvenance = "free-local"
	// CostProvenanceUnknown: the auth lane could not be determined. Surfaced
	// rather than assumed — guessing "metered" is what this type exists to stop.
	CostProvenanceUnknown CostProvenance = "unknown"
)

// AuthLane is an executor's determination of how the current run authenticates.
type AuthLane int

const (
	// AuthLaneUnknown: could not be determined; do not assume either way.
	AuthLaneUnknown AuthLane = iota
	// AuthLaneBilled: per-token charges land on an account.
	AuthLaneBilled
	// AuthLaneSubscription: a seat/plan covers the run; no per-token charge.
	AuthLaneSubscription
)

// ResolveCostProvenance classifies a task's cost given the executor's auth lane.
//
// Zero resolved rates win over the lane: a local Ollama model is free-local
// whatever the auth story, because there is no per-token charge to attribute.
func ResolveCostProvenance(task *Task, lane AuthLane) CostProvenance {
	if task != nil && task.Pricing != nil &&
		task.Pricing.InputTokenCost == 0 && task.Pricing.OutputTokenCost == 0 {
		return CostFreeLocal
	}
	switch lane {
	case AuthLaneBilled:
		return CostMetered
	case AuthLaneSubscription:
		return CostListPriceEquivalent
	default:
		return CostProvenanceUnknown
	}
}

// ResolveCostModel picks the pricing an executor should bill a task at.
//
// Task.Pricing (the per-model rates from models.yml) wins whenever it is
// present; fallback is the executor's own CostModel(), which names a single
// default model and is therefore only correct when that is what actually ran.
//
// A present-but-zero Task.Pricing is honoured, not treated as missing: local
// Ollama models are genuinely free, and falling back to a cloud price table
// for them would invent spend that never happened.
func ResolveCostModel(task *Task, fallback *CostModel) *CostModel {
	if task != nil && task.Pricing != nil {
		return task.Pricing
	}
	return fallback
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

// IsOllamaCloudRoute reports whether a model name selects an Ollama CLOUD model
// rather than an on-device one.
//
// Ollama marks the route with a name suffix parsed as a pure string by its own
// daemon: an untagged model takes ":cloud" (kimi-k3:cloud) and a tagged one
// appends "-cloud" to the TAG (deepseek-v4-flash:0731-cloud). Mirroring that
// grammar rather than inventing one means the two cannot disagree about which
// route a request takes.
//
// Lives in this package because it is needed BOTH for cost provenance (here)
// and for GPU-contention decisions in eval_harness, which imports executor.
// M-MODEL-REGISTRY-SINGLE-SOURCE M1 (D4(a)): the grammar moved to
// internal/modelreg so the registry could become a leaf that executors are
// free to import. This delegate keeps executor's callers and its exported API
// unchanged; there is one implementation, not two.
func IsOllamaCloudRoute(name string) bool { return modelreg.IsOllamaCloudRoute(name) }

// AuthLaneForModel decides how a run authenticates, from the model name alone.
//
// D1 (Mark-ratified 2026-08-26): an Ollama Cloud row is a SUBSCRIPTION lane —
// a flat plan covers it and no per-token charge lands anywhere. Its prices in
// models.yml are IMPUTED from an equivalent metered route so the figure is
// comparable arithmetic over real tokens, which is exactly what
// CostListPriceEquivalent means ("went through a subscription/OAuth lane and
// was never billed"). Reporting these rows as `metered` would claim a spend
// that never happened.
//
// Everything else stays AuthLaneBilled, which is what the executors asserted
// unconditionally before this existed.
func AuthLaneForModel(model string) AuthLane {
	if IsOllamaCloudRoute(model) {
		return AuthLaneSubscription
	}
	return AuthLaneBilled
}
