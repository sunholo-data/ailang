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

	err := checkRegistryCanDispatch(context.Background(), bundle, nil, loadErr, "task-1")
	if err == nil || !strings.Contains(err.Error(), loadErr.Error()) {
		t.Fatalf("checkRegistryCanDispatch() error = %v, want load error", err)
	}
}

func TestCheckRegistryCanDispatchRejectsEmptyRegistry(t *testing.T) {
	bundle := approvalRegistryTestBundle(t, "sprint-planner")

	err := checkRegistryCanDispatch(
		context.Background(), bundle, coordinator.NewAgentRegistry(), nil, "task-1",
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
