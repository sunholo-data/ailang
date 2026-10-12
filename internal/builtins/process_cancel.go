package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/process", Name: "_process_cancel", NumArgs: 1, Effect: "Process",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Con("ProcessHandle")).Returns(T.App("Result", T.Unit(), T.Con("WorkerCancelError"))).Effects("Process")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Process", "cancelProcess", args)
		},
		Metadata: &BuiltinMetadata{
			Description: "Cancel and reap an owned subprocess and its process group",
			LongDesc:    "Force-stops an owned POSIX worker, including one blocked in an AI call, and joins its runtime tasks within two seconds. Pending stdin writes may be discarded. A released or foreign handle returns WorkerHandleInvalid. Windows and JS/WASM return WorkerCancelUnsupported. Remote provider billing is outside this local guarantee.",
			Params:      []ParamDoc{{Name: "handle", Description: "Owned ProcessHandle returned by spawnProcess"}},
			Returns:     "Result[(), WorkerCancelError]",
			Examples:    []Example{{Code: "_process_cancel(handle)", Description: "Stop and join an owned worker"}},
			Since:       "unreleased", Stability: StabilityStable, Tags: []string{"process", "cancellation", "worker", "lifecycle"}, Category: "process", SeeAlso: []string{"_process_spawn_process", "_process_close_stdin"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _process_cancel: %v", err))
	}
}
