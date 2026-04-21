package builtins

import (
	"fmt"
	"math"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Trigonometry and advanced math builtins (v0.5.9)

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
		a, ok := args[0].(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("abs_Int: expected IntValue for arg 0, got %T", args[0])
		}
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
		a, ok := args[0].(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected FloatValue for arg 0, got %T", name, args[0])
		}
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
		a, ok := args[0].(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected FloatValue for arg 0, got %T", name, args[0])
		}
		b, ok := args[1].(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected FloatValue for arg 1, got %T", name, args[1])
		}
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
