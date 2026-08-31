package coordinator

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"time"
)

// TestUnknownAgeIsNotATimeout is the regression arm for the prod incident of
// 2026-08-31.
//
// getTaskAge fell through to time.Since(task.CreatedAt) with no zero check, so a
// task persisted without a created_at aged as ~292 years (the zero time), blew
// past every timeout, and was marked failed on the first tick after dispatch —
// about 57 seconds. Observed in prod on task-a855b349 and task-133e933b, both
// written with created_at = null:
//
//	stale task detector: marking task task-a855b349 as failed: task timed out:
//	no completion received within 22m30s of being queued (age=2562047h47m16.854775807s)
//
// Each such kill posted a failure notice into the agent's own inbox, which the
// backstop sweep then dispatched as fresh work. An unknown age is a data defect
// to report, never a timeout to act on.
//
// MU: restore the bare `return time.Since(task.CreatedAt)` and this fails.
func TestUnknownAgeIsNotATimeout(t *testing.T) {
	store := NewMockStore()
	if err := store.CreateTask(context.Background(), &TaskRecord{
		ID:      "task-133e933b",
		Status:  TaskStatusRunning,
		AgentID: "docparse",
		// CreatedAt and StartedAt deliberately zero: exactly what a
		// created_at = null row reads back as.
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	var logBuf strings.Builder
	d := NewStaleTaskDetector(store, nil, nil, log.New(&logBuf, "", 0))
	d.detectAndMarkStale(context.Background())

	if got := store.calls["MarkTaskFailed"]; got != 0 {
		t.Errorf("a task with no created_at and no started_at was marked failed (%d call(s)); "+
			"time.Since(zero) is ~292 years, so this kills every such task seconds after dispatch", got)
	}
	if !strings.Contains(logBuf.String(), "age unknowable") {
		t.Errorf("an unknowable age must be reported loudly, not passed over in silence; got:\n%s",
			logBuf.String())
	}
}

// TestKnownAgeStillTimesOut keeps the fix honest: the zero-guard must not turn
// the detector off. A genuinely old task is still failed.
func TestKnownAgeStillTimesOut(t *testing.T) {
	store := NewMockStore()
	if err := store.CreateTask(context.Background(), &TaskRecord{
		ID:        "task-genuinely-stale",
		Status:    TaskStatusRunning,
		AgentID:   "docparse",
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	d := NewStaleTaskDetector(store, nil, nil, log.New(io.Discard, "", 0))
	d.detectAndMarkStale(context.Background())

	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Fatalf("MarkTaskFailed called %d times, want 1 — the zero-guard must not disable the detector", got)
	}
}
