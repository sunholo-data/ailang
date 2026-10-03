package mathx

import (
	"math"
	"runtime"
	"testing"
)

// TestHostCensus reports (never fails) how often the host math package
// disagrees with mathx over the deterministic sweep corpus. It is the
// cross-arch audit instrument for #1465: on GOARCH=wasm (no assembly, no
// fusion) the count must be zero; on arm64 it measures the FMA divergence;
// on amd64 it measures the assembly Exp/Log divergence.
func TestHostCensus(t *testing.T) {
	type pair struct {
		name string
		mx   func(float64) float64
		host func(float64) float64
	}
	unary := []pair{
		{"exp", Exp, math.Exp}, {"log", Log, math.Log}, {"log10", Log10, math.Log10},
		{"sin", Sin, math.Sin}, {"cos", Cos, math.Cos}, {"tan", Tan, math.Tan},
		{"asin", Asin, math.Asin}, {"acos", Acos, math.Acos}, {"atan", Atan, math.Atan},
	}
	in := sweepInputs()
	for _, p := range unary {
		diff := 0
		for _, x := range in[p.name] {
			if math.Float64bits(p.mx(x)) != math.Float64bits(p.host(x)) && !(math.IsNaN(p.mx(x)) && math.IsNaN(p.host(x))) {
				diff++
			}
		}
		t.Logf("%s/%s %-5s host-vs-mathx differ: %d / %d", runtime.GOOS, runtime.GOARCH, p.name, diff, len(in[p.name]))
	}
	binary := []struct {
		name string
		mx   func(float64, float64) float64
		host func(float64, float64) float64
	}{{"atan2", Atan2, math.Atan2}, {"pow", Pow, math.Pow}}
	for _, p := range binary {
		diff := 0
		args := sweepPairs(p.name)
		for _, a := range args {
			m, h := p.mx(a[0], a[1]), p.host(a[0], a[1])
			if math.Float64bits(m) != math.Float64bits(h) && !(math.IsNaN(m) && math.IsNaN(h)) {
				diff++
			}
		}
		t.Logf("%s/%s %-5s host-vs-mathx differ: %d / %d", runtime.GOOS, runtime.GOARCH, p.name, diff, len(args))
	}
}
