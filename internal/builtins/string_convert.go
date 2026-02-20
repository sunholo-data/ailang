package builtins

import (
	"fmt"
	"strconv"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// String conversion builtins (floatToStr, intToStr)
// These convert numeric types to string representations.

func init() {
	registerStringConversions()
}

func registerStringConversions() {
	// floatToStr: float -> string
	floatToStrImpl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		f, ok := args[0].(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("floatToStr: expected FloatValue for arg 0, got %T", args[0])
		}
		// Use %g for compact representation (no trailing zeros)
		s := strconv.FormatFloat(f.Value, 'g', -1, 64)
		return &eval.StringValue{Value: s}, nil
	}
	floatToStrType := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float()).Returns(T.String()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_string_floatToStr",
		NumArgs: 1,
		IsPure:  true,
		Type:    floatToStrType,
		Impl:    floatToStrImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert a float to its string representation",
			LongDesc:    "Converts a floating-point number to a string using compact representation (no trailing zeros). Uses Go's %g format which chooses the shortest representation.",
			Params: []ParamDoc{
				{Name: "f", Description: "Float value to convert"},
			},
			Returns: "String representation of the float",
			Examples: []Example{
				{Code: `floatToStr(3.14)`, Description: `Returns "3.14"`},
				{Code: `floatToStr(-0.5)`, Description: `Returns "-0.5"`},
				{Code: `floatToStr(42.0)`, Description: `Returns "42"`},
				{Code: `floatToStr(0.0001)`, Description: `Returns "0.0001"`},
			},
			Since:     "v0.5.10",
			Stability: StabilityStable,
			Tags:      []string{"string", "conversion", "float", "format"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _string_floatToStr: %v", err))
	}

	// intToStr: int -> string
	intToStrImpl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		i, ok := args[0].(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("intToStr: expected IntValue for arg 0, got %T", args[0])
		}
		s := strconv.Itoa(i.Value)
		return &eval.StringValue{Value: s}, nil
	}
	intToStrType := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int()).Returns(T.String()).Build()
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_string_intToStr",
		NumArgs: 1,
		IsPure:  true,
		Type:    intToStrType,
		Impl:    intToStrImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert an integer to its string representation",
			Params: []ParamDoc{
				{Name: "n", Description: "Integer value to convert"},
			},
			Returns: "String representation of the integer",
			Examples: []Example{
				{Code: `intToStr(42)`, Description: `Returns "42"`},
				{Code: `intToStr(-100)`, Description: `Returns "-100"`},
				{Code: `intToStr(0)`, Description: `Returns "0"`},
			},
			Since:     "v0.5.10",
			Stability: StabilityStable,
			Tags:      []string{"string", "conversion", "int", "format"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _string_intToStr: %v", err))
	}
}
