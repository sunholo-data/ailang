package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAgentPrompt(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test description",
		TaskPrompt:  "Write a function that returns 42",
		ExpectedOut: "42",
		Caps:        []string{"IO", "FS"},
	}

	config := DefaultAgentConfig()
	syntaxRef := "Test syntax reference"

	// Test AILANG prompt
	promptAILANG := GenerateAgentPrompt(spec, config, syntaxRef, "ailang")

	expectedSections := []string{
		"Workspace Files",
		"Your Task",
		"Success Criteria",
		"Constraints",
		"Tools Available",
		"Tips",
		"README.md",
		"solution.ail",
		"syntax_reference.md",
		"ailang check",
		"ailang run",
		"IO,FS",
	}

	for _, section := range expectedSections {
		if !strings.Contains(promptAILANG, section) {
			t.Errorf("AILANG prompt missing section: %s", section)
		}
	}

	if !strings.Contains(promptAILANG, "300 seconds") {
		t.Error("AILANG prompt does not include timeout")
	}

	// Test Python prompt
	promptPython := GenerateAgentPrompt(spec, config, syntaxRef, "python")

	pythonSections := []string{
		"Python benchmark",
		"solution.py",
		"python3 solution.py",
	}

	for _, section := range pythonSections {
		if !strings.Contains(promptPython, section) {
			t.Errorf("Python prompt missing section: %s", section)
		}
	}
}

func TestLoadActiveSyntaxReference(t *testing.T) {
	// Test AILANG
	syntaxRef, err := LoadActiveSyntaxReference("ailang")
	if err != nil {
		t.Skipf("Cannot load AILANG syntax reference (prompts/ may not be accessible): %v", err)
	}

	// Check it's not empty
	if len(syntaxRef) == 0 {
		t.Error("AILANG syntax reference is empty")
	}

	// Check it looks like AILANG documentation
	keywords := []string{"AILANG", "func", "let", "match"}
	foundKeywords := 0
	for _, keyword := range keywords {
		if strings.Contains(syntaxRef, keyword) {
			foundKeywords++
		}
	}

	if foundKeywords < 2 {
		t.Errorf("AILANG syntax reference doesn't look like AILANG docs (found %d/%d keywords)", foundKeywords, len(keywords))
	}

	// Test Python
	pythonRef, err := LoadActiveSyntaxReference("python")
	if err != nil {
		t.Skipf("Cannot load Python syntax reference (prompts/python.md may not be accessible): %v", err)
	}

	if len(pythonRef) == 0 {
		t.Error("Python syntax reference is empty")
	}

	// Check it looks like Python documentation
	if !strings.Contains(pythonRef, "Python") && !strings.Contains(pythonRef, "def") {
		t.Error("Python syntax reference doesn't look like Python docs")
	}
}

func TestPrepareWorkspaceWithSyntax(t *testing.T) {
	tmpDir := t.TempDir()

	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test description",
		TaskPrompt:  "Write a function that returns 42",
		ExpectedOut: "42",
		Caps:        []string{"IO"},
	}

	syntaxRef := "# AILANG Syntax\n\nTest syntax content"

	err := PrepareWorkspaceWithSyntax(tmpDir, spec, syntaxRef)
	if err != nil {
		t.Fatalf("PrepareWorkspaceWithSyntax failed: %v", err)
	}

	// Check README.md
	readmePath := filepath.Join(tmpDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md was not created")
	}

	readmeContent, _ := os.ReadFile(readmePath)
	readmeStr := string(readmeContent)
	if !strings.Contains(readmeStr, "test_benchmark") {
		t.Error("README.md does not contain benchmark ID")
	}
	if !strings.Contains(readmeStr, "Test description") {
		t.Error("README.md does not contain description")
	}
	if !strings.Contains(readmeStr, "Write a function that returns 42") {
		t.Error("README.md does not contain task prompt")
	}

	// Check solution.ail
	solutionPath := filepath.Join(tmpDir, "solution.ail")
	if _, err := os.Stat(solutionPath); os.IsNotExist(err) {
		t.Error("solution.ail was not created")
	}

	solutionContent, _ := os.ReadFile(solutionPath)
	solutionStr := string(solutionContent)
	if !strings.Contains(solutionStr, "test_benchmark") {
		t.Error("solution.ail does not contain benchmark ID in comment")
	}
	if !strings.Contains(solutionStr, "TODO") {
		t.Error("solution.ail does not contain TODO marker")
	}
	if !strings.Contains(solutionStr, "func main()") {
		t.Error("solution.ail does not show example main function")
	}

	// Check syntax_reference.md
	syntaxPath := filepath.Join(tmpDir, "syntax_reference.md")
	if _, err := os.Stat(syntaxPath); os.IsNotExist(err) {
		t.Error("syntax_reference.md was not created")
	}

	syntaxContent, _ := os.ReadFile(syntaxPath)
	syntaxStr := string(syntaxContent)
	if syntaxStr != syntaxRef {
		t.Errorf("syntax_reference.md content mismatch.\nExpected: %s\nGot: %s", syntaxRef, syntaxStr)
	}
}

func TestGetDefaultSyntaxReference(t *testing.T) {
	syntaxRef := getDefaultSyntaxReference()

	// Check it contains key AILANG syntax elements
	expectedElements := []string{
		"func",
		"let",
		"match",
		"Effects",
		"print",
		"show",
	}

	for _, elem := range expectedElements {
		if !strings.Contains(syntaxRef, elem) {
			t.Errorf("Default syntax reference missing: %s", elem)
		}
	}
}

func TestEnhancedGenerateAgentPrompt(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test description",
		TaskPrompt:  "Write a function that returns 42",
		ExpectedOut: "42",
		Caps:        []string{"IO"},
	}

	config := DefaultAgentConfig()

	// Test with AILANG
	prompt, syntaxRef, err := EnhancedGenerateAgentPrompt(spec, config, "ailang")
	if err != nil {
		// This is okay - it means prompts/ directory may not be accessible
		// The function should fall back to default syntax reference
		t.Logf("EnhancedGenerateAgentPrompt returned error (expected if prompts/ not accessible): %v", err)
	}

	// Check prompt is not empty
	if len(prompt) == 0 {
		t.Error("Enhanced prompt is empty")
	}

	// Check syntax reference is not empty
	if len(syntaxRef) == 0 {
		t.Error("Syntax reference is empty")
	}

	// Check syntax ref contains AILANG content
	if !strings.Contains(syntaxRef, "AILANG") && !strings.Contains(syntaxRef, "func") {
		t.Error("Syntax reference doesn't look like AILANG documentation")
	}

	// Test with Python (may fall back to default if prompts/python.md not accessible)
	pythonPrompt, pythonRef, err := EnhancedGenerateAgentPrompt(spec, config, "python")
	if err != nil {
		t.Logf("Python prompt returned error (expected if prompts/python.md not accessible): %v", err)
	}

	// Just verify we got some content (may be default fallback)
	if len(pythonPrompt) == 0 {
		t.Error("Python prompt is empty")
	}
	if len(pythonRef) == 0 {
		t.Error("Python syntax reference is empty")
	}

	// Log what we got for debugging
	t.Logf("Python ref preview: %s...", pythonRef[:min(100, len(pythonRef))])
}

func TestPrepareWorkspaceWithEmptySyntax(t *testing.T) {
	tmpDir := t.TempDir()

	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test",
		TaskPrompt:  "Test task",
		ExpectedOut: "42",
		Caps:        []string{},
	}

	// Test with empty syntax ref - should use default
	err := PrepareWorkspaceWithSyntax(tmpDir, spec, "")
	if err != nil {
		t.Fatalf("PrepareWorkspaceWithSyntax failed: %v", err)
	}

	// Check syntax_reference.md was created with default content
	syntaxPath := filepath.Join(tmpDir, "syntax_reference.md")
	syntaxContent, err := os.ReadFile(syntaxPath)
	if err != nil {
		t.Fatalf("Failed to read syntax_reference.md: %v", err)
	}

	syntaxStr := string(syntaxContent)
	if !strings.Contains(syntaxStr, "Minimal") {
		t.Error("Default syntax reference was not used")
	}
}
