package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AgentBenchmarkConfig configures agent-based evaluation
type AgentBenchmarkConfig struct {
	MaxConcurrent     int      // Max parallel Claude sessions
	RequestsPerSecond int      // API rate limit
	TimeoutSeconds    int      // Timeout per benchmark
	MaxIterations     int      // Max agent iterations
	WorkspaceDir      string   // Base workspace directory
	AllowedTools      []string // Tools agent can use
	ClaudePath        string   // Path to claude CLI
}

// DefaultAgentConfig returns sensible defaults
func DefaultAgentConfig() AgentBenchmarkConfig {
	return AgentBenchmarkConfig{
		MaxConcurrent:     10,
		RequestsPerSecond: 1, // Conservative for API quotas
		TimeoutSeconds:    300,
		MaxIterations:     10,
		WorkspaceDir:      "/tmp/ailang_eval",
		AllowedTools:      []string{"Bash", "Read", "Write", "Edit", "Grep"},
		ClaudePath:        "claude", // Use PATH
	}
}

// AgentBenchmarkResult captures agent evaluation outcome
type AgentBenchmarkResult struct {
	BenchmarkID string
	Success     bool
	Iterations  int     // Number of agent turns
	Cost        float64 // Total cost in USD
	DurationMS  int     // Total time in milliseconds
	NumTurns    int     // Conversation turns
	Error       string  // Error message if failed
	SessionID   string  // Claude session ID
	Result      string  // Final result text from agent

	// Token usage details
	Usage      TokenUsage            `json:"usage"`
	ModelUsage map[string]ModelStats `json:"modelUsage"`
}

// TokenUsage captures detailed token metrics
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// ModelStats captures per-model statistics
type ModelStats struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow"`
}

// ClaudeHeadlessResult is the JSON structure returned by `claude -p --output-format json`
type ClaudeHeadlessResult struct {
	Type              string                `json:"type"`
	Subtype           string                `json:"subtype"`
	IsError           bool                  `json:"is_error"`
	DurationMS        int                   `json:"duration_ms"`
	DurationAPIMS     int                   `json:"duration_api_ms"`
	NumTurns          int                   `json:"num_turns"`
	Result            string                `json:"result"`
	SessionID         string                `json:"session_id"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             TokenUsage            `json:"usage"`
	ModelUsage        map[string]ModelStats `json:"modelUsage"`
	PermissionDenials []interface{}         `json:"permission_denials"`
	UUID              string                `json:"uuid"`
}

// RunAgentBenchmark runs a single benchmark using Claude Code headless mode
func RunAgentBenchmark(spec *BenchmarkSpec, config AgentBenchmarkConfig) (*AgentBenchmarkResult, error) {
	// Check Claude CLI is available
	if err := checkClaudeCLI(config.ClaudePath); err != nil {
		return nil, fmt.Errorf("Claude CLI check failed: %w", err)
	}

	// Create isolated workspace for this benchmark
	workspace := filepath.Join(config.WorkspaceDir, fmt.Sprintf("%s_%d", spec.ID, os.Getpid()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	defer os.RemoveAll(workspace) // Cleanup after execution

	// Prepare workspace files
	if err := prepareWorkspace(workspace, spec); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	// Generate agent prompt
	prompt := generateAgentPrompt(spec, config)

	// Run headless Claude session
	result, err := runHeadlessSession(prompt, workspace, config)
	if err != nil {
		return nil, fmt.Errorf("headless session failed: %w", err)
	}

	// Parse result and determine success
	success := determineSuccess(result, spec, workspace)

	return &AgentBenchmarkResult{
		BenchmarkID: spec.ID,
		Success:     success,
		Iterations:  result.NumTurns,
		Cost:        result.TotalCostUSD,
		DurationMS:  result.DurationMS,
		NumTurns:    result.NumTurns,
		Error:       getErrorMessage(result),
		SessionID:   result.SessionID,
		Result:      result.Result,
		Usage:       result.Usage,
		ModelUsage:  result.ModelUsage,
	}, nil
}

// checkClaudeCLI verifies Claude CLI is installed and has correct version
func checkClaudeCLI(claudePath string) error {
	// Check if claude command exists
	cmd := exec.Command("command", "-v", claudePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude command not found. Install with: make setup-claude")
	}

	// Verify version (need v2.0+ for JSON output)
	cmd = exec.Command(claudePath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get claude version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if !strings.Contains(version, "2.") {
		return fmt.Errorf("Claude CLI v2.0+ required for JSON output, got: %s", version)
	}

	return nil
}

// prepareWorkspace creates workspace files for the agent
func prepareWorkspace(workspace string, spec *BenchmarkSpec) error {
	// Create README.md with problem description
	readme := fmt.Sprintf(`# %s

%s

## Task

%s

## Expected Output

%s
`, spec.ID, spec.Description, spec.TaskPrompt, spec.ExpectedOut)

	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte(readme), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Create empty solution.ail stub
	solutionStub := `// Your implementation goes here
// Use 'ailang check solution.ail' to type-check
// Use 'ailang run --entry main --caps IO solution.ail' to test
`
	if err := os.WriteFile(filepath.Join(workspace, "solution.ail"), []byte(solutionStub), 0644); err != nil {
		return fmt.Errorf("failed to write solution.ail: %w", err)
	}

	// Create syntax_reference.md (embed AILANG syntax)
	// For now, reference the active prompt - will be enhanced in Milestone 1.2
	syntaxRef := `# AILANG Syntax Reference

See the active AILANG teaching prompt for complete syntax reference.

Key constructs:
- Functions: func name(x: int): int = x + 1
- Let bindings: let x = 42 in x + 1
- Lambdas: \x. x + 1
- Pattern matching: match expr with | Some(x) -> x | None -> 0
- Effects: func read_file(path: string): string ! {IO, FS}
`
	if err := os.WriteFile(filepath.Join(workspace, "syntax_reference.md"), []byte(syntaxRef), 0644); err != nil {
		return fmt.Errorf("failed to write syntax_reference.md: %w", err)
	}

	return nil
}

// generateAgentPrompt creates the prompt for the agent
func generateAgentPrompt(spec *BenchmarkSpec, config AgentBenchmarkConfig) string {
	return fmt.Sprintf(`You are solving an AILANG benchmark in an isolated workspace.

Workspace files:
- README.md: Problem description and expected output
- solution.ail: Your implementation (currently a stub)
- syntax_reference.md: AILANG language syntax

Your task:
1. Read README.md to understand the problem
2. Read syntax_reference.md for AILANG syntax
3. Implement solution in solution.ail
4. Run: ailang check solution.ail (type check)
5. Run: ailang run --entry main --caps %s solution.ail (execute)
6. Compare output with expected output in README.md
7. Iterate until output matches

Timeout: %d seconds
Max iterations: %d
Success: Output matches expected output exactly (after trimming whitespace)

IMPORTANT: The solution must be in solution.ail, not in a comment or inline code block.
`, strings.Join(spec.Caps, ","), config.TimeoutSeconds, config.MaxIterations)
}

// runHeadlessSession executes Claude Code in headless mode
func runHeadlessSession(prompt, workspace string, config AgentBenchmarkConfig) (*ClaudeHeadlessResult, error) {
	// Build command: claude -p <prompt> --output-format json --workspace <dir> --allowedTools <tools>
	cmd := exec.Command(config.ClaudePath, "-p", prompt,
		"--output-format", "json",
		"--workspace", workspace,
		"--allowedTools", strings.Join(config.AllowedTools, ","),
	)

	// Set timeout
	timeout := time.Duration(config.TimeoutSeconds) * time.Second

	// Run command with timeout
	done := make(chan error, 1)
	var output []byte
	var err error

	go func() {
		output, err = cmd.Output()
		done <- err
	}()

	select {
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("claude session timed out after %d seconds", config.TimeoutSeconds)
	case err := <-done:
		if err != nil {
			// Check for stderr
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("claude failed: %s", string(exitErr.Stderr))
			}
			return nil, fmt.Errorf("claude failed: %w", err)
		}
	}

	// Parse JSON result
	var result ClaudeHeadlessResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse claude JSON output: %w\nOutput: %s", err, string(output))
	}

	return &result, nil
}

// determineSuccess checks if the benchmark succeeded
func determineSuccess(result *ClaudeHeadlessResult, spec *BenchmarkSpec, workspace string) bool {
	// If Claude session errored, it's a failure
	if result.IsError || result.Subtype != "success" {
		return false
	}

	// Check if solution.ail exists and has content
	solutionPath := filepath.Join(workspace, "solution.ail")
	solutionContent, err := os.ReadFile(solutionPath)
	if err != nil || len(solutionContent) == 0 {
		return false
	}

	// Try to run the solution and compare output
	runner := NewAILANGRunner("", spec.Caps)
	runResult, err := runner.Run(string(solutionContent), 10*time.Second)
	if err != nil {
		return false
	}

	// Check if output matches
	if !runResult.RuntimeOk {
		return false
	}

	return CompareOutput(spec.ExpectedOut, runResult.Stdout)
}

// getErrorMessage extracts error message from result
func getErrorMessage(result *ClaudeHeadlessResult) string {
	if result.IsError {
		return result.Result
	}
	if result.Subtype == "error" {
		return result.Result
	}
	return ""
}
