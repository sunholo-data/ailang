package eval_harness

import (
	"context"
	"math"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// costFakeExecutor returns a canned Result without running anything. Embedding
// the interface leaves the methods RunAgentBenchmarkWithExecutor never calls
// unimplemented, as brokenSubject does in canary_filter_test.go.
type costFakeExecutor struct {
	executor.Executor
	name   string
	result executor.Result
}

func (f *costFakeExecutor) Name() string                        { return f.name }
func (f *costFakeExecutor) Capabilities() []executor.Capability { return nil }
func (f *costFakeExecutor) HealthCheck(context.Context) error   { return nil }
func (f *costFakeExecutor) Close() error                        { return nil }
func (f *costFakeExecutor) CostModel() *executor.CostModel      { return nil }
func (f *costFakeExecutor) Execute(context.Context, *executor.Task) (*executor.Result, error) {
	r := f.result
	return &r, nil
}
func (f *costFakeExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	return f.Execute(ctx, task)
}

// TestRunAgentBenchmarkWithExecutor_CostUnion pins the executor-cost union on
// the live agent path (#615): a self-reported $0 is recomputed from banked
// tokens at models.yml rates, a nonzero self-report is kept, and in every case
// CostProvenance is banked exactly as the executor reported it. Provenance is a
// property of the auth lane, not of who did the arithmetic, so recomputing the
// number must never relabel it.
//
// The fake reports Success=false with no Error: that passes the executor-crash
// gate and makes validateSolution return before compiling anything, so the
// test is hermetic (no CLI, network, or ailang run).
func TestRunAgentBenchmarkWithExecutor_CostUnion(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"priced-model": {Pricing: modelreg.Pricing{
			InputPer1K: 1, OutputPer1K: 2, CacheReadPer1K: 0.5, CacheWritePer1K: 4,
		}},
		"free-model": {Pricing: modelreg.Pricing{}},
	}}

	tokens := executor.Result{
		InputTokens:              1000,
		OutputTokens:             500,
		ReasonTokens:             500,
		CacheReadInputTokens:     2000,
		CacheCreationInputTokens: 1000,
	}
	// 1000*1 + (500+500)*2 + 2000*0.5 + 1000*4, per 1K tokens.
	const recomputed = 1.0 + 2.0 + 1.0 + 4.0

	tests := []struct {
		name           string
		model          string
		reportedCost   float64
		provenance     executor.CostProvenance
		wantCost       float64
		wantProvenance string
	}{
		{
			name:           "zero self-report is recomputed and provenance kept",
			model:          "priced-model",
			provenance:     executor.CostListPriceEquivalent,
			wantCost:       recomputed,
			wantProvenance: string(executor.CostListPriceEquivalent),
		},
		{
			name:           "zero self-report on a metered lane stays metered",
			model:          "priced-model",
			provenance:     executor.CostMetered,
			wantCost:       recomputed,
			wantProvenance: string(executor.CostMetered),
		},
		{
			name:           "nonzero self-report is not recomputed",
			model:          "priced-model",
			reportedCost:   0.42,
			provenance:     executor.CostMetered,
			wantCost:       0.42,
			wantProvenance: string(executor.CostMetered),
		},
		{
			name:           "free local model stays zero",
			model:          "free-model",
			provenance:     executor.CostFreeLocal,
			wantCost:       0,
			wantProvenance: string(executor.CostFreeLocal),
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := tokens
			res.NumTurns = 2
			res.CostUSD = tc.reportedCost
			res.CostProvenance = tc.provenance
			fake := &costFakeExecutor{name: "cost-union-fake-" + string(rune('a'+i)), result: res}
			executor.GlobalFactory().Register(fake.name, func(*executor.Config) (executor.Executor, error) {
				return fake, nil
			})

			spec := &BenchmarkSpec{ID: "cost_union", Description: "cost union fixture"}
			cfg := MultiExecutorConfig{
				ExecutorName: fake.name,
				ModelName:    tc.model,
			}
			cfg.WorkspaceDir = t.TempDir()

			out, err := RunAgentBenchmarkWithExecutor(spec, cfg, "ailang")
			if err != nil {
				t.Fatalf("RunAgentBenchmarkWithExecutor: %v", err)
			}
			if math.Abs(out.Cost-tc.wantCost) > 1e-9 {
				t.Errorf("Cost = %v, want %v", out.Cost, tc.wantCost)
			}
			if out.CostProvenance != tc.wantProvenance {
				t.Errorf("CostProvenance = %q, want %q (recomputing cost must not relabel it)",
					out.CostProvenance, tc.wantProvenance)
			}
		})
	}
}
