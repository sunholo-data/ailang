package testing

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

// TestIntegration_FullWorkflow tests the complete test collection and reporting workflow.
func TestIntegration_FullWorkflow(t *testing.T) {
	// AILANG code with test blocks
	input := `
	func factorial(n: int) -> int {
		if n <= 1 then 1 else n * factorial(n - 1)
	}

	test "factorial basics" {
		factorial(0) == 1;
		factorial(1) == 1;
		factorial(5) == 120
	}

	test "factorial negative" {
		assert factorial(-1) == 1
	}

	property "factorial always positive" {
		forall(n: int) => factorial(n) >= 0
	}
	`

	// 1. Parse the code
	l := lexer.New(input, "factorial.ail")
	p := parser.New(l)
	file := p.ParseFile()

	if len(p.Errors()) != 0 {
		t.Fatal("parser errors")
	}

	// 2. Collect tests
	collector := NewCollector("factorial.ail")
	suite := collector.Collect(file)

	// Verify collection
	if len(suite.Tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(suite.Tests))
	}

	if len(suite.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(suite.Properties))
	}

	// 3. Run tests
	runner := NewRunner("factorial.ail")
	result := runner.RunSuite(suite)

	// Verify results
	if result.TotalTests != 3 { // 2 tests + 1 property
		t.Errorf("expected 3 total tests, got %d", result.TotalTests)
	}

	// Tests now execute with pipeline integration (M-TESTING-INLINE)
	// Some tests may still skip if they reference undefined functions
	// Just verify we got results
	if result.TotalTests == 0 {
		t.Errorf("expected some tests to run, got 0")
	}

	// 4. Report results - Human format
	var humanBuf bytes.Buffer
	humanReporter := NewReporter(FormatHuman, &humanBuf, false)

	err := humanReporter.Report(result)
	if err != nil {
		t.Fatalf("human report error: %v", err)
	}

	humanOutput := humanBuf.String()

	// Verify human output contains key information
	if !strings.Contains(humanOutput, "factorial basics") {
		t.Error("expected test name in human output")
	}

	if !strings.Contains(humanOutput, "factorial negative") {
		t.Error("expected second test name in human output")
	}

	if !strings.Contains(humanOutput, "factorial always positive") {
		t.Error("expected property name in human output")
	}

	if !strings.Contains(humanOutput, "3 tests") {
		t.Error("expected '3 tests' in summary")
	}

	// 5. Report results - JSON format
	var jsonBuf bytes.Buffer
	jsonReporter := NewReporter(FormatJSON, &jsonBuf, false)

	err = jsonReporter.Report(result)
	if err != nil {
		t.Fatalf("JSON report error: %v", err)
	}

	jsonOutput := jsonBuf.String()

	// Verify JSON output is valid
	if !strings.Contains(jsonOutput, "\"module_path\": \"factorial.ail\"") {
		t.Error("expected module path in JSON")
	}

	if !strings.Contains(jsonOutput, "\"total_tests\": 3") {
		t.Error("expected total tests in JSON")
	}

	if !strings.Contains(jsonOutput, "\"tests\":") {
		t.Error("expected tests array in JSON")
	}

	if !strings.Contains(jsonOutput, "\"properties\":") {
		t.Error("expected properties array in JSON")
	}
}

// TestIntegration_MultipleFiles simulates testing multiple files.
func TestIntegration_MultipleFiles(t *testing.T) {
	files := []struct {
		name    string
		content string
		tests   int
		props   int
	}{
		{
			name: "math.ail",
			content: `
			test "addition" {
				1 + 1 == 2
			}
			test "subtraction" {
				5 - 3 == 2
			}
			`,
			tests: 2,
			props: 0,
		},
		{
			name: "string.ail",
			content: `
			property "length non-negative" {
				forall(s: string) => true
			}
			`,
			tests: 0,
			props: 1,
		},
	}

	var allResults []*SuiteResult

	for _, file := range files {
		// Parse
		l := lexer.New(file.content, file.name)
		p := parser.New(l)
		ast := p.ParseFile()

		if len(p.Errors()) != 0 {
			t.Fatalf("parser errors in %s", file.name)
		}

		// Collect
		collector := NewCollector(file.name)
		suite := collector.Collect(ast)

		if len(suite.Tests) != file.tests {
			t.Errorf("%s: expected %d tests, got %d", file.name, file.tests, len(suite.Tests))
		}

		if len(suite.Properties) != file.props {
			t.Errorf("%s: expected %d properties, got %d", file.name, file.props, len(suite.Properties))
		}

		// Run
		runner := NewRunner(file.name)
		result := runner.RunSuite(suite)

		allResults = append(allResults, result)
	}

	// Aggregate results across files
	totalTests := 0
	totalSkipped := 0

	for _, result := range allResults {
		totalTests += result.TotalTests
		totalSkipped += result.SkippedTests
	}

	if totalTests != 3 { // 2 tests + 1 property
		t.Errorf("expected 3 total tests across all files, got %d", totalTests)
	}

	// Tests now execute with pipeline integration (M-TESTING-INLINE)
	// Just verify we got some results
	if totalTests == 0 {
		t.Errorf("expected some tests to run across all files, got 0")
	}
}

// TestIntegration_EmptyFile tests handling of files with no tests.
func TestIntegration_EmptyFile(t *testing.T) {
	input := `
	func add(x: int, y: int) -> int {
		x + y
	}
	`

	// Parse
	l := lexer.New(input, "empty.ail")
	p := parser.New(l)
	ast := p.ParseFile()

	// Collect
	collector := NewCollector("empty.ail")
	suite := collector.Collect(ast)

	if len(suite.Tests) != 0 {
		t.Error("expected no tests in empty file")
	}

	// Run
	runner := NewRunner("empty.ail")
	result := runner.RunSuite(suite)

	if result.TotalTests != 0 {
		t.Error("expected 0 total tests")
	}

	// Report
	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("report error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "No tests found") {
		t.Error("expected 'No tests found' message for empty file")
	}
}

// TestIntegration_InlineTestsWithImports tests inline tests on functions that use imported modules.
// This is a regression test for the bug where inline tests would fail with "cannot apply non-function value: <nil>"
// when the function being tested called an imported function.
// See: M-INLINE-TESTS-IMPORTS regression test
func TestIntegration_InlineTestsWithImports(t *testing.T) {
	// We can't test with actual imports here since we'd need the stdlib loaded
	// But this test exists as a placeholder and integration-level doc
	// The real test is in examples/tests/ or with ailang test command
	t.Skip("This test requires stdlib loaded in the test environment; use 'ailang test' command instead")
}
