package smt

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// M-SMT-INTERP-SHOW M2: `"${n}"` with n:int normalizes to
// $builtin._string_intToStr(n), which needs an SMT encoding.
//
// Z3's str.from_int is defined ONLY for non-negative arguments — it returns ""
// for a negative one (verified below in TestZ3StrFromIntIsUndefinedForNegatives).
// AILANG's show/intToStr render -5 as "-5" (strconv.Itoa), so the naive
// (str.from_int n) encoding would be SILENTLY WRONG for every negative input.
// The correct encoding branches on the sign.

// TestEncodeIntToStr_UsesSignBranch pins the emitted form. A regression to a
// bare (str.from_int n) would prove false things about negative inputs.
func TestEncodeIntToStr_UsesSignBranch(t *testing.T) {
	spec, ok := StringBuiltinSpecial["_string_intToStr"]
	if !ok {
		t.Fatal("_string_intToStr has no StringBuiltinSpecial entry")
	}
	arg := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "n"}
	got, err := encodeStringBuiltin(spec, []core.CoreExpr{arg})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	for _, want := range []string{"ite", ">=", "str.from_int", `"-"`, "str.++"} {
		if !strings.Contains(got, want) {
			t.Errorf("encoding %q is missing %q — a bare str.from_int is wrong for negatives", got, want)
		}
	}
}

// TestEncodeIntToStr_ViaStdlibWrapper: a user's own `import std/string (intToStr)`
// must encode too, not just the normalizer's $builtin reference.
func TestEncodeIntToStr_ViaStdlibWrapper(t *testing.T) {
	builtinName, mapped := ResolveStdlibToBuiltin("std/string", "intToStr")
	if !mapped {
		t.Fatal("std/string.intToStr has no stdlib->builtin SMT mapping")
	}
	if builtinName != "_string_intToStr" {
		t.Fatalf("std/string.intToStr maps to %q, want _string_intToStr", builtinName)
	}
}

// TestZ3StrFromIntIsUndefinedForNegatives records the external premise the
// encoding is built on. If a future solver changes this, the test tells us —
// the encoding stays correct either way, because the ite never passes a
// negative to str.from_int.
func TestZ3StrFromIntIsUndefinedForNegatives(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}
	smtlib := `(set-logic ALL)
(assert (not (= (str.from_int (- 5)) "")))
(check-sat)
`
	res, err := Solve(smtlib, DefaultSolverConfig())
	if err != nil {
		t.Fatalf("solve: %v", err)
	}
	if res.Status != StatusVerified {
		t.Errorf("str.from_int(-5) is no longer \"\" (status %v). The encoding is still "+
			"correct — the ite never passes a negative — but this premise has moved: %s",
			res.Status, res.RawOutput)
	}
}

// TestIntToStrEncodingIsExact is the real gate: the SMT encoding must agree with
// strconv.Itoa (what show and intToStr actually do at runtime) on every sampled
// input, and must never render an int as the empty string. Asserted as unsat on
// the NEGATION, so this proves the property rather than merely failing to find a
// counterexample.
func TestIntToStrEncodingIsExact(t *testing.T) {
	if !Z3Available() {
		t.Skip("Z3 not installed")
	}

	spec := StringBuiltinSpecial["_string_intToStr"]
	arg := &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "n"}
	body, err := encodeStringBuiltin(spec, []core.CoreExpr{arg})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	def := fmt.Sprintf("(define-fun showInt ((n Int)) String\n  %s)\n", body)

	samples := []int{0, 1, 7, 9, 10, 42, 99, 100, 12345, -1, -5, -9, -10, -42, -100, -12345}
	for _, n := range samples {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			lit := strconv.Itoa(n)
			smtN := lit
			if n < 0 {
				smtN = fmt.Sprintf("(- %d)", -n)
			}
			smtlib := fmt.Sprintf("(set-logic ALL)\n%s(assert (not (= (showInt %s) %q)))\n(check-sat)\n",
				def, smtN, lit)
			res, err := Solve(smtlib, DefaultSolverConfig())
			if err != nil {
				t.Fatalf("solve: %v", err)
			}
			if res.Status != StatusVerified {
				t.Errorf("showInt(%d) != %q per Z3 (status %v): %s", n, lit, res.Status, res.RawOutput)
			}
		})
	}

	// Symbolic property: no int renders as the empty string. A bare
	// str.from_int would fail this for every negative n.
	t.Run("never_empty_symbolic", func(t *testing.T) {
		smtlib := fmt.Sprintf("(set-logic ALL)\n%s(declare-const n Int)\n(assert (= (str.len (showInt n)) 0))\n(check-sat)\n", def)
		res, err := Solve(smtlib, DefaultSolverConfig())
		if err != nil {
			t.Fatalf("solve: %v", err)
		}
		if res.Status != StatusVerified {
			t.Errorf("some int renders as the empty string (status %v): %s", res.Status, res.RawOutput)
		}
	})
}
