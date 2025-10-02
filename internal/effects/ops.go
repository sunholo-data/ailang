package effects

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// EffOp is a function that implements an effect operation
//
// Effect operations are native Go implementations of effectful functions.
// They receive a capability context and arguments, perform side effects,
// and return a result value.
//
// Parameters:
//   - ctx: The effect context (for capability checking and environment)
//   - args: The arguments passed from AILANG code
//
// Returns:
//   - The result value
//   - An error if the operation fails
//
// Example implementation:
//
//	func ioPrint(ctx *EffContext, args []eval.Value) (eval.Value, error) {
//	    if len(args) != 1 {
//	        return nil, fmt.Errorf("print: expected 1 argument")
//	    }
//	    str := args[0].(*eval.StringValue)
//	    fmt.Print(str.Value)
//	    return &eval.UnitValue{}, nil
//	}
type EffOp func(ctx *EffContext, args []eval.Value) (eval.Value, error)

// Registry holds all effect operations organized by effect name
//
// Structure:
//
//	Registry["IO"]["print"] = ioPrint
//	Registry["IO"]["println"] = ioPrintln
//	Registry["FS"]["readFile"] = fsReadFile
//
// This registry is initialized at package load time with nested maps
// pre-created, making it safe for concurrent reads and allowing
// RegisterOp to work in init() functions.
var Registry = map[string]map[string]EffOp{
	"IO": {},
	"FS": {},
}

// Call invokes an effect operation
//
// This is the main entry point for effect execution. It performs:
//  1. Capability checking (deny if not granted)
//  2. Operation lookup (find the effect implementation)
//  3. Execution (call the EffOp function)
//
// Parameters:
//   - ctx: The effect context (with capability grants)
//   - effectName: The effect name (e.g., "IO", "FS")
//   - opName: The operation name (e.g., "print", "readFile")
//   - args: The arguments to pass to the operation
//
// Returns:
//   - The result value from the operation
//   - An error if capability is missing, operation not found, or execution fails
//
// Example:
//
//	result, err := effects.Call(ctx, "IO", "println", []eval.Value{
//	    &eval.StringValue{Value: "Hello!"},
//	})
func Call(ctx *EffContext, effectName, opName string, args []eval.Value) (eval.Value, error) {
	// Step 1: Check capability
	if err := ctx.RequireCap(effectName); err != nil {
		return nil, err
	}

	// Step 2: Lookup effect
	effectOps, ok := Registry[effectName]
	if !ok {
		return nil, fmt.Errorf("unknown effect: %s", effectName)
	}

	// Step 3: Lookup operation
	op, ok := effectOps[opName]
	if !ok {
		return nil, fmt.Errorf("unknown operation %s in effect %s", opName, effectName)
	}

	// Step 4: Execute operation
	return op(ctx, args)
}

// RegisterOp registers an effect operation
//
// This function is used by effect implementation files (io.go, fs.go) to
// register their operations at package initialization time.
//
// Parameters:
//   - effectName: The effect name (e.g., "IO")
//   - opName: The operation name (e.g., "print")
//   - op: The operation implementation
//
// Example:
//
//	func init() {
//	    RegisterOp("IO", "print", ioPrint)
//	    RegisterOp("IO", "println", ioPrintln)
//	}
func RegisterOp(effectName, opName string, op EffOp) {
	if Registry[effectName] == nil {
		Registry[effectName] = make(map[string]EffOp)
	}
	Registry[effectName][opName] = op
}
