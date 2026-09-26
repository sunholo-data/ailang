package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A replay and a re-run arrive at the same approval row. Confusing them stranded
// eleven tasks on 2026-09-15: rejected, re-dispatched, ran again, and each
// finished `pending_approval` behind a `rejected` record — invisible to
// `coordinator approvals` and refused by approve.
func TestClassifyApprovalCollision(t *testing.T) {
	rejected := func(workID string) *ApprovalRequestRecord {
		ctx := ""
		if workID != "" {
			ctx = `{"work_id":"` + workID + `"}`
		}
		return &ApprovalRequestRecord{ID: "apr-x", TaskID: "task-x", Status: "rejected", ContextJSON: ctx}
	}

	for _, tc := range []struct {
		name     string
		existing *ApprovalRequestRecord
		newWork  string
		want     ApprovalCollision
		why      string
	}{
		{"same work, already rejected", rejected("aaa"), "aaa", CollisionReplay,
			"a redelivered completion must never reopen a decision"},
		{"different work after rejection", rejected("aaa"), "bbb", CollisionNewWork,
			"the standing decision was about work that no longer exists"},
		// A PENDING row was read as "about to be judged anyway, so nothing to
		// do". The card IS what it is judged on, and a re-run replaces the work
		// without replacing the card — measured on task-c0ca6301, where the
		// operator approved a diff naming a file the merge never touched.
		{"pending, same work", &ApprovalRequestRecord{Status: "pending", ContextJSON: `{"work_id":"aaa"}`}, "aaa", CollisionReplay,
			"a redelivered completion must not churn the card"},
		{"pending, DIFFERENT work", &ApprovalRequestRecord{Status: "pending", ContextJSON: `{"work_id":"aaa"}`}, "bbb", CollisionStaleCard,
			"the pending card describes an execution that has been superseded"},
		{"pending, no work id on the record", &ApprovalRequestRecord{Status: "pending"}, "bbb", CollisionReplay,
			"no evidence: rewriting a card on a guess is the worse error"},
		{"pending, no diff source this run", &ApprovalRequestRecord{Status: "pending", ContextJSON: `{"work_id":"aaa"}`}, "", CollisionReplay,
			"a completion with no diff cannot improve the card"},
		// Every approval written before 2026-09-15 has no work id. Reading that
		// as "different" would reopen decisions on every replay.
		{"old record, no work id", rejected(""), "bbb", CollisionUnknown,
			"no evidence must not disturb a recorded decision"},
		{"no diff source this run", rejected("aaa"), "", CollisionUnknown,
			"two no-diff completions are not evidence of the same work"},
		{"no record at all", nil, "bbb", CollisionUnknown, ""},
	} {
		if got := ClassifyApprovalCollision(tc.existing, tc.newWork); got != tc.want {
			t.Errorf("%s: got %v want %v — %s", tc.name, got, tc.want, tc.why)
		}
	}
}

// The work id answers "is this the same change?", not "is this the same
// commit?", so it must survive a re-run that produces an identical diff and
// must not collapse two different ones.
func TestWorkIDForApproval(t *testing.T) {
	a := WorkIDForApproval([]string{"b.go", "a.go"}, "2 files changed")
	b := WorkIDForApproval([]string{"a.go", "b.go"}, "2 files changed")
	if a != b {
		t.Error("file ORDER varies between runs; the set does not")
	}
	if WorkIDForApproval([]string{"a.go"}, "1 file changed") == a {
		t.Error("a different change must get a different id")
	}
	if WorkIDForApproval([]string{"a.go", "b.go"}, "2 files changed, 9 insertions") == a {
		t.Error("the diffstat is part of the identity")
	}
	// No diff source: unknown, NOT a value that compares equal to another unknown.
	if WorkIDForApproval(nil, "") != "" {
		t.Error("no evidence must yield no id")
	}
	if WorkIDForApproval(nil, "  ") != "" {
		t.Error("whitespace is not a diffstat")
	}
}

// The store primitive: reopen only a RESOLVED row, and say whether it moved one.
func TestReopenApprovalForNewWork(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	mk := func(id, status string) {
		if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
			ID: ApprovalIDForTask(id), TaskID: id, Type: string(ApprovalTypeMerge),
			Status: status, ContextJSON: `{"work_id":"old"}`, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	mk("task-rejected", "rejected")
	ok, err := store.ReopenApprovalForNewWork(ctx, "task-rejected", "new work", `{"work_id":"new"}`)
	if err != nil || !ok {
		t.Fatalf("a rejected approval must reopen: ok=%v err=%v", ok, err)
	}
	got, _ := store.GetApprovalRequestByTaskAnyStatus(ctx, "task-rejected")
	if got == nil || got.Status != "pending" {
		t.Fatalf("status = %v, want pending", got)
	}
	if got.ContextJSON != `{"work_id":"new"}` {
		t.Errorf("a reopened approval still describing the OLD change is worse than none: %q", got.ContextJSON)
	}

	// A pending row has nothing to reopen.
	mk("task-pending", "pending")
	if ok, err := store.ReopenApprovalForNewWork(ctx, "task-pending", "x", "{}"); err != nil || ok {
		t.Errorf("pending must not be reopened: ok=%v err=%v", ok, err)
	}
	// A task with no approval at all.
	if ok, err := store.ReopenApprovalForNewWork(ctx, "task-absent", "x", "{}"); err != nil || ok {
		t.Errorf("absent must report false, not error: ok=%v err=%v", ok, err)
	}
	if _, err := store.ReopenApprovalForNewWork(ctx, "", "x", "{}"); err == nil {
		t.Error("an empty task id must be refused")
	}
}

// The production sequence, end to end: run, reject, re-run with NEW work.
//
// Before this fix the second finalisation reported Superseded and the task
// settled into pending_approval behind a rejected record — unapprovable and
// invisible. Eleven tasks did exactly that on 2026-09-15.
func TestFinalize_RerunAfterRejectionReopensTheApproval(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "c.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{ID: "triage", Inbox: "triage", Workspace: dir}); err != nil {
		t.Fatalf("register: %v", err)
	}
	task := &TaskRecord{
		ID: "task-rerun01", AgentID: "triage", Title: "a report", Content: "triage this",
		Type: TaskTypeBugFix, Status: TaskStatusPending, Workspace: dir, CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	deps := &FinalizeDeps{TaskStore: store, AgentRegistry: reg, Owner: "t"}
	in := FinalizeInput{Task: task, Result: &ExecuteResult{Success: true}, Outcome: OutcomeCompleted}

	// Run 1 -> an approval exists, pending.
	if _, err := FinalizeTaskCompletion(ctx, deps, in, fakeDiff{files: []string{"a.md"}, stat: "1 file changed"}); err != nil {
		t.Fatalf("finalize 1: %v", err)
	}
	// A human rejects it.
	if err := store.ResolveApprovalRequestByTask(ctx, task.ID, "rejected", "mark"); err != nil {
		t.Fatalf("reject: %v", err)
	}

	// Run 2 -> RE-DISPATCHED (which is what clears the ledger) and this time it
	// produced DIFFERENT work.
	if err := requeueForRerun(t, store, ctx, task.ID); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if _, err := FinalizeTaskCompletion(ctx, deps, in, fakeDiff{files: []string{"a.md", "b.md"}, stat: "2 files changed"}); err != nil {
		t.Fatalf("finalize 2: %v", err)
	}
	got, err := store.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
	if err != nil || got == nil {
		t.Fatalf("read back: %v", err)
	}
	if got.Status != "pending" {
		t.Fatalf("approval is %q after a re-run with new work — the task is unapprovable and invisible to `coordinator approvals`", got.Status)
	}

	// ...and a REPLAY of the same work must NOT reopen a decision. A replay is a
	// redelivered completion, NOT a re-dispatch, so the ledger is untouched —
	// which is the first line of defence. Clear it anyway, so this asserts the
	// work-id check rather than the ledger.
	if err := store.ResolveApprovalRequestByTask(ctx, task.ID, "rejected", "mark"); err != nil {
		t.Fatalf("reject again: %v", err)
	}
	if err := requeueForRerun(t, store, ctx, task.ID); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if _, err := FinalizeTaskCompletion(ctx, deps, in, fakeDiff{files: []string{"a.md", "b.md"}, stat: "2 files changed"}); err != nil {
		t.Fatalf("finalize 3: %v", err)
	}
	got, _ = store.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
	if got.Status != "rejected" {
		t.Errorf("a replay reopened a human's decision (status=%q) — the opposite mistake, and the worse one", got.Status)
	}
}

// fakeDiff is an ExecutionStrategy that reports a fixed diff.
type fakeDiff struct {
	files []string
	stat  string
}

func (f fakeDiff) Kind() StrategyKind { return StrategyKind(0) }
func (f fakeDiff) DiffSource(context.Context, *TaskRecord) (DiffResult, error) {
	return DiffResult{ChangedFiles: f.files, Stat: f.stat, Patch: "patch"}, nil
}

// The store primitive: refresh only a PENDING row, and say whether it moved one.
func TestRefreshPendingApproval(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	mk := func(id, status string) {
		if err := store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
			ID: ApprovalIDForTask(id), TaskID: id, Type: string(ApprovalTypeMerge),
			Status: status, Description: "old", ContextJSON: `{"work_id":"old"}`, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}

	mk("task-pending", "pending")
	ok, err := store.RefreshPendingApproval(ctx, "task-pending", "new work", `{"work_id":"new"}`)
	if err != nil || !ok {
		t.Fatalf("a pending approval must refresh: ok=%v err=%v", ok, err)
	}
	got, _ := store.GetApprovalRequestByTaskAnyStatus(ctx, "task-pending")
	if got == nil || got.ContextJSON != `{"work_id":"new"}` || got.Description != "new work" {
		t.Fatalf("card still describes the old run: %+v", got)
	}
	if got.Status != "pending" {
		t.Errorf("refreshing must not change the status, got %q", got.Status)
	}

	// A RESOLVED row is not this function's business: overwriting it would edit
	// the evidence behind a decision someone already made.
	mk("task-approved", "approved")
	if ok, err := store.RefreshPendingApproval(ctx, "task-approved", "x", `{"work_id":"x"}`); err != nil || ok {
		t.Errorf("a resolved approval must not be refreshed: ok=%v err=%v", ok, err)
	}
	if got, _ := store.GetApprovalRequestByTaskAnyStatus(ctx, "task-approved"); got.ContextJSON != `{"work_id":"old"}` {
		t.Error("a resolved approval's evidence was rewritten")
	}
	if ok, err := store.RefreshPendingApproval(ctx, "task-absent", "x", "{}"); err != nil || ok {
		t.Errorf("absent must report false, not error: ok=%v err=%v", ok, err)
	}
	if _, err := store.RefreshPendingApproval(ctx, "", "x", "{}"); err == nil {
		t.Error("an empty task id must be refused")
	}
}

// The measured sequence: a task re-runs while its approval is still PENDING, and
// the card must end up describing the work that would actually merge.
//
// task-c0ca6301, 2026-09-15: run 1 at 06:44 wrote the shared backlog file and
// created the approval; the task was re-dispatched at 07:14 and run 2 wrote a
// different file onto the same branch. The card kept run 1's diff, and the
// approval three hours later merged a file the card never named.
func TestFinalize_RerunWhilePendingRefreshesTheCard(t *testing.T) {
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "c.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{ID: "triage", Inbox: "triage", Workspace: dir}); err != nil {
		t.Fatalf("register: %v", err)
	}
	task := &TaskRecord{
		ID: "task-stale01", AgentID: "triage", Title: "a report", Content: "triage this",
		Type: TaskTypeBugFix, Status: TaskStatusPending, Workspace: dir, CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	deps := &FinalizeDeps{TaskStore: store, AgentRegistry: reg, Owner: "t"}
	in := FinalizeInput{Task: task, Result: &ExecuteResult{Success: true}, Outcome: OutcomeCompleted}

	// Run 1: the backlog file.
	if _, err := FinalizeTaskCompletion(ctx, deps, in, fakeDiff{files: []string{"backlog.md"}, stat: "1 file changed, 7 insertions"}); err != nil {
		t.Fatalf("finalize 1: %v", err)
	}
	// Re-dispatched, no decision taken in between.
	if err := requeueForRerun(t, store, ctx, task.ID); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	// Run 2: a DIFFERENT file, which is what the branch now carries.
	if _, err := FinalizeTaskCompletion(ctx, deps, in, fakeDiff{files: []string{"triage/row.md"}, stat: "1 file changed, 11 insertions"}); err != nil {
		t.Fatalf("finalize 2: %v", err)
	}

	got, err := store.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
	if err != nil || got == nil {
		t.Fatalf("read back: %v", err)
	}
	if got.Status != "pending" {
		t.Fatalf("status = %q, want pending: refreshing a card must not resolve or reopen anything", got.Status)
	}
	if !strings.Contains(got.ContextJSON, "triage/row.md") {
		t.Errorf("the card does not name the file that would merge: %s", got.ContextJSON)
	}
	if strings.Contains(got.ContextJSON, "backlog.md") {
		t.Errorf("the card still names run 1's file — this is the approve-A-merge-B bug: %s", got.ContextJSON)
	}

	// A REPLAY of run 2 must leave the card exactly as it is.
	before := got.ContextJSON
	if err := requeueForRerun(t, store, ctx, task.ID); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	if _, err := FinalizeTaskCompletion(ctx, deps, in, fakeDiff{files: []string{"triage/row.md"}, stat: "1 file changed, 11 insertions"}); err != nil {
		t.Fatalf("finalize 3: %v", err)
	}
	got, _ = store.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
	if got.ContextJSON != before {
		t.Errorf("a replay rewrote the card: %s", got.ContextJSON)
	}
}

// requeueForRerun puts a task back through the dispatch path the way production
// does: ResetTaskToPending, then the CLAIM. MarkTaskQueued alone used to stand
// in for this, which worked only because the claim could not refuse — it now
// rejects anything that is not pending, which is the whole point of it.
func requeueForRerun(t *testing.T, store *SQLiteStore, ctx context.Context, taskID string) error {
	t.Helper()
	if err := store.ResetTaskToPending(ctx, taskID); err != nil {
		return err
	}
	return store.MarkTaskQueued(ctx, taskID)
}
