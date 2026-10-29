package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// _array_accumulate: (Array[a], [(int, a)], (a, a) -> a) -> Array[a].
// One copy of arr, then for each (i, x) in list order out[i] = combine(out[i], x).
// It is the generic batch combine (M-PERSISTENT-COLLECTIONS 1c): difference
// arrays, histograms and counting sorts in one O(n + k) pure call. An index
// out of bounds is an error naming the index and the length. scatterAdd stays
// as the packed-float fast path (no callback per write).
func init() {
	registerArrayBuiltin("_array_accumulate", "Combine a list of (index, value) writes into an array with one copy: out[i] = combine(out[i], x), in list order; out-of-bounds is an error", 3,
		func() types.Type {
			T := types.NewBuilder()
			a := T.Var("a")
			arr := &types.TArray{Element: a}
			pair := &types.TTuple{Elements: []types.Type{T.Int(), a}}
			combine := T.Func(a, a).Returns(a).Build()
			return T.Func(arr, T.List(pair), combine).Returns(arr).Build()
		},
		arrayAccumulateImpl)
}

func arrayAccumulateImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("accumulate: arg 0 must be an Array, got %T", args[0])
	}
	writes, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("accumulate: arg 1 must be a list of (int, value), got %T", args[1])
	}
	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("accumulate: FnCallerN not set (evaluator not wired)")
	}
	combine := args[2]
	out := append([]eval.Value(nil), arr.Elements()...)
	for j, w := range writes.Elements {
		t, ok := w.(*eval.TupleValue)
		if !ok || len(t.Elements) != 2 {
			return nil, fmt.Errorf("accumulate: write %d must be an (int, value) pair, got %T", j, w)
		}
		iv, ok := t.Elements[0].(*eval.IntValue)
		if !ok {
			return nil, fmt.Errorf("accumulate: write %d: index must be int, got %T", j, t.Elements[0])
		}
		i := iv.Value
		if i < 0 || i >= len(out) {
			return nil, fmt.Errorf("accumulate: index %d out of bounds (array length: %d)", i, len(out))
		}
		v, err := ctx.FnCallerN(combine, []eval.Value{out[i], t.Elements[1]})
		if err != nil {
			return nil, callbackErr(err, "accumulate: combine failed at write %d", j)
		}
		out[i] = v
	}
	return eval.NewArray(out), nil
}
