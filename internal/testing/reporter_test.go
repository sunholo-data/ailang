package testing

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestReporter_JSON_Empty(t *testing.T) {
	result := NewSuiteResult("test.ail")

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify fields
	if output["module_path"] != "test.ail" {
		t.Errorf("expected module_path 'test.ail', got %v", output["module_path"])
	}

	if output["total_tests"].(float64) != 0 {
		t.Errorf("expected total_tests 0, got %v", output["total_tests"])
	}

	if output["success"].(bool) != false {
		t.Error("expected success false (no tests run)")
	}
}

func TestReporter_JSON_PassingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "test1",
		Status:   StatusPass,
		Duration: 100 * time.Millisecond,
		Location: "test.ail:5:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify summary fields
	if output["passed_tests"].(float64) != 1 {
		t.Errorf("expected passed_tests 1, got %v", output["passed_tests"])
	}

	if output["failed_tests"].(float64) != 0 {
		t.Errorf("expected failed_tests 0, got %v", output["failed_tests"])
	}

	if output["success"].(bool) != true {
		t.Error("expected success true")
	}

	// Verify test details
	tests := output["tests"].([]interface{})
	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}

	test := tests[0].(map[string]interface{})
	if test["name"] != "test1" {
		t.Errorf("expected name 'test1', got %v", test["name"])
	}

	if test["status"] != "pass" {
		t.Errorf("expected status 'pass', got %v", test["status"])
	}

	if test["location"] != "test.ail:5:1" {
		t.Errorf("expected location 'test.ail:5:1', got %v", test["location"])
	}
}

func TestReporter_JSON_FailingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "test1",
		Status:   StatusFail,
		Error:    "assertion failed: expected 4, got 5",
		Duration: 50 * time.Millisecond,
		Location: "test.ail:10:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify summary
	if output["failed_tests"].(float64) != 1 {
		t.Errorf("expected failed_tests 1, got %v", output["failed_tests"])
	}

	if output["success"].(bool) != false {
		t.Error("expected success false")
	}

	// Verify test error
	tests := output["tests"].([]interface{})
	test := tests[0].(map[string]interface{})

	if test["error"] != "assertion failed: expected 4, got 5" {
		t.Errorf("expected error message, got %v", test["error"])
	}
}

func TestReporter_JSON_PropertyResult(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:         "prop1",
		Status:       StatusPass,
		TestsRun:     100,
		Duration:     500 * time.Millisecond,
		Location:     "test.ail:20:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatJSON, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify property details
	props := output["properties"].([]interface{})
	if len(props) != 1 {
		t.Fatalf("expected 1 property, got %d", len(props))
	}

	prop := props[0].(map[string]interface{})
	if prop["name"] != "prop1" {
		t.Errorf("expected name 'prop1', got %v", prop["name"])
	}

	if prop["tests_run"].(float64) != 100 {
		t.Errorf("expected tests_run 100, got %v", prop["tests_run"])
	}

	if prop["status"] != "pass" {
		t.Errorf("expected status 'pass', got %v", prop["status"])
	}
}

func TestReporter_Human_Empty(t *testing.T) {
	result := NewSuiteResult("test.ail")

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false) // No colors for easier testing

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for key sections
	if !strings.Contains(output, "Test Results") {
		t.Error("expected 'Test Results' header")
	}

	if !strings.Contains(output, "Module: test.ail") {
		t.Error("expected module path")
	}

	if !strings.Contains(output, "No tests found") {
		t.Error("expected 'No tests found' message")
	}
}

func TestReporter_Human_PassingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "simple test",
		Status:   StatusPass,
		Duration: 10 * time.Millisecond,
		Location: "test.ail:5:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for test details
	if !strings.Contains(output, "simple test") {
		t.Error("expected test name in output")
	}

	if !strings.Contains(output, "✓") {
		t.Error("expected pass icon")
	}

	if !strings.Contains(output, "All tests passed!") {
		t.Error("expected success message")
	}

	if !strings.Contains(output, "1 passed") {
		t.Error("expected '1 passed' in summary")
	}
}

func TestReporter_Human_FailingTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:     "failing test",
		Status:   StatusFail,
		Error:    "assertion failed",
		Location: "test.ail:10:1",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for test details
	if !strings.Contains(output, "failing test") {
		t.Error("expected test name in output")
	}

	if !strings.Contains(output, "✗") {
		t.Error("expected fail icon")
	}

	if !strings.Contains(output, "assertion failed") {
		t.Error("expected error message in output")
	}

	if !strings.Contains(output, "Some tests failed") {
		t.Error("expected failure message")
	}

	if !strings.Contains(output, "1 failed") {
		t.Error("expected '1 failed' in summary")
	}
}

func TestReporter_Human_SkippedTest(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{
		Name:   "skipped test",
		Status: StatusSkip,
		Error:  "Property testing not yet implemented",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for skip icon
	if !strings.Contains(output, "⊘") {
		t.Error("expected skip icon")
	}

	if !strings.Contains(output, "1 skipped") {
		t.Error("expected '1 skipped' in summary")
	}
}

func TestReporter_Human_MixedResults(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "test1", Status: StatusPass})
	result.AddTestResult(TestResult{Name: "test2", Status: StatusPass})
	result.AddTestResult(TestResult{Name: "test3", Status: StatusFail, Error: "failed"})
	result.AddTestResult(TestResult{Name: "test4", Status: StatusSkip})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check summary counts
	if !strings.Contains(output, "4 tests") {
		t.Error("expected '4 tests' in summary")
	}

	if !strings.Contains(output, "2 passed") {
		t.Error("expected '2 passed' in summary")
	}

	if !strings.Contains(output, "1 failed") {
		t.Error("expected '1 failed' in summary")
	}

	if !strings.Contains(output, "1 skipped") {
		t.Error("expected '1 skipped' in summary")
	}
}

func TestReporter_Human_PropertyResult(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:     "always positive",
		Status:   StatusPass,
		TestsRun: 100,
		Duration: 200 * time.Millisecond,
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for property details
	if !strings.Contains(output, "Properties:") {
		t.Error("expected 'Properties:' section")
	}

	if !strings.Contains(output, "always positive") {
		t.Error("expected property name in output")
	}

	if !strings.Contains(output, "100 cases") {
		t.Error("expected '100 cases' in output")
	}
}

func TestReporter_Human_FailingProperty(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddPropertyResult(PropertyResult{
		Name:         "fails sometimes",
		Status:       StatusFail,
		TestsRun:     50,
		FailingInput: "x=42, y=-1",
		Error:        "property violated",
	})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false)

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check for failure details
	if !strings.Contains(output, "Failing input") {
		t.Error("expected 'Failing input' label")
	}

	if !strings.Contains(output, "x=42, y=-1") {
		t.Error("expected failing input values")
	}

	if !strings.Contains(output, "property violated") {
		t.Error("expected error message")
	}
}

func TestReporter_Colors_Disabled(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "test1", Status: StatusPass})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, false) // Colors disabled

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check that no ANSI escape codes are present
	if strings.Contains(output, "\033[") {
		t.Error("expected no ANSI color codes when colors disabled")
	}
}

func TestReporter_Colors_Enabled(t *testing.T) {
	result := NewSuiteResult("test.ail")
	result.AddTestResult(TestResult{Name: "test1", Status: StatusPass})

	var buf bytes.Buffer
	reporter := NewReporter(FormatHuman, &buf, true) // Colors enabled

	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	output := buf.String()

	// Check that ANSI escape codes ARE present
	if !strings.Contains(output, "\033[") {
		t.Error("expected ANSI color codes when colors enabled")
	}
}

func TestReporter_UnknownFormat(t *testing.T) {
	result := NewSuiteResult("test.ail")

	var buf bytes.Buffer
	reporter := NewReporter("unknown", &buf, false)

	err := reporter.Report(result)
	if err == nil {
		t.Error("expected error for unknown format")
	}

	if !strings.Contains(err.Error(), "unknown output format") {
		t.Errorf("unexpected error message: %v", err)
	}
}
