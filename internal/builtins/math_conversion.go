package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Conversion builtins (intToFloat, floatToInt)

func registerConversions() {
	// intToFloat
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a, ok := args[0].(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("intToFloat: expected IntValue for arg 0, got %T", args[0])
		}
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
		a, ok := args[0].(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("floatToInt: expected FloatValue for arg 0, got %T", args[0])
		}
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
