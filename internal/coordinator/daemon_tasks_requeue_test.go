package coordinator

import (
	"context"
	"errors"
	"testing"
)

// shutdownOnClaimStore cancels the daemon's context the moment a claim
// succeeds: SIGTERM landing between MarkTaskQueued and the dispatch, which is
// what stranded task-d82dc6c8 and task-18a82186 (D-FLEET-40).
type shutdownOnClaimStore struct {
	*SQLiteStore
	cancel context.CancelFunc
}

func (s *shutdownOnClaimStore) MarkTaskQueued(ctx context.Context, id string) error {
	err := s.SQLiteStore.MarkTaskQueued(ctx, id)
	if err == nil {
		s.cancel()
	}
	return err
}

// A task claimed in the tick SIGTERM interrupts must go back to pending, not
// stay queued with no job behind it until the 3-h stale detector fails it.
func TestTaskClaimedDuringShutdownIsRequeued(t *testing.T) {
	d, s := laneTestDaemon(t,
		&TaskRecord{ID: "claimed", AgentID: "sprint-planner", Title: "x", Content: "x", Status: TaskStatusPending},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.ctx = ctx
	d.taskStore = &shutdownOnClaimStore{SQLiteStore: s, cancel: cancel}

	// The codex lane check runs on d.ctx after the claim, so it fails on the
	// cancelled context, as the Pub/Sub publish did in prod.
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}
	assertTaskStatus(t, s, "claimed", TaskStatusPending)
}

// The transient dispatch-error path re-queues on the same helper.
func TestTransientDispatchErrorRequeuesAfterShutdown(t *testing.T) {
	d, s := laneTestDaemon(t,
		&TaskRecord{ID: "queued", AgentID: "designer", Title: "x", Content: "x", Status: TaskStatusQueued},
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d.ctx = ctx

	d.handleDispatchError(&TaskRecord{ID: "queued", AgentID: "designer"}, errors.New("run job: context canceled"))
	assertTaskStatus(t, s, "queued", TaskStatusPending)
}

func assertTaskStatus(t *testing.T, s *SQLiteStore, id string, want TaskStatus) {
	t.Helper()
	got, err := s.GetTask(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != want {
		t.Fatalf("task %s status = %s, want %s", id, got.Status, want)
	}
}
