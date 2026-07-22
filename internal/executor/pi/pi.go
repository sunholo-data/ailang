// Package pi provides an Executor implementation for the pi CLI
// (npm: @mariozechner/pi-coding-agent), a deliberately minimal
// Claude Agent SDK-based coding harness with broad multi-provider reach.
//
// pi emits NDJSON via `pi --mode json` — a different schema from Claude,
// Gemini, Codex, and opencode. See README.md and testdata/ for full
// schema documentation with fixture-backed examples.
//
// Key parser facts (pi 0.70.x):
//   - Top-level events: session, agent_start, turn_start, message_start,
//     message_update, message_end, tool_execution_start, tool_execution_end,
//     turn_end, agent_end. agent_end is the unambiguous terminal event.
//   - Inside message_update, assistantMessageEvent.type ∈ {text_start,
//     text_delta, text_end, thinking_start, thinking_delta, thinking_end,
//     toolcall_start, toolcall_delta, toolcall_end}.
//   - Per-turn token deltas in message_end (role=assistant) — sum across
//     turns for totals (similar pattern to opencode's step_finish).
//   - Cost reported directly in message_end.message.usage.cost.total —
//     summed across turns rather than recomputed from token counts.
//   - Model string: "provider/id" shorthand (e.g. "anthropic/claude-haiku-4-5",
//     "openai/gpt-5.4").
package pi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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

var piTracer = telemetry.Tracer("executor.pi")

// PiExecutor executes tasks using the pi CLI.
type PiExecutor struct {
	piPath         string
	model          string
	timeoutSeconds int
}

// New creates a new PiExecutor.
func New(cfg *executor.Config) (*PiExecutor, error) {
	piPath := cfg.PiPath
	if piPath == "" {
		piPath = "pi"
	}

	model := cfg.PiModel
	if model == "" {
		model = "anthropic/claude-haiku-4-5"
	}

	return &PiExecutor{
		piPath:         piPath,
		model:          model,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier.
func (e *PiExecutor) Name() string {
	return "pi"
}

// Execute runs a task and returns the result.
func (e *PiExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks, parsing the
// pi NDJSON stream into normalized executor events.
func (e *PiExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	ctx, span := telemetry.StartSpan(ctx, piTracer, "pi.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "pi"),
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

	directive := task.Directive
	if task.SystemPrompt != "" {
		directive = task.SystemPrompt + "\n\n" + task.Directive
	}

	args := buildPiArgs(e.getModel(task), task, directive)
	piPath := e.piPath

	cmd := exec.CommandContext(ctx, piPath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_PI] Command: %s --mode json --model %s\n", piPath, e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_PI] Workspace: %s\n", task.Workspace)
	}

	env := executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:        task,
		SessionID:   ourTaskID,
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
		return nil, fmt.Errorf("failed to start pi: %w", err)
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
	idleCheck.Stop()
	defer idleCheck.Stop()

	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	var firstEventSeen atomic.Bool

	done := make(chan error, 1)
	var transcriptBuf strings.Builder
	var rawEvents []map[string]any
	var numTurns int
	var toolCallCount int
	toolCalls := map[string]int{} // per-tool-name histogram (alongside toolCallCount)
	// pi emits per-turn deltas in message_end (role=assistant); sum across turns.
	var inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int
	var totalCostUSD float64
	var sessionID string
	var turnSpan trace.Span
	var stderrBuf strings.Builder
	// Last settled stopReason seen on a message_end / turn_end. Streaming
	// message_update events carry cumulative partial state, so they are
	// deliberately excluded — only settled events are authoritative.
	var lastStopReason string

	// M-EVAL-COST-AND-SPEED-BUDGETS: speed + cost-kill instrumentation.
	// pi emits per-turn token deltas in message_end (role=assistant), so the
	// budget tally is naturally incremental. costKilled signals breach and
	// triggers an early process kill.
	var firstAttemptMs int64 = -1
	var firstStreamEventAt time.Time
	var costKilled bool

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		// pi emits very large lines (full cumulative state in every message_update).
		// 8MB is sufficient headroom for multi-turn long-output runs.
		const maxScannerBuffer = 8 * 1024 * 1024
		stdoutScanner.Buffer(make([]byte, 0, 64*1024), maxScannerBuffer)
		stderrScanner.Buffer(make([]byte, 0, 64*1024), maxScannerBuffer)

		go func() {
			for stderrScanner.Scan() {
				line := stderrScanner.Text()
				stderrBuf.WriteString(line)
				stderrBuf.WriteByte('\n')
			}
		}()

		for stdoutScanner.Scan() {
			line := stdoutScanner.Bytes()
			lastActivity.Store(time.Now().UnixNano())
			if !firstEventSeen.Swap(true) {
				ttftTimer.Stop()
				idleCheck.Reset(idleTimeout)
			}

			ev, err := parsePiEvent(line)
			if err != nil {
				continue
			}

			if ev.Raw != nil {
				rawEvents = append(rawEvents, ev.Raw)
			}

			switch ev.Type {
			case "session":
				if ev.SessionID != "" {
					sessionID = ev.SessionID
				}

			case "turn_start":
				numTurns++
				_, turnSpan = telemetry.StartSpan(ctx, piTracer, "pi.turn",
					trace.WithAttributes(
						attribute.Int("pi.turn_num", numTurns),
					),
				)
				handler.OnTurnStart(numTurns)

			case "message_update":
				if ev.AssistantMessageEvent == nil {
					continue
				}
				ame := ev.AssistantMessageEvent
				switch ame.Type {
				case "text_delta":
					if ame.Delta != "" {
						if firstStreamEventAt.IsZero() {
							firstStreamEventAt = time.Now()
						}
						transcriptBuf.WriteString(ame.Delta)
						handler.OnText(ame.Delta)
						// M-EVAL-COST-AND-SPEED-BUDGETS: first text = candidate solution
						// (when no Write/Edit tool calls have occurred yet).
						if firstAttemptMs < 0 {
							firstAttemptMs = time.Since(startTime).Milliseconds()
						}
					}
				}

			case "tool_execution_start":
				toolCallCount++
				if ev.ToolName != "" {
					toolCalls[ev.ToolName]++
				}
				if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_PI] tool_execution_start: %s\n", ev.ToolName)
				}
				argsStr := string(ev.Args)
				if argsStr == "" {
					argsStr = "{}"
				}
				handler.OnToolUse(ev.ToolName, argsStr)
				// M-EVAL-COST-AND-SPEED-BUDGETS: first Write/Edit = first solution attempt.
				if firstAttemptMs < 0 && (ev.ToolName == "Write" || ev.ToolName == "Edit") {
					firstAttemptMs = time.Since(startTime).Milliseconds()
				}

			case "tool_execution_end":
				output := flattenPiToolResult(ev.Result)
				handler.OnToolResult(ev.ToolName, output)

			case "message_end":
				if ev.Message != nil && ev.Message.StopReason != "" {
					lastStopReason = ev.Message.StopReason
				}
				// Per-turn deltas for assistant messages — sum into totals.
				if ev.Message != nil && ev.Message.Role == "assistant" && ev.Message.Usage != nil {
					u := ev.Message.Usage
					inputTokens += u.Input
					outputTokens += u.Output
					cacheReadTokens += u.CacheRead
					cacheWriteTokens += u.CacheWrite
					totalCostUSD += u.Cost.Total
					// M-EVAL-COST-AND-SPEED-BUDGETS: incremental cost tally on per-turn delta.
					if task.Budget != nil && (u.Input > 0 || u.Output > 0) {
						if _, exceeded := task.Budget.Add(u.Input, u.Output); exceeded {
							costKilled = true
							_ = cmd.Process.Kill()
						}
					}
				}

			case "turn_end":
				if ev.Message != nil && ev.Message.StopReason != "" {
					lastStopReason = ev.Message.StopReason
				}
				if turnSpan != nil {
					if ev.Message != nil && ev.Message.Usage != nil {
						u := ev.Message.Usage
						turnSpan.SetAttributes(
							attribute.Int("pi.input_tokens", u.Input),
							attribute.Int("pi.output_tokens", u.Output),
							attribute.Float64("pi.cost_usd", u.Cost.Total),
						)
					}
					turnSpan.End()
					turnSpan = nil
				}
				handler.OnTurnEnd(numTurns)

			case "agent_end":
				// Terminal — process will exit imminently. No per-event work
				// needed; aggregation already done via message_end events.
			}
		}

		done <- cmd.Wait()
	}()

	for {
		select {
		case err := <-done:
			duration := time.Since(startTime)
			span.SetAttributes(
				attribute.Int("pi.turns", numTurns),
				attribute.Int("pi.tool_calls", toolCallCount),
				attribute.Int("pi.input_tokens", inputTokens),
				attribute.Int("pi.output_tokens", outputTokens),
				attribute.Float64("pi.cost_usd", totalCostUSD),
			)

			// M-EVAL-COST-AND-SPEED-BUDGETS: speed metric.
			var tokensPerSec float64
			if !firstStreamEventAt.IsZero() && outputTokens > 0 {
				if gen := time.Since(firstStreamEventAt).Seconds(); gen > 0 {
					tokensPerSec = float64(outputTokens) / gen
				}
			}

			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				errMsg := fmt.Sprintf("pi exited with error: %v\nstderr: %s", err, stderrBuf.String())
				// Terminal precedence: a cost kill outranks whatever the last
				// stream event said, because CategorizeAgentError trusts
				// FinishReason over the Error string.
				finishReason := executor.FinishError
				if costKilled {
					errMsg = fmt.Sprintf("cost budget exceeded ($%.4f) — %s", task.Budget.KilledAt(), errMsg)
					finishReason = executor.FinishCostExhausted
				}
				return &executor.Result{
					Success:                  false,
					FinishReason:             finishReason,
					Output:                   transcriptBuf.String(),
					Error:                    errMsg,
					DurationMS:               int(duration.Milliseconds()),
					InputTokens:              inputTokens,
					OutputTokens:             outputTokens,
					CacheReadInputTokens:     cacheReadTokens,
					CacheCreationInputTokens: cacheWriteTokens,
					CostUSD:                  totalCostUSD,
					NumTurns:                 numTurns,
					ToolCallCount:            toolCallCount,
					ToolCalls:                toolCalls,
					SessionID:                sessionID,
					ProviderData:             piProviderData(rawEvents),
					CostKilledAt:             task.Budget.KilledAt(),
					FirstAttemptMs:           firstAttemptMs,
					SuccessAtMs:              -1,
					TokensPerSec:             tokensPerSec,
				}, nil
			}

			output := transcriptBuf.String()
			success := output != "" || toolCallCount > 0
			// Clean exit: the stream's own last stop reason is authoritative.
			// Default to "stop" when pi reported none at all.
			finishReason := executor.FinishStop
			if lastStopReason != "" {
				finishReason = normalizePiFinishReason(lastStopReason)
			}
			if costKilled {
				success = false
				finishReason = executor.FinishCostExhausted
			}

			span.SetStatus(codes.Ok, "")
			return &executor.Result{
				Success:                  success,
				FinishReason:             finishReason,
				Output:                   output,
				DurationMS:               int(duration.Milliseconds()),
				InputTokens:              inputTokens,
				OutputTokens:             outputTokens,
				CacheReadInputTokens:     cacheReadTokens,
				CacheCreationInputTokens: cacheWriteTokens,
				CostUSD:                  totalCostUSD,
				NumTurns:                 numTurns,
				ToolCallCount:            toolCallCount,
				ToolCalls:                toolCalls,
				SessionID:                sessionID,
				ProviderData:             piProviderData(rawEvents),
				CostKilledAt:             task.Budget.KilledAt(),
				FirstAttemptMs:           firstAttemptMs,
				SuccessAtMs:              -1,
				TokensPerSec:             tokensPerSec,
			}, nil

		case <-hardTimer.C:
			_ = cmd.Process.Kill()
			span.SetStatus(codes.Error, "hard timeout")
			return &executor.Result{
				Success:        false,
				Error:          fmt.Sprintf("pi exceeded hard timeout (%v)", timeout),
				FinishReason:   executor.FinishTimeout,
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				CostKilledAt:   task.Budget.KilledAt(),
				FirstAttemptMs: firstAttemptMs,
				SuccessAtMs:    -1,
			}, nil

		case <-ttftTimer.C:
			_ = cmd.Process.Kill()
			span.SetStatus(codes.Error, "ttft timeout")
			return &executor.Result{
				Success:        false,
				Error:          fmt.Sprintf("pi produced no output within %v (prefill timeout)", ttftTimeout),
				FinishReason:   executor.FinishTimeout,
				FirstAttemptMs: -1,
				SuccessAtMs:    -1,
			}, nil

		case <-idleCheck.C:
			since := time.Since(time.Unix(0, lastActivity.Load()))
			if since > idleTimeout {
				_ = cmd.Process.Kill()
				span.SetStatus(codes.Error, "generation idle timeout")
				return &executor.Result{
					Success:        false,
					Error:          fmt.Sprintf("pi idle for %v mid-generation (no output)", since),
					FinishReason:   executor.FinishTimeout,
					InputTokens:    inputTokens,
					OutputTokens:   outputTokens,
					CostKilledAt:   task.Budget.KilledAt(),
					FirstAttemptMs: firstAttemptMs,
					SuccessAtMs:    -1,
				}, nil
			}
			idleCheck.Reset(idleTimeout - since)

		case <-ctx.Done():
			_ = cmd.Process.Kill()
			span.SetStatus(codes.Error, ctx.Err().Error())
			return &executor.Result{
				Success:        false,
				Error:          fmt.Sprintf("pi cancelled: %v", ctx.Err()),
				FinishReason:   piCancelFinishReason(ctx.Err()),
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				CostKilledAt:   task.Budget.KilledAt(),
				FirstAttemptMs: firstAttemptMs,
				SuccessAtMs:    -1,
			}, nil
		}
	}
}

// Capabilities returns the list of features this executor supports.
func (e *PiExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapLocalWorkspace,
		executor.CapToolControl,
	}
}

// CostModel returns a placeholder cost model. Pi reports cost directly in
// message_end.usage.cost.total per turn; the executor sums those values into
// Result.CostUSD rather than recomputing from token counts.
func (e *PiExecutor) CostModel() *executor.CostModel {
	return &executor.CostModel{
		ProviderName:    "pi",
		InputTokenCost:  0.0,
		OutputTokenCost: 0.0,
		CacheReadCost:   0.0,
	}
}

// HealthCheck verifies the pi binary exists on PATH and responds.
func (e *PiExecutor) HealthCheck(ctx context.Context) error {
	piPath := e.piPath
	if _, err := exec.LookPath(piPath); err != nil {
		if _, statErr := os.Stat(piPath); statErr != nil {
			return fmt.Errorf("pi CLI not found: %w (install with: npm i -g @mariozechner/pi-coding-agent)", err)
		}
	}
	checkCmd := exec.CommandContext(ctx, piPath, "--version")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("pi --version failed: %w", err)
	}
	return nil
}

// Close releases any resources held by the executor.
func (e *PiExecutor) Close() error {
	return nil
}

func (e *PiExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// buildPiArgs assembles the pi CLI argument vector.
//
// Flags used:
//
//	--mode json        NDJSON event stream
//	-p                 non-interactive (process prompt and exit)
//	--model <prov/id>  model selection via provider-prefix shorthand
//	--no-session       ephemeral run (avoids ~/.pi/sessions/ pollution)
//	--no-tools         when AllowedTools is empty; otherwise --tools <list>
//
// The directive is the trailing positional argument.
func buildPiArgs(model string, task *executor.Task, directive string) []string {
	args := []string{
		"--mode", "json",
		"--model", model,
		"--no-session",
		"-p",
	}

	switch {
	case task.AllowedTools == nil:
		// nil = caller does not specify; let pi's defaults apply.
	case len(task.AllowedTools) == 0:
		args = append(args, "--no-tools")
	default:
		args = append(args, "--tools", strings.Join(task.AllowedTools, ","))
	}

	args = append(args, directive)
	return args
}

// piUsageCost mirrors message_end.message.usage.cost.
type piUsageCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

// piUsage mirrors message_end.message.usage.
type piUsage struct {
	Input       int         `json:"input"`
	Output      int         `json:"output"`
	CacheRead   int         `json:"cacheRead"`
	CacheWrite  int         `json:"cacheWrite"`
	TotalTokens int         `json:"totalTokens"`
	Cost        piUsageCost `json:"cost"`
}

// piMessage captures the per-message envelope.
type piMessage struct {
	Role string `json:"role"`
	// StopReason is pi's per-message stop signal, present on message_start /
	// message_end / turn_end (and mirrored on assistantMessageEvent.partial).
	// Observed values: "stop", "toolUse" (fixtures, pi 0.70.2). A tool-calling
	// turn ends "toolUse" and the run's FINAL turn ends "stop", so only the
	// last value seen is meaningful as a run-level finish reason.
	StopReason string   `json:"stopReason,omitempty"`
	Usage      *piUsage `json:"usage,omitempty"`
}

// piAssistantMessageEvent captures the inner discriminator for message_update.
type piAssistantMessageEvent struct {
	Type         string `json:"type"`
	ContentIndex int    `json:"contentIndex,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Content      string `json:"content,omitempty"`
}

// piToolResult captures the tool_execution_end result envelope.
type piToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// piEvent is the top-level NDJSON event wrapper.
type piEvent struct {
	Type                  string                   `json:"type"`
	Message               *piMessage               `json:"message,omitempty"`
	AssistantMessageEvent *piAssistantMessageEvent `json:"assistantMessageEvent,omitempty"`

	// session event
	SessionID string `json:"id,omitempty"`

	// tool_execution_start / tool_execution_end
	ToolCallID string          `json:"toolCallId,omitempty"`
	ToolName   string          `json:"toolName,omitempty"`
	Args       json.RawMessage `json:"args,omitempty"`
	Result     *piToolResult   `json:"result,omitempty"`
	IsError    bool            `json:"isError,omitempty"`

	// Raw preserves full event for ProviderData (schema-drift tolerance).
	Raw map[string]any `json:"-"`
}

// parsePiEvent parses a single NDJSON line.
// Returns error for non-JSON lines or parse failures; callers skip them.
func parsePiEvent(line []byte) (*piEvent, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, fmt.Errorf("empty line")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("non-JSON line")
	}
	var ev piEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err == nil {
		ev.Raw = raw
	}
	return &ev, nil
}

// flattenPiToolResult joins all text content blocks from a tool_execution_end.
func flattenPiToolResult(r *piToolResult) string {
	if r == nil {
		return ""
	}
	if len(r.Content) == 0 {
		return ""
	}
	if len(r.Content) == 1 {
		return r.Content[0].Text
	}
	var b strings.Builder
	for i, c := range r.Content {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(c.Text)
	}
	return b.String()
}

// piCancelFinishReason distinguishes a deadline-driven context cancellation
// (a timeout by another name) from a caller-driven one (an abort).
func piCancelFinishReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return executor.FinishTimeout
	}
	return executor.FinishError
}

// normalizePiFinishReason maps pi's camelCase stopReason vocabulary onto the
// canonical executor.Finish* values.
//
// Pi is pre-1.0 (0.70.x) and its stop-reason vocabulary is NOT documented
// upstream; "stop" and "toolUse" are the only values observed in captured
// fixtures. Rather than guess at the rest, unrecognized values are passed
// through verbatim so they surface in the banked JSON instead of being
// silently coerced to "stop" (CategorizeAgentError ignores values it doesn't
// know, so pass-through cannot misclassify a run). Re-check this mapping when
// bumping the pinned pi version.
func normalizePiFinishReason(raw string) string {
	switch raw {
	case "stop", "endTurn", "end_turn":
		return executor.FinishStop
	case "toolUse", "tool_use", "toolCalls":
		return executor.FinishToolCalls
	case "maxTokens", "max_tokens", "length":
		return executor.FinishLength
	case "refusal", "safety", "contentFilter":
		return executor.FinishContentFilter
	case "aborted", "error":
		return executor.FinishError
	default:
		return raw
	}
}

// piProviderData wraps raw events as Result.ProviderData.
func piProviderData(events []map[string]any) map[string]any {
	if len(events) == 0 {
		return nil
	}
	return map[string]any{
		"pi_events": events,
	}
}

// Register registers the pi executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("pi", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
