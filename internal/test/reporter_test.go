package test

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/schema"
)

func TestNewReport(t *testing.T) {
	report := NewReport()

	if report.Schema != schema.TestV1 {
		t.Errorf("Expected schema %s, got %s", schema.TestV1, report.Schema)
	}

	if report.RunID == "" {
		t.Error("Expected RunID to be generated")
	}

	if report.Cases == nil {
		t.Error("Expected Cases to be initialized")
	}

	// Check platform info
	if report.Platform.GoVersion != runtime.Version() {
		t.Errorf("Expected Go version %s, got %s", runtime.Version(), report.Platform.GoVersion)
	}

	if report.Platform.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, report.Platform.OS)
	}

	if report.Platform.Arch != runtime.GOARCH {
		t.Errorf("Expected Arch %s, got %s", runtime.GOARCH, report.Platform.Arch)
	}
}

func TestAddCase(t *testing.T) {
	report := NewReport()

	// Add passed case
	report.AddCase(Case{
		SID:    "T#001",
		Suite:  "unit",
		Name:   "test1",
		Status: "passed",
		TimeMs: 10,
	})

	// Add failed case
	report.AddCase(Case{
		SID:    "T#002",
		Suite:  "unit",
		Name:   "test2",
		Status: "failed",
		TimeMs: 15,
		Error:  "assertion failed",
	})

	// Add errored case
	report.AddCase(Case{
		SID:    "T#003",
		Suite:  "integration",
		Name:   "test3",
		Status: "errored",
		TimeMs: 5,
		Error:  "runtime error",
	})

	// Add skipped case
	report.AddCase(Case{
		SID:    "T#004",
		Suite:  "integration",
		Name:   "test4",
		Status: "skipped",
		TimeMs: 0,
		Error:  "dependency not available",
	})

	// Verify counts
	if report.Counts.Total != 4 {
		t.Errorf("Expected total 4, got %d", report.Counts.Total)
	}
	if report.Counts.Passed != 1 {
		t.Errorf("Expected passed 1, got %d", report.Counts.Passed)
	}
	if report.Counts.Failed != 1 {
		t.Errorf("Expected failed 1, got %d", report.Counts.Failed)
	}
	if report.Counts.Errored != 1 {
		t.Errorf("Expected errored 1, got %d", report.Counts.Errored)
	}
	if report.Counts.Skipped != 1 {
		t.Errorf("Expected skipped 1, got %d", report.Counts.Skipped)
	}
}

func TestFinalize(t *testing.T) {
	report := NewReport()
	startTime := time.Now().Add(-100 * time.Millisecond) // Ensure some time has passed

	// Add cases in random order
	report.AddCase(Case{SID: "T#002", Suite: "unit", Name: "b_test", Status: "passed"})
	report.AddCase(Case{SID: "T#001", Suite: "unit", Name: "a_test", Status: "passed"})
	report.AddCase(Case{SID: "T#003", Suite: "integration", Name: "test", Status: "passed"})

	report.Finalize(startTime)

	// Check duration is set
	if report.DurationMs < 100 {
		t.Errorf("Expected DurationMs to be at least 100, got %d", report.DurationMs)
	}

	// Verify sorting by (suite, name)
	if report.Cases[0].Suite != "integration" {
		t.Error("Expected integration suite first")
	}
	if report.Cases[1].Name != "a_test" {
		t.Error("Expected a_test before b_test")
	}
	if report.Cases[2].Name != "b_test" {
		t.Error("Expected b_test last")
	}
}

func TestSetSeed(t *testing.T) {
	report := NewReport()

	if report.Seed != nil {
		t.Error("Expected Seed to be nil initially")
	}

	report.SetSeed(42)

	if report.Seed == nil || *report.Seed != 42 {
		t.Error("Expected Seed to be 42")
	}
}

func TestSetEnvLockDigest(t *testing.T) {
	report := NewReport()

	digest := "sha256:abcd1234"
	report.SetEnvLockDigest(digest)

	if report.EnvLockDigest != digest {
		t.Errorf("Expected digest %s, got %s", digest, report.EnvLockDigest)
	}
}

func TestToJSON(t *testing.T) {
	report := NewReport()
	report.SetSeed(12345)
	report.SetEnvLockDigest("sha256:test")

	// Test empty report
	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify required fields
	if parsed["schema"] != schema.TestV1 {
		t.Errorf("Expected schema %s, got %v", schema.TestV1, parsed["schema"])
	}

	if _, ok := parsed["run_id"]; !ok {
		t.Error("Expected run_id to be present")
	}

	if _, ok := parsed["counts"]; !ok {
		t.Error("Expected counts to be present")
	}

	if _, ok := parsed["cases"]; !ok {
		t.Error("Expected cases array to be present")
	}

	if _, ok := parsed["platform"]; !ok {
		t.Error("Expected platform to be present")
	}

	// Verify seed is included
	if parsed["seed"] != float64(12345) {
		t.Errorf("Expected seed 12345, got %v", parsed["seed"])
	}

	// Verify env lock digest
	if parsed["env_lock_digest"] != "sha256:test" {
		t.Errorf("Expected env_lock_digest sha256:test, got %v", parsed["env_lock_digest"])
	}
}

func TestEmptyReport(t *testing.T) {
	report := EmptyReport()

	if report.Counts.Total != 0 {
		t.Error("Expected total count to be 0")
	}

	if len(report.Cases) != 0 {
		t.Error("Expected no cases")
	}

	// Should produce valid JSON even when empty
	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed for empty report: %v", err)
	}

	// Check for total in the counts object
	if !strings.Contains(string(jsonData), `"total": 0`) && !strings.Contains(string(jsonData), `"total":0`) {
		t.Errorf("Expected JSON to show total as 0, got: %s", string(jsonData))
	}
}

func TestGenerateTestSID(t *testing.T) {
	// Test that SIDs are stable
	sid1 := GenerateTestSID("suite1", "test1")
	sid2 := GenerateTestSID("suite1", "test1")

	if sid1 != sid2 {
		t.Error("Expected same SID for same input")
	}

	// Test that different inputs produce different SIDs
	sid3 := GenerateTestSID("suite2", "test1")
	if sid1 == sid3 {
		t.Errorf("Expected different SID for different suite, got sid1=%s, sid3=%s", sid1, sid3)
	}

	// Verify format
	if !strings.HasPrefix(sid1, "T#") {
		t.Errorf("Expected SID to start with T#, got %s", sid1)
	}
}

func TestTestRunner(t *testing.T) {
	runner := NewRunner()

	// Run successful test
	runner.RunTest("suite1", "test1", func() error {
		return nil
	})

	// Run failing test
	runner.RunTest("suite1", "test2", func() error {
		return &testError{"test failed"}
	})

	// Skip a test
	runner.Skip("suite2", "test3", "not implemented")

	report := runner.GetReport()

	if report.Counts.Total != 3 {
		t.Errorf("Expected 3 total tests, got %d", report.Counts.Total)
	}

	if report.Counts.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", report.Counts.Passed)
	}

	if report.Counts.Failed != 1 {
		t.Errorf("Expected 1 failed test, got %d", report.Counts.Failed)
	}

	if report.Counts.Skipped != 1 {
		t.Errorf("Expected 1 skipped test, got %d", report.Counts.Skipped)
	}
}

// Helper error type
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
