package bestof

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestZ3VerifyTierRanksAboveContracts: a Z3-verified candidate (score 4) outranks runs+contracts (3).
func TestZ3VerifyTierRanksAboveContracts(t *testing.T) {
	v := stubVerifier{m: map[string]Verdict{
		"a": {TypeChecks: true, Runs: true, ContractsPass: true}, // 3
		"b": {TypeChecks: true, Runs: true, Verifies: true},      // 4 (should win)
	}}
	if best, _ := SelectBest([]string{"a", "b"}, v); best != 1 {
		t.Errorf("z3-verified (b) should outrank runs+contracts (a); got idx %d", best)
	}
}

// TestZ3VerifyEndToEnd: real `ailang verify` (Z3 SMT) proves a valid contract and finds a counterexample
// for an invalid one — strictly stronger than runtime contracts. Skips if ailang/z3 absent.
func TestZ3VerifyEndToEnd(t *testing.T) {
	bin, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang not on PATH")
	}
	if _, err := exec.LookPath("z3"); err != nil {
		t.Skip("z3 not installed")
	}
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		src := "module test/v\nexport func f(x: int) -> int ! {} ensures { result > x } = " + body +
			"\nexport func main() -> () ! {IO} = ()\n"
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	provable := write("provable.ail", "x + 1") // x+1 > x for all x
	violable := write("violable.ail", "x - 1") // x-1 > x never
	v := AilangVerifier{Bin: bin, Caps: "IO", VerifyZ3: true, RelaxModules: true}
	if vp := v.Verify(provable); !vp.Verifies {
		t.Errorf("provable should Z3-verify, got %+v", vp)
	}
	if vv := v.Verify(violable); vv.Verifies {
		t.Errorf("violable must NOT Z3-verify (counterexample), got %+v", vv)
	}
	// violable listed first: a plain selector keeps it (runs); the Z3 tier flips to provable.
	if best, _ := SelectBest([]string{violable, provable}, v); best != 1 {
		t.Errorf("Z3 selector must pick the provable candidate (idx 1), got %d", best)
	}
}

// TestZ3CatchesRuntimePassingButUnprovable is the moat's depth: a candidate that PASSES the runtime
// contract on the executed input but is provably wrong for OTHER inputs is caught by Z3 (Verifies=false,
// score 3) and loses to a provably-correct candidate (Verifies=true, score 4). Runtime contracts alone
// (and any untyped harness) can't distinguish them. Skips if ailang/z3 absent.
func TestZ3CatchesRuntimePassingButUnprovable(t *testing.T) {
	bin, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang not on PATH")
	}
	if _, err := exec.LookPath("z3"); err != nil {
		t.Skip("z3 not installed")
	}
	dir := t.TempDir()
	write := func(name, computeBody string) string {
		p := filepath.Join(dir, name)
		src := "module test/c\nimport std/io (println)\n" +
			"export func compute(x: int) -> int ! {} ensures { result > x } = " + computeBody + "\n" +
			"export func main() -> () ! {IO} {\n  let r = compute(3);\n  if r > 0 then println(\"ok\") else println(\"no\")\n}\n"
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	provable := write("provable.ail", "x + 1")                              // x+1>x for all x
	runtimeOnly := write("runtime_only.ail", "if x == 3 then 5 else x - 1") // passes on x=3, Z3 counterexample for x!=3
	v := AilangVerifier{Bin: bin, Caps: "IO", VerifyContracts: true, VerifyZ3: true, RelaxModules: true}

	if vr := v.Verify(runtimeOnly); !vr.ContractsPass {
		t.Errorf("runtime_only should PASS the runtime contract on x=3, got %+v", vr)
	} else if vr.Verifies {
		t.Errorf("runtime_only must FAIL Z3 (counterexample for x!=3), got %+v", vr)
	}
	if vp := v.Verify(provable); !vp.Verifies {
		t.Errorf("provable should Z3-verify, got %+v", vp)
	}
	// runtime_only listed first: a runtime-contract selector keeps it; Z3 flips to the provable one.
	if best, _ := SelectBest([]string{runtimeOnly, provable}, v); best != 1 {
		t.Errorf("Z3 must pick the provably-correct candidate (idx 1) over runtime-only, got %d", best)
	}
}
