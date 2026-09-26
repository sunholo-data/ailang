package statedir_test

import (
	"os/exec"
	"strings"
	"testing"
)

// statedir is a LEAF: standard library only. internal/storage implements the
// Store interfaces of coordinator, messaging and observatory, and every one of
// those resolves its database path through this package — so the moment
// statedir imports any of them the graph has a cycle and stops compiling.
// The Go compiler enforces the cycle; this test enforces the STRICTER rule
// (no internal import at all, no third-party import at all) and names why,
// which a build error does not.
func TestStatedirIsALeaf(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Control: `go list -deps .` always lists the package itself, so a check
	// that merely asserts "no forbidden dep" passes vacuously on a build that
	// did not resolve. Prove the instrument saw a real build by finding a
	// standard-library package this package genuinely imports.
	const mustSee = "path/filepath"
	sawControl := false
	for _, d := range deps {
		if d == mustSee {
			sawControl = true
			break
		}
	}
	if !sawControl {
		t.Fatalf("instrument check failed: %d deps returned but %s absent; statedir joins "+
			"paths, so this build did not resolve and the assertion below would pass vacuously",
			len(deps), mustSee)
	}

	const self = "github.com/sunholo-data/ailang/internal/statedir"
	for _, d := range deps {
		if d == self {
			continue
		}
		// Every non-stdlib import path has a dot in its first element
		// (github.com/…, gopkg.in/…, golang.org/…).
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") {
			t.Errorf("statedir must be a stdlib-only leaf but depends on %s", d)
		}
	}
}
