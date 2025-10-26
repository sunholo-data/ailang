package testing

import (
	"time"

	"github.com/sunholo/ailang/internal/ast"
)

// Runner executes tests and properties.
type Runner struct {
	modulePath string
}

// NewRunner creates a new test runner.
func NewRunner(modulePath string) *Runner {
	return &Runner{
		modulePath: modulePath,
	}
}

// RunSuite executes all tests in a test suite and returns aggregated results.
func (r *Runner) RunSuite(suite *TestSuite) *SuiteResult {
	result := NewSuiteResult(suite.ModulePath)

	// Run all tests
	for _, testCase := range suite.Tests {
		testResult := r.runTest(testCase)
		result.AddTestResult(testResult)
	}

	// Run all properties (basic implementation - full property testing in Days 6-8)
	for _, propCase := range suite.Properties {
		propResult := r.runProperty(propCase)
		result.AddPropertyResult(propResult)
	}

	return result
}

// runTest executes a single test case.
// For Day 4, this is a basic implementation that just validates structure.
// Full evaluation integration will be added when integrated with the pipeline.
func (r *Runner) runTest(testCase TestCase) TestResult {
	start := time.Now()

	result := TestResult{
		Name:     testCase.Name,
		Status:   StatusSkip, // Skip for now - full evaluation in integration
		Location: testCase.Location.String(),
		Error:    "Test execution requires pipeline integration (Day 4 basic implementation)",
	}

	result.Duration = time.Since(start)
	return result
}

// runProperty executes a property-based test (stub for now).
// Full implementation in Days 6-8 with generators and shrinking.
func (r *Runner) runProperty(propCase PropertyCase) PropertyResult {
	start := time.Now()

	result := PropertyResult{
		Name:     propCase.Name,
		Status:   StatusSkip, // Skip for now, implement in Days 6-8
		Location: propCase.Location.String(),
		TestsRun: 0,
		Error:    "Property-based testing not yet implemented (Days 6-8)",
	}

	result.Duration = time.Since(start)
	return result
}

// RunTestsFromFile is a convenience function that parses, collects, and runs tests from a file.
func RunTestsFromFile(filePath string, ast *ast.File) (*SuiteResult, error) {
	// Collect tests from AST
	collector := NewCollector(filePath)
	suite := collector.Collect(ast)

	// Run tests
	runner := NewRunner(filePath)
	result := runner.RunSuite(suite)

	return result, nil
}
