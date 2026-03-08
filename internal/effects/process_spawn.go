//go:build !js

package effects

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/sunholo/ailang/internal/eval"
)

// M-ASYNC-IO Phase 3: Process effect operations for managed subprocess stdin writing.
//
// Three operations:
//   - spawnProcess: spawn subprocess with writable stdin
//   - writeProcessStdin: write bytes to subprocess stdin
//   - closeProcessStdin: close stdin pipe (signals EOF)

func init() {
	RegisterOp("Process", "spawnProcess", ProcessSpawn)
	RegisterOp("Process", "writeProcessStdin", ProcessWriteStdin)
	RegisterOp("Process", "closeProcessStdin", ProcessCloseStdin)
}

// ProcessSpawn spawns a subprocess with a writable stdin pipe.
//
// Args: [cmd: string, args: [string]]
// Returns: ProcessHandle(int) — opaque handle stored in ProcessContext.managed
func ProcessSpawn(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_process_spawn_process: expected 2 arguments (cmd, args), got %d", len(args))
	}

	cmdVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_process_spawn_process: expected string for cmd, got %T", args[0])
	}

	argsVal, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_process_spawn_process: expected list for args, got %T", args[1])
	}
	cmdArgs := make([]string, len(argsVal.Elements))
	for i, elem := range argsVal.Elements {
		strVal, ok := elem.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("_process_spawn_process: args[%d] expected string, got %T", i, elem)
		}
		cmdArgs[i] = strVal.Value
	}

	if ctx.Process == nil {
		return nil, fmt.Errorf("E_PROCESS_NO_CONTEXT: Process effect not configured (missing --caps Process)")
	}

	// Resolve command path via ProcessContext (reuse allowlist + LookPath)
	resolvedPath, err := resolveCommand(ctx.Process, cmdVal.Value)
	if err != nil {
		return nil, fmt.Errorf("_process_spawn_process: %w", err)
	}

	mp, err := NewManagedProcess(context.Background(), resolvedPath, cmdArgs)
	if err != nil {
		return nil, fmt.Errorf("_process_spawn_process: %w", err)
	}

	handleID := ctx.Process.AcquireManagedProcess(mp)
	return makeProcessHandle(handleID), nil
}

// ProcessWriteStdin writes bytes to a managed process's stdin.
//
// Args: [handle: ProcessHandle(int), data: bytes]
// Returns: Result[(), string] — Ok(()) on success, Err(reason) on failure
func ProcessWriteStdin(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_process_write_stdin: expected 2 arguments (handle, data), got %d", len(args))
	}

	handleID, err := extractProcessHandleID(args[0])
	if err != nil {
		return nil, fmt.Errorf("_process_write_stdin: %w", err)
	}

	bytesVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_process_write_stdin: expected bytes for data, got %T", args[1])
	}

	if ctx.Process == nil {
		return nil, fmt.Errorf("E_PROCESS_NO_CONTEXT: Process effect not configured (missing --caps Process)")
	}

	mp, ok := ctx.Process.GetManagedProcess(handleID)
	if !ok {
		return processResultErr("process handle not found"), nil
	}

	if err := mp.Write(bytesVal.Value); err != nil {
		return processResultErr(err.Error()), nil
	}

	return processResultOk(), nil
}

// ProcessCloseStdin closes the stdin pipe of a managed process.
//
// Args: [handle: ProcessHandle(int)]
// Returns: () — unit
func ProcessCloseStdin(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("_process_close_stdin: expected 1 argument (handle), got %d", len(args))
	}

	handleID, err := extractProcessHandleID(args[0])
	if err != nil {
		return nil, fmt.Errorf("_process_close_stdin: %w", err)
	}

	if ctx.Process == nil {
		return nil, fmt.Errorf("E_PROCESS_NO_CONTEXT: Process effect not configured (missing --caps Process)")
	}

	mp, ok := ctx.Process.GetManagedProcess(handleID)
	if !ok {
		// Already closed or never existed — idempotent
		return &eval.UnitValue{}, nil
	}

	mp.CloseStdin()
	ctx.Process.ReleaseManagedProcess(handleID)

	return &eval.UnitValue{}, nil
}

// --- Helper functions ---

// resolveCommand resolves a command name to an absolute path using ProcessContext.
func resolveCommand(pc *ProcessContext, cmdName string) (string, error) {
	if pc == nil {
		resolved, err := exec.LookPath(cmdName)
		if err != nil {
			return "", fmt.Errorf("command not found: %s", cmdName)
		}
		return resolved, nil
	}

	if pc.HasAllowlist {
		resolved, allowed := pc.Allowlist[cmdName]
		if !allowed {
			return "", fmt.Errorf("command not allowed: %s", cmdName)
		}
		if resolved == "" {
			return "", fmt.Errorf("command not found: %s", cmdName)
		}
		return resolved, nil
	}

	resolved, err := exec.LookPath(cmdName)
	if err != nil {
		return "", fmt.Errorf("command not found: %s", cmdName)
	}
	return resolved, nil
}

// makeProcessHandle creates a ProcessHandle(id) ADT value.
func makeProcessHandle(id int) eval.Value {
	return &eval.TaggedValue{
		CtorName: "ProcessHandle",
		Fields:   []eval.Value{&eval.IntValue{Value: id}},
	}
}

// extractProcessHandleID extracts the integer ID from a ProcessHandle(int) ADT value.
func extractProcessHandleID(v eval.Value) (int, error) {
	adt, ok := v.(*eval.TaggedValue)
	if !ok {
		return 0, fmt.Errorf("expected ProcessHandle(int), got %T", v)
	}
	if adt.CtorName != "ProcessHandle" || len(adt.Fields) < 1 {
		return 0, fmt.Errorf("expected ProcessHandle(int), got %s", adt.CtorName)
	}
	intVal, ok := adt.Fields[0].(*eval.IntValue)
	if !ok {
		return 0, fmt.Errorf("expected ProcessHandle(int), got ProcessHandle(%T)", adt.Fields[0])
	}
	return int(intVal.Value), nil
}

// processResultOk creates Ok(()) — a Result success with unit value.
func processResultOk() eval.Value {
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{&eval.UnitValue{}},
	}
}

// processResultErr creates Err(message) — a Result error with string message.
func processResultErr(msg string) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Err",
		Fields:   []eval.Value{&eval.StringValue{Value: msg}},
	}
}
