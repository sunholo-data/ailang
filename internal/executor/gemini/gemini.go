// Package gemini provides an Executor implementation for Gemini CLI.
// Gemini CLI has nearly identical syntax to Claude Code CLI, enabling 80% code reuse.
package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
// Gemini CLI uses nearly identical syntax to Claude:
//
//	gemini -p "prompt" --output-format stream-json -m model
func (e *GeminiExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	sessionID := uuid.New().String()

	// Build command arguments - nearly identical to Claude!
	args := []string{
		"-p", task.Directive, // Same as Claude
		"--output-format", "stream-json", // Same as Claude
		"-m", e.getModel(task), // -m instead of --model
	}

	// System prompt - same flag as Claude
	if task.SystemPrompt != "" {
		args = append([]string{"--system-prompt", task.SystemPrompt}, args...)
	}

	// Add working directory if specified
	if task.Workspace != "" {
		args = append(args, "--add-dir", task.Workspace) // Same as Claude
	}

	cmd := exec.CommandContext(ctx, e.geminiPath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	// Set up environment
	cwd, _ := os.Getwd()
	stdlibPath := filepath.Join(cwd, "std")
	env := os.Environ()
	env = append(env, fmt.Sprintf("AILANG_STDLIB_PATH=%s", stdlibPath))
	if task.Workspace != "" {
		env = append(env, fmt.Sprintf("PWD=%s", task.Workspace))
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

	// Parse streaming output - reuse Claude's NDJSON parser!
	// Gemini CLI uses the same stream-json format
	done := make(chan error, 1)
	var finalResult *geminiHeadlessResult
	var transcriptBuf strings.Builder
	var turnNum int
	var totalInputTokens, totalOutputTokens int

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// Read stderr in background
		go func() {
			for stderrScanner.Scan() {
				_ = stderrScanner.Text()
			}
		}()

		// Parse NDJSON from stdout - same format as Claude!
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			if line == "" {
				continue
			}

			var event map[string]interface{}
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			eventType, _ := event["type"].(string)

			switch eventType {
			case "stream_event":
				streamEvent, _ := event["event"].(map[string]interface{})
				if streamEvent == nil {
					continue
				}

				streamType, _ := streamEvent["type"].(string)
				switch streamType {
				case "message_start":
					turnNum++
					handler.OnTurnStart(turnNum)
					transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))

				case "content_block_start":
					contentBlock, _ := streamEvent["content_block"].(map[string]interface{})
					if contentBlock != nil {
						if blockType, _ := contentBlock["type"].(string); blockType == "tool_use" {
							toolName, _ := contentBlock["name"].(string)
							handler.OnToolUse(toolName, "")
							transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", toolName))
						}
					}

				case "content_block_delta":
					delta, _ := streamEvent["delta"].(map[string]interface{})
					if delta != nil {
						if deltaType, _ := delta["type"].(string); deltaType == "text_delta" {
							text, _ := delta["text"].(string)
							handler.OnText(text)
							transcriptBuf.WriteString(text)
						}
					}

				case "message_stop":
					handler.OnTurnEnd(turnNum)
				}

			case "result":
				if err := json.Unmarshal([]byte(line), &finalResult); err != nil {
					done <- fmt.Errorf("failed to parse final result: %w", err)
					return
				}
				// Extract token usage if available
				if finalResult != nil && finalResult.Usage != nil {
					totalInputTokens = finalResult.Usage.InputTokens
					totalOutputTokens = finalResult.Usage.OutputTokens
				}
			}
		}

		if err := stdoutScanner.Err(); err != nil {
			done <- fmt.Errorf("stdout scanner error: %w", err)
			return
		}

		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	select {
	case <-timer.C:
		_ = cmd.Process.Kill()
		handler.OnError(fmt.Errorf("timeout after %v", timeout))
		return &executor.Result{
			Success:    false,
			Error:      fmt.Sprintf("timeout after %v", timeout),
			DurationMS: int(time.Since(startTime).Milliseconds()),
			NumTurns:   turnNum,
			SessionID:  sessionID,
			Transcript: transcriptBuf.String(),
		}, nil

	case err := <-done:
		duration := time.Since(startTime)

		if err != nil {
			handler.OnError(err)
			return &executor.Result{
				Success:    false,
				Error:      err.Error(),
				DurationMS: int(duration.Milliseconds()),
				NumTurns:   turnNum,
				SessionID:  sessionID,
				Transcript: transcriptBuf.String(),
			}, nil
		}

		if finalResult == nil {
			return &executor.Result{
				Success:      true,
				Output:       "Session completed",
				DurationMS:   int(duration.Milliseconds()),
				NumTurns:     turnNum,
				SessionID:    sessionID,
				Transcript:   transcriptBuf.String(),
				InputTokens:  totalInputTokens,
				OutputTokens: totalOutputTokens,
				CostUSD:      e.CostModel().CalculateCost(executor.TokenUsage{InputTokens: totalInputTokens, OutputTokens: totalOutputTokens}),
			}, nil
		}

		// Calculate cost using Gemini pricing
		cost := e.calculateCost(finalResult)

		return &executor.Result{
			Success:      !finalResult.IsError && finalResult.Subtype == "success",
			Output:       finalResult.Result,
			Error:        getErrorMessage(finalResult),
			DurationMS:   finalResult.DurationMS,
			NumTurns:     finalResult.NumTurns,
			CostUSD:      cost,
			InputTokens:  getInputTokens(finalResult),
			OutputTokens: getOutputTokens(finalResult),
			SessionID:    sessionID,
			Transcript:   transcriptBuf.String(),
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

func (e *GeminiExecutor) calculateCost(result *geminiHeadlessResult) float64 {
	if result == nil || result.Usage == nil {
		return 0
	}
	return e.CostModel().CalculateCost(executor.TokenUsage{
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	})
}

// geminiHeadlessResult matches Gemini CLI output structure
// Similar to Claude's format for compatibility
type geminiHeadlessResult struct {
	Type         string       `json:"type"`
	Subtype      string       `json:"subtype"`
	IsError      bool         `json:"is_error"`
	Result       string       `json:"result"`
	NumTurns     int          `json:"num_turns"`
	DurationMS   int          `json:"duration_ms"`
	TotalCostUSD float64      `json:"total_cost_usd"`
	SessionID    string       `json:"session_id"`
	Usage        *geminiUsage `json:"usage"`
}

type geminiUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func getErrorMessage(result *geminiHeadlessResult) string {
	if result.IsError {
		return result.Result
	}
	if result.Subtype == "error" || result.Subtype == "timeout" {
		return result.Result
	}
	return ""
}

func getInputTokens(result *geminiHeadlessResult) int {
	if result.Usage != nil {
		return result.Usage.InputTokens
	}
	return 0
}

func getOutputTokens(result *geminiHeadlessResult) int {
	if result.Usage != nil {
		return result.Usage.OutputTokens
	}
	return 0
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
