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
	if !fresh.BlocksDuplicate(DedupScope{Since: since}) {
		t.Error("a task completed an hour ago must still suppress an identical request")
	}

	ancient := &TaskRecord{Status: TaskStatusCompleted, CreatedAt: now.Add(-DedupWindow - time.Minute)}
	if ancient.BlocksDuplicate(DedupScope{Since: since}) {
		t.Error("a task older than the window must not suppress — that is what made it permanent")
	}

	// An unknown age is a data defect, not a recent task.
	if (&TaskRecord{Status: TaskStatusRunning}).BlocksDuplicate(DedupScope{Since: since}) {
		t.Error("a task with no CreatedAt must not suppress")
	}
	if (*TaskRecord)(nil).BlocksDuplicate(DedupScope{Since: since}) {
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

	dup, err := store.FindDuplicateTask(ctx, fingerprint, DedupScope{Since: DedupSince(time.Now())})
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

	dup, err = store.FindDuplicateTask(ctx, fingerprint, DedupScope{Since: DedupSince(time.Now())})
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

	dup, err := store.FindDuplicateTask(ctx, fingerprint, DedupScope{Since: DedupSince(time.Now())})
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

// The distinction that has already cost us once: DISCOVERY reads the effective
// patterns (widest possible, so evidence omits nothing), and any SAFETY decision
// reads the declared ones (empty means refuse).
//
// Measured 2026-09-11: the auto-merge scope guard read the effective list, so an
// agent with auto_merge and no declared patterns got `**/*` — the opposite of a
// bound. This pins both halves so the next caller cannot quietly pick the wrong
// accessor.
func TestArtifactPatterns_EffectiveIsForDiscovery_DeclaredIsForSafety(t *testing.T) {
	undeclared := &AgentConfig{ID: "some-new-agent", AutoMerge: true}

	if got := undeclared.GetEffectiveArtifactPatterns(); len(got) != 1 || got[0] != "**/*" {
		t.Errorf("discovery must default to the WIDEST pattern so evidence omits nothing, got %v", got)
	}
	if len(undeclared.ArtifactPatterns) != 0 {
		t.Error("the declared list must stay empty — that is what a safety decision reads as refuse")
	}

	// A declared agent gets its own list from both, and they agree.
	declared := &AgentConfig{ID: "daneel-writer", ArtifactPatterns: []string{"documents/**/*.md"}}
	if got := declared.GetEffectiveArtifactPatterns(); len(got) != 1 || got[0] != "documents/**/*.md" {
		t.Errorf("a declared list must win over the default, got %v", got)
	}
}

// TestBlocksDuplicate_HandoffIsNotADuplicateOfItsParent pins the bug that kept
// the pipeline from EVER running.
//
// Measured in production 2026-09-14, on the first handoff that ever fired:
// design-doc-creator completed task-08032ebc, the approval dispatched
// sprint-planner, and the new task was suppressed one second later against
// task-08032ebc itself. sendAgentHandoffMessage embeds the predecessor's
// request verbatim, so a handoff always simhashes to its parent, and a parent
// at handoff time is always `completed` — which suppresses.
func TestBlocksDuplicate_HandoffIsNotADuplicateOfItsParent(t *testing.T) {
	now := time.Now()
	parent := &TaskRecord{
		ID:        "task-08032ebc",
		AgentID:   "design-doc-creator",
		Status:    TaskStatusCompleted,
		CreatedAt: now.Add(-7 * time.Minute),
	}

	// The handoff, as the daemon builds it: same content, next agent, parent set.
	handoff := DedupScope{
		Since:        DedupSince(now),
		AgentID:      "sprint-planner",
		ParentTaskID: "task-08032ebc",
	}
	if parent.BlocksDuplicate(handoff) {
		t.Error("a handoff was suppressed by the stage it follows — the pipeline cannot advance")
	}

	// Either guard alone must be enough: a same-agent re-entry naming the parent…
	sameAgent := DedupScope{Since: DedupSince(now), AgentID: "design-doc-creator", ParentTaskID: "task-08032ebc"}
	if parent.BlocksDuplicate(sameAgent) {
		t.Error("a task must never be suppressed by its own declared parent")
	}
	// …and a different agent with no parent link.
	otherAgent := DedupScope{Since: DedupSince(now), AgentID: "sprint-planner"}
	if parent.BlocksDuplicate(otherAgent) {
		t.Error("the same words sent to a DIFFERENT agent are different work, not a repeat")
	}
}

// The guards must not open a hole in the thing dedup is actually for: the same
// request, to the same agent, arriving twice.
func TestBlocksDuplicate_GenuineRepeatToTheSameAgentStillSuppressed(t *testing.T) {
	now := time.Now()
	existing := &TaskRecord{
		ID: "task-aaa", AgentID: "pkg-sunholo-email",
		Status: TaskStatusRunning, CreatedAt: now.Add(-2 * time.Minute),
	}
	scope := DedupScope{Since: DedupSince(now), AgentID: "pkg-sunholo-email"}
	if !existing.BlocksDuplicate(scope) {
		t.Error("a redelivery of the same request to the same agent must still be suppressed")
	}
}

// An unknown agent on either side is UNKNOWN, not "matches anything". Guessing
// would silently change which requests are suppressed, in a direction nobody
// chose — older tasks predate agent_id being recorded.
func TestBlocksDuplicate_MissingAgentDoesNotDecide(t *testing.T) {
	now := time.Now()
	legacy := &TaskRecord{ID: "task-old", Status: TaskStatusCompleted, CreatedAt: now.Add(-time.Hour)}
	if !legacy.BlocksDuplicate(DedupScope{Since: DedupSince(now), AgentID: "whoever"}) {
		t.Error("an existing task with no agent must keep its pre-scoping behaviour")
	}
	current := &TaskRecord{ID: "task-new", AgentID: "someone", Status: TaskStatusCompleted, CreatedAt: now.Add(-time.Hour)}
	if !current.BlocksDuplicate(DedupScope{Since: DedupSince(now)}) {
		t.Error("an incoming request with no agent must keep its pre-scoping behaviour")
	}
}

func TestDedupScopeFor_CarriesAgentAndParent(t *testing.T) {
	s := DedupScopeFor(&TaskRecord{AgentID: "sprint-planner", ParentTaskID: "task-parent"}, time.Now())
	if s.AgentID != "sprint-planner" || s.ParentTaskID != "task-parent" {
		t.Errorf("scope lost the fields that decide suppression: %+v", s)
	}
	if s.Since.IsZero() {
		t.Error("scope must always carry the window")
	}
	if got := DedupScopeFor(nil, time.Now()); got.Since.IsZero() {
		t.Error("a nil task must still produce a usable window, not an unbounded one")
	}
}
