package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Process effect builtins for AILANG
// Provides external command execution via os/exec

func init() {
	registerProcessExec()
}

// registerProcessExec registers the _process_exec builtin
func registerProcessExec() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/process",
		Name:    "_process_exec",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Process",
		Type:    makeProcessExecType,
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Process", "exec", args)
		},

		Metadata: &BuiltinMetadata{
			Description: "Execute an external command synchronously",
			LongDesc: `Executes an external command with the given arguments. Returns a Result
type containing either ProcessOutput (stdout, stderr, exitCode, truncated,
resolvedPath) or a ProcessError if the command could not be spawned.

Completion semantics: Ok for ALL completed processes (even non-zero exit).
Err only for infrastructure failures (NotFound, NotAllowed, Timeout, etc.).

The Process capability must be granted to use this function.`,

			Params: []ParamDoc{
				{Name: "cmd", Description: "Command name (resolved via PATH or allowlist)"},
				{Name: "args", Description: "List of string arguments"},
			},

			Returns: "Result[ProcessOutput, ProcessError]",

			Examples: []Example{
				{
					Code:        `let result = _process_exec("echo", ["hello", "world"])`,
					Description: "Execute echo command",
				},
			},

			Since:     "v0.8.0",
			Stability: StabilityStable,
			Tags:      []string{"process", "exec", "command", "shell", "system"},
			Category:  "process",
			SeeAlso:   []string{},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _process_exec: %v", err))
	}
}

// makeProcessExecType builds the type signature for _process_exec
// Type: (String, List[String])
//
//	-> Result[{stdout: Bytes, stderr: Bytes, exitCode: Int, truncated: Bool, resolvedPath: String}, ProcessError]
//	! {Process}
func makeProcessExecType() types.Type {
	T := types.NewBuilder()

	// ProcessOutput type
	outputType := T.Record(
		types.Field("stdout", T.Bytes()),
		types.Field("stderr", T.Bytes()),
		types.Field("exitCode", T.Int()),
		types.Field("truncated", T.Bool()),
		types.Field("resolvedPath", T.String()),
	)

	return T.Func(
		T.String(),         // cmd
		T.List(T.String()), // args
	).Returns(
		T.App("Result", outputType, T.Con("ProcessError")),
	).Effects("Process")
}
