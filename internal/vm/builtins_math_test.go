package vm

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/mathx"
)

// #1465: the strict VM's std/math transcendentals must return the portable
// mathx bits, identical to the interpreter and to every other architecture.

func vmMathInputs(n int) []float64 {
	xs := []float64{0, math.Copysign(0, -1), 0.5, 1, 2, 0.2064590551107192, 1.229317398921793, 1e-300, 1e300, 1 << 29, 1e22}
	s := uint64(0x9E3779B97F4A7C15)
	for i := 0; i < n; i++ {
		s = s*6364136223846793005 + 1442695040888963407
		u := float64(float64(s>>11) / (1 << 53))
		switch i % 4 {
		case 0:
			xs = append(xs, float64(u*20)-10)
		case 1:
			xs = append(xs, float64(u*2)-1)
		case 2:
			xs = append(xs, float64(u*1400)-700)
		default:
			xs = append(xs, math.Float64frombits(s&^(1<<62)))
		}
	}
	return xs
}

func TestVMStdMathUsesPortableBits(t *testing.T) {
	unary := []struct {
		name string
		vm   func([]bytecode.Value) (bytecode.Value, error)
		want func(float64) float64
	}{
		{"exp", builtinMathExp, mathx.Exp}, {"log", builtinMathLog, mathx.Log},
		{"log10", builtinMathLog10, mathx.Log10}, {"sin", builtinMathSin, mathx.Sin},
		{"cos", builtinMathCos, mathx.Cos}, {"tan", builtinMathTan, mathx.Tan},
		{"asin", builtinMathAsin, mathx.Asin}, {"acos", builtinMathAcos, mathx.Acos},
		{"atan", builtinMathAtan, mathx.Atan},
	}
	in := vmMathInputs(4000)
	for _, f := range unary {
		bad := 0
		for _, x := range in {
			v, err := f.vm([]bytecode.Value{bytecode.NewFloat(x)})
			if err != nil {
				t.Fatalf("%s(%v): %v", f.name, x, err)
			}
			w := f.want(x)
			if math.Float64bits(v.Flt) != math.Float64bits(w) && !(math.IsNaN(v.Flt) && math.IsNaN(w)) {
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("VM %s: %d/%d inputs differ from mathx", f.name, bad, len(in))
		}
	}
	binary := []struct {
		name string
		vm   func([]bytecode.Value) (bytecode.Value, error)
		want func(float64, float64) float64
	}{{"atan2", builtinMathAtan2, mathx.Atan2}, {"pow", builtinMathPow, mathx.Pow}}
	for _, f := range binary {
		bad := 0
		for i := 0; i+1 < len(in); i++ {
			x, y := in[i], in[i+1]
			if f.name == "pow" {
				x, y = math.Abs(x), math.Mod(y, 40)
			}
			v, err := f.vm([]bytecode.Value{bytecode.NewFloat(x), bytecode.NewFloat(y)})
			if err != nil {
				t.Fatalf("%s: %v", f.name, err)
			}
			w := f.want(x, y)
			if math.Float64bits(v.Flt) != math.Float64bits(w) && !(math.IsNaN(v.Flt) && math.IsNaN(w)) {
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("VM %s: %d inputs differ from mathx", f.name, bad)
		}
	}
}

func TestVMStdMathExpReportCase(t *testing.T) {
	v, err := builtinMathExp([]bytecode.Value{bytecode.NewFloat(0.2064590551107192)})
	if err != nil {
		t.Fatal(err)
	}
	if got := math.Float64bits(v.Flt); got != 0x3ff3ab48b88c5dbe {
		t.Fatalf("VM exp(0.2064590551107192) bits = %#016x, want 0x3ff3ab48b88c5dbe", got)
	}
}
