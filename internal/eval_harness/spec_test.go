package eval_harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpec(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "test.yml")

	content := `id: test
description: "Test benchmark"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "easy"
expected_gain: "low"
prompt: "Write a program in <LANG>"
expected_stdout: "Hello"
`

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load spec
	spec, err := LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}

	// Verify fields
	if spec.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", spec.ID)
	}

	if len(spec.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(spec.Languages))
	}

	if spec.Prompt != "Write a program in <LANG>" {
		t.Errorf("Unexpected prompt: %s", spec.Prompt)
	}
}

func TestLoadSpec_MissingRequired(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "invalid.yml")

	// Missing 'id' field
	content := `description: "Test"
languages: ["python"]
prompt: "Test"
`

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadSpec(specPath)
	if err == nil {
		t.Error("Expected error for missing required field, got nil")
	}
}

func TestSupportsLanguage(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:        "test",
		Languages: []string{"python", "ailang"},
		Prompt:    "test",
	}

	tests := []struct {
		lang     string
		expected bool
	}{
		{"python", true},
		{"ailang", true},
		{"javascript", false},
		{"", false},
	}

	for _, tt := range tests {
		result := spec.SupportsLanguage(tt.lang)
		if result != tt.expected {
			t.Errorf("SupportsLanguage(%s) = %v, want %v", tt.lang, result, tt.expected)
		}
	}
}

func TestPromptForLanguage(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:     "test",
		Prompt: "Write code in <LANG> that prints hello",
	}

	tests := []struct {
		lang     string
		expected string
	}{
		{"python", "Write code in Python 3 that prints hello"},
		{"ailang", "Write code in AILANG that prints hello"},
	}

	for _, tt := range tests {
		result := spec.PromptForLanguage(tt.lang)
		if result != tt.expected {
			t.Errorf("PromptForLanguage(%s) = %s, want %s", tt.lang, result, tt.expected)
		}
	}
}
