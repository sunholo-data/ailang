package eval_harness

import (
	"context"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"

	"github.com/sunholo-data/ailang/internal/executor"
)

// CanarySkip records a model that failed its pre-flight canary, so the suite
// can report WHY it was dropped instead of silently shrinking the run matrix.
type CanarySkip struct {
	Model  string
	Reason string
}

// canaryRunner is the seam under test: it resolves a model to an executor and
// runs its canary. Production wiring uses runModelCanary.
type canaryRunner func(ctx context.Context, model string) error

// FilterCanaryHealthyModels drops models whose subject cannot complete a step.
//
// # WHY THIS RUNS HERE, PER MODEL
//
// The obvious home for a canary looks like the existing pre-flight HealthCheck
// in agent_runner_multi.go — but that call site sits inside the per-(spec,
// language) run function, so it executes ONCE PER BENCHMARK. A ~90s canary
// there would cost 90s x benchmarks x langs x trials, i.e. hours per suite, and
// would be switched off within a week. This filter runs once per model, next to
// the existing agent-support filter, so the cost is bounded and the skip is
// structural: a dropped model never enters the run matrix, so it cannot bank a
// single row. See m-eval-measurement-contract sprint plan §0 (C1).
//
// Returns the surviving models and one CanarySkip per dropped model.
func FilterCanaryHealthyModels(ctx context.Context, models []string, run canaryRunner) ([]string, []CanarySkip) {
	var healthy []string
	var skipped []CanarySkip

	for _, model := range models {
		if err := run(ctx, model); err != nil {
			skipped = append(skipped, CanarySkip{Model: model, Reason: err.Error()})
			continue
		}
		healthy = append(healthy, model)
	}
	return healthy, skipped
}

// runModelCanary resolves a model to its executor and runs the canary.
//
// A model whose executor cannot even be constructed is NOT skipped here: that
// is a configuration error which the existing per-run path already reports with
// better context. The canary's job is narrowly to catch a subject that is
// present but broken.
func runModelCanary(ctx context.Context, model string) error {
	if modelreg.GlobalModelsConfig == nil {
		return nil
	}
	executorName, agentModelName, err := modelreg.GlobalModelsConfig.GetExecutorForModel(model)
	if err != nil {
		return nil // not a canary failure; let the normal path report it
	}
	exec, err := executor.GlobalFactory().GetExecutor(executorName)
	if err != nil {
		return nil // ditto — construction failure is a config problem, not a dead subject
	}
	defer func() { _ = exec.Close() }()

	// Describe the SUBJECT precisely. The factory hands back an executor built
	// from global defaults (profile "dogfood", a cloud model), so canarying it
	// as-is probes a different configuration than the benchmarks will run — the
	// first real run failed exactly that way. The profile travels the same route
	// the real path uses: task metadata.
	subject := executor.CanarySubject{Model: agentModelName}
	if mc, err := modelreg.GlobalModelsConfig.GetModel(model); err == nil && mc.MotokoProfile != "" {
		subject.Options = map[string]string{"motoko_profile": mc.MotokoProfile}
	}

	if err := executor.RunCanary(ctx, exec, subject); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

// RunModelCanary is the production canaryRunner.
func RunModelCanary(ctx context.Context, model string) error { return runModelCanary(ctx, model) }
