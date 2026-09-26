package coordinator

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// The auto-approve path (Daemon.sendHandoffMessage, from handleAgentHandoffs)
// and the approval path (sendAgentHandoffMessage, from dispatchApprovalHandoffs
// and OnAgentApproved) used to be two senders with two envelope shapes. This
// test drives BOTH real entry points against separate stores for the same
// task and asserts the rows they leave behind are the same, field for field —
// so a future edit to one path that changes what the next stage is handed
// fails here rather than in production (M-V1-SIMPLIFY-S4 M3B).
func TestHandoffSender_AutoAndApprovalPathsProduceTheSameEnvelope(t *testing.T) {
	source := &AgentConfig{ID: "sprint-planner", Label: "Sprint Planner", Inbox: "sprint-planner",
		TriggerOnComplete: []string{"sprint-executor"}, AutoApproveHandoffTo: []string{"sprint-executor"}}
	target := &AgentConfig{ID: "sprint-executor", Label: "Sprint Executor", Inbox: "sprint-executor"}
	task := &TaskRecord{
		ID:             "task-aaaa1111",
		AgentID:        source.ID,
		Title:          "Handoff: Plan: cache columns",
		Content:        "Plan the cache-token columns for the observatory",
		SprintPlanPath: "design_docs/planned/m-cache-columns-sprint-plan.md",
		ChainID:        "chain-7",
		GithubIssue:    912,
		WorktreeID:     "coordinator/task-aaaa1111",
		BaseBranch:     "dev",
		CreatedAt:      time.Now(),
	}
	artifacts := []string{"design_docs/planned/m-cache-columns-sprint-plan.md"}

	openStore := func(name string) messaging.MessageStore {
		t.Helper()
		st, err := messaging.OpenStore(filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.Close() })
		return st
	}
	autoStore := openStore("auto.db")
	approvalStore := openStore("approval.db")

	// Auto-approve path: the daemon, with no publisher (a local plane).
	d := &Daemon{msgStore: autoStore, logger: log.New(io.Discard, "", 0), ctx: context.Background()}
	if err := d.sendHandoffMessage(source, target, task, artifacts); err != nil {
		t.Fatalf("auto path: %v", err)
	}
	// Approval path: no Pub/Sub in the test environment, so the row is stored
	// and the sender reports exactly that — never a swallowed success.
	err := sendAgentHandoffMessage(context.Background(), nil, approvalStore, source, target, task, artifacts, task.GithubIssue)
	if err != nil && !errors.Is(err, errHandoffNotDispatched) {
		t.Fatalf("approval path: %v", err)
	}

	inboxRow := func(st messaging.MessageStore) messaging.InboxMessage {
		t.Helper()
		msgs, err := st.ListInboxMessages(messaging.InboxListOptions{Inbox: target.Inbox})
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 1 {
			t.Fatalf("want exactly one handoff row in %s, got %d", target.Inbox, len(msgs))
		}
		m := msgs[0]
		// Identity and clock differ per store by construction.
		m.ID, m.MessageID, m.CreatedAt, m.Simhash, m.Embedding, m.EmbeddingModel, m.EmbeddingUpdatedAt, m.Envelope = "", "", time.Time{}, nil, "", "", nil, nil
		return m
	}
	autoRow, approvalRow := inboxRow(autoStore), inboxRow(approvalStore)
	if !reflect.DeepEqual(autoRow, approvalRow) {
		t.Errorf("the two paths hand the next stage different rows:\nauto:     %+v\napproval: %+v", autoRow, approvalRow)
	}

	// The fields every consumer reads, asserted by name so a regression names
	// the reader it breaks.
	if autoRow.MessageType != messaging.InboxTypeHandoff {
		t.Errorf("MessageType = %q (daemon_tasks_polling routes on it)", autoRow.MessageType)
	}
	if autoRow.ParentTaskID != task.ID || autoRow.ChainID != task.ChainID {
		t.Errorf("ParentTaskID/ChainID = %q/%q (task hierarchy, DedupScope, chain join)", autoRow.ParentTaskID, autoRow.ChainID)
	}
	if autoRow.CorrelationID != task.ID {
		t.Errorf("CorrelationID = %q (handlers_inbox exposes it as the event's task_id)", autoRow.CorrelationID)
	}
	if autoRow.Title != "Handoff: Plan: cache columns" {
		t.Errorf("Title = %q: the prefix must appear once, however many stages deep", autoRow.Title)
	}
	if want := handoffContent(source, task, task.GithubIssue, artifacts); autoRow.Payload != want {
		t.Errorf("Payload is not handoffContent — the auto path used to quote raw executor output instead:\n%s", autoRow.Payload)
	}

	// No thread-trail message on either path. The daemon used to write one with
	// an empty thread id, which SQLite rejects, so it never existed here anyway.
	for name, st := range map[string]messaging.MessageStore{"auto": autoStore, "approval": approvalStore} {
		msgs, err := st.GetMessages(target.Inbox, target.ID, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 0 {
			t.Errorf("%s path wrote %d thread messages; the envelope is the inbox row only", name, len(msgs))
		}
	}
}

func TestHandoffSender_RefusesWithoutInboxOrStore(t *testing.T) {
	task := &TaskRecord{ID: "task-1", Title: "t"}
	source := &AgentConfig{ID: "a", Label: "A"}
	if err := (handoffSender{}).send(handoff{Source: source, Target: &AgentConfig{ID: "b", Inbox: "b"}, Task: task}); err == nil {
		t.Error("no message store must be an error, not a silent no-op")
	}
	st, err := messaging.OpenStore(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := (handoffSender{msgStore: st}).send(handoff{Source: source, Target: &AgentConfig{ID: "b"}, Task: task}); err == nil {
		t.Error("a target with no inbox must be an error")
	}
	if err := (handoffSender{msgStore: st}).send(handoff{Target: &AgentConfig{ID: "b", Inbox: "b"}, Task: task}); err == nil {
		t.Error("a handoff with no source agent must be an error")
	}
}
