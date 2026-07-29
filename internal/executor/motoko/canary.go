package motoko

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

const (
	// canaryTimeout bounds a single canary attempt.
	//
	// 90s was too tight and skipped a HEALTHY model: a cold local 35B has to be
	// pulled into memory before it emits a token, and the same ollama_fmt
	// configuration completed the identical task fine when run by hand. That is
	// risk R4 from the sprint plan landing in practice — a gate that skips good
	// subjects is worse than no gate, because you stop trusting it. 4 minutes
	// still fails far faster than the full night a dead subject used to burn.
	canaryTimeout = 4 * time.Minute

	// canaryMaxTimeout is the hard ceiling a canary task may declare.
	canaryMaxTimeout = 5 * time.Minute

	// canaryDirective is deliberately the most trivial task that still proves
	// the properties benchmarks depend on: the subject must READ a file (module
	// load + step 0 + tool dispatch all have to work to get this far).
	canaryDirective = "Read the file canary.txt and report its contents. Do not modify anything."

	// canaryTTFT is the prefill budget — how long the subject may take to emit
	// its FIRST event before being killed.
	//
	// THIS WAS THE BUG. Task.TTFTTimeout defaults to 30s, and the canary never
	// set it. A cold local 35B needs longer than that just to load and begin
	// emitting, so the executor killed motoko before any event was parsed and
	// reported "ran no steps" — while the abandoned process went on to finish
	// the task correctly (steps_executed=2, ReadFile dispatched, right answer).
	// The gate was condemning healthy subjects for being cold.
	canaryTTFT = 3 * time.Minute

	// canaryIdle bounds silence AFTER the first event.
	canaryIdle = 3 * time.Minute
)

// CanaryCheck implements executor.CanaryChecker.
//
// It runs one trivial read-only task and asserts the subject actually executed
// a step and dispatched a tool call. HealthCheck already covers "is the binary
// there"; this covers "does it work", which is the gap that let motoko run dead
// for six days in July 2026 while banking 72 phantom benchmark failures.
//
// On an ambiguous failure (context deadline — i.e. slow rather than broken) the
// canary retries ONCE before condemning the model. A gate that skips healthy
// models under load is worse than no gate: it would be turned off.
func (e *MotokoExecutor) CanaryCheck(ctx context.Context, subject executor.CanarySubject) error {
	// HealthCheck FIRST — not just for its own sake. It is what populates
	// e.motokoRepo from `motoko --version`, and findSessionJSONL needs that to
	// locate the session log: motoko's wrapper cd's into the repo, so the JSONL
	// lands there and NOT in our temp workspace. Skipping it made every canary
	// look in an empty temp dir, find nothing, and report a healthy subject as
	// broken.
	if err := e.HealthCheck(ctx); err != nil {
		return fmt.Errorf("canary pre-flight: %w", err)
	}

	err := e.runCanaryOnce(ctx, subject)
	if err == nil {
		return nil
	}
	// Retry only the ambiguous case (timeout/cancellation), never a clean
	// reproducible failure — a real breakage should fail fast and stay failed.
	if ctx.Err() != nil {
		return err // caller cancelled; do not burn another attempt
	}
	if isTimeout(err) {
		return e.runCanaryOnce(ctx, subject)
	}
	return err
}

// runCanaryOnce performs a single canary attempt in a throwaway workspace.
func (e *MotokoExecutor) runCanaryOnce(ctx context.Context, subject executor.CanarySubject) error {
	workspace, err := os.MkdirTemp("", "motoko-canary-*")
	if err != nil {
		return fmt.Errorf("canary: cannot create workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	// The single file the canary asks the subject to read.
	if err := os.WriteFile(filepath.Join(workspace, "canary.txt"), []byte("ok\n"), 0o644); err != nil {
		return fmt.Errorf("canary: cannot seed workspace: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, canaryTimeout)
	defer cancel()

	// Run the SUBJECT's configuration, not the executor default. The profile
	// travels on task metadata exactly as the real benchmark path sends it
	// (see agent_runner_multi), so the canary probes the same thing the
	// benchmarks will.
	model := subject.Model
	if model == "" {
		model = e.model
	}
	task := newCanaryTask(workspace, model)
	if len(subject.Options) > 0 {
		task.Metadata = make(map[string]string, len(subject.Options))
		for k, v := range subject.Options {
			task.Metadata[k] = v
		}
	}
	result, execErr := e.Execute(runCtx, task)

	// A deadline must surface AS a deadline. Execute returns a zero-turn result
	// rather than an error when it is cut short, so without this the ambiguous
	// "slow" case was reported as the unambiguous "subject executed nothing" —
	// which both misdiagnoses a healthy model and defeats the retry that exists
	// precisely for this case.
	if runCtx.Err() != nil && (result == nil || result.NumTurns == 0) {
		return fmt.Errorf("canary did not finish within %s: %w", canaryTimeout, runCtx.Err())
	}
	return evaluateCanaryResult(result, execErr)
}

// canarySeq makes every canary task ID unique within a process. A wall-clock
// timestamp alone is NOT sufficient: consecutive calls can land in the same
// nanosecond, which a test caught immediately.
var canarySeq atomic.Int64

// newCanaryTask builds the canary task. Extracted so a test can assert it stays
// trivial and bounded without executing anything.
func newCanaryTask(workspace, model string) *executor.Task {
	return &executor.Task{
		// Unique per attempt: motoko names its session log after the task ID, so
		// a constant "canary" made every run APPEND to one shared
		// session_canary.jsonl. Concurrent or repeated canaries would then parse
		// each other's events.
		ID:        "canary-" + strconv.FormatInt(canarySeq.Add(1), 36) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Directive: canaryDirective,
		Workspace: workspace,
		Model:     model,
		Timeout:   canaryTimeout,
		// Without these the 30s TTFT default kills a cold local model before it
		// emits anything. See canaryTTFT.
		TTFTTimeout: canaryTTFT,
		IdleTimeout: canaryIdle,
	}
}

// evaluateCanaryResult decides whether a canary run proves the subject works.
//
// The assertions escalate deliberately:
//   - execErr / nil result  → the subject did not come back at all
//   - NumTurns == 0         → came back, never ran a step (the July 2026 shape:
//     the process exits but dies during AILANG module load)
//   - no tool calls         → ran a step but cannot ACT; every benchmark
//     requires writing a file, so a subject that
//     cannot dispatch a tool cannot pass anything
func evaluateCanaryResult(result *executor.Result, execErr error) error {
	if execErr != nil {
		return fmt.Errorf("canary task failed: %w", execErr)
	}
	if result == nil {
		return fmt.Errorf("canary produced no result (subject did not return)")
	}
	// Surface the executor's OWN diagnosis before falling back to a generic
	// one. motoko reports failures IN the Result (Success=false + Error) with a
	// nil Go error, so a bare NumTurns==0 check swallowed messages as specific
	// as "no session JSONL found" and replaced them with "executed nothing" —
	// which sent the investigation at the model for hours when the actual fault
	// was the harness looking in the wrong directory. A gate that hides its own
	// cause is barely better than no gate.
	if !result.Success && result.Error != "" {
		return fmt.Errorf("canary failed: %s", result.Error)
	}
	if result.NumTurns == 0 {
		return fmt.Errorf("canary ran no steps (subject started but executed nothing — typically a module-load or startup failure; check $TMPDIR/motoko-stderr-*.log)")
	}
	if len(result.ToolCalls) == 0 {
		return fmt.Errorf("canary made no tool calls in %d step(s) (subject can emit text but cannot act)", result.NumTurns)
	}
	return nil
}

// isTimeout reports whether err represents a deadline/cancellation, i.e. the
// ambiguous "slow, maybe not broken" case that earns one retry.
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
