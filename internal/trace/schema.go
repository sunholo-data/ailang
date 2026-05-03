// Package trace provides program-level execution trace collection for AILANG.
//
// This captures what happens when AILANG code evaluates: function calls,
// effect invocations, contract check results, and budget consumption.
// Complementary to the chains/observatory system which captures agent-level traces.
//
// M-TRACE-EXPORT Phase 1: Standalone JSONL output via --emit-trace flag.
package trace

// EventType enumerates trace event kinds.
type EventType string

const (
	EventModuleStart   EventType = "module_start"
	EventModuleEnd     EventType = "module_end"
	EventFunctionEnter EventType = "function_enter"
	EventFunctionExit  EventType = "function_exit"
	EventEffect        EventType = "effect"
	EventContractCheck EventType = "contract_check"
	EventBudgetDelta   EventType = "budget_delta"
	EventError         EventType = "error"
)

// TraceEvent is the top-level envelope for all trace events.
// Each event has a type, timestamp, and one populated payload field.
type TraceEvent struct {
	Version     string    `json:"version"`
	Event       EventType `json:"event"`
	TimestampNS int64     `json:"timestamp_ns"`
	Depth       int       `json:"depth,omitempty"`

	// OTEL-compatible span identifiers (M-WASM-TRACE).
	// TraceID is consistent across all events in one execution.
	// SpanID/ParentSpanID form a parent-child tree for distributed tracing.
	TraceID      string `json:"trace_id,omitempty"`
	SpanID       string `json:"span_id,omitempty"`
	ParentSpanID string `json:"parent_span_id,omitempty"`

	// Event-specific payload (exactly one is non-nil per event)
	Module   *ModuleEvent   `json:"module,omitempty"`
	Function *FunctionEvent `json:"function,omitempty"`
	Effect   *EffectEvent   `json:"effect,omitempty"`
	Contract *ContractEvent `json:"contract,omitempty"`
	Budget   *BudgetEvent   `json:"budget,omitempty"`
	Error    *ErrorEvent    `json:"error,omitempty"`
}

// ModuleEvent captures module start/end.
type ModuleEvent struct {
	Name       string   `json:"name"`
	DurationNS int64    `json:"duration_ns,omitempty"`
	Caps       []string `json:"caps,omitempty"`
}

// FunctionEvent captures function enter/exit.
type FunctionEvent struct {
	Name       string   `json:"name"`
	Args       []string `json:"args,omitempty"`
	Result     string   `json:"result,omitempty"`
	DurationNS int64    `json:"duration_ns,omitempty"`
}

// EffectEvent captures an effect invocation.
// Mirrors data available in effects.Call().
type EffectEvent struct {
	EffectName    string   `json:"effect_name"`             // e.g., "IO", "FS", "Net"
	OpName        string   `json:"op_name"`                 // e.g., "println", "readFile"
	Args          []string `json:"args,omitempty"`          //
	Result        string   `json:"result,omitempty"`        //
	Deterministic *bool    `json:"deterministic,omitempty"` // nil=unknown, true=deterministic, false=non-deterministic

	// Route captures dynamic-routing metadata for AI effect events that
	// went through a routing-capable provider (OpenRouter). Nil for direct
	// provider calls and all non-AI effects. Additive — old trace events
	// without this field unmarshal correctly with Route == nil.
	Route *ResolvedRoute `json:"route,omitempty"`
}

// ResolvedRoute captures dynamic-routing metadata for an AI effect event.
//
// Populated when an AI call goes through a routing-capable provider
// (currently: OpenRouter). Nil for direct provider calls. Additive to
// EffectEvent — backward-compatible with old traces (zero-valued
// ResolvedRoute fields parse fine).
//
// CostUSD is stored as a string to preserve provider-reported precision.
type ResolvedRoute struct {
	RequestedModel   string   `json:"requested_model,omitempty"`
	ResolvedModel    string   `json:"resolved_model,omitempty"`
	ResolvedProvider string   `json:"resolved_provider,omitempty"`
	FallbackChain    []string `json:"fallback_chain,omitempty"`
	PromptTokens     int      `json:"prompt_tokens,omitempty"`
	CompletionTokens int      `json:"completion_tokens,omitempty"`
	CachedTokens     int      `json:"cached_tokens,omitempty"`
	ReasoningTokens  int      `json:"reasoning_tokens,omitempty"`
	CostUSD          string   `json:"cost_usd,omitempty"`
}

// nonDeterministicOps maps effect.op pairs that are inherently non-deterministic.
// Used by the collector to flag effect events for replay tolerance.
var nonDeterministicOps = map[string]bool{
	"Clock.now":       true, // Wall clock always varies
	"Clock.sleep":     true, // Real-time delays vary
	"IO.readLine":     true, // Depends on stdin
	"Net.httpGet":     true, // Network responses vary
	"Net.httpPost":    true, // Network responses vary
	"Net.httpRequest": true, // Network responses vary
}

// IsNonDeterministic returns true if the given effect+op pair is known to produce
// non-deterministic results across runs.
func IsNonDeterministic(effectName, opName string) bool {
	return nonDeterministicOps[effectName+"."+opName]
}

// ContractEvent captures a contract check result.
// Mirrors ContractCheck from effects/contracts.go.
type ContractEvent struct {
	Kind     string `json:"kind"` // "requires", "ensures", "invariant"
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Location string `json:"location"`
	Function string `json:"function"`
}

// BudgetEvent captures a budget state change after an effect invocation.
// Mirrors data from effects/budget.go.
type BudgetEvent struct {
	Effect    string `json:"effect"`
	Used      int    `json:"used"`
	Limit     int    `json:"limit"`     // -1 if unlimited
	Remaining int    `json:"remaining"` // -1 if unlimited
	Physical  int    `json:"physical"`  // Physical usage count
}

// ErrorEvent captures an error during execution.
type ErrorEvent struct {
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}
