package eval_harness

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BenchmarkSpec defines a single benchmark task
type BenchmarkSpec struct {
	ID           string            `yaml:"id"`
	Description  string            `yaml:"description"`
	Languages    []string          `yaml:"languages"`
	Entrypoint   string            `yaml:"entrypoint"`
	Caps         []string          `yaml:"caps"`
	Prompt       string            `yaml:"prompt"`       // Inline prompt text (language-agnostic)
	PromptFiles  map[string]string `yaml:"prompt_files"` // Language-specific prompt files: {ailang: "prompts/v0.3.0.md"}
	TaskPrompt   string            `yaml:"task_prompt"`  // Task-specific prompt appended after base prompt
	ExpectedOut  string            `yaml:"expected_stdout"`
	Difficulty   string            `yaml:"difficulty"`
	ExpectedGain string            `yaml:"expected_gain"`
}

// LoadSpec loads a benchmark spec from a YAML file
func LoadSpec(path string) (*BenchmarkSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var spec BenchmarkSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if spec.ID == "" {
		return nil, fmt.Errorf("spec missing required field: id")
	}
	if len(spec.Languages) == 0 {
		return nil, fmt.Errorf("spec missing required field: languages")
	}

	// No backward compatibility - benchmarks must use prompt_files

	// Note: We don't load prompts here anymore - they're loaded per-language in PromptForLanguage()
	// This allows each language to have its own base prompt file

	return &spec, nil
}

// SupportsLanguage checks if the benchmark supports a given language
func (s *BenchmarkSpec) SupportsLanguage(lang string) bool {
	for _, l := range s.Languages {
		if l == lang {
			return true
		}
	}
	return false
}

// PromptForLanguage returns the prompt with language-specific base prompt + task prompt
func (s *BenchmarkSpec) PromptForLanguage(lang string) string {
	var basePrompt string

	// Load language-specific prompt file if available
	if s.PromptFiles != nil {
		if promptFile, ok := s.PromptFiles[lang]; ok {
			data, err := os.ReadFile(promptFile)
			if err == nil {
				basePrompt = string(data)
			}
			// If file not found, fall back to inline prompt or default
		}
	}

	// If no language-specific prompt file, use inline prompt or default
	if basePrompt == "" {
		if s.Prompt != "" {
			basePrompt = s.Prompt
		} else {
			// Default minimal prompt for languages without specific guidance
			basePrompt = getDefaultPrompt(lang)
		}
	}

	// Append task-specific prompt
	fullPrompt := basePrompt
	if s.TaskPrompt != "" {
		fullPrompt = fullPrompt + "\n\n## Task\n\n" + s.TaskPrompt
	}

	// Normalize language names for <LANG> placeholder
	langName := lang
	switch lang {
	case "python":
		langName = "Python 3"
	case "ailang":
		langName = "AILANG"
	}

	// Replace <LANG> placeholder
	return replaceAll(fullPrompt, "<LANG>", langName)
}

// getDefaultPrompt returns a minimal default prompt for a language
func getDefaultPrompt(lang string) string {
	switch lang {
	case "python":
		return "You are an expert Python programmer. Write clean, idiomatic Python code."
	case "ailang":
		return "You are writing code in AILANG, a functional programming language."
	default:
		return "Write clean, idiomatic code in the specified language."
	}
}

// replaceAll is a simple string replacement function
func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := findSubstring(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// findSubstring finds the index of the first occurrence of substr in s
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
