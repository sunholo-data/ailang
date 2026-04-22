package eval_harness

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPythonRunner(t *testing.T) {
	// Skip if Python is not available (e.g., Windows CI)
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err := exec.LookPath("python"); err != nil {
			t.Skip("python not available, skipping")
		}
	}

	runner := NewPythonRunner()

	if runner.Language() != "python" {
		t.Errorf("Expected language 'python', got '%s'", runner.Language())
	}

	// Test simple print
	code := `print("Hello, World!")`
	result, err := runner.Run(code, 5*time.Second)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !result.CompileOk {
		t.Error("Expected CompileOk to be true")
	}

	if !result.RuntimeOk {
		t.Error("Expected RuntimeOk to be true")
	}

	stdout := strings.TrimSpace(result.Stdout)
	if stdout != "Hello, World!" {
		t.Errorf("Expected stdout 'Hello, World!', got '%s'", stdout)
	}
}

func TestPythonRunner_Error(t *testing.T) {
	runner := NewPythonRunner()

	// Test syntax error
	code := `print("unclosed string`
	result, err := runner.Run(code, 5*time.Second)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}

	if result.RuntimeOk {
		t.Error("Expected RuntimeOk to be false for syntax error")
	}

	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestPythonRunner_Timeout(t *testing.T) {
	runner := NewPythonRunner()

	// Test timeout (sleep for longer than timeout)
	code := `import time; time.sleep(10)`
	result, err := runner.Run(code, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}

	if !result.TimedOut {
		t.Error("Expected TimedOut to be true")
	}

	if result.RuntimeOk {
		t.Error("Expected RuntimeOk to be false after timeout")
	}
}

func TestCompareOutput(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		want     bool
	}{
		{"exact match", "hello", "hello", true},
		{"with whitespace", "hello\n", "hello", true},
		{"mismatch", "hello", "goodbye", false},
		{"empty", "", "", true},
		{"multiline match", "line1\nline2", "line1\nline2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareOutput(tt.expected, tt.actual)
			if result != tt.want {
				t.Errorf("CompareOutput() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestAILANGRunnerValidation tests that the validation re-run works correctly
// for agent mode. This was a bug (Feb 2026): the validation runner created its
// workspace outside /tmp/, so module path auto-relaxation didn't apply, causing
// false negatives. Fix: add --relax-modules to runner args.
func TestAILANGRunnerValidation(t *testing.T) {
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not in PATH, skipping integration test")
	}

	// Test with a simple solution that doesn't need stdlib imports
	// (stdlib path depends on being run from project root)
	code := `module benchmark/solution

type Option[a] = Some(a) | None

export func main() -> () ! {IO} {
  let x: Option[int] = Some(42);
  match x {
    Some(v) => println("Got: ${show(v)}"),
    None => println("Nothing")
  };
  let y: Option[int] = None;
  match y {
    Some(v) => println("Got: ${show(v)}"),
    None => println("Empty")
  }
}
`
	spec := &BenchmarkSpec{
		ID:          "adt_validation_test",
		Caps:        []string{"IO"},
		ExpectedOut: "Got: 42\nEmpty\n",
	}

	// Use short timeout — trivial programs should compile+run in <2s
	runner := NewAILANGRunnerWithTask(t.Context(), "", spec.Caps, "", spec)
	runResult, err := runner.Run(code, 3*time.Second)
	if err != nil {
		t.Fatalf("runner error: %v", err)
	}

	if !runResult.CompileOk {
		t.Errorf("Expected CompileOk=true, got false. Stderr: %s", runResult.Stderr)
	}
	if !runResult.RuntimeOk {
		t.Errorf("Expected RuntimeOk=true, got false. Stderr: %s", runResult.Stderr)
	}
	stdoutOk := runResult.RuntimeOk && CompareOutput(spec.ExpectedOut, runResult.Stdout)
	if !stdoutOk {
		t.Errorf("Expected StdoutOk=true, got false. Stdout: %q, Expected: %q", runResult.Stdout, spec.ExpectedOut)
	}
}

// TestAILANGRunnerValidation_MismatchedModule tests that the validation runner
// handles module path mismatches gracefully (agent may write different module names).
func TestAILANGRunnerValidation_MismatchedModule(t *testing.T) {
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not in PATH, skipping integration test")
	}

	// Agent might write "module solution" instead of "module benchmark/solution"
	code := `module solution

export func main() -> () ! {IO} {
  println("hello")
}
`
	spec := &BenchmarkSpec{
		ID:          "test_module_mismatch",
		Caps:        []string{"IO"},
		ExpectedOut: "hello\n",
	}

	// Use short timeout — trivial programs should compile+run in <2s
	runner := NewAILANGRunnerWithTask(t.Context(), "", spec.Caps, "", spec)
	runResult, err := runner.Run(code, 3*time.Second)
	if err != nil {
		t.Fatalf("runner error: %v", err)
	}

	// With --relax-modules, this should still compile and run
	if !runResult.CompileOk {
		t.Errorf("Expected CompileOk=true with --relax-modules, got false. Stderr: %s", runResult.Stderr)
	}
	if !runResult.RuntimeOk {
		t.Errorf("Expected RuntimeOk=true with --relax-modules, got false. Stderr: %s", runResult.Stderr)
	}
	stdoutOk := runResult.RuntimeOk && CompareOutput(spec.ExpectedOut, runResult.Stdout)
	if !stdoutOk {
		t.Errorf("Expected StdoutOk=true, got false. Stdout: %q", runResult.Stdout)
	}
}

func TestGetRunner(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:     "test",
		Caps:   []string{"IO"},
		Prompt: "test",
	}

	tests := []struct {
		lang      string
		expectErr bool
	}{
		{"python", false},
		{"ailang", false},
		{"javascript", false},
		{"go", false},
		{"typescript", true},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			runner, err := GetRunner(tt.lang, spec)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if runner == nil {
					t.Error("Expected runner, got nil")
				}
			}
		})
	}
}

func TestLimitedWriter_WithinLimit(t *testing.T) {
	lw := NewLimitedWriter(100)

	// Write data within limit
	data := []byte("Hello, World!")
	n, err := lw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	output := lw.String()
	if output != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", output)
	}

	if lw.Truncated() {
		t.Error("Expected Truncated() to be false")
	}
}

func TestLimitedWriter_ExceedsLimit(t *testing.T) {
	lw := NewLimitedWriter(10)

	// Write data that exceeds limit
	data := []byte("This is a very long string that exceeds the limit")
	n, err := lw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	// Should report full length to avoid errors
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	output := lw.String()
	// Should contain first 10 bytes + truncation message
	if !strings.HasPrefix(output, "This is a ") {
		t.Errorf("Expected output to start with 'This is a ', got '%s'", output)
	}
	if !strings.Contains(output, "[OUTPUT TRUNCATED") {
		t.Errorf("Expected truncation message in output, got '%s'", output)
	}

	if !lw.Truncated() {
		t.Error("Expected Truncated() to be true")
	}
}

func TestLimitedWriter_MultipleWrites(t *testing.T) {
	lw := NewLimitedWriter(20)

	// Multiple writes within limit
	lw.Write([]byte("Hello"))
	lw.Write([]byte(" "))
	lw.Write([]byte("World"))

	output := lw.String()
	if output != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", output)
	}

	if lw.Truncated() {
		t.Error("Expected Truncated() to be false")
	}
}

func TestLimitedWriter_MultipleWritesExceedLimit(t *testing.T) {
	lw := NewLimitedWriter(10)

	// Multiple writes that exceed limit
	lw.Write([]byte("Hello"))
	lw.Write([]byte(" World"))
	lw.Write([]byte(" Extra"))

	output := lw.String()
	// Should contain first 10 bytes + truncation message
	if !strings.HasPrefix(output, "Hello Worl") {
		t.Errorf("Expected output to start with 'Hello Worl', got '%s'", output)
	}
	if !strings.Contains(output, "[OUTPUT TRUNCATED") {
		t.Errorf("Expected truncation message in output, got '%s'", output)
	}

	if !lw.Truncated() {
		t.Error("Expected Truncated() to be true")
	}
}

func TestJSRunner(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available, skipping")
	}

	r := NewJSRunner()
	if r.Language() != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", r.Language())
	}

	result, err := r.Run(`console.log("Hello, World!")`, 10*time.Second)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !result.CompileOk {
		t.Error("Expected CompileOk=true")
	}
	if !result.RuntimeOk {
		t.Errorf("Expected RuntimeOk=true, stderr: %s", result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %q", result.Stdout)
	}
}

func TestJSRunner_Error(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available, skipping")
	}

	r := NewJSRunner()
	result, err := r.Run(`console.log(undefined_var_xyz)`, 10*time.Second)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}
	if result.RuntimeOk {
		t.Error("Expected RuntimeOk=false for reference error")
	}
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestJSRunner_Timeout(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available, skipping")
	}

	r := NewJSRunner()
	result, err := r.Run(`while(true){}`, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}
	if !result.TimedOut {
		t.Error("Expected TimedOut=true")
	}
}

func TestGoRunner(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available, skipping")
	}

	r := NewGoRunner()
	if r.Language() != "go" {
		t.Errorf("Expected language 'go', got '%s'", r.Language())
	}

	code := `import "fmt"
func main() { fmt.Println("Hello, World!") }`
	result, err := r.Run(code, 30*time.Second)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !result.CompileOk {
		t.Errorf("Expected CompileOk=true, stderr: %s", result.Stderr)
	}
	if !result.RuntimeOk {
		t.Errorf("Expected RuntimeOk=true, stderr: %s", result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %q", result.Stdout)
	}
}

func TestGoRunner_PackageMain(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available, skipping")
	}

	r := NewGoRunner()
	// Code with package main already present
	code := `package main
import "fmt"
func main() { fmt.Println(42) }`
	result, err := r.Run(code, 30*time.Second)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "42" {
		t.Errorf("Expected '42', got %q", result.Stdout)
	}
}

func TestGoRunner_CompileError(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available, skipping")
	}

	r := NewGoRunner()
	result, err := r.Run(`func main() { invalid syntax here }`, 30*time.Second)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}
	if result.RuntimeOk {
		t.Error("Expected RuntimeOk=false for compile error")
	}
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestLimitedWriter_WriteAfterTruncation(t *testing.T) {
	lw := NewLimitedWriter(5)

	// First write exceeds limit
	lw.Write([]byte("Hello World"))

	// Subsequent writes should be discarded
	originalOutput := lw.String()
	lw.Write([]byte("More data"))
	newOutput := lw.String()

	if originalOutput != newOutput {
		t.Error("Expected subsequent writes after truncation to be discarded")
	}

	if !lw.Truncated() {
		t.Error("Expected Truncated() to be true")
	}
}
