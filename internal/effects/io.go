package effects

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

// init registers IO effect operations
func init() {
	RegisterOp("IO", "print", ioPrint)
	RegisterOp("IO", "println", ioPrintln)
	RegisterOp("IO", "readLine", ioReadLine)
	RegisterOp("IO", "writeBytes", ioWriteBytes)
	RegisterOp("IO", "exit", ioExit)
	RegisterOp("IO", "flush", ioFlush)
	RegisterOp("IO", "printErr", ioPrintErr)
	RegisterOp("IO", "eprintln", ioEprintln)
}

// ioPrint implements IO.print(s: String) -> ()
//
// Prints a string to stdout without a trailing newline.
//
// Parameters:
//   - ctx: Effect context (capability check already done by Call())
//   - args: [StringValue] - the string to print
//
// Returns:
//   - UnitValue on success
//   - Error if wrong number/type of arguments
//
// Example AILANG code:
//
//	print("Hello")  -- prints "Hello" without newline
func ioPrint(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("print: expected 1 argument, got %d", len(args))
	}

	str, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("print: expected String, got %T", args[0])
	}

	fmt.Fprint(ctx.GetIOWriter(), str.Value)
	return &eval.UnitValue{}, nil
}

// ioPrintln implements IO.println(s: String) -> ()
//
// Prints a string to stdout with a trailing newline.
//
// Parameters:
//   - ctx: Effect context
//   - args: [StringValue] - the string to print
//
// Returns:
//   - UnitValue on success
//   - Error if wrong number/type of arguments
//
// Example AILANG code:
//
//	println("Hello")  -- prints "Hello\n"
func ioPrintln(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("println: expected 1 argument, got %d", len(args))
	}

	str, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("println: expected String, got %T", args[0])
	}

	fmt.Fprintln(ctx.GetIOWriter(), str.Value)
	return &eval.UnitValue{}, nil
}

// ioReadLine implements IO.readLine() -> String
//
// Reads a line from stdin, blocking until a newline is encountered.
// The trailing newline (and carriage return on Windows) are removed.
//
// Parameters:
//   - ctx: Effect context
//   - args: [] - no arguments
//
// Returns:
//   - StringValue with the line read (without newline)
//   - Empty string on EOF
//   - Error if wrong number of arguments or read fails
//
// Example AILANG code:
//
//	let name = readLine()  -- blocks until user presses Enter
func ioReadLine(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("readLine: expected 0 arguments, got %d", len(args))
	}

	reader := ctx.GetIOReader()
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// Return whatever was read before EOF (may be partial last line)
			line = strings.TrimSuffix(line, "\r")
			return &eval.StringValue{Value: line}, nil
		}
		return nil, fmt.Errorf("readLine: %w", err)
	}

	// Trim trailing newline
	line = strings.TrimSuffix(line, "\n")
	// Also trim \r on Windows
	line = strings.TrimSuffix(line, "\r")

	return &eval.StringValue{Value: line}, nil
}

// ioExit implements IO.exit(code: Int) -> ()
//
// Terminates the AILANG process with the given exit code.
// Uses a sentinel panic (EvalExitCode) that propagates up to the runtime,
// which catches it, flushes telemetry, and calls os.Exit(code).
//
// Parameters:
//   - ctx: Effect context (capability check already done by Call())
//   - args: [IntValue] - the exit code (0-255 typical, OS takes code & 0xFF)
//
// Returns:
//   - Never returns — panics with EvalExitCode sentinel
func ioExit(_ *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("exit: expected 1 argument, got %d", len(args))
	}

	intVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("exit: expected Int, got %T", args[0])
	}

	panic(&eval.EvalExitCode{Code: intVal.Value})
}

// ioWriteBytes implements IO.writeBytes(data: Bytes) -> ()
//
// Writes raw bytes to stdout. Unlike print/println which take strings,
// writeBytes writes binary data directly — enabling piping to external
// tools (e.g., `ailang run ... | afplay -f LEI16 -r 24000 -c 1 -`).
//
// Parameters:
//   - ctx: Effect context
//   - args: [BytesValue] - the bytes to write
//
// Returns:
//   - UnitValue on success
//   - Error if wrong number/type of arguments or write fails
//
// Example AILANG code:
//
//	writeBytes(pcmData)  -- writes raw PCM audio to stdout
func ioWriteBytes(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("writeBytes: expected 1 argument, got %d", len(args))
	}

	bytesVal, ok := args[0].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("writeBytes: expected Bytes, got %T", args[0])
	}

	if _, err := ctx.GetIOWriter().Write(bytesVal.Value); err != nil {
		return nil, fmt.Errorf("writeBytes: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// ioFlush implements IO.flush() -> ()
//
// Flushes any buffered stdout to the terminal. On a TTY, stdout is line-buffered
// (flushed on each newline), so flush() is only needed for partial-line output —
// progress bars, spinners, and game frames that use `\r` without a trailing
// newline. On piped stdout (unbuffered) it is a harmless no-op.
//
// Example AILANG code:
//
//	print("\rWorking... 42%"); flush()  -- appears immediately, no newline needed
func ioFlush(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("flush: expected 0 arguments, got %d", len(args))
	}
	if err := ctx.FlushIO(); err != nil {
		return nil, fmt.Errorf("flush: %w", err)
	}
	return &eval.UnitValue{}, nil
}

// ioPrintErr implements IO.printErr(s: String) -> ()
//
// Writes a string to stderr without a trailing newline. stderr is unbuffered, so
// output appears immediately — useful for progress/diagnostics that should not be
// captured when stdout is piped to a consumer.
func ioPrintErr(_ *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("printErr: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("printErr: expected String, got %T", args[0])
	}
	fmt.Fprint(os.Stderr, str.Value)
	return &eval.UnitValue{}, nil
}

// ioEprintln implements IO.eprintln(s: String) -> ()
//
// Writes a string to stderr with a trailing newline (unbuffered).
func ioEprintln(_ *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("eprintln: expected 1 argument, got %d", len(args))
	}
	str, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("eprintln: expected String, got %T", args[0])
	}
	fmt.Fprintln(os.Stderr, str.Value)
	return &eval.UnitValue{}, nil
}
