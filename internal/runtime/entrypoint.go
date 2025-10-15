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

// CallEntrypoint calls an exported entrypoint function from a module
//
// This method handles function invocation for module entrypoints, supporting
// both 0-arg and multi-arg functions. The function is executed with a properly
// configured evaluator that can resolve cross-module references.
//
// Parameters:
//   - rt: The ModuleRuntime instance
//   - inst: The module instance containing the entrypoint
//   - name: The name of the entrypoint function
//   - args: The arguments to pass to the function (can be empty for 0-arg)
//
// Returns:
//   - The result value from executing the function
//   - An error if the entrypoint doesn't exist, isn't a function, or execution fails
//
// Example:
//
//	result, err := CallEntrypoint(rt, inst, "main", []eval.Value{})
//	if err != nil {
//	    return err
//	}
//	fmt.Println(result.String())
func CallEntrypoint(rt *ModuleRuntime, inst *ModuleInstance, name string, args []eval.Value) (eval.Value, error) {
	// 1. Get the entrypoint from exports
	entrypoint, err := inst.GetExport(name)
	if err != nil {
		return nil, err
	}

	// 2. Verify it's a function
	fn, ok := entrypoint.(*eval.FunctionValue)
	if !ok {
		return nil, fmt.Errorf("entrypoint '%s' is not a function (got %T)", name, entrypoint)
	}

	// 3. Set up resolver for cross-module references
	resolver := newModuleGlobalResolver(inst, rt)
	rt.evaluator.SetGlobalResolver(resolver)

	// 4. Call the function using the evaluator's CallFunction method
	return rt.evaluator.CallFunction(fn, args)
}
