package coordinator

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

// MarkTaskQueued is the CLAIM that decides which dispatcher owns a task.
//
// It could not refuse. Both stores wrote `status = queued` unconditionally, so
// every concurrent caller "succeeded" and every one dispatched. Measured on the
// prod plane 2026-09-17 20:06:36 — one task, ONE creation, three dispatches
// inside 500ms:
//
//	20:06:36.943  Created task task-503c588e ... from cloud message inbox_...
//	20:06:37.944  Cloud dispatch: task task-503c588e → Cloud Run Job
//	20:06:38.214  Cloud dispatch: task task-503c588e → Cloud Run Job
//	20:06:38.418  Cloud dispatch: task task-503c588e → Cloud Run Job
//
// Three Cloud Run executions, three commits, three completions, one task.
func newClaimTestStore(t *testing.T) (*SQLiteStore, context.Context) {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, context.Background()
}

func mkPendingTask(t *testing.T, store *SQLiteStore, ctx context.Context, id string) {
	t.Helper()
	if err := store.CreateTask(ctx, &TaskRecord{
		ID:     id,
		Status: TaskStatusPending,
		Type:   "bug-fix",
	}); err != nil {
		t.Fatalf("create task %s: %v", id, err)
	}
}

func TestMarkTaskQueued_ClaimsOnlyOnce(t *testing.T) {
	store, ctx := newClaimTestStore(t)
	mkPendingTask(t, store, ctx, "task-claim")

	// Control: the first claim must succeed, or "the second one failed" below
	// would pass even if claiming were broken outright.
	if err := store.MarkTaskQueued(ctx, "task-claim"); err != nil {
		t.Fatalf("control failed: first claim returned %v, want nil", err)
	}

	err := store.MarkTaskQueued(ctx, "task-claim")
	if !errors.Is(err, ErrTaskNotClaimable) {
		t.Fatalf("second claim = %v, want ErrTaskNotClaimable — the task would be dispatched twice", err)
	}

	task, err := store.GetTask(ctx, "task-claim")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status != TaskStatusQueued {
		t.Errorf("status = %q, want %q", task.Status, TaskStatusQueued)
	}
}

// A task that has moved past pending must not be re-claimable either — that is
// the shape of the 09-15 fault, where a task already finalised into
// pending_approval was dispatched a second time.
func TestMarkTaskQueued_RefusesNonPendingStatuses(t *testing.T) {
	for _, status := range []TaskStatus{
		TaskStatusRunning,
		TaskStatusPendingApproval,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled,
	} {
		t.Run(string(status), func(t *testing.T) {
			store, ctx := newClaimTestStore(t)
			id := "task-" + string(status)
			if err := store.CreateTask(ctx, &TaskRecord{ID: id, Status: status, Type: "bug-fix"}); err != nil {
				t.Fatalf("create: %v", err)
			}

			if err := store.MarkTaskQueued(ctx, id); !errors.Is(err, ErrTaskNotClaimable) {
				t.Fatalf("claim of a %q task = %v, want ErrTaskNotClaimable", status, err)
			}

			// The refusal must also leave the status alone: silently rewriting a
			// completed task to queued would lose the outcome even if nothing
			// dispatched.
			task, err := store.GetTask(ctx, id)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if task.Status != status {
				t.Errorf("refused claim changed status %q -> %q", status, task.Status)
			}
		})
	}
}

// The fault was a RACE, so the test has to race. Serial calls would pass
// against a claim that merely checks-then-writes without atomicity.
func TestMarkTaskQueued_ConcurrentClaimersYieldExactlyOneWinner(t *testing.T) {
	store, ctx := newClaimTestStore(t)
	mkPendingTask(t, store, ctx, "task-race")

	const claimers = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var won int
	var otherErr error

	start := make(chan struct{})
	for i := 0; i < claimers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release them together
			err := store.MarkTaskQueued(ctx, "task-race")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				won++
			case errors.Is(err, ErrTaskNotClaimable):
			default:
				otherErr = err
			}
		}()
	}
	close(start)
	wg.Wait()

	if otherErr != nil {
		t.Fatalf("unexpected store error during race: %v", otherErr)
	}
	if won != 1 {
		t.Fatalf("%d of %d claimers won, want exactly 1 — each winner is a Cloud Run execution", won, claimers)
	}
}
