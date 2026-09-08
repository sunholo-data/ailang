package coordinator

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// M-COMPLETION-PATH-PARITY M0b/C1 — replay safety for the coordinator-side
// primitives and the ledger.
//
// Each arm applies the operation more than once and asserts the second
// application changed nothing, and each is paired with a control showing the OLD
// path really does fail — so "safe to replay" is demonstrated against a live
// counterexample rather than asserted. Three revisions of the design doc asserted
// these properties without checking them; all three assertions were wrong.

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func approvalFixture() *ApprovalRequestRecord {
	return &ApprovalRequestRecord{
		ID:          "apr-88a9fa95",
		TaskID:      "task-88a9fa95",
		Type:        "merge_handoff",
		Description: "Agent completed work on: deterministic lockfile",
		ContextJSON: `{"handoff_targets":["sprint-planner"]}`,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
}

// TestCreateApprovalIfAbsent_ReplayIsANoOp: the approval id is deterministic, so
// a redelivered completion targets the same row.
func TestCreateApprovalIfAbsent_ReplayIsANoOp(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.CreateApprovalIfAbsent(ctx, approvalFixture())
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if !created {
		t.Fatal("first application reported not-created")
	}

	for i := 0; i < 2; i++ {
		created, err = store.CreateApprovalIfAbsent(ctx, approvalFixture())
		if err != nil {
			t.Fatalf("replay %d returned an error; a replay must be a no-op, not a failure: %v", i+1, err)
		}
		if created {
			t.Errorf("replay %d reported created; the first write must win", i+1)
		}
	}
}

// TestCreateApprovalIfAbsent_DoesNotResurrectAResolvedApproval is the property
// that matters most in production.
//
// On Firestore the pre-existing write is Doc(id).Set, which OVERWRITES. A
// redelivered completion would therefore reset an approval a human has already
// approved back to "pending" — silently undoing a decision, and on an
// auto-approved edge, re-releasing a handoff. The SQLite path must hold the same
// contract even though it fails the opposite way.
func TestCreateApprovalIfAbsent_DoesNotResurrectAResolvedApproval(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.CreateApprovalIfAbsent(ctx, approvalFixture()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.ResolveApprovalRequestByTask(ctx, "task-88a9fa95", "approved", "mark"); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// A redelivered completion re-attempts the same approval.
	if _, err := store.CreateApprovalIfAbsent(ctx, approvalFixture()); err != nil {
		t.Fatalf("replay after resolution: %v", err)
	}

	got, err := store.GetApprovalRequest(ctx, "apr-88a9fa95")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "approved" {
		t.Errorf("status = %q after a replayed completion, want \"approved\" — the replay regressed a human decision", got.Status)
	}
}

// TestCreateApprovalRequest_StillFailsOnReplay is the control. If the bare INSERT
// ever stops erroring, the arm above no longer distinguishes anything.
func TestCreateApprovalRequest_StillFailsOnReplay(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.CreateApprovalRequest(ctx, approvalFixture()); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := store.CreateApprovalRequest(ctx, approvalFixture()); err == nil {
		t.Fatal("control failed: the bare INSERT no longer errors on a duplicate id, so CreateApprovalIfAbsent proves nothing distinctive")
	}
}

func TestCreateApprovalIfAbsent_RejectsMissingID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.CreateApprovalIfAbsent(ctx, nil); err == nil {
		t.Error("accepted a nil approval")
	}
	if _, err := store.CreateApprovalIfAbsent(ctx, &ApprovalRequestRecord{TaskID: "task-x"}); err == nil {
		t.Error("accepted an approval with no id — a generated id cannot collide, so the replay guarantee would be silently void")
	}
}

// --- Ledger ---

func TestLedger_ClaimResolveRoundTrip(t *testing.T) {
	now := time.Now()
	l := FinalizationLedger{}

	l = l.Claim(EffectApproval, "coord-1", now)
	if l.IsDone(EffectApproval) {
		t.Error("a claimed effect is not done")
	}
	if l[EffectApproval].Attempt != 1 {
		t.Errorf("attempt = %d, want 1", l[EffectApproval].Attempt)
	}

	l = l.Resolve(EffectApproval, FinalizationDone, now, "")
	if !l.IsDone(EffectApproval) {
		t.Error("a resolved effect is done")
	}

	encoded, err := MarshalLedger(l)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := UnmarshalLedger(encoded)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !back.IsDone(EffectApproval) {
		t.Error("round trip lost the effect state")
	}
}

// TestLedger_ClaimingADoneEffectIsANoOp: a replay that races a finished finalizer
// must not reopen settled work, or the attempt counter would climb forever and
// the effect could be re-applied after it was superseded.
func TestLedger_ClaimingADoneEffectIsANoOp(t *testing.T) {
	now := time.Now()
	l := FinalizationLedger{}.Claim(EffectHandoff, "coord-1", now)
	l = l.Resolve(EffectHandoff, FinalizationDone, now, "")

	l = l.Claim(EffectHandoff, "coord-2", now.Add(time.Minute))

	if l[EffectHandoff].State != FinalizationDone {
		t.Errorf("state = %s after re-claiming a done effect, want done", l[EffectHandoff].State)
	}
	if l[EffectHandoff].Owner != "coord-1" {
		t.Errorf("owner = %q, want coord-1 — a re-claim must not steal a settled effect", l[EffectHandoff].Owner)
	}
}

// TestLedger_SupersededCountsAsDone: an effect that no longer applies because the
// record moved on must not be retried; re-applying it is exactly the regression
// the CAS guard exists to prevent.
func TestLedger_SupersededCountsAsDone(t *testing.T) {
	now := time.Now()
	l := FinalizationLedger{}.Claim(EffectTaskStatus, "coord-1", now)
	l = l.Resolve(EffectTaskStatus, FinalizationSuperseded, now, "")

	if !l.IsDone(EffectTaskStatus) {
		t.Error("superseded must count as done, or a stale replay would regress a record another step advanced")
	}
}

func TestLedger_StaleClaimIsTakeoverEligible(t *testing.T) {
	now := time.Now()
	l := FinalizationLedger{}.Claim(EffectMetrics, "coord-1", now.Add(-StaleFinalizationClaim-time.Minute))

	if !l.IsStale(EffectMetrics, now) {
		t.Error("a claim older than the threshold must be takeover-eligible, or a crashed owner strands it forever")
	}

	fresh := FinalizationLedger{}.Claim(EffectMetrics, "coord-1", now)
	if fresh.IsStale(EffectMetrics, now) {
		t.Error("a fresh claim must not be stolen")
	}

	done := FinalizationLedger{}.Claim(EffectMetrics, "coord-1", now.Add(-time.Hour))
	done = done.Resolve(EffectMetrics, FinalizationDone, now, "")
	if done.IsStale(EffectMetrics, now) {
		t.Error("a done effect is never stale, however old")
	}
}

func TestLedger_AttemptsAccumulateAcrossRetries(t *testing.T) {
	now := time.Now()
	l := FinalizationLedger{}
	for i := 0; i < MaxFinalizationAttempts; i++ {
		l = l.Claim(EffectHandoff, "coord-1", now)
		l = l.Resolve(EffectHandoff, FinalizationPending, now, "transport blip")
	}
	if got := l[EffectHandoff].Attempt; got != MaxFinalizationAttempts {
		t.Errorf("attempt = %d after %d retries, want %d — without this the bound cannot be enforced", got, MaxFinalizationAttempts, MaxFinalizationAttempts)
	}

	l = l.Resolve(EffectHandoff, FinalizationFailed, now, "gave up")
	if !l.IsExhausted(EffectHandoff) {
		t.Error("an exhausted effect must be terminal and visible, never silently skipped")
	}
	if l[EffectHandoff].Error == "" {
		t.Error("a terminal failure must explain itself")
	}
}

// TestLedger_EmptyIsDistinctFromNull: a task that has never been finalised must
// be distinguishable from one whose ledger failed to serialise.
func TestLedger_EmptyIsDistinctFromNull(t *testing.T) {
	encoded, err := MarshalLedger(FinalizationLedger{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if encoded != "" {
		t.Errorf("an empty ledger encodes as %q, want the empty string", encoded)
	}

	back, err := UnmarshalLedger("")
	if err != nil {
		t.Fatalf("an absent ledger must parse as empty, not error: %v", err)
	}
	if len(back) != 0 {
		t.Errorf("empty ledger parsed to %d entries", len(back))
	}
}

// TestLedgerPersistence_SurvivesAStoreRoundTrip guards V44's failure mode on the
// SQLite side: a ledger that does not survive storage would read back empty on
// every redelivery, and finalisation would re-run every effect while believing
// it had never run — with no error anywhere.
func TestLedgerPersistence_SurvivesAStoreRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	now := time.Now()

	task := &TaskRecord{ID: "task-ledger", Title: "ledger round trip", Status: TaskStatusPending}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	empty, err := store.GetTaskFinalization(ctx, "task-ledger")
	if err != nil {
		t.Fatalf("read before any write must succeed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("a task with no ledger returned %d entries", len(empty))
	}

	l := FinalizationLedger{}.Claim(EffectApproval, "coord-1", now)
	l = l.Resolve(EffectApproval, FinalizationDone, now, "")
	if err := store.SetTaskFinalization(ctx, "task-ledger", l); err != nil {
		t.Fatalf("write: %v", err)
	}

	back, err := store.GetTaskFinalization(ctx, "task-ledger")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !back.IsDone(EffectApproval) {
		t.Error("the ledger did not survive the store round trip — every redelivery would re-run every effect")
	}
}

// TestLedgerPersistence_UnknownTaskIsAnError: writing a ledger for a task that
// does not exist would look like progress while recording nothing.
func TestLedgerPersistence_UnknownTaskIsAnError(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.SetTaskFinalization(ctx, "task-does-not-exist", FinalizationLedger{
		EffectApproval: {State: FinalizationDone},
	})
	if err == nil {
		t.Error("writing a ledger for an unknown task silently succeeded")
	}
}
