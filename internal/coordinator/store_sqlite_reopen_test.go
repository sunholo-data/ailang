package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ReopenTask had no test at all until it needed a second implementation.
//
// It was a method on *SQLiteStore that the interface never mentioned, so the
// cloud plane simply had no way to reopen a task — the eleven cancelled triage
// tasks in prod on 2026-09-15 could not be put back in the queue. These are the
// semantics the Firestore implementation mirrors, and the reason they are
// written down: two implementations of one concept that quietly disagree is the
// fault class this pipeline keeps producing.
func TestReopenTask(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	mk := func(id string, status TaskStatus) {
		now := time.Now()
		task := &TaskRecord{
			ID: id, AgentID: "triage", Title: "a report", Content: "triage this",
			Type: TaskTypeBugFix, Status: status, CreatedAt: now, CompletedAt: &now,
		}
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	// A cancelled task with a resolved approval: both move back to pending.
	mk("task-cancelled", TaskStatusCancelled)
	if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
		ID: ApprovalIDForTask("task-cancelled"), TaskID: "task-cancelled",
		Type: string(ApprovalTypeMerge), Status: "rejected", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("approval: %v", err)
	}
	if err := store.ReopenTask(ctx, "task-cancelled"); err != nil {
		t.Fatalf("reopen cancelled: %v", err)
	}
	got, err := store.GetTask(ctx, "task-cancelled")
	if err != nil || got.Status != TaskStatusPendingApproval {
		t.Fatalf("status = %v (err %v), want pending_approval", got.Status, err)
	}
	if got.CompletedAt != nil {
		// A task awaiting a decision that also claims to have completed is the
		// state the approval queue filters on.
		t.Errorf("completed_at must be cleared, got %v", got.CompletedAt)
	}
	apr, _ := store.GetApprovalRequestByTaskAnyStatus(ctx, "task-cancelled")
	if apr == nil || apr.Status != "pending" {
		t.Fatalf("approval = %v, want pending", apr)
	}

	// A rejected task with NO approval row gets one, or it reopens into a queue
	// that shows nothing.
	mk("task-rejected", TaskStatusRejected)
	if err := store.ReopenTask(ctx, "task-rejected"); err != nil {
		t.Fatalf("reopen rejected: %v", err)
	}
	if apr, _ := store.GetApprovalRequestByTaskAnyStatus(ctx, "task-rejected"); apr == nil || apr.Status != "pending" {
		t.Fatalf("a reopened task must have a pending approval, got %v", apr)
	}

	// Anything else is refused: reopening a completed task would invent a
	// decision point the work has already passed.
	mk("task-completed", TaskStatusCompleted)
	err = store.ReopenTask(ctx, "task-completed")
	if err == nil || !strings.Contains(err.Error(), "cannot reopen") {
		t.Errorf("completed must be refused, got %v", err)
	}
	if err := store.ReopenTask(ctx, "task-missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("a missing task must say so, got %v", err)
	}
}
