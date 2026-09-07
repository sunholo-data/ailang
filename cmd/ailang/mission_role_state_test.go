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
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

type stateRoleExecutor struct {
	calls int
	run   func()
	wait  func(context.Context)
}

func (e *stateRoleExecutor) Name() string { return "pi" }
func (e *stateRoleExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{executor.CapLocalWorkspace}
}
func (e *stateRoleExecutor) HealthCheck(context.Context) error { return nil }
func (e *stateRoleExecutor) CostModel() *executor.CostModel    { return nil }
func (e *stateRoleExecutor) Close() error                      { return nil }
func (e *stateRoleExecutor) Execute(ctx context.Context, t *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, t, nil)
}
func (e *stateRoleExecutor) ExecuteStreaming(ctx context.Context, _ *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	e.calls++
	if e.wait != nil {
		e.wait(ctx)
	}
	if e.run != nil {
		e.run()
	}
	return &executor.Result{Success: true, FinishReason: executor.FinishStop, Output: "artifact"}, nil
}
func (e *stateRoleExecutor) GetExecutor(string) (executor.Executor, error) { return e, nil }
func stateRoleFixture(t *testing.T) (dispatch.Request, *modelreg.ModelsConfig, string, string) {
	t.Helper()
	dir := t.TempDir()
	cli, wire := "pi", "openrouter/deepseek/model"
	cfg := &modelreg.ModelsConfig{Models: map[string]modelreg.ModelConfig{"writer": {Provider: "openrouter", APIName: "deepseek/model", AgentCLI: &cli, AgentModelName: &wire, Pricing: modelreg.Pricing{InputPer1K: .001, OutputPer1K: .002}}}}
	r := dispatch.Request{Version: 1, MissionID: "m", WorkItemID: "w", StageID: "s", AttemptID: "a", Role: "executor", Workspace: dir, InputRevision: "source", Instructions: "implement", Models: []string{"writer"}, TimeoutSeconds: 20, MaxTokens: 1000, MaxCostUSD: 1}
	return r, cfg, filepath.Join(dir, "coordinator.db"), filepath.Join(dir, "receipt.jsonl")
}
func TestMissionDurableRoleRejectsSecondReceipt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("durable receipt directory sync is unsupported on Windows; execution fails closed")
	}
	r, cfg, db, receipt := stateRoleFixture(t)
	e := &stateRoleExecutor{}
	report, err := runDurableMissionRole(context.Background(), r, receipt, db, cfg, e)
	if err != nil || report.Status != "execution_completed" {
		t.Fatalf("execution: %+v %v", report, err)
	}
	if _, err := runDurableMissionRole(context.Background(), r, receipt+".other", db, cfg, e); err == nil {
		t.Fatal("duplicate executed with another receipt path")
	}
	if e.calls != 1 {
		t.Fatalf("calls=%d", e.calls)
	}
	if _, err := os.Stat(receipt + ".other"); !os.IsNotExist(err) {
		t.Fatal("duplicate created receipt")
	}
	var out bytes.Buffer
	if err := runMissionAttempt(context.Background(), []string{"status", "--state-db", db, "--mission", "m", "--work-item", "w", "--stage", "s"}, &out); err != nil {
		t.Fatal(err)
	}
	var saved coordinator.MissionAttempt
	if err := json.Unmarshal(out.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.State != "execution_completed" || strings.Contains(out.String(), "owner_token") {
		t.Fatal("incorrect status or exposed fencing credential")
	}
}
func TestMissionDurableCancellationFencesCompletion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("durable receipt directory sync is unsupported on Windows; execution fails closed")
	}
	r, cfg, db, receipt := stateRoleFixture(t)
	e := &stateRoleExecutor{}
	e.run = func() {
		store, err := coordinator.NewSQLiteStore(db)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		key := coordinator.MissionAttemptKey{MissionID: "m", WorkItemID: "w", StageID: "s"}
		a, err := store.GetMissionAttempt(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}
		if a.State != "running" {
			t.Fatal("executor invoked before running state was persisted")
		}
		var out bytes.Buffer
		if err := runMissionAttempt(context.Background(), []string{"cancel", "--state-db", db, "--mission", "m", "--work-item", "w", "--stage", "s", "--version", fmt.Sprint(a.Version)}, &out); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runDurableMissionRole(context.Background(), r, receipt, db, cfg, e); err == nil {
		t.Fatal("cancelled attempt completion succeeded")
	}
}
func TestMissionDurableDryRunWritesNothing(t *testing.T) {
	r, cfg, db, receipt := stateRoleFixture(t)
	path := filepath.Join(r.Workspace, "request.json")
	body, _ := json.Marshal(r)
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := runMissionRole(context.Background(), []string{"--request", path, "--receipt", receipt, "--state-db", db, "--dry-run"}, &bytes.Buffer{}, cfg, nil); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{db, receipt} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("dry-run wrote %s", p)
		}
	}
	if err := runMissionAttempt(context.Background(), []string{"status", "--state-db", db, "--mission", "m", "--work-item", "w", "--stage", "s"}, &bytes.Buffer{}); err == nil {
		t.Fatal("status created absent database")
	}
}

func TestMissionHeartbeatRenewsThenCancelsWorker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("durable receipt directory sync is unsupported on Windows; execution fails closed")
	}
	r, cfg, db, receipt := stateRoleFixture(t)
	e := &stateRoleExecutor{}
	var observedCancellation bool
	e.wait = func(ctx context.Context) {
		store, err := coordinator.NewSQLiteStore(db)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		key := coordinator.MissionAttemptKey{MissionID: "m", WorkItemID: "w", StageID: "s"}
		initial, err := store.GetMissionAttempt(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.NewTimer(3 * time.Second)
		defer deadline.Stop()
		poll := time.NewTicker(20 * time.Millisecond)
		defer poll.Stop()
		for {
			select {
			case <-deadline.C:
				t.Fatal("heartbeat never renewed lease")
			case <-poll.C:
				current, err := store.GetMissionAttempt(ctx, key)
				if err != nil {
					t.Fatal(err)
				}
				if current.LeaseUntil <= initial.LeaseUntil {
					continue
				}
				if err := store.CancelMissionAttempt(ctx, key, current.Version); err != nil {
					t.Fatal(err)
				}
				select {
				case <-ctx.Done():
					observedCancellation = true
					return
				case <-time.After(time.Second):
					t.Fatal("worker was not cancelled after fence loss")
				}
			}
		}
	}
	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	if _, err := runMissionRoleWithHeartbeat(ctx, r, receipt, db, cfg, e, 20*time.Millisecond); err == nil {
		t.Fatal("lease-loss run succeeded")
	}
	if !observedCancellation {
		t.Fatal("no worker cancellation")
	}
}
