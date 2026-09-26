package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

func TestMissionIterationDryRunQuotaAndFrozenResume(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix runtime support")
	}
	deps, db := iterationTestDeps(t)
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := iteration.Git(context.Background(), repo, args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	git("init")
	git("config", "user.email", "fixture@example.invalid")
	git("config", "user.name", "Fixture")
	git("config", "commit.gpgsign", "false")
	git("remote", "add", "origin", "https://github.com/example/project.git")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", "README.md")
	git("commit", "-m", "fixture")
	base := strings.TrimSpace(git("rev-parse", "HEAD"))
	limits := iteration.Limits{TimeoutSeconds: 30, MaxTokens: 1000, MaxCostUSD: 0.1}
	spec := iteration.Spec{Version: 1, MissionID: "docs", WorkItemID: "item-1", Repository: "github.com/example/project", BaseRevision: base, Brief: "Improve docs", AllowedPaths: []string{"docs/"}, Workflow: "full-v1", Limits: iteration.Limits{TimeoutSeconds: 120, MaxTokens: 4000, MaxCostUSD: 0.4}, AcceptanceCriteria: []iteration.Criterion{{ID: "clear", Text: "Clear docs"}}, Verification: []iteration.Verification{{ID: "check", Argv: []string{"git", "diff", "--check"}, Cwd: ".", TimeoutSeconds: 10}}}
	for _, role := range []string{"designer", "planner", "executor", "evaluator"} {
		spec.Stages = append(spec.Stages, iteration.Stage{ID: role, Role: role, Instructions: "Do approved work", RequiredArtifacts: []string{"docs/result.md"}, Limits: limits})
	}
	body, _ := json.Marshal(spec)
	file := filepath.Join(t.TempDir(), "work-item.json")
	if err := os.WriteFile(file, body, 0600); err != nil {
		t.Fatal(err)
	}
	cli, wire := "pi", "openrouter/minimax/model"
	deps.Models = &modelreg.ModelsConfig{Models: map[string]modelreg.ModelConfig{"fixture-model": {Provider: "openrouter", APIName: "minimax/model", AgentCLI: &cli, AgentModelName: &wire, Pricing: modelreg.Pricing{InputPer1K: .001, OutputPer1K: .002}}}, Roles: map[string][]string{}}
	for _, stage := range spec.Stages {
		deps.Models.Roles[stage.Role] = []string{"fixture-model"}
	}
	deps.Registry = func() (*mission.Registry, error) {
		return &mission.Registry{Missions: []*mission.Mission{{Name: "docs", Workdir: repo}}}, nil
	}
	factory := &missingRoleFactory{}
	deps.Factory = factory
	admissions := 0
	deps.Admit = func(context.Context, dispatch.Candidate) (dispatch.Admission, error) {
		admissions++
		return dispatch.Admission{Allowed: false, Policy: "fixture", Reason: "quota exhausted", ObservedAt: time.Now()}, nil
	}
	var out bytes.Buffer
	if err := runMissionIteration(context.Background(), "iterate", []string{"--work-item", file, "--dry-run"}, &out, deps); err != nil {
		t.Fatal(err)
	}
	if admissions != 0 || factory.calls != 0 {
		t.Fatal("dry run called provider admission or construction")
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatal("dry run created DB")
	}
	binding, err := iteration.LoadBinding(deps.BindingPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(binding.WorkspaceRoot); !os.IsNotExist(err) {
		t.Fatal("dry run created workspace")
	}
	err = runMissionIteration(context.Background(), "iterate", []string{"--work-item", file}, &bytes.Buffer{}, deps)
	if missionErrorExitCode(err) != 3 {
		t.Fatalf("quota wait: %v", err)
	}
	if admissions == 0 || factory.calls != 0 {
		t.Fatal("quota refusal leaked executor construction")
	}
	// Poison current policy. Resume must use the admitted model/repository snapshot.
	deps.Models = &modelreg.ModelsConfig{}
	deps.Registry = func() (*mission.Registry, error) { return nil, fmt.Errorf("live registry must not be loaded") }
	err = runMissionIteration(context.Background(), "resume", []string{"docs", "--work-item", "item-1"}, &bytes.Buffer{}, deps)
	if missionErrorExitCode(err) != 3 {
		t.Fatalf("frozen resume: %v", err)
	}
	if admissions < 2 || factory.calls != 0 {
		t.Fatal("resume did not retain protected route")
	}
	store, err := coordinator.OpenMissionReadOnlyStore(db)
	if err != nil {
		t.Fatal(err)
	}
	item, err := store.GetMissionWorkItem(context.Background(), coordinator.MissionWorkItemKey{MissionID: "docs", WorkItemID: "item-1"})
	if closeErr := store.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	err = runMissionIteration(context.Background(), "cancel", []string{"docs", "--work-item", "item-1", "--version", fmt.Sprint(item.Version)}, &bytes.Buffer{}, deps)
	if missionErrorExitCode(err) != 130 {
		t.Fatalf("cancel quota wait: %v", err)
	}
	before := admissions
	err = runMissionIteration(context.Background(), "resume", []string{"docs", "--work-item", "item-1"}, &bytes.Buffer{}, deps)
	if missionErrorExitCode(err) != 130 || admissions != before || factory.calls != 0 {
		t.Fatalf("terminal resume dispatched: %v", err)
	}
}
