package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Trace builtin functions for AILANG
// These provide trace verification capabilities for testing.
// Part of M-TRACE-TEST (Trace Testing Framework)

func init() {
	registerTraceCheck()
	registerTraceSpanStart()
	registerTraceSpanEnd()
	registerTraceEvent()
}

// ============================================================================
// Trace Test Builtins
// ============================================================================

// registerTraceCheck registers the _trace_check builtin
func registerTraceCheck() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/trace_test",
		Name:    "_trace_check",
		NumArgs: 1,
		IsPure:  true, // Reading trace state is side-effect free
		Effect:  "",   // No effect required
		Type:    makeTraceCheckType,
		Impl:    effects.TraceCheckImpl,

		Metadata: &BuiltinMetadata{
			Description: "Check if a trace span with the given name exists",
			LongDesc:    "Checks if a trace span with the specified name was recorded during program execution. Supports prefix matching for hierarchical span names (e.g., 'compile' matches 'compile.parse').",
			Params: []ParamDoc{
				{Name: "name", Description: "The span name or prefix to check for"},
			},
			Returns: "true if a matching span exists, false otherwise",
			Examples: []Example{
				{Code: `_trace_check("compile.parse")`, Description: "Returns true if compile.parse span was recorded"},
				{Code: `_trace_check("compile")`, Description: "Returns true if any compile.* span exists"},
			},
			SeeAlso:   []string{},
			Since:     "v0.7.1",
			Stability: StabilityExperimental,
			Tags:      []string{"trace", "testing", "telemetry"},
			Category:  "testing",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _trace_check: %v", err))
	}
}

// makeTraceCheckType builds the type signature for _trace_check
// Type: string -> bool
func makeTraceCheckType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Bool()).Build()
}

// ============================================================================
// std/trace Builtins (M-WASM-TRACE)
// ============================================================================

// _trace_span_start: Record a custom span start in the trace stream
func registerTraceSpanStart() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/trace",
		Name:    "_trace_span_start",
		NumArgs: 1,
		Effect:  "Trace",
		Type:    makeTraceStringToUnitType,
		Impl:    traceSpanStartImpl,
		Metadata: &BuiltinMetadata{
			Description: "Record a custom trace span start",
			Params:      []ParamDoc{{Name: "name", Description: "Span name"}},
			Returns:     "Unit",
			Since:       "v0.12.0",
			Stability:   StabilityExperimental,
			Tags:        []string{"trace", "telemetry"},
			Category:    "trace",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _trace_span_start: %v", err))
	}
}

// _trace_span_end: Record a custom span end in the trace stream
func registerTraceSpanEnd() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/trace",
		Name:    "_trace_span_end",
		NumArgs: 1,
		Effect:  "Trace",
		Type:    makeTraceStringToUnitType,
		Impl:    traceSpanEndImpl,
		Metadata: &BuiltinMetadata{
			Description: "Record a custom trace span end",
			Params:      []ParamDoc{{Name: "name", Description: "Span name"}},
			Returns:     "Unit",
			Since:       "v0.12.0",
			Stability:   StabilityExperimental,
			Tags:        []string{"trace", "telemetry"},
			Category:    "trace",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _trace_span_end: %v", err))
	}
}

// _trace_event: Emit a named trace event with data payload
func registerTraceEvent() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/trace",
		Name:    "_trace_event",
		NumArgs: 2,
		Effect:  "Trace",
		Type:    makeTraceEventType,
		Impl:    traceEventImpl,
		Metadata: &BuiltinMetadata{
			Description: "Emit a custom trace event",
			Params: []ParamDoc{
				{Name: "name", Description: "Event name"},
				{Name: "data", Description: "Event data payload"},
			},
			Returns:   "Unit",
			Since:     "v0.12.0",
			Stability: StabilityExperimental,
			Tags:      []string{"trace", "telemetry"},
			Category:  "trace",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _trace_event: %v", err))
	}
}

// Type: string -> () ! {Trace}
func makeTraceStringToUnitType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.Unit()).Effects("Trace")
}

// Type: (string, string) -> () ! {Trace}
func makeTraceEventType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String(), T.String()).Returns(T.Unit()).Effects("Trace")
}

func traceSpanStartImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Trace", ""); err != nil {
		return nil, err
	}
	name := ""
	if len(args) > 0 {
		if s, ok := args[0].(*eval.StringValue); ok {
			name = s.Value
		}
	}
	ctx.RecordFunctionEnter(name, nil)
	return &eval.UnitValue{}, nil
}

func traceSpanEndImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Trace", ""); err != nil {
		return nil, err
	}
	name := ""
	if len(args) > 0 {
		if s, ok := args[0].(*eval.StringValue); ok {
			name = s.Value
		}
	}
	ctx.RecordFunctionExit(name, "")
	return &eval.UnitValue{}, nil
}

func traceEventImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Trace", ""); err != nil {
		return nil, err
	}
	name, data := "", ""
	if len(args) > 0 {
		if s, ok := args[0].(*eval.StringValue); ok {
			name = s.Value
		}
	}
	if len(args) > 1 {
		if s, ok := args[1].(*eval.StringValue); ok {
			data = s.Value
		}
	}
	ctx.RecordEffect("Trace", name, []string{data}, "()")
	return &eval.UnitValue{}, nil
}
