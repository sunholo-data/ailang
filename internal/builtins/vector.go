package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Native [float] vector kernels behind std/embedding (M-NUMERICS-QUICK).
//
// std/embedding's dot/add/subtract were hand-rolled AILANG recursion, one
// evaluator call per element, and add/subtract built their result with
// `x + y :: helper(...)`, which is quadratic on flat-array lists. A 768-dim dot
// measured ~7 us per multiply-add, slower than foldl/zipWith. These loops keep
// the exact semantics the AILANG bodies had, including how mismatched lengths
// are handled (see each function), so callers see only the speed change.
//
// axpy is new and strict: its lengths must match, because a silently
// truncated gradient update is a bug, not a convenience.

func init() {
	registerVecDot()
	registerVecAdd()
	registerVecSub()
	registerVecScale()
	registerVecAxpy()
}

func floatListType() types.Type {
	T := types.NewBuilder()
	return T.List(T.Float())
}

func asFloats(name string, v eval.Value, pos int) ([]float64, error) {
	l, ok := v.(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("%s: arg %d must be a list of float, got %T", name, pos, v)
	}
	out := make([]float64, len(l.Elements))
	for i, e := range l.Elements {
		f, ok := e.(*eval.FloatValue)
		if !ok {
			return nil, fmt.Errorf("%s: arg %d element %d must be float, got %T", name, pos, i, e)
		}
		out[i] = f.Value
	}
	return out, nil
}

func floatsValue(xs []float64) eval.Value {
	elems := make([]eval.Value, len(xs))
	for i, x := range xs {
		elems[i] = &eval.FloatValue{Value: x}
	}
	return &eval.ListValue{Elements: elems}
}

func registerVec(name, desc string, nargs int, typ func() types.Type, impl func(*effects.EffContext, []eval.Value) (eval.Value, error)) {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/embedding",
		Name:    name,
		NumArgs: nargs,
		IsPure:  true,
		Effect:  "",
		Type:    typ,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: desc,
			Since:       "v0.44.2",
			Stability:   StabilityStable,
			Tags:        []string{"vector", "float", "numeric", "embedding"},
			Category:    "embedding",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// _vec_dot: sum of a[i]*b[i] over the common prefix (the AILANG body stopped
// at the shorter list).
func registerVecDot() {
	registerVec("_vec_dot", "Dot product of two float vectors over their common prefix", 2,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(floatListType(), floatListType()).Returns(T.Float()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, err := asFloats("_vec_dot", args[0], 0)
			if err != nil {
				return nil, err
			}
			b, err := asFloats("_vec_dot", args[1], 1)
			if err != nil {
				return nil, err
			}
			n := min(len(a), len(b))
			acc := 0.0
			for i := 0; i < n; i++ {
				// float64(...) forbids FMA fusion so arm64 and amd64 agree bit-for-bit (#1465).
				acc += float64(a[i] * b[i])
			}
			return &eval.FloatValue{Value: acc}, nil
		})
}

// _vec_add: elementwise sum; the longer vector's tail is kept (AILANG body:
// `[] => b` and `[] => a`).
func registerVecAdd() {
	registerVec("_vec_add", "Elementwise sum of two float vectors; the longer one's tail is kept", 2,
		func() types.Type {
			return types.NewBuilder().Func(floatListType(), floatListType()).Returns(floatListType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, err := asFloats("_vec_add", args[0], 0)
			if err != nil {
				return nil, err
			}
			b, err := asFloats("_vec_add", args[1], 1)
			if err != nil {
				return nil, err
			}
			if len(a) < len(b) {
				a, b = b, a
			}
			out := make([]float64, len(a))
			copy(out, a)
			for i := range b {
				out[i] += b[i]
			}
			return floatsValue(out), nil
		})
}

// _vec_sub: elementwise a - b; a's tail is kept when b is shorter, and b's
// tail is dropped when a is shorter (AILANG body: `[] => []` on a, `[] => a` on b).
func registerVecSub() {
	registerVec("_vec_sub", "Elementwise difference a - b; a's tail is kept, b's is dropped", 2,
		func() types.Type {
			return types.NewBuilder().Func(floatListType(), floatListType()).Returns(floatListType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, err := asFloats("_vec_sub", args[0], 0)
			if err != nil {
				return nil, err
			}
			b, err := asFloats("_vec_sub", args[1], 1)
			if err != nil {
				return nil, err
			}
			out := make([]float64, len(a))
			copy(out, a)
			for i := 0; i < min(len(a), len(b)); i++ {
				out[i] -= b[i]
			}
			return floatsValue(out), nil
		})
}

// _vec_scale: s * v elementwise.
func registerVecScale() {
	registerVec("_vec_scale", "Multiply every element of a float vector by a scalar", 2,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Float(), floatListType()).Returns(floatListType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			s, ok := args[0].(*eval.FloatValue)
			if !ok {
				return nil, fmt.Errorf("_vec_scale: arg 0 must be float, got %T", args[0])
			}
			v, err := asFloats("_vec_scale", args[1], 1)
			if err != nil {
				return nil, err
			}
			for i := range v {
				v[i] *= s.Value
			}
			return floatsValue(v), nil
		})
}

// _vec_axpy: a*x + y, the SGD/gradient update in one pass. Lengths must match.
func registerVecAxpy() {
	registerVec("_vec_axpy", "a*x + y elementwise in one pass; x and y must have equal length", 3,
		func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.Float(), floatListType(), floatListType()).Returns(floatListType()).Build()
		},
		func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
			a, ok := args[0].(*eval.FloatValue)
			if !ok {
				return nil, fmt.Errorf("_vec_axpy: arg 0 must be float, got %T", args[0])
			}
			x, err := asFloats("_vec_axpy", args[1], 1)
			if err != nil {
				return nil, err
			}
			y, err := asFloats("_vec_axpy", args[2], 2)
			if err != nil {
				return nil, err
			}
			if len(x) != len(y) {
				return nil, fmt.Errorf("axpy: vector lengths differ (x has %d, y has %d)", len(x), len(y))
			}
			for i := range y {
				// float64(...) forbids FMA fusion so arm64 and amd64 agree bit-for-bit (#1465).
				y[i] += float64(a.Value * x[i])
			}
			return floatsValue(y), nil
		})
}
