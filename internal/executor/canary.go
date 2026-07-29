package executor

import (
	"context"
	"fmt"
)

// CanaryChecker is an OPTIONAL executor capability: proof that the subject can
// actually complete a step, not merely that its binary is present.
//
// # WHY THIS IS SEPARATE FROM HealthCheck
//
// HealthCheck answers "is the CLI installed and configured?". motoko's
// implementation of it verifies the binary, OPENROUTER_API_KEY, and
// `motoko --version`. Every one of those passed, seventy-two times, between
// 2026-07-22 and 2026-07-28 while motoko was completely dead: AILANG's
// effect-row soundness fix (1282767ca) broke motoko's module load, and
// `--version` is handled at the TypeScript argv level and exits BEFORE any
// AILANG module is loaded. The eval harness happily banked 72 phantom
// "benchmark failures" — indistinguishable, downstream, from a model that
// simply could not solve the problems.
//
// A canary closes that gap by running one trivial end-to-end task and
// asserting the subject got as far as dispatching a tool call.
//
// # WHY AN OPTIONAL INTERFACE RATHER THAN AN Executor METHOD
//
// Six executors implement Executor (claude, codex, motoko, pi, opencode,
// managed_agents). Putting CanaryCheck on Executor would force all six to
// implement it now. Go's optional-interface pattern lets executors opt in one
// at a time: RunCanary treats a non-implementer as passing, so adopting the
// gate is strictly additive. See m-eval-measurement-contract M1.
type CanaryChecker interface {
	// CanaryCheck runs one trivial end-to-end task and returns nil only if the
	// subject demonstrably works. Implementations MUST bound their own runtime
	// and MUST respect ctx cancellation.
	CanaryCheck(ctx context.Context, subject CanarySubject) error
}

// CanarySubject identifies WHICH configuration to canary.
//
// This must describe the exact subject the benchmarks will run, not an
// executor default. The first real canary run proved why: it built the
// executor from factory defaults (profile "dogfood", a cloud model) and so
// probed a completely different configuration than the local ollama_fmt one
// under test — reporting a failure that told you nothing about the subject.
// A gate that checks the wrong thing is worse than no gate, because you
// believe it.
//
// Options carries executor-specific settings, mirroring how the real run
// passes them (motoko reads "motoko_profile"). Keeping it a string map rather
// than executor-typed fields is deliberate: this interface is meant to serve
// every executor and every experiment, so it must not grow a field per
// executor.
type CanarySubject struct {
	// Model is the model identifier to run (e.g. "ollama/qwen3.6:35b-a3b-mxfp8").
	Model string
	// Options are executor-specific settings, e.g. {"motoko_profile": "ollama_fmt"}.
	Options map[string]string
}

// CanaryError reports that an executor's canary failed. It is a distinct type
// so callers can attribute the resulting skip as "canary_failed" rather than
// folding it into a generic harness error — the distinction is the entire
// point of the measurement contract.
type CanaryError struct {
	// Executor is the executor name whose canary failed (e.g. "motoko").
	Executor string
	// Err is the underlying cause, preserved for errors.Is/As.
	Err error
}

func (e *CanaryError) Error() string {
	return fmt.Sprintf("executor %s canary failed (subject cannot complete a step): %v", e.Executor, e.Err)
}

// Unwrap exposes the underlying cause so errors.Is reaches it.
func (e *CanaryError) Unwrap() error { return e.Err }

// RunCanary runs exec's canary if it implements CanaryChecker.
//
// Executors that do NOT implement CanaryChecker are treated as passing. That
// is deliberate: the gate must be adoptable per-executor without a flag day.
// A nil return therefore means "not known to be broken", not "verified alive"
// — only an executor that implements CanaryChecker gives the stronger promise.
func RunCanary(ctx context.Context, exec Executor, subject CanarySubject) error {
	checker, ok := exec.(CanaryChecker)
	if !ok {
		return nil
	}
	if err := checker.CanaryCheck(ctx, subject); err != nil {
		return &CanaryError{Executor: exec.Name(), Err: err}
	}
	return nil
}
