package repl

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// The REPL's Eq[Float] is law-compliant: NaN equals NaN (reflexivity), and
// otherwise it is IEEE equality. Pinned so the NaN test can be spelled with
// math.IsNaN instead of the self-comparison a != a.
func TestREPLEqFloatNaNReflexive(t *testing.T) {
	r := New()
	r.initBuiltins()
	methods := r.instances["Eq[Float]"].Methods
	call := func(name string, a, b float64) bool {
		t.Helper()
		fn := methods[name].(*eval.BuiltinFunction)
		v, err := fn.Fn([]eval.Value{&eval.FloatValue{Value: a}, &eval.FloatValue{Value: b}})
		if err != nil {
			t.Fatalf("%s(%v, %v): %v", name, a, b, err)
		}
		return v.(*eval.BoolValue).Value
	}
	nan := math.NaN()
	for _, tc := range []struct {
		a, b    float64
		eq, neq bool
	}{
		{nan, nan, true, false},
		{nan, 1, false, true},
		{1, nan, false, true},
		{1.5, 1.5, true, false},
		{1.5, 2, false, true},
	} {
		if got := call("eq", tc.a, tc.b); got != tc.eq {
			t.Errorf("eq(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.eq)
		}
		if got := call("neq", tc.a, tc.b); got != tc.neq {
			t.Errorf("neq(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.neq)
		}
	}
}
