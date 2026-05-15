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

// TestRunEnsuresProperty_ArithmeticInPredicate verifies that arithmetic operators
// (`+`, `*`, etc.) inside an ensures predicate work — the runner pulls the
// already-lowered Core predicate from Meta.Contracts so dictionary calls have
// already replaced the raw operators (M-DX26 Phase 5.1).
func TestRunEnsuresProperty_ArithmeticInPredicate(t *testing.T) {
	src := `module ensures_test

export pure func add(x: int, y: int) -> int
  ensures { result == x + y }
{
  x + y
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusPass {
		t.Errorf("Expected StatusPass for correct add with arithmetic predicate, got %v (error: %s)", pr.Status, pr.Error)
	}
	if pr.TestsRun != 100 {
		t.Errorf("Expected 100 iterations on Pass, got %d", pr.TestsRun)
	}
}

// TestRunRequiresProperty_TautologyPasses (M-DX26 Phase 5.2) — a `requires`
// clause that is true for every int input should run all 100 iterations and
// report Pass. Previously this failed with `evaluation failed: empty program`.
func TestRunRequiresProperty_TautologyPasses(t *testing.T) {
	src := `module ensures_test

export pure func anyInt(x: int) -> int
  requires { x == x }
{
  x
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusPass {
		t.Errorf("Expected StatusPass for `requires { x == x }`, got %v (error: %s)", pr.Status, pr.Error)
	}
	if pr.TestsRun != 100 {
		t.Errorf("Expected 100 iterations on Pass, got %d", pr.TestsRun)
	}
}

// TestRunRequiresProperty_OutOfContractReportsSkip (M-DX26 Phase 5.2) — a
// `requires` that random inputs frequently violate (e.g. `x >= 0`) should report
// Skip with the offending input, not Fail (the function isn't being called and
// the inputs are simply out-of-contract).
func TestRunRequiresProperty_OutOfContractReportsSkip(t *testing.T) {
	src := `module ensures_test

export pure func absolute(x: int) -> int
  requires { x >= 0 }
{
  x
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusSkip {
		t.Errorf("Expected StatusSkip for unsatisfiable `requires { x >= 0 }`, got %v (error: %s)", pr.Status, pr.Error)
	}
	if !strings.Contains(pr.Error, "requires not satisfied") {
		t.Errorf("Expected 'requires not satisfied' in error, got: %s", pr.Error)
	}
	if !strings.Contains(pr.Error, "x=") {
		t.Errorf("Expected counterexample to include 'x=', got: %s", pr.Error)
	}
}

// TestRunEnsuresProperty_ArithmeticViolation verifies that a buggy implementation
// is caught even when the predicate contains arithmetic.
func TestRunEnsuresProperty_ArithmeticViolation(t *testing.T) {
	src := `module ensures_test

export pure func badAdd(x: int, y: int) -> int
  ensures { result == x + y }
{
  x + y + 1
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) == 0 {
		t.Fatalf("Expected at least one property result, got 0")
	}

	pr := result.Properties[0]
	if pr.Status != StatusFail {
		t.Errorf("Expected StatusFail for buggy add, got %v (error: %s)", pr.Status, pr.Error)
	}
	if !strings.Contains(pr.Error, "ensures violated") {
		t.Errorf("Expected 'ensures violated' in error, got: %s", pr.Error)
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
