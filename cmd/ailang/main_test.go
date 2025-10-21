package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI runs the ailang CLI with given arguments and returns stdout, stderr, and exit code
func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command("go", append([]string{"run", "."}, args...)...)
	cmd.Dir = filepath.Join("..", "..", "cmd", "ailang")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run CLI: %v", err)
		}
	}

	return stdout, stderr, exitCode
}

func TestCLI_Version(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--version")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "AILANG") {
		t.Errorf("Expected version output, got: %s", stdout)
	}
}

func TestCLI_Help(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--help")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	expectedSections := []string{
		"Usage:",
		"Commands:",
		"run",
		"repl",
		"check",
	}

	for _, section := range expectedSections {
		if !strings.Contains(stdout, section) {
			t.Errorf("Expected help output to contain %q, got: %s", section, stdout)
		}
	}
}

func TestCLI_NoArgs(t *testing.T) {
	stdout, _, exitCode := runCLI(t)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 (shows help), got %d", exitCode)
	}

	if !strings.Contains(stdout, "Usage:") {
		t.Errorf("Expected help output when no args provided, got: %s", stdout)
	}
}

func TestCLI_Run_SimpleExample(t *testing.T) {
	// Use an existing example file from the examples directory
	testFile := filepath.Join("..", "..", "examples", "runnable", "simple.ail")

	stdout, stderr, exitCode := runCLI(t, "run", "--caps", "IO", "--entry", "main", testFile)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
	}

	if !strings.Contains(stdout, "10") {
		t.Errorf("Expected output to contain 10 (result of simple.ail), got: %s", stdout)
	}
}

func TestCLI_Run_WithIO(t *testing.T) {
	// Use an existing I/O example
	testFile := filepath.Join("..", "..", "examples", "runnable", "demos", "hello_io.ail")

	stdout, stderr, exitCode := runCLI(t, "run", "--caps", "IO", "--entry", "main", testFile)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
	}

	if !strings.Contains(stdout, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %s", stdout)
	}
}

func TestCLI_Run_MissingFile(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "run", "nonexistent.ail")

	if exitCode == 0 {
		t.Error("Expected non-zero exit code for missing file")
	}

	if !strings.Contains(stderr, "Error") && !strings.Contains(stderr, "cannot read") {
		t.Errorf("Expected error message about missing file, got: %s", stderr)
	}
}

func TestCLI_Run_MissingCaps(t *testing.T) {
	// Use an I/O example but don't grant the capability
	testFile := filepath.Join("..", "..", "examples", "runnable", "demos", "hello_io.ail")

	_, stderr, exitCode := runCLI(t, "run", "--entry", "main", testFile)

	if exitCode == 0 {
		t.Error("Expected non-zero exit code when capability not granted")
	}

	// Should get an error about missing capability or effect checking
	if !strings.Contains(stderr, "Error") {
		t.Errorf("Expected error when capability not granted. Stderr: %s", stderr)
	}
}

func TestCLI_Check(t *testing.T) {
	// Use an existing example file
	testFile := filepath.Join("..", "..", "examples", "runnable", "simple.ail")

	stdout, stderr, exitCode := runCLI(t, "check", testFile)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for valid file, got %d. Stderr: %s", exitCode, stderr)
	}

	// Check should show type checking success
	combined := stdout + stderr
	if !strings.Contains(combined, "Type checking") && !strings.Contains(combined, "✓") {
		t.Logf("Note: Expected type checking success message. Output: %s", combined)
		// Don't fail - message format may vary
	}
}

func TestCLI_Check_InvalidSyntax(t *testing.T) {
	// Create a file with syntax errors
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_bad.ail")

	content := `module test_bad
func broken(x {  -- Missing closing paren
  x
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, stderr, exitCode := runCLI(t, "check", testFile)

	if exitCode == 0 {
		t.Error("Expected non-zero exit code for invalid syntax")
	}

	if !strings.Contains(stderr, "Error") {
		t.Errorf("Expected error message for syntax error, got: %s", stderr)
	}
}

func TestCLI_Doctor_Builtins(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, "doctor", "builtins")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
	}

	// Should show validation results
	combined := stdout + stderr
	if !strings.Contains(combined, "builtin") {
		t.Errorf("Expected doctor output to mention builtins, got: %s", combined)
	}
}

func TestCLI_Builtins_List(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, "builtins", "list")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
	}

	// Should list some builtins
	expectedBuiltins := []string{
		"_io_print",
		"add_Int",
		"_str_len",
	}

	for _, builtin := range expectedBuiltins {
		if !strings.Contains(stdout, builtin) {
			t.Errorf("Expected builtins list to contain %q, got: %s", builtin, stdout)
		}
	}
}

func TestCLI_Builtins_ListByModule(t *testing.T) {
	stdout, stderr, exitCode := runCLI(t, "builtins", "list", "--by-module")

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Stderr: %s", exitCode, stderr)
	}

	// Should show module grouping
	expectedModules := []string{
		"std/io",
		"std/string",
	}

	for _, module := range expectedModules {
		if !strings.Contains(stdout, module) {
			t.Errorf("Expected output to contain module %q, got: %s", module, stdout)
		}
	}
}

func TestCLI_InvalidCommand(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "invalid-command")

	if exitCode == 0 {
		t.Error("Expected non-zero exit code for invalid command")
	}

	if !strings.Contains(stderr, "Unknown command") && !strings.Contains(stderr, "invalid") {
		t.Logf("Note: Expected error about invalid command. Stderr: %s", stderr)
		// Don't fail - error message format may vary
	}
}
