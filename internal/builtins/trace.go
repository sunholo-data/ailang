package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/types"
)

// Trace builtin functions for AILANG
// These provide trace verification capabilities for testing.
// Part of M-TRACE-TEST (Trace Testing Framework)

func init() {
	registerTraceCheck()
}

// ============================================================================
// Trace Test Builtins
// ============================================================================

// registerTraceCheck registers the _trace_check builtin
func registerTraceCheck() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "stdlib/trace_test",
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
