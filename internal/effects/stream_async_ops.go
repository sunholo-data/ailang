package effects

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/sunholo/ailang/internal/eval"
)

// M-ASYNC-IO: New Stream effect operations for multi-source event multiplexing.
//
// These operations extend the Stream effect with:
//   - sourceOfConn: wraps a StreamConnection as an EventSource
//   - asyncReadStdinLines: creates a line-buffered stdin reader source
//   - selectEvents: runs the deterministic multi-source event loop

// StreamSourceOfConn wraps an existing StreamConnection as an EventSource.
//
// Args: [conn: StreamConn(int), name: string, priority: int]
// Returns: StreamSource(int) — opaque handle stored in StreamContext.sources
func StreamSourceOfConn(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("_stream_source_of_conn: expected 3 arguments (conn, name, priority), got %d", len(args))
	}

	connID, err := extractConnID(args[0])
	if err != nil {
		return nil, fmt.Errorf("_stream_source_of_conn: %w", err)
	}

	nameVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_source_of_conn: expected string for name, got %T", args[1])
	}

	priorityVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_stream_source_of_conn: expected int for priority, got %T", args[2])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return nil, fmt.Errorf("_stream_source_of_conn: connection %d not found", connID)
	}

	source := NewConnSource(conn, nameVal.Value, int(priorityVal.Value))
	sourceID := ctx.Stream.AcquireSource(source)

	return makeStreamSource(sourceID), nil
}

// StreamAsyncReadStdinLines creates an EventSource that reads lines from stdin.
//
// Args: [name: string, priority: int]
// Returns: StreamSource(int) — opaque handle stored in StreamContext.sources
func StreamAsyncReadStdinLines(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_async_read_stdin_lines: expected 2 arguments (name, priority), got %d", len(args))
	}

	nameVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_read_stdin_lines: expected string for name, got %T", args[0])
	}

	priorityVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_stream_async_read_stdin_lines: expected int for priority, got %T", args[1])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	source := NewStdinSource(os.Stdin, nameVal.Value, int(priorityVal.Value))
	sourceID := ctx.Stream.AcquireSource(source)

	return makeStreamSource(sourceID), nil
}

// StreamSelectEvents runs the deterministic multi-source event loop.
//
// Args: [sources: list of StreamSource(int), handler: StreamEvent -> bool]
// Returns: unit
func StreamSelectEvents(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_select_events: expected 2 arguments (sources, handler), got %d", len(args))
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	if ctx.FnCaller == nil {
		return nil, fmt.Errorf("_stream_select_events: FnCaller not set on EffContext (evaluator not wired)")
	}

	// Extract source list
	listVal, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_stream_select_events: expected list of StreamSource for sources, got %T", args[0])
	}

	sources := make([]EventSource, 0, len(listVal.Elements))
	for i, elem := range listVal.Elements {
		sourceID, err := extractSourceID(elem)
		if err != nil {
			return nil, fmt.Errorf("_stream_select_events: source[%d]: %w", i, err)
		}

		source, ok := ctx.Stream.GetSource(sourceID)
		if !ok {
			return nil, fmt.Errorf("_stream_select_events: source[%d]: source %d not found", i, sourceID)
		}
		sources = append(sources, source)
	}

	handler := args[1]

	// Use stream context timeouts
	idleTimeout := ctx.Stream.IdleTimeout
	maxDuration := ctx.Stream.MaxDuration

	err := selectEventsLoop(sources, handler, ctx.FnCaller, idleTimeout, maxDuration)
	if err != nil {
		return nil, fmt.Errorf("_stream_select_events: %w", err)
	}

	return &eval.UnitValue{}, nil
}

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

// --- Helper functions ---

// makeStreamSource creates a StreamSource(id) ADT value.
func makeStreamSource(id int) eval.Value {
	return &eval.TaggedValue{
		CtorName: "StreamSource",
		Fields:   []eval.Value{&eval.IntValue{Value: id}},
	}
}

// extractSourceID extracts the integer ID from a StreamSource(int) ADT value.
func extractSourceID(v eval.Value) (int, error) {
	adt, ok := v.(*eval.TaggedValue)
	if !ok {
		return 0, fmt.Errorf("expected StreamSource(int), got %T", v)
	}

	if adt.CtorName != "StreamSource" || len(adt.Fields) < 1 {
		return 0, fmt.Errorf("expected StreamSource(int), got %s", adt.CtorName)
	}

	intVal, ok := adt.Fields[0].(*eval.IntValue)
	if !ok {
		return 0, fmt.Errorf("expected StreamSource(int), got StreamSource(%T)", adt.Fields[0])
	}

	return int(intVal.Value), nil
}
