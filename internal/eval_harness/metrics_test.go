package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name      string
		compileOk bool
		runtimeOk bool
		stdoutOk  bool
		expected  string
	}{
		{"all ok", true, true, true, ErrorCategoryNone},
		{"compile failed", false, true, true, ErrorCategoryCompile},
		{"runtime failed", true, false, true, ErrorCategoryRuntime},
		{"output wrong", true, true, false, ErrorCategoryLogic},
		{"compile and runtime failed", false, false, false, ErrorCategoryCompile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeError(tt.compileOk, tt.runtimeOk, tt.stdoutOk)
			if result != tt.expected {
				t.Errorf("CategorizeError() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		model    string
		tokens   int
		expected float64
	}{
		{"gpt-4", 1000, 0.03},
		{"gpt-3.5-turbo", 1000, 0.001},
		{"claude-3", 1000, 0.015},
		{"unknown-model", 1000, 0.03}, // Defaults to gpt-4
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := CalculateCost(tt.model, tt.tokens)
			if result != tt.expected {
				t.Errorf("CalculateCost(%s, %d) = %f, want %f", tt.model, tt.tokens, result, tt.expected)
			}
		})
	}
}

func TestMetricsLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewMetricsLogger(tmpDir)

	metrics := &RunMetrics{
		ID:            "test",
		Lang:          "python",
		Model:         "gpt-4",
		Seed:          42,
		Tokens:        100,
		CostUSD:       0.003,
		CompileOk:     true,
		RuntimeOk:     true,
		StdoutOk:      true,
		DurationMs:    150,
		ErrorCategory: ErrorCategoryNone,
		Timestamp:     time.Now(),
	}

	// Log metrics
	if err := logger.Log(metrics); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.json"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	// Read and parse file
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var loaded RunMetrics
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify fields
	if loaded.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", loaded.ID)
	}

	if loaded.Tokens != 100 {
		t.Errorf("Expected tokens 100, got %d", loaded.Tokens)
	}
}

func TestNewRunMetrics(t *testing.T) {
	metrics := NewRunMetrics("test", "python", "gpt-4", 42)

	if metrics.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", metrics.ID)
	}

	if metrics.Lang != "python" {
		t.Errorf("Expected lang 'python', got '%s'", metrics.Lang)
	}

	if metrics.Seed != 42 {
		t.Errorf("Expected seed 42, got %d", metrics.Seed)
	}

	// Timestamp should be recent (within 1 second)
	if time.Since(metrics.Timestamp) > time.Second {
		t.Error("Timestamp is not recent")
	}
}
