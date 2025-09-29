package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewManifest(t *testing.T) {
	m := New()
	
	if m.Schema != SchemaVersion {
		t.Errorf("Schema = %s, want %s", m.Schema, SchemaVersion)
	}
	
	if m.SchemaVersion != "1.0.0" {
		t.Errorf("SchemaVersion = %s, want 1.0.0", m.SchemaVersion)
	}
	
	if m.Generator != "ailang verify-examples" {
		t.Errorf("Generator = %s, want 'ailang verify-examples'", m.Generator)
	}
	
	if len(m.Examples) != 0 {
		t.Errorf("Examples should be empty, got %d", len(m.Examples))
	}
}

func TestManifestValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Manifest)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid manifest",
			modify:  func(m *Manifest) {},
			wantErr: false,
		},
		{
			name: "invalid schema version",
			modify: func(m *Manifest) {
				m.Schema = "ailang.manifest/v2"
			},
			wantErr: true,
			errMsg:  "unsupported schema version",
		},
		{
			name: "duplicate example path",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Path: "test.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
					{Path: "test.ail", Status: StatusBroken, Mode: ModeFile, Broken: &BrokenInfo{ErrorCode: "PAR001"}},
				}
				m.UpdateStatistics()
			},
			wantErr: true,
			errMsg:  "duplicate example path",
		},
		{
			name: "missing path",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Status: StatusWorking, Mode: ModeFile},
				}
			},
			wantErr: true,
			errMsg:  "missing path",
		},
		{
			name: "missing status",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Path: "test.ail", Mode: ModeFile},
				}
			},
			wantErr: true,
			errMsg:  "missing status",
		},
		{
			name: "invalid status",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Path: "test.ail", Status: "invalid", Mode: ModeFile},
				}
			},
			wantErr: true,
			errMsg:  "invalid status",
		},
		{
			name: "working without expected",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Path: "test.ail", Status: StatusWorking, Mode: ModeFile},
				}
				m.UpdateStatistics()
			},
			wantErr: true,
			errMsg:  "missing expected output",
		},
		{
			name: "broken without error code",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Path: "test.ail", Status: StatusBroken, Mode: ModeFile, 
					 Broken: &BrokenInfo{Reason: "test"}},
				}
				m.UpdateStatistics()
			},
			wantErr: true,
			errMsg:  "missing error code",
		},
		{
			name: "non-ail extension",
			modify: func(m *Manifest) {
				m.Examples = []Example{
					{Path: "test.txt", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
				}
				m.UpdateStatistics()
			},
			wantErr: true,
			errMsg:  "must have .ail extension",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			tt.modify(m)
			
			err := m.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message should contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestStatisticsCalculation(t *testing.T) {
	m := New()
	m.Examples = []Example{
		{Path: "working1.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
		{Path: "working2.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
		{Path: "working3.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
		{Path: "broken1.ail", Status: StatusBroken, Mode: ModeFile, 
		 Broken: &BrokenInfo{Reason: "test", ErrorCode: "PAR001"}},
		{Path: "broken2.ail", Status: StatusBroken, Mode: ModeFile,
		 Broken: &BrokenInfo{Reason: "test", ErrorCode: "PAR002"}},
		{Path: "experimental1.ail", Status: StatusExperimental, Mode: ModeREPL},
	}
	
	m.UpdateStatistics()
	
	if m.Statistics.Total != 6 {
		t.Errorf("Total = %d, want 6", m.Statistics.Total)
	}
	
	if m.Statistics.Working != 3 {
		t.Errorf("Working = %d, want 3", m.Statistics.Working)
	}
	
	if m.Statistics.Broken != 2 {
		t.Errorf("Broken = %d, want 2", m.Statistics.Broken)
	}
	
	if m.Statistics.Experimental != 1 {
		t.Errorf("Experimental = %d, want 1", m.Statistics.Experimental)
	}
	
	expectedCoverage := 3.0 / 6.0
	if m.Statistics.Coverage != expectedCoverage {
		t.Errorf("Coverage = %f, want %f", m.Statistics.Coverage, expectedCoverage)
	}
}

func TestFindExample(t *testing.T) {
	m := New()
	m.Examples = []Example{
		{Path: "test1.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
		{Path: "test2.ail", Status: StatusBroken, Mode: ModeFile,
		 Broken: &BrokenInfo{Reason: "test", ErrorCode: "PAR001"}},
	}
	
	// Test finding existing example
	ex, found := m.FindExample("test1.ail")
	if !found {
		t.Error("Should find test1.ail")
	}
	if ex.Status != StatusWorking {
		t.Errorf("Status = %s, want %s", ex.Status, StatusWorking)
	}
	
	// Test finding non-existent example
	_, found = m.FindExample("test3.ail")
	if found {
		t.Error("Should not find test3.ail")
	}
}

func TestGetWorkingExamples(t *testing.T) {
	m := New()
	m.Examples = []Example{
		{Path: "working1.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
		{Path: "broken.ail", Status: StatusBroken, Mode: ModeFile,
		 Broken: &BrokenInfo{Reason: "test", ErrorCode: "PAR001"}},
		{Path: "working2.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{}},
	}
	
	working := m.GetWorkingExamples()
	if len(working) != 2 {
		t.Errorf("GetWorkingExamples() returned %d, want 2", len(working))
	}
	
	for _, ex := range working {
		if ex.Status != StatusWorking {
			t.Errorf("Got non-working example: %s", ex.Path)
		}
	}
}

func TestSchemaDigest(t *testing.T) {
	m := New()
	m.UpdateSchemaDigest()
	
	if m.SchemaDigest == "" {
		t.Error("SchemaDigest should not be empty")
	}
	
	if !strings.HasPrefix(m.SchemaDigest, "sha256:") {
		t.Errorf("SchemaDigest should start with 'sha256:', got %s", m.SchemaDigest)
	}
	
	// Verify digest is stable
	digest1 := m.calculateSchemaDigest()
	digest2 := m.calculateSchemaDigest()
	if digest1 != digest2 {
		t.Error("Schema digest should be deterministic")
	}
}

func TestGenerateREADMESection(t *testing.T) {
	m := New()
	m.GeneratedAt = time.Date(2024, 9, 29, 12, 0, 0, 0, time.UTC)
	m.Examples = []Example{
		{Path: "working.ail", Status: StatusWorking, Mode: ModeFile, 
		 Description: "A working example", Expected: &Expected{}},
		{Path: "broken.ail", Status: StatusBroken, Mode: ModeFile,
		 Broken: &BrokenInfo{
			Reason: "func not implemented",
			ErrorCode: "PAR003",
			Requires: []string{"func", "tests"},
			TrackedIssue: "https://github.com/sunholo/ailang/issues/1",
		}},
		{Path: "experimental.ail", Status: StatusExperimental, Mode: ModeREPL,
		 RequiresFeatures: []string{"effects"}, SkipReason: "Not ready"},
	}
	m.UpdateStatistics()
	
	readme := m.GenerateREADMESection()
	
	// Check for required sections
	if !strings.Contains(readme, "## Example Status") {
		t.Error("Missing '## Example Status' header")
	}
	
	if !strings.Contains(readme, "Coverage: 33.3%") {
		t.Error("Missing coverage percentage")
	}
	
	if !strings.Contains(readme, "✅ Working Examples") {
		t.Error("Missing working examples section")
	}
	
	if !strings.Contains(readme, "❌ Broken Examples") {
		t.Error("Missing broken examples section")
	}
	
	if !strings.Contains(readme, "🧪 Experimental Examples") {
		t.Error("Missing experimental examples section")
	}
	
	if !strings.Contains(readme, "working.ail") {
		t.Error("Missing working.ail in output")
	}
	
	if !strings.Contains(readme, "[#1](https://github.com/sunholo/ailang/issues/1)") {
		t.Error("Issue link not formatted correctly")
	}
	
	if !strings.Contains(readme, "2024-09-29 12:00:00 UTC") {
		t.Error("Missing timestamp")
	}
}

func TestLoadSaveManifest(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "manifest_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	
	// Create and save manifest
	m1 := New()
	m1.Examples = []Example{
		{Path: "test.ail", Status: StatusWorking, Mode: ModeFile, Expected: &Expected{
			Stdout: "hello\n",
			ExitCode: 0,
		}},
	}
	m1.UpdateStatistics()
	
	if err := m1.Save(manifestPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	// Check file exists
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("Manifest file not created: %v", err)
	}
	
	// Load and verify
	m2, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if m2.Schema != m1.Schema {
		t.Errorf("Schema mismatch: got %s, want %s", m2.Schema, m1.Schema)
	}
	
	if len(m2.Examples) != 1 {
		t.Errorf("Examples count = %d, want 1", len(m2.Examples))
	}
	
	if m2.Examples[0].Path != "test.ail" {
		t.Errorf("Example path = %s, want test.ail", m2.Examples[0].Path)
	}
	
	if m2.Statistics.Total != 1 {
		t.Errorf("Total = %d, want 1", m2.Statistics.Total)
	}
}

func TestEnvironmentDefaults(t *testing.T) {
	ex := Example{
		Path: "test.ail",
		Status: StatusWorking,
		Mode: ModeFile,
		Expected: &Expected{},
		Environment: &Environment{
			Seed: 42,
			Locale: "en_US.UTF-8",
			Timezone: "America/New_York",
		},
	}
	
	if ex.Environment.Seed != 42 {
		t.Errorf("Seed = %d, want 42", ex.Environment.Seed)
	}
	
	if ex.Environment.Locale != "en_US.UTF-8" {
		t.Errorf("Locale = %s, want en_US.UTF-8", ex.Environment.Locale)
	}
	
	if ex.Environment.Timezone != "America/New_York" {
		t.Errorf("Timezone = %s, want America/New_York", ex.Environment.Timezone)
	}
}