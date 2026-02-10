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
	EffectName string   `json:"effect_name"` // e.g., "IO", "FS", "Net"
	OpName     string   `json:"op_name"`     // e.g., "println", "readFile"
	Args       []string `json:"args,omitempty"`
	Result     string   `json:"result,omitempty"`
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
