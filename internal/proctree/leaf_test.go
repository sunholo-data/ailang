package proctree_test

import (
	"os"
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
	// Every GOOS, not just the host: platform files carry their own imports,
	// and a Linux-only x/sys import passed this test on a darwin laptop and
	// failed only in Linux CI (2026-10-16, dumpable_linux.go).
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) { assertProctreeLeaf(t, goos) })
	}
}

func assertProctreeLeaf(t *testing.T, goos string) {
	const module = "github.com/sunholo-data/ailang/"

	cmd := exec.Command("go", "list", "-deps", ".")
	cmd.Env = append(os.Environ(), "GOOS="+goos)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("GOOS=%s go list -deps: %v", goos, err)
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
