package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// runInlineTestsOnSource is a test helper: write AILANG source to a temp file,
// parse it, and run inline tests via RunTestsFromFile. Returns the suite result.
func runInlineTestsOnSource(t *testing.T, source string) *SuiteResult {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_input.ail")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
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

// TestClusterEvalWithImportedHelper reproduces the bug where a tested function
// calls a local helper that itself calls an imported stdlib function.
// Before the fix: "cluster harness evaluation failed: cannot apply non-function value: <nil>"
func TestClusterEvalWithImportedHelper(t *testing.T) {
	source := `module test_cluster_import

import std/string (substring)

func find_char(i: int, s: string) -> int =
  if i < 0 then -1
  else if substring(s, i, i + 1) == "/" then i
  else find_char(i - 1, s)

func uses_it(s: string) -> int
  tests [
    ("/home/user", 5)
  ]
  { find_char(10, s) }
`
	result := runInlineTestsOnSource(t, source)
	if result.FailedTests > 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests == 0 {
		t.Error("expected at least 1 test to pass")
	}
}

// TestAliasImportCollision reproduces the bug where two modules export the same
// function name and one is imported with an alias.
// Before the fix: "harness evaluation failed: _list_length: expected List, got *eval.StringValue"
func TestAliasImportCollision(t *testing.T) {
	source := `module test_alias_collision

import std/string (length as str_len)
import std/list (length)

func check_str_len(s: string) -> int
  tests [
    ("hello", 5),
    ("", 0)
  ]
  { str_len(s) }
`
	result := runInlineTestsOnSource(t, source)
	if result.FailedTests > 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests == 0 {
		t.Error("expected at least 1 test to pass")
	}
}

// TestADTConstructorFromImportedModule reproduces the bug where inline test bodies use
// ADT constructors (Some/None) imported from another module.
// Before the fix: "harness evaluation failed: failed to resolve global $adt.make_Option_Some:
// module $adt not found or function make_Option_Some not in module"
func TestADTConstructorFromImportedModule(t *testing.T) {
	source := `module test_adt_harness

import std/option (Some, None)

func wrap_some(n: int) -> bool
  tests [
    (1, true),
    (0, true)
  ]
  { match Some(n) { Some(_) => true, None => false } }

func is_none(n: int) -> bool
  tests [
    (0, false)
  ]
  { match None { Some(_) => true, None => false } }
`
	result := runInlineTestsOnSource(t, source)
	if result.FailedTests > 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests == 0 {
		t.Error("expected at least 1 test to pass")
	}
}

// TestADTConstructorInCluster reproduces the bug on the cluster evaluation path:
// the tested function calls a local helper that uses an imported ADT constructor.
// Before the fix: same "$adt not found" error via EvaluateInlineTestsWithCluster.
func TestADTConstructorInCluster(t *testing.T) {
	source := `module test_adt_cluster

import std/option (Some, None)

func wrap(n: int) -> bool =
  match Some(n) { Some(_) => true, None => false }

func tested(n: int) -> bool
  tests [
    (1, true),
    (0, true)
  ]
  { wrap(n) }
`
	result := runInlineTestsOnSource(t, source)
	if result.FailedTests > 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests == 0 {
		t.Error("expected at least 1 test to pass")
	}
}

// firstFailureError returns the error message of the first failed test, for diagnostics.
func firstFailureError(r *SuiteResult) string {
	for _, tr := range r.Tests {
		if tr.Status == StatusFail {
			return tr.Error
		}
	}
	return "(no error recorded)"
}
