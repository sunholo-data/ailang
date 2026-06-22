package astedit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestInjectContract_VerifyEndToEnd is the R1-moat composition: inject a PROVIDED `ensures` into a
// contract-LESS candidate, then `ailang run --verify-contracts` must REJECT the runs-but-wrong one
// and PASS the correct one. Skips if ailang isn't on PATH (CI without a build).
func TestInjectContract_VerifyEndToEnd(t *testing.T) {
	bin, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang not on PATH")
	}
	base := "module test/cand\nimport std/io (println)\n" +
		"export func compute(x: int) -> int ! {} = %s\n" +
		"export func main() -> () ! {IO} {\n  let r = compute(3);\n  if r > 0 then println(\"p\") else println(\"n\")\n}\n"
	const spec = "ensures { result > 0 }"
	cases := []struct {
		name       string
		body       string
		wantReject bool
	}{
		{"wrong", "x - 100", true},    // compute(3) = -97 violates ensures
		{"right", "x * x + 1", false}, // compute(3) = 10 satisfies ensures
	}
	for _, c := range cases {
		injected, err := InjectContract(fmt.Sprintf(base, c.body), "cand.ail", "compute", spec)
		if err != nil {
			t.Fatalf("%s: inject: %v", c.name, err)
		}
		f := filepath.Join(t.TempDir(), "cand.ail")
		if err := os.WriteFile(f, []byte(injected), 0o644); err != nil {
			t.Fatal(err)
		}
		err = exec.Command(bin, "run", "--entry", "main", "--caps", "IO", "--verify-contracts", "--relax-modules", f).Run()
		if rejected := err != nil; rejected != c.wantReject {
			t.Errorf("%s: --verify-contracts rejected=%v, want %v (injected:\n%s)", c.name, rejected, c.wantReject, injected)
		}
	}
}
