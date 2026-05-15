package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// runEnsuresFromSource writes the source to a temp file (ExtractFunctionBinding
// re-reads source from disk), then elaborates, runs the test runner, and
// returns aggregated results.
func runEnsuresFromSource(t *testing.T, src string) *SuiteResult {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ensures_test.ail")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	l := lexer.New(src, tmpFile)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	collector := NewCollector(tmpFile)
	suite := collector.Collect(file)

	runner := NewRunner(tmpFile)
	runner.executor.SetSourceFile(file)
	return runner.RunSuite(suite)
}

// TestRunEnsuresProperty_ViolationReportsCounterexample verifies that an
// intentionally buggy function with an ensures clause is caught and reported
// as a Fail with a counterexample (input value), not as a synthetic-source
// "empty program" error.
func TestRunEnsuresProperty_ViolationReportsCounterexample(t *testing.T) {
	src := `module ensures_test

export pure func clampBuggy(x: int) -> int
  ensures { result >= 0 && result <= 10 }
{
  if x < 0 then -1
  else if x > 10 then 10
  else x
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusFail {
		t.Errorf("Expected StatusFail for buggy clamp, got %v (error: %s)", pr.Status, pr.Error)
	}
	if !strings.Contains(pr.Error, "ensures violated") {
		t.Errorf("Expected error to contain 'ensures violated', got: %s", pr.Error)
	}
	if !strings.Contains(pr.Error, "x=") {
		t.Errorf("Expected counterexample to include parameter name 'x', got: %s", pr.Error)
	}
}

// TestRunEnsuresProperty_CorrectImplPasses verifies a correctly-implemented
// function with an ensures clause runs all 100 iterations and reports Pass.
func TestRunEnsuresProperty_CorrectImplPasses(t *testing.T) {
	src := `module ensures_test

export pure func clampOk(x: int) -> int
  ensures { result >= 0 && result <= 10 }
{
  if x < 0 then 0
  else if x > 10 then 10
  else x
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusPass {
		t.Errorf("Expected StatusPass for correct clamp, got %v (error: %s)", pr.Status, pr.Error)
	}
	if pr.TestsRun != 100 {
		t.Errorf("Expected 100 iterations on Pass, got %d", pr.TestsRun)
	}
}

// TestRunEnsuresProperty_MultiArgPredicateReferencesParams verifies that an
// ensures predicate referencing both `result` and a function parameter
// (e.g. `result >= x`) evaluates correctly — both names must be in scope.
//
// NOTE: arithmetic ops (`+`, `*`) in predicates require operator dictionary
// elaboration which the test harness doesn't perform. v1 of M-DX26 Phase 5
// supports comparison-only predicates against function parameters; arithmetic
// in predicates is tracked as a follow-up limitation.
func TestRunEnsuresProperty_MultiArgPredicateReferencesParams(t *testing.T) {
	src := `module ensures_test

export pure func maxOf(x: int, y: int) -> int
  ensures { result >= x }
{
  if x >= y then x else y
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusPass {
		t.Errorf("Expected StatusPass for correct maxOf, got %v (error: %s)", pr.Status, pr.Error)
	}
}

// TestRunEnsuresProperty_MultiArgViolationReportsAllArgs verifies that
// multi-argument counterexamples include all parameter names in the report.
func TestRunEnsuresProperty_MultiArgViolationReportsAllArgs(t *testing.T) {
	src := `module ensures_test

-- Buggy: returns the smaller value, but the contract claims result >= x.
-- Hence ensures will be violated whenever x > y.
export pure func badMax(x: int, y: int) -> int
  ensures { result >= x }
{
  if x >= y then y else x
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusFail {
		t.Fatalf("Expected StatusFail for buggy max, got %v (error: %s)", pr.Status, pr.Error)
	}
	if !strings.Contains(pr.Error, "x=") || !strings.Contains(pr.Error, "y=") {
		t.Errorf("Expected counterexample to include both 'x=' and 'y=', got: %s", pr.Error)
	}
}
