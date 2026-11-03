package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// #2062 (M-SMT-FLOAT-DIV-MOD-SORT-DISPATCH): float `/` was encoded as truncating
// Int division, float `%` as Int `mod`, and string ordering as Int `<`. The
// reporter's true contracts came back VIOLATION (bogus counterexample r = 7/4)
// or ERROR (Z3 sort mismatch), which tripped PUB006 and blocked publishing.

const floatDivSrc = `module m

export pure func tidalRadialOrbit(r: float) -> float
  requires { r > 1.5 }
  ensures { result > 0.0 }
{
  (1.0 / (2.0 * r * r * r)) * (2.0 + 3.0 / (2.0 * r - 3.0))
}

export pure func tidalTransverse(r: float) -> float
  requires { r > 0.0 }
  ensures { result > 0.0 }
{
  1.0 / (2.0 * r * r * r)
}

export pure func fmodPos(x: float) -> float
  requires { x > 0.1 }
  ensures { result >= 0.0 }
{
  7.5 % x
}

export pure func fmodNegDividend(x: float) -> float
  requires { x > 0.1 }
  ensures { result <= 0.0 }
{
  (0.0 - 7.5) % x
}

export pure func strLt(a: string, b: string) -> bool
  requires { a < b }
  ensures { result }
{
  a < b
}

export pure func strGe(a: string, b: string) -> bool
  ensures { result == (b <= a) }
{
  a >= b
}

export pure func intDiv(x: int) -> int
  requires { x != 0 }
  ensures { result == 7 / x }
{
  7 / x
}

export pure func halfIsSmaller(r: float) -> float
  requires { r > 2.0 }
  ensures { result > 1.0 }
{
  1.0 / r
}

export pure func fmodWrongSign(x: float) -> float
  requires { x > 0.1 }
  ensures { result > 0.0 }
{
  (0.0 - 7.5) % x
}

export pure func strLtWrong(a: string, b: string) -> bool
  ensures { result }
{
  a < b
}
`

func TestVerify_FloatDivModAndStringOrderingBySort(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	got := verifyStatuses(t, bin, writeTempAil(t, floatDivSrc))
	want := map[string]string{
		// The reporter's repro and its siblings: true contracts verify.
		"tidalRadialOrbit": "verified",
		"tidalTransverse":  "verified",
		"fmodPos":          "verified",
		"fmodNegDividend":  "verified",
		"strLt":            "verified",
		"strGe":            "verified",
		"intDiv":           "verified",
		// False contracts are still refuted: the Real encoding is not vacuous.
		"halfIsSmaller": "counterexample",
		"fmodWrongSign": "counterexample",
		"strLtWrong":    "counterexample",
	}
	for fn, status := range want {
		r, ok := got[fn]
		if !ok {
			t.Fatalf("%s missing from verify results: %+v", fn, got)
		}
		if r.Status != status {
			t.Errorf("%s: status = %q (%s), want %q", fn, r.Status, r.Reason, status)
		}
	}
}
