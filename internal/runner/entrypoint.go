package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/runtime/argdecode"
	"github.com/sunholo-data/ailang/internal/types"
)

// ModuleExecParams contains all parameters needed for module execution.
type ModuleExecParams struct {
	Filename          string
	Iface             *iface.Iface
	Modules           map[string]*loader.LoadedModule
	Entry             string
	ArgsJSON          string
	Print             bool
	NoPrint           bool
	MaxRecursionDepth int

	// M-BYTECODE-2D M3: bytecode VM dispatch path. When BytecodeMode is set,
	// ExecuteModuleEntrypoint compiles the entry function (and its module)
	// to bytecode and runs it on the VM, with an evaluator-bridge for any
	// function the compiler couldn't lower. PipelineResult must be non-nil
	// in this mode — the bytecode compile reuses its AST/Core/CoreTI rather
	// than re-running the pipeline.
	BytecodeMode   bool
	StrictBytecode bool
	Quiet          bool
	PipelineResult *pipeline.Result
}

// ExecuteModuleEntrypoint loads a module, resolves the entrypoint, and executes it.
func ExecuteModuleEntrypoint(rt *runtime.ModuleRuntime, params ModuleExecParams) error {
	// Set recursion depth limit
	if params.MaxRecursionDepth > 0 {
		rt.GetEvaluator().SetMaxRecursionDepth(params.MaxRecursionDepth)
	}

	// Pre-load modules from pipeline result
	if params.Modules != nil {
		for path, loaded := range params.Modules {
			rt.PreloadModule(path, loaded)
		}
	}

	// Load and evaluate module
	inst, err := rt.LoadAndEvaluate(params.Iface.Module)
	if err != nil {
		return fmt.Errorf("module evaluation failed: %w", err)
	}

	// Resolve entrypoint
	entry := params.Entry
	fnExport, exists := params.Iface.Exports[entry]

	if !exists {
		// Auto-select entrypoint if possible
		if entry == "main" {
			var zeroArgFuncs []string
			for name, export := range params.Iface.Exports {
				if export.Type != nil {
					if fnType, isFn := export.Type.Type.(*types.TFunc2); isFn {
						if len(fnType.Params) == 0 {
							zeroArgFuncs = append(zeroArgFuncs, name)
						}
					}
				}
			}

			// Case 1: Exactly one zero-arg function
			if len(zeroArgFuncs) == 1 {
				entry = zeroArgFuncs[0]
				fnExport = params.Iface.Exports[entry]
				exists = true
			} else if len(zeroArgFuncs) > 1 {
				// Case 2: Multiple zero-arg functions, try "test"
				for _, name := range zeroArgFuncs {
					if name == "test" {
						entry = name
						fnExport = params.Iface.Exports[entry]
						exists = true
						break
					}
				}
			}
		}

		if !exists {
			exportNames := []string{}
			for name := range params.Iface.Exports {
				exportNames = append(exportNames, name)
			}
			return fmt.Errorf("entrypoint '%s' not found in module\nAvailable exports: %v", entry, exportNames)
		}
	}

	// Check function type
	scheme := fnExport.Type
	if scheme == nil {
		return fmt.Errorf("entrypoint '%s' has no type information", entry)
	}

	fnType, isFn := scheme.Type.(*types.TFunc2)
	if !isFn {
		return fmt.Errorf("entrypoint '%s' is not a function (has type %s)", entry, scheme.Type)
	}

	// Get entrypoint value
	entrypointVal, err := inst.GetExport(entry)
	if err != nil {
		exportNames := runtime.GetExportNames(inst)
		return fmt.Errorf("entrypoint '%s' not found in module %s\nAvailable exports: %s",
			entry, params.Iface.Module, strings.Join(exportNames, ", "))
	}

	// Check arity
	arity, err := runtime.GetArity(entrypointVal)
	if err != nil {
		return fmt.Errorf("entrypoint '%s' is not a function: %w", entry, err)
	}
	if arity > 1 {
		return fmt.Errorf("entrypoint '%s' takes %d parameters. v0.2.0 supports 0 or 1.\nSuggestion: wrap as 'wrapper(p:{...}) -> ...' and pass --args-json", entry, arity)
	}

	// Decode arguments
	args, err := decodeEntrypointArgs(params.ArgsJSON, fnType, entry)
	if err != nil {
		return err
	}

	// M-BYTECODE-2D M3: bytecode VM dispatch path. The bridge is wired in
	// non-strict mode so that EvalOnly functions (effectful, polymorphic-
	// dictionary, ADT-shaped) trap back to the evaluator transparently. In
	// strict mode the bridge is intentionally NOT wired: any call into an
	// EvalOnly stub becomes a hard VM error so the strict gate fails loudly.
	if params.BytecodeMode {
		ranOnVM, vmErr := tryRunEntryViaVM(rt, inst, params, entry, args)
		if ranOnVM {
			return nil
		}
		if params.StrictBytecode {
			return fmt.Errorf("bytecode execution failed: %w", vmErr)
		}
		if !params.Quiet {
			fmt.Fprintf(os.Stderr, "%s bytecode path unavailable (%v); falling back to evaluator\n", yellow("⚠"), vmErr)
		}
		// Fall through to evaluator path below.
	}

	// Call the entrypoint function
	execResult, err := runtime.CallEntrypoint(rt, inst, entry, args)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Print result if not Unit and not suppressed
	if execResult.Type() != "unit" && !params.NoPrint {
		if params.Print {
			fmt.Println(execResult.String())
		}
	}

	return nil
}

// decodeEntrypointArgs decodes JSON arguments based on function type signature
func decodeEntrypointArgs(argsJSON string, fnType *types.TFunc2, entry string) ([]eval.Value, error) {
	var args []eval.Value

	if len(fnType.Params) == 0 {
		// M-DX10: Zero-arg functions in surface syntax are actually unit-arg in core
		// Pass unit argument for entry invocation
		if argsJSON != "null" {
			return nil, fmt.Errorf("entrypoint '%s' takes no arguments, but --args-json was provided", entry)
		}
		args = []eval.Value{&eval.UnitValue{}} // Unit-argument model
	} else if len(fnType.Params) == 1 {
		// Single-arg function - decode JSON to match parameter type
		argVal, err := argdecode.DecodeJSON(argsJSON, fnType.Params[0])
		if err != nil {
			return nil, fmt.Errorf("failed to decode arguments: %w", err)
		}
		args = []eval.Value{argVal}
	} else {
		// Multi-arg functions not yet supported
		return nil, fmt.Errorf("entrypoint '%s' has %d parameters (only 0 or 1 supported in v0.2.0)", entry, len(fnType.Params))
	}

	return args, nil
}

// PrintNonModuleResult prints the result of non-module file execution.
func PrintNonModuleResult(value eval.Value, print, noprint bool) {
	if value != nil && value.Type() != "unit" && !noprint {
		if print {
			fmt.Println(value.String())
		}
	}
}
