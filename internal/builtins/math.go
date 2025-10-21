package builtins

import (
	"fmt"
	"math"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Math, comparison, logic, and conversion builtins for AILANG
// These provide the fundamental numeric and boolean operations

func init() {
	registerArithmetic()
	registerComparisons()
	registerLogic()
	registerConversions()
}

// ============================================================================
// Arithmetic Builtins (Int and Float operations)
// ============================================================================

func registerArithmetic() {
	// Integer arithmetic
	registerBuiltinWithMeta("add_Int", 2, true, intIntToInt(func(a, b int) int { return a + b }),
		"Add two integers", []string{"math", "arithmetic", "add", "plus"})
	registerBuiltinWithMeta("sub_Int", 2, true, intIntToInt(func(a, b int) int { return a - b }),
		"Subtract two integers", []string{"math", "arithmetic", "subtract", "minus"})
	registerBuiltinWithMeta("mul_Int", 2, true, intIntToInt(func(a, b int) int { return a * b }),
		"Multiply two integers", []string{"math", "arithmetic", "multiply", "times"})
	registerBuiltinWithMeta("div_Int", 2, true, intIntToIntErr(func(a, b int) (int, error) {
		if b == 0 {
			return 0, eval.NewRuntimeError("RT_DIV0", "Division by zero", nil)
		}
		return a / b, nil
	}), "Divide two integers (errors on division by zero)", []string{"math", "arithmetic", "divide"})
	registerBuiltinWithMeta("mod_Int", 2, true, intIntToIntErr(func(a, b int) (int, error) {
		if b == 0 {
			return 0, eval.NewRuntimeError("RT_DIV0", "Modulo by zero", nil)
		}
		return a % b, nil
	}), "Integer modulo operation (errors on modulo by zero)", []string{"math", "arithmetic", "modulo", "remainder"})
	registerBuiltinWithMeta("neg_Int", 1, true, intToInt(func(a int) int { return -a }),
		"Negate an integer", []string{"math", "arithmetic", "negate", "negative"})

	// Float arithmetic (with special IEEE 754 behavior)
	registerBuiltinWithMeta("add_Float", 2, true, floatFloatToFloat(func(a, b float64) float64 { return a + b }),
		"Add two floats", []string{"math", "arithmetic", "add", "plus", "float"})
	registerBuiltinWithMeta("sub_Float", 2, true, floatFloatToFloat(func(a, b float64) float64 { return a - b }),
		"Subtract two floats", []string{"math", "arithmetic", "subtract", "minus", "float"})
	registerBuiltinWithMeta("mul_Float", 2, true, floatFloatToFloat(func(a, b float64) float64 { return a * b }),
		"Multiply two floats", []string{"math", "arithmetic", "multiply", "times", "float"})
	registerBuiltinWithMeta("div_Float", 2, true, floatDivFloat,
		"Divide two floats (returns Inf for division by zero, IEEE 754)", []string{"math", "arithmetic", "divide", "float"})
	registerBuiltinWithMeta("mod_Float", 2, true, floatModFloat,
		"Float modulo operation (returns NaN for modulo by zero, IEEE 754)", []string{"math", "arithmetic", "modulo", "float"})
	registerBuiltinWithMeta("neg_Float", 1, true, floatNegFloat,
		"Negate a float", []string{"math", "arithmetic", "negate", "negative", "float"})
}

// floatDivFloat: division with IEEE 754 behavior (returns Inf for div-by-zero)
func floatDivFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.FloatValue)
	b := args[1].(*eval.FloatValue)
	if b.Value == 0.0 {
		// IEEE 754 behavior: return +/-Inf
		if a.Value >= 0 {
			return &eval.FloatValue{Value: math.Inf(1)}, nil
		}
		return &eval.FloatValue{Value: math.Inf(-1)}, nil
	}
	return &eval.FloatValue{Value: a.Value / b.Value}, nil
}

// floatModFloat: modulo with IEEE 754 behavior (returns NaN for mod-by-zero)
func floatModFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.FloatValue)
	b := args[1].(*eval.FloatValue)
	if b.Value == 0.0 {
		return &eval.FloatValue{Value: math.NaN()}, nil
	}
	return &eval.FloatValue{Value: math.Mod(a.Value, b.Value)}, nil
}

// floatNegFloat: negation
func floatNegFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a := args[0].(*eval.FloatValue)
	return &eval.FloatValue{Value: -a.Value}, nil
}

// Helper: wrap a simple int->int function
func intToInt(fn func(int) int) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: fn(a.Value)}, nil
	}
}

// Helper: wrap a simple (int,int)->int function
func intIntToInt(fn func(int, int) int) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		return &eval.IntValue{Value: fn(a.Value, b.Value)}, nil
	}
}

// Helper: wrap a (int,int)->(int,error) function
func intIntToIntErr(fn func(int, int) (int, error)) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		result, err := fn(a.Value, b.Value)
		if err != nil {
			return nil, err
		}
		return &eval.IntValue{Value: result}, nil
	}
}

// Helper: wrap a (float,float)->float function
func floatFloatToFloat(fn func(float64, float64) float64) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.FloatValue{Value: fn(a.Value, b.Value)}, nil
	}
}

// registerBuiltinWithMeta is a convenience wrapper for arithmetic builtins with metadata
func registerBuiltinWithMeta(name string, numArgs int, isPure bool, impl func(*effects.EffContext, []eval.Value) (eval.Value, error), description string, tags []string) {
	// Determine type based on name suffix
	var typeFunc func() types.Type
	var paramDocs []ParamDoc
	var returns string

	if strings.HasSuffix(name, "_Int") {
		if numArgs == 1 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Int()).Returns(T.Int()).Build()
			}
			paramDocs = []ParamDoc{{Name: "a", Description: "Integer operand"}}
			returns = "Integer result"
		} else if numArgs == 2 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Int(), T.Int()).Returns(T.Int()).Build()
			}
			paramDocs = []ParamDoc{
				{Name: "a", Description: "First integer operand"},
				{Name: "b", Description: "Second integer operand"},
			}
			returns = "Integer result"
		}
	} else if strings.HasSuffix(name, "_Float") {
		if numArgs == 1 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Float()).Returns(T.Float()).Build()
			}
			paramDocs = []ParamDoc{{Name: "a", Description: "Float operand"}}
			returns = "Float result"
		} else if numArgs == 2 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Float(), T.Float()).Returns(T.Float()).Build()
			}
			paramDocs = []ParamDoc{
				{Name: "a", Description: "First float operand"},
				{Name: "b", Description: "Second float operand"},
			}
			returns = "Float result"
		}
	}

	if typeFunc == nil {
		panic(fmt.Sprintf("cannot infer type for builtin %s", name))
	}

	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    name,
		NumArgs: numArgs,
		IsPure:  isPure,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params:      paramDocs,
			Returns:     returns,
			Since:       "v0.1.0",
			Stability:   StabilityStable,
			Tags:        tags,
			Category:    "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// ============================================================================
// Comparison Builtins (eq, ne, lt, le, gt, ge for Int, Float, String, Bool)
// ============================================================================

func registerComparisons() {
	// Int comparisons
	registerCmpWithMeta("eq_Int", func(a, b int) bool { return a == b },
		"Test if two integers are equal", []string{"comparison", "equality", "int"})
	registerCmpWithMeta("ne_Int", func(a, b int) bool { return a != b },
		"Test if two integers are not equal", []string{"comparison", "inequality", "int"})
	registerCmpWithMeta("lt_Int", func(a, b int) bool { return a < b },
		"Test if first integer is less than second", []string{"comparison", "ordering", "int"})
	registerCmpWithMeta("le_Int", func(a, b int) bool { return a <= b },
		"Test if first integer is less than or equal to second", []string{"comparison", "ordering", "int"})
	registerCmpWithMeta("gt_Int", func(a, b int) bool { return a > b },
		"Test if first integer is greater than second", []string{"comparison", "ordering", "int"})
	registerCmpWithMeta("ge_Int", func(a, b int) bool { return a >= b },
		"Test if first integer is greater than or equal to second", []string{"comparison", "ordering", "int"})

	// Float comparisons
	registerCmpFloatWithMeta("eq_Float", func(a, b float64) bool { return a == b },
		"Test if two floats are equal (IEEE 754 equality)", []string{"comparison", "equality", "float"})
	registerCmpFloatWithMeta("ne_Float", func(a, b float64) bool { return a != b },
		"Test if two floats are not equal (IEEE 754 equality)", []string{"comparison", "inequality", "float"})
	registerCmpFloatWithMeta("lt_Float", func(a, b float64) bool { return a < b },
		"Test if first float is less than second", []string{"comparison", "ordering", "float"})
	registerCmpFloatWithMeta("le_Float", func(a, b float64) bool { return a <= b },
		"Test if first float is less than or equal to second", []string{"comparison", "ordering", "float"})
	registerCmpFloatWithMeta("gt_Float", func(a, b float64) bool { return a > b },
		"Test if first float is greater than second", []string{"comparison", "ordering", "float"})
	registerCmpFloatWithMeta("ge_Float", func(a, b float64) bool { return a >= b },
		"Test if first float is greater than or equal to second", []string{"comparison", "ordering", "float"})

	// String comparisons
	registerCmpStringWithMeta("eq_String", func(a, b string) bool { return a == b },
		"Test if two strings are equal", []string{"comparison", "equality", "string"})
	registerCmpStringWithMeta("ne_String", func(a, b string) bool { return a != b },
		"Test if two strings are not equal", []string{"comparison", "inequality", "string"})
	registerCmpStringWithMeta("lt_String", func(a, b string) bool { return a < b },
		"Test if first string is lexicographically less than second", []string{"comparison", "ordering", "string", "lexicographic"})
	registerCmpStringWithMeta("le_String", func(a, b string) bool { return a <= b },
		"Test if first string is lexicographically less than or equal to second", []string{"comparison", "ordering", "string", "lexicographic"})
	registerCmpStringWithMeta("gt_String", func(a, b string) bool { return a > b },
		"Test if first string is lexicographically greater than second", []string{"comparison", "ordering", "string", "lexicographic"})
	registerCmpStringWithMeta("ge_String", func(a, b string) bool { return a >= b },
		"Test if first string is lexicographically greater than or equal to second", []string{"comparison", "ordering", "string", "lexicographic"})

	// Bool comparisons
	registerCmpBoolWithMeta("eq_Bool", func(a, b bool) bool { return a == b },
		"Test if two booleans are equal", []string{"comparison", "equality", "bool"})
	registerCmpBoolWithMeta("ne_Bool", func(a, b bool) bool { return a != b },
		"Test if two booleans are not equal", []string{"comparison", "inequality", "bool"})
}

// registerCmpWithMeta registers an Int comparison with metadata
func registerCmpWithMeta(name string, fn func(int, int) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int(), T.Int()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First integer"},
				{Name: "b", Description: "Second integer"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerCmpFloatWithMeta registers a Float comparison with metadata
func registerCmpFloatWithMeta(name string, fn func(float64, float64) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float(), T.Float()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First float"},
				{Name: "b", Description: "Second float"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerCmpStringWithMeta registers a String comparison with metadata
func registerCmpStringWithMeta(name string, fn func(string, string) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.StringValue)
		b := args[1].(*eval.StringValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First string"},
				{Name: "b", Description: "Second string"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerCmpBoolWithMeta registers a Bool comparison with metadata
func registerCmpBoolWithMeta(name string, fn func(bool, bool) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		b := args[1].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool(), T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First boolean"},
				{Name: "b", Description: "Second boolean"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// ============================================================================
// Logic Builtins (and, or, not)
// ============================================================================

func registerLogic() {
	registerLogicOpWithMeta("and_Bool", func(a, b bool) bool { return a && b },
		"Logical AND operation", []string{"logic", "boolean", "and", "conjunction"})
	registerLogicOpWithMeta("or_Bool", func(a, b bool) bool { return a || b },
		"Logical OR operation", []string{"logic", "boolean", "or", "disjunction"})
	registerLogicUnaryWithMeta("not_Bool", func(a bool) bool { return !a },
		"Logical NOT operation (negation)", []string{"logic", "boolean", "not", "negation"})
}

// registerLogicOpWithMeta registers a binary logic operation with metadata
func registerLogicOpWithMeta(name string, fn func(bool, bool) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		b := args[1].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool(), T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First boolean"},
				{Name: "b", Description: "Second boolean"},
			},
			Returns:   "Result of logical operation",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "logic",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerLogicUnaryWithMeta registers a unary logic operation with metadata
func registerLogicUnaryWithMeta(name string, fn func(bool) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 1,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "Boolean value"},
			},
			Returns:   "Negated boolean value",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "logic",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// ============================================================================
// Conversion Builtins (intToFloat, floatToInt)
// ============================================================================

func registerConversions() {
	// intToFloat
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		return &eval.FloatValue{Value: float64(a.Value)}, nil
	}
	type1 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int()).Returns(T.Float()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    "intToFloat",
		NumArgs: 1,
		IsPure:  true,
		Type:    type1,
		Impl:    impl1,
		Metadata: &BuiltinMetadata{
			Description: "Convert an integer to a float",
			Params: []ParamDoc{
				{Name: "n", Description: "Integer to convert"},
			},
			Returns: "Float representation of the integer",
			Examples: []Example{
				{Code: "intToFloat(42)", Description: "Returns 42.0"},
				{Code: "intToFloat(-5)", Description: "Returns -5.0"},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"conversion", "type-conversion", "int", "float"},
			Category:  "conversion",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register intToFloat: %v", err))
	}

	// floatToInt
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		return &eval.IntValue{Value: int(a.Value)}, nil
	}
	type2 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float()).Returns(T.Int()).Build()
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    "floatToInt",
		NumArgs: 1,
		IsPure:  true,
		Type:    type2,
		Impl:    impl2,
		Metadata: &BuiltinMetadata{
			Description: "Convert a float to an integer (truncates towards zero)",
			LongDesc:    "Converts a floating-point number to an integer by truncating the decimal part. Positive numbers round down, negative numbers round up (towards zero).",
			Params: []ParamDoc{
				{Name: "x", Description: "Float to convert"},
			},
			Returns: "Integer part of the float (truncated towards zero)",
			Examples: []Example{
				{Code: "floatToInt(42.9)", Description: "Returns 42"},
				{Code: "floatToInt(-5.7)", Description: "Returns -5"},
				{Code: "floatToInt(3.0)", Description: "Returns 3"},
			},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"conversion", "type-conversion", "float", "int", "truncate"},
			Category:  "conversion",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register floatToInt: %v", err))
	}
}
