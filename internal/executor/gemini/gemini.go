// Package gemini provides an Executor implementation for Gemini CLI.
// Updated Dec 2025 to support stream-json output format for real-time streaming.
package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var geminiTracer = telemetry.Tracer("executor.gemini")

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
// Now uses stream-json output format for true NDJSON streaming like Claude
func (e *GeminiExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	// Start OTEL span for Gemini execution
	ctx, span := telemetry.StartSpan(ctx, geminiTracer, "gemini.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "gemini"),
			attribute.String("executor.model", e.model),
			attribute.String("task.workspace", task.Workspace),
			attribute.String("task.directive", telemetry.Truncate(task.Directive, 500)),
		),
	)
	defer span.End()

	// Update handler's context so child spans (turns, tools) are properly nested
	// under this gemini.execute span rather than being siblings
	if ctxHandler, ok := handler.(executor.ContextAwareHandler); ok {
		ctxHandler.SetContext(ctx)
	}

	// Use task ID for hierarchy tracking (Gemini generates its own session ID)
	// We'll capture Gemini's session ID from the init event and store both
	ourTaskID := task.ID
	if ourTaskID == "" {
		ourTaskID = uuid.New().String()
	}
	span.SetAttributes(attribute.String("exec.task_id", ourTaskID))

	// sessionID will be updated from Gemini's init event
	sessionID := ourTaskID // Default to our task ID

	// Build command arguments for Gemini CLI
	// Gemini CLI doesn't support system prompts, so prepend to directive if needed
	directive := task.Directive
	if task.SystemPrompt != "" {
		directive = task.SystemPrompt + "\n\n" + task.Directive
	}

	args := []string{
		directive,                        // Positional prompt (with system prompt prepended if any)
		"--output-format", "stream-json", // Stream NDJSON for real-time events
		"-m", e.getModel(task), // Model selection
		"-y", // Auto-approve all tool uses (YOLO mode)
	}

	// Add working directory if specified
	if task.Workspace != "" {
		args = append(args, "--include-directories", task.Workspace)
	}

	// Resolve gemini path - scan NVM versions newest-first (required for regex /v flag)
	geminiPath := e.geminiPath
	if nvmPath := executor.FindNVMBinary("gemini"); nvmPath != "" {
		geminiPath = nvmPath
	}

	cmd := exec.CommandContext(ctx, geminiPath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	// Debug: print command being executed
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Command: %s -m %s --output-format stream-json\n", geminiPath, e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Workspace: %s\n", task.Workspace)
	}

	// Set up environment using shared builder (M-UNIFIED-AI-CONTROL-PLANE)
	env := executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:                  task,
		SessionID:             sessionID,
		Context:               ctx,
		EnableGeminiTelemetry: true,
	})

	// Ensure matching Node version is on PATH (required for Gemini CLI's regex /v flag)
	if nvmBinDir := executor.FindNVMNodeBinDir("gemini"); nvmBinDir != "" {
		currentPath := os.Getenv("PATH")
		env = executor.UpdateEnvVar(env, "PATH", nvmBinDir+":"+currentPath)
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

	// Set up timeouts: hard ceiling + idle detection (v0.8.1)
	timeout := task.Timeout
	if timeout == 0 {
		timeout = time.Duration(e.timeoutSeconds) * time.Second
	}
	idleTimeout := task.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 3 * time.Minute
	}

	hardTimer := time.NewTimer(timeout)
	defer hardTimer.Stop()
	idleCheck := time.NewTimer(idleTimeout)
	defer idleCheck.Stop()

	// Track last activity time (atomic for goroutine safety)
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	// Parse streaming NDJSON output
	done := make(chan error, 1)
	var finalResult *geminiStreamResult
	var transcriptBuf strings.Builder
	var turnNum int
	var toolCallCount int
	var inputTokens, outputTokens int
	var turnSpan trace.Span // Track current turn's OTEL span
	var stderrBuf strings.Builder

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// Increase buffer size to 1MB to handle large JSON output from Gemini CLI
		// (e.g., tool results with large file contents or base64-encoded images)
		const maxScannerBuffer = 1024 * 1024 // 1MB
		stdoutScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)
		stderrScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)

		// Read stderr in background - capture for diagnostics
		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				stderrBuf.WriteString(line + "\n")
				if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[GEMINI_STDERR] %s\n", line)
				}
			}
		}()

		// Parse NDJSON from stdout
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			if line == "" {
				continue
			}

			// Track activity for idle timeout detection (v0.8.1)
			lastActivity.Store(time.Now().UnixNano())

			// Debug: log raw NDJSON lines to discover actual Gemini CLI event format
			if os.Getenv("DEBUG_AGENT") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI_RAW] %s\n", line)
			}

			var event geminiStreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Failed to parse NDJSON: %v\n", err)
				}
				continue
			}

			switch event.Type {
			case "init":
				// Session initialization - capture Gemini's session ID
				if event.SessionID != "" {
					sessionID = event.SessionID
					// Store both our task_id and Gemini's session_id in the span
					span.SetAttributes(attribute.String("exec.gemini_session_id", event.SessionID))
				}
				turnNum++
				// End previous turn span if still open
				if turnSpan != nil {
					turnSpan.End()
				}
				// Start new turn span as child of gemini.execute
				_, turnSpan = telemetry.StartSpan(ctx, geminiTracer, "exec.turn",
					trace.WithAttributes(
						attribute.Int("turn.number", turnNum),
						attribute.String("session.id", sessionID),
					),
				)
				handler.OnTurnStart(turnNum)
				transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))

			case "message":
				if event.Role == "assistant" {
					// Assistant response text
					handler.OnText(event.Content)
					transcriptBuf.WriteString(event.Content)
				} else if event.Role == "user" {
					// User message (just log it)
					transcriptBuf.WriteString(fmt.Sprintf("[USER] %s\n", event.Content))
				}

			case "tool_use":
				// Tool invocation (Gemini CLI uses "tool_use", not "tool_call")
				toolCallCount++
				handler.OnToolUse(event.ToolName, string(event.Parameters))
				transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", event.ToolName))

			case "tool_result":
				// Tool result - each tool result triggers a new assistant turn
				handler.OnToolResult(event.ToolName, event.Output)
				// Count new turn: tool_result → next assistant response = new conversation turn
				handler.OnTurnEnd(turnNum)
				if turnSpan != nil {
					turnSpan.End()
				}
				turnNum++
				_, turnSpan = telemetry.StartSpan(ctx, geminiTracer, "exec.turn",
					trace.WithAttributes(
						attribute.Int("turn.number", turnNum),
						attribute.String("session.id", sessionID),
					),
				)
				handler.OnTurnStart(turnNum)
				transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))

			case "result":
				// Final result with stats
				finalResult = &geminiStreamResult{
					Status:       event.Status,
					TotalTokens:  event.Stats.TotalTokens,
					InputTokens:  event.Stats.InputTokens,
					OutputTokens: event.Stats.OutputTokens,
					DurationMS:   event.Stats.DurationMS,
					ToolCalls:    event.Stats.ToolCalls,
				}
				inputTokens = event.Stats.InputTokens
				outputTokens = event.Stats.OutputTokens
				handler.OnTurnEnd(turnNum)
				// End turn span
				if turnSpan != nil {
					turnSpan.End()
					turnSpan = nil
				}

			default:
				// Log unrecognized event types to help discover actual Gemini CLI format
				if os.Getenv("DEBUG_AGENT") != "" && event.Type != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Unrecognized event type: %q\n", event.Type)
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

	// Wait for completion, hard timeout, or idle timeout (v0.8.1)
	for {
		select {
		case <-hardTimer.C:
			// Hard ceiling reached — kill unconditionally
			_ = cmd.Process.Kill()
			timeoutErr := fmt.Errorf("timeout after %v (hard ceiling)", timeout)
			handler.OnError(timeoutErr)
			span.RecordError(timeoutErr)
			span.SetStatus(codes.Error, "timeout (hard ceiling)")
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", false),
			)
			return &executor.Result{
				Success:       false,
				Error:         fmt.Sprintf("timeout after %v (hard ceiling)", timeout),
				DurationMS:    int(time.Since(startTime).Milliseconds()),
				NumTurns:      turnNum,
				ToolCallCount: toolCallCount,
				SessionID:     sessionID,
				Transcript:    transcriptBuf.String(),
			}, nil

		case <-idleCheck.C:
			// Check if agent has been idle too long
			last := time.Unix(0, lastActivity.Load())
			idle := time.Since(last)
			if idle >= idleTimeout {
				// Agent is stuck — kill it
				_ = cmd.Process.Kill()
				idleErr := fmt.Errorf("timeout after %v idle (no output for %v, total runtime %v)",
					idleTimeout, idle.Round(time.Second), time.Since(startTime).Round(time.Second))
				handler.OnError(idleErr)
				span.RecordError(idleErr)
				span.SetStatus(codes.Error, "timeout (idle)")
				span.SetAttributes(
					attribute.Int("task.turns", turnNum),
					attribute.Bool("task.success", false),
				)
				return &executor.Result{
					Success:       false,
					Error:         fmt.Sprintf("timeout after %v idle (no output for %v, total runtime %v)", idleTimeout, idle.Round(time.Second), time.Since(startTime).Round(time.Second)),
					DurationMS:    int(time.Since(startTime).Milliseconds()),
					NumTurns:      turnNum,
					ToolCallCount: toolCallCount,
					SessionID:     sessionID,
					Transcript:    transcriptBuf.String(),
				}, nil
			}
			// Activity detected since last check — reset idle timer for remaining time
			remaining := idleTimeout - idle
			idleCheck.Reset(remaining)

		case err := <-done:
			// Normal completion (success or error)
			duration := time.Since(startTime)

			if err != nil {
				handler.OnError(err)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				span.SetAttributes(
					attribute.Int("task.turns", turnNum),
					attribute.Bool("task.success", false),
				)
				// Include stderr in error for diagnostics
				errMsg := err.Error()
				if stderrContent := stderrBuf.String(); stderrContent != "" {
					errMsg = fmt.Sprintf("%s\nstderr: %s", errMsg, stderrContent)
				}
				return &executor.Result{
					Success:       false,
					Error:         errMsg,
					DurationMS:    int(duration.Milliseconds()),
					NumTurns:      turnNum,
					ToolCallCount: toolCallCount,
					SessionID:     sessionID,
					Transcript:    transcriptBuf.String(),
				}, nil
			}

			// Calculate cost
			cost := e.CostModel().CalculateCost(executor.TokenUsage{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
			})

			success := true
			if finalResult != nil && finalResult.Status != "success" {
				success = false
			}

			// Record span attributes
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", success),
				attribute.Int("task.duration_ms", int(duration.Milliseconds())),
				attribute.Int64("task.tokens_in", int64(inputTokens)),
				attribute.Int64("task.tokens_out", int64(outputTokens)),
				attribute.Float64("task.cost_usd", cost),
			)
			if !success {
				span.SetStatus(codes.Error, "task failed")
			} else {
				span.SetStatus(codes.Ok, "task completed successfully")
			}

			return &executor.Result{
				Success:       success,
				Output:        transcriptBuf.String(),
				DurationMS:    int(duration.Milliseconds()),
				NumTurns:      turnNum,
				ToolCallCount: toolCallCount,
				CostUSD:       cost,
				InputTokens:   inputTokens,
				OutputTokens:  outputTokens,
				SessionID:     sessionID,
				Transcript:    transcriptBuf.String(),
			}, nil
		}
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
	// Check if gemini binary exists - scan NVM versions newest-first
	geminiPath := e.geminiPath
	if nvmPath := executor.FindNVMBinary("gemini"); nvmPath != "" {
		geminiPath = nvmPath
	}

	_, err := exec.LookPath(geminiPath)
	if err != nil {
		// Also try the resolved path directly (LookPath fails for absolute paths on some systems)
		if _, statErr := os.Stat(geminiPath); statErr != nil {
			return fmt.Errorf("gemini CLI not found: %w (install with: npm i -g @google/gemini-cli)", err)
		}
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

// geminiStreamEvent represents a single NDJSON event from stream-json output
type geminiStreamEvent struct {
	Type       string            `json:"type"` // init, message, tool_use, tool_result, result
	Timestamp  string            `json:"timestamp"`
	SessionID  string            `json:"session_id,omitempty"`
	Model      string            `json:"model,omitempty"`
	Role       string            `json:"role,omitempty"` // user, assistant
	Content    string            `json:"content,omitempty"`
	Delta      bool              `json:"delta,omitempty"` // true if streaming delta
	ToolName   string            `json:"tool_name,omitempty"`
	ToolID     string            `json:"tool_id,omitempty"`
	Parameters json.RawMessage   `json:"parameters,omitempty"`
	Output     string            `json:"output,omitempty"`
	Status     string            `json:"status,omitempty"` // success, error
	Stats      geminiStreamStats `json:"stats,omitempty"`
}

// geminiStreamStats contains stats from the final result event
type geminiStreamStats struct {
	TotalTokens  int `json:"total_tokens"`
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	DurationMS   int `json:"duration_ms"`
	ToolCalls    int `json:"tool_calls"`
}

// geminiStreamResult is parsed from the final "result" event
type geminiStreamResult struct {
	Status       string
	TotalTokens  int
	InputTokens  int
	OutputTokens int
	DurationMS   int
	ToolCalls    int
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
