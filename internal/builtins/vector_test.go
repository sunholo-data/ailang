package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func fl(xs ...float64) *eval.ListValue { return floatsValue(xs).(*eval.ListValue) }

func toFloats(t *testing.T, v eval.Value) []float64 {
	t.Helper()
	out, err := asFloats("test", v, 0)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func callVec(t *testing.T, name string, args ...eval.Value) (eval.Value, error) {
	t.Helper()
	spec, ok := specRegistry[name]
	if !ok {
		t.Fatalf("%s not registered", name)
	}
	return spec.Impl(nil, args)
}

func TestVecKernelsKeepTheAILANGBodiesSemantics(t *testing.T) {
	v, _ := callVec(t, "_vec_dot", fl(1, 2, 3), fl(4, 5))
	if v.(*eval.FloatValue).Value != 14 {
		t.Fatalf("dot over common prefix = %v, want 14", v)
	}
	v, _ = callVec(t, "_vec_add", fl(1), fl(10, 20))
	if g := toFloats(t, v); len(g) != 2 || g[0] != 11 || g[1] != 20 {
		t.Fatalf("add keeps the longer tail: got %v", g)
	}
	v, _ = callVec(t, "_vec_sub", fl(3, 5), fl(1))
	if g := toFloats(t, v); len(g) != 2 || g[0] != 2 || g[1] != 5 {
		t.Fatalf("sub keeps a's tail: got %v", g)
	}
	v, _ = callVec(t, "_vec_sub", fl(3), fl(1, 9))
	if g := toFloats(t, v); len(g) != 1 || g[0] != 2 {
		t.Fatalf("sub drops b's tail: got %v", g)
	}
}

func TestVecKernelsDoNotMutateInputs(t *testing.T) {
	x, y := fl(1, 2), fl(10, 20)
	if _, err := callVec(t, "_vec_axpy", &eval.FloatValue{Value: 2}, x, y); err != nil {
		t.Fatal(err)
	}
	if _, err := callVec(t, "_vec_scale", &eval.FloatValue{Value: 3}, x); err != nil {
		t.Fatal(err)
	}
	if g := toFloats(t, y); g[0] != 10 || g[1] != 20 {
		t.Fatalf("axpy mutated y: %v", g)
	}
	if g := toFloats(t, x); g[0] != 1 || g[1] != 2 {
		t.Fatalf("scale mutated x: %v", g)
	}
}

func TestVecKernelsRejectBadInput(t *testing.T) {
	if _, err := callVec(t, "_vec_axpy", &eval.FloatValue{Value: 1}, fl(1, 2), fl(1)); err == nil || !strings.Contains(err.Error(), "lengths differ") {
		t.Fatalf("axpy length mismatch: %v", err)
	}
	mixed := &eval.ListValue{Elements: []eval.Value{&eval.FloatValue{Value: 1}, &eval.IntValue{Value: 2}}}
	if _, err := callVec(t, "_vec_dot", mixed, fl(1, 2)); err == nil || !strings.Contains(err.Error(), "element 1 must be float") {
		t.Fatalf("non-float element: %v", err)
	}
}

func TestListRangeBounds(t *testing.T) {
	v, _ := listRangeImpl(nil, []eval.Value{&eval.IntValue{Value: -2}, &eval.IntValue{Value: 2}})
	if g := toInts(t, v); !equalInts(g, []int{-2, -1, 0, 1}) {
		t.Fatalf("range(-2, 2) = %v", g)
	}
	if _, err := listRangeImpl(nil, []eval.Value{&eval.IntValue{Value: 0}, &eval.IntValue{Value: maxRangeLen + 1}}); err == nil {
		t.Fatal("oversized range not rejected")
	}
}
