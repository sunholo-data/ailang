package main

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func TestCancelOneTask(t *testing.T) {
	ctx := context.Background()
	store, err := coordinator.NewSQLiteStore(t.TempDir() + "/c.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	mk := func(id string) {
		if err := store.CreateTask(ctx, &coordinator.TaskRecord{ID: id, AgentID: "a", Title: "t", Content: id, Status: coordinator.TaskStatusPending}); err != nil {
			t.Fatal(err)
		}
	}
	mk("task-p1")
	if err := cancelOneTask(ctx, store, "task-p1", true); err != nil {
		t.Fatalf("pending: %v", err)
	}
	if got, _ := store.GetTask(ctx, "task-p1"); got.Status != coordinator.TaskStatusCancelled {
		t.Fatalf("status = %s", got.Status)
	}

	mk("task-a1")
	if err := store.MarkTaskQueued(ctx, "task-a1"); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkTaskRunning(ctx, "task-a1", "pi", ""); err != nil {
		t.Fatal(err)
	}
	// Refused BEFORE the confirmation prompt (yes=false, no stdin): asking
	// "cancel?" for a task that cannot be cancelled would be a false choice.
	if err := cancelOneTask(ctx, store, "task-a1", false); err == nil || !strings.Contains(err.Error(), "running") {
		t.Fatalf("running: err = %v, want a refusal naming the status, before any prompt", err)
	}

	// Awaiting approval: point at reject, which also resolves the card.
	mk("task-r1")
	if err := store.MarkTaskPendingApproval(ctx, "task-r1", "", "", "", "", &coordinator.ExecuteResult{}); err != nil {
		t.Fatal(err)
	}
	if err := cancelOneTask(ctx, store, "task-r1", true); err == nil || !strings.Contains(err.Error(), "coordinator reject task-r1") {
		t.Fatalf("pending_approval: err = %v, want the reject hint", err)
	}

	if err := cancelOneTask(ctx, store, "task-missing", true); err == nil {
		t.Fatal("a missing task must be an error")
	}
}
