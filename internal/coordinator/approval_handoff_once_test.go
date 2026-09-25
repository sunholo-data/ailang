package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// M-TASK-STATUS-TRUTH S3/S4 — a handoff fires once, and "do not fire" survives
// a reboot.
//
// Measured in prod: every approval that owed a handoff sent it TWICE — once by
// the approve path, which never recorded that it had, and again 30–80 min later
// when the coordinator next booted and triggerMissedHandoffs found the approval
// "without triggered handoffs" (task-08032ebc, task-080f4657, task-38dcb44a,
// task-90bb931d). And `prs --landed --apply`, which printed "handoffs NOT
// fired", recorded nothing either, so the next boot fired 22 of them
// (2026-09-23 15:17).
//
// Every test here asserts TWO things: exactly one row per target, AND that row
// was delivered (notified, or swept into the drain). A test that only counts
// rows passes on zero deliveries — the silent drop this work exists to remove.

type onceFixture struct {
	store    *SQLiteStore
	msgs     *messaging.Store
	registry *AgentRegistry
	task     *TaskRecord
	notified *notifyLog
}

type notifyLog struct {
	mu   sync.Mutex
	ids  []string
	fail map[string]bool // inbox -> fail the notify
}

func (n *notifyLog) notify(msg *messaging.InboxMessage) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.fail[msg.ToInbox] {
		return errHandoffNotDispatched
	}
	n.ids = append(n.ids, msg.ID)
	return nil
}

func (n *notifyLog) count(id string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	c := 0
	for _, x := range n.ids {
		if x == id {
			c++
		}
	}
	return c
}

func newOnceFixture(t *testing.T, targets ...string) *onceFixture {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	msgs, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		t.Fatalf("msgs: %v", err)
	}
	t.Cleanup(func() { _ = msgs.Close() })

	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{ID: "design-doc-creator", Label: "Design Doc Creator", Inbox: "design-doc-creator", TriggerOnComplete: targets}); err != nil {
		t.Fatal(err)
	}
	for _, tgt := range targets {
		if err := reg.Register(&AgentConfig{ID: tgt, Label: tgt, Inbox: tgt}); err != nil {
			t.Fatal(err)
		}
	}

	task := &TaskRecord{ID: "task-080f4657", Title: "Daneel design 5f382fb6", Content: "design it",
		Status: TaskStatusPendingApproval, AgentID: "design-doc-creator", CreatedAt: time.Now()}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompareAndSetTaskStatus(ctx, task.ID, AllTaskStatuses(), TaskStatusPendingApproval); err != nil {
		t.Fatal(err)
	}
	ctxJSON, _ := json.Marshal(map[string]interface{}{"handoff_targets": targets, "source_agent": "design-doc-creator"})
	if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
		ID: ApprovalIDForTask(task.ID), TaskID: task.ID, Type: "merge_handoff",
		ContextJSON: string(ctxJSON), Status: "pending", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	n := &notifyLog{fail: map[string]bool{}}
	prev := approvalHandoffNotify
	approvalHandoffNotify = n.notify
	t.Cleanup(func() { approvalHandoffNotify = prev })

	return &onceFixture{store: store, msgs: msgs, registry: reg, task: task, notified: n}
}

func (f *onceFixture) approve(t *testing.T, skipHandoffs bool) *ApprovalResult {
	t.Helper()
	res, err := ProcessApprovalRequest(context.Background(), &ApprovalParams{
		TaskID: f.task.ID, Action: "approve", ApprovedBy: "test", Channel: "cli",
		Store: f.store, MsgStore: f.msgs, AgentRegistry: f.registry,
		SkipMerge: true, SkipHandoffs: skipHandoffs,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	return res
}

// boot runs the daemon's startup recovery exactly as a fresh instance would.
func (f *onceFixture) boot(t *testing.T) {
	t.Helper()
	d := &Daemon{taskStore: f.store, msgStore: f.msgs, agentRegistry: f.registry,
		ctx: context.Background(), logger: log.New(io.Discard, "", 0)}
	if _, err := d.triggerMissedHandoffs(); err != nil {
		t.Fatalf("recovery: %v", err)
	}
}

func (f *onceFixture) rows(t *testing.T, inbox string) []messaging.InboxMessage {
	t.Helper()
	got, err := f.msgs.ListInboxMessages(messaging.InboxListOptions{Inbox: inbox})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// sweep runs one backstop pass in dispatch mode and returns what it enqueued.
func (f *onceFixture) sweep(t *testing.T) map[string]bool {
	t.Helper()
	adapter := &PubSubInboxAdapter{logger: log.New(io.Discard, "", 0)}
	s := NewBackstopSweep(f.msgs, f.registry, adapter, log.New(io.Discard, "", 0))
	s.mode = BackstopDispatch
	s.SweepOnce(context.Background())
	got := map[string]bool{}
	for _, m := range adapter.buffered {
		got[m.ID] = true
	}
	return got
}

// (a) MU: revert handoffSender.send to a plain InsertInboxMessage and this fails
// (the boot writes a second row).
func TestHandoffOnce_ApproveThenReboot(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner")
	f.approve(t, false)
	f.boot(t)
	f.boot(t) // two instances during a rollout (prod 09-23: 00157 + 00158)

	rows := f.rows(t, "sprint-planner")
	if len(rows) != 1 {
		t.Fatalf("%d handoff rows after approve + two boots, want 1 — every approval handed off twice in prod", len(rows))
	}
	if got := f.notified.count(rows[0].ID); got != 1 {
		t.Errorf("handoff notified %d times, want exactly 1", got)
	}
}

// (b) The crash window: the row was written, then the process died before the
// notify and before any marker. MU: make send notify on a collision and the
// notify count below becomes 1 from recovery — fine — but drop the sweep's
// handling of handoff rows and delivery fails.
func TestHandoffOnce_CrashAfterWriteIsDeliveredBySweep(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner")
	ctx := context.Background()
	// The approval resolved, the handoff row landed, nothing else happened.
	if err := f.store.ResolveApprovalRequestByTask(ctx, f.task.ID, "approved", "test"); err != nil {
		t.Fatal(err)
	}
	id := HandoffMessageID(f.task.ID, "sprint-planner")
	if _, err := f.msgs.PutMessageIfAbsent(ctx, &messaging.InboxMessage{
		ID: id, FromAgent: "coordinator", ToInbox: "sprint-planner",
		MessageType: messaging.InboxTypeHandoff, Title: "Handoff: x", ParentTaskID: f.task.ID,
		Status: messaging.InboxStatusUnread,
	}); err != nil {
		t.Fatal(err)
	}

	f.boot(t)

	if rows := f.rows(t, "sprint-planner"); len(rows) != 1 {
		t.Fatalf("%d rows after recovery over a crashed send, want 1", len(rows))
	}
	delivered := f.notified.count(id) > 0 || f.sweep(t)[id]
	if !delivered {
		t.Fatal("the one handoff row was never delivered: not notified by recovery and not swept — " +
			"exactly-once must not mean exactly-zero")
	}
}

// (c) Partial failure: two targets, the second one's notify fails. A retry must
// not duplicate the first and must still deliver the second.
// MU: revert send to a plain insert and sprint-planner gets two rows.
func TestHandoffOnce_PartialFailureRetriesOnlyTheMissingTarget(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner", "sprint-executor")
	f.notified.fail["sprint-executor"] = true

	res := f.approve(t, false)
	if res == nil {
		t.Fatal("no result")
	}
	f.notified.fail["sprint-executor"] = false
	f.boot(t)

	planner := f.rows(t, "sprint-planner")
	executor := f.rows(t, "sprint-executor")
	if len(planner) != 1 || len(executor) != 1 {
		t.Fatalf("rows: planner=%d executor=%d, want 1 and 1", len(planner), len(executor))
	}
	if got := f.notified.count(planner[0].ID); got != 1 {
		t.Errorf("planner notified %d times, want 1 — the retry re-sent a target that had succeeded", got)
	}
	execID := executor[0].ID
	if f.notified.count(execID) == 0 && !f.sweep(t)[execID] {
		t.Error("the executor handoff whose notify failed was never delivered")
	}
}

// (d) Suppression is atomic with the approval and survives a reboot.
// MU: resolve with plain ResolveApprovalRequestByTask when SkipHandoffs and the
// boot fires the handoff (this is 2026-09-23 15:17 in miniature).
func TestHandoffOnce_SuppressedSurvivesReboot(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner")
	f.approve(t, true)

	suppressed, err := f.store.ApprovalHandoffsSuppressed(context.Background(), f.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !suppressed {
		t.Fatal("approval resolved with SkipHandoffs does not record the suppression")
	}

	f.boot(t)
	if rows := f.rows(t, "sprint-planner"); len(rows) != 0 {
		t.Fatalf("%d handoff(s) fired after the operator suppressed them", len(rows))
	}
}

// (f) Suppression binds the GitHub-label producer too, not only recovery.
// MU: remove the suppression check from sendAgentHandoffMessage and this fails.
func TestHandoffOnce_SuppressedBindsTheLabelPath(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner")
	f.approve(t, true)

	source := f.registry.GetAgentByID("design-doc-creator")
	target := f.registry.GetAgentByID("sprint-planner")
	err := sendAgentHandoffMessage(context.Background(), f.store, f.msgs, source, target, f.task, nil, 0)
	if !errors.Is(err, errHandoffSuppressed) {
		t.Errorf("label-path send after suppression returned %v, want errHandoffSuppressed", err)
	}
	if rows := f.rows(t, "sprint-planner"); len(rows) != 0 {
		t.Fatalf("%d handoff(s) written by the label path after suppression", len(rows))
	}
}

// (e) Recovery bounds itself: an approval outside the window fires nothing and
// is resolved as expired, so it is never fetched again.
// MU: drop the expiry mark and the second pass returns the approval again.
func TestHandoffOnce_RecoveryExpiresWhatItWillNotFire(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner")
	ctx := context.Background()
	if err := f.store.ResolveApprovalRequestByTask(ctx, f.task.ID, "approved", "test"); err != nil {
		t.Fatal(err)
	}
	// Age the approval past the window.
	if _, err := f.store.db.ExecContext(ctx, "UPDATE approval_requests SET created_at = ? WHERE task_id = ?",
		time.Now().Add(-HandoffRecoveryWindow-time.Hour), f.task.ID); err != nil {
		t.Fatal(err)
	}

	f.boot(t)
	if rows := f.rows(t, "sprint-planner"); len(rows) != 0 {
		t.Fatalf("recovery fired %d handoff(s) for an approval older than the window", len(rows))
	}
	left, err := f.store.ListApprovedMergeHandoffsWithoutTrigger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("%d out-of-window approval(s) still returned after a pass — the set grows forever", len(left))
	}
}

// An approval that owes nothing is a decided approval: recovery records it
// instead of re-listing it on every boot.
// MU: drop the MarkApprovalHandoffsTriggered in the no-targets branch and this fails.
func TestHandoffOnce_NothingOwedIsRecorded(t *testing.T) {
	f := newOnceFixture(t) // no targets
	ctx := context.Background()
	if err := f.store.ResolveApprovalRequestByTask(ctx, f.task.ID, "approved", "test"); err != nil {
		t.Fatal(err)
	}
	f.boot(t)
	left, err := f.store.ListApprovedMergeHandoffsWithoutTrigger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("an approval that owes no handoff is still listed after recovery (%d) — re-scanned every boot", len(left))
	}
}

// One boot's recovery is bounded: a backlog larger than the batch drains across
// boots rather than all at once.
// MU: drop the LIMIT from the SQLite recovery query and the first boot clears
// everything, failing the "one left" assertion.
func TestHandoffOnce_RecoveryWorkIsBoundedPerBoot(t *testing.T) {
	f := newOnceFixture(t, "sprint-planner")
	ctx := context.Background()
	old := time.Now().Add(-HandoffRecoveryWindow - time.Hour)
	for i := 0; i <= HandoffRecoveryBatch; i++ { // batch + 1 stale approvals
		id := fmt.Sprintf("task-stale-%03d", i)
		if err := f.store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
			ID: ApprovalIDForTask(id), TaskID: id, Type: "merge_handoff", Status: "approved", CreatedAt: old,
		}); err != nil {
			t.Fatal(err)
		}
	}
	f.boot(t)
	left, err := f.store.ListApprovedMergeHandoffsWithoutTrigger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 {
		t.Fatalf("%d undecided approvals after one boot, want 1 — a boot must take at most %d", len(left), HandoffRecoveryBatch)
	}
	f.boot(t)
	if left, _ = f.store.ListApprovedMergeHandoffsWithoutTrigger(ctx); len(left) != 0 {
		t.Errorf("%d still undecided after a second boot — the backlog must drain", len(left))
	}
}
