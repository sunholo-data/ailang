package runtime

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// BuiltinRegistry holds native Go implementations of builtin functions
//
// Builtins are functions implemented in Go that can be called from AILANG modules.
// They are identified by names starting with underscore (e.g., _io_print).
//
// The registry provides:
//   - Type-safe function implementations routed through the effect system
//   - Runtime access via GetBuiltin()
//   - Automatic registration of stdlib functions
//
// Thread-safety: The registry is initialized once and read-only after that,
// so it is safe to use concurrently.
type BuiltinRegistry struct {
	builtins  map[string]eval.Value
	evaluator *eval.CoreEvaluator // Reference to evaluator for EffContext access
}

// NewBuiltinRegistry creates a new builtin registry with all stdlib functions registered
//
// Parameters:
//   - evaluator: The evaluator (needed to access EffContext during builtin calls)
//
// Returns:
//   - A fully-initialized BuiltinRegistry
func NewBuiltinRegistry(evaluator *eval.CoreEvaluator) *BuiltinRegistry {
	br := &BuiltinRegistry{
		builtins:  make(map[string]eval.Value),
		evaluator: evaluator,
	}
	br.registerArithmeticBuiltins()
	br.registerEffectBuiltins()
	return br
}

// Get looks up a builtin function by name
//
// Parameters:
//   - name: The builtin function name (e.g., "_io_print")
//
// Returns:
//   - The builtin function value if found
//   - A boolean indicating whether the builtin was found
func (br *BuiltinRegistry) Get(name string) (eval.Value, bool) {
	val, ok := br.builtins[name]
	return val, ok
}

// registerArithmeticBuiltins registers pure arithmetic and comparison operators
//
// These builtins are direct delegations to the eval package's builtin implementations.
// They don't require capabilities as they're pure functions.
func (br *BuiltinRegistry) registerArithmeticBuiltins() {
	// Wrap each eval.BuiltinFunc as an eval.BuiltinFunction (which implements eval.Value)
	for name := range eval.Builtins {
		// Capture name for closure
		builtinName := name
		br.builtins[name] = &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				// Delegate to eval.CallBuiltin which handles all the reflection
				return eval.CallBuiltin(builtinName, args)
			},
		}
	}
}

// registerEffectBuiltins registers all effect-based builtin functions
//
// These builtins route through the effect system, requiring capability grants.
//
// Builtins registered:
//   - IO effect: _io_print, _io_println, _io_readLine
//   - FS effect: _fs_readFile, _fs_writeFile, _fs_exists
func (br *BuiltinRegistry) registerEffectBuiltins() {
	// IO effect builtins
	br.builtins["_io_print"] = &eval.BuiltinFunction{
		Name: "_io_print",
		Fn: func(args []eval.Value) (eval.Value, error) {
			ctx := br.getEffContext()
			if ctx == nil {
				return nil, fmt.Errorf("_io_print: no effect context available")
			}
			return effects.Call(ctx, "IO", "print", args)
		},
	}

	br.builtins["_io_println"] = &eval.BuiltinFunction{
		Name: "_io_println",
		Fn: func(args []eval.Value) (eval.Value, error) {
			ctx := br.getEffContext()
			if ctx == nil {
				return nil, fmt.Errorf("_io_println: no effect context available")
			}
			return effects.Call(ctx, "IO", "println", args)
		},
	}

	br.builtins["_io_readLine"] = &eval.BuiltinFunction{
		Name: "_io_readLine",
		Fn: func(args []eval.Value) (eval.Value, error) {
			ctx := br.getEffContext()
			if ctx == nil {
				return nil, fmt.Errorf("_io_readLine: no effect context available")
			}
			return effects.Call(ctx, "IO", "readLine", args)
		},
	}

	// FS effect builtins
	br.builtins["_fs_readFile"] = &eval.BuiltinFunction{
		Name: "_fs_readFile",
		Fn: func(args []eval.Value) (eval.Value, error) {
			ctx := br.getEffContext()
			if ctx == nil {
				return nil, fmt.Errorf("_fs_readFile: no effect context available")
			}
			return effects.Call(ctx, "FS", "readFile", args)
		},
	}

	br.builtins["_fs_writeFile"] = &eval.BuiltinFunction{
		Name: "_fs_writeFile",
		Fn: func(args []eval.Value) (eval.Value, error) {
			ctx := br.getEffContext()
			if ctx == nil {
				return nil, fmt.Errorf("_fs_writeFile: no effect context available")
			}
			return effects.Call(ctx, "FS", "writeFile", args)
		},
	}

	br.builtins["_fs_exists"] = &eval.BuiltinFunction{
		Name: "_fs_exists",
		Fn: func(args []eval.Value) (eval.Value, error) {
			ctx := br.getEffContext()
			if ctx == nil {
				return nil, fmt.Errorf("_fs_exists: no effect context available")
			}
			return effects.Call(ctx, "FS", "exists", args)
		},
	}
}

// getEffContext retrieves the EffContext from the evaluator
//
// Returns:
//   - The EffContext if available, nil otherwise
func (br *BuiltinRegistry) getEffContext() *effects.EffContext {
	if br.evaluator == nil {
		return nil
	}
	ctx := br.evaluator.GetEffContext()
	if ctx == nil {
		return nil
	}
	effCtx, ok := ctx.(*effects.EffContext)
	if !ok {
		return nil
	}
	return effCtx
}
