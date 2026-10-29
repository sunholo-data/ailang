package coordinator

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// A shadow's completion posts no notice to the agent's inbox; an ordinary
// completion still does (the portal and sidecar poll for it).
func TestCompletionNoticeSkippedForShadow(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	msgs, err := messaging.OpenStore(filepath.Join(dir, "msgs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = msgs.Close() })
	registry := NewAgentRegistry()
	if err := registry.Register(&AgentConfig{ID: "pkg-x", Inbox: "pkg:sunholo/x"}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, rec := range []*TaskRecord{
		{ID: "task-p", Title: "x", Content: "x", AgentID: "pkg-x", Status: TaskStatusPending},
		{ID: "task-p-shadow", ShadowOf: "task-p", Title: "x", Content: "x", AgentID: "pkg-x", Status: TaskStatusPending},
	} {
		if err := store.CreateTask(ctx, rec); err != nil {
			t.Fatal(err)
		}
		if _, err := store.CompareAndSetTaskStatus(ctx, rec.ID, AllTaskStatuses(), TaskStatusRunning); err != nil {
			t.Fatal(err)
		}
	}
	h := NewCompletionHandler(nil, store, msgs, registry, log.New(io.Discard, "", 0))
	notices := func() int {
		list, err := msgs.ListInboxMessages(messaging.InboxListOptions{Inbox: "pkg:sunholo/x"})
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}
	deliver := func(id string) {
		data, _ := json.Marshal(pubsub.TaskCompletion{TaskID: id, AgentID: "pkg-x", Status: "failed", NumTurns: 1})
		if err := h.HandleCompletion(data, nil); err != nil {
			t.Fatal(err)
		}
	}

	deliver("task-p-shadow")
	if st, _ := store.GetTask(ctx, "task-p-shadow"); st.Status != TaskStatusFailed {
		t.Fatalf("shadow completion was not processed (status %s); the assertion below would be vacuous", st.Status)
	}
	if n := notices(); n != 0 {
		t.Fatalf("shadow completion posted %d inbox notices", n)
	}
	deliver("task-p")
	if n := notices(); n != 1 {
		t.Fatalf("ordinary completion posted %d inbox notices, want 1", n)
	}
}

// The local worker never runs a pending shadow: it is cloud-only in v1, and a
// local run would have full side effects.
func TestLocalWorkerSkipsShadow(t *testing.T) {
	d, s := shadowTestDaemon(t, nil, &TaskRecord{
		ID: "task-p-shadow", ShadowOf: "task-p", Title: "x", Content: "x", Status: TaskStatusPending,
	})
	d.executor = &TaskExecutor{} // non-nil so the local path is taken
	if err := d.executeTaskQueue(); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetTask(d.ctx, "task-p-shadow"); got.Status != TaskStatusPending {
		t.Fatalf("local worker touched a shadow: status %s", got.Status)
	}
}
