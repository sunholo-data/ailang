package testing

import (
	"fmt"
	"time"

	"github.com/sunholo/ailang/internal/ast"
)

// Runner executes tests and properties.
type Runner struct {
	modulePath string
	executor   *Executor
}

// NewRunner creates a new test runner.
func NewRunner(modulePath string) *Runner {
	return &Runner{
		modulePath: modulePath,
		executor:   NewExecutor(modulePath),
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
func (r *Runner) runTest(testCase TestCase) TestResult {
	start := time.Now()

	result := TestResult{
		Name:     testCase.Name,
		Location: testCase.Location.String(),
	}

	// For inline tests, Body contains tuple expressions: (input, expected)
	if testCase.IsInline {
		// Each expression in Body should be a tuple (input, expected)
		for i, expr := range testCase.Body {
			// Expected format: Tuple with 2 elements
			tuple, ok := expr.(*ast.Tuple)
			if !ok || len(tuple.Elements) != 2 {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("inline test %d: expected (input, expected) tuple, got %T", i, expr)
				result.Duration = time.Since(start)
				return result
			}

			input := tuple.Elements[0]
			expected := tuple.Elements[1]

			// Build function call: functionName(input)
			// FunctionCtx contains the function name being tested
			functionCall := &ast.FuncCall{
				Func: &ast.Identifier{Name: testCase.FunctionCtx},
				Args: []ast.Expr{input},
				Pos:  input.Position(),
			}

			// Evaluate function call
			actualValue, err := r.executor.EvaluateExpression(functionCall)
			if err != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: failed to evaluate %s(%v): %v", i, testCase.FunctionCtx, input, err)
				result.Duration = time.Since(start)
				return result
			}

			// Evaluate expected expression
			expectedValue, err := r.executor.EvaluateExpression(expected)
			if err != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: failed to evaluate expected: %v", i, err)
				result.Duration = time.Since(start)
				return result
			}

			// Compare values
			if !r.executor.CompareValues(actualValue, expectedValue) {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: expected %v, got %v", i, expectedValue, actualValue)
				result.Duration = time.Since(start)
				return result
			}
		}

		// All tests passed
		result.Status = StatusPass
	} else {
		// Non-inline tests (test "name" { ... } blocks)
		// For now, skip these - they're less common
		result.Status = StatusSkip
		result.Error = "Named test blocks not yet implemented"
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
