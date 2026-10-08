package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// `ailang verify` on matches the SMT match form cannot express. Before the fix,
// a match on int/string/bool literals was emitted as an SMT `match` (a Z3
// parse error on every such function — strings even lost their quotes), and a
// guarded arm was emitted with its guard dropped, which is unsound. The file
// mode also exited 0 when every function errored.

// TestVerify_LiteralMatchVerifies: literal matches on int, string and bool
// (with and without a wildcard) encode as ite chains and VERIFY.
func TestVerify_LiteralMatchVerifies(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m

export type Outcome = Rejected | Proceeded deriving (Eq)

export func intMatch(n: int) -> int ! {}
ensures { result >= 1 }
{
  match n { 0 => 1, 7 => 4, _ => 2 }
}

export func strMatch(s: string) -> int ! {}
ensures { result >= 0 }
{
  match s { "a b" => 1, _ => 0 }
}

export func boolMatch(b: bool) -> int ! {}
ensures { result >= 0 }
{
  match b { true => 1, false => 0 }
}

export func guarded(n: int) -> int ! {}
ensures { result >= 0 }
{
  match n { 0 => 1, k if k > 5 => k, k if k < 0 => 0 - k, _ => 3 }
}

export func adtEq(n: int, o: Outcome) -> bool ! {}
ensures { result == ((o == match n { 0 => Rejected, _ => Proceeded }) && n >= 0) }
{
  (o == match n { 0 => Rejected, _ => Proceeded }) && n >= 0
}
`)
	got := verifyStatuses(t, bin, f)
	for _, fn := range []string{"intMatch", "strMatch", "boolMatch", "guarded", "adtEq"} {
		if r := got[fn]; r.Status != "verified" {
			t.Errorf("%s status = %q (reason %q), want verified", fn, r.Status, r.Reason)
		}
	}
}

// TestVerify_GuardIsNotDropped: each contract holds only if the guard is
// ignored, so a guard-dropping encoding would wrongly VERIFY them.
func TestVerify_GuardIsNotDropped(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	f := writeTempAil(t, `module m

export type Sign = Neg | Pos

export func intGuard(n: int) -> int ! {}
ensures { result >= 0 }
{
  match n { k if k > 100 => 1, _ => 0 - 1 }
}

export func adtGuard(x: Sign, n: int) -> int ! {}
ensures { result >= 0 }
{
  match x { Neg if n > 100 => 1, Neg => 0 - 1, Pos => 1 }
}
`)
	got := verifyStatuses(t, bin, f)
	if r := got["intGuard"]; r.Status != "counterexample" {
		t.Errorf("intGuard status = %q (reason %q), want counterexample", r.Status, r.Reason)
	}
	// A guarded constructor match has no sound encoding yet: it must skip, never verify.
	if r := got["adtGuard"]; r.Status != "skipped" {
		t.Errorf("adtGuard status = %q (reason %q), want skipped", r.Status, r.Reason)
	}
}

// TestVerify_SolverErrorExitsNonZero: a function the solver errors on is not
// a pass — file mode exits 1, matching --package and ai-check.
func TestVerify_SolverErrorExitsNonZero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake solver is a shell script")
	}
	bin := buildAilang(t)
	fake := filepath.Join(t.TempDir(), "z3")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'Z3 version 4.16.0 - 64 bit'; exit 0; fi\necho '(error \"boom\")'\nexit 1\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_Z3_PATH", fake)
	f := writeTempAil(t, `module m

export func f(n: int) -> int ! {}
ensures { result >= 0 }
{
  if n > 0 then n else 0
}
`)
	stdout, stderr, code := runAilangBin(t, bin, "verify", f)
	if code != 1 {
		t.Fatalf("exit = %d, want 1 on a solver error\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
}
