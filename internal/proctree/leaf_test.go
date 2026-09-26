package proctree_test

import (
	"os/exec"
	"strings"
	"testing"
)

// M-V1-SIMPLIFY-S3 M5. proctree is a LEAF: stdlib and syscall only.
//
// internal/smt and internal/pkg are language-core packages (see
// internal/diag/closure_test.go) and they call proctree. If proctree ever
// reached internal/executor — where the previous shared copy lived — the
// core would link the executors and the closure test would fail. This test
// names the reason before that one fires.
func TestProctreeIsALeaf(t *testing.T) {
	const module = "github.com/sunholo-data/ailang/"

	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Control: `go list -deps .` always lists the package itself, so an
	// empty result would pass every absence check vacuously. os/exec is a
	// genuine dependency; if it is missing the build did not resolve.
	const mustSee = "os/exec"
	sawControl := false
	for _, d := range deps {
		if d == mustSee {
			sawControl = true
			break
		}
	}
	if !sawControl {
		t.Fatalf("instrument check failed: %d deps returned but %s absent", len(deps), mustSee)
	}

	for _, d := range deps {
		if strings.HasPrefix(d, module) && d != module+"internal/proctree" {
			t.Errorf("proctree must be a leaf but depends on %s; move the symbol DOWN, never import up", d)
		}
		// Every non-stdlib import path has a dot in its first element.
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") && !strings.HasPrefix(d, module) {
			t.Errorf("proctree must be stdlib-only but depends on %s", d)
		}
	}
}
