package eval_analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHistoryPreservation verifies that running eval-report multiple times preserves history
func TestHistoryPreservation(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_dashboard.json")

	// Create existing dashboard with 2 versions in history
	existing := &DashboardJSON{
		Version:   "v0.3.9",
		Timestamp: "2025-10-15T00:00:00Z",
		TotalRuns: 100,
		History: []HistoryEntry{
			{Version: "v0.3.9", Timestamp: "2025-10-15T00:00:00Z", TotalRuns: 100, SuccessCount: 50, SuccessRate: 0.5},
			{Version: "v0.3.8", Timestamp: "2025-10-01T00:00:00Z", TotalRuns: 90, SuccessCount: 40, SuccessRate: 0.44},
		},
	}
	writeJSON(t, tmpFile, existing)

	// Create new matrix for v0.3.10
	matrix := &PerformanceMatrix{
		Version:   "v0.3.10",
		Timestamp: time.Now(),
		TotalRuns: 120,
		Languages: map[string]*LanguageStats{
			"ailang": {TotalRuns: 60, SuccessRate: 0.6},
			"python": {TotalRuns: 60, SuccessRate: 0.7},
		},
	}

	results := []*BenchmarkResult{
		{StdoutOk: true},
		{StdoutOk: true},
		{StdoutOk: false},
	}

	// Export (should preserve history)
	_, err := ExportBenchmarkJSON(matrix, nil, results, tmpFile)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read result
	result := readDashboard(t, tmpFile)

	// Verify: 3 versions now (v0.3.10, v0.3.9, v0.3.8)
	if len(result.History) != 3 {
		t.Fatalf("Expected 3 history entries, got %d", len(result.History))
	}

	// Check order (newest first)
	if result.History[0].Version != "v0.3.10" {
		t.Errorf("Expected history[0] = v0.3.10, got %s", result.History[0].Version)
	}
	if result.History[1].Version != "v0.3.9" {
		t.Errorf("Expected history[1] = v0.3.9, got %s", result.History[1].Version)
	}
	if result.History[2].Version != "v0.3.8" {
		t.Errorf("Expected history[2] = v0.3.8, got %s", result.History[2].Version)
	}

	// Verify current version is v0.3.10
	if result.Version != "v0.3.10" {
		t.Errorf("Expected version = v0.3.10, got %s", result.Version)
	}
}

// TestDuplicateVersionUpdate verifies that rerunning with same version updates entry
func TestDuplicateVersionUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_dashboard.json")

	// Existing history has v0.3.9 with 50% success
	existing := &DashboardJSON{
		Version:   "v0.3.9",
		Timestamp: "2025-10-15T00:00:00Z",
		TotalRuns: 100,
		History: []HistoryEntry{
			{Version: "v0.3.9", Timestamp: "2025-10-15T00:00:00Z", TotalRuns: 100, SuccessCount: 50, SuccessRate: 0.50},
		},
	}
	writeJSON(t, tmpFile, existing)

	// Export v0.3.9 again (rerun with updated data - now 60% success)
	matrix := &PerformanceMatrix{
		Version:   "v0.3.9",
		Timestamp: time.Now(),
		TotalRuns: 120,
		Languages: map[string]*LanguageStats{
			"ailang": {TotalRuns: 60, SuccessRate: 0.6},
			"python": {TotalRuns: 60, SuccessRate: 0.6},
		},
	}

	results := make([]*BenchmarkResult, 120)
	for i := 0; i < 72; i++ { // 72/120 = 60%
		results[i] = &BenchmarkResult{StdoutOk: true}
	}
	for i := 72; i < 120; i++ {
		results[i] = &BenchmarkResult{StdoutOk: false}
	}

	_, err := ExportBenchmarkJSON(matrix, nil, results, tmpFile)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read result
	result := readDashboard(t, tmpFile)

	// Verify: Still 1 entry, but updated
	if len(result.History) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(result.History))
	}

	// Check success rate updated to 60%
	if result.History[0].SuccessRate != 0.6 {
		t.Errorf("Expected success rate = 0.6, got %f", result.History[0].SuccessRate)
	}

	// Check total runs updated
	if result.History[0].TotalRuns != 120 {
		t.Errorf("Expected total runs = 120, got %d", result.History[0].TotalRuns)
	}
}

// TestValidationCatchesMissingVersion verifies validation rejects missing version
func TestValidationCatchesMissingVersion(t *testing.T) {
	d := &DashboardJSON{
		Version: "", // Missing!
		History: []HistoryEntry{{Version: "v0.3.9"}},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if err.Error() != "version required" {
		t.Errorf("Expected 'version required', got %v", err)
	}
}

// TestValidationCatchesDuplicates verifies validation rejects duplicate versions in history
func TestValidationCatchesDuplicates(t *testing.T) {
	d := &DashboardJSON{
		Version:   "v0.3.9",
		Timestamp: "2025-10-15T00:00:00Z",
		History: []HistoryEntry{
			{Version: "v0.3.9"},
			{Version: "v0.3.9"}, // Duplicate!
		},
	}

	err := d.Validate()
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if err.Error() != "duplicate version in history: v0.3.9" {
		t.Errorf("Expected 'duplicate version' error, got %v", err)
	}
}

// TestAtomicWriteRollsBackOnError verifies corrupted writes don't destroy old file
func TestAtomicWriteRollsBackOnError(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_dashboard.json")

	// Write valid initial data
	initial := &DashboardJSON{
		Version:   "v0.3.9",
		Timestamp: "2025-10-15T00:00:00Z",
		History:   []HistoryEntry{{Version: "v0.3.9"}},
	}
	writeJSON(t, tmpFile, initial)

	// Try to write invalid data (missing version)
	invalid := &DashboardJSON{
		Version:   "", // Invalid!
		Timestamp: "2025-10-16T00:00:00Z",
		History:   []HistoryEntry{{Version: "v0.3.10"}},
	}

	err := writeJSONAtomic(tmpFile, invalid)
	if err == nil {
		t.Fatal("Expected write to fail validation")
	}

	// Verify: Original file unchanged
	result := readDashboard(t, tmpFile)
	if result.Version != "v0.3.9" {
		t.Errorf("Expected v0.3.9 (original), got %s", result.Version)
	}

	// Verify: No temp file left behind
	tmpPath := tmpFile + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should be cleaned up")
	}
}

// TestLoadExistingDashboard verifies loading handles missing files
func TestLoadExistingDashboard(t *testing.T) {
	tmpDir := t.TempDir()
	missingFile := filepath.Join(tmpDir, "nonexistent.json")

	// Should return empty dashboard, not error
	dashboard, err := loadExistingDashboard(missingFile)
	if err != nil {
		t.Fatalf("Expected nil error for missing file, got %v", err)
	}

	if dashboard == nil {
		t.Fatal("Expected non-nil dashboard")
	}

	if len(dashboard.History) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(dashboard.History))
	}
}

// TestMergeHistoryOrder verifies history maintains reverse chronological order
func TestMergeHistoryOrder(t *testing.T) {
	dashboard := &DashboardJSON{
		History: []HistoryEntry{
			{Version: "v0.3.9"},
			{Version: "v0.3.8"},
			{Version: "v0.3.7"},
		},
	}

	// Add v0.3.10 (should be prepended)
	newEntry := HistoryEntry{Version: "v0.3.10"}
	mergeHistory(dashboard, newEntry)

	if len(dashboard.History) != 4 {
		t.Fatalf("Expected 4 entries, got %d", len(dashboard.History))
	}

	expected := []string{"v0.3.10", "v0.3.9", "v0.3.8", "v0.3.7"}
	for i, version := range expected {
		if dashboard.History[i].Version != version {
			t.Errorf("history[%d]: expected %s, got %s", i, version, dashboard.History[i].Version)
		}
	}
}

// Helper functions

func writeJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
}

func readDashboard(t *testing.T, path string) *DashboardJSON {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var dashboard DashboardJSON
	if err := json.Unmarshal(data, &dashboard); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	return &dashboard
}
