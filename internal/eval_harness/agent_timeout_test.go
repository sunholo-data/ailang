package eval_harness

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// TestEffectiveAgentTimeoutSecs pins the resolution order: spec timeout (else
// the flag default), raised to the model's hard_timeout_secs, raised to an
// EXPLICIT --agent-timeout floor. Before 2026-11-03 an explicit --agent-timeout
// was silently replaced by any spec timeout.
func TestEffectiveAgentTimeoutSecs(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"local-slow": {Budgets: modelreg.Budgets{HardTimeoutSecs: 3600}},
		"cloud-fast": {Budgets: modelreg.Budgets{HardTimeoutSecs: 300}},
	}}
	cases := []struct {
		name               string
		specT, flagT, minT int
		key                string
		want               int
	}{
		{"spec wins over flag default", 600, 60, 0, "unregistered", 600},
		{"flag default when spec has none", 0, 60, 0, "unregistered", 60},
		{"local hard timeout raises spec", 600, 60, 0, "local-slow", 3600},
		{"spec above hard timeout stays", 900, 60, 0, "cloud-fast", 900},
		{"explicit floor raises local wall clock", 600, 10800, 10800, "local-slow", 10800},
		{"floor below effective is a no-op", 600, 120, 120, "local-slow", 3600},
		{"floor applies to unregistered models", 600, 2400, 2400, "unregistered", 2400},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &AgentBenchmarkConfig{TimeoutSeconds: c.flagT, MinTimeoutSeconds: c.minT}
			if got := EffectiveAgentTimeoutSecs(&BenchmarkSpec{Timeout: c.specT}, cfg, c.key); got != c.want {
				t.Fatalf("got %d, want %d", got, c.want)
			}
		})
	}
}

// TestBuildEvalExecutorTaskUsesEffectiveTimeout: the executor task and the
// banked agent_timeout_secs must read the same resolution.
func TestBuildEvalExecutorTaskUsesEffectiveTimeout(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"local-slow": {Budgets: modelreg.Budgets{HardTimeoutSecs: 3600}},
	}}
	spec := &BenchmarkSpec{ID: "cp", Timeout: 600}
	for _, minT := range []int{0, 10800} {
		cfg := &MultiExecutorConfig{AgentBenchmarkConfig: AgentBenchmarkConfig{TimeoutSeconds: 60, MinTimeoutSeconds: minT}, ConfigKey: "local-slow"}
		x, err := BuildEvalExecutorTask(spec, cfg, "ollama/x", t.TempDir(), "", "")
		if err != nil {
			t.Fatal(err)
		}
		want := time.Duration(EffectiveAgentTimeoutSecs(spec, &cfg.AgentBenchmarkConfig, "local-slow")) * time.Second
		if x.Timeout != want || (minT == 0 && want != time.Hour) || (minT > 0 && want != 3*time.Hour) {
			t.Fatalf("min=%d: task timeout %s, effective %s", minT, x.Timeout, want)
		}
	}
}
