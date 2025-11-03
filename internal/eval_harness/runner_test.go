package eval_harness

import (
	"strings"
	"testing"
	"time"
)

func TestPythonRunner(t *testing.T) {
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
		{"javascript", true},
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
