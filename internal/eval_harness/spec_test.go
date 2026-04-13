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

	// Test Python - should use guidelines as base + task appended
	pythonResult := spec.PromptForLanguage("python")

	// Python result should contain the default guidelines
	if !containsSubstring(pythonResult, "expert Python programmer") {
		t.Errorf("PromptForLanguage(python) should contain Python guidelines, got: %s", pythonResult[:min(100, len(pythonResult))])
	}

	// Python result should contain the task with <LANG> replaced
	if !containsSubstring(pythonResult, "## Task") {
		t.Errorf("PromptForLanguage(python) should contain '## Task' section")
	}
	if !containsSubstring(pythonResult, "prints hello") {
		t.Errorf("PromptForLanguage(python) should contain task description 'prints hello'")
	}
	if !containsSubstring(pythonResult, "Python 3") {
		t.Errorf("PromptForLanguage(python) should replace <LANG> with 'Python 3'")
	}

	// Test AILANG - should always include teaching prompt with task appended
	ailangResult := spec.PromptForLanguage("ailang")

	// For AILANG, the result should:
	// 1. Start with the teaching prompt (contains "AILANG v0.6")
	// 2. Have the task description appended after "## Task"
	if len(ailangResult) < 100 {
		t.Errorf("PromptForLanguage(ailang) result too short, expected teaching prompt + task")
	}

	// Check that teaching prompt is included (starts with # AILANG)
	if ailangResult[:9] != "# AILANG " {
		t.Errorf("PromptForLanguage(ailang) should start with teaching prompt header, got: %s", ailangResult[:50])
	}

	// Check that task description is appended
	if !containsSubstring(ailangResult, "## Task") {
		t.Errorf("PromptForLanguage(ailang) should contain '## Task' section")
	}
	if !containsSubstring(ailangResult, "prints hello") {
		t.Errorf("PromptForLanguage(ailang) should contain task description 'prints hello'")
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	return findSubstring(s, substr) != -1
}
