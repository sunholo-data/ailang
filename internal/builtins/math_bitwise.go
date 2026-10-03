package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
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
	// #1481: no operator; std/math exports it as shiftRightLogical.
	registerBuiltinWithMeta("shiftRightLogical_Int", 2, true, shiftRightLogicalImpl,
		"Logical (zero-filling) right shift of an integer's 64-bit pattern by n bits; the >>> of reference hash code",
		[]string{"math", "bitwise", "shift", "right", "logical", "unsigned"})
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

// shiftRightLogicalImpl shifts the 64-bit two's-complement pattern right,
// filling with zeros: int64(uint64(a) >> n). A count of 64 or more yields 0
// (Go's unsigned-shift semantics, as in every uint64 reference). A negative
// count is RT_SHIFT, exactly as for << and >>.
func shiftRightLogicalImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	a, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("shiftRightLogical_Int: expected IntValue for arg 0, got %T", args[0])
	}
	b, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("shiftRightLogical_Int: expected IntValue for arg 1, got %T", args[1])
	}
	if b.Value < 0 {
		return nil, eval.NewRuntimeError("RT_SHIFT", "negative shift amount", map[string]interface{}{
			"amount": b.Value,
		})
	}
	return &eval.IntValue{Value: int(uint64(a.Value) >> uint(b.Value))}, nil
}
