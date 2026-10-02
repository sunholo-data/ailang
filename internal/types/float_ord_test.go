package types

import (
	"math"
	"testing"
)

// M-FLOAT-ORD-ONE-SEMANTICS (#1419): IEEE ordered comparisons.

func TestFloatOrdIEEE(t *testing.T) {
	nan := math.NaN()
	ops := map[string]func(a, b float64) bool{
		"lt": FloatLt, "lte": FloatLte, "gt": FloatGt, "gte": FloatGte,
	}
	for name, op := range ops {
		for _, pair := range [][2]float64{{nan, 1}, {1, nan}, {nan, nan}, {nan, math.Inf(1)}, {math.Inf(-1), nan}} {
			if op(pair[0], pair[1]) {
				t.Errorf("%s(%v, %v) = true, want false (NaN is unordered)", name, pair[0], pair[1])
			}
		}
	}
	negZero := math.Copysign(0, -1)
	cases := []struct {
		name string
		got  bool
		want bool
	}{
		{"1<2", FloatLt(1, 2), true},
		{"2<1", FloatLt(2, 1), false},
		{"2<=2", FloatLte(2, 2), true},
		{"2>1", FloatGt(2, 1), true},
		{"1>=2", FloatGte(1, 2), false},
		{"-Inf<+Inf", FloatLt(math.Inf(-1), math.Inf(1)), true},
		{"0<-0", FloatLt(0, negZero), false},
		{"-0<=0", FloatLte(negZero, 0), true},
		{"0>=-0", FloatGte(0, negZero), true},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestFloatMinMaxPropagateNaN(t *testing.T) {
	nan := math.NaN()
	for _, v := range []float64{FloatMin(nan, 1), FloatMin(1, nan), FloatMax(nan, 1), FloatMax(1, nan)} {
		if !math.IsNaN(v) {
			t.Errorf("min/max with a NaN operand = %v, want NaN in both argument orders", v)
		}
	}
	if FloatMin(1, 2) != 1 || FloatMax(1, 2) != 2 {
		t.Errorf("ordinary min/max wrong")
	}
}

// The Ord[Float] dictionary is what monomorphic comparisons on the evaluator
// resolve to; it must answer IEEE (the mutation anchor for the parity test).
func TestOrdFloatDictionaryIsIEEE(t *testing.T) {
	r := NewDictionaryRegistry()
	nan := math.NaN()
	for _, m := range []string{"lt", "lte", "gt", "gte"} {
		impl, ok := r.LookupMethod("prelude", "Ord", &TCon{Name: "float"}, m)
		if !ok {
			t.Fatalf("missing Ord[float].%s", m)
		}
		f, ok := impl.(func(float64, float64) bool)
		if !ok {
			t.Fatalf("Ord[float].%s: unexpected impl type %T", m, impl)
		}
		if f(nan, 1) || f(1, nan) {
			t.Errorf("Ord[float].%s answers true with a NaN operand", m)
		}
	}
	for _, m := range []string{"min", "max"} {
		impl, ok := r.LookupMethod("prelude", "Ord", &TCon{Name: "float"}, m)
		if !ok {
			t.Fatalf("missing Ord[float].%s", m)
		}
		f := impl.(func(float64, float64) float64)
		if !math.IsNaN(f(nan, 1)) || !math.IsNaN(f(1, nan)) {
			t.Errorf("Ord[float].%s does not propagate NaN", m)
		}
	}
}
