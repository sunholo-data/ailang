package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindNVMBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("NVM uses Unix paths; not applicable on Windows")
	}
	// Create a fake NVM directory structure
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	nvmDir := filepath.Join(tmpDir, ".nvm", "versions", "node")

	// Create versions: v18.20.0 (has claude), v20.0.0 (no claude), v22.5.0 (has claude)
	for _, ver := range []string{"v18.20.0", "v20.0.0", "v22.5.0"} {
		binDir := filepath.Join(nvmDir, ver, "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Put "claude" binary in v18 and v22 (not v20)
	for _, ver := range []string{"v18.20.0", "v22.5.0"} {
		binPath := filepath.Join(nvmDir, ver, "bin", "claude")
		if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// FindNVMBinary should return v22.5.0 (newest with the binary, proper semver sort)
	result := FindNVMBinary("claude")
	expected := filepath.Join(nvmDir, "v22.5.0", "bin", "claude")
	if result != expected {
		t.Errorf("FindNVMBinary(\"claude\") = %q, want %q", result, expected)
	}

	// Binary not installed anywhere
	result = FindNVMBinary("nonexistent")
	if result != "" {
		t.Errorf("FindNVMBinary(\"nonexistent\") = %q, want empty", result)
	}
}

func TestFindNVMBinary_NoNVM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("NVM uses Unix paths; not applicable on Windows")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// No .nvm directory at all
	result := FindNVMBinary("claude")
	if result != "" {
		t.Errorf("FindNVMBinary with no NVM dir = %q, want empty", result)
	}
}

func TestFindNVMNodeBinDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("NVM uses Unix paths; not applicable on Windows")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create one version with gemini
	binDir := filepath.Join(tmpDir, ".nvm", "versions", "node", "v25.0.0", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "gemini"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	result := FindNVMNodeBinDir("gemini")
	if result != binDir {
		t.Errorf("FindNVMNodeBinDir(\"gemini\") = %q, want %q", result, binDir)
	}

	// Not found
	result = FindNVMNodeBinDir("nonexistent")
	if result != "" {
		t.Errorf("FindNVMNodeBinDir(\"nonexistent\") = %q, want empty", result)
	}
}

func TestFindNVMBinary_SemverSort(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("NVM uses Unix paths; not applicable on Windows")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	nvmDir := filepath.Join(tmpDir, ".nvm", "versions", "node")

	// Create versions that would sort wrong with string comparison:
	// String sort: v9.0.0 > v25.5.0 (because "9" > "2")
	// Semver sort: v25.5.0 > v9.0.0 (correct)
	for _, ver := range []string{"v9.0.0", "v18.20.0", "v25.5.0"} {
		binDir := filepath.Join(nvmDir, ver, "bin")
		if err := os.MkdirAll(binDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	result := FindNVMBinary("claude")
	expected := filepath.Join(nvmDir, "v25.5.0", "bin", "claude")
	if result != expected {
		t.Errorf("FindNVMBinary semver sort: got %q, want %q", result, expected)
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch int
	}{
		{"v25.5.0", 25, 5, 0},
		{"v9.0.0", 9, 0, 0},
		{"v18.20.8", 18, 20, 8},
		{"v0.0.1", 0, 0, 1},
		{"invalid", 0, 0, 0},
		{"v1.2", 0, 0, 0}, // too few parts
	}
	for _, tt := range tests {
		major, minor, patch := parseSemver(tt.input)
		if major != tt.major || minor != tt.minor || patch != tt.patch {
			t.Errorf("parseSemver(%q) = (%d,%d,%d), want (%d,%d,%d)",
				tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
		}
	}
}

func TestFindNativeBinary(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Map Go arch to VSCode arch
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	platform := runtime.GOOS + "-" + arch

	// Create a fake VSCode extension with native binary
	extDir := filepath.Join(tmpDir, ".vscode", "extensions",
		fmt.Sprintf("anthropic.claude-code-2.1.45-%s", platform),
		"resources", "native-binary")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(extDir, "claude")
	if err := os.WriteFile(binPath, []byte("native-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	result := FindNativeBinary("claude")
	if result != binPath {
		t.Errorf("FindNativeBinary(\"claude\") = %q, want %q", result, binPath)
	}

	// Non-existent binary
	result = FindNativeBinary("nonexistent")
	if result != "" {
		t.Errorf("FindNativeBinary(\"nonexistent\") = %q, want empty", result)
	}
}

func TestFindNativeBinary_MultipleVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	platform := runtime.GOOS + "-" + arch

	// Create two extension versions
	for _, ver := range []string{"2.1.37", "2.1.45"} {
		extDir := filepath.Join(tmpDir, ".vscode", "extensions",
			fmt.Sprintf("anthropic.claude-code-%s-%s", ver, platform),
			"resources", "native-binary")
		if err := os.MkdirAll(extDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(extDir, "claude"), []byte("native"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	result := FindNativeBinary("claude")
	// Should pick the higher version (2.1.45)
	if result == "" {
		t.Fatal("FindNativeBinary returned empty, expected a path")
	}
	expected := filepath.Join(tmpDir, ".vscode", "extensions",
		fmt.Sprintf("anthropic.claude-code-2.1.45-%s", platform),
		"resources", "native-binary", "claude")
	if result != expected {
		t.Errorf("FindNativeBinary picked %q, want %q (newest)", result, expected)
	}
}

func TestRemoveEnvVar(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"CLAUDECODE=1",
		"HOME=/home/user",
		"CLAUDECODE_EXTRA=2", // Should NOT be removed (different key)
	}

	result := RemoveEnvVar(env, "CLAUDECODE")
	if len(result) != 3 {
		t.Errorf("RemoveEnvVar: got %d entries, want 3: %v", len(result), result)
	}
	for _, v := range result {
		if v == "CLAUDECODE=1" {
			t.Error("RemoveEnvVar did not remove CLAUDECODE=1")
		}
	}
	// Verify CLAUDECODE_EXTRA is kept (prefix matching should be exact key=)
	found := false
	for _, v := range result {
		if v == "CLAUDECODE_EXTRA=2" {
			found = true
		}
	}
	if !found {
		t.Error("RemoveEnvVar incorrectly removed CLAUDECODE_EXTRA")
	}
}

func TestRemoveEnvVar_NotPresent(t *testing.T) {
	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	result := RemoveEnvVar(env, "NONEXISTENT")
	if len(result) != 2 {
		t.Errorf("RemoveEnvVar on missing key: got %d entries, want 2", len(result))
	}
}
