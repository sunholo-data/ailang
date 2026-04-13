package eval_harness

import (
	"fmt"
	"os"

	promptpkg "github.com/sunholo/ailang/internal/prompt"
	"gopkg.in/yaml.v3"
)

// BenchmarkSpec defines a single benchmark task
type BenchmarkSpec struct {
	ID           string            `yaml:"id"`
	Description  string            `yaml:"description"`
	Languages    []string          `yaml:"languages"`
	Entrypoint   string            `yaml:"entrypoint"`
	Caps         []string          `yaml:"caps"`
	Prompt       string            `yaml:"prompt"`        // Inline prompt text (language-agnostic)
	PromptFiles  map[string]string `yaml:"prompt_files"`  // Language-specific prompt files: {ailang: "prompts/v0.3.0.md"}
	TaskPrompt   string            `yaml:"task_prompt"`   // Task-specific prompt appended after base prompt
	ContractSpec string            `yaml:"contract_spec"` // Optional: AILANG contract specification for Z3 verification
	Z3Hints      string            `yaml:"z3_hints"`      // Optional: Pre-computed Z3 counterexample descriptions for known traps
	ExpectedOut  string            `yaml:"expected_stdout"`
	Difficulty   string            `yaml:"difficulty"`
	ExpectedGain string            `yaml:"expected_gain"`
	Timeout      int               `yaml:"timeout"` // Agent timeout in seconds (default: 60)

	// Test infrastructure: stdin, CLI args, and input files
	Stdin      string            `yaml:"stdin,omitempty"`       // Stdin data to pipe to the program
	CliArgs    []string          `yaml:"cli_args,omitempty"`    // CLI arguments to pass after the script
	InputFiles map[string]string `yaml:"input_files,omitempty"` // Files to create in workspace: {filename: content}
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
	var taskDescription string

	// For AILANG: ALWAYS use teaching prompt as base, treat s.Prompt as task description
	// This ensures the model always has AILANG syntax reference
	if lang == "ailang" {
		// Always load the teaching prompt for AILANG
		basePrompt = getDefaultPrompt("ailang")

		// The inline prompt field is the task description for AILANG
		if s.TaskPrompt != "" {
			taskDescription = s.TaskPrompt
		} else if s.Prompt != "" {
			taskDescription = s.Prompt
		}
	} else {
		// For other languages (e.g. Python): use language guidelines as base,
		// and the benchmark's prompt field as the task description.
		// This mirrors the AILANG path: base_prompt + "## Task" + task.

		// Load language-specific prompt file if available
		if s.PromptFiles != nil {
			if promptFile, ok := s.PromptFiles[lang]; ok {
				data, err := os.ReadFile(promptFile)
				if err == nil {
					basePrompt = string(data)
				}
			}
		}

		// If no language-specific prompt file, use default guidelines
		if basePrompt == "" {
			basePrompt = getDefaultPrompt(lang)
		}

		// The inline prompt field is the task description (same as AILANG path)
		if s.TaskPrompt != "" {
			taskDescription = s.TaskPrompt
		} else if s.Prompt != "" {
			taskDescription = s.Prompt
		}
	}

	// Build full prompt
	fullPrompt := basePrompt
	if taskDescription != "" {
		fullPrompt = fullPrompt + "\n\n## Task\n\n" + taskDescription
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
		// For AILANG, use the active prompt from the central prompt package
		// Single source of truth: internal/prompt (also used by `ailang prompt` CLI)
		activePrompt, err := promptpkg.LoadPrompt("")
		if err == nil {
			return activePrompt
		}
		// Fallback if prompt loader fails
		return "You are writing code in AILANG, a functional programming language."
	default:
		return "Write clean, idiomatic code in the specified language."
	}
}

// FormatContractSpec returns a formatted contract specification block for prompt injection.
// When verify is true and the spec has a ContractSpec, returns a formatted block.
// Otherwise returns empty string (backward compatible).
func (s *BenchmarkSpec) FormatContractSpec(verify bool) string {
	if !verify || s.ContractSpec == "" {
		return ""
	}
	return fmt.Sprintf(`FORMAL SPECIFICATION (your solution MUST satisfy these contracts):
`+"```ailang"+`
%s
`+"```"+`
Run `+"`ailang ai-check solution.ail`"+` to verify your solution against these contracts.`, s.ContractSpec)
}

// FormatZ3Hints returns a formatted Z3 hints block for prompt injection.
// Only returns content when the spec has Z3Hints defined.
func (s *BenchmarkSpec) FormatZ3Hints() string {
	if s.Z3Hints == "" {
		return ""
	}
	return fmt.Sprintf(`KNOWN EDGE CASES (from formal verification analysis):

%s

These describe traps that a naive implementation would fall into.
Design your solution to handle these cases correctly from the start.`, s.Z3Hints)
}

// EvalCondition represents a named experimental condition that controls
// what information is included in the LLM prompt. Conditions are treated
// like languages — each creates a separate evaluation job.
type EvalCondition struct {
	Name                string // "baseline", "contract", "z3_guided", "full", "tool_aware", "agent_prompt", or "" for legacy
	IncludeContract     bool   // Include contract_spec in prompt
	IncludeZ3Hints      bool   // Include z3_hints in prompt
	IncludeDevtools     bool   // Append devtools prompt to system prompt
	IncludeToolGuidance bool   // Include general contract-writing + ai-check guidance (no spec given)
	EnableVerify        bool   // Enable Z3 verification (standard mode repair + post-hoc check)
	UseAgentPrompt      bool   // Use compact agent coding prompt instead of full teaching prompt
}

// ValidConditionNames lists all recognized condition names
var ValidConditionNames = []string{"baseline", "contract", "z3_guided", "full", "tool_aware", "agent_prompt"}

// ResolveCondition returns the settings for a named condition.
// If name is empty, returns legacy behavior using the explicit --verify/--devtools-prompt flags.
func ResolveCondition(name string, legacyVerify, legacyDevtools bool) EvalCondition {
	switch name {
	case "baseline":
		return EvalCondition{Name: "baseline"}
	case "contract":
		return EvalCondition{
			Name:            "contract",
			IncludeContract: true,
			EnableVerify:    true,
		}
	case "z3_guided":
		return EvalCondition{
			Name:            "z3_guided",
			IncludeContract: true,
			IncludeZ3Hints:  true,
			EnableVerify:    true,
		}
	case "full":
		return EvalCondition{
			Name:            "full",
			IncludeContract: true,
			IncludeZ3Hints:  true,
			IncludeDevtools: true,
			EnableVerify:    true,
		}
	case "tool_aware":
		return EvalCondition{
			Name:                "tool_aware",
			IncludeToolGuidance: true,
			EnableVerify:        true,
		}
	case "agent_prompt":
		// Uses compact agent coding prompt (~180 lines) instead of full teaching prompt (~1600 lines)
		return EvalCondition{
			Name:           "agent_prompt",
			UseAgentPrompt: true,
		}
	default:
		// Legacy mode: use explicit flag values
		return EvalCondition{
			Name:            "",
			IncludeContract: legacyVerify,
			EnableVerify:    legacyVerify,
			IncludeDevtools: legacyDevtools,
		}
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
