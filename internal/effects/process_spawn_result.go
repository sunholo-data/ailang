//go:build !js

package effects

import (
	"errors"

	"github.com/sunholo-data/ailang/internal/eval"
)

func init() { RegisterOp("Process", "spawnProcessResult", ProcessSpawnResult) }

// ProcessSpawnResult returns startup failures as ProcessError values. Capability
// policy and malformed internal calls remain evaluator errors.
func ProcessSpawnResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	cmd, cmdArgs, err := processSpawnArgs(args)
	if err != nil {
		return nil, err
	}
	v, err := spawnManaged(ctx, cmd, cmdArgs)
	return processSpawnResult(v, err), nil
}

func processSpawnResult(handle eval.Value, err error) eval.Value {
	if err == nil {
		return &eval.TaggedValue{ModulePath: "std/result", TypeName: "Result", CtorName: "Ok", Fields: []eval.Value{handle}}
	}
	var denial *ProcessDenial
	if errors.As(err, &denial) {
		return makeProcessResultErr(denial.Ctor, denial.Detail)
	}
	var startup *processSpawnStartupError
	if errors.As(err, &startup) {
		err = startup.Unwrap()
	}
	return makeProcessResultErr("SpawnFailed", err.Error())
}
