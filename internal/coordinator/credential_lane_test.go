package coordinator

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"testing"
)

func TestCredentialLane(t *testing.T) {
	cases := []struct {
		agent    *AgentConfig
		provider string
		want     string
	}{
		{&AgentConfig{ID: "sprint-planner"}, "codex", laneCodexSubscription},
		{nil, "codex", laneCodexSubscription},
		{&AgentConfig{ID: "metered", AuthMode: "apikey"}, "codex", ""},
		{&AgentConfig{ID: "doc"}, "claude", ""},
		{&AgentConfig{ID: "doc"}, "pi", ""},
	}
	for _, c := range cases {
		if got := credentialLane(c.agent, c.provider); got != c.want {
			t.Errorf("credentialLane(%+v, %q) = %q, want %q", c.agent, c.provider, got, c.want)
		}
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
