package runtime

import (
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// getArity returns the arity (number of parameters) of a value
//
// Parameters:
//   - val: The value to check (should be a FunctionValue)
//
// Returns:
//   - The number of parameters the function takes
//   - An error if the value is not a function
func GetArity(val eval.Value) (int, error) {
	fn, ok := val.(*eval.FunctionValue)
	if !ok {
		return 0, fmt.Errorf("value is not a function (got %T)", val)
	}

	return len(fn.Params), nil
}

// GetExportNames returns a sorted list of export names from a module instance
//
// This is a helper function for error messages.
//
// Parameters:
//   - inst: The module instance
//
// Returns:
//   - A slice of export names
func GetExportNames(inst *ModuleInstance) []string {
	return inst.ListExports()
}
