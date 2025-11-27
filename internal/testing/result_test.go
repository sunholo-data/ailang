package testing

import (
	"testing"
	"time"
)

func TestSuiteResult_NewSuiteResult(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	if sr.ModulePath != "test.ail" {
		t.Errorf("expected module path 'test.ail', got %q", sr.ModulePath)
	}

	if sr.TotalTests != 0 {
		t.Errorf("expected 0 tests, got %d", sr.TotalTests)
	}

	if sr.PassedTests != 0 {
		t.Errorf("expected 0 passed tests, got %d", sr.PassedTests)
	}

	if sr.FailedTests != 0 {
		t.Errorf("expected 0 failed tests, got %d", sr.FailedTests)
	}
}

func TestSuiteResult_AddTestResult_Pass(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	result := TestResult{
		Name:     "test1",
		Status:   StatusPass,
		Duration: 100 * time.Millisecond,
	}

	sr.AddTestResult(result)

	if sr.TotalTests != 1 {
		t.Errorf("expected 1 total test, got %d", sr.TotalTests)
	}

	if sr.PassedTests != 1 {
		t.Errorf("expected 1 passed test, got %d", sr.PassedTests)
	}

	if sr.FailedTests != 0 {
		t.Errorf("expected 0 failed tests, got %d", sr.FailedTests)
	}

	if sr.TotalDuration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %v", sr.TotalDuration)
	}
}

func TestSuiteResult_AddTestResult_Fail(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	result := TestResult{
		Name:   "test1",
		Status: StatusFail,
		Error:  "assertion failed",
	}

	sr.AddTestResult(result)

	if sr.FailedTests != 1 {
		t.Errorf("expected 1 failed test, got %d", sr.FailedTests)
	}

	if sr.PassedTests != 0 {
		t.Errorf("expected 0 passed tests, got %d", sr.PassedTests)
	}
}

func TestSuiteResult_AddTestResult_Skip(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	result := TestResult{
		Name:   "test1",
		Status: StatusSkip,
	}

	sr.AddTestResult(result)

	if sr.SkippedTests != 1 {
		t.Errorf("expected 1 skipped test, got %d", sr.SkippedTests)
	}

	if sr.PassedTests != 0 {
		t.Errorf("expected 0 passed tests, got %d", sr.PassedTests)
	}
}

func TestSuiteResult_AddPropertyResult(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	result := PropertyResult{
		Name:     "prop1",
		Status:   StatusPass,
		TestsRun: 100,
		Duration: 200 * time.Millisecond,
	}

	sr.AddPropertyResult(result)

	if sr.TotalTests != 1 {
		t.Errorf("expected 1 total test, got %d", sr.TotalTests)
	}

	if sr.PassedTests != 1 {
		t.Errorf("expected 1 passed test, got %d", sr.PassedTests)
	}

	if len(sr.Properties) != 1 {
		t.Errorf("expected 1 property result, got %d", len(sr.Properties))
	}
}

func TestSuiteResult_Success_AllPass(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	sr.AddTestResult(TestResult{Name: "test1", Status: StatusPass})
	sr.AddTestResult(TestResult{Name: "test2", Status: StatusPass})

	if !sr.Success() {
		t.Error("expected Success() to be true when all tests pass")
	}
}

func TestSuiteResult_Success_OneFails(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	sr.AddTestResult(TestResult{Name: "test1", Status: StatusPass})
	sr.AddTestResult(TestResult{Name: "test2", Status: StatusFail})

	if sr.Success() {
		t.Error("expected Success() to be false when any test fails")
	}
}

func TestSuiteResult_Success_NoTests(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	if sr.Success() {
		t.Error("expected Success() to be false when no tests run")
	}
}

func TestSuiteResult_Summary(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	sr.AddTestResult(TestResult{Name: "test1", Status: StatusPass, Duration: 10 * time.Millisecond})
	sr.AddTestResult(TestResult{Name: "test2", Status: StatusPass, Duration: 20 * time.Millisecond})
	sr.AddTestResult(TestResult{Name: "test3", Status: StatusFail, Duration: 15 * time.Millisecond})

	summary := sr.Summary()

	// Check that summary contains key information
	expectedParts := []string{"3 tests", "2 passed", "1 failed", "0 skipped"}
	for _, part := range expectedParts {
		if !contains(summary, part) {
			t.Errorf("expected summary to contain %q, got: %s", part, summary)
		}
	}
}

func TestSuiteResult_Summary_NoTests(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	summary := sr.Summary()

	if summary != "No tests found" {
		t.Errorf("expected 'No tests found', got %q", summary)
	}
}

func TestSuiteResult_MultipleResults(t *testing.T) {
	sr := NewSuiteResult("test.ail")

	sr.AddTestResult(TestResult{Name: "test1", Status: StatusPass, Duration: 10 * time.Millisecond})
	sr.AddTestResult(TestResult{Name: "test2", Status: StatusFail, Duration: 20 * time.Millisecond})
	sr.AddTestResult(TestResult{Name: "test3", Status: StatusSkip, Duration: 5 * time.Millisecond})
	sr.AddPropertyResult(PropertyResult{Name: "prop1", Status: StatusPass, Duration: 50 * time.Millisecond})

	if sr.TotalTests != 4 {
		t.Errorf("expected 4 total tests, got %d", sr.TotalTests)
	}

	if sr.PassedTests != 2 {
		t.Errorf("expected 2 passed tests, got %d", sr.PassedTests)
	}

	if sr.FailedTests != 1 {
		t.Errorf("expected 1 failed test, got %d", sr.FailedTests)
	}

	if sr.SkippedTests != 1 {
		t.Errorf("expected 1 skipped test, got %d", sr.SkippedTests)
	}

	expectedDuration := 85 * time.Millisecond
	if sr.TotalDuration != expectedDuration {
		t.Errorf("expected duration %v, got %v", expectedDuration, sr.TotalDuration)
	}

	if len(sr.Tests) != 3 {
		t.Errorf("expected 3 test results, got %d", len(sr.Tests))
	}

	if len(sr.Properties) != 1 {
		t.Errorf("expected 1 property result, got %d", len(sr.Properties))
	}
}

// Helper function - uses findSubstring from executor.go
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}
