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
	registerTrigonometry()
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

// ============================================================================
// Trigonometry and Advanced Math Builtins (v0.5.10)
// ============================================================================

func registerTrigonometry() {
	// sin: float -> float
	registerTrigFunc("_math_sin", math.Sin,
		"Compute sine of angle in radians",
		[]ParamDoc{{Name: "radians", Description: "Angle in radians"}},
		"Sine of the angle",
		[]Example{
			{Code: "sin(0.0)", Description: "Returns 0.0"},
			{Code: "sin(PI() / 2.0)", Description: "Returns 1.0"},
		},
		[]string{"math", "trigonometry", "sin", "sine"})

	// cos: float -> float
	registerTrigFunc("_math_cos", math.Cos,
		"Compute cosine of angle in radians",
		[]ParamDoc{{Name: "radians", Description: "Angle in radians"}},
		"Cosine of the angle",
		[]Example{
			{Code: "cos(0.0)", Description: "Returns 1.0"},
			{Code: "cos(PI())", Description: "Returns -1.0"},
		},
		[]string{"math", "trigonometry", "cos", "cosine"})

	// tan: float -> float
	registerTrigFunc("_math_tan", math.Tan,
		"Compute tangent of angle in radians",
		[]ParamDoc{{Name: "radians", Description: "Angle in radians"}},
		"Tangent of the angle",
		[]Example{
			{Code: "tan(0.0)", Description: "Returns 0.0"},
			{Code: "tan(PI() / 4.0)", Description: "Returns 1.0"},
		},
		[]string{"math", "trigonometry", "tan", "tangent"})

	// asin: float -> float
	registerTrigFunc("_math_asin", math.Asin,
		"Compute arcsine (inverse sine) returning radians",
		[]ParamDoc{{Name: "x", Description: "Value in range [-1, 1]"}},
		"Angle in radians in range [-PI/2, PI/2]",
		[]Example{
			{Code: "asin(0.0)", Description: "Returns 0.0"},
			{Code: "asin(1.0)", Description: "Returns PI/2"},
		},
		[]string{"math", "trigonometry", "asin", "arcsine", "inverse"})

	// acos: float -> float
	registerTrigFunc("_math_acos", math.Acos,
		"Compute arccosine (inverse cosine) returning radians",
		[]ParamDoc{{Name: "x", Description: "Value in range [-1, 1]"}},
		"Angle in radians in range [0, PI]",
		[]Example{
			{Code: "acos(1.0)", Description: "Returns 0.0"},
			{Code: "acos(0.0)", Description: "Returns PI/2"},
		},
		[]string{"math", "trigonometry", "acos", "arccosine", "inverse"})

	// atan: float -> float
	registerTrigFunc("_math_atan", math.Atan,
		"Compute arctangent (inverse tangent) returning radians",
		[]ParamDoc{{Name: "x", Description: "Any float value"}},
		"Angle in radians in range [-PI/2, PI/2]",
		[]Example{
			{Code: "atan(0.0)", Description: "Returns 0.0"},
			{Code: "atan(1.0)", Description: "Returns PI/4"},
		},
		[]string{"math", "trigonometry", "atan", "arctangent", "inverse"})

	// atan2: (float, float) -> float
	registerTrigFunc2("_math_atan2", math.Atan2,
		"Compute two-argument arctangent for angle calculation",
		[]ParamDoc{
			{Name: "y", Description: "Y coordinate"},
			{Name: "x", Description: "X coordinate"},
		},
		"Angle in radians in range [-PI, PI]",
		[]Example{
			{Code: "atan2(1.0, 1.0)", Description: "Returns PI/4"},
			{Code: "atan2(0.0, 1.0)", Description: "Returns 0.0"},
		},
		[]string{"math", "trigonometry", "atan2", "angle"})

	// sqrt: float -> float
	registerTrigFunc("_math_sqrt", math.Sqrt,
		"Compute square root",
		[]ParamDoc{{Name: "x", Description: "Non-negative float value"}},
		"Square root of x (NaN if x < 0)",
		[]Example{
			{Code: "sqrt(4.0)", Description: "Returns 2.0"},
			{Code: "sqrt(2.0)", Description: "Returns ~1.414"},
		},
		[]string{"math", "sqrt", "square", "root"})

	// pow: (float, float) -> float
	registerTrigFunc2("_math_pow", math.Pow,
		"Compute x raised to the power y",
		[]ParamDoc{
			{Name: "x", Description: "Base value"},
			{Name: "y", Description: "Exponent"},
		},
		"x^y",
		[]Example{
			{Code: "pow(2.0, 3.0)", Description: "Returns 8.0"},
			{Code: "pow(4.0, 0.5)", Description: "Returns 2.0"},
		},
		[]string{"math", "pow", "power", "exponent"})

	// exp: float -> float
	registerTrigFunc("_math_exp", math.Exp,
		"Compute e^x (exponential function)",
		[]ParamDoc{{Name: "x", Description: "Exponent value"}},
		"e raised to the power x",
		[]Example{
			{Code: "exp(0.0)", Description: "Returns 1.0"},
			{Code: "exp(1.0)", Description: "Returns ~2.718 (e)"},
		},
		[]string{"math", "exp", "exponential", "e"})

	// log: float -> float
	registerTrigFunc("_math_log", math.Log,
		"Compute natural logarithm (base e)",
		[]ParamDoc{{Name: "x", Description: "Positive float value"}},
		"Natural logarithm of x",
		[]Example{
			{Code: "log(1.0)", Description: "Returns 0.0"},
			{Code: "log(E())", Description: "Returns 1.0"},
		},
		[]string{"math", "log", "logarithm", "ln", "natural"})

	// log10: float -> float
	registerTrigFunc("_math_log10", math.Log10,
		"Compute base-10 logarithm",
		[]ParamDoc{{Name: "x", Description: "Positive float value"}},
		"Base-10 logarithm of x",
		[]Example{
			{Code: "log10(10.0)", Description: "Returns 1.0"},
			{Code: "log10(100.0)", Description: "Returns 2.0"},
		},
		[]string{"math", "log10", "logarithm"})

	// floor: float -> float
	registerTrigFunc("_math_floor", math.Floor,
		"Round down to nearest integer",
		[]ParamDoc{{Name: "x", Description: "Float value"}},
		"Largest integer <= x",
		[]Example{
			{Code: "floor(3.7)", Description: "Returns 3.0"},
			{Code: "floor(-2.3)", Description: "Returns -3.0"},
		},
		[]string{"math", "floor", "round", "truncate"})

	// ceil: float -> float
	registerTrigFunc("_math_ceil", math.Ceil,
		"Round up to nearest integer",
		[]ParamDoc{{Name: "x", Description: "Float value"}},
		"Smallest integer >= x",
		[]Example{
			{Code: "ceil(3.2)", Description: "Returns 4.0"},
			{Code: "ceil(-2.7)", Description: "Returns -2.0"},
		},
		[]string{"math", "ceil", "ceiling", "round"})

	// round: float -> float
	registerTrigFunc("_math_round", math.Round,
		"Round to nearest integer (half away from zero)",
		[]ParamDoc{{Name: "x", Description: "Float value"}},
		"Nearest integer to x",
		[]Example{
			{Code: "round(3.5)", Description: "Returns 4.0"},
			{Code: "round(-2.5)", Description: "Returns -3.0"},
		},
		[]string{"math", "round", "nearest"})

	// abs_Float: float -> float
	registerTrigFunc("_math_abs_Float", math.Abs,
		"Compute absolute value of a float",
		[]ParamDoc{{Name: "x", Description: "Float value"}},
		"Absolute value of x",
		[]Example{
			{Code: "abs_Float(-3.14)", Description: "Returns 3.14"},
			{Code: "abs_Float(2.0)", Description: "Returns 2.0"},
		},
		[]string{"math", "abs", "absolute", "float"})

	// abs_Int: int -> int
	absIntImpl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		v := a.Value
		if v < 0 {
			v = -v
		}
		return &eval.IntValue{Value: v}, nil
	}
	absIntType := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int()).Returns(T.Int()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    "_math_abs_Int",
		NumArgs: 1,
		IsPure:  true,
		Type:    absIntType,
		Impl:    absIntImpl,
		Metadata: &BuiltinMetadata{
			Description: "Compute absolute value of an integer",
			Params:      []ParamDoc{{Name: "x", Description: "Integer value"}},
			Returns:     "Absolute value of x",
			Examples: []Example{
				{Code: "abs_Int(-42)", Description: "Returns 42"},
				{Code: "abs_Int(5)", Description: "Returns 5"},
			},
			Since:     "v0.5.10",
			Stability: StabilityStable,
			Tags:      []string{"math", "abs", "absolute", "int"},
			Category:  "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _math_abs_Int: %v", err))
	}

	// PI: (()) -> float (mathematical constant)
	// Note: Takes unit argument as workaround for nullary function issue (M-DX10)
	piImpl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		// Ignore unit argument
		return &eval.FloatValue{Value: math.Pi}, nil
	}
	piType := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Unit()).Returns(T.Float()).Build()
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    "_math_PI",
		NumArgs: 1,
		IsPure:  true,
		Type:    piType,
		Impl:    piImpl,
		Metadata: &BuiltinMetadata{
			Description: "Return the mathematical constant PI (3.14159...)",
			Returns:     "The value of PI",
			Examples: []Example{
				{Code: "PI()", Description: "Returns 3.141592653589793"},
			},
			Since:     "v0.5.10",
			Stability: StabilityStable,
			Tags:      []string{"math", "constant", "pi"},
			Category:  "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _math_PI: %v", err))
	}

	// E: (()) -> float (Euler's number)
	// Note: Takes unit argument as workaround for nullary function issue (M-DX10)
	eImpl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		// Ignore unit argument
		return &eval.FloatValue{Value: math.E}, nil
	}
	eType := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Unit()).Returns(T.Float()).Build()
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    "_math_E",
		NumArgs: 1,
		IsPure:  true,
		Type:    eType,
		Impl:    eImpl,
		Metadata: &BuiltinMetadata{
			Description: "Return Euler's number e (2.71828...)",
			Returns:     "The value of e",
			Examples: []Example{
				{Code: "E()", Description: "Returns 2.718281828459045"},
			},
			Since:     "v0.5.10",
			Stability: StabilityStable,
			Tags:      []string{"math", "constant", "e", "euler"},
			Category:  "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _math_E: %v", err))
	}
}

// registerTrigFunc registers a unary float->float math function
func registerTrigFunc(name string, fn func(float64) float64, desc string, params []ParamDoc, returns string, examples []Example, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		return &eval.FloatValue{Value: fn(a.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float()).Returns(T.Float()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    name,
		NumArgs: 1,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: desc,
			Params:      params,
			Returns:     returns,
			Examples:    examples,
			Since:       "v0.5.10",
			Stability:   StabilityStable,
			Tags:        tags,
			Category:    "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerTrigFunc2 registers a binary (float,float)->float math function
func registerTrigFunc2(name string, fn func(float64, float64) float64, desc string, params []ParamDoc, returns string, examples []Example, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.FloatValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float(), T.Float()).Returns(T.Float()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: desc,
			Params:      params,
			Returns:     returns,
			Examples:    examples,
			Since:       "v0.5.10",
			Stability:   StabilityStable,
			Tags:        tags,
			Category:    "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}
