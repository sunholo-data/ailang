// Package codex provides an Executor implementation for OpenAI Codex CLI.
//
// Codex CLI emits a different NDJSON schema than Claude/Gemini: events are
// flat records with top-level "type":"message", "text", and "tokens_used"
// fields (see internal/executor/codex_compat_test.go for the compatibility
// analysis). A dedicated parser is required.
package codex

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
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var codexTracer = telemetry.Tracer("executor.codex")

// CodexExecutor executes tasks using OpenAI Codex CLI.
type CodexExecutor struct {
	codexPath      string
	model          string
	timeoutSeconds int
}

// New creates a new CodexExecutor.
func New(cfg *executor.Config) (*CodexExecutor, error) {
	codexPath := cfg.CodexPath
	if codexPath == "" {
		codexPath = "codex"
	}

	model := cfg.CodexModel
	if model == "" {
		model = "gpt-5-codex"
	}

	return &CodexExecutor{
		codexPath:      codexPath,
		model:          model,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier.
func (e *CodexExecutor) Name() string {
	return "codex"
}

// Execute runs a task and returns the result.
func (e *CodexExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks, parsing the
// Codex NDJSON stream into normalized executor events.
func (e *CodexExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	ctx, span := telemetry.StartSpan(ctx, codexTracer, "codex.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "codex"),
			attribute.String("executor.model", e.getModel(task)),
			attribute.String("task.workspace", task.Workspace),
			attribute.String("task.directive", telemetry.Truncate(task.Directive, 500)),
		),
	)
	defer span.End()

	if ctxHandler, ok := handler.(executor.ContextAwareHandler); ok {
		ctxHandler.SetContext(ctx)
	}

	ourTaskID := task.ID
	if ourTaskID == "" {
		ourTaskID = uuid.New().String()
	}
	span.SetAttributes(attribute.String("exec.task_id", ourTaskID))

	sessionID := ourTaskID

	directive := task.Directive
	if task.SystemPrompt != "" {
		directive = task.SystemPrompt + "\n\n" + task.Directive
	}

	// Codex CLI non-interactive exec mode with JSON streaming.
	// Reference: https://developers.openai.com/codex/cli/
	// Flags confirmed against M2 research:
	//   exec                          non-interactive subcommand
	//   --json                        NDJSON output on stdout
	//   --skip-git-repo-check         allow non-git workspaces
	//   --dangerously-bypass-approvals-and-sandbox  permission bypass
	//   --model <name>                model selection
	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--dangerously-bypass-approvals-and-sandbox",
		"--model", e.getModel(task),
		directive,
	}

	codexPath := e.codexPath

	cmd := exec.CommandContext(ctx, codexPath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_CODEX] Command: %s exec --json --model %s\n", codexPath, e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_CODEX] Workspace: %s\n", task.Workspace)
	}

	env := executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:      task,
		SessionID: sessionID,
		Context:   ctx,
	})
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

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

	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	done := make(chan error, 1)
	var transcriptBuf strings.Builder
	var rawEvents []map[string]any
	var turnNum int
	var toolCallCount int
	var inputTokens, outputTokens int
	var turnSpan trace.Span
	var stderrBuf strings.Builder
	sawResult := false

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		const maxScannerBuffer = 1024 * 1024
		stdoutScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)
		stderrScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)

		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				stderrBuf.WriteString(line + "\n")
				if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[CODEX_STDERR] %s\n", line)
				}
			}
		}()

		for stdoutScanner.Scan() {
			line := strings.TrimSpace(stdoutScanner.Text())
			if line == "" {
				continue
			}
			lastActivity.Store(time.Now().UnixNano())

			if os.Getenv("DEBUG_AGENT") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_CODEX_RAW] %s\n", line)
			}

			ev, err := parseCodexEvent([]byte(line))
			if err != nil {
				// Non-JSON preamble lines are possible; skip silently unless debugging.
				if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_CODEX] skip non-JSON line: %v\n", err)
				}
				continue
			}
			rawEvents = append(rawEvents, ev.Raw)

			switch ev.Type {
			case "session", "session_start", "init":
				if ev.SessionID != "" {
					sessionID = ev.SessionID
					span.SetAttributes(attribute.String("exec.codex_session_id", ev.SessionID))
				}
				turnNum++
				if turnSpan != nil {
					turnSpan.End()
				}
				_, turnSpan = telemetry.StartSpan(ctx, codexTracer, "exec.turn",
					trace.WithAttributes(
						attribute.Int("turn.number", turnNum),
						attribute.String("session.id", sessionID),
					),
				)
				handler.OnTurnStart(turnNum)
				transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))

			case "message":
				if turnNum == 0 {
					turnNum = 1
					_, turnSpan = telemetry.StartSpan(ctx, codexTracer, "exec.turn",
						trace.WithAttributes(
							attribute.Int("turn.number", turnNum),
							attribute.String("session.id", sessionID),
						),
					)
					handler.OnTurnStart(turnNum)
					transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))
				}
				if ev.TurnNumber > turnNum {
					if turnSpan != nil {
						turnSpan.End()
					}
					turnNum = ev.TurnNumber
					_, turnSpan = telemetry.StartSpan(ctx, codexTracer, "exec.turn",
						trace.WithAttributes(
							attribute.Int("turn.number", turnNum),
							attribute.String("session.id", sessionID),
						),
					)
					handler.OnTurnStart(turnNum)
					transcriptBuf.WriteString(fmt.Sprintf("\n[TURN %d]\n", turnNum))
				}
				if ev.Text != "" {
					handler.OnText(ev.Text)
					transcriptBuf.WriteString(ev.Text)
				}
				// Codex emits cumulative tokens_used per message (running total),
				// not per-turn deltas — matching OpenAI API usage semantics.
				// Take the max so we end up with the final cumulative total.
				if ev.Tokens.Input > inputTokens {
					inputTokens = ev.Tokens.Input
				}
				if ev.Tokens.Output > outputTokens {
					outputTokens = ev.Tokens.Output
				}

			case "tool_use", "tool_call":
				toolCallCount++
				handler.OnToolUse(ev.ToolName, string(ev.Parameters))
				transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", ev.ToolName))

			case "tool_result":
				handler.OnToolResult(ev.ToolName, ev.Output)
				transcriptBuf.WriteString(fmt.Sprintf("[TOOL_RESULT] %s\n", ev.Output))

			case "result", "turn_complete", "message_stop":
				if ev.Tokens.Input > 0 || ev.Tokens.Output > 0 {
					if ev.Tokens.Input > inputTokens {
						inputTokens = ev.Tokens.Input
					}
					if ev.Tokens.Output > outputTokens {
						outputTokens = ev.Tokens.Output
					}
				}
				sawResult = true
				handler.OnTurnEnd(turnNum)
				if turnSpan != nil {
					turnSpan.End()
					turnSpan = nil
				}

			default:
				if os.Getenv("DEBUG_AGENT") != "" && ev.Type != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_CODEX] unknown event type: %q\n", ev.Type)
				}
			}
		}

		if turnSpan != nil {
			turnSpan.End()
		}

		if err := stdoutScanner.Err(); err != nil {
			done <- fmt.Errorf("stdout scanner error: %w", err)
			return
		}
		done <- cmd.Wait()
	}()

	for {
		select {
		case <-hardTimer.C:
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
				Error:         timeoutErr.Error(),
				DurationMS:    int(time.Since(startTime).Milliseconds()),
				NumTurns:      turnNum,
				ToolCallCount: toolCallCount,
				SessionID:     sessionID,
				Transcript:    transcriptBuf.String(),
				ProviderData:  providerData(rawEvents),
			}, nil

		case <-idleCheck.C:
			last := time.Unix(0, lastActivity.Load())
			idle := time.Since(last)
			if idle >= idleTimeout {
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
					Error:         idleErr.Error(),
					DurationMS:    int(time.Since(startTime).Milliseconds()),
					NumTurns:      turnNum,
					ToolCallCount: toolCallCount,
					SessionID:     sessionID,
					Transcript:    transcriptBuf.String(),
					ProviderData:  providerData(rawEvents),
				}, nil
			}
			idleCheck.Reset(idleTimeout - idle)

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
					ProviderData:  providerData(rawEvents),
				}, nil
			}

			cost := e.CostModel().CalculateCost(executor.TokenUsage{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
			})
			success := sawResult

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
				ProviderData:  providerData(rawEvents),
			}, nil
		}
	}
}

// Capabilities returns the list of features this executor supports.
func (e *CodexExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapLocalWorkspace,
	}
}

// CostModel returns pricing for gpt-5-codex (the default Codex model).
// Source: https://platform.openai.com/docs/pricing
// gpt-5-codex: $1.25/$10.00 per 1M tokens = $0.00125/$0.01 per 1K.
func (e *CodexExecutor) CostModel() *executor.CostModel {
	return &executor.CostModel{
		ProviderName:    "openai",
		InputTokenCost:  0.00125,
		OutputTokenCost: 0.01,
		CacheReadCost:   0.000125,
	}
}

// HealthCheck verifies the codex binary exists on PATH and responds.
func (e *CodexExecutor) HealthCheck(ctx context.Context) error {
	codexPath := e.codexPath
	if _, err := exec.LookPath(codexPath); err != nil {
		if _, statErr := os.Stat(codexPath); statErr != nil {
			return fmt.Errorf("codex CLI not found: %w (install with: npm i -g @openai/codex)", err)
		}
	}
	checkCmd := exec.CommandContext(ctx, codexPath, "--version")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("codex --version failed: %w", err)
	}
	if os.Getenv("OPENAI_API_KEY") == "" {
		// Auth may also come from `codex login` cache; warn but do not fail.
		if os.Getenv("DEBUG_AGENT") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_CODEX] OPENAI_API_KEY unset; relying on codex login cache\n")
		}
	}
	return nil
}

// Close releases any resources held by the executor.
func (e *CodexExecutor) Close() error {
	return nil
}

func (e *CodexExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// codexTokens captures the flat token structure Codex emits.
type codexTokens struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// codexEvent is the normalized Codex NDJSON event shape.
// Codex emits flat records (see codex_compat_test.go):
//
//	{"type":"message","turn_number":N,"text":"...","tokens_used":{"input":N,"output":N}}
//
// Unknown fields are preserved in Raw for ProviderData.
type codexEvent struct {
	Type       string          `json:"type"`
	TurnNumber int             `json:"turn_number,omitempty"`
	Text       string          `json:"text,omitempty"`
	Tokens     codexTokens     `json:"tokens_used,omitempty"`
	SessionID  string          `json:"session_id,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
	Output     string          `json:"output,omitempty"`
	Role       string          `json:"role,omitempty"`

	// Raw preserves the full event map for ProviderData (tolerance to schema drift).
	Raw map[string]any `json:"-"`
}

// parseCodexEvent parses a single NDJSON line into a codexEvent.
// Non-JSON and unparseable lines return an error; callers should skip them
// rather than fail hard (Codex CLI may emit non-JSON preamble on stdout).
func parseCodexEvent(line []byte) (*codexEvent, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, fmt.Errorf("empty line")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("non-JSON line")
	}
	var ev codexEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	// Tolerate schema drift: always capture the raw map.
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err == nil {
		ev.Raw = raw
	}
	return &ev, nil
}

// providerData wraps the list of raw events as the Result.ProviderData map.
func providerData(events []map[string]any) map[string]any {
	if len(events) == 0 {
		return nil
	}
	return map[string]any{
		"codex_events": events,
	}
}

// Register registers the Codex executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("codex", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
