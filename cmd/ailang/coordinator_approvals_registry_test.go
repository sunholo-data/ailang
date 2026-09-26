package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func TestCheckRegistryCanDispatchSurfacesLoadError(t *testing.T) {
	bundle := approvalRegistryTestBundle(t, "sprint-planner")
	loadErr := errors.New("config is malformed")

	err := checkRegistryCanDispatch(context.Background(), bundle, nil, loadErr, "task-1", "approve")
	if err == nil || !strings.Contains(err.Error(), loadErr.Error()) {
		t.Fatalf("checkRegistryCanDispatch() error = %v, want load error", err)
	}
}

func TestCheckRegistryCanDispatchRejectsEmptyRegistry(t *testing.T) {
	bundle := approvalRegistryTestBundle(t, "sprint-planner")

	err := checkRegistryCanDispatch(
		context.Background(), bundle, coordinator.NewAgentRegistry(), nil, "task-1", "approve",
	)
	if err == nil || !strings.Contains(err.Error(), "has no entry") {
		t.Fatalf("checkRegistryCanDispatch() error = %v, want missing-agent error", err)
	}
}

func approvalRegistryTestBundle(t *testing.T, agentID string) *coordinatorStoreBundle {
	t.Helper()
	store, err := coordinator.NewSQLiteStore(filepath.Join(t.TempDir(), "coordinator.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.CreateTask(context.Background(), &coordinator.TaskRecord{
		ID: "task-1", AgentID: agentID,
	}); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	return &coordinatorStoreBundle{Store: store, Mode: "gcp (test)"}
}

// A rejection fires no handoffs, so an unresolvable agent cannot cost anything —
// and refusing one strands the task instead of protecting it.
//
// Measured 2026-09-17 on task-c063b6d2: its agent `daneel-design-ailang` was
// deleted in the 12 Sept revert, the approval had been pending seven days for
// work already merged by PR #1138, and the guard refused the rejection with
// advice ("point $AILANG_CONFIG at the plane's config") that could not help,
// because the agent exists in no registry anywhere.
func TestCheckRegistryCanDispatchAllowsRejectionOfAnOrphanedAgent(t *testing.T) {
	bundle := approvalRegistryTestBundle(t, "deleted-agent")

	if err := checkRegistryCanDispatch(
		context.Background(), bundle, coordinator.NewAgentRegistry(), nil, "task-1", "reject",
	); err != nil {
		t.Fatalf("rejection must not be blocked by an unresolvable agent: %v", err)
	}
	// The same task, approved, must STILL be refused: approving is what loses
	// the handoff topology.
	if err := checkRegistryCanDispatch(
		context.Background(), bundle, coordinator.NewAgentRegistry(), nil, "task-1", "approve",
	); err == nil {
		t.Fatal("approval of an unresolvable agent must still be refused")
	}
}
