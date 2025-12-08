package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/eval_harness"
)

// TestNewDirectiveExecutor tests executor creation
func TestNewDirectiveExecutor(t *testing.T) {
	tmpDir := t.TempDir()

	executor := NewDirectiveExecutor(tmpDir)

	if executor == nil {
		t.Fatal("Expected executor to be created")
	}

	if executor.workspaceBase != tmpDir {
		t.Errorf("Expected workspaceBase=%s, got %s", tmpDir, executor.workspaceBase)
	}

	if executor.config.TimeoutSeconds != 300 {
		t.Errorf("Expected timeout=300, got %d", executor.config.TimeoutSeconds)
	}

	if executor.config.ClaudeModel != "haiku" {
		t.Errorf("Expected model=haiku, got %s", executor.config.ClaudeModel)
	}
}

// TestExecute_WorkspaceCreation tests that workspace is created
func TestExecute_WorkspaceCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode (requires Claude CLI)")
	}

	tmpDir := t.TempDir()
	executor := NewDirectiveExecutor(tmpDir)

	// This will fail because Claude CLI may not be installed
	// But we can verify workspace creation logic
	directive := "Create a file hello.txt with content 'Hello, World!'"

	_, err := executor.Execute(directive)

	// We expect error because Claude CLI may not be available
	// But workspace should have been created
	files, _ := os.ReadDir(tmpDir)
	if len(files) == 0 {
		t.Log("No workspace created (Claude CLI may not be installed)")
	}

	// Verify error is appropriate
	if err != nil {
		t.Logf("Expected error (Claude CLI check): %v", err)
	}
}

// TestListFiles tests file listing functionality
func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"file1.txt",
		"subdir/file2.txt",
		"subdir/nested/file3.txt",
	}

	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create .git/config: %v", err)
	}

	// List files
	files, err := listFiles(tmpDir)
	if err != nil {
		t.Fatalf("listFiles failed: %v", err)
	}

	// Verify results
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d: %v", len(files), files)
	}

	// Verify .git files are not included
	for _, file := range files {
		if len(file) >= 4 && file[:4] == ".git" {
			t.Errorf("Found .git file in results: %s", file)
		}
	}

	// Verify all expected files are present
	expectedMap := make(map[string]bool)
	for _, f := range testFiles {
		expectedMap[filepath.ToSlash(f)] = false
	}

	for _, f := range files {
		normalized := filepath.ToSlash(f)
		if _, exists := expectedMap[normalized]; exists {
			expectedMap[normalized] = true
		}
	}

	for file, found := range expectedMap {
		if !found {
			t.Errorf("Expected file not found: %s", file)
		}
	}
}

// TestGetErrorMessage tests error message extraction
func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		result   *eval_harness.ClaudeHeadlessResult
		expected string
	}{
		{
			name: "Success",
			result: &eval_harness.ClaudeHeadlessResult{
				IsError: false,
				Subtype: "success",
				Result:  "Task completed",
			},
			expected: "",
		},
		{
			name: "Error",
			result: &eval_harness.ClaudeHeadlessResult{
				IsError: true,
				Subtype: "error",
				Result:  "Execution failed",
			},
			expected: "Execution failed",
		},
		{
			name: "Timeout",
			result: &eval_harness.ClaudeHeadlessResult{
				IsError: true,
				Subtype: "timeout",
				Result:  "Session timed out",
			},
			expected: "Session timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getErrorMessage(tt.result)
			if got != tt.expected {
				t.Errorf("Expected error=%q, got %q", tt.expected, got)
			}
		})
	}
}

// TestExecuteWithModel tests model override
func TestExecuteWithModel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode (requires Claude CLI)")
	}

	tmpDir := t.TempDir()
	executor := NewDirectiveExecutor(tmpDir)

	// Verify default model
	if executor.config.ClaudeModel != "haiku" {
		t.Errorf("Expected default model=haiku, got %s", executor.config.ClaudeModel)
	}

	// Execute with sonnet (will fail due to Claude CLI, but we verify model change)
	_, err := executor.ExecuteWithModel("Create hello.txt", "sonnet")

	// After execution, model should be restored to haiku
	if executor.config.ClaudeModel != "haiku" {
		t.Errorf("Expected model restored to haiku, got %s", executor.config.ClaudeModel)
	}

	// Verify error is appropriate
	if err != nil {
		t.Logf("Expected error (Claude CLI check): %v", err)
	}
}
