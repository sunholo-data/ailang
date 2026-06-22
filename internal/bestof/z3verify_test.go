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
