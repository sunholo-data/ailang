package repl

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// ApplyClosure invokes an AILANG closure (FunctionValue) with the given arguments.
// For multi-arg closures, it applies arguments one at a time (curried application).
// This is used by the WASM bridge to invoke AILANG callbacks from JavaScript.
func (r *REPL) ApplyClosure(fn eval.Value, args []eval.Value) (eval.Value, error) {
	if fn == nil {
		return nil, fmt.Errorf("ApplyClosure: nil function")
	}

	if len(args) == 0 {
		// Zero-arg call: apply with unit
		return r.evaluator.CallValue(fn, &eval.UnitValue{})
	}

	// Apply arguments one at a time (curried)
	current := fn
	for i, arg := range args {
		result, err := r.evaluator.CallValue(current, arg)
		if err != nil {
			return nil, fmt.Errorf("ApplyClosure: error applying arg %d: %w", i, err)
		}
		current = result
	}

	return current, nil
}
