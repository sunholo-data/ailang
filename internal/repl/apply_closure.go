package repl

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/runtime"
)

// ApplyClosure invokes an AILANG closure (FunctionValue) with the given arguments.
// For multi-arg closures, it applies arguments one at a time (curried application).
// This is used by the WASM bridge to invoke AILANG callbacks from JavaScript.
//
// M-WASM-CLOSURE-ENV: When the REPL has a ModuleRegistry, ApplyClosure creates
// an evaluator with a RegistryResolver so that cross-module VarGlobal references
// (imported functions) resolve correctly. Without this, only same-module Var
// references work (via the closure's captured env chain).
func (r *REPL) ApplyClosure(fn eval.Value, args []eval.Value) (eval.Value, error) {
	if fn == nil {
		return nil, fmt.Errorf("ApplyClosure: nil function")
	}

	// Choose evaluator: if we have a module registry, create one with RegistryResolver
	// so that closures containing VarGlobal references (imported functions) resolve correctly.
	// This mirrors the pattern in InvokeExport (module_registry.go:715-730).
	callEvaluator := r.evaluator
	if r.registry != nil {
		callEvaluator = r.makeRegistryEvaluator()
	}

	if len(args) == 0 {
		// Zero-arg call: apply with unit
		return callEvaluator.CallValue(fn, &eval.UnitValue{})
	}

	// Apply arguments one at a time (curried)
	current := fn
	for i, arg := range args {
		result, err := callEvaluator.CallValue(current, arg)
		if err != nil {
			return nil, fmt.Errorf("ApplyClosure: error applying arg %d: %w", i, err)
		}
		current = result
	}

	return current, nil
}

// makeRegistryEvaluator creates an evaluator configured with a RegistryResolver
// that can resolve both builtin functions and cross-module imports.
// This is needed when closures from loaded modules reference imported functions
// (elaborated as core.VarGlobal), which the REPL's default BuiltinOnlyResolver cannot handle.
func (r *REPL) makeRegistryEvaluator() *eval.CoreEvaluator {
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	registryResolver := NewRegistryResolver(r.registry, builtinRegistry)
	evaluator.SetGlobalResolver(registryResolver)

	// Propagate effect context so effects work in the closure
	if r.effContext != nil {
		evaluator.SetEffContext(r.effContext)
	}

	// Enable experimental binop shim (same as InvokeExport)
	evaluator.SetExperimentalBinopShim(true)

	// Register type class dictionaries for arithmetic/comparison operations
	registerPreludeInstancesForEvaluator(evaluator)

	return evaluator
}
