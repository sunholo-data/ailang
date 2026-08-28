package prompt

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionMetadata_FrozenFieldRoundTrips(t *testing.T) {
	var manifest VersionsManifest
	err := json.Unmarshal([]byte(`{"versions":{"f":{"frozen":{"at":"2026-08-27","reason":"banked","evidence_count":7,"evidence_example":"x.json"}},"m":{}}}`), &manifest)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Versions["f"].Frozen == nil || manifest.Versions["f"].Frozen.EvidenceCount != 7 {
		t.Fatalf("marker lost: %#v", manifest)
	}
	if manifest.Versions["m"].Frozen != nil {
		t.Fatal("unmarked entry became frozen")
	}
}

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

	// The error must name the unresolvable version and point somewhere useful.
	assertUnresolvedVersionError(t, err, "v99.99.99")
}

// TestLoadPrompt_BinaryVersionExplainsNamespace guards the distinction that
// caused an external report to read a healthy toolchain as eighteen minor
// versions of drift: prompt versions are their own series, so asking for the
// BINARY's version must say so rather than just "not found".
func TestLoadPrompt_BinaryVersionExplainsNamespace(t *testing.T) {
	_, err := LoadPrompt("0.34.0")
	if err == nil {
		t.Fatal("LoadPrompt(\"0.34.0\") should have failed but didn't")
	}
	if !strings.Contains(err.Error(), "their own series") {
		t.Errorf("a binary-shaped version should explain the namespace, got: %v", err)
	}

	// A plain typo within the prompt series must NOT get that explanation —
	// otherwise the hint is noise that appears on every mistake.
	_, err = LoadPrompt("v0.3.999")
	if err == nil {
		t.Fatal("LoadPrompt(\"v0.3.999\") should have failed but didn't")
	}
	if strings.Contains(err.Error(), "their own series") {
		t.Errorf("an in-series typo should not get the namespace explanation, got: %v", err)
	}
}

// assertUnresolvedVersionError checks the shared shape of an unknown-version
// error: it names the version the caller asked for, and tells them how to find
// the real ones.
func assertUnresolvedVersionError(t *testing.T, err error, version string) {
	t.Helper()
	msg := err.Error()
	if !strings.Contains(msg, version) {
		t.Errorf("error should name the requested version %q, got: %v", version, err)
	}
	if !strings.Contains(msg, "--list") {
		t.Errorf("error should point at 'ailang prompt --list', got: %v", err)
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

	assertUnresolvedVersionError(t, err, "v99.99.99")
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
