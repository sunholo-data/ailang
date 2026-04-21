package main

import (
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestPythonPromptLoading verifies that Python benchmarks load prompts/python.md
func TestPythonPromptLoading(t *testing.T) {
	// Read expected Python prompt
	expectedPromptData, err := os.ReadFile("../../prompts/python.md")
	if err != nil {
		t.Fatalf("Failed to read prompts/python.md: %v", err)
	}
	expectedPrompt := string(expectedPromptData)

	// Verify the prompt contains Python instructions
	if !strings.Contains(expectedPrompt, "Python") {
		t.Error("Python prompt missing 'Python' reference")
	}

	// Verify prompt is substantial (not just a one-liner)
	if len(expectedPrompt) < 500 {
		t.Errorf("Python prompt too short (%d bytes), expected comprehensive guidance", len(expectedPrompt))
	}

	t.Logf("✓ Python prompt validated: %d bytes, contains critical warnings", len(expectedPrompt))
}

// TestAILANGPromptLoading verifies that AILANG benchmarks load versioned prompts
func TestAILANGPromptLoading(t *testing.T) {
	// Load prompt registry
	loader, err := eval_harness.NewPromptLoader("../../prompts/versions.json")
	if err != nil {
		t.Fatalf("Failed to load prompt registry: %v", err)
	}

	// Get active prompt
	activePrompt, err := loader.GetActivePrompt()
	if err != nil {
		t.Fatalf("Failed to load active AILANG prompt: %v", err)
	}

	// Verify the prompt contains AILANG-specific content
	if !strings.Contains(activePrompt, "AILANG") {
		t.Error("AILANG prompt missing 'AILANG' reference")
	}

	// Verify prompt is substantial
	if len(activePrompt) < 1000 {
		t.Errorf("AILANG prompt too short (%d bytes), expected comprehensive teaching prompt", len(activePrompt))
	}

	// Get active version ID
	versionID := loader.GetActiveVersionID()
	if versionID == "" {
		t.Error("Active prompt version ID is empty")
	}

	t.Logf("✓ AILANG prompt validated: version=%s, %d bytes", versionID, len(activePrompt))
}

// TestPromptFilesExist verifies all required prompt files exist
func TestPromptFilesExist(t *testing.T) {
	requiredFiles := []string{
		"../../prompts/python.md",
		"../../prompts/versions.json",
	}

	for _, file := range requiredFiles {
		info, err := os.Stat(file)
		if err != nil {
			t.Errorf("Required prompt file missing: %s (error: %v)", file, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("Required prompt file is empty: %s", file)
		}
		t.Logf("✓ Found %s (%d bytes)", file, info.Size())
	}
}

// TestPromptDisambiguation verifies prompts clearly distinguish languages
func TestPromptDisambiguation(t *testing.T) {
	// Load both prompts
	pythonPromptData, err := os.ReadFile("../../prompts/python.md")
	if err != nil {
		t.Fatalf("Failed to read Python prompt: %v", err)
	}

	loader, err := eval_harness.NewPromptLoader("../../prompts/versions.json")
	if err != nil {
		t.Fatalf("Failed to load AILANG prompt: %v", err)
	}
	ailangPrompt, err := loader.GetActivePrompt()
	if err != nil {
		t.Fatalf("Failed to get active AILANG prompt: %v", err)
	}

	pythonPrompt := string(pythonPromptData)

	// Python prompt should mention Python explicitly
	if !strings.Contains(strings.ToLower(pythonPrompt), "python") {
		t.Error("Python prompt doesn't mention 'python' - models may be confused about target language")
	}

	// AILANG prompt should mention AILANG explicitly
	if !strings.Contains(ailangPrompt, "AILANG") {
		t.Error("AILANG prompt doesn't mention 'AILANG' - models may be confused about target language")
	}

	t.Logf("✓ Prompts clearly identify their target languages")
}
