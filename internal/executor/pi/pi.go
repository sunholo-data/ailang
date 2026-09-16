// Package pi provides an Executor implementation for the pi CLI
// (npm: @earendil-works/pi-coding-agent, formerly @mariozechner), a deliberately minimal
// Claude Agent SDK-based coding harness with broad multi-provider reach.
//
// pi emits NDJSON via `pi --mode json` — a different schema from Claude,
// Gemini, Codex, and opencode. See README.md and testdata/ for full
// schema documentation with fixture-backed examples.
//
// Key parser facts (pi 0.85.x; see events.go and README.md for the 0.73→0.85 drift):
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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/proctree"
	"github.com/sunholo-data/ailang/internal/strutil"
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

	// Cached `pi --version` (health.go). Probed lazily, once.
	probe executor.VersionProbe
}

// ExpectedPackage / ExpectedVersion name the pi this executor was written
// against. M2 asserts the running binary matches; M1 only records it.
const (
	ExpectedPackage = "@earendil-works/pi-coding-agent"
	ExpectedVersion = "0.85.1"
)

// New creates a new PiExecutor.
func New(cfg *executor.Config) (*PiExecutor, error) {
	piPath := cfg.PiPath
	if piPath == "" {
		piPath = "pi"
	}

	model := cfg.PiModel
	// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (D2(a)): NO DEFAULT. An empty model is
	// permitted HERE because the coordinator constructs an executor before it
	// knows the task, then supplies Task.Model per task. The fail-loud lives at
	// the point of USE (getModel) rather than construction — checking here would
	// reject the normal path where the model arrives with the task.

	return &PiExecutor{
		piPath:         piPath,
		model:          model,
		timeoutSeconds: cfg.TimeoutSeconds,
		probe:          executor.VersionProbe{CLI: "pi", Path: piPath},
	}, nil
}

// Name returns the executor identifier.
func (e *PiExecutor) Name() string {
	return "pi"
}

// Execute runs a task and returns the result.
func (e *PiExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	if err := e.requireModel(task); err != nil {
		return nil, err
	}
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks, parsing the
// pi NDJSON stream into normalized executor events.
func (e *PiExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	if err := e.requireModel(task); err != nil {
		return nil, err
	}
	res, err := e.executeStreaming(ctx, task, handler)
	if res != nil {
		// Stamped on EVERY result shape (clean, error, timeout, cancel) so a
		// banked failure says which harness failed. Empty when unprobeable.
		res.ExecutorVersion = e.Version(ctx)
		// The EFFECTIVE tool policy (M-AGENT-AILANG-ONLY-EXECUTION M1): what
		// buildPiArgs passed, or the sentinel when pi's defaults applied.
		res.ToolPolicy = effectiveToolPolicy(task)
		res.PolicyDigest = executor.PolicyDigest(task.PolicyPath)
	}
	return res, err
}

func (e *PiExecutor) executeStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	ctx, span := telemetry.StartSpan(ctx, piTracer, "pi.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "pi"),
			attribute.String("executor.model", e.getModel(task)),
			attribute.String("task.workspace", task.Workspace),
			attribute.String("task.directive", strutil.Truncate(task.Directive, 500)),
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

	args, err := buildPiArgs(e.getModel(task), task, directive)
	if err != nil {
		return nil, err
	}
	piPath := e.piPath

	cmd := exec.CommandContext(ctx, piPath, args...)
	proctree.Configure(cmd)
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
	var inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasonTokens int
	var thrashKilledAt int
	var totalCostUSD float64
	var sessionID string
	var turnSpan trace.Span
	var stderrBuf strings.Builder
	// Last settled stopReason seen on a message_end / turn_end. Streaming
	// message_update events carry cumulative partial state, so they are
	// deliberately excluded — only settled events are authoritative.
	var lastStopReason string
	var lastRawStopReason string // provider's own value (0.84+); "" on older wires

	// M-EVAL-COST-AND-SPEED-BUDGETS: speed + cost-kill instrumentation.
	// pi emits per-turn token deltas in message_end (role=assistant), so the
	// budget tally is naturally incremental. costKilled signals breach and
	// triggers an early process kill.
	var firstAttemptMs int64 = -1
	var firstStreamEventAt time.Time
	var costKilled bool

	// M-PI-HARNESS-UPGRADE M2 (D4): drift is recorded or fatal, never silent.
	// An unrecognised event TYPE is upstream adding a feature — counted and
	// banked. A MISSING FIELD a banked metric depends on is our record being
	// wrong — fatal, with the run named as wire_drift.
	unknownEvents := map[string]int{}
	unparsedLines := 0
	var retries piRetries
	var wireDriftErr string

	go func() {
		// executor.LineReader: a Scanner token cap turns one long line into a
		// failed task. Rationale in internal/executor/linescan.go.
		stdoutScanner := executor.NewLineReader(stdout)
		stderrScanner := executor.NewLineReader(stderr)

		// pi emits very large lines (full cumulative state in every message_update).
		// 8MB is sufficient headroom for multi-turn long-output runs.

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
				// Non-JSON preamble (a provider warning, a deprecation notice) is
				// tolerated but COUNTED — banked as pi_unparsed_lines so a stream
				// that is half noise is visible in the row, not just quietly thin.
				unparsedLines++
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

			case "agent_start", "message_start", "agent_settled":
				// Known, no per-event work. agent_settled is the real "done"
				// signal since 0.84; agent_end may carry willRetry=true.

			case "auto_retry_start":
				// pi retries a failed provider call internally (0.84+). Without
				// this, wall-clock and cost inflate with no visible cause.
				retries.observeStart(ev.Attempt, ev.MaxAttempts)

			case "auto_retry_end":
				retries.observeEnd(ev.Attempt, ev.Success)

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
				// M-EVAL-COST-AND-SPEED-BUDGETS: first write/edit = first solution
				// attempt. pi's builtin tools are LOWERCASE in every version; the
				// capitalised comparison this replaced never matched, so this
				// metric fell through to first-text on every pi run before M2.
				if firstAttemptMs < 0 && (ev.ToolName == "write" || ev.ToolName == "edit") {
					firstAttemptMs = time.Since(startTime).Milliseconds()
				}

			case "tool_execution_end":
				output := flattenPiToolResult(ev.Result)
				handler.OnToolResult(ev.ToolName, output)

			case "message_end":
				if ev.Message != nil && ev.Message.StopReason != "" {
					lastStopReason = ev.Message.StopReason
					lastRawStopReason = ev.Message.RawStopReason
				}
				// D4: an assistant message_end WITHOUT usage means the cost record
				// for this run is wrong. Bank nothing silently — name it and fail.
				if ev.Message != nil && ev.Message.Role == "assistant" && ev.Message.Usage == nil && wireDriftErr == "" {
					wireDriftErr = "pi: message_end without usage on an assistant message — the banked token/cost record would be wrong (wire_drift; pinned " + ExpectedPackage + "@" + ExpectedVersion + ")"
				}
				// Per-turn deltas for assistant messages — sum into totals.
				if ev.Message != nil && ev.Message.Role == "assistant" && ev.Message.Usage != nil {
					u := ev.Message.Usage
					inputTokens += u.Input
					// D5: reasoning is inside output on the wire; the Result
					// contract wants them disjoint.
					outputTokens += u.Output - u.Reasoning
					reasonTokens += u.Reasoning
					cacheReadTokens += u.CacheRead
					cacheWriteTokens += u.CacheWrite
					totalCostUSD += u.Cost.Total
					if tp, over := capExceeded(task.MaxTokensPerBench, inputTokens, cacheWriteTokens, outputTokens); over && thrashKilledAt == 0 {
						thrashKilledAt = tp
						proctree.Kill(cmd)
					}
					if chargeCache(task, u.CacheRead, u.CacheWrite) {
						costKilled = true
						proctree.Kill(cmd)
					}
					// M-EVAL-COST-AND-SPEED-BUDGETS: incremental cost tally on per-turn delta.
					if task.Budget != nil && (u.Input > 0 || u.Output > 0) {
						if _, exceeded := task.Budget.Add(u.Input, u.Output); exceeded {
							costKilled = true
							proctree.Kill(cmd)
						}
					}
				}

			case "turn_end":
				if ev.Message != nil && ev.Message.StopReason != "" {
					lastStopReason = ev.Message.StopReason
					lastRawStopReason = ev.Message.RawStopReason
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

			default:
				unknownEvents[ev.Type]++
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
				if thrashKilledAt > 0 {
					finishReason = executor.FinishThrashAborted
					errMsg = fmt.Sprintf("token budget exceeded (%d > %d) — %s", thrashKilledAt, task.MaxTokensPerBench, errMsg)
				}
				if costKilled {
					errMsg = fmt.Sprintf("cost budget exceeded ($%.4f) — %s", task.Budget.KilledAt(), errMsg)
					finishReason = executor.FinishCostExhausted
				}
				return &executor.Result{
					Success:                  false,
					FinishReason:             finishReason,
					ThrashKilledAt:           thrashKilledAt,
					Output:                   transcriptBuf.String(),
					Error:                    errMsg,
					DurationMS:               int(duration.Milliseconds()),
					InputTokens:              inputTokens,
					OutputTokens:             outputTokens,
					ReasonTokens:             reasonTokens,
					CacheReadInputTokens:     cacheReadTokens,
					CacheCreationInputTokens: cacheWriteTokens,
					CostUSD:                  totalCostUSD,
					CostProvenance:           executor.ResolveCostProvenance(task, executor.AuthLaneForModel(task.Model)),
					NumTurns:                 numTurns,
					ToolCallCount:            toolCallCount,
					ToolCalls:                toolCalls,
					SessionID:                sessionID,
					ProviderData:             piProviderData(rawEvents, unknownEvents, unparsedLines, &retries, lastRawStopReason),
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
				finishReason = normalizePiFinishReasonWithRaw(lastStopReason, lastRawStopReason)
			}
			errMsg := ""
			if wireDriftErr != "" {
				// D4: the model may have finished; the RECORD cannot be trusted.
				success = false
				finishReason = executor.FinishWireDrift
				errMsg = wireDriftErr
			}
			if costKilled {
				success = false
				finishReason = executor.FinishCostExhausted
			}
			if thrashKilledAt > 0 && !costKilled {
				success = false
				finishReason = executor.FinishThrashAborted
			}

			span.SetStatus(codes.Ok, "")
			return &executor.Result{
				Success:                  success,
				FinishReason:             finishReason,
				ThrashKilledAt:           thrashKilledAt,
				Output:                   output,
				Error:                    errMsg,
				DurationMS:               int(duration.Milliseconds()),
				InputTokens:              inputTokens,
				OutputTokens:             outputTokens,
				ReasonTokens:             reasonTokens,
				CacheReadInputTokens:     cacheReadTokens,
				CacheCreationInputTokens: cacheWriteTokens,
				CostUSD:                  totalCostUSD,
				CostProvenance:           executor.ResolveCostProvenance(task, executor.AuthLaneForModel(task.Model)),
				NumTurns:                 numTurns,
				ToolCallCount:            toolCallCount,
				ToolCalls:                toolCalls,
				SessionID:                sessionID,
				ProviderData:             piProviderData(rawEvents, unknownEvents, unparsedLines, &retries, lastRawStopReason),
				CostKilledAt:             task.Budget.KilledAt(),
				FirstAttemptMs:           firstAttemptMs,
				SuccessAtMs:              -1,
				TokensPerSec:             tokensPerSec,
			}, nil

		case <-hardTimer.C:
			proctree.Kill(cmd)
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
			proctree.Kill(cmd)
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
				proctree.Kill(cmd)
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
			proctree.Kill(cmd)
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

// requireModel is the D2(a) fail-loud point (M-MODEL-REGISTRY-SINGLE-SOURCE M6).
//
// The check lives at the ENTRY to execution rather than at construction,
// because the coordinator builds an executor before it knows the task and
// then supplies Task.Model per task — rejecting an empty model at
// construction would break the normal path. It lives here rather than inside
// getModel to avoid threading an error through every call site of a helper
// that runs after this guard has already passed.
func (e *PiExecutor) requireModel(task *executor.Task) error {
	if e.getModel(task) == "" {
		return executor.ErrUnresolvedModel("pi", "PiModel")
	}
	return nil
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
