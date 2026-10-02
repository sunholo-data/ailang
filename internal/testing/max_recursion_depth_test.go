package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// `ailang test --max-recursion-depth` (stapledons_godot, 2026-10-01): a 10^4-step
// pure sweep hit RT_REC_003 at the default 10,000, and the error told the user to
// raise --max-recursion-depth, a flag `ailang test` did not accept. The limit
// now reaches every evaluator the test executor builds.
const deepSweepSource = `module deep_sweep

pure func go(i: int, n: int, acc: int) -> int = if i >= n then acc else go(i + 1, n, acc + 1)

test "sweep 12000" { go(0, 12000, 0) == 12000 }
`

func runWithDepth(t *testing.T, depth int) *SuiteResult {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "deep_sweep.ail")
	if err := os.WriteFile(path, []byte(deepSweepSource), 0o644); err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(deepSweepSource, path))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	cfg := TestConfig{WorkspaceRoot: dir, SeedMode: SeedModeDerived, MaxRecursionDepth: depth}
	res, err := RunTestsFromFileWithConfig(path, file, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestMaxRecursionDepth_ReachesTestBodies(t *testing.T) {
	if res := runWithDepth(t, 0); res.FailedTests != 1 || !strings.Contains(firstFailureError(res), "RT_REC_003") {
		t.Fatalf("default depth: failed=%d err=%q, want RT_REC_003", res.FailedTests, firstFailureError(res))
	}
	if res := runWithDepth(t, 20000); res.FailedTests != 0 || res.PassedTests != 1 {
		t.Fatalf("depth 20000: passed=%d failed=%d err=%q", res.PassedTests, res.FailedTests, firstFailureError(res))
	}
}
