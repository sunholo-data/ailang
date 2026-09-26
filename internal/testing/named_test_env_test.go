package testing

// Regression tests for #906: two fixes verified live at HEAD (2026-09-13) but
// unprotected — both defects lived in the named-test body LIFTING path
// (executor.go's _namedtest_body_ temp file + pipeline run), and either could
// regress with the suite still green.
//
//   Defect 1 — a named-test-only file (no `tests [...]` block anywhere) lost
//   the $builtin module, so anything the evaluator cannot fold inline
//   (`not`, stdlib imports) failed with EVA002 "module not compiled: $builtin".
//
//   Defect 2 — in a project whose ailang.toml sets module_prefix, a named test
//   body importing a sibling by project path failed with LDR001: the lifted
//   module resolved relative to its own synthetic location instead of the
//   project root. A flat layout never reproduced it.
//
// Expected contract (issue #906): a named test body sees exactly what the
// enclosing module sees — its imports, and the builtins any AILANG
// expression needs.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// runTestsOnFile parses source AT its real path (so the lifting path sees the
// module's true location and can find the package manifest) and runs its tests
// via RunTestsFromFile. Unlike runInlineTestsOnSource, this keeps the file in
// its project layout instead of a flat temp dir — required to reproduce the
// module_prefix defect.
func runTestsOnFile(t *testing.T, path, source string) *SuiteResult {
	t.Helper()
	l := lexer.New(source, path)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	result, err := RunTestsFromFile(path, file)
	if err != nil {
		t.Fatalf("RunTestsFromFile: %v", err)
	}
	return result
}

// TestNamedTestOnly_FileHasBuiltinsAndStdlib pins defect 1: a file whose only
// tests are named blocks (no `tests [...]` anywhere) can use a $builtin
// (`not`) and a stdlib import (`startsWith`).
func TestNamedTestOnly_FileHasBuiltinsAndStdlib(t *testing.T) {
	source := `module named_only_repro

import std/string (startsWith)

test "folds inline, passes" { 1 + 1 == 2 }
test "needs a builtin"      { not false }
test "needs the stdlib"     { startsWith("abc", "ab") }
`
	result := runInlineTestsOnSource(t, source)

	if result.FailedTests > 0 {
		t.Errorf("named-test-only file: expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 3 {
		t.Errorf("named-test-only file: expected 3 passing tests, got %d", result.PassedTests)
	}
}

// TestNamedTest_ModulePrefixProjectImport pins defect 2: in a project with
// module_prefix = "src", a named test body can import a sibling by its
// project path (`import src/core/prompts`) — the lifted body must resolve
// against the project root, not its own synthetic temp location.
func TestNamedTest_ModulePrefixProjectImport(t *testing.T) {
	proj := t.TempDir()

	manifest := `[package]
name = "test/ldrprobe"
version = "0.1.0"
edition = "1"
module_prefix = "src"
`
	if err := os.WriteFile(filepath.Join(proj, "ailang.toml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write ailang.toml: %v", err)
	}
	core := filepath.Join(proj, "src", "core")
	if err := os.MkdirAll(core, 0755); err != nil {
		t.Fatalf("mkdir src/core: %v", err)
	}
	prompts := `module src/core/prompts

export func with_cache_hint(s: string, hint: string) -> string {
  s
}
`
	if err := os.WriteFile(filepath.Join(core, "prompts.ail"), []byte(prompts), 0644); err != nil {
		t.Fatalf("write prompts.ail: %v", err)
	}
	probe := `module src/core/probe
import src/core/prompts (with_cache_hint)
test "named only" { with_cache_hint("s", "") == "s" }
`
	probePath := filepath.Join(core, "probe.ail")
	if err := os.WriteFile(probePath, []byte(probe), 0644); err != nil {
		t.Fatalf("write probe.ail: %v", err)
	}

	result := runTestsOnFile(t, probePath, probe)

	if result.FailedTests > 0 {
		t.Errorf("module_prefix named test: expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 1 {
		t.Errorf("module_prefix named test: expected 1 passing test, got %d", result.PassedTests)
	}
}
