package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/stream", Name: "_stream_cancel_process_source", NumArgs: 1,
		IsPure: false, Effect: "Stream",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Con("StreamSource")).Returns(T.App("Result", T.Unit(), T.Con("WorkerCancelError"))).Effects("Stream", "Process")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Stream", "cancelProcessSource", args)
		},
		Metadata: &BuiltinMetadata{
			Description: "Stop and join an owned subprocess source within two seconds",
			LongDesc:    "Stops the local owned process group and joins its reader and child Wait. Requires Stream and Process. Stdin and connection sources are rejected. Windows and JS/WASM return WorkerCancelUnsupported. A released handle returns WorkerHandleInvalid.",
			Params:      []ParamDoc{{Name: "source", Description: "Process source returned by asyncExecProcess"}},
			Returns:     "Result[unit, WorkerCancelError]", Since: "unreleased", Stability: StabilityExperimental,
			Tags: []string{"stream", "process", "cancel", "worker"}, Category: "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("register _stream_cancel_process_source: %v", err))
	}
}
