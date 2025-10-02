package effects

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/eval"
)

// init registers IO effect operations
func init() {
	RegisterOp("IO", "print", ioPrint)
	RegisterOp("IO", "println", ioPrintln)
	RegisterOp("IO", "readLine", ioReadLine)
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

	fmt.Print(str.Value)
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

	fmt.Println(str.Value)
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

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// Return empty string on EOF
			return &eval.StringValue{Value: ""}, nil
		}
		return nil, fmt.Errorf("readLine: %w", err)
	}

	// Trim trailing newline
	line = strings.TrimSuffix(line, "\n")
	// Also trim \r on Windows
	line = strings.TrimSuffix(line, "\r")

	return &eval.StringValue{Value: line}, nil
}
