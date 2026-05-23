// Package opencode provides an Executor implementation for the opencode CLI.
//
// opencode CLI emits NDJSON via `opencode run --format json` — a different schema
// from Claude, Gemini, and Codex. See opencode_compat_test.go for the full schema
// documentation with fixture assertions.
//
// Key differences from Codex:
//   - Per-step token deltas: sum step_finish.part.tokens.input/output across events
//     (Codex emits cumulative running totals; opencode emits per-step deltas)
//   - Session resume: --session <sessionID> (not built into the directive)
//   - Model string: "provider/model" format (e.g. "anthropic/claude-haiku-4-5",
//     "ollama/gemma4:latest")
//   - Ollama local models: configure ~/.config/opencode/opencode.jsonc with
//     custom provider block; see testdata/opencode_ollama_config.jsonc
package opencode

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

var opencodeTracer = telemetry.Tracer("executor.opencode")

// OpenCodeExecutor executes tasks using the opencode CLI.
type OpenCodeExecutor struct {
	opencodePath   string
	model          string
	timeoutSeconds int
}

// New creates a new OpenCodeExecutor.
func New(cfg *executor.Config) (*OpenCodeExecutor, error) {
	opencodePath := cfg.OpenCodePath
	if opencodePath == "" {
		opencodePath = "opencode"
	}

	model := cfg.OpenCodeModel
	if model == "" {
		model = "anthropic/claude-haiku-4-5"
	}

	return &OpenCodeExecutor{
		opencodePath:   opencodePath,
		model:          model,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier.
func (e *OpenCodeExecutor) Name() string {
	return "opencode"
}

// Execute runs a task and returns the result.
func (e *OpenCodeExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks, parsing the
// opencode NDJSON stream into normalized executor events.
func (e *OpenCodeExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	ctx, span := telemetry.StartSpan(ctx, opencodeTracer, "opencode.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "opencode"),
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

	// opencode CLI run subcommand with JSON streaming.
	// Reference: opencode_compat_test.go (live-captured schema documentation)
	// Flags:
	//   run                                 non-interactive run subcommand
	//   --format json                       NDJSON output on stdout
	//   --dangerously-skip-permissions      auto-approve tool permissions
	//   --model <provider/model>            model selection (e.g. "anthropic/claude-haiku-4-5")
	//   --session <sessionID>               resume a prior session (for iteration > 1)
	args := []string{
		"run",
		"--format", "json",
		"--dangerously-skip-permissions",
		"--model", e.getModel(task),
	}

	// Resume session if this is a subsequent iteration (M-TRANSCRIPT pattern).
	if task.ResumeSessionID != "" {
		args = append(args, "--session", task.ResumeSessionID)
	}

	// The directive is the positional message argument.
	args = append(args, directive)

	opencodePath := e.opencodePath

	cmd := exec.CommandContext(ctx, opencodePath, args...)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_OPENCODE] Command: %s run --format json --model %s\n", opencodePath, e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_OPENCODE] Workspace: %s\n", task.Workspace)
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
		return nil, fmt.Errorf("failed to start opencode: %w", err)
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
	// ttftTimer fires if no output arrives before the first event (prefill budget).
	// Stopped as soon as the first stdout event is received.
	ttftTimer := time.NewTimer(ttftTimeout)
	defer ttftTimer.Stop()
	// idleCheck only meaningful after first event — generation idle window.
	idleCheck := time.NewTimer(idleTimeout)
	idleCheck.Stop() // paused until first event arrives
	defer idleCheck.Stop()

	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	var firstEventSeen atomic.Bool

	done := make(chan error, 1)
	var transcriptBuf strings.Builder
	var rawEvents []map[string]any
	var numSteps int
	var toolCallCount int
	// opencode emits per-step deltas; sum across step_finish events.
	var inputTokens, outputTokens int
	var totalCostUSD float64
	var stepSpan trace.Span
	var stderrBuf strings.Builder
	var lastSessionID string

	// M-EVAL-COST-AND-SPEED-BUDGETS: speed + cost-kill instrumentation.
	// firstAttemptMs records ms from task start to the first Write/Edit tool
	// call (or the first text event if no tool calls). costKilled signals
	// budget breach during step_finish — we kill the process and exit.
	//
	// NOTE: opencode's step_finish carries per-step token deltas, so the
	// budget tally is naturally incremental and mid-stream cancellation works.
	// Some opencode plugins/providers may not emit step_finish (or emit it
	// only at session end); for those, the tally still works correctly but
	// mid-stream killing degrades to end-of-stream killing.
	var firstAttemptMs int64 = -1
	var firstStreamEventAt time.Time
	var costKilled bool

	// M-EVAL-OS-LONGITUDINAL Phase 1: thrash detection for $0 local models.
	// When cumulative input+output tokens exceed task.MaxTokensPerBench, we
	// kill the subprocess and surface a thrash_aborted error category.
	var thrashKilled bool
	var thrashKilledAtTokens int

	go func() {
		stdoutScanner := bufio.NewScanner(stdout)
		stderrScanner := bufio.NewScanner(stderr)

		const maxScannerBuffer = 1024 * 1024
		stdoutScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)
		stderrScanner.Buffer(make([]byte, 0, maxScannerBuffer), maxScannerBuffer)

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
				// First event: prefill done — stop TTFT timer, start generation idle check.
				ttftTimer.Stop()
				idleCheck.Reset(idleTimeout)
			}

			ev, err := parseOpenCodeEvent(line)
			if err != nil {
				// Non-JSON preamble (status messages, warnings) — skip cleanly.
				continue
			}

			// Track session ID for result (enables --session resume on next iteration).
			if ev.SessionID != "" {
				lastSessionID = ev.SessionID
			}

			// Accumulate raw events for ProviderData (schema-drift tolerance).
			if ev.Raw != nil {
				rawEvents = append(rawEvents, ev.Raw)
			}

			switch ev.Type {
			case "step_start":
				numSteps++
				if firstStreamEventAt.IsZero() {
					firstStreamEventAt = time.Now()
				}
				_, stepSpan = telemetry.StartSpan(ctx, opencodeTracer, "opencode.step",
					trace.WithAttributes(
						attribute.Int("opencode.step_num", numSteps),
						attribute.String("opencode.message_id", ev.Part.MessageID),
					),
				)
				handler.OnTurnStart(numSteps)

			case "text":
				text := ev.Part.Text
				if text != "" {
					transcriptBuf.WriteString(text)
					handler.OnText(text)
					// M-EVAL-COST-AND-SPEED-BUDGETS: first text = candidate solution if
					// no Write/Edit tool calls were made earlier.
					if firstAttemptMs < 0 {
						firstAttemptMs = time.Since(startTime).Milliseconds()
					}
				}

			case "tool_use":
				toolCallCount++
				toolName := ev.Part.Tool
				if os.Getenv("DEBUG_AGENT") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_OPENCODE] tool_use: %s status=%s\n", toolName, ev.Part.State.Status)
				}
				inputStr := string(ev.Part.State.Input)
				if ev.Part.State.Status == "completed" {
					handler.OnToolResult(toolName, ev.Part.State.Output)
				} else {
					handler.OnToolUse(toolName, inputStr)
				}
				// M-EVAL-COST-AND-SPEED-BUDGETS: first Write/Edit tool call = first solution attempt.
				if firstAttemptMs < 0 && (toolName == "Write" || toolName == "Edit") {
					firstAttemptMs = time.Since(startTime).Milliseconds()
				}

			case "step_finish":
				// Per-step delta: sum across all step_finish events.
				inputTokens += ev.Part.Tokens.Input
				outputTokens += ev.Part.Tokens.Output
				totalCostUSD += ev.Part.Cost

				// M-EVAL-COST-AND-SPEED-BUDGETS: incremental cost tally on per-step deltas.
				if task.Budget != nil && (ev.Part.Tokens.Input > 0 || ev.Part.Tokens.Output > 0) {
					if _, exceeded := task.Budget.Add(ev.Part.Tokens.Input, ev.Part.Tokens.Output); exceeded {
						costKilled = true
						_ = cmd.Process.Kill()
					}
				}

				// M-EVAL-OS-LONGITUDINAL Phase 1: thrash detection on cumulative
				// tokens. Local Ollama models have $0 cost so the cost-budget
				// path never trips — this is the only safety net against runaway
				// 2.88M-token thrashing observed in fizzbuzz.
				if task.MaxTokensPerBench > 0 && !thrashKilled {
					if inputTokens+outputTokens > task.MaxTokensPerBench {
						thrashKilled = true
						thrashKilledAtTokens = inputTokens + outputTokens
						fmt.Fprintf(os.Stderr, "[OPENCODE] thrash abort: cumulative tokens %d exceeded MaxTokensPerBench=%d\n",
							thrashKilledAtTokens, task.MaxTokensPerBench)
						_ = cmd.Process.Kill()
					}
				}

				if stepSpan != nil {
					stepSpan.SetAttributes(
						attribute.Int("opencode.input_tokens", ev.Part.Tokens.Input),
						attribute.Int("opencode.output_tokens", ev.Part.Tokens.Output),
						attribute.Float64("opencode.cost_usd", ev.Part.Cost),
					)
					stepSpan.End()
					stepSpan = nil
				}
			}
		}

		done <- cmd.Wait()
	}()

	for {
		select {
		case err := <-done:
			duration := time.Since(startTime)
			span.SetAttributes(
				attribute.Int("opencode.steps", numSteps),
				attribute.Int("opencode.tool_calls", toolCallCount),
				attribute.Int("opencode.input_tokens", inputTokens),
				attribute.Int("opencode.output_tokens", outputTokens),
				attribute.Float64("opencode.cost_usd", totalCostUSD),
			)

			// M-EVAL-COST-AND-SPEED-BUDGETS: speed metrics.
			tokensPerSec := computeTokensPerSec(outputTokens, firstStreamEventAt)

			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				errMsg := fmt.Sprintf("opencode exited with error: %v\nstderr: %s", err, stderrBuf.String())
				success := false
				if costKilled {
					errMsg = fmt.Sprintf("cost budget exceeded ($%.4f) — %s", task.Budget.KilledAt(), errMsg)
				}
				if thrashKilled {
					errMsg = fmt.Sprintf("thrash abort: cumulative tokens %d exceeded MaxTokensPerBench=%d — %s",
						thrashKilledAtTokens, task.MaxTokensPerBench, errMsg)
				}
				return &executor.Result{
					Success:        success,
					Output:         transcriptBuf.String(),
					Error:          errMsg,
					DurationMS:     int(duration.Milliseconds()),
					InputTokens:    inputTokens,
					OutputTokens:   outputTokens,
					CostUSD:        totalCostUSD,
					NumTurns:       numSteps,
					ToolCallCount:  toolCallCount,
					SessionID:      lastSessionID,
					ProviderData:   opencodeProviderData(rawEvents),
					CostKilledAt:   task.Budget.KilledAt(),
					ThrashKilledAt: thrashKilledAtTokens,
					FirstAttemptMs: firstAttemptMs,
					SuccessAtMs:    -1,
					TokensPerSec:   tokensPerSec,
				}, nil
			}

			output := transcriptBuf.String()
			success := output != "" || toolCallCount > 0
			if costKilled || thrashKilled {
				success = false
			}

			span.SetStatus(codes.Ok, "")
			return &executor.Result{
				Success:        success,
				Output:         output,
				DurationMS:     int(duration.Milliseconds()),
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				CostUSD:        totalCostUSD,
				NumTurns:       numSteps,
				ToolCallCount:  toolCallCount,
				SessionID:      lastSessionID,
				ProviderData:   opencodeProviderData(rawEvents),
				CostKilledAt:   task.Budget.KilledAt(),
				ThrashKilledAt: thrashKilledAtTokens,
				FirstAttemptMs: firstAttemptMs,
				SuccessAtMs:    -1,
				TokensPerSec:   tokensPerSec,
			}, nil

		case <-hardTimer.C:
			_ = cmd.Process.Kill()
			span.SetStatus(codes.Error, "hard timeout")
			return &executor.Result{
				Success:        false,
				Error:          fmt.Sprintf("opencode exceeded hard timeout (%v)", timeout),
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
				Error:          fmt.Sprintf("opencode produced no output within %v (prefill timeout)", ttftTimeout),
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
					Error:          fmt.Sprintf("opencode idle for %v mid-generation (no output)", since),
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
				Error:          fmt.Sprintf("opencode cancelled: %v", ctx.Err()),
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
func (e *OpenCodeExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapLocalWorkspace,
	}
}

// CostModel returns a generic default cost model for opencode.
// Actual cost depends on the provider/model string used; opencode reports
// per-step cost in step_finish events which the executor sums into CostUSD.
// The CostModel here is used only for pre-flight estimates when no live data
// is available; the real cost comes from summing step_finish.part.cost.
func (e *OpenCodeExecutor) CostModel() *executor.CostModel {
	return &executor.CostModel{
		ProviderName:    "opencode",
		InputTokenCost:  0.0,
		OutputTokenCost: 0.0,
		CacheReadCost:   0.0,
	}
}

// HealthCheck verifies the opencode binary exists on PATH and responds.
func (e *OpenCodeExecutor) HealthCheck(ctx context.Context) error {
	opencodePath := e.opencodePath
	if _, err := exec.LookPath(opencodePath); err != nil {
		if _, statErr := os.Stat(opencodePath); statErr != nil {
			return fmt.Errorf("opencode CLI not found: %w (install with: npm i -g opencode-ai)", err)
		}
	}
	checkCmd := exec.CommandContext(ctx, opencodePath, "--version")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("opencode --version failed: %w", err)
	}
	return nil
}

// Close releases any resources held by the executor.
func (e *OpenCodeExecutor) Close() error {
	return nil
}

func (e *OpenCodeExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// opencodeEventPart captures the type-specific payload.
type opencodeEventPart struct {
	// Common
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
	Type      string `json:"type"`

	// text event
	Text string `json:"text,omitempty"`

	// tool_use event
	Tool   string `json:"tool,omitempty"`
	CallID string `json:"callID,omitempty"`
	State  struct {
		Status string          `json:"status"`
		Input  json.RawMessage `json:"input"`
		Output string          `json:"output"`
		Title  string          `json:"title"`
	} `json:"state,omitempty"`

	// step_finish event
	Reason string `json:"reason,omitempty"`
	Tokens struct {
		Total     int `json:"total"`
		Input     int `json:"input"`
		Output    int `json:"output"`
		Reasoning int `json:"reasoning"`
		Cache     struct {
			Write int `json:"write"`
			Read  int `json:"read"`
		} `json:"cache"`
	} `json:"tokens,omitempty"`
	Cost float64 `json:"cost,omitempty"`
}

// opencodeNDJSON is the top-level event wrapper.
type opencodeNDJSON struct {
	Type      string            `json:"type"`
	Timestamp int64             `json:"timestamp"`
	SessionID string            `json:"sessionID"`
	Part      opencodeEventPart `json:"part"`

	// Raw preserves full event for ProviderData (schema-drift tolerance).
	Raw map[string]any `json:"-"`
}

// parseOpenCodeEvent parses a single NDJSON line.
// Returns error for non-JSON lines or parse failures; callers skip them.
func parseOpenCodeEvent(line []byte) (*opencodeNDJSON, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, fmt.Errorf("empty line")
	}
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("non-JSON line")
	}
	var ev opencodeNDJSON
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

// computeTokensPerSec returns OutputTokens / generation_seconds, where
// generation_seconds spans from the first stream event to "now". Returns
// 0 if either side is unmeasured. Used by Result.TokensPerSec.
func computeTokensPerSec(outputTokens int, firstStreamEventAt time.Time) float64 {
	if firstStreamEventAt.IsZero() || outputTokens <= 0 {
		return 0
	}
	gen := time.Since(firstStreamEventAt).Seconds()
	if gen <= 0 {
		return 0
	}
	return float64(outputTokens) / gen
}

// opencodeProviderData wraps raw events as Result.ProviderData.
func opencodeProviderData(events []map[string]any) map[string]any {
	if len(events) == 0 {
		return nil
	}
	return map[string]any{
		"opencode_events": events,
	}
}

// Register registers the opencode executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("opencode", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
