package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// Bitwise builtins for integer-only operations.
// These are registered as the lowered form of bitwise operators (& ^ ~ << >>).

func registerBitwise() {
	registerBuiltinWithMeta("bitwiseAnd_Int", 2, true, intIntToInt(func(a, b int) int { return a & b }),
		"Bitwise AND of two integers", []string{"math", "bitwise", "and"})
	registerBuiltinWithMeta("bitwiseXor_Int", 2, true, intIntToInt(func(a, b int) int { return a ^ b }),
		"Bitwise XOR of two integers", []string{"math", "bitwise", "xor"})
	registerBuiltinWithMeta("bitwiseOr_Int", 2, true, intIntToInt(func(a, b int) int { return a | b }),
		"Bitwise OR of two integers", []string{"math", "bitwise", "or"})
	registerBuiltinWithMeta("bitwiseNot_Int", 1, true, intToInt(func(a int) int { return ^a }),
		"Bitwise NOT (complement) of an integer", []string{"math", "bitwise", "not", "complement"})
	registerBuiltinWithMeta("shiftLeft_Int", 2, true, shiftLeftImpl,
		"Left shift an integer by n bits", []string{"math", "bitwise", "shift", "left"})
	registerBuiltinWithMeta("shiftRight_Int", 2, true, shiftRightImpl,
		"Arithmetic right shift an integer by n bits", []string{"math", "bitwise", "shift", "right"})
}

func shiftLeftImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("shiftLeft_Int: expected IntValue for arg 0, got %T", args[0])
	}
	b, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("shiftLeft_Int: expected IntValue for arg 1, got %T", args[1])
	}
	if b.Value < 0 {
		return nil, eval.NewRuntimeError("RT_SHIFT", "negative shift amount", map[string]interface{}{
			"amount": b.Value,
		})
	}
	return &eval.IntValue{Value: a.Value << uint(b.Value)}, nil
}

func shiftRightImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("shiftRight_Int: expected IntValue for arg 0, got %T", args[0])
	}
	b, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("shiftRight_Int: expected IntValue for arg 1, got %T", args[1])
	}
	if b.Value < 0 {
		return nil, eval.NewRuntimeError("RT_SHIFT", "negative shift amount", map[string]interface{}{
			"amount": b.Value,
		})
	}
	return &eval.IntValue{Value: a.Value >> uint(b.Value)}, nil
}
