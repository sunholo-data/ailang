package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// _trace_emit builtin (M-COG-RUNTIME-BROWSER M5).
//
// AILANG signature:
//
//   func emit(span_name: string, duration_ns: int) -> () ! {Trace}
//
// Side-effect-only emission for the cognitive event log. Does NOT
// affect the existing trace.Collector --emit-trace jsonl output;
// behavior-equivalence preserved.

func init() {
	registerTraceEmit()
}

func registerTraceEmit() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Trace", "emit", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.Int()).Returns(T.Unit()).Effects("Trace")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/trace", Name: "_trace_emit", NumArgs: 2,
		IsPure: false, Effect: "Trace", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Emit a cognitive-trace event (span_name + duration) — sidechannel to the cognitive event log; does NOT affect --emit-trace output",
			Params: []ParamDoc{
				{Name: "span_name", Description: "Name of the span"},
				{Name: "duration_ns", Description: "Span duration in nanoseconds"},
			},
			Returns:   "Unit",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"cognitive-os", "trace"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _trace_emit: %v", err))
	}
}
