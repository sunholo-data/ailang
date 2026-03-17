package repl

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/types"
)

// CallExport formats a function call expression string for use with the REPL.
// This returns the expression that can be evaluated via ProcessExpression.
// Arguments are converted to their AILANG string representation.
func (mr *ModuleRegistry) CallExport(moduleName, funcName string, args []eval.Value) (string, error) {
	// Verify the export exists
	_, err := mr.GetExport(moduleName, funcName)
	if err != nil {
		return "", err
	}

	// Build curried call expression: moduleName.funcName(arg1)(arg2)...
	// First we need to import the module, then call the function
	expr := funcName
	for _, arg := range args {
		argStr := formatArgument(arg)
		expr = fmt.Sprintf("%s(%s)", expr, argStr)
	}

	return expr, nil
}

// InvokeExport calls an exported function directly with the provided arguments.
// This method bypasses REPL string evaluation and calls the function closure
// directly, ensuring that captured imports (like decode, encode from std/json)
// are properly resolved from the function's environment.
//
// Returns the result value and any error that occurred during execution.
func (mr *ModuleRegistry) InvokeExport(moduleName, funcName string, args []eval.Value) (eval.Value, error) {
	// Get the export
	export, err := mr.GetExport(moduleName, funcName)
	if err != nil {
		return nil, err
	}

	// Get the function value
	fn, ok := export.Value.(*eval.FunctionValue)
	if !ok {
		return nil, fmt.Errorf("export %s.%s is not a function (got %T)", moduleName, funcName, export.Value)
	}

	// Create an evaluator with the RegistryResolver so that any global references
	// (builtins, imports from other modules) can be resolved during execution
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	registryResolver := NewRegistryResolver(mr, builtinRegistry)
	evaluator.SetGlobalResolver(registryResolver)

	// Set effect context if available (enables AI, IO, and other effects in WASM)
	// M-ITERATIVE-LIST: Always set an EffContext so FnCaller/FnCallerN are wired
	// for iterative builtins (_list_map, _list_foldl, etc.)
	if mr.effContext != nil {
		evaluator.SetEffContext(mr.effContext)
	} else {
		evaluator.SetEffContext(effects.NewEffContext(nil))
	}

	// Enable experimental binop shim (handles float equality until OpLowering is complete)
	evaluator.SetExperimentalBinopShim(true)

	// Register type class dictionaries (Num, Eq, Ord, Fractional) so that
	// module functions using arithmetic, comparisons, or string operations work.
	registerPreludeInstancesForEvaluator(evaluator)

	// Apply arguments, handling both multi-param and curried functions.
	// Multi-param: func f(a, b) compiled with Params=["a","b"] -- needs all args at once.
	// Curried: func f(a)(b) compiled as nested lambdas -- apply one at a time.
	var result eval.Value = fn
	remaining := args
	for len(remaining) > 0 {
		funcVal, isFn := result.(*eval.FunctionValue)
		if !isFn {
			return nil, fmt.Errorf("too many arguments: value is not a function (got %T) with %d args remaining", result, len(remaining))
		}

		arity := len(funcVal.Params)
		if arity <= 0 {
			arity = 1
		}

		if arity <= len(remaining) {
			// Apply arity-many arguments at once
			result, err = evaluator.CallFunction(funcVal, remaining[:arity])
			if err != nil {
				applied := len(args) - len(remaining)
				return nil, fmt.Errorf("error applying argument(s) %d-%d: %w", applied+1, applied+arity, err)
			}
			remaining = remaining[arity:]
		} else {
			// More params than remaining args -- apply what we have one at a time
			// (shouldn't normally happen, but handle gracefully)
			result, err = evaluator.CallFunction(funcVal, remaining)
			if err != nil {
				return nil, fmt.Errorf("error applying final arguments: %w", err)
			}
			remaining = nil
		}
	}

	return result, nil
}

// registerPreludeInstances registers standard type class instances with the linker.
// This mirrors the instances registered in the REPL.
func registerPreludeInstances(linker *link.Linker) {
	// Helper to create builtin functions
	wrapInt2 := func(name string, f func(int64, int64) int64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("expected 2 arguments")
				}
				x, ok1 := args[0].(*eval.IntValue)
				y, ok2 := args[1].(*eval.IntValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("expected int arguments")
				}
				return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
			},
		}
	}

	wrapFloat2 := func(name string, f func(float64, float64) float64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("expected 2 arguments")
				}
				x, ok1 := args[0].(*eval.FloatValue)
				y, ok2 := args[1].(*eval.FloatValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("expected float arguments")
				}
				return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
			},
		}
	}

	// Num[Int]
	numInt := core.DictValue{
		TypeClass: "Num",
		Type:      "Int",
		Methods: map[string]interface{}{
			"add": wrapInt2("add", func(a, b int64) int64 { return a + b }),
			"sub": wrapInt2("sub", func(a, b int64) int64 { return a - b }),
			"mul": wrapInt2("mul", func(a, b int64) int64 { return a * b }),
			"div": wrapInt2("div", func(a, b int64) int64 { return a / b }),
		},
	}

	// Num[Float]
	numFloat := core.DictValue{
		TypeClass: "Num",
		Type:      "Float",
		Methods: map[string]interface{}{
			"add": wrapFloat2("add", func(a, b float64) float64 { return a + b }),
			"sub": wrapFloat2("sub", func(a, b float64) float64 { return a - b }),
			"mul": wrapFloat2("mul", func(a, b float64) float64 { return a * b }),
			"div": wrapFloat2("div", func(a, b float64) float64 { return a / b }),
		},
	}

	// Eq[Int]
	eqInt := core.DictValue{
		TypeClass: "Eq", Type: "Int",
		Methods: map[string]interface{}{
			"eq": &eval.BuiltinFunction{Name: "eq_Int", Fn: func(args []eval.Value) (eval.Value, error) {
				x, ok1 := args[0].(*eval.IntValue)
				y, ok2 := args[1].(*eval.IntValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("eq_Int: expected IntValue for arg 0, got %T", args[0])
				}
				return &eval.BoolValue{Value: x.Value == y.Value}, nil
			}},
			"neq": &eval.BuiltinFunction{Name: "neq_Int", Fn: func(args []eval.Value) (eval.Value, error) {
				x, ok1 := args[0].(*eval.IntValue)
				y, ok2 := args[1].(*eval.IntValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("neq_Int: expected IntValue, got %T", args[0])
				}
				return &eval.BoolValue{Value: x.Value != y.Value}, nil
			}},
		},
	}

	// Eq[String]
	eqString := core.DictValue{
		TypeClass: "Eq", Type: "String",
		Methods: map[string]interface{}{
			"eq": &eval.BuiltinFunction{Name: "eq_String", Fn: func(args []eval.Value) (eval.Value, error) {
				x, ok1 := args[0].(*eval.StringValue)
				y, ok2 := args[1].(*eval.StringValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("eq_String: expected StringValue, got %T", args[0])
				}
				return &eval.BoolValue{Value: x.Value == y.Value}, nil
			}},
			"neq": &eval.BuiltinFunction{Name: "neq_String", Fn: func(args []eval.Value) (eval.Value, error) {
				x, ok1 := args[0].(*eval.StringValue)
				y, ok2 := args[1].(*eval.StringValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("neq_String: expected StringValue, got %T", args[0])
				}
				return &eval.BoolValue{Value: x.Value != y.Value}, nil
			}},
		},
	}

	// Register with canonical keys
	for methodName := range numInt.Methods {
		key := types.MakeDictionaryKey("prelude", "Num", &types.TCon{Name: "Int"}, methodName)
		linker.AddDictionary(key, numInt)
	}

	for methodName := range numFloat.Methods {
		key := types.MakeDictionaryKey("prelude", "Num", &types.TCon{Name: "Float"}, methodName)
		linker.AddDictionary(key, numFloat)
	}

	for methodName := range eqInt.Methods {
		key := types.MakeDictionaryKey("prelude", "Eq", &types.TCon{Name: "Int"}, methodName)
		linker.AddDictionary(key, eqInt)
	}

	for methodName := range eqString.Methods {
		key := types.MakeDictionaryKey("prelude", "Eq", &types.TCon{Name: "String"}, methodName)
		linker.AddDictionary(key, eqString)
	}
}

// registerPreludeInstancesForEvaluator registers all type class dictionaries
// (Num, Eq, Ord, Fractional) with a CoreEvaluator so that functions using
// arithmetic, comparisons, or string operations work via InvokeExport.
func registerPreludeInstancesForEvaluator(evaluator *eval.CoreEvaluator) {
	// --- Wrapper helpers ---
	wrapInt2 := func(name string, f func(int64, int64) int64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
		}}
	}

	wrapFloat2 := func(name string, f func(float64, float64) float64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
		}}
	}

	wrapIntCmp2 := func(name string, f func(int64, int64) bool) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.BoolValue{Value: f(int64(x.Value), int64(y.Value))}, nil
		}}
	}

	wrapFloatCmp2 := func(name string, f func(float64, float64) bool) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{Name: name, Fn: func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.BoolValue{Value: f(x.Value, y.Value)}, nil
		}}
	}

	// --- Build all type class instances ---
	instances := map[string]core.DictValue{
		"Num[Int]": {
			TypeClass: "Num", Type: "Int",
			Methods: map[string]interface{}{
				"add": wrapInt2("add", func(a, b int64) int64 { return a + b }),
				"sub": wrapInt2("sub", func(a, b int64) int64 { return a - b }),
				"mul": wrapInt2("mul", func(a, b int64) int64 { return a * b }),
				"div": wrapInt2("div", func(a, b int64) int64 { return a / b }),
			},
		},
		"Num[Float]": {
			TypeClass: "Num", Type: "Float",
			Methods: map[string]interface{}{
				"add": wrapFloat2("add", func(a, b float64) float64 { return a + b }),
				"sub": wrapFloat2("sub", func(a, b float64) float64 { return a - b }),
				"mul": wrapFloat2("mul", func(a, b float64) float64 { return a * b }),
				"div": wrapFloat2("div", func(a, b float64) float64 { return a / b }),
			},
		},
		"Eq[Int]": {
			TypeClass: "Eq", Type: "Int",
			Methods: map[string]interface{}{
				"eq":  wrapIntCmp2("eq", func(a, b int64) bool { return a == b }),
				"neq": wrapIntCmp2("neq", func(a, b int64) bool { return a != b }),
			},
		},
		"Eq[Float]": {
			TypeClass: "Eq", Type: "Float",
			Methods: map[string]interface{}{
				"eq":  wrapFloatCmp2("eq", func(a, b float64) bool { return a == b }),
				"neq": wrapFloatCmp2("neq", func(a, b float64) bool { return a != b }),
			},
		},
		"Eq[String]": {
			TypeClass: "Eq", Type: "String",
			Methods: map[string]interface{}{
				"eq": &eval.BuiltinFunction{Name: "eq_String", Fn: func(args []eval.Value) (eval.Value, error) {
					if len(args) != 2 {
						return nil, fmt.Errorf("eq_String: expected 2 arguments")
					}
					x, ok1 := args[0].(*eval.StringValue)
					y, ok2 := args[1].(*eval.StringValue)
					if !ok1 || !ok2 {
						return nil, fmt.Errorf("eq_String: expected StringValue arguments, got %T and %T", args[0], args[1])
					}
					return &eval.BoolValue{Value: x.Value == y.Value}, nil
				}},
				"neq": &eval.BuiltinFunction{Name: "neq_String", Fn: func(args []eval.Value) (eval.Value, error) {
					if len(args) != 2 {
						return nil, fmt.Errorf("neq_String: expected 2 arguments")
					}
					x, ok1 := args[0].(*eval.StringValue)
					y, ok2 := args[1].(*eval.StringValue)
					if !ok1 || !ok2 {
						return nil, fmt.Errorf("neq_String: expected StringValue arguments, got %T and %T", args[0], args[1])
					}
					return &eval.BoolValue{Value: x.Value != y.Value}, nil
				}},
			},
		},
		"Ord[Int]": {
			TypeClass: "Ord", Type: "Int",
			Methods: map[string]interface{}{
				"lt":  wrapIntCmp2("lt", func(a, b int64) bool { return a < b }),
				"lte": wrapIntCmp2("lte", func(a, b int64) bool { return a <= b }),
				"gt":  wrapIntCmp2("gt", func(a, b int64) bool { return a > b }),
				"gte": wrapIntCmp2("gte", func(a, b int64) bool { return a >= b }),
			},
			Provides: []string{"Eq[Int]"},
		},
		"Ord[Float]": {
			TypeClass: "Ord", Type: "Float",
			Methods: map[string]interface{}{
				"lt":  wrapFloatCmp2("lt", func(a, b float64) bool { return a < b }),
				"lte": wrapFloatCmp2("lte", func(a, b float64) bool { return a <= b }),
				"gt":  wrapFloatCmp2("gt", func(a, b float64) bool { return a > b }),
				"gte": wrapFloatCmp2("gte", func(a, b float64) bool { return a >= b }),
			},
			Provides: []string{"Eq[Float]"},
		},
	}

	// Register each instance with canonical keys for evaluator lookups
	for _, dict := range instances {
		typeForKey := &types.TCon{Name: dict.Type}
		for methodName := range dict.Methods {
			canonicalKey := types.MakeDictionaryKey("prelude", dict.TypeClass, typeForKey, methodName)
			evaluator.AddDictionary(canonicalKey, dict)
		}
		// Also register the base dictionary (no method name) for lookups
		baseKey := types.MakeDictionaryKey("prelude", dict.TypeClass, typeForKey, "")
		evaluator.AddDictionary(baseKey, dict)
	}
}
