package eval_harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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
	ClaudeModel       string   // Claude model to use (haiku, sonnet, opus, or full name)
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
		ClaudePath:        "claude",  // Use PATH
		ClaudeModel:       "haiku",   // Default to Haiku for cost efficiency
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

	// Solution and session log for inspection
	SolutionCode  string `json:"solution_code,omitempty"`  // Generated solution code
	SessionLog    string `json:"session_log,omitempty"`    // Full Claude session log
	PromptVersion string `json:"prompt_version,omitempty"` // Version of teaching prompt used

	// Validation flags (match standard eval format for downstream compatibility)
	CompileOk bool   `json:"compile_ok"` // Did solution parse/compile?
	RuntimeOk bool   `json:"runtime_ok"` // Did solution run without error?
	StdoutOk  bool   `json:"stdout_ok"`  // Did output match expected?
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
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
	Transcript        string                `json:"-"` // Full conversation transcript (not in JSON, set by streaming)
}

// RunAgentBenchmark runs a single benchmark using Claude Code headless mode
// language parameter specifies which language to run (ailang, python, etc.)
func RunAgentBenchmark(spec *BenchmarkSpec, config AgentBenchmarkConfig, language string) (*AgentBenchmarkResult, error) {
	// Default to ailang if not specified
	if language == "" {
		language = "ailang"
	}

	// Check Claude CLI is available
	if err := checkClaudeCLI(config.ClaudePath); err != nil {
		return nil, fmt.Errorf("Claude CLI check failed: %w", err)
	}

	// Create isolated workspace for this benchmark (include language for separation)
	workspace := filepath.Join(config.WorkspaceDir, fmt.Sprintf("%s_%s_%d", spec.ID, language, os.Getpid()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Only cleanup workspace if DEBUG_AGENT is not set (allows inspection)
	if os.Getenv("DEBUG_AGENT") == "" {
		defer os.RemoveAll(workspace)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Workspace preserved: %s\n", workspace)
	}

	// Create placeholder solution file that Claude will overwrite
	// For AILANG: Create benchmark/ subdirectory and pre-populate with correct module declaration
	// For Python: Create solution.py in workspace root
	var solutionPath string
	var placeholder string

	if language == "ailang" {
		// Create benchmark/ subdirectory
		benchmarkDir := filepath.Join(workspace, "benchmark")
		if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create benchmark dir: %w", err)
		}

		// Create solution.ail with correct module declaration
		solutionPath = filepath.Join(benchmarkDir, "solution.ail")
		placeholder = `module benchmark/solution

// TODO: Add your solution code here
// The module declaration above is correct - just add your function definitions below

`
	} else {
		// Python - simple placeholder in workspace root
		solutionPath = filepath.Join(workspace, "solution.py")
		placeholder = "# TODO: Write your Python solution here\n"
	}

	// Generate split prompts: system (language knowledge) + task (benchmark description)
	// System prompt loaded from prompts/versions.json (versioned teaching prompts)
	// Task prompt loaded from generic .txt templates
	// Pass solutionPath so prompt can include full path
	systemPrompt, taskPrompt, promptVersion, err := GenerateAgentPromptsWithSystemPrompt(spec, config, language, "", solutionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate prompts: %w", err)
	}

	if err := os.WriteFile(solutionPath, []byte(placeholder), 0644); err != nil {
		return nil, fmt.Errorf("failed to create solution placeholder: %w", err)
	}

	// Run headless Claude session with streaming (always enabled for transcript capture)
	// Streaming mode captures full conversation log which is essential for debugging failures
	// Uses --system-prompt flag for language knowledge, -p for task instructions
	var result *ClaudeHeadlessResult
	result, err = runHeadlessSessionStreaming(systemPrompt, taskPrompt, workspace, config)
	if err != nil {
		// Still try to capture partial results even if session failed
		// (though with recent changes, runHeadlessSessionStreaming should return a result, not error)
		return nil, fmt.Errorf("headless session failed: %w", err)
	}

	// Parse result and determine success - returns detailed validation results
	validation := determineSuccess(result, spec, workspace, language)

	// Read solution code from workspace BEFORE defer cleanup runs
	// Claude should have written to solution.py or solution.ail
	solutionCode, err := os.ReadFile(solutionPath)
	if err != nil {
		// If file doesn't exist or can't be read, solution is empty
		solutionCode = []byte{}
		if os.Getenv("DEBUG_AGENT") != "" {
			fmt.Fprintf(os.Stderr, "[WARN] Could not read solution file %s: %v\n", solutionPath, err)
		}
	}

	// Write transcript to file BEFORE defer cleanup (transcript is returned in result.Transcript)
	// This ensures the file exists when we try to read it back
	if result.Transcript != "" {
		sessionLogPath := filepath.Join(workspace, "claude_session.log")
		if err := os.WriteFile(sessionLogPath, []byte(result.Transcript), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to write session log: %v\n", err)
		}
	}

	// Use transcript from result directly (already formatted)
	sessionLog := result.Transcript

	// Overall success is when all validations pass
	success := validation.CompileOk && validation.RuntimeOk && validation.StdoutOk

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
		// Add solution and session log for inspection
		SolutionCode:  string(solutionCode),
		SessionLog:    string(sessionLog),
		PromptVersion: promptVersion, // Track which prompt version was used
		// Validation flags (match standard eval format)
		CompileOk: validation.CompileOk,
		RuntimeOk: validation.RuntimeOk,
		StdoutOk:  validation.StdoutOk,
		Stdout:    validation.Stdout,
		Stderr:    validation.Stderr,
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

// Note: prepareWorkspace and generateAgentPrompt moved to agent_prompt.go
// for better organization and enhanced prompt generation with full syntax reference

// runHeadlessSession executes Claude Code in headless mode
func runHeadlessSession(prompt, workspace string, config AgentBenchmarkConfig) (*ClaudeHeadlessResult, error) {
	// Generate UUID for session ID (Claude CLI requires valid UUID)
	sessionID := uuid.New().String()

	// Build command: claude -p <prompt> --output-format json --model <model> --session-id <id>
	// Note: --add-dir grants tool access to workspace directory
	cmd := exec.Command(config.ClaudePath, "-p", prompt,
		"--output-format", "json",
		"--model", config.ClaudeModel,
		"--session-id", sessionID,
		"--add-dir", workspace,
		"--allowedTools", strings.Join(config.AllowedTools, ","),
	)

	// Set working directory to workspace so relative paths work
	cmd.Dir = workspace

	// DEBUG: Capture stderr for visibility
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	// DEBUG: If DEBUG_AGENT is set, print command being run
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Running: %s\n", strings.Join(cmd.Args, " "))
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Workspace: %s\n", workspace)
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Prompt length: %d chars\n", len(prompt))
	}

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
		stderr := stderrBuf.String()
		if stderr != "" {
			fmt.Fprintf(os.Stderr, "[TIMEOUT] Claude stderr:\n%s\n", stderr)
		}
		return nil, fmt.Errorf("claude session timed out after %d seconds", config.TimeoutSeconds)
	case err := <-done:
		if err != nil {
			stderr := stderrBuf.String()
			if stderr != "" {
				fmt.Fprintf(os.Stderr, "[ERROR] Claude stderr:\n%s\n", stderr)
			}
			return nil, fmt.Errorf("claude failed: %w\nStderr: %s", err, stderr)
		}
	}

	// DEBUG: Print stderr even on success if DEBUG_AGENT is set
	if os.Getenv("DEBUG_AGENT") != "" {
		stderr := stderrBuf.String()
		if stderr != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Claude stderr:\n%s\n", stderr)
		}
	}

	// Save session log to workspace for inspection
	logPath := filepath.Join(workspace, "claude_session.log")
	logData := fmt.Sprintf("=== Claude Session Log ===\n\nPrompt:\n%s\n\nStderr:\n%s\n\nJSON Output:\n%s\n",
		prompt, stderrBuf.String(), string(output))
	if err := os.WriteFile(logPath, []byte(logData), 0644); err != nil {
		// Don't fail if we can't write log, just warn
		fmt.Fprintf(os.Stderr, "[WARN] Failed to write session log: %v\n", err)
	}

	// Parse JSON result
	var result ClaudeHeadlessResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse claude JSON output: %w\nOutput: %s", err, string(output))
	}

	return &result, nil
}

// ValidationResult holds detailed validation results for agent benchmarks
type ValidationResult struct {
	CompileOk bool
	RuntimeOk bool
	StdoutOk  bool
	Stdout    string
	Stderr    string
}

// determineSuccess checks if the benchmark succeeded and returns detailed validation results
func determineSuccess(result *ClaudeHeadlessResult, spec *BenchmarkSpec, workspace string, language string) ValidationResult {
	// If Claude session errored, everything fails
	if result.IsError || result.Subtype != "success" {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    result.Result,
		}
	}

	// Check if solution file exists and has content
	// For AILANG: benchmark/solution.ail
	// For Python: solution.py
	var solutionPath string
	if language == "ailang" {
		solutionPath = filepath.Join(workspace, "benchmark", "solution.ail")
	} else {
		solutionPath = filepath.Join(workspace, "solution.py")
	}

	solutionContent, err := os.ReadFile(solutionPath)
	if err != nil || len(solutionContent) == 0 {
		return ValidationResult{
			CompileOk: false,
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    fmt.Sprintf("Solution file not found or empty: %v", err),
		}
	}

	// Run solution based on language
	if language == "python" {
		// Run Python solution
		cmd := exec.Command("python3", solutionPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return ValidationResult{
				CompileOk: true,  // Python doesn't have compile phase
				RuntimeOk: false, // Runtime error
				StdoutOk:  false,
				Stdout:    string(output),
				Stderr:    fmt.Sprintf("Python execution failed: %v", err),
			}
		}
		stdoutOk := CompareOutput(spec.ExpectedOut, string(output))
		return ValidationResult{
			CompileOk: true,
			RuntimeOk: true,
			StdoutOk:  stdoutOk,
			Stdout:    string(output),
		}
	}

	// Run AILANG solution
	runner := NewAILANGRunner("", spec.Caps)
	runResult, err := runner.Run(string(solutionContent), 10*time.Second)
	if err != nil {
		return ValidationResult{
			CompileOk: false, // Compilation failed
			RuntimeOk: false,
			StdoutOk:  false,
			Stderr:    err.Error(),
		}
	}

	// AILANG ran - check runtime and output
	stdoutOk := runResult.RuntimeOk && CompareOutput(spec.ExpectedOut, runResult.Stdout)
	return ValidationResult{
		CompileOk: runResult.CompileOk,
		RuntimeOk: runResult.RuntimeOk,
		StdoutOk:  stdoutOk,
		Stdout:    runResult.Stdout,
		Stderr:    runResult.Stderr,
	}
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

// getSolutionFilename returns the language-specific solution filename
func getSolutionFilename(language string) string {
	switch language {
	case "python":
		return "solution.py"
	case "ailang":
		return "solution.ail"
	default:
		// Default to ailang for backward compatibility
		return "solution.ail"
	}
}
