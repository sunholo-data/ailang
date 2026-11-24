// Package agy provides an Executor for Google's Antigravity CLI (`agy`), the
// Google lane that edits the local workspace — on a Google AI Pro/Ultra
// subscription (flat rate) or on the locked Gemini API key (metered).
//
// Design: design_docs/planned/m-agy-executor.md. Wire schema and fixtures:
// events.go and testdata/. Auth routes and the permission posture: route.go.
//
// Three facts the parser must not trust away (measured on agy 1.5.15):
//   - A soft-denied tool exits 0 with status SUCCESS; the denial shows only in
//     result.denied_actions. Banked as a failure.
//   - A missing sign-in exits 1 with status ERROR "authentication failed or
//     timed out", after printing a login URL to stderr.
//   - Any invocation may start a background self-update unless
//     AGY_CLI_DISABLE_AUTO_UPDATE=true is set (routeEnv sets it).
package agy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/proctree"
	"github.com/sunholo-data/ailang/internal/strutil"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var agyTracer = telemetry.Tracer("executor.agy")

// ExpectedVersion is the agy this executor's parser was written against, and
// the version tools/agy/install.sh pins. HealthCheck refuses any other.
const ExpectedVersion = "1.5.15"

// printTimeoutMargin is how much earlier than the hard deadline agy's own
// --print-timeout fires, so agy can exit cleanly (code 3) before we kill it.
const printTimeoutMargin = 15 * time.Second

// AgyExecutor executes tasks with the agy CLI.
type AgyExecutor struct {
	agyPath        string
	model          string
	timeoutSeconds int
	probe          executor.VersionProbe
}

// New creates an AgyExecutor. An empty model is allowed here because the
// coordinator supplies Task.Model per task; requireModel fails loudly at use.
func New(cfg *executor.Config) (*AgyExecutor, error) {
	path := cfg.AgyPath
	if path == "" || path == "agy" {
		path = defaultAgyPath()
	}
	return &AgyExecutor{
		agyPath:        path,
		model:          cfg.AgyModel,
		timeoutSeconds: cfg.TimeoutSeconds,
		probe: executor.VersionProbe{CLI: "agy", Path: path,
			Env: []string{"AGY_CLI_DISABLE_AUTO_UPDATE=true"}},
	}, nil
}

// defaultAgyPath resolves the agy binary: the pinned install of
// tools/agy/install.sh (config.AgyBinFor), then PATH, then ~/.local/bin.
func defaultAgyPath() string {
	h, herr := os.UserHomeDir()
	// The pinned install first: no interactive session runs it, so nothing
	// self-updates it (config.AgyBinFor). PATH and ~/.local/bin are fallbacks
	// for an install that predates it; checkPin refuses them unless they
	// happen to be the pinned version.
	if herr == nil {
		if p := config.AgyBinFor(h); fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath("agy"); err == nil {
		return p
	}
	if herr == nil {
		if p := filepath.Join(h, ".local", "bin", "agy"); fileExists(p) {
			return p
		}
	}
	return "agy"
}

// Name returns the executor identifier.
func (e *AgyExecutor) Name() string { return "agy" }

// Execute runs a task and returns the result.
func (e *AgyExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}

// ExecuteStreaming runs a task with real-time event callbacks.
func (e *AgyExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	if task == nil {
		return nil, errors.New("agy: nil task")
	}
	if task.EvalContainment != nil {
		return nil, executor.ErrEvalContainmentAdmissionClosed
	}
	if e.getModel(task) == "" {
		return nil, executor.ErrUnresolvedModel("agy", "AgyModel")
	}
	// The pin is enforced on EVERY run, not only where a caller remembers
	// HealthCheck: `ailang exec agy` (the mission lane probe) never called it, so
	// a self-updated binary ran unpinned (2026-11-24). The probe is cached per
	// executor, so this is one `agy --version` per instance.
	if err := e.HealthCheck(ctx); err != nil {
		return nil, err
	}
	res, err := e.executeStreaming(ctx, task, handler)
	if res != nil {
		res.ExecutorVersion = e.Version(ctx)
		if res.ToolPolicy == nil {
			// Set by executeStreaming once the toolchain is known; a run that failed
			// before that point still banks the posture it would have used.
			res.ToolPolicy = effectiveToolPolicy(task, nil)
		}
		res.PolicyDigest = executor.PolicyDigest(task.PolicyPath)
	}
	return res, err
}

// buildArgs is the agy command line for one run.
func buildArgs(cliModel, directive string, printTimeout time.Duration) []string {
	args := []string{
		"--model", cliModel,
		"--mode", "accept-edits",
		"--sandbox",
		"--output-format", "stream-json",
		"--disable-slash-commands",
	}
	if printTimeout > 0 {
		args = append(args, "--print-timeout", fmt.Sprintf("%ds", int(printTimeout.Seconds())))
	}
	return append(args, "-p", directive)
}

func (e *AgyExecutor) executeStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	model := e.getModel(task)
	route, cliModel := splitRoute(model)

	ctx, span := telemetry.StartSpan(ctx, agyTracer, "agy.execute",
		trace.WithAttributes(
			attribute.String("executor.name", "agy"),
			attribute.String("executor.model", model),
			attribute.String("agy.route", route),
			attribute.String("task.workspace", task.Workspace),
			attribute.String("task.directive", strutil.Truncate(task.Directive, 500)),
		),
	)
	defer span.End()
	if ctxHandler, ok := handler.(executor.ContextAwareHandler); ok {
		ctxHandler.SetContext(ctx)
	}
	taskID := task.ID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	home, err := routeHome(route)
	if err != nil {
		return nil, err
	}
	tc, err := toolchainFor(home)
	if err != nil {
		return nil, err
	}
	allow, deny := permissionRules(task, tc.writable())
	if err := ensureSettings(home, route, allow, deny); err != nil {
		return nil, err
	}
	routeVars, err := routeEnv(route, home, tc)
	if err != nil {
		return nil, err
	}

	directive := task.Directive
	if task.SystemPrompt != "" {
		directive = task.SystemPrompt + "\n\n" + task.Directive
	}

	timeout := task.Timeout
	if timeout == 0 {
		timeout = time.Duration(e.timeoutSeconds) * time.Second
	}
	if timeout <= 0 {
		return nil, errors.New("agy: no timeout (Task.Timeout and Config.TimeoutSeconds are both zero); refusing an unbounded run")
	}
	printTimeout := timeout - printTimeoutMargin
	if printTimeout < 5*time.Second {
		printTimeout = 0 // too short to matter; the hard deadline alone bounds it
	}

	cmd := exec.CommandContext(ctx, e.agyPath, buildArgs(cliModel, directive, printTimeout)...)
	proctree.Configure(cmd)
	if task.Workspace != "" {
		// agy scopes "workspace" writes by the RESOLVED cwd. A symlinked spelling
		// in PWD (macOS /var -> /private/var) lets the model write to a path agy
		// then judges outside the workspace and soft-denies (measured 2026-11-23:
		// write_file(Edit) denied in a t.TempDir, intermittently, by which
		// spelling the model chose). One spelling for both closes it.
		ws := task.Workspace
		if r, err := filepath.EvalSymlinks(ws); err == nil {
			ws = r
		}
		cmd.Dir = ws
		routeVars = append(routeVars, "PWD="+ws)
	}
	env, err := executor.BuildEnvironment(executor.EnvironmentOptions{
		Task:        task,
		ExecutorEnv: routeVars,
		SessionID:   taskID,
		Executor:    "agy",
		Model:       model,
		Context:     ctx,
	})
	if err != nil {
		return nil, err
	}
	cmd.Env = env
	envNames := executor.EnvNamesDigest(env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("agy: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("agy: stderr pipe: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start agy (%s): %w (install with: make install-agy)", e.agyPath, err)
	}

	st := &streamState{toolCalls: map[string]int{}, toolSteps: map[int]bool{}, unknownEvents: map[string]int{}}
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	done := make(chan error, 1)
	stderrDone := make(chan struct{})

	go func() {
		sr := executor.NewLineReader(stderr)
		for sr.Scan() {
			st.stderr.WriteString(sr.Text())
			st.stderr.WriteByte('\n')
		}
		close(stderrDone)
	}()
	go func() {
		lr := executor.NewLineReader(stdout)
		for lr.Scan() {
			lastActivity.Store(time.Now().UnixNano())
			if st.observe(lr.Bytes(), task, handler) {
				proctree.Kill(cmd)
			}
		}
		<-stderrDone
		done <- cmd.Wait()
	}()

	hard := time.NewTimer(timeout)
	defer hard.Stop()
	idleTimeout := task.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 5 * time.Minute
	}
	idle := time.NewTicker(idleTimeout / 4)
	defer idle.Stop()

	finish := func(r *executor.Result) *executor.Result {
		r.DurationMS = int(time.Since(start).Milliseconds())
		r.EnvNamesDigest = envNames
		r.ToolPolicy = effectiveToolPolicy(task, tc.writable())
		r.SuccessAtMs = -1
		r.FirstAttemptMs = st.firstAttemptMs(start)
		r.CostKilledAt = task.Budget.KilledAt()
		st.fill(r, task, model, route)
		span.SetAttributes(
			attribute.Int("agy.tool_calls", r.ToolCallCount),
			attribute.Int("agy.input_tokens", r.InputTokens),
			attribute.Int("agy.output_tokens", r.OutputTokens),
			attribute.Float64("agy.cost_usd", r.CostUSD),
			attribute.String("agy.cost_provenance", string(r.CostProvenance)),
		)
		if r.Success {
			span.SetStatus(codes.Ok, "")
		} else {
			span.SetStatus(codes.Error, strutil.Truncate(r.Error, 200))
		}
		return r
	}

	for {
		select {
		case waitErr := <-done:
			// The leader exited; sweep anything it left in its process group.
			_ = proctree.KillGroup(cmd.Process.Pid)
			return finish(st.verdict(waitErr, route)), nil
		case <-hard.C:
			joinErr := killAndJoin(cmd, done, terminationGrace)
			return finish(&executor.Result{Success: false, FinishReason: executor.FinishTimeout,
				Error: fmt.Sprintf("agy exceeded hard timeout (%v)", timeout)}), joinErr
		case <-idle.C:
			if since := time.Since(time.Unix(0, lastActivity.Load())); since > idleTimeout {
				joinErr := killAndJoin(cmd, done, terminationGrace)
				return finish(&executor.Result{Success: false, FinishReason: executor.FinishIdleKilled,
					Error: fmt.Sprintf("agy produced no output for %v", since.Round(time.Second))}), joinErr
			}
		case <-ctx.Done():
			joinErr := killAndJoin(cmd, done, terminationGrace)
			return finish(&executor.Result{Success: false, FinishReason: executor.FinishError,
				Error: fmt.Sprintf("agy cancelled: %v", ctx.Err())}), joinErr
		}
	}
}

// terminationGrace bounds the wait for a killed agy to be reaped.
var terminationGrace = 10 * time.Second

// killAndJoin kills agy's process group and waits, bounded, for the reader goroutine
// to see it exit (stdout closed, Wait returned). The agy token file refreshes itself,
// so a cloud job may only write it back and release its lease once nothing that could
// refresh it is alive: a failed join returns executor.ErrProcessTerminationUnconfirmed
// and the credential is quarantined (design_docs/planned/m-agy-cloud-executor.md §2).
// A grandchild holding stdout open counts as alive, deliberately.
func killAndJoin(cmd *exec.Cmd, done <-chan error, grace time.Duration) error {
	killErr := proctree.KillGroup(cmd.Process.Pid)
	select {
	case <-done:
		return nil
	case <-time.After(grace):
		return fmt.Errorf("%w: agy did not exit within %v of being killed (kill: %v)",
			executor.ErrProcessTerminationUnconfirmed, grace, killErr)
	}
}

// Capabilities returns the features this executor supports.
func (e *AgyExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{executor.CapStreaming, executor.CapLocalWorkspace, executor.CapToolControl}
}

// CostModel returns the registry rate card for the configured model.
func (e *AgyExecutor) CostModel() *executor.CostModel {
	return executor.HarnessCostModel("agy", e.model)
}

// Close releases nothing; each run is one subprocess.
func (e *AgyExecutor) Close() error { return nil }

func (e *AgyExecutor) getModel(task *executor.Task) string {
	if task != nil && task.Model != "" {
		return task.Model
	}
	return e.model
}

// HealthCheck verifies agy exists and is the pinned version.
func (e *AgyExecutor) HealthCheck(ctx context.Context) error {
	if _, err := exec.LookPath(e.agyPath); err != nil && !fileExists(e.agyPath) {
		return fmt.Errorf("agy CLI not found (%s): %w — install with: make install-agy", e.agyPath, err)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	v, err := e.probe.Raw(probeCtx)
	if err != nil {
		return err
	}
	if v != ExpectedVersion {
		return fmt.Errorf("agy version mismatch: expected %s, running %s (make install-agy installs the pin; bump tools/agy/install.sh and ExpectedVersion together after re-verifying the wire)", ExpectedVersion, v)
	}
	return nil
}

// Version returns "agy@<version>", or "" when unprobeable (UNMEASURED).
func (e *AgyExecutor) Version(ctx context.Context) string { return e.probe.Identity(ctx) }

// Register registers the agy executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("agy", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() { Register() }

// signinHint is the actionable form of an auth failure.
func signinHint(route string) string {
	if route == routeAPIKey {
		return "agy rejected the API key (check GEMINI_AGENTS_API_KEY)"
	}
	return "agy is not signed in for the subscription route: run `make agy-signin` once, attended, with the dedicated Google AI account"
}
