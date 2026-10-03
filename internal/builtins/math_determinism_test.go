package builtins

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/mathx"
)

// #1465: the interpreter's std/math transcendentals must return the portable
// mathx bits — never the host math package's, which differ between arm64 and
// amd64. Run with -count=20: pure builtins must be repeatable too.

// determinismInputs is a fixed, arch-independent spread of float64 inputs:
// integer LCG state mapped through exact operations only.
func determinismInputs(n int) []float64 {
	xs := []float64{0, math.Copysign(0, -1), 0.5, 1, 2, 0.2064590551107192, 1.229317398921793, 1e-300, 1e300, 1 << 29, 1e22}
	s := uint64(0x2545F4914F6CDD1D)
	for i := 0; i < n; i++ {
		s = s*6364136223846793005 + 1442695040888963407
		u := float64(float64(s>>11) / (1 << 53)) // [0,1), exact
		switch i % 4 {
		case 0:
			xs = append(xs, float64(u*20)-10)
		case 1:
			xs = append(xs, float64(u*2)-1)
		case 2:
			xs = append(xs, float64(u*1400)-700)
		default:
			xs = append(xs, math.Float64frombits(s&^(1<<62))) // finite, any exponent
		}
	}
	return xs
}

func sameBits(a, b float64) bool {
	return math.Float64bits(a) == math.Float64bits(b) || (math.IsNaN(a) && math.IsNaN(b))
}

func TestStdMathUsesPortableBits(t *testing.T) {
	unary := map[string]func(float64) float64{
		"_math_exp": mathx.Exp, "_math_log": mathx.Log, "_math_log10": mathx.Log10,
		"_math_sin": mathx.Sin, "_math_cos": mathx.Cos, "_math_tan": mathx.Tan,
		"_math_asin": mathx.Asin, "_math_acos": mathx.Acos, "_math_atan": mathx.Atan,
	}
	in := determinismInputs(4000)
	for name, want := range unary {
		spec, ok := GetSpec(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		bad := 0
		for _, x := range in {
			res, err := spec.Impl(nil, []eval.Value{&eval.FloatValue{Value: x}})
			if err != nil {
				t.Fatalf("%s(%v): %v", name, x, err)
			}
			if got := res.(*eval.FloatValue).Value; !sameBits(got, want(x)) {
				if bad < 3 {
					t.Errorf("%s(%v) = %v (%#x), want portable %v (%#x)", name, x, got, math.Float64bits(got), want(x), math.Float64bits(want(x)))
				}
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("%s: %d/%d inputs differ from mathx", name, bad, len(in))
		}
	}
	binary := map[string]func(float64, float64) float64{"_math_atan2": mathx.Atan2, "_math_pow": mathx.Pow}
	for name, want := range binary {
		spec, ok := GetSpec(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		bad := 0
		for i := 0; i+1 < len(in); i++ {
			x, y := in[i], in[i+1]
			if name == "_math_pow" {
				x = math.Abs(x)
				y = math.Mod(y, 40)
			}
			res, err := spec.Impl(nil, []eval.Value{&eval.FloatValue{Value: x}, &eval.FloatValue{Value: y}})
			if err != nil {
				t.Fatalf("%s(%v, %v): %v", name, x, y, err)
			}
			if got := res.(*eval.FloatValue).Value; !sameBits(got, want(x, y)) {
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("%s: %d inputs differ from mathx", name, bad)
		}
	}
}

// The report's exact case (#1465): one value on every architecture.
func TestStdMathExpReportCase(t *testing.T) {
	spec, _ := GetSpec("_math_exp")
	res, err := spec.Impl(nil, []eval.Value{&eval.FloatValue{Value: 0.2064590551107192}})
	if err != nil {
		t.Fatal(err)
	}
	if got := math.Float64bits(res.(*eval.FloatValue).Value); got != 0x3ff3ab48b88c5dbe {
		t.Fatalf("exp(0.2064590551107192) bits = %#016x, want 0x3ff3ab48b88c5dbe (1.229317398921793)", got)
	}
}

// Fused multiply-add in the float kernels changed results on arm64 only.
// Each case is built so the separately-rounded answer is exactly 0 while a
// fused x*y+z would leave the tiny residual -2^-60.
func TestFloatKernelsDoNotFuse(t *testing.T) {
	const e = 1.0 / (1 << 30)
	list := func(xs ...float64) eval.Value { return floatsValue(xs) }
	arr := func(xs ...float64) eval.Value { return eval.NewFloatArray(xs) }
	call := func(name string, args ...eval.Value) eval.Value {
		t.Helper()
		spec, ok := GetSpec(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		v, err := spec.Impl(nil, args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return v
	}
	if got := call("_vec_dot", list(1, 1+e), list(-1, 1-e)).(*eval.FloatValue).Value; got != 0 {
		t.Errorf("_vec_dot fused: got %g, want 0", got)
	}
	if got := call("_array_f_dot", arr(1, 1+e), arr(-1, 1-e)).(*eval.FloatValue).Value; got != 0 {
		t.Errorf("_array_f_dot fused: got %g, want 0", got)
	}
	ys, err := asFloats("t", call("_vec_axpy", &eval.FloatValue{Value: 1 + e}, list(1-e), list(-1)), 0)
	if err != nil || ys[0] != 0 {
		t.Errorf("_vec_axpy fused: got %v (%v), want [0]", ys, err)
	}
	as, err := arrayFloats("t", call("_array_f_axpy", &eval.FloatValue{Value: 1 + e}, arr(1-e), arr(-1)), 0)
	if err != nil || as[0] != 0 {
		t.Errorf("_array_f_axpy fused: got %v (%v), want [0]", as, err)
	}
}
