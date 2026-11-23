package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// Two verifier leaks from the stapledons Kerr/celestial contracts (2026-11-02).
// verifyStatuses fails the test on any "unknown constant" in the output.

// Float literals reached Z3 in Go exponent notation ("1e+09" → unknown
// constant "e+09"), and whole numbers as Int numerals ("1" → sort mismatch
// in a Real define-const). Each contract holds only because of the value.
func TestVerify_ScientificFloatLiteralsEncode(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m
export pure func scale(x: float) -> float ! {} requires { x >= 0.0 } ensures { result >= 0.0 } { x * 1000000000.0 }
export pure func tiny() -> float ! {} ensures { result > 0.0 && result < 0.000000001 } { 2.5e-11 }
export pure func neg() -> float ! {} ensures { result < -100000000.0 } { -1.5e8 }
export pure func one() -> float ! {} ensures { result > 0.5 } { 1.0 }
`)
	got := verifyStatuses(t, bin, f)
	for _, fn := range []string{"scale", "tiny", "neg", "one"} {
		if r := got[fn]; r.Status != "verified" {
			t.Fatalf("%s status = %q (reason %q), want verified", fn, r.Status, r.Reason)
		}
	}
}

// A local helper calling std/math.sqrt was inlined as a define-fun with a raw
// `(sqrt s)` body, so Z3 hard-errored (stapledons, 2026-11-02); it was then
// skipped. std/math.sqrt is now an uninterpreted function with IEEE-true facts
// (sqrt maps [0,1] into [0,1]), so horizon is proved — and nothing raw reaches
// Z3 (verifyStatuses fails on any "unknown constant").
func TestVerify_HelperCallingSqrtVerifies(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m
import std/math (sqrt)
pure func root(s: float) -> float = sqrt((1.0-s)*(1.0+s))
pure func dbl(s: float) -> float = s * 2.0
export pure func horizon(s: float) -> float requires { s >= 0.0 && s < 1.0 } ensures { result >= 0.5 && result <= 1.0 } { (1.0+root(s))/2.0 }
export pure func twice(s: float) -> float requires { s >= 0.0 } ensures { result >= 0.0 } { dbl(s) }
`)
	got := verifyStatuses(t, bin, f)
	if r := got["horizon"]; r.Status != "verified" {
		t.Fatalf("horizon status = %q (reason %q), want verified", r.Status, r.Reason)
	}
	if r := got["twice"]; r.Status != "verified" {
		t.Fatalf("twice status = %q (reason %q), want verified", r.Status, r.Reason)
	}
}
