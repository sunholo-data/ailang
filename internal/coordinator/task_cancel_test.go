package coordinator

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func newCancelStore(t *testing.T, status TaskStatus) (*SQLiteStore, string) {
	t.Helper()
	store, err := NewSQLiteStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	task := &TaskRecord{ID: "task-cancel1", AgentID: "a", Title: "t", Content: "c", Status: TaskStatusPending}
	if err := store.CreateTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if status != TaskStatusPending {
		if _, err := store.db.Exec("UPDATE tasks SET status = ? WHERE id = ?", status, task.ID); err != nil {
			t.Fatal(err)
		}
	}
	return store, task.ID
}

func statusOf(t *testing.T, s *SQLiteStore, id string) TaskStatus {
	t.Helper()
	task, err := s.GetTask(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return task.Status
}

func TestMarkTaskCancelledOnlyFromPending(t *testing.T) {
	ctx := context.Background()
	s, id := newCancelStore(t, TaskStatusPending)
	if err := s.MarkTaskCancelled(ctx, id); err != nil {
		t.Fatalf("cancel pending: %v", err)
	}
	if got := statusOf(t, s, id); got != TaskStatusCancelled {
		t.Fatalf("status = %s, want cancelled", got)
	}
	for _, st := range []TaskStatus{TaskStatusQueued, TaskStatusRunning, TaskStatusPendingApproval, TaskStatusCompleted} {
		s, id := newCancelStore(t, st)
		err := s.MarkTaskCancelled(ctx, id)
		if !errors.Is(err, ErrTaskNotCancellable) || !strings.Contains(err.Error(), string(st)) {
			t.Errorf("%s: err = %v, want ErrTaskNotCancellable naming the status", st, err)
		}
		if got := statusOf(t, s, id); got != st {
			t.Errorf("%s: status changed to %s", st, got)
		}
	}
}

// The 2026-09-30 race: a task cancelled while a dispatch attempt is in flight
// must stay cancelled when that attempt fails and resets it.
func TestResetTaskToPendingDoesNotResurrectACancelledTask(t *testing.T) {
	ctx := context.Background()
	for st, want := range map[TaskStatus]TaskStatus{
		TaskStatusQueued:    TaskStatusPending,
		TaskStatusRunning:   TaskStatusPending, // worktree-limit recovery
		TaskStatusCancelled: TaskStatusCancelled,
		TaskStatusCompleted: TaskStatusCompleted,
		TaskStatusFailed:    TaskStatusFailed,
	} {
		s, id := newCancelStore(t, st)
		if err := s.ResetTaskToPending(ctx, id); err != nil {
			t.Fatalf("%s: %v", st, err)
		}
		if got := statusOf(t, s, id); got != want {
			t.Errorf("reset from %s: status = %s, want %s", st, got, want)
		}
	}
}

// Cancel first, then the dispatcher's claim: the claim must lose.
func TestCancelledTaskCannotBeClaimed(t *testing.T) {
	ctx := context.Background()
	s, id := newCancelStore(t, TaskStatusPending)
	if err := s.MarkTaskCancelled(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkTaskQueued(ctx, id); !errors.Is(err, ErrTaskNotClaimable) {
		t.Fatalf("claim of a cancelled task: err = %v, want ErrTaskNotClaimable", err)
	}
}
