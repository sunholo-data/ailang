package diag_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// M-V1-SIMPLIFICATION-PROGRAM Phase 1.1 (sprint M-V1-SIMPLIFY-S1 M6).
//
// The language core — what `ailang run/check/fmt/prompt/repl` need — must be a
// LEAF of the platform. Today it is not: `effects` reaches `ai`, `secrets`,
// sqlite and websocket; `pipeline` and `repl` reach `telemetry` and through it
// otel + grpc; `prompt` reaches `mcp_client`; `builtins` links the ollama SDK.
// So `ailang fmt` links 112 of 123 internal packages and a 100 MB binary.
// (124 until M-V1-SIMPLIFY-S5 M4 deleted internal/storage/migrate; the live
// numbers are the generated section of ARCHITECTURE.md, not this comment.)
//
// ARCHITECTURE.md draws the boundary but scripts/check_boundaries.sh polices
// it with a grep over direct imports of 13 x 4 packages, so it cannot see any
// of the above (they are transitive, or outside its two lists). This test uses
// the real import graph (`go list -deps`) instead.
//
// It is landed WITH the current leaks listed in closure_expected_violations.txt
// and it fails in BOTH directions:
//   - a violation not in the list  → a new leak; fix it or justify it there
//   - a listed violation no longer present → the list is stale; trim it
//
// so the list can only shrink. When it is empty the boundary is real and the
// gate row "closure_platform_packages = 0 / closure_leak_roots = 0" in
// tools/simplicity_metrics.sh is green by construction.
//
// The three lists (language roots, platform deny-list, third-party leak roots)
// live in tools/simplicity_metrics.sh and are READ from there, so the metric and
// the gate can never disagree about what "the core" is.

const module = "github.com/sunholo-data/ailang"

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		t.Fatalf("go list -m: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// scriptList extracts NAME="a b c" from tools/simplicity_metrics.sh.
func scriptList(t *testing.T, root, name string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "tools", "simplicity_metrics.sh"))
	if err != nil {
		t.Fatalf("read metrics script: %v", err)
	}
	re := regexp.MustCompile(`(?m)^` + name + `="([^"]+)"`)
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatalf("tools/simplicity_metrics.sh has no %s=\"...\" line; the metric and this gate share that list", name)
	}
	return strings.Fields(string(m[1]))
}

func expectedViolations(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open("closure_expected_violations.txt")
	if err != nil {
		t.Fatalf("open expected-violations list: %v", err)
	}
	defer f.Close()
	want := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		want[line] = false // false = not yet seen this run
	}
	return want
}

func TestLanguageCoreIsALeaf(t *testing.T) {
	root := repoRoot(t)
	roots := scriptList(t, root, "LANGUAGE_ROOTS")
	platform := scriptList(t, root, "PLATFORM_PKGS")
	leaks := scriptList(t, root, "LEAK_ROOTS")

	args := []string{"list", "-deps"}
	for _, r := range roots {
		args = append(args, "./"+r)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Control: prove the graph resolved before trusting any absence. The parser
	// is a genuine dependency of every language root; if it is missing, the
	// build did not resolve and every "not present" below would pass vacuously.
	sawControl := false
	for _, d := range deps {
		if d == module+"/internal/parser" {
			sawControl = true
			break
		}
	}
	if !sawControl {
		t.Fatalf("instrument check failed: %d deps returned but %s/internal/parser absent", len(deps), module)
	}

	// Collect actual violations, keyed the way the expected list spells them.
	actual := map[string]bool{}
	for _, d := range deps {
		for _, p := range platform {
			full := module + "/" + p
			if d == full || strings.HasPrefix(d, full+"/") {
				actual[p] = true
			}
		}
		for _, l := range leaks {
			if strings.HasPrefix(d, l) {
				actual[l] = true
			}
		}
	}

	want := expectedViolations(t)
	var newLeaks, stale []string
	for v := range actual {
		if _, ok := want[v]; ok {
			want[v] = true
		} else {
			newLeaks = append(newLeaks, v)
		}
	}
	for v, seen := range want {
		if !seen {
			stale = append(stale, v)
		}
	}
	sort.Strings(newLeaks)
	sort.Strings(stale)

	for _, v := range newLeaks {
		t.Errorf("NEW leak into the language core: %s is now reachable from %v.\n"+
			"The core must be a leaf of the platform. Move the needed symbol DOWN, register it "+
			"from cmd/ailang at init, or — only with a written reason — add it to "+
			"internal/diag/closure_expected_violations.txt.", v, roots)
	}
	for _, v := range stale {
		t.Errorf("closure_expected_violations.txt lists %s but the core no longer reaches it. "+
			"Delete that line — the list may only shrink, and a stale entry would hide the "+
			"leak coming back.", v)
	}
}
