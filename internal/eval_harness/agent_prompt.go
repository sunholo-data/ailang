package eval_harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateAgentPrompt creates a comprehensive prompt for the agent
// This version loads from a language-specific template file for easy editing
func GenerateAgentPrompt(spec *BenchmarkSpec, config AgentBenchmarkConfig, syntaxRef string, language string) string {
	// Determine template path based on language
	var templatePath string
	switch language {
	case "python":
		templatePath = "internal/eval_harness/templates/agent_prompt_python.txt"
	case "ailang":
		templatePath = "internal/eval_harness/templates/agent_prompt.txt"
	default:
		templatePath = "internal/eval_harness/templates/agent_prompt.txt"
	}

	// Try to load template from file
	templateBytes, err := os.ReadFile(templatePath)

	var template string
	if err != nil {
		// Fallback to hardcoded template if file not found
		template = getDefaultPromptTemplate(language)
	} else {
		template = string(templateBytes)
	}

	// Replace placeholders
	template = strings.ReplaceAll(template, "{{CAPS}}", strings.Join(spec.Caps, ","))
	template = strings.ReplaceAll(template, "{{TIMEOUT}}", fmt.Sprintf("%d", config.TimeoutSeconds))
	template = strings.ReplaceAll(template, "{{MAX_ITERATIONS}}", fmt.Sprintf("%d", config.MaxIterations))

	return template
}

// getDefaultPromptTemplate returns a fallback template if file not found
func getDefaultPromptTemplate(language string) string {
	// Return language-specific default
	if language == "python" {
		return getDefaultPythonTemplate()
	}
	return getDefaultAILANGTemplate()
}

// getDefaultAILANGTemplate returns the AILANG template
func getDefaultAILANGTemplate() string {
	return `You are solving an AILANG benchmark in an isolated workspace.

## Workspace Files

- **README.md**: Problem description and expected output
- **solution.ail**: Your implementation (currently empty - you will write this)
- **syntax_reference.md**: Complete AILANG syntax reference

## Your Task

1. Read README.md to understand the problem and expected output
2. Read syntax_reference.md for AILANG syntax (important!)
3. Write your solution in solution.ail
4. Type-check: Run 'ailang check solution.ail'
5. Test: Run 'ailang run --entry main --caps {{CAPS}} solution.ail'
6. Compare output with expected output from README.md
7. If output doesn't match, iterate and fix
8. Repeat steps 4-7 until output matches exactly

## Success Criteria

✓ solution.ail compiles without errors
✓ solution.ail runs without runtime errors
✓ Output matches expected output exactly (whitespace is trimmed)

## Constraints

- Timeout: {{TIMEOUT}} seconds
- Max iterations: {{MAX_ITERATIONS}}
- Solution must be in solution.ail (not inline or in comments)

## Tools Available

You have access to:
- Bash: Run ailang commands, check files
- Read: Read README.md, syntax_reference.md
- Write: Create solution.ail
- Edit: Modify solution.ail
- Grep: Search for patterns in files

## Tips

- Start simple: Get basic structure working first
- Use ailang check frequently to catch type errors early
- If you get a type error, read the error message carefully
- AILANG is functional - no mutable variables, use recursion or higher-order functions
- All effects must be declared in function signatures (e.g., ! {IO})
- Pattern matching is exhaustive - cover all cases

Good luck!
`
}

// LoadActiveSyntaxReference loads the active teaching prompt for a language
func LoadActiveSyntaxReference(language string) (string, error) {
	// Handle different languages
	switch language {
	case "ailang":
		return loadAILANGPrompt()
	case "python":
		return loadPythonPrompt()
	default:
		// Default to AILANG for unknown languages
		return loadAILANGPrompt()
	}
}

// loadAILANGPrompt loads the active AILANG teaching prompt
func loadAILANGPrompt() (string, error) {
	// Load versions.json to find active prompt
	versionsPath := "prompts/versions.json"

	// Use the PromptLoader to get active prompt
	loader, err := NewPromptLoader(versionsPath)
	if err != nil {
		return "", fmt.Errorf("failed to create prompt loader: %w", err)
	}

	activePrompt, err := loader.GetActivePrompt()
	if err != nil {
		return "", fmt.Errorf("failed to get active prompt: %w", err)
	}

	return activePrompt, nil
}

// loadPythonPrompt loads the Python teaching prompt
func loadPythonPrompt() (string, error) {
	// Python prompt is in prompts/ (same location as AILANG prompts)
	pythonPromptPath := "prompts/python.md"
	data, err := os.ReadFile(pythonPromptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read python prompt: %w", err)
	}

	return string(data), nil
}

// PrepareWorkspaceWithSyntax creates workspace files with full AILANG syntax reference
func PrepareWorkspaceWithSyntax(workspace string, spec *BenchmarkSpec, syntaxRef string) error {
	// Create README.md with problem description
	readme := fmt.Sprintf(`# %s

%s

## Task

%s

## Expected Output

%s

## Notes

- Your implementation should go in solution.ail
- Run 'ailang check solution.ail' to type-check
- Run 'ailang run --entry main --caps %s solution.ail' to test
- See syntax_reference.md for complete AILANG syntax
`, spec.ID, spec.Description, spec.TaskPrompt, spec.ExpectedOut, strings.Join(spec.Caps, ","))

	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Create empty solution.ail stub with helpful comment
	solutionStub := fmt.Sprintf(`// %s
//
// TODO: Implement your solution here
//
// Capabilities required: %s
//
// Quick start:
// 1. Define your main function: func main(): <return-type> ! {%s} = ...
// 2. Use pattern matching for complex logic: match expr { ... }
// 3. Use recursion for loops
// 4. Run 'ailang check solution.ail' to verify syntax
// 5. Run 'ailang run --entry main --caps %s solution.ail' to test

`, spec.ID, strings.Join(spec.Caps, ","), strings.Join(spec.Caps, ","), strings.Join(spec.Caps, ","))

	if err := os.WriteFile(filepath.Join(workspace, "solution.ail"), []byte(solutionStub), 0644); err != nil {
		return fmt.Errorf("failed to write solution.ail: %w", err)
	}

	// Create syntax_reference.md with full AILANG teaching prompt
	if syntaxRef == "" {
		syntaxRef = getDefaultSyntaxReference()
	}

	if err := os.WriteFile(filepath.Join(workspace, "syntax_reference.md"), []byte(syntaxRef), 0644); err != nil {
		return fmt.Errorf("failed to write syntax_reference.md: %w", err)
	}

	return nil
}

// getDefaultSyntaxReference returns a minimal syntax reference if full prompt unavailable
func getDefaultSyntaxReference() string {
	return `# AILANG Syntax Reference (Minimal)

## Basic Syntax

**Functions:**
func name(x: int): int = x + 1

**Let bindings:**
let x = 42 in x + 1

**Lambdas:**
\x. x + 1

**Pattern matching:**
match expr {
  Some(x) => x,
  None => 0
}

**Effects:**
func read_file(path: string): string ! {IO, FS} = ...

**Blocks:**
{
  let x = 1;
  let y = 2;
  x + y
}

## Common Builtins

- print(s: string): unit ! {IO}
- show(x: a): string (converts any value to string)
- intToFloat(i: int): float
- floatToInt(f: float): int

## Tips

- AILANG is pure functional - no mutable state
- Use recursion instead of loops
- All effects must be declared in function signatures
- Pattern matching must be exhaustive

For complete documentation, see the full AILANG teaching prompt.
`
}

// EnhancedGenerateAgentPrompt is a wrapper that loads syntax and generates prompt
// language parameter determines which teaching prompt to load (ailang, python, etc.)
func EnhancedGenerateAgentPrompt(spec *BenchmarkSpec, config AgentBenchmarkConfig, language string) (string, string, error) {
	// Load syntax reference for the specified language
	syntaxRef, err := LoadActiveSyntaxReference(language)
	if err != nil {
		// Fall back to minimal reference
		syntaxRef = getDefaultSyntaxReference()
	}

	// Generate prompt
	prompt := GenerateAgentPrompt(spec, config, syntaxRef, language)

	return prompt, syntaxRef, nil
}

// getDefaultPythonTemplate returns the Python template
func getDefaultPythonTemplate() string {
	return `You are solving a Python benchmark in an isolated workspace.

## Workspace Files

- **README.md**: Problem description and expected output
- **solution.py**: Your implementation (currently empty - you will write this)
- **syntax_reference.md**: Python language reference

## Your Task

1. Read README.md to understand the problem and expected output
2. Read syntax_reference.md for Python syntax guidance
3. Write your solution in solution.py
4. Test: Run 'python3 solution.py'
5. Compare output with expected output from README.md
6. If output doesn't match, iterate and fix
7. Repeat steps 4-6 until output matches exactly

## Success Criteria

✓ solution.py runs without errors
✓ Output matches expected output exactly (whitespace is trimmed)

## Constraints

- Timeout: {{TIMEOUT}} seconds
- Max iterations: {{MAX_ITERATIONS}}
- Solution must be in solution.py (not inline or in comments)

## Tools Available

You have access to:
- Bash: Run python3 commands, check files
- Read: Read README.md, syntax_reference.md
- Write: Create solution.py
- Edit: Modify solution.py
- Grep: Search for patterns in files

## Tips

- Start simple: Get basic structure working first
- Test frequently with 'python3 solution.py'
- Use Python 3 standard library - no external packages needed
- Follow PEP 8 style guidelines
- Use type hints for clarity
- Handle edge cases explicitly

Good luck!
`
}
