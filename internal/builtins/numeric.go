// Package builtins provides numeric conversion builtins for AILANG.
// These support JSON encoding/decoding which requires int<->float conversions.
package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	registerIntToFloat()
	registerFloatToInt()
}

// registerIntToFloat registers _int_to_float: int -> float
// Converts an integer to a floating point number.
// Used by JSON encoding to produce JNumber values.
func registerIntToFloat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "core",
		Name:    "_int_to_float",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeIntToFloatType,
		Impl:    intToFloatImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _int_to_float: %v", err))
	}
}

func makeIntToFloatType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Int()).Returns(T.Float()).Build()
}

func intToFloatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	intVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("intToFloat: expected int, got %s", ailangTypeName(args[0], "int"))
	}
	return &eval.FloatValue{Value: float64(intVal.Value)}, nil
}

// registerFloatToInt registers _float_to_int: float -> int
// Converts a floating point number to an integer by truncation.
// Used by JSON decoding to extract integer values from JNumber.
func registerFloatToInt() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "core",
		Name:    "_float_to_int",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeFloatToIntType,
		Impl:    floatToIntImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _float_to_int: %v", err))
	}
}

func makeFloatToIntType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Float()).Returns(T.Int()).Build()
}

func floatToIntImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	floatVal, ok := args[0].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("floatToInt: expected float, got %s", ailangTypeName(args[0], "float"))
	}
	return &eval.IntValue{Value: int(floatVal.Value)}, nil
}
