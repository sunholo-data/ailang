package coordinator

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// M-TASK-STATUS-TRUTH S1 — the claim records WHEN it claimed.
//
// The stale detector's timeout reads "no completion received within X of being
// queued", but nothing recorded when a task was queued, so the clock fell back to
// CreatedAt — which is the MESSAGE's timestamp. These pin the writer half: the
// claim stamps queued_at, every read path returns it, and startup recovery ages
// from it too.

func newQueuedAtStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "coordinator.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// MU: drop queued_at from MarkTaskQueued's UPDATE and this fails.
func TestMarkTaskQueuedStampsQueuedAt(t *testing.T) {
	ctx := context.Background()
	store := newQueuedAtStore(t)
	messageTime := time.Now().Add(-154 * time.Hour)
	if err := store.CreateTask(ctx, &TaskRecord{
		ID: "task-old-message", Title: "recovered by the sweep", Status: TaskStatusPending,
		AgentID: "ailang-core-triage", CreatedAt: messageTime,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	before := time.Now().Add(-time.Second)
	if err := store.MarkTaskQueued(ctx, "task-old-message"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	got, err := store.GetTask(ctx, "task-old-message")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.QueuedAt == nil {
		t.Fatal("GetTask: queued_at not recorded by the claim — the stale clock has nothing to start from " +
			"and falls back to the message's age")
	}
	if got.QueuedAt.Before(before) {
		t.Errorf("queued_at = %v, want the claim time (>= %v)", got.QueuedAt, before)
	}
	if !got.CreatedAt.Equal(messageTime) {
		// CreatedAt keeps meaning "when the work was requested": dedup and the
		// landed-card cutoff both read it.
		t.Errorf("created_at changed to %v by the claim, want the message time %v", got.CreatedAt, messageTime)
	}

	// Every read path, not just GetTask: the detector reads through ListTasks.
	list, err := store.ListTasks(ctx, &TaskFilter{Status: []TaskStatus{TaskStatusQueued}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].QueuedAt == nil {
		t.Fatalf("ListTasks does not carry queued_at (%d rows) — the detector reads this path", len(list))
	}
}

// MU: restore `created_at < ?` as the queued-task arm of RecoverStaleTasks and this fails.
func TestRecoverStaleTasksAgesFromQueuedAt(t *testing.T) {
	ctx := context.Background()
	store := newQueuedAtStore(t)
	if err := store.CreateTask(ctx, &TaskRecord{
		ID: "task-just-claimed", Title: "old message, fresh claim", Status: TaskStatusPending,
		AgentID: "design-doc-creator", CreatedAt: time.Now().Add(-50 * time.Hour),
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.MarkTaskQueued(ctx, "task-just-claimed"); err != nil {
		t.Fatalf("claim: %v", err)
	}

	n, err := store.RecoverStaleTasks(ctx, time.Hour)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if n != 0 {
		t.Errorf("RecoverStaleTasks cancelled %d task(s) claimed seconds ago because the message is 50h old", n)
	}
}
