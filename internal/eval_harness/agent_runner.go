package eval_harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/eval_harness/langreg"
)

// AgentBenchmarkConfig configures agent-based evaluation
type AgentBenchmarkConfig struct {
	// NOTE: MaxConcurrent (formerly the -agent-parallel flag) was removed
	// 2026-05-23. It was never actually read for dispatch control; the real
	// semaphore is the -parallel flag handled in cmd/ailang/eval_parallel.go.
	// Keeping the field caused recurring user confusion (passing -agent-parallel 1
	// expecting serial execution and getting -parallel=10 oversubscription).
	RequestsPerSecond  int           // API rate limit
	TimeoutSeconds     int           // Timeout per benchmark
	WorkspaceDir       string        // Base workspace directory
	AllowedTools       []string      // Tools agent can use
	ClaudePath         string        // Path to claude CLI
	ClaudeModel        string        // Claude model to use (haiku, sonnet, opus, or full name)
	Verify             bool          // Enable contract verification (M-CONTRACT-EVAL)
	DevtoolsPrompt     string        // Devtools prompt content to append to system prompt (M-CONTRACT-EVAL)
	AgentPromptContent string        // Agent coding prompt content (replaces teaching prompt when UseAgentPrompt condition is active)
	Condition          EvalCondition // Experimental condition (overrides Verify/DevtoolsPrompt when set)
	MicroragMode       MicroragMode  // μRAG subprocess env mode (M-BRAIN-MICRORAG): on/off/auto

	// MaxTokensPerBench (M-EVAL-OS-LONGITUDINAL Phase 1, v0.23.0): hard
	// token-budget ceiling per benchmark for thrash detection on $0 local
	// models. 0 = unlimited (legacy behaviour). Plumbs through to
	// executor.Task.MaxTokensPerBench. Set via the -max-tokens-per-bench CLI
	// flag in cmd/ailang/eval_suite.go.
	MaxTokensPerBench int
}

// DefaultAgentConfig returns sensible defaults
func DefaultAgentConfig() AgentBenchmarkConfig {
	return AgentBenchmarkConfig{
		RequestsPerSecond: 1, // Conservative for API quotas
		TimeoutSeconds:    300,
		WorkspaceDir:      "/tmp/ailang_eval",
		AllowedTools:      []string{"Bash", "Read", "Write", "Edit", "Grep"},
		ClaudePath:        "claude", // Use PATH
		ClaudeModel:       "haiku",  // Default to Haiku for cost efficiency
	}
}

// AgentBenchmarkResult captures agent evaluation outcome
type AgentBenchmarkResult struct {
	BenchmarkID   string
	Executor      string // Executor used: "claude", "gemini", etc.
	Success       bool
	Iterations    int     // Number of agent turns
	Cost          float64 // Total cost in USD
	DurationMS    int     // Total time in milliseconds
	NumTurns      int     // Conversation turns
	ToolCallCount int     // Number of tool invocations (validates agentic behavior)
	Error         string  // Error message if failed
	SessionID     string  // Session ID from executor
	Result        string  // Final result text from agent

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

	// Timing breakdown
	TTFTSeconds float64 `json:"ttft_seconds,omitempty"` // Time to first token in seconds

	// Cross-harness grouping
	ModelFamily string `json:"model_family,omitempty"` // Logical model family (e.g. "claude-sonnet-4-6"); empty = no grouping

	// Contract verification results (M-CONTRACT-EVAL)
	VerifyOk        bool   `json:"verify_ok"`             // All contracts verified
	VerifyVerified  int    `json:"verify_verified"`       // Count of verified functions
	VerifyCounterex int    `json:"verify_counterexample"` // Count of counterexamples
	VerifySkipped   int    `json:"verify_skipped"`        // Count of skipped functions
	VerifyErrors    int    `json:"verify_errors"`         // Count of Z3 errors
	VerifyJSON      string `json:"verify_json,omitempty"` // Full ai-check JSON output

	// Cost-and-speed budget metrics (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
	// Populated from executor.Result. Zero values mean "not measured".
	CostKilledAt   float64 `json:"cost_killed_at,omitempty"`   // > 0 if execution stopped because cost budget exceeded
	FirstAttemptMs int64   `json:"first_attempt_ms,omitempty"` // ms from task start to first solution submission
	SuccessAtMs    int64   `json:"success_at_ms,omitempty"`    // ms from task start to first passing solution (-1 = never)
	TokensPerSec   float64 `json:"tokens_per_sec,omitempty"`   // OutputTokens / generation_seconds

	// FinishReason (M-EVAL-SWEET-SPOT, v0.19.0) — passed through from
	// executor.Result.FinishReason. Drives typed error_category classification
	// via CategorizeAgentError.
	FinishReason string `json:"finish_reason,omitempty"`

	// Context-compaction telemetry (M-AILANG-SEMANTIC-CONTEXT, v0.26.0) — passed
	// through from executor.Result. Leading indicator of convergence thrash.
	CompactionCount     int `json:"compaction_count,omitempty"`
	CompactionFirstStep int `json:"first_compaction_step,omitempty"`
	CompactionMaxLevel  int `json:"compaction_level_max,omitempty"`
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
		return nil, fmt.Errorf("claude CLI check failed: %w", err)
	}

	// Create isolated workspace for this benchmark (include language for separation)
	workspace := filepath.Join(config.WorkspaceDir, fmt.Sprintf("%s_%s_%d", spec.ID, language, os.Getpid()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create minimal .git folder so Claude treats workspace as standalone project
	// This prevents Claude from walking up and finding the parent AILANG repo
	gitDir := filepath.Join(workspace, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .git folder: %w", err)
	}

	// Only cleanup workspace if DEBUG_AGENT is not set (allows inspection)
	if os.Getenv("DEBUG_AGENT") == "" {
		defer os.RemoveAll(workspace)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG_AGENT] Workspace preserved: %s\n", workspace)
	}

	// Seed benchmark input files so the agent can actually run/test its solution
	// (e.g. cli_args reads numbers.txt). Mirrors the standard-runner layout.
	if err := seedInputFiles(workspace, spec); err != nil {
		return nil, err
	}

	// Create placeholder solution file that Claude will overwrite
	// For AILANG: Create benchmark/ subdirectory and pre-populate with correct module declaration
	// For Python: Create solution.py in workspace root
	var solutionPath string
	var placeholder string

	if language == "ailang" {
		// Create benchmark/ subdirectory — AILANG requires module path to match dir
		benchmarkDir := filepath.Join(workspace, "benchmark")
		if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create benchmark dir: %w", err)
		}
		solutionPath = filepath.Join(benchmarkDir, "solution.ail")
		placeholder = `module benchmark/solution
// ⚠️ DO NOT CHANGE THE MODULE DECLARATION ABOVE! ⚠️
// It MUST match the file path (benchmark/solution.ail)
// Changing it will cause MOD010 error: "module declaration doesn't match canonical path"

// TODO: Add your solution code below this line

`
	} else {
		// All other languages (python, javascript, go, …) — place solution in workspace root.
		lang, err := langreg.Get(language)
		if err != nil {
			return nil, fmt.Errorf("unknown language %q: %w", language, err)
		}
		solutionPath = filepath.Join(workspace, lang.SolutionFilename())
		placeholder = "# TODO: Write your solution here\n"
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
	result, err = RunHeadlessSessionStreaming(spec, systemPrompt, taskPrompt, workspace, config)
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

	// M-CONTRACT-EVAL: Post-hoc contract verification for agent mode
	var verifyResult *AICheckResult
	var verifyRawJSON string
	if config.Verify && spec.ContractSpec != "" && language == "ailang" && validation.CompileOk {
		verifyResult, verifyRawJSON, _ = RunAICheck("", solutionPath, 5*time.Second)
	}

	// Overall success is when all validations pass
	success := validation.CompileOk && validation.RuntimeOk && validation.StdoutOk

	agentResult := &AgentBenchmarkResult{
		BenchmarkID: spec.ID,
		Executor:    "claude", // Legacy runner always uses Claude Code
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
	}

	// M-CONTRACT-EVAL: Populate verify fields if verification was run
	if verifyResult != nil {
		agentResult.VerifyVerified = verifyResult.Verify.Verified
		agentResult.VerifyCounterex = verifyResult.Verify.Counterexample
		agentResult.VerifySkipped = verifyResult.Verify.Skipped
		agentResult.VerifyErrors = verifyResult.Verify.Errors
		agentResult.VerifyOk = verifyResult.Verify.Available && verifyResult.Verify.Counterexample == 0 && verifyResult.Verify.Errors == 0
		agentResult.VerifyJSON = verifyRawJSON
	}

	return agentResult, nil
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
		return fmt.Errorf("claude CLI v2.0+ required for JSON output, got: %s", version)
	}

	return nil
}

// Note: prepareWorkspace and generateAgentPrompt moved to agent_prompt.go
// for better organization and enhanced prompt generation with full syntax reference

// runHeadlessSession executes Claude Code in headless mode
//
//nolint:unused // Kept for backwards compatibility, superseded by runHeadlessSessionStreaming
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

	// Override environment to make workspace the "project" directory
	// This prevents Claude from finding the parent AILANG repo and creating files there
	env := os.Environ()
	// Remove parent project dir markers
	filteredEnv := make([]string, 0, len(env))
	for _, e := range env {
		// Skip environment variables that might leak parent project context
		if strings.HasPrefix(e, "PWD=") || strings.HasPrefix(e, "OLDPWD=") {
			continue
		}
		filteredEnv = append(filteredEnv, e)
	}
	// Set PWD to workspace so Claude knows this is the "project"
	filteredEnv = append(filteredEnv, fmt.Sprintf("PWD=%s", workspace))
	// Apply μRAG mode (M-BRAIN-MICRORAG): force AILANG_MICRORAG_ENABLED for A/B comparison.
	filteredEnv = config.MicroragMode.ApplyToEnv(filteredEnv)
	cmd.Env = filteredEnv

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

	// Locate solution file via the language registry.
	var solutionPath string
	if language == "ailang" {
		solutionPath = filepath.Join(workspace, "benchmark", "solution.ail")
	} else {
		lang, langErr := langreg.Get(language)
		if langErr != nil {
			return ValidationResult{Stderr: fmt.Sprintf("unknown language %q: %v", language, langErr)}
		}
		solutionPath = filepath.Join(workspace, lang.SolutionFilename())
	}

	solutionContent, err := os.ReadFile(solutionPath)
	if err != nil || len(solutionContent) == 0 {
		return ValidationResult{
			Stderr: fmt.Sprintf("solution file not found or empty: %v", err),
		}
	}

	// Obtain a runner for this language via the registry.  AILANG gets its
	// special runner (needs spec for caps/stdin/input_files); everything else
	// uses GetRunnerWithContext which routes through langreg.
	var runner LanguageRunner
	if language == "ailang" {
		runner = NewAILANGRunnerWithTask(context.Background(), "", spec.Caps, "", spec)
	} else {
		r, runErr := GetRunnerWithContext(context.Background(), language, spec, "")
		if runErr != nil {
			return ValidationResult{Stderr: fmt.Sprintf("no runner for %q: %v", language, runErr)}
		}
		runner = r
	}

	timeout := 30 * time.Second
	runResult, err := runner.Run(string(solutionContent), timeout)
	if err != nil {
		return ValidationResult{Stderr: fmt.Sprintf("validation runner error: %v", err)}
	}

	stdoutOk := runResult.RuntimeOk && GradeStdout(spec, runResult.Stdout, string(solutionContent))
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
//
//nolint:unused // Kept for backwards compatibility
func getSolutionFilename(language string) string {
	lang, err := langreg.Get(language)
	if err != nil {
		return "solution.ail"
	}
	return lang.SolutionFilename()
}
