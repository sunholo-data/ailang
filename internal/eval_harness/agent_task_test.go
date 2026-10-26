package eval_harness

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildEvalExecutorTaskKeepsOrdinaryControls(t *testing.T) {
	workspace := t.TempDir()
	spec := &BenchmarkSpec{ID: "owned-source", Timeout: 17, AgentEnv: map[string]string{"TASK_FIXTURE": "${WORKSPACE}/input"}}
	cfg := &MultiExecutorConfig{AgentBenchmarkConfig: AgentBenchmarkConfig{TimeoutSeconds: 10, MaxTokensPerBench: 345}, ModelName: "unregistered-fixture", ChainID: "chain", StageID: "stage"}
	x, err := BuildEvalExecutorTask(spec, cfg, "fixture-model", workspace, "system", "directive")
	if err != nil {
		t.Fatal(err)
	}
	if x.Timeout != 17*time.Second || x.Workspace != workspace || x.Model != "fixture-model" || x.MaxTokensPerBench != 345 || !x.EvalShellGuard {
		t.Fatalf("lost task controls: %+v", x)
	}
	if filepath.Clean(x.ExtraEnv["TASK_FIXTURE"]) != filepath.Join(workspace, "input") || x.Metadata["chain_id"] != "chain" || x.Metadata["stage_id"] != "stage" {
		t.Fatalf("lost env/provenance: %+v", x)
	}
	if x.SystemPrompt != "system" || x.Directive != "directive" || x.ResumeSessionID != "" {
		t.Fatalf("changed template/freshness: %+v", x)
	}
}

func TestBuildEvalExecutorTaskRejectsUnknownProfile(t *testing.T) {
	cfg := &MultiExecutorConfig{AgentBenchmarkConfig: AgentBenchmarkConfig{ToolPolicy: "nonexistent"}}
	if _, err := BuildEvalExecutorTask(&BenchmarkSpec{ID: "test"}, cfg, "fixture", t.TempDir(), "", ""); err == nil {
		t.Fatal("unknown profile accepted")
	}
}
