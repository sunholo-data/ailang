package prompt

import (
	"strings"
	"testing"
)

func TestLoadPrompt_ActiveVersion(t *testing.T) {
	// Test loading with empty string (should use active version)
	content, err := LoadPrompt("")
	if err != nil {
		t.Fatalf("LoadPrompt(\"\") failed: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("LoadPrompt(\"\") returned empty content")
	}

	// Should contain AILANG teaching content
	if !strings.Contains(content, "AILANG") {
		t.Error("Prompt content doesn't contain 'AILANG'")
	}
}

func TestLoadPrompt_Latest(t *testing.T) {
	// Test loading with "latest" (should use active version)
	content, err := LoadPrompt("latest")
	if err != nil {
		t.Fatalf("LoadPrompt(\"latest\") failed: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("LoadPrompt(\"latest\") returned empty content")
	}

	// Should match the active version content
	activeContent, err := LoadPrompt("")
	if err != nil {
		t.Fatalf("LoadPrompt(\"\") failed: %v", err)
	}

	if content != activeContent {
		t.Error("LoadPrompt(\"latest\") != LoadPrompt(\"\") - should be the same")
	}
}

func TestLoadPrompt_SpecificVersion(t *testing.T) {
	// Test loading a specific version that should exist
	content, err := LoadPrompt("v0.3.24")
	if err != nil {
		t.Fatalf("LoadPrompt(\"v0.3.24\") failed: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("LoadPrompt(\"v0.3.24\") returned empty content")
	}

	// Should contain AILANG teaching content
	if !strings.Contains(content, "AILANG") {
		t.Error("v0.3.24 prompt content doesn't contain 'AILANG'")
	}
}

func TestLoadPrompt_InvalidVersion(t *testing.T) {
	// Test loading with invalid version (should error)
	_, err := LoadPrompt("v99.99.99")
	if err == nil {
		t.Fatal("LoadPrompt(\"v99.99.99\") should have failed but didn't")
	}

	// Error should mention the version wasn't found
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error message should mention 'not found', got: %v", err)
	}
}

func TestGetActiveVersion(t *testing.T) {
	active, err := GetActiveVersion()
	if err != nil {
		t.Fatalf("GetActiveVersion() failed: %v", err)
	}

	if active == "" {
		t.Fatal("GetActiveVersion() returned empty string")
	}

	// Active version should be valid version string (e.g., "v0.4.2")
	if !strings.HasPrefix(active, "v") {
		t.Errorf("Active version %q doesn't start with 'v'", active)
	}
}

func TestListVersions(t *testing.T) {
	versions, err := ListVersions()
	if err != nil {
		t.Fatalf("ListVersions() failed: %v", err)
	}

	if len(versions) == 0 {
		t.Fatal("ListVersions() returned empty list")
	}

	// Should include some known versions
	hasV0324 := false
	hasV040 := false
	for _, v := range versions {
		if v == "v0.3.24" {
			hasV0324 = true
		}
		if v == "v0.4.0" {
			hasV040 = true
		}
	}

	if !hasV0324 {
		t.Error("ListVersions() missing v0.3.24")
	}
	if !hasV040 {
		t.Error("ListVersions() missing v0.4.0")
	}
}

func TestGetVersionMetadata(t *testing.T) {
	// Test getting metadata for a specific version
	metadata, err := GetVersionMetadata("v0.3.24")
	if err != nil {
		t.Fatalf("GetVersionMetadata(\"v0.3.24\") failed: %v", err)
	}

	if metadata.File == "" {
		t.Error("Metadata file path is empty")
	}

	if metadata.Description == "" {
		t.Error("Metadata description is empty")
	}

	// File should be prompts/v0.3.24.md
	expectedFile := "prompts/v0.3.24.md"
	if metadata.File != expectedFile {
		t.Errorf("Expected file %q, got %q", expectedFile, metadata.File)
	}
}

func TestGetVersionMetadata_Latest(t *testing.T) {
	// Test getting metadata with "latest" (should resolve to active)
	metadata, err := GetVersionMetadata("latest")
	if err != nil {
		t.Fatalf("GetVersionMetadata(\"latest\") failed: %v", err)
	}

	// Should match metadata for active version
	activeVersion, err := GetActiveVersion()
	if err != nil {
		t.Fatalf("GetActiveVersion() failed: %v", err)
	}

	activeMetadata, err := GetVersionMetadata(activeVersion)
	if err != nil {
		t.Fatalf("GetVersionMetadata(%q) failed: %v", activeVersion, err)
	}

	if metadata.File != activeMetadata.File {
		t.Errorf("Latest metadata doesn't match active version metadata")
	}
}

func TestGetVersionMetadata_Invalid(t *testing.T) {
	// Test getting metadata for invalid version
	_, err := GetVersionMetadata("v99.99.99")
	if err == nil {
		t.Fatal("GetVersionMetadata(\"v99.99.99\") should have failed but didn't")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error message should mention 'not found', got: %v", err)
	}
}

func TestLoadVersionsManifest(t *testing.T) {
	// Test loading the versions manifest directly
	manifest, err := loadVersionsManifest()
	if err != nil {
		t.Fatalf("loadVersionsManifest() failed: %v", err)
	}

	if manifest.SchemaVersion == "" {
		t.Error("Manifest schema version is empty")
	}

	if len(manifest.Versions) == 0 {
		t.Error("Manifest has no versions")
	}

	if manifest.Active == "" {
		t.Error("Manifest active version is empty")
	}

	// Active version should exist in versions map
	_, ok := manifest.Versions[manifest.Active]
	if !ok {
		t.Errorf("Active version %q not found in versions map", manifest.Active)
	}
}
