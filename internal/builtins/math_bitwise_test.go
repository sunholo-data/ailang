package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestBitwiseAnd(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	tests := []struct {
		a, b     int
		expected int
	}{
		{12, 10, 8},    // 1100 & 1010 = 1000
		{0, 0, 0},      // 0 & 0 = 0
		{-1, 255, 255}, // all-ones & 0xFF = 0xFF
		{7, 3, 3},      // 0111 & 0011 = 0011
		{255, 0, 0},    // any & 0 = 0
	}

	impl := intIntToInt(func(a, b int) int { return a & b })
	for _, tt := range tests {
		result, err := impl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(tt.a),
			testctx.MakeInt(tt.b),
		})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, testctx.GetInt(result), "bitwiseAnd(%d, %d)", tt.a, tt.b)
	}
}

func TestBitwiseXor(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	tests := []struct {
		a, b     int
		expected int
	}{
		{12, 10, 6},   // 1100 ^ 1010 = 0110
		{0, 0, 0},     // 0 ^ 0 = 0
		{255, 255, 0}, // x ^ x = 0
		{0, 255, 255}, // 0 ^ x = x
		{-1, 0, -1},   // -1 ^ 0 = -1
	}

	impl := intIntToInt(func(a, b int) int { return a ^ b })
	for _, tt := range tests {
		result, err := impl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(tt.a),
			testctx.MakeInt(tt.b),
		})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, testctx.GetInt(result), "bitwiseXor(%d, %d)", tt.a, tt.b)
	}
}

func TestBitwiseOr(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	tests := []struct {
		a, b     int
		expected int
	}{
		{12, 10, 14},  // 1100 | 1010 = 1110
		{0, 0, 0},     // 0 | 0 = 0
		{8, 7, 15},    // 1000 | 0111 = 1111
		{255, 0, 255}, // x | 0 = x
	}

	impl := intIntToInt(func(a, b int) int { return a | b })
	for _, tt := range tests {
		result, err := impl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(tt.a),
			testctx.MakeInt(tt.b),
		})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, testctx.GetInt(result), "bitwiseOr(%d, %d)", tt.a, tt.b)
	}
}

func TestBitwiseNot(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	tests := []struct {
		a        int
		expected int
	}{
		{0, -1},     // ~0 = -1 (all bits set)
		{-1, 0},     // ~(-1) = 0
		{1, -2},     // ~1 = -2
		{255, -256}, // ~0xFF = -256
	}

	impl := intToInt(func(a int) int { return ^a })
	for _, tt := range tests {
		result, err := impl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(tt.a),
		})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, testctx.GetInt(result), "bitwiseNot(%d)", tt.a)
	}
}

func TestShiftLeft(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	tests := []struct {
		a, n     int
		expected int
	}{
		{1, 0, 1},                     // 1 << 0 = 1
		{1, 4, 16},                    // 1 << 4 = 16
		{3, 2, 12},                    // 11 << 2 = 1100 = 12
		{1, 63, -9223372036854775808}, // 1 << 63 = min int64 (sign bit)
	}

	for _, tt := range tests {
		result, err := shiftLeftImpl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(tt.a),
			testctx.MakeInt(tt.n),
		})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, testctx.GetInt(result), "shiftLeft(%d, %d)", tt.a, tt.n)
	}
}

func TestShiftRight(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	tests := []struct {
		a, n     int
		expected int
	}{
		{16, 2, 4},   // 16 >> 2 = 4
		{255, 4, 15}, // 11111111 >> 4 = 00001111
		{-1, 1, -1},  // arithmetic right shift preserves sign
		{8, 0, 8},    // x >> 0 = x
	}

	for _, tt := range tests {
		result, err := shiftRightImpl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(tt.a),
			testctx.MakeInt(tt.n),
		})
		require.NoError(t, err)
		assert.Equal(t, tt.expected, testctx.GetInt(result), "shiftRight(%d, %d)", tt.a, tt.n)
	}
}

func TestShiftNegativeAmount(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	_, err := shiftLeftImpl(ctx.EffContext, []eval.Value{
		testctx.MakeInt(1),
		testctx.MakeInt(-1),
	})
	assert.Error(t, err, "shiftLeft with negative amount should error")

	_, err = shiftRightImpl(ctx.EffContext, []eval.Value{
		testctx.MakeInt(1),
		testctx.MakeInt(-1),
	})
	assert.Error(t, err, "shiftRight with negative amount should error")
}

func TestBitwiseRegistration(t *testing.T) {
	bitwiseNames := []string{
		"bitwiseAnd_Int", "bitwiseXor_Int", "bitwiseOr_Int",
		"bitwiseNot_Int", "shiftLeft_Int", "shiftRight_Int",
	}

	for _, name := range bitwiseNames {
		spec, ok := GetSpec(name)
		if !ok {
			t.Errorf("builtin %s not registered", name)
			continue
		}
		assert.True(t, spec.IsPure, "%s should be pure", name)
		if name == "bitwiseNot_Int" {
			assert.Equal(t, 1, spec.NumArgs, "%s should have 1 arg", name)
		} else {
			assert.Equal(t, 2, spec.NumArgs, "%s should have 2 args", name)
		}
	}
}

func TestBitwiseDeterminism(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	for i := 0; i < 20; i++ {
		result, err := shiftLeftImpl(ctx.EffContext, []eval.Value{
			testctx.MakeInt(1099511628211),
			testctx.MakeInt(17),
		})
		require.NoError(t, err)
		assert.Equal(t, 1099511628211<<17, testctx.GetInt(result))
	}
}
