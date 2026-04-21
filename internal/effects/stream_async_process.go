//go:build !js

package effects

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/sunholo-data/ailang/internal/eval"
)

// StreamAsyncExecProcess spawns a subprocess and delivers its stdout as SourceBytes events.
//
// Args: [cmd: string, args: [string], name: string, priority: int, chunkSize: int]
// Returns: StreamSource(int) — opaque handle stored in StreamContext.sources
// Requires: Process capability (for subprocess spawning) + Stream capability (for source creation)
func StreamAsyncExecProcess(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 5 {
		return nil, fmt.Errorf("_stream_async_exec_process: expected 5 arguments (cmd, args, name, priority, chunkSize), got %d", len(args))
	}

	cmdVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_exec_process: expected string for cmd, got %T", args[0])
	}

	argsVal, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_exec_process: expected list for args, got %T", args[1])
	}
	cmdArgs := make([]string, len(argsVal.Elements))
	for i, elem := range argsVal.Elements {
		strVal, ok := elem.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("_stream_async_exec_process: args[%d] expected string, got %T", i, elem)
		}
		cmdArgs[i] = strVal.Value
	}

	nameVal, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_exec_process: expected string for name, got %T", args[2])
	}

	priorityVal, ok := args[3].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_exec_process: expected int for priority, got %T", args[3])
	}

	chunkSizeVal, ok := args[4].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_exec_process: expected int for chunkSize, got %T", args[4])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	// Resolve command path via ProcessContext (reuse allowlist + LookPath)
	cmdName := cmdVal.Value
	var resolvedPath string

	pc := ctx.Process
	if pc == nil {
		// No Process context — resolve via LookPath directly
		resolved, err := exec.LookPath(cmdName)
		if err != nil {
			return nil, fmt.Errorf("_stream_async_exec_process: command not found: %s", cmdName)
		}
		resolvedPath = resolved
	} else if pc.HasAllowlist {
		resolved, allowed := pc.Allowlist[cmdName]
		if !allowed {
			return nil, fmt.Errorf("_stream_async_exec_process: command not allowed: %s", cmdName)
		}
		if resolved == "" {
			return nil, fmt.Errorf("_stream_async_exec_process: command not found: %s", cmdName)
		}
		resolvedPath = resolved
	} else {
		resolved, err := exec.LookPath(cmdName)
		if err != nil {
			return nil, fmt.Errorf("_stream_async_exec_process: command not found: %s", cmdName)
		}
		resolvedPath = resolved
	}

	source, err := NewProcessSource(
		context.Background(),
		resolvedPath,
		cmdArgs,
		nameVal.Value,
		int(priorityVal.Value),
		int(chunkSizeVal.Value),
	)
	if err != nil {
		return nil, fmt.Errorf("_stream_async_exec_process: %w", err)
	}

	sourceID := ctx.Stream.AcquireSource(source)
	return makeStreamSource(sourceID), nil
}
