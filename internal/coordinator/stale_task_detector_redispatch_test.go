package coordinator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"
)

// M-COORDINATOR-EXECUTION-TRUST M3, tier 2 of the retry design.

func staleTask(id string, attempts int) *TaskRecord {
	return &TaskRecord{
		ID:           id,
		Status:       TaskStatusRunning,
		AgentID:      "docparse",
		AttemptCount: attempts,
		CreatedAt:    time.Now().Add(-24 * time.Hour), // unambiguously past any timeout
	}
}

func detectorFor(t *testing.T, task *TaskRecord) (*StaleTaskDetector, *MockStore, *strings.Builder) {
	t.Helper()
	store := NewMockStore()
	if err := store.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	var buf strings.Builder
	return NewStaleTaskDetector(store, nil, nil, log.New(&buf, "", 0)), store, &buf
}

// An infra-class timeout inside the cap gets its one re-dispatch, and is NOT
// also marked failed — doing both would duplicate work and close the task under
// the retry.
func TestInfraTimeoutIsReDispatchedOnce(t *testing.T) {
	d, store, _ := detectorFor(t, staleTask("task-1", 0))
	called := 0
	d.WithReDispatcher(func(_ context.Context, task *TaskRecord) error {
		called++
		task.AttemptCount++
		return nil
	})
	d.detectAndMarkStale(context.Background())

	if called != 1 {
		t.Errorf("expected exactly 1 re-dispatch, got %d", called)
	}
	if got := store.calls["MarkTaskFailed"]; got != 0 {
		t.Errorf("a re-dispatched task must not also be marked failed (got %d calls)", got)
	}
}

// MU-13: the cap is hard at the detector, not just in the predicate.
func TestCappedTaskIsFailedNotReDispatched(t *testing.T) {
	d, store, _ := detectorFor(t, staleTask("task-2", MaxTaskExecutions))
	called := 0
	d.WithReDispatcher(func(context.Context, *TaskRecord) error { called++; return nil })
	d.detectAndMarkStale(context.Background())

	if called != 0 {
		t.Errorf("a task at the execution cap must not be re-dispatched, got %d calls", called)
	}
	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Errorf("a capped task must reach a terminal state, got %d MarkTaskFailed calls", got)
	}
}

// MU-13c: without an explicit opt-in the detector re-dispatches nothing. Three
// other components can move a task toward terminal (V23); none of them may hold
// this hook, and the safe default for any caller that has not opted in is the
// pre-M3 behaviour.
func TestStaleDetectorIsTheSoleReDispatcher(t *testing.T) {
	d, store, _ := detectorFor(t, staleTask("task-3", 0))
	d.detectAndMarkStale(context.Background()) // no WithReDispatcher

	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Errorf("report-only default must still close the task, got %d", got)
	}
}

// A losing compare-and-set must be OBSERVABLE, and must not leave the task in
// limbo — it falls through to the terminal path.
func TestLosingReDispatchIsLoudAndStillTerminal(t *testing.T) {
	d, store, buf := detectorFor(t, staleTask("task-4", 0))
	d.WithReDispatcher(func(context.Context, *TaskRecord) error {
		return fmt.Errorf("compare-and-set lost: another instance already advanced this task")
	})
	d.detectAndMarkStale(context.Background())

	if !strings.Contains(buf.String(), "not re-dispatched") {
		t.Errorf("a losing re-dispatch must say so; log was:\n%s", buf.String())
	}
	if got := store.calls["MarkTaskFailed"]; got != 1 {
		t.Errorf("a task whose re-dispatch lost must still reach terminal, got %d", got)
	}
}
