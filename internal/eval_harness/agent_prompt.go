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
	template = strings.ReplaceAll(template, "{{PYTHON_VERSION}}", DetectedPythonVersion())

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

// loadPythonPrompt loads the Python teaching prompt and substitutes the
// runtime version placeholder so the prompt advertises the same Python
// version that the grader will actually invoke.
func loadPythonPrompt() (string, error) {
	// Python prompt is in prompts/ (same location as AILANG prompts)
	pythonPromptPath := "prompts/python.md"
	data, err := os.ReadFile(pythonPromptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read python prompt: %w", err)
	}

	return strings.ReplaceAll(string(data), "{{PYTHON_VERSION}}", DetectedPythonVersion()), nil
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
3. Write your solution in solution.py (target Python {{PYTHON_VERSION}})
4. Test: Run 'uv run --python {{PYTHON_VERSION}} solution.py'
5. Compare output with expected output from README.md
6. If output doesn't match, iterate and fix
7. Repeat steps 4-6 until output matches exactly

## Success Criteria

✓ solution.py runs without errors
✓ Output matches expected output exactly (whitespace is trimmed)

## Constraints

- Timeout: {{TIMEOUT}} seconds
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
- Test frequently with 'uv run --python {{PYTHON_VERSION}} solution.py'
- Runtime is pinned to **Python {{PYTHON_VERSION}}** via uv — features up to {{PYTHON_VERSION}} (match/case, unions, PEP 695 type statement) are fair game; anything newer will SyntaxError
- Use the Python standard library - no external packages needed
- Follow PEP 8 style guidelines
- Use type hints for clarity
- Handle edge cases explicitly

Good luck!
`
}

// LoadSystemPromptForLanguage loads the versioned teaching prompt for a language
// This is used with Claude CLI's --system-prompt flag
func LoadSystemPromptForLanguage(language string, promptVersion string) (string, string, error) {
	versionsPath := "prompts/versions.json"
	loader, err := NewPromptLoader(versionsPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create prompt loader: %w", err)
	}

	var prompt string
	var versionUsed string

	switch language {
	case "ailang":
		// Use specified version or active version
		if promptVersion != "" {
			prompt, err = loader.LoadPrompt(promptVersion)
			if err != nil {
				return "", "", fmt.Errorf("failed to load AILANG prompt version %s: %w", promptVersion, err)
			}
			versionUsed = promptVersion
		} else {
			prompt, err = loader.GetActivePrompt()
			if err != nil {
				return "", "", fmt.Errorf("failed to load active AILANG prompt: %w", err)
			}
			// Get active version ID from registry
			if loader.registry != nil && loader.registry.Active != "" {
				versionUsed = loader.registry.Active
			} else {
				versionUsed = "active"
			}
		}

	case "python":
		// Python uses "python" entry from versions.json
		prompt, err = loader.LoadPrompt("python")
		if err != nil {
			return "", "", fmt.Errorf("failed to load Python prompt: %w", err)
		}
		versionUsed = "python"

	default:
		return "", "", fmt.Errorf("unsupported language: %s", language)
	}

	return prompt, versionUsed, nil
}

// LoadTaskPromptTemplate loads the generic .txt template for the initial agent prompt
// This explains the benchmark task (what to solve), not the language syntax
func LoadTaskPromptTemplate(language string) (string, error) {
	var templatePath string
	switch language {
	case "python":
		templatePath = "internal/eval_harness/templates/agent_task_python.txt"
	case "ailang":
		templatePath = "internal/eval_harness/templates/agent_task_ailang.txt"
	default:
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		// Fallback to hardcoded template if file not found
		return getDefaultTaskTemplate(language), nil
	}

	return string(data), nil
}

// getDefaultTaskTemplate returns a fallback task template
func getDefaultTaskTemplate(language string) string {
	if language == "python" {
		return `You are solving a Python benchmark.

## Task

{{DESCRIPTION}}

**IMPORTANT: Write your complete solution to: {{SOLUTION_PATH}}**

Expected output:
{{EXPECTED_OUTPUT}}

## Constraints

- Timeout: {{TIMEOUT}} seconds
- Write your solution to: **{{SOLUTION_PATH}}**
- Your solution must produce output matching the expected output exactly

Good luck!`
	}

	return `You are solving an AILANG benchmark.

## Task

{{DESCRIPTION}}

**IMPORTANT: Write your complete solution to: {{SOLUTION_PATH}}**

Expected output:
{{EXPECTED_OUTPUT}}

## Constraints

- Timeout: {{TIMEOUT}} seconds
- Write your solution to: **{{SOLUTION_PATH}}**
- Run your solution with: ailang run --entry main --caps {{CAPS}} solution.ail
- Your solution must produce output matching the expected output exactly

## Verification (REQUIRED)

**Before finishing, you MUST verify your solution works correctly:**

1. **Run your solution:** ` + "`ailang run --entry main --caps {{CAPS}} solution.ail`" + `
2. **Check the output matches expected output exactly** (no extra newlines, spaces, etc.)
3. **If output doesn't match, fix and re-run until it matches**
4. **Only finish once you've confirmed output is correct**

**Available tools:** You have access to ` + "`ailang`" + ` command for:
- Running code: ` + "`ailang run`" + `
- Type checking: ` + "`ailang check`" + `
- Testing: ` + "`ailang test`" + `

Good luck!`
}

// GenerateAgentPromptsWithSystemPrompt generates split prompts for --system-prompt flag
// Returns: (systemPrompt, taskPrompt, promptVersionUsed, error)
func GenerateAgentPromptsWithSystemPrompt(spec *BenchmarkSpec, config AgentBenchmarkConfig, language string, promptVersion string, solutionPath string) (string, string, string, error) {
	// Load system prompt (language knowledge)
	// If UseAgentPrompt condition is active and we have agent prompt content, use that instead
	var systemPrompt string
	var versionUsed string
	var err error

	cond := config.Condition
	if cond.UseAgentPrompt && config.AgentPromptContent != "" && language == "ailang" {
		systemPrompt = config.AgentPromptContent
		versionUsed = "agent-prompt"
	} else {
		systemPrompt, versionUsed, err = LoadSystemPromptForLanguage(language, promptVersion)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to load system prompt: %w", err)
		}
	}

	// Load task prompt template
	taskTemplate, err := LoadTaskPromptTemplate(language)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to load task template: %w", err)
	}

	// Replace placeholders in task prompt
	taskPrompt := strings.ReplaceAll(taskTemplate, "{{DESCRIPTION}}", spec.Description+"\n\n"+spec.TaskPrompt)
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{EXPECTED_OUTPUT}}", spec.ExpectedOut)
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{CAPS}}", strings.Join(spec.Caps, ","))
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{TIMEOUT}}", fmt.Sprintf("%d", config.TimeoutSeconds))
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{SOLUTION_PATH}}", solutionPath)
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{PYTHON_VERSION}}", DetectedPythonVersion())

	// Resolve condition: use explicit condition if set, otherwise fall back to legacy Verify flag
	// cond was already set to config.Condition at function start
	if cond.Name == "" {
		cond = ResolveCondition("", config.Verify, config.DevtoolsPrompt != "")
	}

	// Replace {{CONTRACT_SPEC}} placeholder based on condition (M-CONTRACT-EVAL)
	contractSpecBlock := spec.FormatContractSpec(cond.IncludeContract)
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{CONTRACT_SPEC}}", contractSpecBlock)

	// Replace {{Z3_HINTS}} placeholder based on condition
	var z3HintsBlock string
	if cond.IncludeZ3Hints {
		z3HintsBlock = spec.FormatZ3Hints()
	}
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{Z3_HINTS}}", z3HintsBlock)

	// Replace {{TOOL_GUIDANCE}} placeholder based on condition
	var toolGuidanceBlock string
	if cond.IncludeToolGuidance {
		toolGuidanceBlock = getToolAwareGuidance()
	}
	taskPrompt = strings.ReplaceAll(taskPrompt, "{{TOOL_GUIDANCE}}", toolGuidanceBlock)

	// Replace {{VERIFICATION_STEPS}} and {{EXTRA_TOOLS}} based on condition
	if cond.IncludeToolGuidance {
		taskPrompt = strings.ReplaceAll(taskPrompt, "{{VERIFICATION_STEPS}}", getToolAwareVerificationSteps(strings.Join(spec.Caps, ",")))
		taskPrompt = strings.ReplaceAll(taskPrompt, "{{EXTRA_TOOLS}}", "- **Verify contracts:** `ailang ai-check solution.ail` (Z3 finds counterexamples in your pure functions)")
	} else {
		taskPrompt = strings.ReplaceAll(taskPrompt, "{{VERIFICATION_STEPS}}", getDefaultVerificationSteps(strings.Join(spec.Caps, ",")))
		taskPrompt = strings.ReplaceAll(taskPrompt, "{{EXTRA_TOOLS}}", "")
	}

	// Replace <LANG> placeholder with actual language name (used in 31/35 benchmarks)
	// e.g., "Write a program in <LANG>" → "Write a program in Python"
	languageName := language
	if language == "python" {
		languageName = "Python"
	} else if language == "ailang" {
		languageName = "AILANG"
	}
	taskPrompt = strings.ReplaceAll(taskPrompt, "<LANG>", languageName)

	// Append devtools prompt to system prompt based on condition
	if cond.IncludeDevtools && config.DevtoolsPrompt != "" {
		systemPrompt = systemPrompt + "\n\n" + config.DevtoolsPrompt
	}

	return systemPrompt, taskPrompt, versionUsed, nil
}

// getToolAwareGuidance returns general guidance about using AILANG contracts
// and Z3 verification as a debugging tool. This does NOT include any
// benchmark-specific contract specs — the LLM must write its own contracts.
func getToolAwareGuidance() string {
	return `## Debugging with Contracts (recommended)

AILANG has built-in formal verification. You can write contracts on pure functions
and use Z3 to automatically find bugs — this is often faster than manual debugging.

**How it works:**

1. Add ` + "`requires`" + ` (preconditions) and ` + "`ensures`" + ` (postconditions) to your pure functions:
` + "```" + `
export pure func safeSub(a: int, b: int) -> int
  requires { a >= 0, b >= 0 }
  ensures { result >= 0 }
= if a >= b then a - b else 0
` + "```" + `

2. Run ` + "`ailang ai-check solution.ail`" + ` to verify. Z3 will either:
   - **VERIFIED**: Your function satisfies the contract for ALL valid inputs
   - **COUNTEREXAMPLE**: A specific input that violates your contract (e.g., "a=2, b=100 → result=-98")

3. Use counterexamples to fix your logic, then re-check.

**What to write contracts for:**
- Range bounds: ` + "`ensures { result >= 0 }`" + `, ` + "`ensures { result <= maxValue }`" + `
- Consistency: ` + "`ensures { result == true }`" + ` for functions that prove invariants
- Relationships: ` + "`ensures { result <= inputParam }`" + ` when output shouldn't exceed input

**What Z3 can verify:**
- Pure functions with integer arithmetic, comparisons, boolean logic
- Works with recursive functions, pattern matching, and if-then-else

**Suggested workflow:**
1. Write your solution with ` + "`ensures`" + ` contracts on pure functions
2. Run ` + "`ailang ai-check solution.ail`" + ` — if Z3 finds a counterexample, fix the bug
3. Run ` + "`ailang run`" + ` to check expected output
4. Iterate until both ai-check and output are correct`
}

// getDefaultVerificationSteps returns the standard verification steps (no ai-check)
func getDefaultVerificationSteps(caps string) string {
	return `1. **Run your solution:**
   ` + "```" + `
   ailang run --entry main --caps ` + caps + ` benchmark/solution.ail
   ` + "```" + `

2. **Compare output carefully:**
   - Check that every line matches expected output
   - No extra spaces or formatting differences

3. **If output doesn't match:**
   - Fix your code
   - Re-run to verify
   - Repeat until output matches exactly

4. **Only finish once verified!**`
}

// getToolAwareVerificationSteps returns verification steps that integrate ai-check
// into the required workflow, not as an optional sidebar
func getToolAwareVerificationSteps(caps string) string {
	return `1. **Write your solution with contracts:**
   - Add ` + "`ensures { ... }`" + ` postconditions to your pure functions
   - Express what you believe MUST hold (e.g., ` + "`result >= 0`" + `, ` + "`result <= input`" + `)

2. **Verify contracts with Z3 BEFORE running:**
   ` + "```" + `
   ailang ai-check benchmark/solution.ail
   ` + "```" + `
   - If Z3 reports COUNTEREXAMPLE: fix your logic using the counterexample inputs
   - If Z3 reports VERIFIED: your function is correct for ALL inputs
   - Re-run ai-check after every fix

3. **Run your solution:**
   ` + "```" + `
   ailang run --entry main --caps ` + caps + ` benchmark/solution.ail
   ` + "```" + `

4. **Compare output carefully:**
   - Check that every line matches expected output

5. **If output doesn't match:**
   - Add or tighten ` + "`ensures`" + ` contracts to narrow down the bug
   - Run ` + "`ailang ai-check`" + ` again — Z3 counterexamples pinpoint the exact input that breaks
   - Fix, re-check, re-run — repeat until correct

6. **Only finish once BOTH ai-check passes AND output matches!**`
}
