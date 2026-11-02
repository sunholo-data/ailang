package smt

import (
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #1481: 0x8000000000000000 now parses to math.MinInt64. Negating it in Go
// wraps back to MinInt64, which produced the malformed "(- -9223372036854775808)".
func TestEncodeLitNegativeInts(t *testing.T) {
	cases := map[int64]string{
		0:                    "0",
		42:                   "42",
		-1:                   "(- 1)",
		-7046029254386353131: "(- 7046029254386353131)",
		math.MaxInt64:        "9223372036854775807",
		math.MinInt64:        "(- 9223372036854775808)",
	}
	for v, want := range cases {
		got, err := encodeLit(&core.Lit{Kind: core.IntLit, Value: v})
		if err != nil {
			t.Fatalf("encodeLit(%d): %v", v, err)
		}
		if got != want {
			t.Errorf("encodeLit(%d) = %q, want %q", v, got, want)
		}
	}
}

// SMT-LIB has no exponent syntax: %g wrote 1e9 as "1e+09", which Z3 read as
// the unknown constant "e+09" (stapledons report, 2026-11-02). A whole number
// needs ".0" or it is an Int numeral (sort mismatch in a Real define-const).
func TestEncodeLitFloatsArePositional(t *testing.T) {
	cases := map[float64]string{
		0.5:      "0.5",
		1.0:      "1.0",
		1e9:      "1000000000.0",
		-1e8:     "(- 100000000.0)",
		1e-12:    "0.000000000001",
		-2.5e-11: "(- 0.000000000025)",
	}
	for v, want := range cases {
		got, err := encodeLit(&core.Lit{Kind: core.FloatLit, Value: v})
		if err != nil {
			t.Fatalf("encodeLit(%g): %v", v, err)
		}
		if got != want {
			t.Errorf("encodeLit(%g) = %q, want %q", v, got, want)
		}
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got, err := encodeLit(&core.Lit{Kind: core.FloatLit, Value: v}); err == nil {
			t.Errorf("encodeLit(%v) = %q, want an error", v, got)
		}
	}
}
