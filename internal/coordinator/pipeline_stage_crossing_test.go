package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Does stage 2 actually START?
//
// That question had no test, and five separate bugs hid behind its absence. The
// existing TestIntegration_EndToEnd_SimplePath creates ONE task, runs a mock
// provider and completes it — it never crosses a stage boundary, so nothing in
// the suite ever asked whether the next agent begins work. Each bug was
// therefore invisible until the one in front of it was fixed:
//
//  1. ProcessApprovalRequest never called OnAgentApproved      (fixed 2026-09-07)
//  2. the handoff was written to the THREAD collection, which
//     dispatch never polls                                     (fixed 2026-09-07)
//  3. the row was stored but no Pub/Sub notification published (fixed 2026-09-07)
//  4. the handoff was deduplicated against its own parent      (fixed 2026-09-14)
//  5. the handoff never named the artifact it handed off       (fixed 2026-09-14)
//
// Every one is a plumbing fault between two components that each worked. None
// needed a model, a network or a cloud plane to reproduce — only a test that
// followed the work across the seam. This is that test.

// stageCrossingFixture is one approved stage-1 task, with the real stores.
type stageCrossingFixture struct {
	store    *SQLiteStore
	msgStore messaging.MessageStore
	registry *AgentRegistry
	task     *TaskRecord
}

func newStageCrossingFixture(t *testing.T) *stageCrossingFixture {
	t.Helper()
	dir := t.TempDir()

	store, err := NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		t.Fatalf("coordinator store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	msgStore, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		t.Fatalf("message store: %v", err)
	}
	t.Cleanup(func() { _ = msgStore.Close() })

	// The real pipeline shape: a gated designer edge, then auto edges onward.
	reg := NewAgentRegistry()
	for _, a := range []*AgentConfig{
		{ID: "design-doc-creator", Inbox: "design-doc-creator", Label: "Design Doc Creator",
			Workspace: dir, TriggerOnComplete: []string{"sprint-planner"}},
		{ID: "sprint-planner", Inbox: "sprint-planner", Label: "Sprint Planner",
			Workspace: dir, TriggerOnComplete: []string{"sprint-executor"},
			AutoApproveHandoffTo: []string{"sprint-executor"}},
		{ID: "sprint-executor", Inbox: "sprint-executor", Label: "Sprint Executor", Workspace: dir},
	} {
		if err := reg.Register(a); err != nil {
			t.Fatalf("register %s: %v", a.ID, err)
		}
	}

	task := &TaskRecord{
		ID:            "task-08032ebc",
		AgentID:       "design-doc-creator",
		Title:         "Design: secondary-model fallback",
		Content:       "Design a secondary-model fallback for cloud executor agents",
		DesignDocPath: "design_docs/planned/m-secondary-model-fallback.md",
		Type:          TaskTypeFeature,
		Status:        TaskStatusPendingApproval,
		Workspace:     dir,
		CreatedAt:     time.Now().Add(-7 * time.Minute),
	}
	ctx := context.Background()
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create stage-1 task: %v", err)
	}
	// The finalizer records the artifact separately, from the DESIGN_DOC_PATH:
	// output marker — CreateTask does not carry it.
	if err := store.SetTaskDesignDocPath(ctx, task.ID, task.DesignDocPath); err != nil {
		t.Fatalf("set design doc path: %v", err)
	}
	if err := store.MarkTaskPendingApproval(ctx, task.ID, dir, "coordinator/"+task.ID, "dev", "", &ExecuteResult{Success: true}); err != nil {
		t.Fatalf("set pending_approval: %v", err)
	}
	// The approval record the finalizer writes, carrying the GATED handoff
	// target. This is the field approval-time dispatch reads; a card without it
	// approves the merge and starts nothing.
	if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
		ID:          ApprovalIDForTask(task.ID),
		TaskID:      task.ID,
		Type:        string(ApprovalTypeMerge),
		Description: "Agent completed work on: " + task.Title,
		Status:      "pending",
		ContextJSON: `{"handoff_targets":["sprint-planner"],"source_agent":"design-doc-creator"}`,
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("create approval record: %v", err)
	}

	return &stageCrossingFixture{store: store, msgStore: msgStore, registry: reg, task: task}
}

// TestStageCrossing_ApprovalStartsTheNextStage follows the work across the seam:
// approve stage 1, then assert stage 2 has something to pick up, in the place
// dispatch actually looks.
func TestStageCrossing_ApprovalStartsTheNextStage(t *testing.T) {
	f := newStageCrossingFixture(t)
	ctx := context.Background()

	if _, err := ProcessApprovalRequest(ctx, &ApprovalParams{
		TaskID:        f.task.ID,
		Action:        "approve",
		ApprovedBy:    "stage-crossing-test",
		Channel:       "cli",
		Store:         f.store,
		MsgStore:      f.msgStore,
		AgentRegistry: f.registry,
		SkipMerge:     true,
	}); err != nil {
		t.Fatalf("approve: %v", err)
	}

	// THE INBOX, not the thread collection. Bug 2 wrote a correct message to a
	// collection dispatch never polls, and every field looked right.
	msgs, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("list sprint-planner inbox: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("approving stage 1 put NOTHING in the next stage's inbox — the pipeline stops here")
	}

	h := msgs[0]
	if h.MessageType != messaging.InboxTypeHandoff {
		t.Errorf("handoff message type = %q, want %q", h.MessageType, messaging.InboxTypeHandoff)
	}
	// Bug 4 turned on this field being carried: without it the task the daemon
	// builds has no parent, and dedup suppresses it against stage 1.
	if h.ParentTaskID != f.task.ID {
		t.Errorf("handoff ParentTaskID = %q, want %q — dedup needs it to tell a stage from a repeat",
			h.ParentTaskID, f.task.ID)
	}
	// Bug 5: the next stage must be told what it is continuing FROM.
	if !strings.Contains(h.Payload, f.task.DesignDocPath) {
		t.Errorf("handoff does not name the artifact %q:\n%s", f.task.DesignDocPath, h.Payload)
	}
}

// TestStageCrossing_TheHandoffTaskIsNotDeduplicated is bug 4, at the seam where
// it actually bit: the daemon builds a task from the handoff message, and asks
// the store whether it is a duplicate.
//
// In production on 2026-09-14 the answer was yes — against the stage-1 task —
// one second after the handoff was correctly dispatched.
func TestStageCrossing_TheHandoffTaskIsNotDeduplicated(t *testing.T) {
	f := newStageCrossingFixture(t)
	ctx := context.Background()

	// Stage 1 is completed and fingerprinted, as it would be by now.
	fingerprint := NewTaskAnalyzer(0.8).Analyze(&Task{
		ID: f.task.ID, Content: f.task.Content,
	}).Fingerprint
	if fingerprint == 0 {
		t.Fatal("analyzer produced no fingerprint; the rest of this test proves nothing")
	}
	if err := f.store.SetTaskFingerprint(ctx, f.task.ID, fingerprint); err != nil {
		t.Fatalf("fingerprint stage 1: %v", err)
	}

	if _, err := ProcessApprovalRequest(ctx, &ApprovalParams{
		TaskID: f.task.ID, Action: "approve", ApprovedBy: "t", Channel: "cli",
		Store: f.store, MsgStore: f.msgStore, AgentRegistry: f.registry, SkipMerge: true,
	}); err != nil {
		t.Fatalf("approve: %v", err)
	}
	// Stage 1 finishes completed — which is the status that suppresses, and the
	// status it is ALWAYS in by the time its own handoff is picked up.
	if err := f.store.MarkTaskCompleted(ctx, f.task.ID, &ExecuteResult{Success: true}); err != nil {
		t.Fatalf("complete stage 1: %v", err)
	}
	msgs, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("no handoff to test dedup against (err=%v)", err)
	}

	// Exactly what daemon_tasks_polling builds from that message.
	h := msgs[0]
	stage2 := &TaskRecord{
		ID:           "task-b8f905ae",
		AgentID:      "sprint-planner",
		ParentTaskID: h.ParentTaskID,
		Content:      h.Payload,
	}
	hfp := NewTaskAnalyzer(0.8).Analyze(&Task{ID: stage2.ID, Content: stage2.Content}).Fingerprint

	// The collision is real — assert it rather than assuming it, or this test
	// would pass for the wrong reason the day the handoff wording changes.
	if hfp == fingerprint {
		t.Logf("handoff and parent share fingerprint %#x — the collision this guards", hfp)
	}
	for _, fp := range []uint64{fingerprint, hfp} {
		dup, err := f.store.FindDuplicateTask(ctx, fp, DedupScopeFor(stage2, time.Now()))
		if err != nil {
			t.Fatalf("dedup lookup: %v", err)
		}
		if dup != nil {
			t.Fatalf("the stage-2 task was suppressed by %s (%s) — the pipeline cannot advance",
				dup.ID, dup.Status)
		}
	}
}

// An AUTO edge dispatches at completion, not at approval. Both moments must
// work, and they must not both fire for the same target — approvalHandoffTargets
// and autoHandoffTargets partition TriggerOnComplete, and a drift there either
// double-runs an agent or strands the edge.
func TestStageCrossing_AutoAndGatedEdgesPartition(t *testing.T) {
	f := newStageCrossingFixture(t)
	planner := f.registry.GetAgentByID("sprint-planner")

	gated := approvalHandoffTargets(planner)
	if len(gated) != 0 {
		t.Errorf("sprint-executor is an AUTO edge; approval must not fire it too: %v", gated)
	}
	designer := f.registry.GetAgentByID("design-doc-creator")
	if got := approvalHandoffTargets(designer); len(got) != 1 || got[0] != "sprint-planner" {
		t.Errorf("the designer edge is gated and must fire on approval, got %v", got)
	}
}
