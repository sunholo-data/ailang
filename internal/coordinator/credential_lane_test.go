package coordinator

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// failTaskAt fails a task as a completion would and backdates the failure.
func failTaskAt(t *testing.T, s *SQLiteStore, id string, ago time.Duration) {
	t.Helper()
	if err := s.MarkTaskFailed(context.Background(), id, errors.New("executor failed: codex task failed: signal: killed")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE tasks SET completed_at = ? WHERE id = ?", time.Now().Add(-ago), id); err != nil {
		t.Fatal(err)
	}
}

// The 2026-10-16 shape (D-FLEET-39): attempt 0 of task-d61964fa published
// `failed` and released the lease, the Cloud Run retry took it again, and the
// planner handoff dispatched 3m47s later died on "credential already owned".
// The failed task must keep the lane while its retry can still be running.
func TestCodexTaskWaitsWhileAFailedTasksRetryMayHoldTheLane(t *testing.T) {
	d, s := laneTestDaemon(t,
		&TaskRecord{ID: "d61964fa", AgentID: "sprint-executor", Title: "x", Content: "x", Status: TaskStatusQueued},
		&TaskRecord{ID: "handoff", AgentID: "sprint-planner", Title: "y", Content: "y", Status: TaskStatusPending},
	)
	failTaskAt(t, s, "d61964fa", 3*time.Minute+47*time.Second)
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(d.ctx, "handoff")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TaskStatusPending {
		t.Fatalf("handoff status = %s, want pending while the failed task's retry may hold the codex lane", got.Status)
	}
}

func TestFailedTaskHoldsTheLaneOnlyForItsRetryWindow(t *testing.T) {
	d, s := laneTestDaemon(t,
		// Pending, so it occupies nothing until it fails below.
		&TaskRecord{ID: "codex-recent", AgentID: "sprint-executor", Title: "a", Content: "a", Status: TaskStatusPending},
		&TaskRecord{ID: "codex-old", AgentID: "sprint-executor", Title: "b", Content: "b", Status: TaskStatusQueued},
		&TaskRecord{ID: "claude-recent", AgentID: "designer", Title: "c", Content: "c", Status: TaskStatusQueued},
	)
	// sprint-executor declares no timeout here, so its window is the default + grace.
	window := (*AgentConfig)(nil).GetEffectiveTimeout() + credentialLaneRetryGrace
	failTaskAt(t, s, "codex-old", window+time.Minute)
	failTaskAt(t, s, "claude-recent", time.Minute)
	if h, err := d.credentialLaneHolder(d.ctx, laneCodexSubscription, "self", map[string]string{}); err != nil || h != "" {
		t.Fatalf("holder = %q, %v; want free: one codex failure is past its retry window, the other failure is not on the lane", h, err)
	}
	failTaskAt(t, s, "codex-recent", window-time.Minute)
	h, err := d.credentialLaneHolder(d.ctx, laneCodexSubscription, "self", map[string]string{})
	if err != nil || !strings.HasPrefix(h, "codex-recent ") {
		t.Fatalf("holder = %q, %v; want codex-recent, failed inside its retry window", h, err)
	}
}

func TestCredentialLane(t *testing.T) {
	for provider, want := range map[string]string{"codex": laneCodexSubscription, "agy": laneAgySubscription, "claude": "", "pi": "", "gemini_agents": ""} {
		if got := credentialLane(provider); got != want {
			t.Errorf("credentialLane(%q) = %q, want %q", provider, got, want)
		}
	}
	// Separate credentials, separate lanes: a running codex task must not hold
	// back an agy task, or the reverse (M-AGY-CLOUD).
	if laneAgySubscription == laneCodexSubscription || laneAgySubscription == "" {
		t.Fatalf("agy lane %q must be its own non-empty lane", laneAgySubscription)
	}
}

func laneTestDaemon(t *testing.T, tasks ...*TaskRecord) (*Daemon, *SQLiteStore) {
	t.Helper()
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	for _, task := range tasks {
		if err := s.CreateTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	registry := NewAgentRegistry()
	for _, a := range []*AgentConfig{
		{ID: "sprint-planner", Inbox: "sprint-planner", Workspace: "a/b", Capabilities: []string{"docs"}, Provider: "codex", ExecutorVariant: "codex"},
		{ID: "sprint-executor", Inbox: "sprint-executor", Workspace: "a/b", Capabilities: []string{"docs"}, Provider: "codex", ExecutorVariant: "codex"},
		{ID: "designer", Inbox: "designer", Workspace: "a/b", Capabilities: []string{"docs"}, Provider: "claude"},
	} {
		if err := registry.Register(a); err != nil {
			t.Fatal(err)
		}
	}
	return &Daemon{taskStore: s, agentRegistry: registry, ctx: ctx, logger: log.New(io.Discard, "", 0)}, s
}

// The 2026-10-13 shape: a planner handoff arrives while another codex job runs.
// It must wait as pending, not reach dispatch (a nil publisher would panic).
func TestCodexTaskWaitsWhileTheLaneIsHeld(t *testing.T) {
	d, s := laneTestDaemon(t,
		&TaskRecord{ID: "running", AgentID: "sprint-executor", Title: "x", Content: "x", Status: TaskStatusRunning},
		&TaskRecord{ID: "handoff", AgentID: "sprint-planner", Title: "y", Content: "y", Status: TaskStatusPending},
	)
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(d.ctx, "handoff")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != TaskStatusPending {
		t.Fatalf("handoff status = %s, want pending while the codex lane is held", got.Status)
	}
}

func TestCredentialLaneHolder(t *testing.T) {
	d, _ := laneTestDaemon(t,
		&TaskRecord{ID: "self", AgentID: "sprint-planner", Title: "a", Content: "a", Status: TaskStatusQueued},
		&TaskRecord{ID: "other-lane", AgentID: "designer", Title: "b", Content: "b", Status: TaskStatusRunning},
		&TaskRecord{ID: "done", AgentID: "sprint-executor", Title: "c", Content: "c", Status: TaskStatusFailed},
	)
	// Only self and non-codex/terminal tasks in flight: the lane is free.
	if h, err := d.credentialLaneHolder(d.ctx, laneCodexSubscription, "self", map[string]string{}); err != nil || h != "" {
		t.Fatalf("holder = %q, %v; want free", h, err)
	}
	// A lane claimed earlier in the same tick is held even before the store shows it.
	claimed := map[string]string{laneCodexSubscription: "earlier"}
	if h, _ := d.credentialLaneHolder(d.ctx, laneCodexSubscription, "self", claimed); h != "earlier" {
		t.Fatalf("holder = %q, want the task claimed this tick", h)
	}
}
