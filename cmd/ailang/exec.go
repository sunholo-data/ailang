// Package main provides the ailang exec command for unified AI execution.
// This is the single entry point for all AI operations, supporting both CLI-based
// (Claude Code, Gemini CLI) and API-based (OpenAI, Anthropic, Ollama) providers.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/ai"
	"github.com/sunholo/ailang/internal/ai/anthropic"
	"github.com/sunholo/ailang/internal/ai/gemini"
	"github.com/sunholo/ailang/internal/ai/ollama"
	"github.com/sunholo/ailang/internal/ai/openai"
	"github.com/sunholo/ailang/internal/executor"
	_ "github.com/sunholo/ailang/internal/executor/claude"
	_ "github.com/sunholo/ailang/internal/executor/gemini"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var execTracer = otel.Tracer("ailang.exec")

// ExecEvent represents a streaming event in NDJSON format
type ExecEvent struct {
	Type      string      `json:"type"`
	Provider  string      `json:"provider,omitempty"`
	Model     string      `json:"model,omitempty"`
	TaskID    string      `json:"task_id,omitempty"`
	Turn      int         `json:"turn,omitempty"`
	Content   string      `json:"content,omitempty"`
	Tool      string      `json:"tool,omitempty"`
	Input     string      `json:"input,omitempty"`
	Output    string      `json:"output,omitempty"`
	Success   bool        `json:"success,omitempty"`
	Error     string      `json:"error,omitempty"`
	Duration  int         `json:"duration_ms,omitempty"`
	TokensIn  int         `json:"tokens_in,omitempty"`
	TokensOut int         `json:"tokens_out,omitempty"`
	CostUSD   float64     `json:"cost_usd,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// emitEvent writes an NDJSON event to stdout
func emitEvent(event ExecEvent) {
	data, _ := json.Marshal(event)
	fmt.Println(string(data))
}

// runExec implements the unified AI execution command
func runExec() {
	fs := flag.NewFlagSet("exec", flag.ExitOnError)

	// Core flags
	// Default workspace to current directory so Claude/Gemini have file access
	cwd, _ := os.Getwd()
	workspace := fs.String("workspace", cwd, "Working directory for the task (default: current directory)")
	model := fs.String("model", "", "Model to use (provider-specific, e.g., haiku, gemini-2.5-flash)")
	timeout := fs.Duration("timeout", 10*time.Minute, "Execution timeout")
	taskID := fs.String("task-id", "", "Task ID for tracing and hierarchy")
	parentTaskID := fs.String("parent-task-id", "", "Parent task ID for hierarchy linking")
	systemPrompt := fs.String("system-prompt", "", "System prompt to prepend")

	// Mode flags
	apiOnly := fs.Bool("api-only", false, "Use API provider instead of CLI executor (no file editing)")
	registerTask := fs.Bool("register-task", false, "Register task in Observatory")
	dryRun := fs.Bool("dry-run", false, "Validate without executing")

	// Output flags
	streamJSON := fs.Bool("stream-json", true, "Output NDJSON streaming events")
	quiet := fs.Bool("quiet", false, "Suppress streaming output, only show final result")
	jsonOutput := fs.Bool("json", false, "Output result as single JSON object (for programmatic use)")

	// Parse arguments (normalize flags first)
	args := flag.Args()[1:] // Skip "exec" command
	args = normalizeArgsForFlags(args, []string{
		"workspace", "model", "timeout", "task-id", "parent-task-id",
		"system-prompt", "api-only", "register-task", "dry-run",
		"stream-json", "quiet", "json",
	})

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Validate arguments
	if fs.NArg() < 1 {
		printExecHelp()
		os.Exit(1)
	}

	provider := strings.ToLower(fs.Arg(0))
	directive := ""
	if fs.NArg() >= 2 {
		directive = strings.Join(fs.Args()[1:], " ")
	}

	if directive == "" {
		fmt.Fprintf(os.Stderr, "%s: directive required\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Usage: ailang exec %s \"your directive here\"\n", provider)
		os.Exit(1)
	}

	// Validate provider
	validProviders := map[string]bool{
		"claude": true, "gemini": true, "openai": true,
		"anthropic": true, "ollama": true,
	}
	if !validProviders[provider] {
		fmt.Fprintf(os.Stderr, "%s: unknown provider %q\n", red("Error"), provider)
		fmt.Fprintf(os.Stderr, "Valid providers: claude, gemini, openai, anthropic, ollama\n")
		os.Exit(1)
	}

	// Initialize telemetry
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-exec")
	if err != nil {
		// Non-fatal - continue without telemetry
		if !*quiet {
			fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
		}
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract parent trace context from environment
	ctx = telemetry.ExtractTraceContext(ctx)

	// Generate task ID if not provided - use UUID for unified session tracking
	// This UUID is used as both the task ID (for hierarchy) and session ID (for Claude Code CLI)
	if *taskID == "" {
		*taskID = uuid.New().String()
	}

	// Inherit parent task from environment if not explicitly provided
	// This enables automatic hierarchy linking when one exec spawns another
	if *parentTaskID == "" {
		*parentTaskID = os.Getenv("AILANG_PARENT_TASK_ID")
	}

	// If still no parent task, use generic root marker for analytics
	// This ensures all tasks appear in Observatory hierarchy views
	if *parentTaskID == "" {
		*parentTaskID = "root"
	}

	// Export current task ID so child processes inherit the hierarchy
	os.Setenv("AILANG_PARENT_TASK_ID", *taskID)

	// Create root span for this execution
	ctx, span := execTracer.Start(ctx, "ailang.exec",
		trace.WithAttributes(
			attribute.String("exec.provider", provider),
			attribute.String("exec.task_id", *taskID),
			attribute.String("exec.workspace", *workspace),
			attribute.Bool("exec.api_only", *apiOnly),
		),
	)
	defer span.End()

	if *parentTaskID != "" {
		span.SetAttributes(attribute.String("exec.parent_task_id", *parentTaskID))
	}

	// Emit start event (suppressed when --json is used for clean programmatic output)
	if *streamJSON && !*quiet && !*jsonOutput {
		emitEvent(ExecEvent{
			Type:     "exec_start",
			Provider: provider,
			Model:    *model,
			TaskID:   *taskID,
		})
	}

	// Dry run mode - just validate and exit
	if *dryRun {
		fmt.Printf("Dry run: would execute %s with directive: %s\n", provider, truncateString(directive, 50))
		span.SetStatus(codes.Ok, "dry run")
		if *streamJSON && !*quiet && !*jsonOutput {
			emitEvent(ExecEvent{
				Type:    "exec_end",
				Success: true,
			})
		}
		return
	}

	// Execute based on mode
	// Suppress streaming when --json is used for clean programmatic output
	streamEvents := *streamJSON && !*quiet && !*jsonOutput
	var result *executor.Result
	if *apiOnly {
		result, err = executeAPI(ctx, provider, directive, *model, *systemPrompt, *timeout, streamEvents)
	} else {
		result, err = executeCLI(ctx, provider, directive, *workspace, *model, *systemPrompt, *taskID, *timeout, streamEvents)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if *streamJSON && !*quiet && !*jsonOutput {
			emitEvent(ExecEvent{
				Type:  "exec_end",
				Error: err.Error(),
			})
		}
		// For JSON output, emit error as JSON
		if *jsonOutput {
			output := map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
			jsonBytes, _ := json.Marshal(output)
			fmt.Println(string(jsonBytes))
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Record result in span
	span.SetAttributes(
		attribute.Bool("exec.success", result.Success),
		attribute.Int("exec.duration_ms", result.DurationMS),
		attribute.Int("exec.num_turns", result.NumTurns),
		attribute.Int("exec.input_tokens", result.InputTokens),
		attribute.Int("exec.output_tokens", result.OutputTokens),
		attribute.Float64("exec.cost_usd", result.CostUSD),
	)

	if result.Success {
		span.SetStatus(codes.Ok, "execution completed")
	} else {
		span.SetStatus(codes.Error, result.Error)
	}

	// Emit end event
	if *streamJSON && !*quiet && !*jsonOutput {
		emitEvent(ExecEvent{
			Type:      "exec_end",
			Success:   result.Success,
			Error:     result.Error,
			Duration:  result.DurationMS,
			TokensIn:  result.InputTokens,
			TokensOut: result.OutputTokens,
			CostUSD:   result.CostUSD,
		})
	}

	// Single JSON output (for programmatic use by eval harness, etc.)
	if *jsonOutput {
		output := map[string]interface{}{
			"success":       result.Success,
			"output":        result.Output,
			"error":         result.Error,
			"input_tokens":  result.InputTokens,
			"output_tokens": result.OutputTokens,
			"cost_usd":      result.CostUSD,
			"duration_ms":   result.DurationMS,
			"num_turns":     result.NumTurns,
		}
		jsonBytes, _ := json.Marshal(output)
		fmt.Println(string(jsonBytes))
	}

	// Register task in Observatory if requested
	if *registerTask { //nolint:staticcheck // TODO: Implement Observatory task registration in M3
		_ = parentTaskID // Used when registration is implemented
	}

	// Exit with error code if execution failed
	if !result.Success {
		os.Exit(1)
	}
}

// executeCLI uses the CLI executor (Claude Code, Gemini CLI)
func executeCLI(ctx context.Context, provider, directive, workspace, model, systemPrompt, taskID string, timeout time.Duration, streamJSON bool) (*executor.Result, error) {
	// Get executor from factory
	exec, err := executor.GlobalFactory().GetExecutor(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get executor for %s: %w", provider, err)
	}

	// Build task
	task := &executor.Task{
		ID:           taskID,
		Directive:    directive,
		SystemPrompt: systemPrompt,
		Workspace:    workspace,
		Timeout:      timeout,
		Model:        model,
	}

	// Create spanning event handler that creates child spans for hierarchy tracking
	handler := newSpanningEventHandler(ctx, taskID, streamJSON)

	// Execute with streaming
	return exec.ExecuteStreaming(ctx, task, handler)
}

// executeAPI uses the API provider directly (no file editing)
func executeAPI(ctx context.Context, provider, directive, model, systemPrompt string, timeout time.Duration, streamJSON bool) (*executor.Result, error) {
	// Create API client based on provider
	var client ai.Provider
	var err error

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable required")
		}
		client = openai.NewClient(apiKey)
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable required")
		}
		client = anthropic.NewClient(apiKey)
	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY environment variable required")
		}
		client = gemini.NewClient(apiKey)
	case "ollama":
		endpoint := os.Getenv("OLLAMA_HOST")
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		var ollamaErr error
		client, ollamaErr = ollama.NewClient(ollama.WithEndpoint(endpoint))
		if ollamaErr != nil {
			return nil, fmt.Errorf("failed to create ollama client: %w", ollamaErr)
		}
	default:
		return nil, fmt.Errorf("API mode not supported for provider %s (use CLI mode)", provider)
	}

	// Build request
	req := &ai.Request{
		Model:        model,
		SystemPrompt: systemPrompt,
		UserPrompt:   directive,
	}

	// Execute
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := client.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	duration := time.Since(startTime)

	// Emit text event if streaming
	if streamJSON && resp.Text != "" {
		emitEvent(ExecEvent{
			Type:    "text",
			Content: resp.Text,
		})
	}

	// Convert to executor.Result format
	return &executor.Result{
		Success:      true,
		Output:       resp.Text,
		DurationMS:   int(duration.Milliseconds()),
		NumTurns:     1,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      0, // API providers don't return cost directly
	}, nil
}

// spanningEventHandler creates OTEL child spans from streaming events
// for hierarchical tracing in the dashboard while also emitting NDJSON
type spanningEventHandler struct {
	ctx         context.Context
	tracer      trace.Tracer
	taskID      string
	streamJSON  bool
	currentTurn int
	turnSpan    trace.Span
	toolSpans   map[string]trace.Span // tool name -> active span
}

// newSpanningEventHandler creates a handler that creates child spans
func newSpanningEventHandler(ctx context.Context, taskID string, streamJSON bool) *spanningEventHandler {
	return &spanningEventHandler{
		ctx:        ctx,
		tracer:     execTracer,
		taskID:     taskID,
		streamJSON: streamJSON,
		toolSpans:  make(map[string]trace.Span),
	}
}

// SetContext updates the handler's context for proper span hierarchy.
// Called by the executor after creating its span, so turn/tool spans
// become children of the executor's span rather than siblings.
func (h *spanningEventHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *spanningEventHandler) OnTurnStart(turnNum int) {
	h.currentTurn = turnNum
	// Create child span for this turn
	_, h.turnSpan = h.tracer.Start(h.ctx, "exec.turn",
		trace.WithAttributes(
			attribute.Int("turn.number", turnNum),
			attribute.String("exec.task_id", h.taskID),
		),
	)
	// Also emit NDJSON for backward compatibility
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type: "turn_start",
			Turn: turnNum,
		})
	}
}

func (h *spanningEventHandler) OnText(text string) {
	// Text is recorded on turn span as an event, not a separate span
	if h.turnSpan != nil {
		h.turnSpan.AddEvent("text", trace.WithAttributes(
			attribute.String("text.content", truncateString(text, 500)),
		))
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:    "text",
			Content: text,
		})
	}
}

func (h *spanningEventHandler) OnToolUse(toolName, input string) {
	// Create child span for tool use (child of current turn)
	_, toolSpan := h.tracer.Start(h.ctx, "exec.tool_use",
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
			attribute.String("tool.input", truncateString(input, 1000)),
			attribute.String("exec.task_id", h.taskID),
			attribute.Int("turn.number", h.currentTurn),
		),
	)
	h.toolSpans[toolName] = toolSpan

	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:  "tool_use",
			Tool:  toolName,
			Input: input,
		})
	}
}

func (h *spanningEventHandler) OnToolResult(toolName, output string) {
	// End the matching tool span with the result
	if span, ok := h.toolSpans[toolName]; ok {
		span.SetAttributes(attribute.String("tool.output", truncateString(output, 1000)))
		span.End()
		delete(h.toolSpans, toolName)
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:   "tool_result",
			Tool:   toolName,
			Output: output,
		})
	}
}

func (h *spanningEventHandler) OnTurnEnd(turnNum int) {
	// End any remaining tool spans (shouldn't happen normally)
	for name, span := range h.toolSpans {
		span.SetAttributes(attribute.Bool("tool.incomplete", true))
		span.End()
		delete(h.toolSpans, name)
	}
	// End turn span
	if h.turnSpan != nil {
		h.turnSpan.End()
		h.turnSpan = nil
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type: "turn_end",
			Turn: turnNum,
		})
	}
}

func (h *spanningEventHandler) OnError(err error) {
	// Record error on turn span if active
	if h.turnSpan != nil {
		h.turnSpan.RecordError(err)
		h.turnSpan.SetStatus(codes.Error, err.Error())
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:  "error",
			Error: err.Error(),
		})
	}
}

// printExecHelp prints help for the exec command
func printExecHelp() {
	fmt.Print(`Usage: ailang exec <provider> "directive" [options]

Execute an AI task using the specified provider.

Providers:
  claude      Claude Code CLI (agentic coding with file editing)
  gemini      Gemini CLI (agentic coding with file editing)
  openai      OpenAI API (text generation only, use --api-only)
  anthropic   Anthropic API (text generation only, use --api-only)
  ollama      Ollama (local models, text generation only, use --api-only)

Options:
  --workspace DIR       Working directory for the task
  --model NAME          Model to use (provider-specific)
  --timeout DURATION    Execution timeout (default: 10m)
  --task-id ID          Task ID for tracing and hierarchy
  --parent-task-id ID   Parent task ID for hierarchy linking
  --system-prompt TEXT  System prompt to prepend

Mode Options:
  --api-only            Use API provider instead of CLI (no file editing)
  --register-task       Register task in Observatory
  --dry-run             Validate without executing

Output Options:
  --stream-json         Output NDJSON streaming events (default: true)
  --quiet               Suppress streaming output

Examples:
  # Run Claude Code to fix a bug
  ailang exec claude "Fix the null pointer in parser.go" --workspace /repo

  # Run Gemini CLI with specific model
  ailang exec gemini "Add tests for the login function" --model gemini-2.5-pro

  # Use OpenAI API for text generation
  ailang exec openai "Explain recursion" --api-only

  # Execute with parent task context (from coordinator)
  ailang exec claude "Implement feature" --task-id=task_123 --parent-task-id=task_parent
`)
}
