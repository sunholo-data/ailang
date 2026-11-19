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
	if x.Timeout != 17*time.Second || x.Workspace != workspace || x.Model != "fixture-model" || x.MaxTokensPerBench != 345 || !x.EvalShellGuard || !x.EvalRun {
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

// Tool profile "lean" turns on MinimalHarnessContext (pi: short fixed system prompt, no
// skills/extensions/context discovery); any other profile leaves pi's defaults.
func TestBuildEvalExecutorTaskLeanProfile(t *testing.T) {
	cfg := &MultiExecutorConfig{AgentBenchmarkConfig: AgentBenchmarkConfig{ToolPolicy: "lean"}}
	x, err := BuildEvalExecutorTask(&BenchmarkSpec{ID: "lean"}, cfg, "m", t.TempDir(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !x.MinimalHarnessContext || len(x.AllowedTools) != 4 {
		t.Fatalf("lean profile: MinimalHarnessContext=%v tools=%v", x.MinimalHarnessContext, x.AllowedTools)
	}
	cfg.ToolPolicy = "Read,Bash,Edit,Write" // same tools, NOT the lean lane
	x, err = BuildEvalExecutorTask(&BenchmarkSpec{ID: "list"}, cfg, "m", t.TempDir(), "", "")
	if err != nil || x.MinimalHarnessContext {
		t.Fatalf("a plain tool list must not drop the harness context: %v %v", x.MinimalHarnessContext, err)
	}
}
