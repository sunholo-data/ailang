// Package claude provides an Executor implementation for Claude Code CLI.
package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var claudeTracer = otel.Tracer("executor.claude")

// ClaudeExecutor executes tasks using Claude Code CLI
type ClaudeExecutor struct {
	claudePath     string
	model          string
	allowedTools   []string
	permissionMode string
	timeoutSeconds int
}

// New creates a new ClaudeExecutor
func New(cfg *executor.Config) (*ClaudeExecutor, error) {
	claudePath := cfg.ClaudePath
	if claudePath == "" {
		claudePath = "claude"
	}

	model := cfg.ClaudeModel
	if model == "" {
		model = "haiku"
	}

	tools := cfg.ClaudeTools
	if len(tools) == 0 {
		tools = []string{"Bash", "Read", "Write", "Edit", "Grep", "Glob"}
	}

	permission := cfg.ClaudePermission
	if permission == "" {
		permission = "bypassPermissions"
	}

	return &ClaudeExecutor{
		claudePath:     claudePath,
		model:          model,
		allowedTools:   tools,
		permissionMode: permission,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier
func (e *ClaudeExecutor) Name() string {
	return "claude"
}

// Execute runs a task and returns the result
func (e *ClaudeExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks
func (e *ClaudeExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	// Start OTEL span for Claude execution
	ctx, span := claudeTracer.Start(ctx, "claude.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "claude"),
			attribute.String("executor.model", e.model),
			attribute.String("task.workspace", task.Workspace),
		),
	)
	defer span.End()

	// Update handler's context so child spans (turns, tools) are properly nested
	// under this claude.execute span rather than being siblings
	if ctxHandler, ok := handler.(executor.ContextAwareHandler); ok {
		ctxHandler.SetContext(ctx)
	}

	// Use task ID as session ID for unified hierarchy tracking
	// This allows: 1) consistent span correlation, 2) Claude session resume using task ID
	// IMPORTANT: Claude Code CLI requires session IDs to be valid UUIDs
	sessionID := task.ID
	if sessionID == "" {
		// Fallback if task has no ID (shouldn't happen in normal use)
		sessionID = uuid.New().String()
	} else if !isValidUUID(sessionID) {
		// Task ID is not a valid UUID (e.g., "exec-test", "task-123")
		// Generate a UUID for Claude's session, but keep task.ID for hierarchy tracking
		sessionID = uuid.New().String()
	}
	span.SetAttributes(attribute.String("session.id", sessionID))

	// Build command arguments
	args := []string{
		"-p", task.Directive,
		"--output-format", "stream-json",
		"--include-partial-messages",
		"--verbose",
		"--permission-mode", e.permissionMode,
		"--model", e.getModel(task),
		"--session-id", sessionID,
	}

	if task.SystemPrompt != "" {
		args = append([]string{"--system-prompt", task.SystemPrompt}, args...)
	}

	if task.Workspace != "" {
		args = append(args, "--add-dir", task.Workspace)
	}

	// Use task-specific tools if specified, otherwise fall back to executor config
	tools := e.allowedTools
	if len(task.AllowedTools) > 0 {
		tools = task.AllowedTools
	}
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}

	cmd := exec.CommandContext(ctx, e.claudePath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	// Set up environment using shared builder (M-UNIFIED-AI-CONTROL-PLANE)
	cmd.Env = executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:                  task,
		SessionID:             sessionID,
		Context:               ctx,
		EnableClaudeTelemetry: true,
	})

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
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Set up timeout
	timeout := task.Timeout
	if timeout == 0 {
		timeout = time.Duration(e.timeoutSeconds) * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Parse streaming output
	done := make(chan error, 1)
	var finalResult *claudeHeadlessResult
	var transcriptBuf strings.Builder
	var turnNum int
	var turnSpan trace.Span // Track current turn's OTEL span

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// Read stderr in background
		go func() {
			for stderrScanner.Scan() {
				// Discard stderr for now
				_ = stderrScanner.Text()
			}
		}()

		// Parse NDJSON from stdout
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
					// End previous turn span if still open
					if turnSpan != nil {
						turnSpan.End()
					}
					// Start new turn span as child of claude.execute
					_, turnSpan = claudeTracer.Start(ctx, "exec.turn",
						trace.WithAttributes(
							attribute.Int("turn.number", turnNum),
							attribute.String("session.id", sessionID),
						),
					)
					handler.OnTurnStart(turnNum)
					transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))

				case "content_block_start":
					contentBlock, _ := streamEvent["content_block"].(map[string]interface{})
					if contentBlock != nil {
						if blockType, _ := contentBlock["type"].(string); blockType == "tool_use" {
							toolName, _ := contentBlock["name"].(string)
							// Extract tool input if available (may be in initial block or come via delta)
							toolInput := ""
							if input, ok := contentBlock["input"]; ok {
								if inputMap, ok := input.(map[string]interface{}); ok {
									inputBytes, _ := json.Marshal(inputMap)
									toolInput = string(inputBytes)
								} else if inputStr, ok := input.(string); ok {
									toolInput = inputStr
								}
							}
							handler.OnToolUse(toolName, toolInput)
							transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", toolName))
						}
					}

				case "content_block_delta":
					delta, _ := streamEvent["delta"].(map[string]interface{})
					if delta != nil {
						deltaType, _ := delta["type"].(string)
						switch deltaType {
						case "text_delta":
							text, _ := delta["text"].(string)
							handler.OnText(text)
							transcriptBuf.WriteString(text)
						case "input_json_delta":
							// Tool input streaming - accumulate but we already called OnToolUse
							// The input will be visible in the initial tool_use block
							_ = delta["partial_json"]
						}
					}

				case "content_block_stop":
					// Tool block completed - could signal OnToolResult here if we had a result
					// Claude Code sends tool results as separate user messages
					_ = streamEvent["index"]

				case "message_stop":
					handler.OnTurnEnd(turnNum)
					// End turn span
					if turnSpan != nil {
						turnSpan.End()
						turnSpan = nil
					}
				}

			case "result":
				if err := json.Unmarshal([]byte(line), &finalResult); err != nil {
					// Cleanup turn span before early return
					if turnSpan != nil {
						turnSpan.End()
					}
					done <- fmt.Errorf("failed to parse final result: %w", err)
					return
				}
			}
		}

		// Cleanup any open turn span
		if turnSpan != nil {
			turnSpan.End()
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
		timeoutErr := fmt.Errorf("timeout after %v", timeout)
		handler.OnError(timeoutErr)
		span.RecordError(timeoutErr)
		span.SetStatus(codes.Error, "timeout")
		span.SetAttributes(
			attribute.Int("task.turns", turnNum),
			attribute.Bool("task.success", false),
		)
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
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", false),
			)
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
			span.SetStatus(codes.Ok, "session completed")
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", true),
				attribute.Int("task.duration_ms", int(duration.Milliseconds())),
			)
			return &executor.Result{
				Success:    true,
				Output:     "Session completed",
				DurationMS: int(duration.Milliseconds()),
				NumTurns:   turnNum,
				SessionID:  sessionID,
				Transcript: transcriptBuf.String(),
			}, nil
		}

		success := !finalResult.IsError && finalResult.Subtype == "success"
		span.SetAttributes(
			attribute.Int("task.turns", finalResult.NumTurns),
			attribute.Bool("task.success", success),
			attribute.Int("task.duration_ms", finalResult.DurationMS),
			attribute.Int64("task.tokens_in", int64(finalResult.Usage.InputTokens)),
			attribute.Int64("task.tokens_out", int64(finalResult.Usage.OutputTokens)),
			attribute.Float64("task.cost_usd", finalResult.TotalCostUSD),
		)
		if !success {
			span.SetStatus(codes.Error, getErrorMessage(finalResult))
		} else {
			span.SetStatus(codes.Ok, "task completed successfully")
		}

		return &executor.Result{
			Success:                  success,
			Output:                   finalResult.Result,
			Error:                    getErrorMessage(finalResult),
			DurationMS:               finalResult.DurationMS,
			NumTurns:                 finalResult.NumTurns,
			CostUSD:                  finalResult.TotalCostUSD,
			InputTokens:              finalResult.Usage.InputTokens,
			OutputTokens:             finalResult.Usage.OutputTokens,
			CacheReadInputTokens:     finalResult.Usage.CacheReadInputTokens,
			CacheCreationInputTokens: finalResult.Usage.CacheCreationInputTokens,
			SessionID:                sessionID,
			Transcript:               transcriptBuf.String(),
		}, nil
	}
}

// Capabilities returns the list of features this executor supports
func (e *ClaudeExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapToolControl,
		executor.CapSessionResume,
		executor.CapLocalWorkspace,
	}
}

// CostModel returns pricing information for cost calculations
func (e *ClaudeExecutor) CostModel() *executor.CostModel {
	// Default to Haiku pricing
	return &executor.CostModel{
		ProviderName:    "anthropic",
		InputTokenCost:  0.001,  // $1.00 per 1M
		OutputTokenCost: 0.005,  // $5.00 per 1M
		CacheReadCost:   0.0001, // $0.10 per 1M
	}
}

// HealthCheck verifies the executor is configured and accessible
func (e *ClaudeExecutor) HealthCheck(ctx context.Context) error {
	// Check if claude binary exists
	_, err := exec.LookPath(e.claudePath)
	if err != nil {
		return fmt.Errorf("claude CLI not found: %w (install from https://claude.ai/code)", err)
	}
	return nil
}

// Close releases any resources held by the executor
func (e *ClaudeExecutor) Close() error {
	return nil
}

func (e *ClaudeExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// claudeHeadlessResult matches Claude CLI output structure
type claudeHeadlessResult struct {
	Type         string      `json:"type"`
	Subtype      string      `json:"subtype"`
	IsError      bool        `json:"is_error"`
	Result       string      `json:"result"`
	NumTurns     int         `json:"num_turns"`
	DurationMS   int         `json:"duration_ms"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	SessionID    string      `json:"session_id"`
	Usage        claudeUsage `json:"usage"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func getErrorMessage(result *claudeHeadlessResult) string {
	if result.IsError {
		return result.Result
	}
	if result.Subtype == "error" || result.Subtype == "timeout" {
		return result.Result
	}
	return ""
}

// isValidUUID checks if a string is a valid UUID format
// Claude Code CLI requires session IDs to be valid UUIDs
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// Register registers the Claude executor with the global factory
func Register() {
	executor.GlobalFactory().Register("claude", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
