package coordinator

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// The cloud path must land an evaluator's verdict on its parent's approval, as
// the daemon path does. It never did: on 2026-10-27 no pending prod approval
// carried an evaluation, including one the evaluator had scored PASS 71.
func TestFinalize_EvaluatorVerdictReachesParentApproval(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outcome CompletionOutcome
		output  string
		errMsg  string
		want    string
	}{
		{"verdict line", OutcomeNoChanges, "review done\nEVALUATION_VERDICT: FAIL score=84 reasons=CI red", "", "FAIL score=84 reasons=CI red"},
		{"no marker", OutcomeNoChanges, "EVALUATION_RESULT: fail\nEVALUATION_SCORE: 84/100", "", "UNAVAILABLE"},
		{"evaluator failed", OutcomeFailed, "", "claude task failed: exit status 1", "UNAVAILABLE reason=evaluator task task-matrix failed: claude task failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newFinalizeHarness(t, &AgentConfig{ID: "design-doc-creator", Inbox: "design-doc-creator", EvaluatesParent: true})
			ctx := context.Background()
			if err := h.store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{ID: "apr-parent", TaskID: "task-parent", Type: "merge", Status: "pending"}); err != nil {
				t.Fatal(err)
			}
			h.task.ParentTaskID = "task-parent"

			report, err := FinalizeTaskCompletion(ctx, h.deps, FinalizeInput{
				Task:         h.task,
				Result:       &ExecuteResult{Output: tc.output, Error: tc.errMsg, SessionID: "s"},
				Outcome:      tc.outcome,
				SkipApproval: true,
			}, nilStrategy{kind: StrategyKindCloud})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(report.Applied, EffectEvaluation) {
				t.Fatalf("evaluation not applied: applied=%v skipped=%v", report.Applied, report.Skipped)
			}
			got, err := h.store.GetApprovalRequest(ctx, "apr-parent")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(got.Evaluation, tc.want) {
				t.Errorf("evaluation = %q, want prefix %q", got.Evaluation, tc.want)
			}
		})
	}
}

// Without a parent the verdict has nowhere to go; that must be recorded, not
// passed over.
func TestFinalize_EvaluatorWithoutParentIsNotApplied(t *testing.T) {
	h := newFinalizeHarness(t, &AgentConfig{ID: "design-doc-creator", Inbox: "design-doc-creator", EvaluatesParent: true})
	report, err := FinalizeTaskCompletion(context.Background(), h.deps, FinalizeInput{
		Task: h.task, Result: &ExecuteResult{Output: "EVALUATION_VERDICT: PASS score=90"}, Outcome: OutcomeNoChanges, SkipApproval: true,
	}, nilStrategy{kind: StrategyKindCloud})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(report.Applied, EffectEvaluation) {
		t.Fatalf("evaluation applied with no parent: %v", report.Applied)
	}
	if e, ok := report.Ledger[EffectEvaluation]; !ok || !strings.Contains(e.Error, "no parent_task_id") {
		t.Errorf("ledger entry = %+v, want the missing-parent error", e)
	}
}

// A non-evaluator agent never touches approvals' evaluation field.
func TestFinalize_NonEvaluatorSkipsEvaluation(t *testing.T) {
	h := newFinalizeHarness(t, &AgentConfig{ID: "design-doc-creator", Inbox: "design-doc-creator"})
	report := h.finalize(t, OutcomeCompleted, false)
	if slices.Contains(report.Applied, EffectEvaluation) {
		t.Fatalf("evaluation applied for a non-evaluator: %v", report.Applied)
	}
}
