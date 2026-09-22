package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEqEvaluatorVMParity pins the seam between the two implementations of ==
// (M-EQ-DERIVE-CONTAINERS R-D6): the evaluator's valuesStructurallyEqual and the
// bytecode VM's runtimeEq/Value.Equal. The programs are PURE on purpose: an
// effectful main is bridged back to the evaluator under --bytecode and would
// compare the evaluator with itself.
//
// Unifying the two is out of scope for this sprint; each known divergence is
// asserted as documented, so fixing it makes this test fail and forces the
// expectation to be updated deliberately.
func TestEqEvaluatorVMParity(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		eval, vm string // expected result per backend
	}{
		// Records compare by field NAME in the evaluator and by POSITION in the
		// VM (which relies on canonical field order); literals written in a
		// different order must still agree.
		{"record field order", "mkA() == mkB()", "true", "true"},
		{"record field order neq", "mkA() == mkC()", "false", "false"},
		{"list", "[1, 2] == [1, 2]", "true", "true"},
		{"option", "Some(1) == None", "false", "false"},
		// Eq[Float] is lawful (NaN == NaN), and structural equality composes it.
		{"nested NaN", "[0.0 / 0.0] == [0.0 / 0.0]", "true", "true"},
		// DOCUMENTED DIVERGENCE: the VM's OpEq uses IEEE on a top-level float
		// (NaN != NaN) while the evaluator's Eq[Float] dictionary is lawful.
		// A blocker for any default --bytecode flip; do not "fix" by editing
		// this expectation.
		{"top-level NaN (divergent)", "(0.0 / 0.0) == (0.0 / 0.0)", "true", "false"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "eqparity.ail")
			prog := fmt.Sprintf(`module test/eqparity

type P = {x: int, y: string} deriving (Eq)

func mkA() -> P { {x: 1, y: "a"} }
func mkB() -> P { {y: "a", x: 1} }
func mkC() -> P { {y: "b", x: 1} }

export func main() -> bool = %s
`, c.body)
			if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AILANG_NO_CACHE", "1")
			for _, backend := range []struct {
				name string
				args []string
				want string
			}{
				{"evaluator", []string{"run", "--relax-modules", src}, c.eval},
				{"vm", []string{"run", "--bytecode", "--relax-modules", src}, c.vm},
			} {
				stdout, stderr, code := runCLI(t, backend.args...)
				if code != 0 {
					t.Fatalf("%s: exit %d\nstderr=%s", backend.name, code, stderr)
				}
				if backend.name == "vm" && !strings.Contains(stderr, "via bytecode VM") {
					t.Fatalf("vm: program did not run on the VM\nstderr=%s", stderr)
				}
				// Status lines share stdout; the program's value is the last line.
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				if got := lines[len(lines)-1]; got != backend.want {
					t.Errorf("%s: %s = %q, want %q", backend.name, c.body, got, backend.want)
				}
			}
		})
	}
}
