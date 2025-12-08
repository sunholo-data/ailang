package testing

import (
	"fmt"
	"time"
)

// TestStatus represents the outcome of a test execution.
type TestStatus string

const (
	StatusPass TestStatus = "pass"
	StatusFail TestStatus = "fail"
	StatusSkip TestStatus = "skip"
)

// TestResult represents the outcome of a single test execution.
type TestResult struct {
	Name     string        // Test name
	Status   TestStatus    // Pass/fail/skip
	Duration time.Duration // Execution time
	Error    string        // Error message (if failed)
	Location string        // Source location (file:line:col)
}

// PropertyResult represents the outcome of a property-based test.
type PropertyResult struct {
	Name         string        // Property name
	Status       TestStatus    // Pass/fail/skip
	Duration     time.Duration // Total execution time
	TestsRun     int           // Number of test cases generated
	FailingInput string        // Minimal failing input (if failed)
	Error        string        // Error message (if failed)
	Location     string        // Source location
}

// SuiteResult aggregates results from all tests in a test suite.
type SuiteResult struct {
	ModulePath    string           // Module path
	Tests         []TestResult     // All test results
	Properties    []PropertyResult // All property results
	TotalTests    int              // Total number of tests
	PassedTests   int              // Number of passing tests
	FailedTests   int              // Number of failing tests
	SkippedTests  int              // Number of skipped tests
	TotalDuration time.Duration    // Total execution time
}

// NewSuiteResult creates a new empty suite result.
func NewSuiteResult(modulePath string) *SuiteResult {
	return &SuiteResult{
		ModulePath:    modulePath,
		Tests:         []TestResult{},
		Properties:    []PropertyResult{},
		TotalTests:    0,
		PassedTests:   0,
		FailedTests:   0,
		SkippedTests:  0,
		TotalDuration: 0,
	}
}

// AddTestResult adds a test result and updates counters.
func (sr *SuiteResult) AddTestResult(result TestResult) {
	sr.Tests = append(sr.Tests, result)
	sr.TotalTests++
	sr.TotalDuration += result.Duration

	switch result.Status {
	case StatusPass:
		sr.PassedTests++
	case StatusFail:
		sr.FailedTests++
	case StatusSkip:
		sr.SkippedTests++
	}
}

// AddPropertyResult adds a property result and updates counters.
func (sr *SuiteResult) AddPropertyResult(result PropertyResult) {
	sr.Properties = append(sr.Properties, result)
	sr.TotalTests++
	sr.TotalDuration += result.Duration

	switch result.Status {
	case StatusPass:
		sr.PassedTests++
	case StatusFail:
		sr.FailedTests++
	case StatusSkip:
		sr.SkippedTests++
	}
}

// Success returns true if all tests passed.
func (sr *SuiteResult) Success() bool {
	return sr.FailedTests == 0 && sr.TotalTests > 0
}

// Summary returns a human-readable summary of the results.
func (sr *SuiteResult) Summary() string {
	if sr.TotalTests == 0 {
		return "No tests found"
	}

	return fmt.Sprintf("%d tests: %d passed, %d failed, %d skipped (%v)",
		sr.TotalTests, sr.PassedTests, sr.FailedTests, sr.SkippedTests, sr.TotalDuration)
}
