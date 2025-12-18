// Package gemini provides an Executor implementation for Gemini CLI.
// Updated Dec 2025 to handle Gemini CLI's actual JSON output format.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"
)

// GeminiExecutor executes tasks using Gemini CLI
type GeminiExecutor struct {
	geminiPath     string
	model          string
	timeoutSeconds int
}

// New creates a new GeminiExecutor
func New(cfg *executor.Config) (*GeminiExecutor, error) {
	geminiPath := cfg.GeminiPath
	if geminiPath == "" {
		geminiPath = "gemini"
	}

	model := cfg.GeminiModel
	if model == "" {
		model = "gemini-3-flash-preview" // Default to Gemini 3 Flash
	}

	return &GeminiExecutor{
		geminiPath:     geminiPath,
		model:          model,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier
func (e *GeminiExecutor) Name() string {
	return "gemini"
}

// Execute runs a task and returns the result
func (e *GeminiExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks
// Gemini CLI outputs a single JSON object at the end, not streaming NDJSON like Claude
func (e *GeminiExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	sessionID := uuid.New().String()

	// Build command arguments for Gemini CLI
	// Gemini CLI doesn't support system prompts, so prepend to directive if needed
	directive := task.Directive
	if task.SystemPrompt != "" {
		directive = task.SystemPrompt + "\n\n" + task.Directive
	}

	args := []string{
		directive,                 // Positional prompt (with system prompt prepended if any)
		"--output-format", "json", // Get JSON output (stream-json doesn't stream)
		"-m", e.getModel(task), // Model selection
		"-y", // Auto-approve all tool uses (YOLO mode for benchmarks)
	}

	// Add working directory if specified
	if task.Workspace != "" {
		args = append(args, "--include-directories", task.Workspace)
	}

	// Resolve gemini path - prefer Node 22 installation (required for regex /v flag)
	geminiPath := e.geminiPath
	homeDir, _ := os.UserHomeDir()
	node22GeminiPath := filepath.Join(homeDir, ".nvm", "versions", "node", "v22.20.0", "bin", "gemini")
	if _, err := os.Stat(node22GeminiPath); err == nil {
		geminiPath = node22GeminiPath
	}

	cmd := exec.CommandContext(ctx, geminiPath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	// Debug: print command being executed
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Command: %s -m %s\n", geminiPath, e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Workspace: %s\n", task.Workspace)
	}

	// Set up environment
	cwd, _ := os.Getwd()
	stdlibPath := filepath.Join(cwd, "std")
	env := os.Environ()
	env = append(env, fmt.Sprintf("AILANG_STDLIB_PATH=%s", stdlibPath))
	if task.Workspace != "" {
		env = append(env, fmt.Sprintf("PWD=%s", task.Workspace))
	}

	// Ensure Node 22+ is used (required for Gemini CLI's regex /v flag)
	// Prepend nvm's Node 22 path to PATH if available
	node22Path := filepath.Join(homeDir, ".nvm", "versions", "node", "v22.20.0", "bin")
	if _, err := os.Stat(node22Path); err == nil {
		currentPath := os.Getenv("PATH")
		env = updateEnvVar(env, "PATH", node22Path+":"+currentPath)
	}

	cmd.Env = env

	// Create pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	startTime := time.Now()
	handler.OnTurnStart(1)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start gemini: %w", err)
	}

	// Set up timeout
	timeout := task.Timeout
	if timeout == 0 {
		timeout = time.Duration(e.timeoutSeconds) * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Read output
	done := make(chan error, 1)
	var stdoutBuf, stderrBuf bytes.Buffer

	go func() {
		// Read stderr in background (debug output)
		go func() {
			_, _ = io.Copy(&stderrBuf, stderr)
		}()

		// Read stdout (JSON result)
		_, _ = io.Copy(&stdoutBuf, stdout)

		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	select {
	case <-timer.C:
		_ = cmd.Process.Kill()
		handler.OnError(fmt.Errorf("timeout after %v", timeout))
		handler.OnTurnEnd(1)
		return &executor.Result{
			Success:    false,
			Error:      fmt.Sprintf("timeout after %v", timeout),
			DurationMS: int(time.Since(startTime).Milliseconds()),
			NumTurns:   1,
			SessionID:  sessionID,
			Transcript: stderrBuf.String(),
		}, nil

	case err := <-done:
		duration := time.Since(startTime)
		handler.OnTurnEnd(1)

		if err != nil {
			handler.OnError(err)
			return &executor.Result{
				Success:    false,
				Error:      fmt.Sprintf("%v\nStderr: %s", err, stderrBuf.String()),
				DurationMS: int(duration.Milliseconds()),
				NumTurns:   1,
				SessionID:  sessionID,
				Transcript: stderrBuf.String(),
			}, nil
		}

		// Parse Gemini CLI JSON output
		var geminiResult geminiCLIResult
		if err := json.Unmarshal(stdoutBuf.Bytes(), &geminiResult); err != nil {
			return &executor.Result{
				Success:      false,
				Error:        fmt.Sprintf("failed to parse gemini output: %v", err),
				Output:       stdoutBuf.String(),
				DurationMS:   int(duration.Milliseconds()),
				NumTurns:     1,
				SessionID:    sessionID,
				Transcript:   stderrBuf.String(),
				InputTokens:  0,
				OutputTokens: 0,
			}, nil
		}

		// Extract token usage from stats
		inputTokens, outputTokens := geminiResult.extractTokenUsage()

		// Calculate cost
		cost := e.CostModel().CalculateCost(executor.TokenUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		})

		// Report token text to handler
		if geminiResult.Response != "" {
			handler.OnText(geminiResult.Response)
		}

		return &executor.Result{
			Success:      true,
			Output:       geminiResult.Response,
			DurationMS:   int(duration.Milliseconds()),
			NumTurns:     geminiResult.countTurns(),
			CostUSD:      cost,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			SessionID:    geminiResult.SessionID,
			Transcript:   stderrBuf.String(),
		}, nil
	}
}

// Capabilities returns the list of features this executor supports
func (e *GeminiExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapLocalWorkspace,
	}
}

// CostModel returns pricing information for Gemini 3 Flash
func (e *GeminiExecutor) CostModel() *executor.CostModel {
	// Gemini 3 Flash pricing: $0.50/$3.00 per 1M tokens
	return &executor.CostModel{
		ProviderName:    "google",
		InputTokenCost:  0.0005,  // $0.50 per 1M = $0.0005 per 1K
		OutputTokenCost: 0.003,   // $3.00 per 1M = $0.003 per 1K
		CacheReadCost:   0.00005, // Estimate for context caching
	}
}

// HealthCheck verifies the executor is configured and accessible
func (e *GeminiExecutor) HealthCheck(ctx context.Context) error {
	// Check if gemini binary exists
	_, err := exec.LookPath(e.geminiPath)
	if err != nil {
		return fmt.Errorf("gemini CLI not found: %w (install with: npm i -g @google/gemini-cli)", err)
	}

	// Check for API key or gcloud auth
	if os.Getenv("GEMINI_API_KEY") == "" && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		// Try to check gcloud auth status
		checkCmd := exec.CommandContext(ctx, "gcloud", "auth", "print-access-token")
		if err := checkCmd.Run(); err != nil {
			return fmt.Errorf("no Gemini credentials found: set GEMINI_API_KEY, GOOGLE_APPLICATION_CREDENTIALS, or run 'gcloud auth login'")
		}
	}

	return nil
}

// Close releases any resources held by the executor
func (e *GeminiExecutor) Close() error {
	return nil
}

func (e *GeminiExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// geminiCLIResult matches the actual Gemini CLI JSON output format
type geminiCLIResult struct {
	SessionID string      `json:"session_id"`
	Response  string      `json:"response"`
	Stats     geminiStats `json:"stats"`
}

// geminiStats contains usage statistics from Gemini CLI
type geminiStats struct {
	Models map[string]geminiModelStats `json:"models"`
	Tools  geminiToolStats             `json:"tools"`
}

// geminiModelStats contains per-model usage data
type geminiModelStats struct {
	API    geminiAPIStats   `json:"api"`
	Tokens geminiTokenStats `json:"tokens"`
}

type geminiAPIStats struct {
	TotalRequests  int `json:"totalRequests"`
	TotalErrors    int `json:"totalErrors"`
	TotalLatencyMs int `json:"totalLatencyMs"`
}

type geminiTokenStats struct {
	Prompt     int `json:"prompt"`
	Candidates int `json:"candidates"`
	Total      int `json:"total"`
	Cached     int `json:"cached"`
	Thoughts   int `json:"thoughts"`
	Tool       int `json:"tool"`
}

type geminiToolStats struct {
	TotalCalls      int `json:"totalCalls"`
	TotalSuccess    int `json:"totalSuccess"`
	TotalFail       int `json:"totalFail"`
	TotalDurationMs int `json:"totalDurationMs"`
}

// extractTokenUsage aggregates token usage across all models used
func (r *geminiCLIResult) extractTokenUsage() (inputTokens, outputTokens int) {
	for _, modelStats := range r.Stats.Models {
		inputTokens += modelStats.Tokens.Prompt
		outputTokens += modelStats.Tokens.Candidates
	}
	return inputTokens, outputTokens
}

// countTurns estimates the number of turns based on API requests
func (r *geminiCLIResult) countTurns() int {
	totalRequests := 0
	for _, modelStats := range r.Stats.Models {
		totalRequests += modelStats.API.TotalRequests
	}
	if totalRequests == 0 {
		return 1
	}
	return totalRequests
}

// updateEnvVar updates or adds an environment variable in a slice
func updateEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// Register registers the Gemini executor with the global factory
func Register() {
	executor.GlobalFactory().Register("gemini", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
