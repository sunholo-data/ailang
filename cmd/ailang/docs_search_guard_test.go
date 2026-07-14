package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestDocsSearch_StillDesignDocSimHash is the M5 regression guard: the `docs search`
// subcommand (dispatched at docs.go on args[0]=="search", BEFORE flag parsing) must
// remain the design-doc SimHash search — M1's new --all-functions positional filter
// must NOT intercept it, and it must NOT become a stdlib search.
func TestDocsSearch_StillDesignDocSimHash(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	repoRoot := findRepoRootForTest(t)

	cmd := exec.Command(bin, "docs", "search", "timestamp")
	cmd.Dir = repoRoot // design-doc SimHash scans the repo's design_docs/
	cmd.Env = append(os.Environ(), "AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docs search failed: %v\n%s", err, out)
	}
	got := string(out)

	// It is still the SimHash search over design docs (not stdlib exports).
	if !strings.Contains(got, "SimHash search") {
		t.Errorf("docs search no longer runs the SimHash search — did M1 intercept it?\n%s", got)
	}
	// Results are design_docs/* paths, NOT `std/*.func:` lines from --all-functions.
	// SimHash prints native separators (design_docs\... on Windows) — normalize first.
	if !strings.Contains(strings.ReplaceAll(got, `\`, "/"), "design_docs/") {
		t.Errorf("docs search should return design_docs results, got:\n%s", got)
	}
	// It must NOT have degraded into the all-functions stdlib dump.
	if strings.Contains(got, "std/clock.now:") {
		t.Errorf("docs search leaked into the --all-functions stdlib dump:\n%s", got)
	}
}
