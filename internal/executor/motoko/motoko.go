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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var motokoTracer = telemetry.Tracer("executor.motoko")

// writeMotokoSystemPrompt writes the system prompt to a file INSIDE the workspace
// and returns its absolute path (M-MOTOKO-SYSTEM-ROLE). motoko's headless host
// reads the system prompt from SYSTEM_MD and (index.ts systemPromptForWorkspace)
// REJECTS any path outside the workdir — so the file must live under the
// workspace, not /tmp. Returns "" if workspace is empty.
func writeMotokoSystemPrompt(workspace, content string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("no workspace for system prompt")
	}
	p := filepath.Join(workspace, ".motoko_system.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// MotokoExecutor executes tasks using the motoko_agent CLI.
type MotokoExecutor struct {
	motokoPath     string
	model          string
	profile        string
	timeoutSeconds int

	// Version metadata, populated by HealthCheck via `motoko --version`.
	// All four default to "unknown" if the version query fails (older
	// motoko binaries pre-M2c hang on any flag). Used for telemetry +
	// drift detection across eval runs.
	tuiVersion  string
	gitRev      string
	ailangBuilt string
	motokoRepo  string

	// HealthCheck deduplication (M-MOTOKO-PARALLEL-EXECUTION-ISOLATION
	// v0.18.2 follow-up): without this, parallel eval-suite runs spawn
	// `motoko --version` once PER TASK — N parallel tasks = N concurrent
	// bun startups racing on shared node_modules + .bun/cache. Once-per-
	// executor caching means HealthCheck pays the bun-startup cost ONCE
	// regardless of parallel-N, dramatically reducing the startup-race
	// surface area. The version query result is immutable for the
	// executor's lifetime so caching is safe.
	healthCheckOnce sync.Once
	healthCheckErr  error
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
	effectiveProfile := e.profile
	if p := task.Metadata["motoko_profile"]; p != "" {
		effectiveProfile = p
	}
	ctx, span := telemetry.StartSpan(ctx, motokoTracer, "motoko.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "motoko"),
			attribute.String("executor.model", e.getModel(task)),
			attribute.String("executor.profile", effectiveProfile),
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

	// System-prompt delivery (M-MOTOKO-SYSTEM-ROLE). Historically the AILANG-
	// generated system prompt (task.SystemPrompt — teaching content + agent-mode
	// output-delivery override) was concatenated into the positional task arg, so
	// motoko sent an EMPTY system-role message (confirmed by the request-dump,
	// 2026-06-18) while pi sends a real agentic system message. motoko's headless
	// host reads its system prompt from SYSTEM_MD (a file path that must live
	// INSIDE the workspace). When AILANG_MOTOKO_SYSTEM_ROLE=1, deliver
	// task.SystemPrompt there as a proper system-role message and pass only
	// task.Directive as the task (no user-message duplication). Default (unset)
	// keeps the legacy fold-into-directive behaviour so the rotation is unchanged
	// until A/B-validated.
	directive := task.Directive
	var systemPromptPath string
	// M-MOTOKO-AGENT-SYSTEM-PROMPT (exploratory): the diff vs pi is that motoko enters
	// eval with an EMPTY system role while pi/opencode carry a lean AGENTIC coding prompt
	// (the AILANG teaching is the same for all and lives in the user message — controlled
	// variable). When AILANG_MOTOKO_AGENT_SYSTEM_FILE points to a file, use ITS content as
	// the system-role prompt (a lean agentic prompt, NOT the teaching) and keep the teaching
	// in the user message. This isolates exactly one variable: empty vs lean-agentic system.
	if agentSys := os.Getenv("AILANG_MOTOKO_AGENT_SYSTEM_FILE"); agentSys != "" && task.Workspace != "" {
		if content, rerr := os.ReadFile(agentSys); rerr == nil && len(content) > 0 {
			if p, werr := writeMotokoSystemPrompt(task.Workspace, string(content)); werr == nil {
				systemPromptPath = p
				defer func() { _ = os.Remove(p) }()
			}
		}
		if task.SystemPrompt != "" {
			directive = task.SystemPrompt + "\n\n" + task.Directive // teaching stays in the user message
		}
	} else if task.SystemPrompt != "" {
		if os.Getenv("AILANG_MOTOKO_SYSTEM_ROLE") == "1" && task.Workspace != "" {
			if p, werr := writeMotokoSystemPrompt(task.Workspace, task.SystemPrompt); werr == nil {
				systemPromptPath = p
				defer func() { _ = os.Remove(p) }()
			} else {
				directive = task.SystemPrompt + "\n\n" + task.Directive // fallback on write error
			}
		} else {
			directive = task.SystemPrompt + "\n\n" + task.Directive // legacy default
		}
	}

	// motoko CLI: positional task argument; env vars carry the rest.
	// --headless forces batch mode (MOTOKO_HEADLESS=1) so motoko runs without the
	// interactive bun TUI and emits the session JSONL the adapter parses below.
	// Without it, `motoko <directive>` launches the TUI, which hangs with no TTY
	// and never writes events → 0-byte JSONL → "terminated without emitting
	// run_summary" + step-budget hang. The motoko_agent feat/ollama-local-profile
	// branch made headless opt-in (it was implicit at f7b26c8, the commit this
	// adapter was first validated against). See M-MOTOKO-OLLAMA-LOOP-CONVERGENCE.
	// M-MOTOKO-RIG-WEDGE-FIX (2026-06-29): make a hung motoko impossible to wedge
	// the rig. Before this, the main run used the incoming ctx (no per-benchmark
	// deadline) with a bare cmd.Run(), so a hung subprocess blocked cmd.Run()
	// FOREVER — which also hung the eval-suite goroutine waiting on it. A hung
	// trial's env-server then squatted the fixed ENV_PORT=8080 for 10h, silently
	// crashing every later spawn with "no run_summary". Defense in depth:
	//   1) clear any orphaned env-server still holding 8080 before we spawn;
	//   2) wrap the run in a hard timeout so cmd.Run() can NEVER block forever;
	//   3) Setpgid + a group-kill Cancel so the timeout reaps the env-server child
	//      too (a bare Process.Kill leaves it orphaned on 8080).
	// Proper upstream fix is an ephemeral env-server port (see ENV_PORT note above).
	e.clearStalePort8080()
	// Bound by the PER-TASK agent budget (task.Timeout = --agent-timeout / benchmark
	// spec.Timeout, set per-run by the eval runner). NOT e.timeoutSeconds: that is the
	// executor factory default (300s) and is NOT updated per-run, so using it killed long
	// agent runs (docx) mid-step-3. e.timeoutSeconds is only a floor; the 3h ceiling is the
	// hung-run backstop so cmd.Run() can never block forever.
	runTimeout := task.Timeout
	// Neither e.timeoutSeconds (factory default 300s) nor the --agent-timeout default (60s) is a
	// reliable per-run budget, so an implausibly-short value is treated as "unset": fall back to a
	// generous 3h hang-backstop rather than killing a normal (slow) agent run mid-step. The
	// rig-wedge fix's job is to bound a genuine HANG, not to enforce a tight budget; a real
	// explicit budget (e.g. --agent-timeout 7200 for docx) is >= the floor and is respected.
	if runTimeout < 5*time.Minute {
		runTimeout = 3 * time.Hour
	}
	runCtx, cancelRun := context.WithTimeout(ctx, runTimeout)
	defer cancelRun()
	cmd := exec.CommandContext(runCtx, e.motokoPath, "--headless", directive)
	setProcessGroup(cmd) // own process group so the env-server child dies with it
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = killProcessGroup(cmd.Process.Pid)
		}
		return nil
	}
	cmd.WaitDelay = 10 * time.Second // let the group-kill land before Wait returns
	if task.Workspace != "" {
		cmd.Dir = task.Workspace
	}

	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Command: %s <directive>\n", e.motokoPath)
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Workspace: %s\n", task.Workspace)
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Model:     %s\n", e.getModel(task))
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] Profile:   %s\n", effectiveProfile)
		fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] SessionID: %s\n", sessionID)
	}

	// M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2): create a per-task
	// AILANG cache dir so parallel motoko sessions don't race on writes to
	// MOTOKO_REPO/src/core/.ailang/cache/compile/.../core.gob. Pre-v0.18.2,
	// 2+ parallel `ailang run` invocations against the same project would
	// both detect fresh sources, both compile, both os.WriteFile() the same
	// .gob — last writer wins, partial writes corrupted the file, downstream
	// reads crashed before runtime initialization → 0-byte JSONL → adapter
	// reported "motoko terminated without emitting run_summary (likely
	// crash)". This isolates the write target without copying the source
	// tree (which would also pull in 200MB+ of node_modules/dist). Phase 1
	// findings: design_docs/planned/v0_18_2/m-motoko-parallel-execution-
	// isolation.md
	taskCacheDir, taskCacheCleanup, err := setupTaskCacheDir(sessionID)
	if err != nil {
		return nil, fmt.Errorf("setup per-task cache dir: %w", err)
	}
	defer taskCacheCleanup()

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
		"MOTOKO_HEADLESS=1", // batch mode (no interactive TUI) — see --headless above
		"MOTOKO_CONFIG="+effectiveProfile,
		"MOTOKO_SESSION_ID="+sessionID,
		"AILANG_CACHE_DIR="+taskCacheDir,
		// M-MOTOKO-SYSTEM-ROLE: when set, motoko reads its system-role message
		// from this file (must be inside the workspace). Empty string is a no-op.
		"SYSTEM_MD="+systemPromptPath,
		// ENV_PORT must match the port motoko's backend.ail connects to. On the
		// feat/ollama-local-profile branch the AILANG core dials the STATIC
		// cfg.url (:8080 in the ollama/dogfood profiles), so an ephemeral bind
		// (ENV_PORT=0) leaves the core unable to reach its own env-server → no AI
		// calls → 0-event JSONL → hang. Pin it to 8080 to match cfg.url.
		//
		// NOTE: the prior ENV_PORT=0 was a parallel-spawn hardening
		// (M-MOTOKO-EVAL-HARNESS-HARDENING, 2026-05-08) so N concurrent sessions
		// don't race on a fixed port. The rig runs motoko at --parallel 1, so a
		// fixed port is safe here. The proper cross-repo fix (for parallel too)
		// is to have backend.ail connect to the port startEnvServer() actually
		// bound, rather than the static cfg.url — tracked for a motoko_agent PR.
		"ENV_PORT=8080",
	)
	if task.Workspace != "" {
		env = append(env, "WORKDIR="+task.Workspace)
	}
	// M-OLLAMA-PER-MODEL-MAX-TOKENS: forward the model's declared output budget
	// (models.yml max_output_tokens) so motoko's ollama /v1 request uses the model's
	// strength instead of std/ai's 4096 default — qwen3.6 reasons ~4k+ tokens and
	// truncates (finish=length) before the tool call at 4096. resolveOllamaMaxTokens
	// reads this env (override > floor). Needs the var in motoko's RuntimeProcess
	// env allowlist (motoko_agent PR); the 16384 floor covers it until then.
	if task.MaxOutputTokens > 0 {
		env = append(env, fmt.Sprintf("AILANG_OLLAMA_MAX_TOKENS=%d", task.MaxOutputTokens))
	}
	// M-MOTOKO-EVAL-HARNESS-HARDENING M5a (gaps #3, #9): forward cost rates
	// from Task.Budget (sourced from models.yml by the eval harness) so
	// motoko's cost_warning + cost_exhausted thresholds fire and run_summary
	// reports a non-zero cost_usd. Pre-M5a, motoko's profile had no cost_rates
	// for openrouter/anthropic models, leading to CostUSD=0 across every
	// motoko-* model. Source-of-truth flow: AILANG models.yml → CostBudget
	// (per-1K USD) → motoko env vars (per-1M millicents).
	if task.Budget != nil {
		if task.Budget.InputPer1K > 0 {
			// per-1K USD × 1e8 = per-1M millicents
			//   (×1000 for K→M, ×100 for $→¢, ×1000 for ¢→m¢)
			inputMillicents := int64(task.Budget.InputPer1K * 1e8)
			env = append(env, fmt.Sprintf("MOTOKO_COST_INPUT_PER_1M_MILLICENTS=%d", inputMillicents))
		}
		if task.Budget.OutputPer1K > 0 {
			outputMillicents := int64(task.Budget.OutputPer1K * 1e8)
			env = append(env, fmt.Sprintf("MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS=%d", outputMillicents))
		}
		// M-EVAL-SWEET-SPOT-FOLLOWUP (v0.19.0): forward the hard cost cap so
		// motoko's internal budget gate actually fires `finish_reason=
		// "cost_exhausted"` when the cumulative cost crosses it. Pre-this,
		// the cap was recorded as metadata only — gemma-4-26b famously ran
		// to $0.94 against a $0.50 cap on balanced_parens (43 min, 222
		// turns) because motoko never knew about the cap. Source-of-truth
		// flow: models.yml budgets.max_cost_usd → CostBudget.MaxUSD →
		// AI_MAX_COST_USD_CENTS (motoko reads this in src/core/config.ail).
		// Motoko expects cents; CostBudget.MaxUSD is dollars → ×100.
		if task.Budget.MaxUSD > 0 {
			maxUSDCents := int64(task.Budget.MaxUSD * 100)
			if maxUSDCents > 0 {
				env = append(env, fmt.Sprintf("AI_MAX_COST_USD_CENTS=%d", maxUSDCents))
			}
		}
	}
	cmd.Env = env

	// Capture subprocess stderr to a per-task file. When the subprocess
	// crashes BEFORE writing JSONL (the v0.18.2 H4 dur=0 pattern), the
	// JSONL parser produces "motoko terminated without emitting run_summary"
	// — but the actual crash reason is in stderr. Pre-fix: stderr was
	// dropped.
	stderrBuf := &strings.Builder{}
	cmd.Stderr = stderrBuf

	// M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2) Phase 2 debugging:
	// also tee stderr to a per-task file so we can see what failing parallel
	// spawns actually output. Always-on (cheap, files are small).
	stderrLogPath := filepath.Join(os.TempDir(), "motoko-stderr-"+strings.TrimPrefix(sessionID, "session_")+".log")
	stderrFile, fileErr := os.Create(stderrLogPath)
	if fileErr == nil {
		cmd.Stderr = io.MultiWriter(stderrBuf, stderrFile)
		defer stderrFile.Close()
	}

	startTime := time.Now()
	if err := cmd.Run(); err != nil {
		// Process failure is NOT necessarily a task failure — the JSONL may
		// still contain a valid run_summary with finish_reason="error".
		// Continue to parse; only fail-hard on unparseable / missing JSONL.
		if os.Getenv("DEBUG_AGENT") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] subprocess exit error: %v\n", err)
			fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] stderr file: %s (size: %d)\n", stderrLogPath, stderrBuf.Len())
			if stderrBuf.Len() > 0 {
				fmt.Fprintf(os.Stderr, "[DEBUG_MOTOKO] subprocess stderr (last 2KB):\n%s\n",
					tailString(stderrBuf.String(), 2048))
			}
		}
	}

	wallDurationMS := int(time.Since(startTime).Milliseconds())

	// Locate the session JSONL motoko wrote during the run.
	jsonlPath, findErr := findSessionJSONL(task.Workspace, sessionID, e.motokoRepo)
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

// HealthCheck verifies the motoko binary exists, is executable, and (when
// available) reports its version + git rev.
//
// HISTORY: motoko had no `--version` mode prior to M-MOTOKO-EVAL-HARNESS-
// HARDENING (v0.18.1) — every flag was treated as task input by the agent
// loop and would spawn an LLM call (and hang waiting for the TUI). M2c
// added `motoko --version` which now exits 0 with structured key=value
// output (tui_version, git_rev, ailang_built, motoko_repo).
//
// HealthCheck:
//  1. Verifies binary existence + executability (always required)
//  2. Verifies OPENROUTER_API_KEY is set (wrapper pre-flight requirement)
//  3. Calls `motoko --version` with a 5s timeout — if it succeeds, the
//     version + git_rev are stashed in MotokoExecutor for telemetry.
//     Failure here is NON-FATAL (older motoko binaries pre-M2c hang on
//     any flag; we degrade to "version unknown" rather than refusing
//     the executor).
func (e *MotokoExecutor) HealthCheck(ctx context.Context) error {
	// M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2): cache the result
	// once-per-executor. Pre-cache, the eval harness called HealthCheck
	// per-task, which spawned `motoko --version` per-task — at parallel-N,
	// N concurrent bun startups raced on shared node_modules + .bun/cache,
	// causing N-1 of them to die with exit 1 silently (the dur=0 pattern
	// in Phase 1 captures). Caching means we pay the bun-startup cost
	// once and parallel siblings see the cached result.
	e.healthCheckOnce.Do(func() {
		e.healthCheckErr = e.runHealthCheck(ctx)
	})
	return e.healthCheckErr
}

// runHealthCheck performs the actual one-time validation. Called from
// HealthCheck under sync.Once so it runs exactly once per executor lifetime.
func (e *MotokoExecutor) runHealthCheck(ctx context.Context) error {
	motokoPath := e.motokoPath
	resolvedPath := motokoPath
	if abs, err := exec.LookPath(motokoPath); err == nil {
		resolvedPath = abs
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("motoko CLI not found at %q: %w (build from sunholo-data/motoko_agent or set MotokoPath)", motokoPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("motoko path %q is a directory, expected an executable", motokoPath)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("motoko binary at %q is not executable (chmod +x)", motokoPath)
	}
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set — motoko routes ALL models via OpenRouter; set this env var or expect every Execute to fail at the wrapper's pre-flight check")
	}

	// Best-effort version query (M-MOTOKO-EVAL-HARNESS-HARDENING M2c). Older
	// motoko binaries (pre-M2c) treat --version as task input and hang. The
	// 5s timeout caps that worst case; on timeout/error we leave version
	// fields at their default ("unknown") and proceed.
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, vErr := exec.CommandContext(versionCtx, motokoPath, "--version").Output()
	if vErr == nil {
		e.parseVersionOutput(string(out))
	}

	// M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2) M4-M5: warn loudly
	// about operational gotchas that wasted hours of debugging this sprint.
	// Both are warnings (stderr), NOT errors — they don't block execution.
	// The user can ignore them at their own risk.
	e.warnIfStaleBunProcesses()
	if e.motokoRepo != "" {
		e.warnIfStaleAilangLock(e.motokoRepo)
	}

	return nil
}

// warnIfStaleBunProcesses scans for lingering bun processes that hold ports
// in the wrapper's pick_free_port range (18080-18099). These are typically
// orphaned from interrupted prior runs; they cause new motoko spawns to hit
// EADDRINUSE silently if the wrapper's lsof probe races with them. This
// sprint's investigation lost ~2 hours to exactly this scenario before the
// stale processes were noticed and killed. Fix: warn the operator with a
// one-liner remediation command.
func (e *MotokoExecutor) warnIfStaleBunProcesses() {
	out, err := exec.Command("lsof", "-i", "-P").Output()
	if err != nil {
		return // lsof not available; not critical
	}
	// Look for TCP LISTEN sockets in the 18080-18099 port range (motoko
	// wrapper's reserved range for env-server).
	bunPids := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "bun") || !strings.Contains(line, "(LISTEN)") {
			continue
		}
		// Match :180XX where XX is 80-99
		if !strings.Contains(line, ":1808") && !strings.Contains(line, ":1809") {
			continue
		}
		// Extract PID (column 2 in lsof output).
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			bunPids[fields[1]] = true
		}
	}
	if len(bunPids) > 0 {
		pids := make([]string, 0, len(bunPids))
		for pid := range bunPids {
			pids = append(pids, pid)
		}
		fmt.Fprintf(os.Stderr, "[motoko/healthcheck] WARNING: %d stale bun process(es) hold ports in motoko's range (18080-18099): PIDs %s\n",
			len(bunPids), strings.Join(pids, ", "))
		fmt.Fprintf(os.Stderr, "[motoko/healthcheck]          These can cause parallel motoko spawns to hit EADDRINUSE. Cleanup: pkill -9 -f 'bun.*src/tui'\n")
	}
}

// clearStalePort8080 kills any orphaned motoko host (bun running src/tui) still
// LISTENing on the FIXED env-server port 8080 before we spawn. The eval adapter
// pins ENV_PORT=8080 and the rig runs --parallel 1, so a holder at spawn time is
// never a legitimate concurrent run — it is an orphan from a crashed/SIGKILLed
// prior run. Left in place it makes THIS run silently crash with "no run_summary"
// (the 10h rig-wedge of 2026-06-29). Best-effort: if lsof/ps are absent (Windows)
// or nothing holds 8080, this is a no-op. It only kills a confirmed motoko host —
// an unrelated service on 8080 is warned about, never killed.
func (e *MotokoExecutor) clearStalePort8080() {
	out, err := exec.Command("lsof", "-nP", "-iTCP:8080", "-sTCP:LISTEN", "-t").Output()
	if err != nil || len(out) == 0 {
		return
	}
	for _, pidStr := range strings.Fields(string(out)) {
		pid, perr := strconv.Atoi(pidStr)
		if perr != nil {
			continue
		}
		desc := ""
		if cmdOut, cerr := exec.Command("ps", "-o", "command=", "-p", pidStr).Output(); cerr == nil {
			desc = strings.TrimSpace(string(cmdOut))
		}
		if !strings.Contains(desc, "bun") || !strings.Contains(desc, "src/tui") {
			fmt.Fprintf(os.Stderr, "[motoko/healthcheck] WARNING: port 8080 held by non-motoko PID %d (%s) — not killing; this run may fail to bind its env-server\n", pid, desc)
			continue
		}
		fmt.Fprintf(os.Stderr, "[motoko/healthcheck] LOUD: killing STALE motoko env-server PID %d squatting port 8080 (orphan from a crashed/hung run; would otherwise crash this run with 'no run_summary')\n", pid)
		if kerr := killProcessGroup(pid); kerr != nil {
			_ = killProcess(pid)
		}
	}
}

// warnIfStaleAilangLock checks whether motoko's ailang.lock matches the
// current disk state of its dependencies. When the operator publishes a new
// version of an extension package (e.g. ran `ailang publish` for v0.1.1
// while motoko's lock still records v0.1.0), the lock-vs-disk drift causes
// AILANG type-checking to fail with cryptic effect-row mismatches in
// registry_generated.ail. This sprint's investigation lost ~1 hour to
// exactly this scenario.
func (e *MotokoExecutor) warnIfStaleAilangLock(motokoRepo string) {
	lockPath := filepath.Join(motokoRepo, "ailang.lock")
	if _, err := os.Stat(lockPath); err != nil {
		return // no lock file, nothing to compare
	}
	// Don't try to recompute hashes here — too expensive for HealthCheck.
	// Instead, check the lock's mtime against the package source mtimes;
	// if any package source is newer than the lock, the lock is stale.
	lockInfo, err := os.Stat(lockPath)
	if err != nil {
		return
	}
	packagesDir := filepath.Dir(filepath.Dir(motokoRepo)) + "/ailang-packages/packages"
	if _, err := os.Stat(packagesDir); err != nil {
		return // ailang-packages not at the canonical sibling path; skip
	}
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tomlPath := filepath.Join(packagesDir, entry.Name(), "ailang.toml")
		tomlInfo, err := os.Stat(tomlPath)
		if err != nil {
			continue
		}
		if tomlInfo.ModTime().After(lockInfo.ModTime()) {
			fmt.Fprintf(os.Stderr, "[motoko/healthcheck] WARNING: %s/ailang.toml is newer than %s\n",
				entry.Name(), lockPath)
			fmt.Fprintf(os.Stderr, "[motoko/healthcheck]          Stale lock causes effect-row mismatch errors. Fix: cd %s && ailang lock && ailang generate-extension-registry\n",
				motokoRepo)
			return // one warning is enough; don't spam per-package
		}
	}
}

// parseVersionOutput populates the executor's version fields from the
// `motoko --version` key=value output. Lines that don't match the expected
// format are silently ignored.
func (e *MotokoExecutor) parseVersionOutput(out string) {
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key, val := line[:idx], strings.TrimSpace(line[idx+1:])
		switch key {
		case "tui_version":
			e.tuiVersion = val
		case "git_rev":
			e.gitRev = val
		case "ailang_built":
			e.ailangBuilt = val
		case "motoko_repo":
			e.motokoRepo = val
		}
	}
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
