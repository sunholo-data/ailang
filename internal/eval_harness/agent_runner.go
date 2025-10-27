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
	SolutionCode string `json:"solution_code,omitempty"` // Generated solution code
	SessionLog   string `json:"session_log,omitempty"`   // Full Claude session log
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

	// Create isolated workspace for this benchmark
	workspace := filepath.Join(config.WorkspaceDir, fmt.Sprintf("%s_%d", spec.ID, os.Getpid()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Only cleanup workspace if DEBUG_AGENT is not set (allows inspection)
	if os.Getenv("DEBUG_AGENT") == "" {
		defer os.RemoveAll(workspace)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Workspace preserved: %s\n", workspace)
	}

	// Generate enhanced prompt with full syntax reference for the specified language
	prompt, syntaxRef, err := EnhancedGenerateAgentPrompt(spec, config, language)
	if err != nil {
		return nil, fmt.Errorf("failed to generate prompt: %w", err)
	}

	// Prepare workspace files with full syntax reference
	if err := PrepareWorkspaceWithSyntax(workspace, spec, syntaxRef); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	// Run headless Claude session with streaming (always enabled for transcript capture)
	// Streaming mode captures full conversation log which is essential for debugging failures
	var result *ClaudeHeadlessResult
	result, err = runHeadlessSessionStreaming(prompt, workspace, config)
	if err != nil {
		// Still try to capture partial results even if session failed
		// (though with recent changes, runHeadlessSessionStreaming should return a result, not error)
		return nil, fmt.Errorf("headless session failed: %w", err)
	}

	// Parse result and determine success
	success := determineSuccess(result, spec, workspace)

	// Write transcript to file BEFORE defer cleanup (transcript is returned in result.Transcript)
	// This ensures the file exists when we try to read it back
	if result.Transcript != "" {
		sessionLogPath := filepath.Join(workspace, "claude_session.log")
		if err := os.WriteFile(sessionLogPath, []byte(result.Transcript), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to write session log: %v\n", err)
		}
	}

	// Capture solution code (use language-specific filename)
	solutionFilename := getSolutionFilename(language)
	solutionPath := filepath.Join(workspace, solutionFilename)
	solutionCode, _ := os.ReadFile(solutionPath) // Ignore error, field will be empty if read fails

	// Use transcript from result directly (already formatted)
	sessionLog := result.Transcript

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
		SolutionCode: string(solutionCode),
		SessionLog:   string(sessionLog),
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
