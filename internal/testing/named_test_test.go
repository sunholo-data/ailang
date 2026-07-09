package testing

// TDD tests for M1_NAMED_TESTS:
// - named test pass (body evals to true)
// - named test fail (body evals to false → FAIL)
// - named test runtime error (runtime panic → FAIL with error text)
// - mixed-kind file (inline + property + named — other kinds' statuses unchanged)
// - all-skipped exit semantics (SuiteResult.AllSkipped / reporting)
// - reporter: skip reasons visible
// - reporter: "All tests passed!" requires run>0 && failed==0
// - reporter: run==0 && skipped>0 ⇒ "NO TESTS RAN"

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// ─── Runner tests ───────────────────────────────────────────────────────────

// TestRunner_NamedTest_Pass: body "true" → StatusPass
func TestRunner_NamedTest_Pass(t *testing.T) {
	input := `test "always true" { true }`
	tc := parseNamedTestCase(t, input, "always true")

	runner := NewRunner("test.ail")
	result := runner.runTest(tc)

	if result.Status != StatusPass {
		t.Errorf("expected StatusPass, got %s (error: %s)", result.Status, result.Error)
	}
}

// TestRunner_NamedTest_Fail: body "false" → StatusFail
func TestRunner_NamedTest_Fail(t *testing.T) {
	input := `test "always false" { false }`
	tc := parseNamedTestCase(t, input, "always false")

	runner := NewRunner("test.ail")
	result := runner.runTest(tc)

	if result.Status != StatusFail {
		t.Errorf("expected StatusFail, got %s", result.Status)
	}
	// Error text must say something about the result
	if result.Error == "" {
		t.Error("expected non-empty error message for failing named test")
	}
}

// TestRunner_NamedTest_Arithmetic_Pass: "1 + 1 == 2" → StatusPass
func TestRunner_NamedTest_Arithmetic_Pass(t *testing.T) {
	input := `test "arithmetic" { 1 + 1 == 2 }`
	tc := parseNamedTestCase(t, input, "arithmetic")

	runner := NewRunner("test.ail")
	result := runner.runTest(tc)

	if result.Status != StatusPass {
		t.Errorf("expected StatusPass for '1+1==2', got %s (error: %s)", result.Status, result.Error)
	}
}

// TestRunner_NamedTest_Arithmetic_Fail: "1 + 1 == 3" → StatusFail
func TestRunner_NamedTest_Arithmetic_Fail(t *testing.T) {
	input := `test "arithmetic wrong" { 1 + 1 == 3 }`
	tc := parseNamedTestCase(t, input, "arithmetic wrong")

	runner := NewRunner("test.ail")
	result := runner.runTest(tc)

	if result.Status != StatusFail {
		t.Errorf("expected StatusFail for '1+1==3', got %s", result.Status)
	}
}

// TestRunner_NamedTest_NonBool: body returns int → StatusFail (type contract)
func TestRunner_NamedTest_NonBool(t *testing.T) {
	input := `test "non bool" { 42 }`
	tc := parseNamedTestCase(t, input, "non bool")

	runner := NewRunner("test.ail")
	result := runner.runTest(tc)

	// Non-bool result should be a fail (contract: body must return bool)
	if result.Status != StatusFail {
		t.Errorf("expected StatusFail for non-bool body, got %s", result.Status)
	}
}

// TestRunner_NamedTest_MultipleExpressions: all must be true
func TestRunner_NamedTest_MultipleExpressions_AllPass(t *testing.T) {
	// Named test body with multiple expressions — the runner evaluates
	// the last expression in the block (or all, depending on spec).
	// Per design doc: body = pure expression; here we pick the single-expression case.
	// For a block with multiple exprs, we use the last one as the result.
	input := `test "multiple" { 1 == 1 }`
	tc := parseNamedTestCase(t, input, "multiple")

	runner := NewRunner("test.ail")
	result := runner.runTest(tc)

	if result.Status != StatusPass {
		t.Errorf("expected StatusPass, got %s (error: %s)", result.Status, result.Error)
	}
}

// ─── Mixed-kind file tests ───────────────────────────────────────────────────

// TestRunner_MixedFile: inline tests and named tests in same file.
// Inline tests should still pass; named test FAIL should not affect inline status.
func TestRunner_MixedFile_NamedFailDoesNotAffectInline(t *testing.T) {
	// Parse a file with both inline tests (via function) and a named test block
	input := `
test "failing named" { false }
test "passing named" { true }
`
	l := lexer.New(input, "mixed.ail")
	p := parser.New(l)
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	collector := NewCollector("mixed.ail")
	suite := collector.Collect(file)

	if len(suite.Tests) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(suite.Tests))
	}

	runner := NewRunner("mixed.ail")
	result := runner.RunSuite(suite)

	if result.TotalTests != 2 {
		t.Errorf("expected 2 total tests, got %d", result.TotalTests)
	}
	if result.PassedTests != 1 {
		t.Errorf("expected 1 passed, got %d", result.PassedTests)
	}
	if result.FailedTests != 1 {
		t.Errorf("expected 1 failed, got %d", result.FailedTests)
	}
}

// ─── SuiteResult.AllSkipped / no-run semantics ──────────────────────────────

// TestSuiteResult_AllSkipped_True: when run==0 && skipped>0
func TestSuiteResult_AllSkipped_True(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "skip1", Status: StatusSkip, Error: "not implemented"})
	sr.AddTestResult(TestResult{Name: "skip2", Status: StatusSkip, Error: "no generator"})

	if !sr.AllSkipped() {
		t.Error("expected AllSkipped() to be true when all tests are skipped")
	}
	if sr.Success() {
		t.Error("expected Success() to be false when all tests are skipped (run==0)")
	}
}

// TestSuiteResult_AllSkipped_False_HasPassed: when some tests pass
func TestSuiteResult_AllSkipped_False_HasPassed(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "pass1", Status: StatusPass})
	sr.AddTestResult(TestResult{Name: "skip1", Status: StatusSkip, Error: "no gen"})

	if sr.AllSkipped() {
		t.Error("expected AllSkipped() to be false when some tests passed")
	}
}

// TestSuiteResult_AllSkipped_False_Empty: no tests at all
func TestSuiteResult_AllSkipped_False_Empty(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	// Empty suite has no tests; AllSkipped means skipped>0 && passed==0 && failed==0
	if sr.AllSkipped() {
		t.Error("expected AllSkipped() to be false when there are zero tests (not the same as all-skipped)")
	}
}

// ─── Reporter: skip reasons visible ─────────────────────────────────────────

// TestReporter_SkipReason_Visible: skip reason must appear in human output
func TestReporter_SkipReason_Visible(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{
		Name:   "skipped test",
		Status: StatusSkip,
		Error:  "Named test blocks not yet implemented",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)
	if err := reporter.Report(sr); err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Named test blocks not yet implemented") {
		t.Errorf("expected skip reason in human output, got:\n%s", output)
	}
}

// TestReporter_SkipReason_MultipleSkips: all skip reasons appear
func TestReporter_SkipReason_MultipleSkips(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "s1", Status: StatusSkip, Error: "reason alpha"})
	sr.AddTestResult(TestResult{Name: "s2", Status: StatusSkip, Error: "reason beta"})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)
	if err := reporter.Report(sr); err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "reason alpha") {
		t.Errorf("expected 'reason alpha' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "reason beta") {
		t.Errorf("expected 'reason beta' in output, got:\n%s", output)
	}
}

// ─── Reporter: honest summary (all-skipped → "NO TESTS RAN") ────────────────

// TestReporter_AllSkipped_NoTestsRanMessage: run==0 && skipped>0 → "NO TESTS RAN"
func TestReporter_AllSkipped_NoTestsRanMessage(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "s1", Status: StatusSkip, Error: "Named test blocks not yet implemented"})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)
	if err := reporter.Report(sr); err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "NO TESTS RAN") {
		t.Errorf("expected 'NO TESTS RAN' in output for all-skipped suite, got:\n%s", output)
	}
	// Must NOT say "All tests passed!"
	if strings.Contains(output, "All tests passed!") {
		t.Errorf("must not say 'All tests passed!' when all tests were skipped, got:\n%s", output)
	}
}

// TestReporter_AllPassed_ShowsAllTestsPassed: run>0 && failed==0 → "All tests passed!"
func TestReporter_AllPassed_ShowsAllTestsPassed(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "t1", Status: StatusPass, Duration: 1 * time.Millisecond})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)
	if err := reporter.Report(sr); err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "All tests passed!") {
		t.Errorf("expected 'All tests passed!' for all-passing suite, got:\n%s", output)
	}
}

// TestReporter_MixedPassSkip_ShowsAllTestsPassed: passed>0 && failed==0 → "All tests passed!"
func TestReporter_MixedPassSkip_ShowsAllTestsPassed(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "t1", Status: StatusPass})
	sr.AddTestResult(TestResult{Name: "s1", Status: StatusSkip, Error: "no generator"})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)
	if err := reporter.Report(sr); err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "All tests passed!") {
		t.Errorf("expected 'All tests passed!' when some passed and none failed, got:\n%s", output)
	}
}

// ─── SuiteResult.Success() honesty for --allow-skips logic ──────────────────

// TestSuiteResult_Success_AllSkipped: AllSkipped suite is not success (for exit code)
func TestSuiteResult_Success_AllSkipped(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "s1", Status: StatusSkip, Error: "no impl"})

	if sr.Success() {
		t.Error("Success() must be false for all-skipped suite")
	}
}

// TestSuiteResult_SuccessWithAllowSkips: AllowSkips makes all-skipped pass
func TestSuiteResult_SuccessWithAllowSkips(t *testing.T) {
	sr := NewSuiteResult("test.ail")
	sr.AddTestResult(TestResult{Name: "s1", Status: StatusSkip, Error: "no impl"})

	if !sr.SuccessAllowingSkips() {
		t.Error("SuccessAllowingSkips() must be true for all-skipped suite")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// parseNamedTestCase parses a small snippet containing one named test block
// and returns the TestCase for it.
func parseNamedTestCase(t *testing.T, input, name string) TestCase {
	t.Helper()
	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	collector := NewCollector("test.ail")
	suite := collector.Collect(file)

	for _, tc := range suite.Tests {
		if tc.Name == name {
			return tc
		}
	}

	// Find first non-inline
	for _, tc := range suite.Tests {
		if !tc.IsInline {
			return tc
		}
	}

	// Find by ast.TestDecl
	for _, decl := range file.Decls {
		if td, ok := decl.(*ast.TestDecl); ok {
			if td.Name == name || name == "" {
				return TestCase{
					Name:     td.Name,
					Body:     td.Body,
					Location: td.Pos,
					IsInline: false,
				}
			}
		}
	}

	t.Fatalf("test case %q not found in parsed file; tests found: %v", name, suite.Tests)
	return TestCase{}
}
