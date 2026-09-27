package builtins

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// The kernels sum in index order, so a non-associative input (values chosen so
// that a different order changes the float result) gives one fixed answer.
// Run with -count=20: pure builtins must be deterministic.
func TestArrayFloatKernelsDeterministic(t *testing.T) {
	xs := make([]float64, 10000)
	for i := range xs {
		xs[i] = math.Pow(-1, float64(i)) * (1e16 / float64(i+1))
	}
	a := eval.NewFloatArray(xs)
	want := 0.0
	for _, x := range xs {
		want += x
	}
	sum := lookupImpl(t, "_array_f_sum")
	for i := 0; i < 20; i++ {
		got, err := sum(nil, []eval.Value{a})
		if err != nil {
			t.Fatal(err)
		}
		if got.(*eval.FloatValue).Value != want {
			t.Fatalf("run %d: sum = %v, want %v (index order)", i, got, want)
		}
	}
}

// A boxed float array (the empty array, or one built by an untyped path) is
// read element by element and gives the same answer as the packed store.
func TestArrayFloatKernelsReadBoxedStore(t *testing.T) {
	dot := lookupImpl(t, "_array_f_dot")
	packed := eval.NewFloatArray([]float64{1, 2, 3})
	boxed := eval.NewArray([]eval.Value{&eval.FloatValue{Value: 1}, &eval.FloatValue{Value: 2}, &eval.IntValue{Value: 3}})
	if _, err := dot(nil, []eval.Value{packed, boxed}); err == nil {
		t.Fatal("a non-float element must be an error, not a silent coercion")
	}
	empty := eval.NewArray([]eval.Value{})
	got, err := dot(nil, []eval.Value{empty, eval.NewFloatArray(nil)})
	if err != nil || got.(*eval.FloatValue).Value != 0 {
		t.Fatalf("dot of two empty arrays: %v, %v", got, err)
	}
}

func lookupImpl(t *testing.T, name string) func(ctx any, args []eval.Value) (eval.Value, error) {
	t.Helper()
	spec, ok := GetSpec(name)
	if !ok {
		t.Fatalf("builtin %s not registered", name)
	}
	return func(_ any, args []eval.Value) (eval.Value, error) { return spec.Impl(nil, args) }
}
