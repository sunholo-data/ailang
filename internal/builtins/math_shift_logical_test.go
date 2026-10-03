package builtins

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

// #1481: logical (zero-filling) right shift — the `>>>` of reference hash and
// PRNG code. `>>` stays arithmetic. Known answers are m-pure-prng §6's.
func TestShiftRightLogical(t *testing.T) {
	spec, ok := GetSpec("shiftRightLogical_Int")
	if !ok {
		t.Fatal("shiftRightLogical_Int not registered")
	}
	if !spec.IsPure || spec.NumArgs != 2 {
		t.Fatalf("shiftRightLogical_Int: pure=%v arity=%d, want pure arity 2", spec.IsPure, spec.NumArgs)
	}
	ctx := testctx.NewMockEffContext()
	cases := []struct{ a, n, want int }{
		{-1, 1, math.MaxInt64},
		{-1, 63, 1},
		{math.MinInt64, 63, 1},
		{math.MinInt64, 1, 4611686018427387904},
		{-7046029254386353131, 30, 10617743077}, // SplitMix64 mixer step
		{16, 2, 4},
		{255, 4, 15},
		{8, 0, 8},
		{-1, 0, -1},
		{-1, 64, 0}, // count >= 64 shifts every bit out (uint64 semantics)
		{12345, 200, 0},
	}
	for i := 0; i < 20; i++ { // pure builtin: repeatable
		for _, c := range cases {
			res, err := spec.Impl(ctx.EffContext, []eval.Value{&eval.IntValue{Value: c.a}, &eval.IntValue{Value: c.n}})
			if err != nil {
				t.Fatalf("shiftRightLogical(%d, %d): %v", c.a, c.n, err)
			}
			if got := res.(*eval.IntValue).Value; got != c.want {
				t.Fatalf("shiftRightLogical(%d, %d) = %d, want %d", c.a, c.n, got, c.want)
			}
		}
	}
	if _, err := spec.Impl(ctx.EffContext, []eval.Value{&eval.IntValue{Value: 1}, &eval.IntValue{Value: -1}}); err == nil {
		t.Fatal("negative shift amount must be an RT_SHIFT error, like >>")
	}
}
