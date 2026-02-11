package trace

import "fmt"

// Mismatch describes a difference between two trace event streams.
type Mismatch struct {
	Index    int    `json:"index"`             // Event index where mismatch occurred
	Field    string `json:"field"`             // Which field differs
	Expected string `json:"expected"`          // Value from baseline trace
	Actual   string `json:"actual"`            // Value from replay trace
	Context  string `json:"context,omitempty"` // Human-readable context
}

// CompareResult holds the result of comparing two traces.
type CompareResult struct {
	Match      bool       `json:"match"`
	Mismatches []Mismatch `json:"mismatches,omitempty"`
	BaselineN  int        `json:"baseline_events"`
	ReplayN    int        `json:"replay_events"`
}

// CompareTraces compares a baseline trace against a replay trace.
// Timestamps and durations are ignored (non-deterministic).
// All other fields are compared by event type.
func CompareTraces(baseline, replay []TraceEvent) CompareResult {
	result := CompareResult{
		BaselineN: len(baseline),
		ReplayN:   len(replay),
	}

	// Compare events up to the shorter length
	n := len(baseline)
	if len(replay) < n {
		n = len(replay)
	}

	for i := 0; i < n; i++ {
		b, r := baseline[i], replay[i]
		mismatches := compareEvent(i, b, r)
		result.Mismatches = append(result.Mismatches, mismatches...)
	}

	// Report length difference
	if len(baseline) != len(replay) {
		result.Mismatches = append(result.Mismatches, Mismatch{
			Index:    n,
			Field:    "event_count",
			Expected: fmt.Sprintf("%d", len(baseline)),
			Actual:   fmt.Sprintf("%d", len(replay)),
			Context:  fmt.Sprintf("baseline has %d events, replay has %d", len(baseline), len(replay)),
		})
	}

	result.Match = len(result.Mismatches) == 0
	return result
}

// compareEvent compares two events and returns mismatches.
// Skips TimestampNS and DurationNS (non-deterministic).
func compareEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch

	if b.Event != r.Event {
		mm = append(mm, Mismatch{
			Index:    index,
			Field:    "event",
			Expected: string(b.Event),
			Actual:   string(r.Event),
			Context:  fmt.Sprintf("event type mismatch at index %d", index),
		})
		return mm // No point comparing payloads if types differ
	}

	if b.Depth != r.Depth {
		mm = append(mm, Mismatch{
			Index:    index,
			Field:    "depth",
			Expected: fmt.Sprintf("%d", b.Depth),
			Actual:   fmt.Sprintf("%d", r.Depth),
			Context:  contextString(b),
		})
	}

	switch b.Event {
	case EventFunctionEnter, EventFunctionExit:
		mm = append(mm, compareFunctionEvent(index, b, r)...)
	case EventEffect:
		mm = append(mm, compareEffectEvent(index, b, r)...)
	case EventContractCheck:
		mm = append(mm, compareContractEvent(index, b, r)...)
	case EventBudgetDelta:
		mm = append(mm, compareBudgetEvent(index, b, r)...)
	case EventModuleStart, EventModuleEnd:
		mm = append(mm, compareModuleEvent(index, b, r)...)
	case EventError:
		mm = append(mm, compareErrorEvent(index, b, r)...)
	}

	return mm
}

func compareFunctionEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch
	bf, rf := b.Function, r.Function
	if bf == nil || rf == nil {
		if bf != rf { // one nil, one not
			mm = append(mm, Mismatch{Index: index, Field: "function", Expected: fmt.Sprintf("%v", bf), Actual: fmt.Sprintf("%v", rf)})
		}
		return mm
	}

	ctx := contextString(b)

	if bf.Name != rf.Name {
		mm = append(mm, Mismatch{Index: index, Field: "function.name", Expected: bf.Name, Actual: rf.Name, Context: ctx})
	}
	if !sliceEqual(bf.Args, rf.Args) {
		mm = append(mm, Mismatch{Index: index, Field: "function.args", Expected: fmt.Sprintf("%v", bf.Args), Actual: fmt.Sprintf("%v", rf.Args), Context: ctx})
	}
	if bf.Result != rf.Result {
		mm = append(mm, Mismatch{Index: index, Field: "function.result", Expected: bf.Result, Actual: rf.Result, Context: ctx})
	}
	// DurationNS intentionally skipped (non-deterministic)
	return mm
}

func compareEffectEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch
	be, re := b.Effect, r.Effect
	if be == nil || re == nil {
		if be != re {
			mm = append(mm, Mismatch{Index: index, Field: "effect", Expected: fmt.Sprintf("%v", be), Actual: fmt.Sprintf("%v", re)})
		}
		return mm
	}

	ctx := contextString(b)

	if be.EffectName != re.EffectName {
		mm = append(mm, Mismatch{Index: index, Field: "effect.effect_name", Expected: be.EffectName, Actual: re.EffectName, Context: ctx})
	}
	if be.OpName != re.OpName {
		mm = append(mm, Mismatch{Index: index, Field: "effect.op_name", Expected: be.OpName, Actual: re.OpName, Context: ctx})
	}
	if !sliceEqual(be.Args, re.Args) {
		mm = append(mm, Mismatch{Index: index, Field: "effect.args", Expected: fmt.Sprintf("%v", be.Args), Actual: fmt.Sprintf("%v", re.Args), Context: ctx})
	}
	if be.Result != re.Result {
		mm = append(mm, Mismatch{Index: index, Field: "effect.result", Expected: be.Result, Actual: re.Result, Context: ctx})
	}
	return mm
}

func compareContractEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch
	bc, rc := b.Contract, r.Contract
	if bc == nil || rc == nil {
		if bc != rc {
			mm = append(mm, Mismatch{Index: index, Field: "contract", Expected: fmt.Sprintf("%v", bc), Actual: fmt.Sprintf("%v", rc)})
		}
		return mm
	}

	ctx := contextString(b)

	if bc.Kind != rc.Kind {
		mm = append(mm, Mismatch{Index: index, Field: "contract.kind", Expected: bc.Kind, Actual: rc.Kind, Context: ctx})
	}
	if bc.Passed != rc.Passed {
		mm = append(mm, Mismatch{Index: index, Field: "contract.passed", Expected: fmt.Sprintf("%t", bc.Passed), Actual: fmt.Sprintf("%t", rc.Passed), Context: ctx})
	}
	if bc.Message != rc.Message {
		mm = append(mm, Mismatch{Index: index, Field: "contract.message", Expected: bc.Message, Actual: rc.Message, Context: ctx})
	}
	return mm
}

func compareBudgetEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch
	bb, rb := b.Budget, r.Budget
	if bb == nil || rb == nil {
		if bb != rb {
			mm = append(mm, Mismatch{Index: index, Field: "budget", Expected: fmt.Sprintf("%v", bb), Actual: fmt.Sprintf("%v", rb)})
		}
		return mm
	}

	ctx := contextString(b)

	if bb.Effect != rb.Effect {
		mm = append(mm, Mismatch{Index: index, Field: "budget.effect", Expected: bb.Effect, Actual: rb.Effect, Context: ctx})
	}
	if bb.Used != rb.Used {
		mm = append(mm, Mismatch{Index: index, Field: "budget.used", Expected: fmt.Sprintf("%d", bb.Used), Actual: fmt.Sprintf("%d", rb.Used), Context: ctx})
	}
	if bb.Limit != rb.Limit {
		mm = append(mm, Mismatch{Index: index, Field: "budget.limit", Expected: fmt.Sprintf("%d", bb.Limit), Actual: fmt.Sprintf("%d", rb.Limit), Context: ctx})
	}
	if bb.Remaining != rb.Remaining {
		mm = append(mm, Mismatch{Index: index, Field: "budget.remaining", Expected: fmt.Sprintf("%d", bb.Remaining), Actual: fmt.Sprintf("%d", rb.Remaining), Context: ctx})
	}
	return mm
}

func compareModuleEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch
	bm, rm := b.Module, r.Module
	if bm == nil || rm == nil {
		if bm != rm {
			mm = append(mm, Mismatch{Index: index, Field: "module", Expected: fmt.Sprintf("%v", bm), Actual: fmt.Sprintf("%v", rm)})
		}
		return mm
	}

	if bm.Name != rm.Name {
		mm = append(mm, Mismatch{Index: index, Field: "module.name", Expected: bm.Name, Actual: rm.Name})
	}
	// DurationNS intentionally skipped (non-deterministic)
	// Caps compared for module_start
	if b.Event == EventModuleStart && !sliceEqual(bm.Caps, rm.Caps) {
		mm = append(mm, Mismatch{Index: index, Field: "module.caps", Expected: fmt.Sprintf("%v", bm.Caps), Actual: fmt.Sprintf("%v", rm.Caps)})
	}
	return mm
}

func compareErrorEvent(index int, b, r TraceEvent) []Mismatch {
	var mm []Mismatch
	be, re := b.Error, r.Error
	if be == nil || re == nil {
		if be != re {
			mm = append(mm, Mismatch{Index: index, Field: "error", Expected: fmt.Sprintf("%v", be), Actual: fmt.Sprintf("%v", re)})
		}
		return mm
	}

	if be.Message != re.Message {
		mm = append(mm, Mismatch{Index: index, Field: "error.message", Expected: be.Message, Actual: re.Message})
	}
	if be.Location != re.Location {
		mm = append(mm, Mismatch{Index: index, Field: "error.location", Expected: be.Location, Actual: re.Location})
	}
	return mm
}

// contextString returns a human-readable context string for an event.
func contextString(e TraceEvent) string {
	switch e.Event {
	case EventFunctionEnter:
		if e.Function != nil {
			return fmt.Sprintf("function_enter %q at depth %d", e.Function.Name, e.Depth)
		}
	case EventFunctionExit:
		if e.Function != nil {
			return fmt.Sprintf("function_exit %q at depth %d", e.Function.Name, e.Depth)
		}
	case EventEffect:
		if e.Effect != nil {
			return fmt.Sprintf("effect %s.%s at depth %d", e.Effect.EffectName, e.Effect.OpName, e.Depth)
		}
	case EventContractCheck:
		if e.Contract != nil {
			return fmt.Sprintf("contract %s (passed=%t) at depth %d", e.Contract.Kind, e.Contract.Passed, e.Depth)
		}
	case EventBudgetDelta:
		if e.Budget != nil {
			return fmt.Sprintf("budget %s (used=%d/%d) at depth %d", e.Budget.Effect, e.Budget.Used, e.Budget.Limit, e.Depth)
		}
	}
	return fmt.Sprintf("%s at depth %d", e.Event, e.Depth)
}

// sliceEqual compares two string slices.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
