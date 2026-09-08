package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

func TestMissionRoleDryRunAndStrictFlags(t *testing.T) {
	dir := t.TempDir()
	testutil.SetHomeDir(t, dir)
	cli := "pi"
	wire := "openrouter/minimax/model"
	cfg := &modelreg.ModelsConfig{Models: map[string]modelreg.ModelConfig{"author": {Provider: "openai", APIName: "gpt"}, "judge": {Provider: "openrouter", APIName: "minimax/model", AgentCLI: &cli, AgentModelName: &wire, Pricing: modelreg.Pricing{InputPer1K: .001, OutputPer1K: .002}}}}
	r := dispatch.Request{Version: 1, MissionID: "m", WorkItemID: "w", StageID: "s", AttemptID: "a", Role: "evaluator", Workspace: dir, InputRevision: "sha", Instructions: "review", Models: []string{"judge"}, AuthorModels: []string{"author"}, TimeoutSeconds: 10, MaxTokens: 1000, MaxCostUSD: .1}
	b, _ := json.Marshal(r)
	request := filepath.Join(dir, "request.json")
	if err := os.WriteFile(request, b, 0600); err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(dir, "receipt.jsonl")
	var out bytes.Buffer
	if err := runMissionRole(context.Background(), []string{"--request", request, "--receipt", receipt, "--dry-run"}, &out, cfg, nil); err != nil {
		t.Fatal(err)
	}
	var p dispatch.Plan
	if err := json.Unmarshal(out.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Candidates) != 1 || p.Candidates[0].Executor != "pi" {
		t.Fatalf("incorrect plan: %+v", p)
	}
	if _, err := os.Stat(receipt); !os.IsNotExist(err) {
		t.Fatal("dry run wrote receipt")
	}
	for _, args := range [][]string{{"--typo"}, {"--request", request, "unexpected"}, {"--request", request}, {"--dry-run"}} {
		if err := runMissionRole(context.Background(), args, &bytes.Buffer{}, cfg, nil); err == nil {
			t.Errorf("accepted flags %v", args)
		}
	}
}

// A missing adapter must produce a durable blocked report and a nonzero result.
func TestMissionRoleBlockedAndExclusiveReceipt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("durable receipt directory sync is unsupported on Windows; execution fails closed")
	}
	dir := t.TempDir()
	testutil.SetHomeDir(t, dir)
	cli, wire := "pi", "openrouter/minimax/model"
	cfg := &modelreg.ModelsConfig{Models: map[string]modelreg.ModelConfig{"judge": {Provider: "openrouter", APIName: "minimax/model", AgentCLI: &cli, AgentModelName: &wire, Pricing: modelreg.Pricing{InputPer1K: .001, OutputPer1K: .002}}}}
	r := dispatch.Request{Version: 1, MissionID: "m", WorkItemID: "w", StageID: "s", AttemptID: "a", Role: "designer", Workspace: dir, InputRevision: "sha", Instructions: "design", Models: []string{"judge"}, TimeoutSeconds: 10, MaxTokens: 1000, MaxCostUSD: .1}
	b, _ := json.Marshal(r)
	request, receipt := filepath.Join(dir, "request.json"), filepath.Join(dir, "receipt.jsonl")
	if err := os.WriteFile(request, b, 0600); err != nil {
		t.Fatal(err)
	}
	f := &missingRoleFactory{}
	args := []string{"--request", request, "--receipt", receipt}
	var out bytes.Buffer
	if err := runMissionRole(context.Background(), args, &out, cfg, f); err == nil {
		t.Fatal("blocked execution succeeded")
	}
	var report dispatch.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != "blocked" || report.ArtifactVerified {
		t.Fatalf("invalid blocked report: %+v", report)
	}
	before, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := runMissionRole(context.Background(), args, &bytes.Buffer{}, cfg, f); err == nil {
		t.Fatal("existing receipt accepted")
	}
	after, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || f.calls != 1 {
		t.Fatal("duplicate touched receipt or factory")
	}
}

type missingRoleFactory struct{ calls int }

func (f *missingRoleFactory) GetExecutor(string) (executor.Executor, error) {
	f.calls++
	return nil, fmt.Errorf("fixture: executor unavailable")
}
