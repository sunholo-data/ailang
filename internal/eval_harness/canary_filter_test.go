package eval_harness

import (
	"context"
	"errors"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"strings"
	"testing"
)

func TestFilterCanaryHealthyModels_AllHealthy(t *testing.T) {
	models := []string{"motoko-a", "motoko-b"}
	healthy, skipped := FilterCanaryHealthyModels(context.Background(), models,
		func(context.Context, string) error { return nil })

	if len(healthy) != 2 {
		t.Errorf("healthy = %v, want both models", healthy)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
}

// TestFilterCanaryHealthyModels_DropsBrokenSubject is the core of M1: the
// July 2026 outage condition must remove the model from the run matrix, and
// the reason must survive so the operator learns WHY.
func TestFilterCanaryHealthyModels_DropsBrokenSubject(t *testing.T) {
	broken := "motoko-local-qwen3-6-35b-a3b-mxfp8"
	healthy, skipped := FilterCanaryHealthyModels(context.Background(),
		[]string{"opencode-x", broken, "pi-y"},
		func(_ context.Context, m string) error {
			if m == broken {
				return errors.New("canary ran no steps (subject started but executed nothing)")
			}
			return nil
		})

	if len(healthy) != 2 || healthy[0] != "opencode-x" || healthy[1] != "pi-y" {
		t.Errorf("healthy = %v, want the two working models", healthy)
	}
	if len(skipped) != 1 {
		t.Fatalf("skipped = %v, want exactly the broken model", skipped)
	}
	if skipped[0].Model != broken {
		t.Errorf("skipped model = %q, want %q", skipped[0].Model, broken)
	}
	if !strings.Contains(skipped[0].Reason, "no steps") {
		t.Errorf("skip reason must explain the failure, got %q", skipped[0].Reason)
	}
}

// TestFilterCanaryHealthyModels_OneCanaryPerModel guards the whole point of
// C1: if this ever runs per benchmark instead of per model the gate becomes
// unaffordable and gets disabled.
func TestFilterCanaryHealthyModels_OneCanaryPerModel(t *testing.T) {
	calls := map[string]int{}
	models := []string{"a", "b", "c"}
	FilterCanaryHealthyModels(context.Background(), models, func(_ context.Context, m string) error {
		calls[m]++
		return nil
	})

	for _, m := range models {
		if calls[m] != 1 {
			t.Errorf("model %s canaried %d times, want exactly 1", m, calls[m])
		}
	}
}

// TestFilterCanaryHealthyModels_AllBrokenYieldsEmpty ensures a totally dead
// rotation produces an empty matrix rather than silently running anyway.
func TestFilterCanaryHealthyModels_AllBrokenYieldsEmpty(t *testing.T) {
	healthy, skipped := FilterCanaryHealthyModels(context.Background(),
		[]string{"a", "b"},
		func(context.Context, string) error { return errors.New("dead") })

	if len(healthy) != 0 {
		t.Errorf("healthy = %v, want empty", healthy)
	}
	if len(skipped) != 2 {
		t.Errorf("skipped = %v, want both", skipped)
	}
}

// --- Integration: the REAL runModelCanary path (factory + RunCanary) --------

// brokenSubject stands in for a motoko whose binary is present but whose core
// is dead — the exact July 2026 condition. Using a registered fake keeps this
// test hermetic: reproducing the real outage would need mk-ast + bun + ollama,
// which CI does not have.
type brokenSubject struct{ executor.Executor }

func (b *brokenSubject) Name() string { return "broken-subject" }
func (b *brokenSubject) Close() error { return nil }
func (b *brokenSubject) CanaryCheck(context.Context, executor.CanarySubject) error {
	return errors.New("canary ran no steps (subject started but executed nothing — typically a module-load failure)")
}

type healthySubject struct{ executor.Executor }

func (h *healthySubject) Name() string                                              { return "healthy-subject" }
func (h *healthySubject) Close() error                                              { return nil }
func (h *healthySubject) CanaryCheck(context.Context, executor.CanarySubject) error { return nil }

// TestRunModelCanary_EndToEnd drives the production resolver: models.yml lookup
// -> executor factory -> RunCanary -> CanaryError. This is the wiring that
// would have caught the six-day outage; a unit test of the filter alone would
// still pass if the wiring were broken.
func TestRunModelCanary_EndToEnd(t *testing.T) {
	executor.GlobalFactory().Register("broken-subject", func(*executor.Config) (executor.Executor, error) {
		return &brokenSubject{}, nil
	})
	executor.GlobalFactory().Register("healthy-subject", func(*executor.Config) (executor.Executor, error) {
		return &healthySubject{}, nil
	})

	brokenCLI, healthyCLI := "broken-subject", "healthy-subject"
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"dead-model":  {AgentCLI: &brokenCLI},
		"alive-model": {AgentCLI: &healthyCLI},
	}}

	healthy, skipped := FilterCanaryHealthyModels(context.Background(),
		[]string{"alive-model", "dead-model"}, RunModelCanary)

	if len(healthy) != 1 || healthy[0] != "alive-model" {
		t.Errorf("healthy = %v, want [alive-model]", healthy)
	}
	if len(skipped) != 1 || skipped[0].Model != "dead-model" {
		t.Fatalf("skipped = %v, want dead-model", skipped)
	}
	// The operator must be able to see it was a canary failure specifically.
	if !strings.Contains(skipped[0].Reason, "canary failed") {
		t.Errorf("reason should identify this as a canary failure, got %q", skipped[0].Reason)
	}
	if !strings.Contains(skipped[0].Reason, "no steps") {
		t.Errorf("reason should carry the underlying detail, got %q", skipped[0].Reason)
	}
}
