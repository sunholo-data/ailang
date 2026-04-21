package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Process effect builtins for AILANG
// Provides external command execution via os/exec

func init() {
	registerProcessExec()
	registerProcessSpawn()
	registerProcessWriteStdin()
	registerProcessCloseStdin()
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

// registerProcessSpawn registers the _process_spawn_process builtin
func registerProcessSpawn() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/process",
		Name:    "_process_spawn_process",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Process",
		Type:    makeProcessSpawnType,
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Process", "spawnProcess", args)
		},

		Metadata: &BuiltinMetadata{
			Description: "Spawn a subprocess with writable stdin",
			LongDesc: `Spawns a long-running subprocess with a writable stdin pipe.
Stdout and stderr are discarded. Use writeProcessStdin to send data
and closeProcessStdin to signal EOF.

The Process capability must be granted to use this function.`,

			Params: []ParamDoc{
				{Name: "cmd", Description: "Command name (resolved via PATH or allowlist)"},
				{Name: "args", Description: "List of string arguments"},
			},

			Returns: "ProcessHandle",

			Examples: []Example{
				{
					Code:        `let handle = _process_spawn_process("cat", [])`,
					Description: "Spawn cat with writable stdin",
				},
			},

			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"process", "spawn", "stdin", "pipe", "streaming"},
			Category:  "process",
			SeeAlso:   []string{"_process_write_stdin", "_process_close_stdin"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _process_spawn_process: %v", err))
	}
}

// registerProcessWriteStdin registers the _process_write_stdin builtin
func registerProcessWriteStdin() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/process",
		Name:    "_process_write_stdin",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Process",
		Type:    makeProcessWriteStdinType,
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Process", "writeProcessStdin", args)
		},

		Metadata: &BuiltinMetadata{
			Description: "Write bytes to subprocess stdin",
			LongDesc: `Writes bytes to a managed process's stdin pipe. Returns Ok(())
on success or Err(reason) if the pipe is closed or buffer full.

Non-blocking: uses a 256-slot write channel with backpressure.`,

			Params: []ParamDoc{
				{Name: "handle", Description: "ProcessHandle from spawnProcess"},
				{Name: "data", Description: "Bytes to write to stdin"},
			},

			Returns: "Result[(), string]",

			Examples: []Example{
				{
					Code:        `let result = _process_write_stdin(handle, bytes)`,
					Description: "Write bytes to subprocess stdin",
				},
			},

			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"process", "stdin", "write", "pipe", "streaming"},
			Category:  "process",
			SeeAlso:   []string{"_process_spawn_process", "_process_close_stdin"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _process_write_stdin: %v", err))
	}
}

// registerProcessCloseStdin registers the _process_close_stdin builtin
func registerProcessCloseStdin() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/process",
		Name:    "_process_close_stdin",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "Process",
		Type:    makeProcessCloseStdinType,
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Process", "closeProcessStdin", args)
		},

		Metadata: &BuiltinMetadata{
			Description: "Close subprocess stdin pipe (signals EOF)",
			LongDesc: `Closes the stdin pipe of a managed process, signaling EOF.
The subprocess will receive EOF on its next read from stdin.
Safe to call multiple times (idempotent).`,

			Params: []ParamDoc{
				{Name: "handle", Description: "ProcessHandle from spawnProcess"},
			},

			Returns: "()",

			Examples: []Example{
				{
					Code:        `_process_close_stdin(handle)`,
					Description: "Close stdin pipe",
				},
			},

			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"process", "stdin", "close", "pipe", "eof"},
			Category:  "process",
			SeeAlso:   []string{"_process_spawn_process", "_process_write_stdin"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _process_close_stdin: %v", err))
	}
}

// makeProcessSpawnType builds: (String, List[String]) -> ProcessHandle ! {Process}
func makeProcessSpawnType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),         // cmd
		T.List(T.String()), // args
	).Returns(
		T.Con("ProcessHandle"),
	).Effects("Process")
}

// makeProcessWriteStdinType builds: (ProcessHandle, Bytes) -> Result[(), String] ! {Process}
func makeProcessWriteStdinType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("ProcessHandle"), // handle
		T.Bytes(),              // data
	).Returns(
		T.App("Result", T.Unit(), T.String()),
	).Effects("Process")
}

// makeProcessCloseStdinType builds: (ProcessHandle) -> () ! {Process}
func makeProcessCloseStdinType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("ProcessHandle"), // handle
	).Returns(
		T.Unit(),
	).Effects("Process")
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
