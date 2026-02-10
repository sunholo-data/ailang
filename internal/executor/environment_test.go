package executor

import (
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

	// FindNVMBinary should return v22.5.0 (newest with the binary)
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
