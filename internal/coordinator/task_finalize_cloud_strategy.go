package coordinator

import (
	"context"

	"github.com/sunholo-data/ailang/internal/pubsub"
)

// CloudStrategy is the Cloud Run Jobs executor's half of finalisation
// (M-COMPLETION-PATH-PARITY M1).
//
// The cloud coordinator has no clone and no worktree — it passes "" for
// worktreePath — so it cannot run `git diff` at any price. The executor is the
// component that can, and already does: it computes changed files from its clone
// point before pushing. M3 extends the completion payload with the two commit
// SHAs and the diff so this can read them; until then DiffSource reports the
// absence explicitly rather than returning an empty diff, because an approval
// card that shows a confident "Files (0)" gets approved blind.
type CloudStrategy struct {
	Completion pubsub.TaskCompletion
}

func (s *CloudStrategy) Kind() StrategyKind { return StrategyKindCloud }

func (s *CloudStrategy) DiffSource(ctx context.Context, task *TaskRecord) (DiffResult, error) {
	// Both SHAs, or nothing. A diff bounded by a branch name is not replay-stable
	// — the branch can move or be deleted between delivery attempts — so a
	// completion missing either SHA cannot honour the contract and says so.
	if s.Completion.BaseCommit == "" || s.Completion.HeadCommit == "" {
		return DiffResult{}, ErrNoDiffSource
	}
	return DiffResult{
		Stat:         s.Completion.DiffStat,
		ChangedFiles: s.Completion.ChangedFiles,
		Patch:        s.Completion.Diff,
	}, nil
}

// completionOutcome maps the executor's reported status onto the finalisation
// outcome. An unrecognised status is reported as such rather than defaulted:
// treating an unknown value as success is what let "the agent was structurally
// prevented from doing anything" report as "completed" for six days.
func completionOutcome(status string) (CompletionOutcome, bool) {
	switch TaskStatus(status) {
	case TaskStatusCompleted:
		return OutcomeCompleted, true
	case TaskStatusNoChanges:
		return OutcomeNoChanges, true
	case TaskStatusFailed:
		return OutcomeFailed, true
	default:
		return "", false
	}
}
