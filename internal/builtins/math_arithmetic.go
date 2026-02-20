package builtins

import (
	"fmt"
	"math"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// Arithmetic builtins for Int and Float operations

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
	registerBuiltinWithMeta("double_Int", 1, true, intToInt(func(a int) int { return a * 2 }),
		"Double an integer", []string{"math", "arithmetic", "double", "test"})

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
	a, ok := args[0].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("div_Float: expected FloatValue for arg 0, got %T", args[0])
	}
	b, ok := args[1].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("div_Float: expected FloatValue for arg 1, got %T", args[1])
	}
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
	a, ok := args[0].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("mod_Float: expected FloatValue for arg 0, got %T", args[0])
	}
	b, ok := args[1].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("mod_Float: expected FloatValue for arg 1, got %T", args[1])
	}
	if b.Value == 0.0 {
		return &eval.FloatValue{Value: math.NaN()}, nil
	}
	return &eval.FloatValue{Value: math.Mod(a.Value, b.Value)}, nil
}

// floatNegFloat: negation
func floatNegFloat(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a, ok := args[0].(*eval.FloatValue)
	if !ok {
		return nil, fmt.Errorf("neg_Float: expected FloatValue for arg 0, got %T", args[0])
	}
	return &eval.FloatValue{Value: -a.Value}, nil
}
