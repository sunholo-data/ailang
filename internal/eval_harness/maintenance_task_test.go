package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

func TestMaintenanceTaskBuilderFreezesRegistryAndPrompt(t *testing.T) {
	old := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = old })
	cli, wire := "pi", "provider/model"
	modelreg.GlobalModelsConfig = &modelreg.ModelsConfig{Models: map[string]modelreg.ModelConfig{"fixture": {APIName: wire, AgentCLI: &cli, AgentModelName: &wire}}}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prompt := []byte("frozen teaching prompt")
	if err = os.WriteFile(filepath.Join(root, "prompt.txt"), prompt, 0600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(prompt)
	digest, err := MaintenanceModelControlsSHA256("fixture", 100)
	if err != nil {
		t.Fatal(err)
	}
	settings := MaintenanceExecutorSettings{Schema: "maintenance.executor-settings/1", ModelKey: "fixture", ModelControlsSHA256: digest, HarnessVersion: "pi@0.85.1", AllowedTools: []string{"Read", "Write", "Edit", "Bash"}, SystemPrompts: map[string]MaintenanceFileRef{"ailang": {Path: "prompt.txt", SHA256: hex.EncodeToString(h[:])}}}
	b, _ := json.Marshal(settings)
	s := sha256.Sum256(b)
	path := filepath.Join(root, "settings.json")
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	e := &MaintenanceExecution{Root: root, Plan: MaintenanceRunPlan{Model: wire, Harness: settings.HarnessVersion, SettingsDigest: hex.EncodeToString(s[:])}, Manifest: MaintenanceExecutionManifest{Settings: MaintenanceFileRef{Path: path, SHA256: hex.EncodeToString(s[:])}, Limits: MaintenanceLimits{MaxStageSeconds: 30, MaxTokens: 100}, Isolation: MaintenanceIsolation{Backend: "pi"}}}
	prepared := 0
	build, err := NewMaintenanceTaskBuilder(e, func(task *executor.Task) error { prepared++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	task, err := build(MaintenanceTrajectoryPlan{ID: "tr", Language: "ailang"}, MaintenanceStagePlan{ID: "s1"}, root, "current request")
	if err != nil {
		t.Fatal(err)
	}
	if prepared != 1 || task.Directive != "current request" || task.SystemPrompt != string(prompt) || !task.IsolateFromAmbientContext || !task.IsolateFromProjectExtensions || task.Iteration != 1 || task.ResumeSessionID != "" || task.MaxTokensPerBench != 100 {
		t.Fatalf("task drift: %+v", task)
	}
	e.Plan.Trajectories = []MaintenanceTrajectoryPlan{{Language: "ailang"}, {Language: "python"}}
	if _, err = NewMaintenanceTaskBuilder(e, func(*executor.Task) error { return nil }); err == nil {
		t.Fatal("missing counterpart teaching prompt discovered after partial dispatch")
	}
	e.Plan.Trajectories = nil
	changed := modelreg.GlobalModelsConfig.Models["fixture"]
	changed.MaxOutputTokens = 987
	modelreg.GlobalModelsConfig.Models["fixture"] = changed
	if _, err = build(MaintenanceTrajectoryPlan{Language: "ailang"}, MaintenanceStagePlan{}, root, "request"); err == nil {
		t.Fatal("registry drift allowed")
	}
	if _, err = NewMaintenanceTaskBuilder(e, nil); err == nil {
		t.Fatal("missing trusted containment preparation allowed")
	}
}
