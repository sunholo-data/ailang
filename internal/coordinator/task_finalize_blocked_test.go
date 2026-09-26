package coordinator

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// M-TASK-STATUS-TRUTH S2 — `blocked` is an outcome, not an unknown string.
//
// The executor learned to say "I could not start, and why" on 2026-09-14
// (coordinator_cloud.go, task_blocked.go). The consumer never learned to hear
// it: completionOutcome had no case, the handler logged "unknown completion
// status" and returned, and the task sat `queued` until the stale detector
// reported it as a 3h TIMEOUT. Measured in prod 2026-09-23 on task-91337be6
// ("Sprint JSON failed the sprint-executor mandatory validation gate") — a
// precise, actionable reason, delivered, and then reported as the opposite.

// MU: remove the `case TaskStatusBlocked` from completionOutcome and this fails.
func TestCompletionOutcomeAcceptsEveryExecutorStatus(t *testing.T) {
	for _, st := range ExecutorCompletionStatuses() {
		if _, ok := completionOutcome(string(st)); !ok {
			t.Errorf("the executor may publish %q but completionOutcome rejects it — the completion "+
				"is dropped and the task is later reported as timed out", st)
		}
	}
}

// The contract list must name what the producer actually emits, or the test
// above guards nothing.
func TestExecutorCompletionStatusesIncludesBlocked(t *testing.T) {
	want := map[TaskStatus]bool{TaskStatusCompleted: true, TaskStatusNoChanges: true, TaskStatusFailed: true, TaskStatusBlocked: true}
	got := map[TaskStatus]bool{}
	for _, st := range ExecutorCompletionStatuses() {
		got[st] = true
	}
	for st := range want {
		if !got[st] {
			t.Errorf("ExecutorCompletionStatuses omits %q, which coordinator_cloud.go publishes", st)
		}
	}
}

// MU: map OutcomeBlocked to TaskStatusFailed in nextTaskStatus and this fails.
func TestFinalize_BlockedRow(t *testing.T) {
	// An auto-approved edge, so a handoff WOULD fire on success: blocked must
	// not fire it — the next stage would build on work nobody did.
	h := newFinalizeHarness(t, handoffAgent(true))
	h.finalize(t, OutcomeBlocked, false)
	ctx := context.Background()

	if got := h.taskStatus(t); got != TaskStatusBlocked {
		t.Errorf("task status = %q, want %q", got, TaskStatusBlocked)
	}
	if h.approval(t) != nil {
		t.Error("an approval card was created for work the agent did not attempt")
	}
	if got := len(h.handoffs(t)); got != 0 {
		t.Errorf("%d handoff(s) dispatched from a blocked task", got)
	}

	stage, err := h.obs.GetStage(ctx, h.stage)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if stage.Status != observatory.StageStatusFailed {
		t.Errorf("stage status = %q, want %q", stage.Status, observatory.StageStatusFailed)
	}
	if stage.ErrorMessage == "" {
		t.Error("the blocked reason was not recorded on the stage — it is the one thing a human needs")
	}
	chain, err := h.obs.GetChain(ctx, h.chain, observatory.ChainReadOptions{})
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	if chain.Status != observatory.ChainStatusFailed {
		t.Errorf("chain status = %q, want %q (a blocked chain must not read active or completed)", chain.Status, observatory.ChainStatusFailed)
	}
}
