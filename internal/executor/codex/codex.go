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

const (
	spanExecTurn  = "exec.turn"
	attrTurnNum   = "turn.number"
	attrSessionID = "session.id"
	fmtTurnHeader = "\n[TURN %d]\n"
)

// startTurn opens a new turn span, fires OnTurnStart, and appends the turn
// header to transcriptBuf. It ends any in-progress turn span first.
func startTurn(
	ctx context.Context,
	n int,
	sessionID string,
	prev *trace.Span,
	buf *strings.Builder,
	handler executor.EventHandler,
) trace.Span {
	if prev != nil && *prev != nil {
		(*prev).End()
	}
	_, s := telemetry.StartSpan(ctx, codexTracer, spanExecTurn,
		trace.WithAttributes(
			attribute.Int(attrTurnNum, n),
			attribute.String(attrSessionID, sessionID),
		),
	)
	handler.OnTurnStart(n)
	buf.WriteString(fmt.Sprintf(fmtTurnHeader, n))
	return s
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
		Task:        task,
		SessionID:   sessionID,
		Context:     ctx,
		GCPProject:  task.GCPProject,
		GCPLocation: task.GCPLocation,
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
	ttftTimeout := task.TTFTTimeout
	if ttftTimeout == 0 {
		ttftTimeout = 30 * time.Second
	}

	hardTimer := time.NewTimer(timeout)
	defer hardTimer.Stop()
	ttftTimer := time.NewTimer(ttftTimeout)
	defer ttftTimer.Stop()
	idleCheck := time.NewTimer(idleTimeout)
	idleCheck.Stop() // paused until first event arrives
	defer idleCheck.Stop()

	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	var firstEventSeen atomic.Bool
	done := make(chan error, 1)
	var transcriptBuf strings.Builder
	var rawEvents []map[string]any
	var turnNum int
	var toolCallCount int
	var inputTokens, outputTokens, cachedInputTokens int
	var turnSpan trace.Span
	var stderrBuf strings.Builder
	sawResult := false

	// M-EVAL-COST-AND-SPEED-BUDGETS: speed + cost-kill instrumentation.
	// Codex emits cumulative input/output tokens (old format: per-message;
	// new format: at turn.completed). We compute deltas to feed Budget.Add()
	// incrementally. costKilled signals breach and triggers an early kill.
	var firstAttemptMs int64 = -1
	var firstStreamEventAt time.Time
	var costKilled bool
	var prevBudgetIn, prevBudgetOut int

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
			if !firstEventSeen.Swap(true) {
				ttftTimer.Stop()
				idleCheck.Reset(idleTimeout)
			}

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
			// ── New format (codex CLI v0.1+): thread/item stream ──────────────────
			case "thread.started":
				if ev.ThreadID != "" {
					sessionID = ev.ThreadID
					span.SetAttributes(attribute.String("exec.codex_session_id", ev.ThreadID))
				}

			case "turn.started":
				turnNum++
				if firstStreamEventAt.IsZero() {
					firstStreamEventAt = time.Now()
				}
				turnSpan = startTurn(ctx, turnNum, sessionID, &turnSpan, &transcriptBuf, handler)

			case "item.completed":
				if ev.Item != nil {
					switch ev.Item.Type {
					case "file_change", "command_execution":
						// Each completed file write or shell command = one tool call.
						toolCallCount++
						toolLabel := ev.Item.Type
						if ev.Item.Command != "" {
							toolLabel = ev.Item.Command
						}
						handler.OnToolUse(toolLabel, "")
						transcriptBuf.WriteString(fmt.Sprintf("[TOOL] %s\n", toolLabel))
						// M-EVAL-COST-AND-SPEED-BUDGETS: first file_change = first solution attempt.
						if firstAttemptMs < 0 && ev.Item.Type == "file_change" {
							firstAttemptMs = time.Since(startTime).Milliseconds()
						}
					case "agent_message":
						if ev.Item.Text != "" {
							handler.OnText(ev.Item.Text)
							transcriptBuf.WriteString(ev.Item.Text)
							// M-EVAL-COST-AND-SPEED-BUDGETS: fallback first-attempt
							// signal when no file_change has occurred yet.
							if firstAttemptMs < 0 {
								firstAttemptMs = time.Since(startTime).Milliseconds()
							}
						}
					}
				}

			case "turn.completed":
				// New format reports token usage on turn.completed.
				if ev.Usage != nil {
					if ev.Usage.InputTokens > inputTokens {
						inputTokens = ev.Usage.InputTokens
					}
					if ev.Usage.OutputTokens > outputTokens {
						outputTokens = ev.Usage.OutputTokens
					}
					if ev.Usage.CachedInputTokens > cachedInputTokens {
						cachedInputTokens = ev.Usage.CachedInputTokens
					}
					// M-EVAL-COST-AND-SPEED-BUDGETS: per-turn cumulative→delta.
					if task.Budget != nil {
						deltaIn := inputTokens - prevBudgetIn
						deltaOut := outputTokens - prevBudgetOut
						if deltaIn < 0 {
							deltaIn = 0
						}
						if deltaOut < 0 {
							deltaOut = 0
						}
						prevBudgetIn = inputTokens
						prevBudgetOut = outputTokens
						if deltaIn > 0 || deltaOut > 0 {
							if _, exceeded := task.Budget.Add(deltaIn, deltaOut); exceeded {
								costKilled = true
								_ = cmd.Process.Kill()
							}
						}
					}
				}
				// New format has no separate "result" event — turn.completed
				// is the terminal signal that the session produced output.
				sawResult = true
				handler.OnTurnEnd(turnNum)
				if turnSpan != nil {
					turnSpan.End()
					turnSpan = nil
				}

			// ── Old format (pre-v0.1): flat message/tool_use stream ───────────────
			case "session", "session_start", "init":
				if ev.SessionID != "" {
					sessionID = ev.SessionID
					span.SetAttributes(attribute.String("exec.codex_session_id", ev.SessionID))
				}
				if turnNum == 0 {
					turnNum++
					turnSpan = startTurn(ctx, turnNum, sessionID, &turnSpan, &transcriptBuf, handler)
				}

			case "message":
				if firstStreamEventAt.IsZero() {
					firstStreamEventAt = time.Now()
				}
				if turnNum == 0 {
					turnNum = 1
					turnSpan = startTurn(ctx, turnNum, sessionID, &turnSpan, &transcriptBuf, handler)
				}
				if ev.TurnNumber > turnNum {
					turnNum = ev.TurnNumber
					turnSpan = startTurn(ctx, turnNum, sessionID, &turnSpan, &transcriptBuf, handler)
				}
				if ev.Text != "" {
					handler.OnText(ev.Text)
					transcriptBuf.WriteString(ev.Text)
					// M-EVAL-COST-AND-SPEED-BUDGETS: first non-empty assistant text
					// = first solution attempt (old format has no separate file_change event).
					if firstAttemptMs < 0 {
						firstAttemptMs = time.Since(startTime).Milliseconds()
					}
				}
				// Old format: cumulative tokens_used per message.
				if ev.Tokens.Input > inputTokens {
					inputTokens = ev.Tokens.Input
				}
				if ev.Tokens.Output > outputTokens {
					outputTokens = ev.Tokens.Output
				}
				// M-EVAL-COST-AND-SPEED-BUDGETS: cumulative→delta tally.
				if task.Budget != nil {
					deltaIn := inputTokens - prevBudgetIn
					deltaOut := outputTokens - prevBudgetOut
					if deltaIn < 0 {
						deltaIn = 0
					}
					if deltaOut < 0 {
						deltaOut = 0
					}
					prevBudgetIn = inputTokens
					prevBudgetOut = outputTokens
					if deltaIn > 0 || deltaOut > 0 {
						if _, exceeded := task.Budget.Add(deltaIn, deltaOut); exceeded {
							costKilled = true
							_ = cmd.Process.Kill()
						}
					}
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
				// M-EVAL-COST-AND-SPEED-BUDGETS: residual tally at terminal event.
				if task.Budget != nil {
					deltaIn := inputTokens - prevBudgetIn
					deltaOut := outputTokens - prevBudgetOut
					if deltaIn < 0 {
						deltaIn = 0
					}
					if deltaOut < 0 {
						deltaOut = 0
					}
					prevBudgetIn = inputTokens
					prevBudgetOut = outputTokens
					if deltaIn > 0 || deltaOut > 0 {
						_, _ = task.Budget.Add(deltaIn, deltaOut)
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
			freshInput, cachedInput := splitCodexInputTokens(inputTokens, cachedInputTokens)
			return &executor.Result{
				Success:              false,
				Error:                timeoutErr.Error(),
				FinishReason:         executor.FinishTimeout,
				DurationMS:           int(time.Since(startTime).Milliseconds()),
				NumTurns:             turnNum,
				ToolCallCount:        toolCallCount,
				SessionID:            sessionID,
				Transcript:           transcriptBuf.String(),
				ProviderData:         providerData(rawEvents),
				InputTokens:          freshInput,
				OutputTokens:         outputTokens,
				CacheReadInputTokens: cachedInput,
				CostKilledAt:         task.Budget.KilledAt(),
				FirstAttemptMs:       firstAttemptMs,
				SuccessAtMs:          -1,
			}, nil

		case <-ttftTimer.C:
			_ = cmd.Process.Kill()
			ttftErr := fmt.Errorf("codex produced no output within %v (prefill timeout)", ttftTimeout)
			handler.OnError(ttftErr)
			span.RecordError(ttftErr)
			span.SetStatus(codes.Error, "ttft timeout")
			span.SetAttributes(
				attribute.Int("task.turns", turnNum),
				attribute.Bool("task.success", false),
			)
			return &executor.Result{
				Success:        false,
				Error:          ttftErr.Error(),
				FinishReason:   executor.FinishTimeout,
				DurationMS:     int(time.Since(startTime).Milliseconds()),
				NumTurns:       turnNum,
				ToolCallCount:  toolCallCount,
				SessionID:      sessionID,
				Transcript:     transcriptBuf.String(),
				ProviderData:   providerData(rawEvents),
				FirstAttemptMs: -1,
				SuccessAtMs:    -1,
			}, nil

		case <-idleCheck.C:
			last := time.Unix(0, lastActivity.Load())
			idle := time.Since(last)
			if idle >= idleTimeout {
				_ = cmd.Process.Kill()
				idleErr := fmt.Errorf("codex idle for %v mid-generation (no output)", idle.Round(time.Second))
				handler.OnError(idleErr)
				span.RecordError(idleErr)
				span.SetStatus(codes.Error, "generation idle timeout")
				span.SetAttributes(
					attribute.Int("task.turns", turnNum),
					attribute.Bool("task.success", false),
				)
				freshInput, cachedInput := splitCodexInputTokens(inputTokens, cachedInputTokens)
				return &executor.Result{
					Success:              false,
					Error:                idleErr.Error(),
					FinishReason:         executor.FinishTimeout,
					DurationMS:           int(time.Since(startTime).Milliseconds()),
					NumTurns:             turnNum,
					ToolCallCount:        toolCallCount,
					SessionID:            sessionID,
					Transcript:           transcriptBuf.String(),
					ProviderData:         providerData(rawEvents),
					InputTokens:          freshInput,
					OutputTokens:         outputTokens,
					CacheReadInputTokens: cachedInput,
					CostKilledAt:         task.Budget.KilledAt(),
					FirstAttemptMs:       firstAttemptMs,
					SuccessAtMs:          -1,
				}, nil
			}
			idleCheck.Reset(idleTimeout - idle)

		case err := <-done:
			duration := time.Since(startTime)

			// M-EVAL-COST-AND-SPEED-BUDGETS: speed metric.
			var tokensPerSec float64
			if !firstStreamEventAt.IsZero() && outputTokens > 0 {
				if gen := time.Since(firstStreamEventAt).Seconds(); gen > 0 {
					tokensPerSec = float64(outputTokens) / gen
				}
			}

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
				// Terminal precedence: a cost kill outranks the generic exit
				// error, because CategorizeAgentError trusts FinishReason
				// over the Error string.
				finishReason := executor.FinishError
				if costKilled {
					errMsg = fmt.Sprintf("cost budget exceeded ($%.4f) — %s", task.Budget.KilledAt(), errMsg)
					finishReason = executor.FinishCostExhausted
				}
				freshInput, cachedInput := splitCodexInputTokens(inputTokens, cachedInputTokens)
				return &executor.Result{
					Success:              false,
					Error:                errMsg,
					FinishReason:         finishReason,
					DurationMS:           int(duration.Milliseconds()),
					NumTurns:             turnNum,
					ToolCallCount:        toolCallCount,
					SessionID:            sessionID,
					Transcript:           transcriptBuf.String(),
					ProviderData:         providerData(rawEvents),
					InputTokens:          freshInput,
					OutputTokens:         outputTokens,
					CacheReadInputTokens: cachedInput,
					CostKilledAt:         task.Budget.KilledAt(),
					FirstAttemptMs:       firstAttemptMs,
					SuccessAtMs:          -1,
					TokensPerSec:         tokensPerSec,
				}, nil
			}

			freshInput, cachedInput := splitCodexInputTokens(inputTokens, cachedInputTokens)
			cost := e.CostModel().CalculateCost(executor.TokenUsage{
				InputTokens:          freshInput,
				OutputTokens:         outputTokens,
				CacheReadInputTokens: cachedInput,
			})
			success := sawResult
			// The codex CLI's --json stream carries NO model-level stop reason
			// (see the schema note on codexEvent), so "stop" here asserts only
			// what this executor can actually observe: a terminal event
			// arrived and the process exited cleanly. A codex run that refused
			// the task, tripped a content filter, or truncated at the token cap
			// is indistinguishable from a clean finish at this layer.
			finishReason := executor.FinishStop
			if !success {
				finishReason = executor.FinishError
			}
			if costKilled {
				success = false
				finishReason = executor.FinishCostExhausted
			}

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
				Success:              success,
				FinishReason:         finishReason,
				Output:               transcriptBuf.String(),
				DurationMS:           int(duration.Milliseconds()),
				NumTurns:             turnNum,
				ToolCallCount:        toolCallCount,
				CostUSD:              cost,
				InputTokens:          freshInput,
				OutputTokens:         outputTokens,
				CacheReadInputTokens: cachedInput,
				SessionID:            sessionID,
				Transcript:           transcriptBuf.String(),
				ProviderData:         providerData(rawEvents),
				CostKilledAt:         task.Budget.KilledAt(),
				FirstAttemptMs:       firstAttemptMs,
				SuccessAtMs:          -1,
				TokensPerSec:         tokensPerSec,
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

// Register registers the Codex executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("codex", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
