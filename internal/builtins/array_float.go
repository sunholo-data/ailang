package builtins

import (
	"fmt"
	"math"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Float kernels and bulk updates on Array (M-NUMERICS-VEC-ARRAY M3 + M5).
//
// The kernels are typed on Array[float] and run on the packed []float64 store
// directly; a boxed float array (only an empty one, in a typed program) is read
// element by element. Unlike the std/embedding [float] versions, which keep the
// lenient common-prefix semantics of their old AILANG bodies, every two-vector
// kernel here is strict: a length mismatch is an error, because a silently
// truncated vector op is a bug. Summation is a sequential loop in index order,
// so results are bit-reproducible (A1).
//
// updateMany/scatterAdd are the bulk-update model (D3 option i): one copy of
// the array, then k writes, so k updates cost O(n + k) instead of O(n*k).

func init() {
	registerArrayFloatReduce("_array_f_dot", "Dot product of two float arrays of equal length", 2)
	registerArrayFloatReduce("_array_f_sum", "Sum of a float array, in index order", 1)
	registerArrayFloatArgmax()
	registerArrayFloatBinary("_array_f_add", "Elementwise a + b of two float arrays of equal length", func(x, y float64) float64 { return x + y })
	registerArrayFloatBinary("_array_f_sub", "Elementwise a - b of two float arrays of equal length", func(x, y float64) float64 { return x - y })
	registerArrayFloatBinary("_array_f_mul", "Elementwise a * b of two float arrays of equal length", func(x, y float64) float64 { return x * y })
	registerArrayFloatScale()
	registerArrayFloatAxpy()
	registerArrayUpdateMany()
	registerArrayScatterAdd()
}

func floatArrayType() types.Type {
	return &types.TArray{Element: types.NewBuilder().Float()}
}

// arrayFloats returns the float contents of an Array[float]. A packed array's
// slice is shared and must not be modified.
func arrayFloats(name string, v eval.Value, pos int) ([]float64, error) {
	arr, ok := v.(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("%s: arg %d must be an Array[float], got %T", name, pos, v)
	}
	if xs, packed := arr.Floats(); packed {
		return xs, nil
	}
	out := make([]float64, arr.Len())
	for i := range out {
		e, _ := arr.Get(int64(i))
		f, ok := e.(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("%s: arg %d element %d must be float, got %T", name, pos, i, e)
		}
		out[i] = f.Value
	}
	return out, nil
}

func sameLength(name string, a, b []float64) error {
	if len(a) != len(b) {
		return fmt.Errorf("%s: array lengths differ (%d and %d)", name, len(a), len(b))
	}
	return nil
}

func registerArrayBuiltin(name, desc string, nargs int, typ func() types.Type, impl func(*effects.EffContext, []eval.Value) (eval.Value, error)) {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    name,
		NumArgs: nargs,
		IsPure:  true,
		Effect:  "",
		Type:    typ,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: desc,
			Since:       "v0.46.0",
			Stability:   StabilityExperimental,
			Tags:        []string{"array", "float", "numeric"},
			Category:    "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// publicName strips the builtin prefix for error messages ("_array_f_dot" -> "dot").
func publicName(name string) string {
	for _, p := range []string{"_array_f_", "_array_"} {
		if len(name) > len(p) && name[:len(p)] == p {
			return name[len(p):]
		}
	}
	return name
}

// _array_f_dot (2 args) and _array_f_sum (1 arg): float reductions.
func registerArrayFloatReduce(name, desc string, nargs int) {
	pub := publicName(name)
	registerArrayBuiltin(name, desc, nargs,
		func() types.Type {
			T := types.NewBuilder()
			params := make([]types.Type, nargs)
			for i := range params {
				params[i] = floatArrayType()
			}
			return T.Func(params...).Returns(T.Float()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, err := arrayFloats(pub, args[0], 0)
			if err != nil {
				return nil, err
			}
			acc := 0.0
			if nargs == 1 {
				for _, x := range a {
					acc += x
				}
				return &eval.FloatValue{Value: acc}, nil
			}
			b, err := arrayFloats(pub, args[1], 1)
			if err != nil {
				return nil, err
			}
			if err := sameLength(pub, a, b); err != nil {
				return nil, err
			}
			for i := range a {
				acc += a[i] * b[i]
			}
			return &eval.FloatValue{Value: acc}, nil
		})
}

// _array_f_argmax: index of the first largest element. NaN never compares
// greater, so NaNs are skipped; an all-NaN array returns 0. Empty is an error.
func registerArrayFloatArgmax() {
	registerArrayBuiltin("_array_f_argmax", "Index of the first largest element of a non-empty float array (NaNs are skipped)", 1,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(floatArrayType()).Returns(T.Int()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, err := arrayFloats("argmax", args[0], 0)
			if err != nil {
				return nil, err
			}
			if len(a) == 0 {
				return nil, fmt.Errorf("argmax: empty array has no largest element")
			}
			best := 0
			for i, x := range a {
				if math.IsNaN(a[best]) && !math.IsNaN(x) || x > a[best] {
					best = i
				}
			}
			return &eval.IntValue{Value: best}, nil
		})
}

// _array_f_add/sub/mul: elementwise, equal lengths.
func registerArrayFloatBinary(name, desc string, op func(x, y float64) float64) {
	pub := publicName(name)
	registerArrayBuiltin(name, desc, 2,
		func() types.Type {
			return types.NewBuilder().Func(floatArrayType(), floatArrayType()).Returns(floatArrayType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, err := arrayFloats(pub, args[0], 0)
			if err != nil {
				return nil, err
			}
			b, err := arrayFloats(pub, args[1], 1)
			if err != nil {
				return nil, err
			}
			if err := sameLength(pub, a, b); err != nil {
				return nil, err
			}
			out := make([]float64, len(a))
			for i := range a {
				out[i] = op(a[i], b[i])
			}
			return eval.NewFloatArray(out), nil
		})
}

// _array_f_scale: s * v.
func registerArrayFloatScale() {
	registerArrayBuiltin("_array_f_scale", "Multiply every element of a float array by a scalar", 2,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Float(), floatArrayType()).Returns(floatArrayType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			s, ok := args[0].(*eval.FloatValue)
			if !ok {
				return nil, fmt.Errorf("scale: arg 0 must be float, got %T", args[0])
			}
			v, err := arrayFloats("scale", args[1], 1)
			if err != nil {
				return nil, err
			}
			out := make([]float64, len(v))
			for i, x := range v {
				out[i] = s.Value * x
			}
			return eval.NewFloatArray(out), nil
		})
}

// _array_f_axpy: a*x + y in one pass, equal lengths.
func registerArrayFloatAxpy() {
	registerArrayBuiltin("_array_f_axpy", "a*x + y elementwise in one pass; x and y must have equal length", 3,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Float(), floatArrayType(), floatArrayType()).Returns(floatArrayType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, ok := args[0].(*eval.FloatValue)
			if !ok {
				return nil, fmt.Errorf("axpy: arg 0 must be float, got %T", args[0])
			}
			x, err := arrayFloats("axpy", args[1], 1)
			if err != nil {
				return nil, err
			}
			y, err := arrayFloats("axpy", args[2], 2)
			if err != nil {
				return nil, err
			}
			if err := sameLength("axpy", x, y); err != nil {
				return nil, err
			}
			out := make([]float64, len(y))
			for i := range y {
				out[i] = y[i] + a.Value*x[i]
			}
			return eval.NewFloatArray(out), nil
		})
}

// _array_update_many: one copy, then each (index, value) write in list order,
// so a later write to the same index wins. Any index out of bounds is an error.
func registerArrayUpdateMany() {
	registerArrayBuiltin("_array_update_many", "Apply a list of (index, value) writes with one copy; later writes win; out-of-bounds is an error", 2,
		func() types.Type {
			T := types.NewBuilder()
			a := T.Var("a")
			arr := &types.TArray{Element: a}
			pair := &types.TTuple{Elements: []types.Type{T.Int(), a}}
			return T.Func(arr, T.List(pair)).Returns(arr).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			arr, ok := args[0].(*eval.ArrayValue)
			if !ok {
				return nil, fmt.Errorf("updateMany: arg 0 must be an Array, got %T", args[0])
			}
			writes, ok := args[1].(*eval.ListValue)
			if !ok {
				return nil, fmt.Errorf("updateMany: arg 1 must be a list of (int, value), got %T", args[1])
			}
			n := arr.Len()
			idxOf := func(w eval.Value) (int, eval.Value, error) {
				t, ok := w.(*eval.TupleValue)
				if !ok || len(t.Elements) != 2 {
					return 0, nil, fmt.Errorf("updateMany: each write must be an (int, value) pair, got %T", w)
				}
				iv, ok := t.Elements[0].(*eval.IntValue)
				if !ok {
					return 0, nil, fmt.Errorf("updateMany: index must be int, got %T", t.Elements[0])
				}
				if iv.Value < 0 || iv.Value >= n {
					return 0, nil, fmt.Errorf("updateMany: index %d out of bounds (array length: %d)", iv.Value, n)
				}
				return iv.Value, t.Elements[1], nil
			}
			if xs, packed := arr.Floats(); packed {
				out := append([]float64(nil), xs...)
				allFloat := true
				for _, w := range writes.Elements {
					i, v, err := idxOf(w)
					if err != nil {
						return nil, err
					}
					f, ok := v.(*eval.FloatValue)
					if !ok {
						allFloat = false
						break
					}
					out[i] = f.Value
				}
				if allFloat {
					return eval.NewFloatArray(out), nil
				}
			}
			out := append([]eval.Value(nil), arr.Elements()...)
			for _, w := range writes.Elements {
				i, v, err := idxOf(w)
				if err != nil {
					return nil, err
				}
				out[i] = v
			}
			return eval.NewArray(out), nil
		})
}

// _array_scatter_add: out[idx[j]] += xs[j] for every j, the gradient-accumulate
// shape. idx and xs must have equal length; an index out of bounds is an error.
func registerArrayScatterAdd() {
	registerArrayBuiltin("_array_scatter_add", "Add xs[j] into position idx[j] for every j, with one copy", 3,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(floatArrayType(), T.List(T.Int()), floatListType()).Returns(floatArrayType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			base, err := arrayFloats("scatterAdd", args[0], 0)
			if err != nil {
				return nil, err
			}
			idxList, ok := args[1].(*eval.ListValue)
			if !ok {
				return nil, fmt.Errorf("scatterAdd: arg 1 must be a list of int, got %T", args[1])
			}
			xs, err := asFloats("scatterAdd", args[2], 2)
			if err != nil {
				return nil, err
			}
			if len(idxList.Elements) != len(xs) {
				return nil, fmt.Errorf("scatterAdd: %d indices but %d values", len(idxList.Elements), len(xs))
			}
			out := append([]float64(nil), base...)
			for j, e := range idxList.Elements {
				iv, ok := e.(*eval.IntValue)
				if !ok {
					return nil, fmt.Errorf("scatterAdd: index %d must be int, got %T", j, e)
				}
				if iv.Value < 0 || iv.Value >= len(out) {
					return nil, fmt.Errorf("scatterAdd: index %d out of bounds (array length: %d)", iv.Value, len(out))
				}
				out[iv.Value] += xs[j]
			}
			return eval.NewFloatArray(out), nil
		})
}
