package motoko

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// newCanaryTask builds the canary task. Extracted so a test can assert it stays
// trivial and bounded without executing anything.
func newCanaryTask(workspace, model string) *executor.Task {
	return &executor.Task{
		ID:        "canary",
		Directive: canaryDirective,
		Workspace: workspace,
		Model:     model,
		Timeout:   canaryTimeout,
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
