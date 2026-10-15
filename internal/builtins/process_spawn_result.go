package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/process", Name: "_process_spawn_result", NumArgs: 2, Effect: "Process",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String(), T.List(T.String())).Returns(T.App("Result", T.Con("ProcessHandle"), T.Con("ProcessError"))).Effects("Process")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Process", "spawnProcessResult", args)
		},
		Metadata: &BuiltinMetadata{
			Description: "Spawn a subprocess with writable stdin and recoverable startup errors",
			LongDesc:    "Returns Ok(ProcessHandle) or Err(NotFound, NotAllowed, SpawnFailed). Uses the same authorization and owned lifecycle as spawnProcess. Missing Process capability remains a policy error. Admission failure stops and reaps any started child before returning no usable handle.",
			Params:      []ParamDoc{{Name: "cmd", Description: "Literal command name or path"}, {Name: "args", Description: "Command arguments without shell expansion"}},
			Returns:     "Result[ProcessHandle, ProcessError]",
			Examples:    []Example{{Code: `_process_spawn_result("cat", [])`, Description: "Try to start a managed worker"}},
			Since:       "unreleased", Stability: StabilityStable, Tags: []string{"process", "result", "worker"}, Category: "process", SeeAlso: []string{"_process_spawn_process", "_process_cancel"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _process_spawn_result: %v", err))
	}
}
