package coordinator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Reported 2026-09-12 by Daneel. Attempt 2 of a task that had FAILED at git push
// came back `{"status":"deduplicated","error_msg":"Skipped: similar to recent
// task task-f02c283d"}` — task-f02c283d being the failure itself. The deploy key
// under test was never exercised, and no rewording-free path existed to retry.

// The headline case.
func TestFailedTaskDoesNotSuppressARetry(t *testing.T) {
	if DedupSuppresses(TaskStatusFailed) {
		t.Fatal("a failed task must not suppress a retry — the failure IS the reason for the retry")
	}
}

func TestWorkAlreadyInFlightOrDoneStillSuppresses(t *testing.T) {
	// The dedup exists for redeliveries and double-sends; removing it entirely
	// would be the opposite bug.
	for _, s := range []TaskStatus{
		TaskStatusPending, TaskStatusQueued, TaskStatusRunning,
		TaskStatusPendingApproval, TaskStatusCompleted,
	} {
		if !DedupSuppresses(s) {
			t.Errorf("%q is in flight or done — an identical request must still be skipped", s)
		}
	}
	for _, s := range []TaskStatus{
		TaskStatusFailed, TaskStatusRejected, TaskStatusCancelled,
		TaskStatusNoChanges, TaskStatusDuplicate,
	} {
		if DedupSuppresses(s) {
			t.Errorf("%q did not produce the work — a retry must be allowed", s)
		}
	}
}

// An unknown status must not suppress: the cost of one extra run is a task, the
// cost of the other direction is a request that can never be made again.
func TestUnknownStatusDoesNotSuppress(t *testing.T) {
	if DedupSuppresses(TaskStatus("invented_later")) {
		t.Fatal("an unrecognised status must not suppress a request")
	}
}

// Mirrors TestEveryStatusConsumerHandlesEveryStatus: the next status someone
// adds gets a deliberate answer here rather than falling into the zero value.
func TestEveryStatusHasADedupClassification(t *testing.T) {
	for _, s := range AllTaskStatuses() {
		if _, ok := dedupSuppressesByStatus[s]; !ok {
			t.Errorf("status %q has no dedup classification — add it to dedupSuppressesByStatus", s)
		}
	}
}

// The second false claim in that error message: "recent". The match had no time
// bound at all, so an identical request was suppressed by a fingerprint from any
// point in history, forever.
func TestDedupIsBoundedInTime(t *testing.T) {
	now := time.Now()
	since := DedupSince(now)

	fresh := &TaskRecord{Status: TaskStatusCompleted, CreatedAt: now.Add(-time.Hour)}
	if !fresh.BlocksDuplicate(since) {
		t.Error("a task completed an hour ago must still suppress an identical request")
	}

	ancient := &TaskRecord{Status: TaskStatusCompleted, CreatedAt: now.Add(-DedupWindow - time.Minute)}
	if ancient.BlocksDuplicate(since) {
		t.Error("a task older than the window must not suppress — that is what made it permanent")
	}

	// An unknown age is a data defect, not a recent task.
	if (&TaskRecord{Status: TaskStatusRunning}).BlocksDuplicate(since) {
		t.Error("a task with no CreatedAt must not suppress")
	}
	if (*TaskRecord)(nil).BlocksDuplicate(since) {
		t.Error("nil must not suppress")
	}
}

// End to end through the store: the exact shape of the reported incident.
func TestFindDuplicateTask_SkipsFailedPredecessor(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()
	fingerprint := uint64(0xDEADBEEF01)

	failed := &TaskRecord{
		ID:        "task-f02c283d",
		Title:     "attempt 1",
		Content:   "write the document and push",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusFailed,
		CreatedAt: time.Now().Add(-8 * time.Hour),
	}
	if err := store.CreateTask(ctx, failed); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.SetTaskFingerprint(ctx, failed.ID, fingerprint); err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	dup, err := store.FindDuplicateTask(ctx, fingerprint, DedupSince(time.Now()))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if dup != nil {
		t.Fatalf("a failed predecessor must not block the retry, got %s (%s)", dup.ID, dup.Status)
	}

	// ...but the same content while attempt 2 is actually running still does.
	running := &TaskRecord{
		ID:        "task-90eb14f8",
		Title:     "attempt 2",
		Content:   "write the document and push",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusRunning,
		CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, running); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.SetTaskFingerprint(ctx, running.ID, fingerprint); err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	dup, err = store.FindDuplicateTask(ctx, fingerprint, DedupSince(time.Now()))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if dup == nil || dup.ID != running.ID {
		t.Fatalf("work in flight must still suppress a third copy, got %v", dup)
	}
}

func TestFindDuplicateTask_IgnoresTasksOlderThanTheWindow(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()
	fingerprint := uint64(0xDEADBEEF02)

	old := &TaskRecord{
		ID:        "task-ancient",
		Title:     "the same request, last month",
		Content:   "publish the weekly note",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusCompleted,
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
	}
	if err := store.CreateTask(ctx, old); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.SetTaskFingerprint(ctx, old.ID, fingerprint); err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	dup, err := store.FindDuplicateTask(ctx, fingerprint, DedupSince(time.Now()))
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if dup != nil {
		t.Fatalf("a month-old task must not suppress today's identical request, got %s", dup.ID)
	}
}

// The second half of the report: the deduplicated completion carried no
// correlation_id, so the requester could not match it to its request and waited
// through three polls for a reply it had already been sent.
func TestDedupCompletionCarriesCorrelationID(t *testing.T) {
	// No registry entry for the agent: InboxForAgent is nil-safe and the inbox
	// falls back to the agent id, which is what this path already does.
	d := newTestDaemonWithMsgStore(t)

	const messageID = "inbox_1757700000000_90eb14f8"
	dup := &TaskRecord{ID: "task-f02c283d", Status: TaskStatusRunning}
	d.publishDedupCompletion("task-90eb14f8", "daneel-writer", messageID, dup)

	msgs, err := d.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "daneel-writer"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected exactly one completion, got %d", len(msgs))
	}
	got := msgs[0]

	if got.CorrelationID != messageID {
		t.Errorf("correlation_id = %q, want the originating message %q — without it the requester cannot attribute the reply", got.CorrelationID, messageID)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(got.Payload), &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if payload["correlation_id"] != messageID {
		t.Errorf("payload correlation_id = %v, want %q", payload["correlation_id"], messageID)
	}
	if payload["original_task_status"] != string(TaskStatusRunning) {
		t.Errorf("payload must say WHAT suppressed this, got %v", payload["original_task_status"])
	}
}
