package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// skipIfBinaryNotAvailable skips the test if ailang binary is not in PATH
func skipIfBinaryNotAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not in PATH - skipping integration test")
	}
}

// TestPromptCommand_Help tests the --help flag
func TestPromptCommand_Help(t *testing.T) {
	skipIfBinaryNotAvailable(t)
	cmd := exec.Command("ailang", "prompt", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should contain usage information
	if !strings.Contains(outputStr, "Usage: ailang prompt") {
		t.Error("Help output missing usage line")
	}

	if !strings.Contains(outputStr, "--version") {
		t.Error("Help output missing --version flag")
	}

	if !strings.Contains(outputStr, "--list") {
		t.Error("Help output missing --list flag")
	}

	if !strings.Contains(outputStr, "--info") {
		t.Error("Help output missing --info flag")
	}
}

// TestPromptCommand_List tests the --list flag
func TestPromptCommand_List(t *testing.T) {
	skipIfBinaryNotAvailable(t)
	cmd := exec.Command("ailang", "prompt", "--list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should list available versions
	if !strings.Contains(outputStr, "Available prompt versions") {
		t.Error("List output missing header")
	}

	// Should contain some known versions
	if !strings.Contains(outputStr, "v0.3.24") {
		t.Error("List output missing v0.3.24")
	}

	if !strings.Contains(outputStr, "v0.4.0") {
		t.Error("List output missing v0.4.0")
	}

	// Should mark active version
	if !strings.Contains(outputStr, "*") {
		t.Error("List output doesn't mark active version")
	}
}

// TestPromptCommand_Info tests the --info flag
func TestPromptCommand_Info(t *testing.T) {
	cmd := exec.Command("ailang", "prompt", "--version", "v0.3.24", "--info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show metadata
	if !strings.Contains(outputStr, "Version:") {
		t.Error("Info output missing Version field")
	}

	if !strings.Contains(outputStr, "File:") {
		t.Error("Info output missing File field")
	}

	if !strings.Contains(outputStr, "Created:") {
		t.Error("Info output missing Created field")
	}

	if !strings.Contains(outputStr, "Description:") {
		t.Error("Info output missing Description field")
	}

	// Should contain version number
	if !strings.Contains(outputStr, "v0.3.24") {
		t.Error("Info output missing version number")
	}
}

// TestPromptCommand_Default tests getting the default/active prompt
func TestPromptCommand_Default(t *testing.T) {
	cmd := exec.Command("ailang", "prompt")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should contain AILANG content
	if !strings.Contains(outputStr, "AILANG") {
		t.Error("Prompt output doesn't contain 'AILANG'")
	}

	// Should be substantial content (at least 1000 characters)
	if len(outputStr) < 1000 {
		t.Errorf("Prompt output too short (%d chars), expected >1000", len(outputStr))
	}
}

// TestPromptCommand_SpecificVersion tests getting a specific version
func TestPromptCommand_SpecificVersion(t *testing.T) {
	cmd := exec.Command("ailang", "prompt", "--version", "v0.3.24")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should contain AILANG content
	if !strings.Contains(outputStr, "AILANG") {
		t.Error("Prompt output doesn't contain 'AILANG'")
	}

	// Should be substantial content
	if len(outputStr) < 1000 {
		t.Errorf("Prompt output too short (%d chars), expected >1000", len(outputStr))
	}
}

// TestPromptCommand_Latest tests the "latest" keyword
func TestPromptCommand_Latest(t *testing.T) {
	cmd := exec.Command("ailang", "prompt", "--version", "latest")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should contain AILANG content
	if !strings.Contains(outputStr, "AILANG") {
		t.Error("Prompt output doesn't contain 'AILANG'")
	}

	// Should match default prompt (no version specified)
	cmdDefault := exec.Command("ailang", "prompt")
	defaultOutput, err := cmdDefault.CombinedOutput()
	if err != nil {
		t.Fatalf("Default command failed: %v", err)
	}

	if string(output) != string(defaultOutput) {
		t.Error("'latest' version doesn't match default version")
	}
}

// TestPromptCommand_InvalidVersion tests error handling for invalid version
func TestPromptCommand_InvalidVersion(t *testing.T) {
	skipIfBinaryNotAvailable(t)
	cmd := exec.Command("ailang", "prompt", "--version", "v99.99.99")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with error
	if err == nil {
		t.Fatal("Command should have failed for invalid version")
	}

	stderrStr := stderr.String()

	// Should contain error message
	if !strings.Contains(stderrStr, "not found") && !strings.Contains(stderrStr, "Error") {
		t.Errorf("Error message doesn't mention version not found: %s", stderrStr)
	}
}

// TestPromptCommand_Piping tests that output is suitable for piping
func TestPromptCommand_Piping(t *testing.T) {
	cmd := exec.Command("ailang", "prompt")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v\nStderr: %s", err, stderr.String())
	}

	// Prompt content should go to stdout
	if stdout.Len() == 0 {
		t.Error("No output to stdout")
	}

	// Should not have error output
	if stderr.Len() > 0 {
		t.Errorf("Unexpected stderr output: %s", stderr.String())
	}
}

// TestPromptCommand_InfoWithoutVersion tests --info requires --version
func TestPromptCommand_InfoWithoutVersion(t *testing.T) {
	// When --info is used without --version, it should default to latest
	cmd := exec.Command("ailang", "prompt", "--info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show info for latest/active version
	if !strings.Contains(outputStr, "Version:") {
		t.Error("Info output missing Version field")
	}

	// Should be marked as active
	if !strings.Contains(outputStr, "(active)") {
		t.Error("Info for latest should show (active) marker")
	}
}

// TestPromptCommand_InGlobalPath tests that ailang is in PATH
func TestPromptCommand_InGlobalPath(t *testing.T) {
	// This test verifies the binary is installed correctly
	_, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang not in PATH - run 'make install' first")
	}

	// If we got here, ailang is in PATH
	// Run a simple command to verify it works
	cmd := exec.Command("ailang", "prompt", "--help")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("ailang command failed even though it's in PATH: %v", err)
	}
}

// TestPromptCommand_EndsWithNewline tests that prompt ends with newline for piping
func TestPromptCommand_EndsWithNewline(t *testing.T) {
	skipIfBinaryNotAvailable(t)
	cmd := exec.Command("ailang", "prompt")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Note: Some prompts might not end with newline, but it's good practice
	// This test documents the behavior rather than enforcing it
	if len(output) > 0 && output[len(output)-1] != '\n' {
		t.Log("Note: Prompt doesn't end with newline (may affect piping)")
	}
}

// TestMain ensures we're running from project root
func TestMain(m *testing.M) {
	// Check if we're in the project root
	if _, err := os.Stat("prompts/versions.json"); os.IsNotExist(err) {
		// Try to find project root
		if err := os.Chdir("../.."); err != nil {
			panic("Cannot find project root")
		}
	}

	os.Exit(m.Run())
}
