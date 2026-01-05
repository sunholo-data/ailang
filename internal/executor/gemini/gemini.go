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
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/executor"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var geminiTracer = otel.Tracer("executor.gemini")

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
	ctx, span := geminiTracer.Start(ctx, "gemini.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "gemini"),
			attribute.String("executor.model", e.model),
			attribute.String("task.workspace", task.Workspace),
		),
	)
	defer span.End()

	sessionID := uuid.New().String()
	span.SetAttributes(attribute.String("session.id", sessionID))

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
		fmt.Fprintf(os.Stderr, "[DEBUG_GEMINI] Command: %s -m %s --output-format stream-json\n", geminiPath, e.getModel(task))
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
	node22Path := filepath.Join(homeDir, ".nvm", "versions", "node", "v22.20.0", "bin")
	if _, err := os.Stat(node22Path); err == nil {
		currentPath := os.Getenv("PATH")
		env = updateEnvVar(env, "PATH", node22Path+":"+currentPath)
	}

	// Inject W3C trace context for distributed tracing
	// This enables ailang run commands spawned by Gemini to link back to this trace
	env = telemetry.InjectTraceContext(ctx, env)

	// Inject correlation IDs for fallback linking
	env = telemetry.InjectCorrelationIDs(env, task.ID, sessionID)

	// Enable Gemini CLI telemetry for trace export
	// Gemini CLI supports full traces (unlike Claude Code which only does metrics/events)
	env = append(env, "GEMINI_TELEMETRY_ENABLED=true")

	// Build resource attributes for trace linking (M-TASK-HIERARCHY)
	// Merges existing attributes from environment with task-specific attributes
	resourceAttrs := buildResourceAttributes(task, sessionID)
	env = append(env, fmt.Sprintf("OTEL_RESOURCE_ATTRIBUTES=%s", resourceAttrs))

	// For GCP export, check OTLP_GOOGLE_CLOUD_PROJECT first (Gemini CLI standard), fallback to GOOGLE_CLOUD_PROJECT
	project := os.Getenv("OTLP_GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	// Set telemetry target based on available configuration
	// Priority: GEMINI_TELEMETRY_TARGET env > GCP if project is set > default "gcp"
	if target := os.Getenv("GEMINI_TELEMETRY_TARGET"); target != "" {
		env = append(env, fmt.Sprintf("GEMINI_TELEMETRY_TARGET=%s", target))
	} else if project != "" {
		env = append(env, "GEMINI_TELEMETRY_TARGET=gcp")
	}

	// Pass both project env vars for compatibility
	if project != "" {
		env = append(env, fmt.Sprintf("GOOGLE_CLOUD_PROJECT=%s", project))
		env = append(env, fmt.Sprintf("OTLP_GOOGLE_CLOUD_PROJECT=%s", project))
	}

	// Configure OTEL exporter for trace collection
	// Priority: parent env > default to local observatory server
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// Default to local observatory for unified trace collection
		endpoint = "http://localhost:1957"
	}
	env = append(env, fmt.Sprintf("OTEL_EXPORTER_OTLP_ENDPOINT=%s", endpoint))

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

	// Parse streaming NDJSON output
	done := make(chan error, 1)
	var finalResult *geminiStreamResult
	var transcriptBuf strings.Builder
	var turnNum int
	var inputTokens, outputTokens int

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// Read stderr in background (debug/startup output)
		go func() {
			for stderrScanner.Scan() {
				// Discard stderr (startup messages, etc.)
				_ = stderrScanner.Text()
			}
		}()

		// Parse NDJSON from stdout
		for stdoutScanner.Scan() {
			line := stdoutScanner.Text()
			if line == "" {
				continue
			}

			var event geminiStreamEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				continue
			}

			switch event.Type {
			case "init":
				// Session initialization
				if event.SessionID != "" {
					sessionID = event.SessionID
				}
				turnNum++
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

			case "tool_call":
				// Tool invocation
				handler.OnToolUse(event.ToolName, event.ToolInput)
				transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", event.ToolName))

			case "tool_result":
				// Tool result
				handler.OnToolResult(event.ToolName, event.ToolOutput)

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
			Success:      success,
			Output:       transcriptBuf.String(),
			DurationMS:   int(duration.Milliseconds()),
			NumTurns:     turnNum,
			CostUSD:      cost,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
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

// geminiStreamEvent represents a single NDJSON event from stream-json output
type geminiStreamEvent struct {
	Type       string            `json:"type"` // init, message, tool_call, tool_result, result
	Timestamp  string            `json:"timestamp"`
	SessionID  string            `json:"session_id,omitempty"`
	Model      string            `json:"model,omitempty"`
	Role       string            `json:"role,omitempty"` // user, assistant
	Content    string            `json:"content,omitempty"`
	Delta      bool              `json:"delta,omitempty"` // true if streaming delta
	ToolName   string            `json:"tool_name,omitempty"`
	ToolInput  string            `json:"tool_input,omitempty"`
	ToolOutput string            `json:"tool_output,omitempty"`
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

// buildResourceAttributes creates OTEL_RESOURCE_ATTRIBUTES value.
// Merges existing attributes from environment with task-specific attributes.
// Priority: existing env attrs + task Metadata + default attrs.
func buildResourceAttributes(task *executor.Task, sessionID string) string {
	attrs := make(map[string]string)

	// 1. Start with existing environment attributes (preserve user settings)
	if existing := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); existing != "" {
		for _, pair := range strings.Split(existing, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				attrs[parts[0]] = parts[1]
			}
		}
	}

	// 2. Add task Metadata attributes (from Observatory context via coordinator)
	if task.Metadata != nil {
		for k, v := range task.Metadata {
			if strings.HasPrefix(k, "ailang.") && v != "" {
				attrs[k] = v
			}
		}
	}

	// 3. Add default attributes (lowest priority, don't overwrite)
	if _, exists := attrs["ailang.task_id"]; !exists && task.ID != "" {
		attrs["ailang.task_id"] = task.ID
	}
	if _, exists := attrs["ailang.session_id"]; !exists && sessionID != "" {
		attrs["ailang.session_id"] = sessionID
	}
	if _, exists := attrs["ailang.source"]; !exists {
		attrs["ailang.source"] = "coordinator"
	}

	// Build final attribute string
	var parts []string
	for k, v := range attrs {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}
