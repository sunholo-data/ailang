package eval_harness

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BenchmarkSpec defines a single benchmark task
type BenchmarkSpec struct {
	ID           string   `yaml:"id"`
	Description  string   `yaml:"description"`
	Languages    []string `yaml:"languages"`
	Entrypoint   string   `yaml:"entrypoint"`
	Caps         []string `yaml:"caps"`
	Prompt       string   `yaml:"prompt"`      // Inline prompt text
	PromptFile   string   `yaml:"prompt_file"` // Path to prompt file (relative to repo root)
	TaskPrompt   string   `yaml:"task_prompt"` // Task-specific prompt appended after base prompt
	ExpectedOut  string   `yaml:"expected_stdout"`
	Difficulty   string   `yaml:"difficulty"`
	ExpectedGain string   `yaml:"expected_gain"`
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

	// Load prompt from file if specified
	if spec.PromptFile != "" {
		promptData, err := os.ReadFile(spec.PromptFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file %s: %w", spec.PromptFile, err)
		}
		spec.Prompt = string(promptData)
	}

	// Append task-specific prompt if specified
	if spec.TaskPrompt != "" {
		if spec.Prompt != "" {
			spec.Prompt = spec.Prompt + "\n\n" + spec.TaskPrompt
		} else {
			spec.Prompt = spec.TaskPrompt
		}
	}

	// Now validate that we have a prompt
	if spec.Prompt == "" {
		return nil, fmt.Errorf("spec missing prompt (must specify either 'prompt', 'prompt_file', or 'task_prompt')")
	}

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

// PromptForLanguage returns the prompt with <LANG> replaced by the target language
func (s *BenchmarkSpec) PromptForLanguage(lang string) string {
	// Normalize language names
	langName := lang
	switch lang {
	case "python":
		langName = "Python 3"
	case "ailang":
		langName = "AILANG"
	}

	// Simple string replacement
	return replaceAll(s.Prompt, "<LANG>", langName)
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
