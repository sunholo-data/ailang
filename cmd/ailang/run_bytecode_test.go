package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_RunBytecode_Arithmetic verifies that `ailang run --bytecode`
// compiles a simple .ail file through the bytecode pipeline and prints
// the VM result. This is the M2 acceptance test for M-BYTECODE-2D.
func TestCLI_RunBytecode_Arithmetic(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "bcrun_arith.ail")
	if err := os.WriteFile(src, []byte("module test/bcrun_arith\n\nexport func main() -> int = 1 + 2 * 3\n"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	stdout, stderr, exitCode := runCLI(t, "run", "--bytecode", "--relax-modules", src)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr=%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "7") {
		t.Errorf("expected stdout to contain VM result 7, got %q\nstderr=%s", stdout, stderr)
	}
	// The "Running ... via bytecode VM" status line goes to stderr.
	if !strings.Contains(stderr, "via bytecode VM") {
		t.Errorf("expected stderr to mention bytecode VM run, got %q", stderr)
	}
}

// TestCLI_RunBytecode_FallbackOnEffect verifies that an effectful program
// (which the bytecode compiler can't handle yet) falls back to the
// evaluator transparently when --strict-bytecode is NOT set.
func TestCLI_RunBytecode_FallbackOnEffect(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "bcrun_fb.ail")
	srcContent := `module test/bcrun_fb

import std/io (println)

export func main() -> () ! {IO} = println("hello from evaluator")
`
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	stdout, stderr, exitCode := runCLI(t, "run", "--bytecode", "--caps", "IO", "--relax-modules", src)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (transparent fallback), got %d\nstderr=%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "hello from evaluator") {
		t.Errorf("expected evaluator to print fallback output, got stdout=%q\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "falling back to evaluator") {
		t.Errorf("expected fallback warning on stderr, got %q", stderr)
	}
}

// TestCLI_RunBytecode_DivByZero_ReportsLine verifies that a runtime error in
// the bytecode VM surfaces the source file:line of the offending statement
// (M-BYTECODE-2D milestone M4_LINEINFO).
func TestCLI_RunBytecode_DivByZero_ReportsLine(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "divzero.ail")
	// Line 1: module
	// Line 2: blank
	// Line 3: export func main() -> int = 10 / 0
	srcContent := "module test/divzero\n\nexport func main() -> int = 10 / 0\n"
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	_, stderr, exitCode := runCLI(t, "run", "--bytecode", "--strict-bytecode", "--relax-modules", src)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit (divide by zero), got 0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "division by zero") {
		t.Errorf("expected stderr to mention divide by zero, got %q", stderr)
	}
	// The runtime error should carry a file:line marker. Don't pin the exact
	// line — the lower pass may attribute it to the func decl rather than the
	// expression — but require *some* file:line shaped token from this file.
	if !strings.Contains(stderr, "divzero.ail:") {
		t.Errorf("expected stderr to contain source file:line, got %q", stderr)
	}
}

// TestCLI_RunBytecode_StrictFails verifies --strict-bytecode exits non-zero
// instead of falling back when the program can't be compiled.
func TestCLI_RunBytecode_StrictFails(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "bcrun_strict.ail")
	srcContent := `module test/bcrun_strict

import std/io (println)

export func main() -> () ! {IO} = println("nope")
`
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	_, stderr, exitCode := runCLI(t, "run", "--bytecode", "--strict-bytecode", "--caps", "IO", "--relax-modules", src)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "bytecode execution failed") {
		t.Errorf("expected strict-bytecode failure message, got %q", stderr)
	}
}
