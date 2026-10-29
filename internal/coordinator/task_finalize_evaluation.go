package coordinator

import (
	"context"
	"fmt"
)

// The evaluation effect: an EvaluatesParent agent's verdict lands on its
// parent's approval (M-PIPELINE-RECONCILIATION M2), on the cloud path too.
//
// Only the daemon path called attachEvaluationToParent, so no cloud evaluator
// verdict ever reached an approval. Measured 2026-10-27: none of the nine
// pending prod approvals carried an evaluation, including task-002d8bc0, which
// the evaluator scored PASS 71 on 2026-10-12. A dead evaluator is a verdict too
// (UNAVAILABLE), so this runs for every outcome, never only on success.

func (f *finalizer) wantsEvaluation() bool {
	if f.in.Task.ShadowOf != "" || f.deps.AgentRegistry == nil {
		return false
	}
	agent := f.deps.AgentRegistry.GetAgentByID(f.in.Task.AgentID)
	return agent != nil && agent.EvaluatesParent
}

func (f *finalizer) applyEvaluation(ctx context.Context) (FinalizationState, error) {
	if f.in.Task.ParentTaskID == "" {
		// A wiring mistake, not a normal condition: the handoff failed to carry
		// parent_task_id. Failing the effect records it on the ledger.
		return FinalizationFailed, fmt.Errorf("evaluator task %s has no parent_task_id — its verdict cannot reach any approval", f.in.Task.ID)
	}
	verdict := f.evaluationVerdict()
	if err := f.deps.TaskStore.UpdateApprovalEvaluationByTask(ctx, f.in.Task.ParentTaskID, verdict.String()); err != nil {
		return FinalizationPending, fmt.Errorf("attach evaluation %q to parent %s: %w", verdict.String(), f.in.Task.ParentTaskID, err)
	}
	f.deps.logf("Evaluation attached to parent %s's approval: %s (from evaluator task %s)",
		f.in.Task.ParentTaskID, verdict.String(), f.in.Task.ID)
	return FinalizationDone, nil
}

func (f *finalizer) evaluationVerdict() EvaluationVerdict {
	var output, errMsg string
	if f.in.Result != nil {
		output, errMsg = f.in.Result.Output, f.in.Result.Error
	}
	switch f.in.Outcome {
	case OutcomeFailed, OutcomeBlocked:
		if errMsg == "" {
			errMsg = string(f.in.Outcome)
		}
		return UnavailableVerdict(fmt.Sprintf("evaluator task %s %s: %s", f.in.Task.ID, f.in.Outcome, truncateVerdict(errMsg, 300)))
	default:
		// An evaluator only reads, so completed and no_changes both carry a verdict.
		return ExtractEvaluationVerdict(output)
	}
}
