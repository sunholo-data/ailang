// Package motoko provides an Executor implementation for the motoko_agent CLI
// (the first production-scale coding agent built on AILANG).
//
// motoko_agent is structurally different from claude/gemini/codex/opencode/pi:
// it is an AILANG-native agent loop (~5,200 LOC of `.ail` modules + 9 published
// motoko-ext-* extension packages) wrapped in a bun TUI. It writes a structured
// session JSONL to ${WORKDIR}/.motoko/logfile/session_*.jsonl during the run
// (NOT to stdout — the TUI consumes stdout for rendering). Schema v1 of that
// JSONL is documented in motoko_agent's design_docs/implemented/motoko_agent/
// m-motoko-eval-instrumentation.md (commits 0c006be + 84fa449 on PR #6).
//
// The adapter:
//   - Spawns `motoko "<task>"` as a subprocess with WORKDIR / MODEL /
//     MOTOKO_CONFIG / MOTOKO_SESSION_ID env vars.
//   - After the run completes, locates the session JSONL by matching the
//     MOTOKO_SESSION_ID we set (or falling back to the newest file in the
//     workdir's .motoko/logfile/ directory).
//   - Parses the JSONL line-by-line into normalized executor.Result fields,
//     preferring the terminal `run_summary` event for totals (input_tokens,
//     output_tokens, cache_read_input_tokens, cache_creation_input_tokens,
//     total_cost_usd, finish_reason, duration_ms) and falling back to summing
//     per-step `thinking` events when run_summary is absent (crash case).
//   - Round-trips unknown fields into Result.ProviderData["motoko_events"]
//     for forward-compat with future schema versions.
//
// Trust boundary: motoko's autonomous bash tool can touch anything reachable
// from the workspace. The cloud Job (M4 of M-MOTOKO-EXECUTOR-ADAPTER) binds
// only OPENROUTER + OPENAI + GEMINI keys — explicitly NOT ANTHROPIC_API_KEY,
// matching the Pi-precedent cost-control rule (see EXECUTOR_SHAPE.md §8).
//
// Cross-references:
//   - EXECUTOR_SHAPE.md: docs/internal/EXECUTOR_SHAPE.md (the contract)
//   - Closest analogue: internal/executor/opencode/ (NDJSON parser pattern)
//   - Closest deployment analogue: docker/Dockerfile.agent-pi (CLI install)
//   - Schema spec: motoko_agent design doc (linked above)
package motoko

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var motokoTracer = telemetry.Tracer("executor.motoko")

// MotokoExecutor executes tasks using the motoko_agent CLI.
type MotokoExecutor struct {
	motokoPath     string
	model          string
	profile        string
	timeoutSeconds int
}

// New creates a new MotokoExecutor.
func New(cfg *executor.Config) (*MotokoExecutor, error) {
	motokoPath := cfg.MotokoPath
	if motokoPath == "" {
		motokoPath = "motoko"
	}

	model := cfg.MotokoModel
	if model == "" {
		model = "openrouter/anthropic/claude-haiku-4-5"
	}

	profile := cfg.MotokoProfile
	if profile == "" {
		profile = "dogfood"
	}

	return &MotokoExecutor{
		motokoPath:     motokoPath,
		model:          model,
		profile:        profile,
		timeoutSeconds: cfg.TimeoutSeconds,
	}, nil
}

// Name returns the executor identifier.
func (e *MotokoExecutor) Name() string {
	return "motoko"
}

// Execute runs a task and returns the result.
func (e *MotokoExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task. Streaming events come from parsing the session
// JSONL file as it grows (M2 will add the streaming goroutine; M1 ships with
// post-completion parse only).
func (e *MotokoExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	ctx, span := telemetry.StartSpan(ctx, motokoTracer, "motoko.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "motoko"),
			attribute.String("executor.model", e.getModel(task)),
			attribute.String("executor.profile", e.profile),
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

	// Generate a deterministic session_id we'll inject via env so we can find
	// the JSONL file without listing the directory. Format mirrors motoko's
	// own derive_session_id() output: "session_<unique>".
	sessionID := "session_" + ourTaskID
	span.SetAttributes(attribute.String("motoko.session_id", sessionID))

	directive := task.Directive
	if task.SystemPrompt != "" {
		directive = task.SystemPrompt + "\n\n" + task.Directive
	}

	// motoko CLI: positional task argument; env vars carry the rest.
	cmd := exec.CommandContext(ctx, e.motokoPath, directive)
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Command: %s <directive>\n", e.motokoPath)
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Workspace: %s\n", task.Workspace)
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Model:     %s\n", e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Profile:   %s\n", e.profile)
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] SessionID: %s\n", sessionID)
	}

	env := executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:        task,
		SessionID:   sessionID,
		Context:     ctx,
		GCPProject:  task.GCPProject,
		GCPLocation: task.GCPLocation,
	})
	// motoko-specific env vars — see motoko_agent docs for semantics.
	env = append(env,
		"MODEL="+e.getModel(task),
		"MOTOKO_CONFIG="+e.profile,
		"MOTOKO_SESSION_ID="+sessionID,
	)
	if task.Workspace != "" {
		env = append(env, "WORKDIR="+task.Workspace)
	}
	cmd.Env = env

	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		// Process failure is NOT necessarily a task failure — the JSONL may
		// still contain a valid run_summary with finish_reason="error".
		// Continue to parse; only fail-hard on unparseable / missing JSONL.
		if os.Getenv("DEBUG_AGENT") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] subprocess exit error: %v\n", err)
		}
	}

	wallDurationMS := int(time.Since(startTime).Milliseconds())

	// Locate the session JSONL motoko wrote during the run.
	jsonlPath, findErr := findSessionJSONL(task.Workspace, sessionID)
	if findErr != nil {
		span.SetStatus(codes.Error, "session jsonl not found")
		return &executor.Result{
			Success:    false,
			Error:      fmt.Sprintf("motoko ran but no session JSONL found: %v", findErr),
			DurationMS: wallDurationMS,
			SessionID:  sessionID,
		}, nil
	}

	result, parseErr := parseSessionJSONL(jsonlPath)
	if parseErr != nil {
		span.SetStatus(codes.Error, "session jsonl parse failed")
		return &executor.Result{
			Success:    false,
			Error:      fmt.Sprintf("motoko session JSONL parse failed: %v", parseErr),
			DurationMS: wallDurationMS,
			SessionID:  sessionID,
		}, nil
	}

	// run_summary may carry its own duration_ms (from motoko's internal
	// wall clock); prefer it when present, otherwise fall back to our
	// subprocess wall time.
	if result.DurationMS == 0 {
		result.DurationMS = wallDurationMS
	}
	if result.SessionID == "" {
		result.SessionID = sessionID
	}

	// Surface metrics for any MetricsHandler observers.
	if mh, ok := handler.(executor.MetricsHandler); ok {
		mh.OnMetrics(executor.ExecutionMetrics{
			NumTurns:     result.NumTurns,
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			CostUSD:      result.CostUSD,
			DurationMS:   result.DurationMS,
			SessionID:    result.SessionID,
			Success:      result.Success,
		})
	}

	span.SetAttributes(
		attribute.Int("ai.tokens_in", result.InputTokens),
		attribute.Int("ai.tokens_out", result.OutputTokens),
		attribute.Float64("ai.cost_usd", result.CostUSD),
		attribute.Int("exec.turns", result.NumTurns),
		attribute.Bool("exec.success", result.Success),
	)
	span.SetStatus(codes.Ok, "")
	return result, nil
}

// Capabilities returns the list of features this executor supports.
func (e *MotokoExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapLocalWorkspace,
		executor.CapStructuredOutput,
	}
}

// CostModel returns a generic default cost model for motoko.
// Actual cost depends on the underlying provider (resolved via OpenRouter for
// motoko-* models). The CostModel here is used only for pre-flight estimates;
// the real cost comes from motoko's run_summary.total_cost_usd field, which
// the parser populates into Result.CostUSD directly.
func (e *MotokoExecutor) CostModel() *executor.CostModel {
	return &executor.CostModel{
		ProviderName:    "motoko",
		InputTokenCost:  0.0,
		OutputTokenCost: 0.0,
		CacheReadCost:   0.0,
	}
}

// HealthCheck verifies the motoko binary exists on PATH and responds.
func (e *MotokoExecutor) HealthCheck(ctx context.Context) error {
	motokoPath := e.motokoPath
	if _, err := exec.LookPath(motokoPath); err != nil {
		if _, statErr := os.Stat(motokoPath); statErr != nil {
			return fmt.Errorf("motoko CLI not found: %w (build from sunholo-data/motoko_agent)", err)
		}
	}
	checkCmd := exec.CommandContext(ctx, motokoPath, "--version")
	if err := checkCmd.Run(); err != nil {
		// motoko --version may not be implemented; fall back to --help which
		// every CLI supports.
		helpCmd := exec.CommandContext(ctx, motokoPath, "--help")
		if helpErr := helpCmd.Run(); helpErr != nil {
			return fmt.Errorf("motoko binary not responsive: --version: %v; --help: %w", err, helpErr)
		}
	}
	return nil
}

// Close releases any resources held by the executor.
func (e *MotokoExecutor) Close() error {
	return nil
}

func (e *MotokoExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// Register registers the motoko executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("motoko", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
