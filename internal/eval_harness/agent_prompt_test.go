package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The teaching prompt the live agent path sends as its system prompt.
func TestLoadSystemPromptForLanguage(t *testing.T) {
	// Test AILANG
	syntaxRef, _, err := LoadSystemPromptForLanguage("ailang", "")
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
	pythonRef, _, err := LoadSystemPromptForLanguage("python", "")
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

// TestLanguagePlaceholderReplacement verifies that <LANG> is replaced correctly
func TestLanguagePlaceholderReplacement(t *testing.T) {
	// Create temporary directory structure with test prompts
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	templatesDir := filepath.Join(tmpDir, "internal/eval_harness/templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Python prompt file
	pythonPrompt := "# Python Guidelines\nWrite clean Python code."
	pythonPath := filepath.Join(promptsDir, "python.md")
	if err := os.WriteFile(pythonPath, []byte(pythonPrompt), 0644); err != nil {
		t.Fatal(err)
	}

	// Create AILANG prompt file
	ailangPrompt := "# AILANG Guidelines\nWrite functional AILANG code."
	ailangPath := filepath.Join(promptsDir, "ailang.md")
	if err := os.WriteFile(ailangPath, []byte(ailangPrompt), 0644); err != nil {
		t.Fatal(err)
	}

	// Create task templates
	pythonTemplate := "Python task: {{DESCRIPTION}}\nExpected: {{EXPECTED_OUTPUT}}"
	pythonTemplatePath := filepath.Join(templatesDir, "agent_task_python.txt")
	if err := os.WriteFile(pythonTemplatePath, []byte(pythonTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	ailangTemplate := "AILANG task: {{DESCRIPTION}}\nExpected: {{EXPECTED_OUTPUT}}"
	ailangTemplatePath := filepath.Join(templatesDir, "agent_task_ailang.txt")
	if err := os.WriteFile(ailangTemplatePath, []byte(ailangTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	// Create registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"python":  {File: "prompts/python.md", Hash: "PLACEHOLDER"},
			"ailang1": {File: "prompts/ailang.md", Hash: "PLACEHOLDER"},
		},
		Active: "ailang1",
	}
	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Change working directory to tmpDir for the test
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test ADT implementation",
		TaskPrompt:  "Write a program in <LANG> that implements an Option type.",
		ExpectedOut: "Result: 5.0",
		Caps:        []string{"IO"},
	}

	config := DefaultAgentConfig()

	// Test Python
	_, taskPromptPython, _, err := GenerateAgentPromptsWithSystemPrompt(spec, config, "python", "", "solution.py")
	if err != nil {
		t.Fatalf("GenerateAgentPromptsWithSystemPrompt failed for Python: %v", err)
	}

	// Verify <LANG> was replaced with "Python" (capitalized)
	if strings.Contains(taskPromptPython, "<LANG>") {
		t.Error("Python task prompt still contains <LANG> placeholder")
	}
	if !strings.Contains(taskPromptPython, "Write a program in Python") {
		t.Error("Python task prompt doesn't contain 'Write a program in Python'")
	}

	// Test AILANG
	_, taskPromptAILANG, _, err := GenerateAgentPromptsWithSystemPrompt(spec, config, "ailang", "", "benchmark/solution.ail")
	if err != nil {
		t.Fatalf("GenerateAgentPromptsWithSystemPrompt failed for AILANG: %v", err)
	}

	// Verify <LANG> was replaced with "AILANG"
	if strings.Contains(taskPromptAILANG, "<LANG>") {
		t.Error("AILANG task prompt still contains <LANG> placeholder")
	}
	if !strings.Contains(taskPromptAILANG, "Write a program in AILANG") {
		t.Error("AILANG task prompt doesn't contain 'Write a program in AILANG'")
	}

	// M-AGENT-PROMPT-MODE: the agent-mode output-delivery override is ON by
	// default for agent runs; opt OUT with AILANG_AGENT_OUTPUT_DELIVERY=0.
	t.Setenv("AILANG_AGENT_OUTPUT_DELIVERY", "0")
	sysOff, _, _, err := GenerateAgentPromptsWithSystemPrompt(spec, config, "ailang", "", "benchmark/solution.ail")
	if err != nil {
		t.Fatalf("gen (delivery off): %v", err)
	}
	if strings.Contains(sysOff, "Output delivery (AGENT MODE)") {
		t.Error("agent-mode override present when AILANG_AGENT_OUTPUT_DELIVERY=0")
	}
	t.Setenv("AILANG_AGENT_OUTPUT_DELIVERY", "") // unset → default ON
	sysOn, _, _, err := GenerateAgentPromptsWithSystemPrompt(spec, config, "ailang", "", "benchmark/solution.ail")
	if err != nil {
		t.Fatalf("gen (delivery default): %v", err)
	}
	if !strings.Contains(sysOn, "Output delivery (AGENT MODE)") || !strings.Contains(sysOn, "file-write tool") {
		t.Error("agent-mode override missing by default (should be ON)")
	}
}

// TestSystemPromptLanguageSeparation verifies Python gets Python prompt, AILANG gets AILANG prompt
func TestSystemPromptLanguageSeparation(t *testing.T) {
	// Create temporary directory structure with test prompts
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.Mkdir(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	templatesDir := filepath.Join(tmpDir, "internal/eval_harness/templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Python prompt file (should NOT contain AILANG-specific content)
	pythonPrompt := "# Python Programming Guidelines\n\nYou are an expert Python programmer. Use PEP 8 style."
	pythonPath := filepath.Join(promptsDir, "python.md")
	if err := os.WriteFile(pythonPath, []byte(pythonPrompt), 0644); err != nil {
		t.Fatal(err)
	}

	// Create AILANG prompt file (should contain AILANG-specific content)
	ailangPrompt := "# AILANG Programming Guidelines\n\nUse func main() for entry points. Run with ailang run."
	ailangPath := filepath.Join(promptsDir, "ailang.md")
	if err := os.WriteFile(ailangPath, []byte(ailangPrompt), 0644); err != nil {
		t.Fatal(err)
	}

	// Create task templates
	pythonTemplate := "Python task: {{DESCRIPTION}}"
	pythonTemplatePath := filepath.Join(templatesDir, "agent_task_python.txt")
	if err := os.WriteFile(pythonTemplatePath, []byte(pythonTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	ailangTemplate := "AILANG task: {{DESCRIPTION}}"
	ailangTemplatePath := filepath.Join(templatesDir, "agent_task_ailang.txt")
	if err := os.WriteFile(ailangTemplatePath, []byte(ailangTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	// Create registry
	registry := PromptRegistry{
		SchemaVersion: "1.0",
		Versions: map[string]PromptVersion{
			"python":  {File: "prompts/python.md", Hash: "PLACEHOLDER"},
			"v0.3.23": {File: "prompts/ailang.md", Hash: "PLACEHOLDER"},
		},
		Active: "v0.3.23",
	}
	registryPath := filepath.Join(promptsDir, "versions.json")
	data, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Change working directory to tmpDir for the test
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	spec := &BenchmarkSpec{
		ID:          "test_benchmark",
		Description: "Test",
		TaskPrompt:  "Write a program",
		ExpectedOut: "42",
		Caps:        []string{"IO"},
	}

	config := DefaultAgentConfig()

	// Test Python gets Python system prompt
	systemPromptPython, _, promptVersion, err := GenerateAgentPromptsWithSystemPrompt(spec, config, "python", "", "solution.py")
	if err != nil {
		t.Fatalf("GenerateAgentPromptsWithSystemPrompt failed for Python: %v", err)
	}

	if promptVersion != "python" {
		t.Errorf("Python prompt version incorrect: got %q, want %q", promptVersion, "python")
	}

	// Python system prompt should contain Python-specific content
	if !strings.Contains(systemPromptPython, "Python") {
		t.Error("Python system prompt doesn't contain 'Python'")
	}

	// Python system prompt should NOT contain AILANG-specific keywords
	if strings.Contains(systemPromptPython, "ailang run") || strings.Contains(systemPromptPython, "func main()") {
		t.Error("Python system prompt contains AILANG-specific content")
	}

	// Test AILANG gets AILANG system prompt
	systemPromptAILANG, _, promptVersionAILANG, err := GenerateAgentPromptsWithSystemPrompt(spec, config, "ailang", "", "benchmark/solution.ail")
	if err != nil {
		t.Fatalf("GenerateAgentPromptsWithSystemPrompt failed for AILANG: %v", err)
	}

	// AILANG should use the active version from registry
	if promptVersionAILANG != "v0.3.23" {
		t.Errorf("AILANG prompt version incorrect: got %q, want %q", promptVersionAILANG, "v0.3.23")
	}

	// AILANG system prompt should contain AILANG-specific content
	if !strings.Contains(systemPromptAILANG, "AILANG") && !strings.Contains(systemPromptAILANG, "ailang") {
		t.Error("AILANG system prompt doesn't contain 'AILANG' or 'ailang'")
	}
}

// A non-active prompt version requested for an agent run must be the one loaded
// (and reported), not silently the active prompt.
func TestGenerateAgentPrompts_UsesRequestedPromptVersion(t *testing.T) {
	spec := &BenchmarkSpec{ID: "pv", TaskPrompt: "Write a program in <LANG> that prints 1"}
	cfg := DefaultAgentConfig()
	_, _, used, err := GenerateAgentPromptsWithSystemPrompt(spec, cfg, "ailang", "v0.16.10", "benchmark/solution.ail")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if used != "v0.16.10" {
		t.Fatalf("prompt version used = %q, want v0.16.10", used)
	}
}
