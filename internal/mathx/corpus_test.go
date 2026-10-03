package mathx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
)

// The sweep corpus is generated, not stored: a fixed splitmix64 stream turns
// into the same float64 inputs on every architecture (integer arithmetic and
// exact int→float conversions only), so the corpus itself cannot drift.
// golden_test.go pins a SHA-256 of every output's bits per function.

const sweepN = 20000

type splitmix struct{ s uint64 }

func (r *splitmix) next() uint64 {
	r.s += 0x9e3779b97f4a7c15
	z := r.s
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// unit returns a float64 in [0, 1) built from 53 random bits (exact).
func (r *splitmix) unit() float64 { return float64(float64(r.next()>>11) / (1 << 53)) }

// span returns a value in [lo, hi). lo and hi are chosen so the product and
// sum below are exact-or-irrelevant: the inputs only need to be reproducible,
// and every operation here is a single IEEE op with no fusable pair because
// of the explicit conversion.
func (r *splitmix) span(lo, hi float64) float64 { return lo + float64(r.unit()*(hi-lo)) }

// anyBits returns a float64 with random sign, exponent and mantissa, masked
// to finite values.
func (r *splitmix) anyFinite() float64 {
	for {
		f := math.Float64frombits(r.next())
		if !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f
		}
	}
}

// specials are the boundary values every function is fed.
var specials = []float64{
	0, math.Copysign(0, -1), 1, -1, 0.5, -0.5, 2, -2, 0.7, -0.7, 0.66, 0.6600000000000001,
	math.Pi, -math.Pi, math.Pi / 2, -math.Pi / 2, math.Pi / 4, 3 * math.Pi / 4,
	math.E, math.Ln2, 1e-300, -1e-300, 1e300, -1e300,
	math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, 2.2250738585072014e-308,
	math.MaxFloat64, -math.MaxFloat64, 1.0 / (1 << 28), -1.0 / (1 << 28),
	709.782712893384, 709.78, 7.09782712893383973096e+02, -745.1332191019411, -745.13,
	-7.45133219101941108420e+02, 1 << 29, (1 << 29) - 1, 1<<29 + 1, 1e22, 1 << 60,
	2.41421356237309504880, 1 - 1e-16, 1 + 2.220446049250313e-16,
	0.2064590551107192, 1.229317398921793, 1.2293173989217931,
	math.Inf(1), math.Inf(-1), math.NaN(),
}

func sweepInputs() map[string][]float64 {
	out := map[string][]float64{}
	gen := func(name string, seed uint64, draw func(r *splitmix) float64) {
		r := &splitmix{s: seed}
		xs := append([]float64(nil), specials...)
		for i := 0; i < sweepN; i++ {
			xs = append(xs, draw(r))
		}
		out[name] = xs
	}
	gen("exp", 1, func(r *splitmix) float64 {
		switch r.next() % 3 {
		case 0:
			return r.span(-746, 710)
		case 1:
			return r.span(-2, 2)
		default:
			return r.span(-1e-6, 1e-6)
		}
	})
	logDraw := func(r *splitmix) float64 {
		switch r.next() % 3 {
		case 0:
			return math.Abs(r.anyFinite())
		case 1:
			return r.span(0, 10)
		default:
			return r.span(0.5, 2)
		}
	}
	gen("log", 2, logDraw)
	gen("log10", 3, logDraw)
	trigDraw := func(r *splitmix) float64 {
		switch r.next() % 4 {
		case 0:
			return r.span(-7, 7)
		case 1:
			return r.span(-1e4, 1e4)
		case 2:
			return r.span(5e8, 6e8) // straddles reduceThreshold (1<<29)
		default:
			return r.anyFinite() // Payne-Hanek territory
		}
	}
	gen("sin", 4, trigDraw)
	gen("cos", 5, trigDraw)
	gen("tan", 6, trigDraw)
	gen("asin", 7, func(r *splitmix) float64 { return r.span(-1, 1) })
	gen("acos", 8, func(r *splitmix) float64 { return r.span(-1, 1) })
	gen("atan", 9, func(r *splitmix) float64 {
		if r.next()%2 == 0 {
			return r.span(-5, 5)
		}
		return r.anyFinite()
	})
	return out
}

func sweepPairs(name string) [][2]float64 {
	var out [][2]float64
	for _, a := range specials {
		for _, b := range []float64{0, 1, -1, 0.5, -0.5, 2, 3, -3, 2.5, math.Inf(1), math.Inf(-1), math.NaN()} {
			out = append(out, [2]float64{a, b}, [2]float64{b, a})
		}
	}
	switch name {
	case "atan2":
		r := &splitmix{s: 10}
		for i := 0; i < sweepN; i++ {
			if r.next()%2 == 0 {
				out = append(out, [2]float64{r.span(-10, 10), r.span(-10, 10)})
			} else {
				out = append(out, [2]float64{r.anyFinite(), r.anyFinite()})
			}
		}
	case "pow":
		r := &splitmix{s: 11}
		for i := 0; i < sweepN; i++ {
			switch r.next() % 4 {
			case 0:
				out = append(out, [2]float64{r.span(0, 10), r.span(-20, 20)})
			case 1:
				out = append(out, [2]float64{r.span(-10, 10), float64(int64(r.next()%61) - 30)})
			case 2:
				out = append(out, [2]float64{r.span(0.9, 1.1), r.span(-2000, 2000)})
			default:
				out = append(out, [2]float64{math.Abs(r.anyFinite()), r.span(-3, 3)})
			}
		}
	}
	return out
}

// digest hashes the bit patterns of a sequence of outputs. NaNs are folded
// to one canonical pattern: NaN payloads are not part of AILANG's contract
// (show prints "NaN" for all of them).
func digest(vals []float64) string {
	h := sha256.New()
	var b [8]byte
	for _, v := range vals {
		bits := math.Float64bits(v)
		if math.IsNaN(v) {
			bits = 0x7ff8000000000001
		}
		binary.LittleEndian.PutUint64(b[:], bits)
		h.Write(b[:])
	}
	return hex.EncodeToString(h.Sum(nil))
}
