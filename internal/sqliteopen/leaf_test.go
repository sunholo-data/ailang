package sqliteopen_test

import (
	"os/exec"
	"strings"
	"testing"
)

// sqliteopen is a LEAF: standard library, database/sql and the go-sqlite3
// driver, nothing under internal/. Every SQLite store (coordinator,
// messaging, observatory, platform/sharedmem) opens through it, so the
// moment it imports any of them the graph has a cycle. The compiler enforces
// the cycle; this test enforces the stricter rule and names why.
func TestSqliteopenIsALeaf(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Control: the driver is the one third-party dependency the package
	// genuinely has; a listing without it did not resolve and the absence
	// assertion below would pass vacuously.
	const mustSee = "github.com/mattn/go-sqlite3"
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

	const module = "github.com/sunholo-data/ailang/"
	const self = module + "internal/sqliteopen"
	for _, d := range deps {
		if d == self {
			continue
		}
		if strings.HasPrefix(d, module) {
			t.Errorf("sqliteopen must be a leaf but depends on %s; move the needed symbol DOWN, never import up", d)
			continue
		}
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") && d != mustSee {
			t.Errorf("sqliteopen may depend on the standard library and %s only, but depends on %s", mustSee, d)
		}
	}
}
