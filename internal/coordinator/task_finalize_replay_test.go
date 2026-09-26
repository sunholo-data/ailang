package coordinator

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// M-COMPLETION-PATH-PARITY M1 / C1 — replay and supersession.
//
// Pub/Sub push is at-least-once with a 60s ack deadline, and the coordinator's
// terminal-state guard does not cover pending_approval, so the same completion
// arrives more than once. These arms assert what the second arrival must NOT do.
//
// Two properties carry the weight, because cross-store atomicity is unavailable
// on either path: every effect is idempotent, and every status write is
// compare-and-set. The ledger records progress on top; nothing depends on it
// being atomic.

// TestFinalize_RedeliveryDispatchesOneHandoff is the failure that matters most
// for unsupervised chaining: a duplicate handoff starts the next agent twice, on
// the same work, in the same repo.
func TestFinalize_RedeliveryDispatchesOneHandoff(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(true))

	for i := 0; i < 3; i++ {
		h.finalize(t, OutcomeCompleted, true)
	}

	if got := h.handoffs(t); len(got) != 1 {
		t.Errorf("sprint-planner received %d handoffs after 3 deliveries, want 1 — the next agent would start %d times on the same work", len(got), len(got))
	}
}

// TestFinalize_RedeliveryCreatesOneApproval: a second approval for one task means
// two cards in the queue for the same decision.
func TestFinalize_RedeliveryCreatesOneApproval(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		h.finalize(t, OutcomeCompleted, false)
	}

	pending, err := h.store.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	count := 0
	for _, a := range pending {
		if a.TaskID == h.task.ID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("%d approvals for one task after 3 deliveries, want 1", count)
	}
}

// TestFinalize_RedeliveryDoesNotDoubleCountCost is the bug the design doc
// asserted away for three revisions: both metrics writers accumulate, so without
// absolute setters a replayed completion inflates the chain's cost.
func TestFinalize_RedeliveryDoesNotDoubleCountCost(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		h.finalize(t, OutcomeCompleted, false)
	}

	stage, err := h.obs.GetStage(ctx, h.stage)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.Cost != 0.42 {
		t.Errorf("stage cost = %v after 3 deliveries, want 0.42", stage.Cost)
	}
	chain, err := h.obs.GetChain(ctx, h.chain, observatory.ChainReadOptions{})
	if err != nil {
		t.Fatalf("get chain: %v", err)
	}
	if chain.TotalCost != 0.42 {
		t.Errorf("chain total_cost = %v after 3 deliveries, want 0.42 — a replayed completion inflated the bill", chain.TotalCost)
	}
	if chain.StagesCompleted > 1 {
		t.Errorf("stages_completed = %d after 3 deliveries of ONE stage, want at most 1", chain.StagesCompleted)
	}
}

// TestFinalize_DoesNotRegressAnApprovedTask is the hole a quorum reviewer found
// in the original design: an absolute status write is repeatable, but not
// idempotent once another step has legitimately advanced the record.
//
// The sequence: finalisation writes pending_approval, a human approves, then the
// redelivery arrives. Without the compare-and-set the replay reverts the task to
// pending_approval — silently undoing the approval, and on an auto-approved edge
// re-releasing the handoff.
func TestFinalize_DoesNotRegressAnApprovedTask(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	ctx := context.Background()

	h.finalize(t, OutcomeCompleted, false)
	if got := h.taskStatus(t); got != TaskStatusPendingApproval {
		t.Fatalf("setup: status = %q, want pending_approval", got)
	}

	// A human approves and the task is merged.
	if _, err := h.store.CompareAndSetTaskStatus(ctx, h.task.ID, []TaskStatus{TaskStatusPendingApproval}, TaskStatusCompleted); err != nil {
		t.Fatalf("approve: %v", err)
	}

	// The redelivery arrives afterwards.
	report := h.finalize(t, OutcomeCompleted, false)

	if got := h.taskStatus(t); got != TaskStatusCompleted {
		t.Errorf("status = %q after a late redelivery, want completed — the replay reverted a human decision", got)
	}
	if !report.Ledger.IsDone(EffectTaskStatus) {
		t.Error("the superseded status effect must be recorded as settled, or the sweep will retry it forever")
	}
}

// TestFinalize_SupersededIsRecordedNotRetried: an effect that no longer applies
// must be terminal in the ledger. If it stayed pending, the ten-minute sweep
// would pick it up again on every pass.
func TestFinalize_SupersededIsRecordedNotRetried(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	ctx := context.Background()

	// The task is cancelled out from under the finalisation.
	if _, err := h.store.CompareAndSetTaskStatus(ctx, h.task.ID, AllTaskStatuses(), TaskStatusCancelled); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	report := h.finalize(t, OutcomeCompleted, false)

	if got := h.taskStatus(t); got != TaskStatusCancelled {
		t.Errorf("status = %q, want cancelled — finalisation overwrote a cancellation", got)
	}
	entry := report.Ledger[EffectTaskStatus]
	if entry.State != FinalizationSuperseded {
		t.Errorf("ledger state = %q, want superseded", entry.State)
	}
	_ = ctx
}

// TestFinalize_LedgerRecordsWhatRan: the report must distinguish "not applicable
// to this outcome" from "should have run and did not". Silence about an effect is
// how the original defect survived four months.
func TestFinalize_LedgerRecordsWhatRan(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(true))
	report := h.finalize(t, OutcomeCompleted, true)

	for _, effect := range []string{EffectTaskStatus, EffectStageStatus, EffectChainStatus, EffectMetrics, EffectHandoff} {
		if !report.Ledger.IsDone(effect) {
			t.Errorf("effect %s did not complete: %+v", effect, report.Ledger[effect])
		}
	}
	// Approval and stage error do not apply to a skip_approval success.
	if report.Ledger.IsDone(EffectApproval) {
		t.Error("an approval was recorded for a skip_approval completion")
	}
	if report.Ledger.IsDone(EffectStageError) {
		t.Error("a stage error was recorded for a successful completion")
	}
	if len(report.Skipped) == 0 {
		t.Error("no effects were reported as skipped; the report cannot distinguish not-applicable from silently-dropped")
	}
}

// TestFinalize_LedgerSurvivesAcrossDeliveries: the second delivery must SEE the
// first one's work. If the ledger did not persist, every redelivery would re-run
// every effect while believing none had run — which is survivable only because
// the effects are idempotent, and is exactly the state the design must not rely on.
func TestFinalize_LedgerSurvivesAcrossDeliveries(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(true))
	ctx := context.Background()

	h.finalize(t, OutcomeCompleted, true)

	stored, err := h.store.GetTaskFinalization(ctx, h.task.ID)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if !stored.IsDone(EffectHandoff) {
		t.Fatal("the handoff effect did not persist to the ledger")
	}

	report := h.finalize(t, OutcomeCompleted, true)
	for _, s := range report.Skipped {
		if s == EffectHandoff+" (already done)" {
			return
		}
	}
	t.Errorf("the second delivery did not recognise the handoff as already done: skipped=%v", report.Skipped)
}

// TestFinalize_UnregisteredHandoffTargetIsLoud: a configured edge pointing at an
// agent that does not exist must not disappear quietly — that is how a pipeline
// stops without anyone noticing.
func TestFinalize_UnregisteredHandoffTargetIsLoud(t *testing.T) {
	agent := &AgentConfig{
		ID:                   "design-doc-creator",
		Inbox:                "design-doc-creator",
		TriggerOnComplete:    []string{"agent-that-does-not-exist"},
		AutoApproveHandoffTo: []string{"agent-that-does-not-exist"},
	}
	h := newFinalizeHarness(t, agent)

	report := h.finalize(t, OutcomeCompleted, true)

	// The effect is attempted (the edge is configured) and dispatches nothing.
	if len(h.handoffs(t)) != 0 {
		t.Error("a handoff was dispatched to an unregistered agent")
	}
	if report.Ledger[EffectHandoff].State == "" {
		t.Error("the handoff effect was not attempted at all; a broken edge must be visible in the ledger")
	}
}

// TestFinalize_WithoutALedgerStoreIsAnError: proceeding with an unknown ledger
// would re-run every effect while believing none had run.
func TestFinalize_RequiresATaskWithAnID(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))

	_, err := FinalizeTaskCompletion(context.Background(), h.deps, FinalizeInput{
		Task:    &TaskRecord{},
		Result:  &ExecuteResult{},
		Outcome: OutcomeCompleted,
	}, nilStrategy{kind: StrategyKindCloud})
	if err == nil {
		t.Error("finalisation accepted a task with no id")
	}
}

var _ = messaging.InboxTypeHandoff
