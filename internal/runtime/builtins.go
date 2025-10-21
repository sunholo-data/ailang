package runtime

import (
	"fmt"

	"github.com/sunholo/ailang/internal/builtins"
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

	// Use new spec-based registry (M-DX1 migration complete in v0.3.10)
	br.registerFromSpecRegistry()

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

// registerFromSpecRegistry registers builtins from the new spec-based registry
// This is the new centralized registration path (enabled with AILANG_BUILTINS_REGISTRY=1)
func (br *BuiltinRegistry) registerFromSpecRegistry() {
	specs := builtins.AllSpecs()

	for name, spec := range specs {
		// Capture spec for closure
		builtinSpec := spec

		br.builtins[name] = &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				ctx := br.getEffContext()
				if ctx == nil && !builtinSpec.IsPure {
					return nil, fmt.Errorf("%s: no effect context available", builtinSpec.Name)
				}
				return builtinSpec.Impl(ctx, args)
			},
		}
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
