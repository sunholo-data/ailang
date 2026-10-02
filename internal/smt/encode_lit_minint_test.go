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
