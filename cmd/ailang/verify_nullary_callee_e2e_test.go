package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// M-SMT-NULLARY-CALLEE end-to-end tests: `ailang verify` must encode a zero-arg pure
// function as an SMT constant instead of skipping every caller with
// `unencodable type "()"` (Daneel report, 2026-09-26).

type verifyResultRow struct {
	Function string `json:"function"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

func verifyStatuses(t *testing.T, bin string, args ...string) map[string]verifyResultRow {
	t.Helper()
	stdout, stderr, _ := runAilangBin(t, bin, append([]string{"verify", "--json"}, args...)...)
	for _, bad := range []string{"unknown constant", "unknown sort", "Sort mismatch", "unit literals cannot"} {
		if strings.Contains(stdout+stderr, bad) {
			t.Fatalf("verify leaked %q; output:\n%s\n%s", bad, stdout, stderr)
		}
	}
	var payload struct {
		Results []verifyResultRow `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("verify --json invalid: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	out := make(map[string]verifyResultRow, len(payload.Results))
	for _, r := range payload.Results {
		out[r.Function] = r
	}
	return out
}

func writeTempAil(t *testing.T, src string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "m.ail")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestVerify_NullaryConstantExampleVerifies: the exact repro, a chained constant and a
// constant named in an ensures clause all VERIFY (not skip).
func TestVerify_NullaryConstantExampleVerifies(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	got := verifyStatuses(t, bin, "examples/runnable/contracts/nullary_constant_verify.ail")
	for _, fn := range []string{"bound", "turnBound", "withChained"} {
		r, ok := got[fn]
		if !ok {
			t.Fatalf("%s missing from verify results: %+v", fn, got)
		}
		if r.Status != "verified" {
			t.Fatalf("%s status = %q (reason %q), want verified", fn, r.Status, r.Reason)
		}
	}
}

// TestVerify_NullaryConstantValueIsEncoded proves the constant is really encoded, not
// trivially dropped: each contract here is false ONLY because of the constant's value,
// so Z3 must report a counterexample.
func TestVerify_NullaryConstantValueIsEncoded(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m

export pure func cap() -> int { 1000 }
export pure func doubled() -> int { cap() * 2 }

export pure func tight(x: int) -> int
requires { x >= 0 }
ensures { result <= x + 999 } { x + cap() }

export pure func chainedTight(x: int) -> int
requires { x >= 0 }
ensures { result <= x + 1999 } { x + doubled() }

export pure func ensuresTight(x: int) -> int
requires { x >= 0 }
ensures { result > x + cap() } { x + 1000 }
`)
	got := verifyStatuses(t, bin, "--relax-modules", f)
	for _, fn := range []string{"tight", "chainedTight", "ensuresTight"} {
		if r := got[fn]; r.Status != "counterexample" {
			t.Fatalf("%s status = %q (reason %q), want counterexample", fn, r.Status, r.Reason)
		}
	}
}

// TestVerify_ContractPredicateCalleeIsEncoded: a user function called from an ensures
// predicate is resolved as a define-fun, so the predicate is actually checked — true
// verifies, false yields a counterexample.
func TestVerify_ContractPredicateCalleeIsEncoded(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m
type V = A | B
export func legal(v: V) -> bool ! {} { match v { A => true, B => true } }
export func strict(v: V) -> bool ! {} { match v { A => true, B => false } }
export func pick(n: int) -> V ! {}
requires { n >= 0 }
ensures { legal(result) }
{ if n > 5 then B else A }
export func pickStrict(n: int) -> V ! {}
requires { n >= 0 }
ensures { strict(result) }
{ if n > 5 then B else A }
`)
	got := verifyStatuses(t, bin, "--relax-modules", f)
	if r := got["pick"]; r.Status != "verified" {
		t.Fatalf("pick status = %q (reason %q), want verified", r.Status, r.Reason)
	}
	if r := got["pickStrict"]; r.Status != "counterexample" {
		t.Fatalf("pickStrict status = %q (reason %q), want counterexample", r.Status, r.Reason)
	}
}

// TestVerify_UnitReturningCalleeStillSkips: a callee returning `()` is genuinely
// unencodable; the fix must not let it through.
func TestVerify_UnitReturningCalleeStillSkips(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m
export pure func noop() -> () { () }
export pure func usesUnit(x: int) -> int
requires { x >= 0 }
ensures { result >= 0 } {
  let _u = noop();
  x
}
`)
	got := verifyStatuses(t, bin, "--relax-modules", f)
	r := got["usesUnit"]
	if r.Status != "skipped" || !strings.Contains(r.Reason, `"()"`) {
		t.Fatalf("usesUnit = %+v, want skipped naming \"()\"", r)
	}
}

// TestVerify_ContractCalleeSortGate: callees reached from a contract predicate are now
// resolved as define-funs, so the callee-sort gate must cover contracts too — an
// Option[float] callee in an ensures clause skips with UNENCODABLE_TYPE rather than
// leaking an undeclared sort into Z3.
func TestVerify_ContractCalleeSortGate(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m
import std/option (Option, Some, None)
export func conv(x: float) -> Option[float] ! {} { Some(x * 2.0) }
export func ok(v: Option[float]) -> bool ! {} { match v { Some(y) => y >= 0.0, None => true } }
export func grade(x: float) -> float ! {}
requires { x >= 0.0 }
ensures { ok(conv(result)) } { x }
`)
	got := verifyStatuses(t, bin, "--relax-modules", f)
	r := got["grade"]
	if r.Status != "skipped" || !strings.Contains(r.Reason, "Option[float]") {
		t.Fatalf("grade = %+v, want skipped naming Option[float]", r)
	}
}
