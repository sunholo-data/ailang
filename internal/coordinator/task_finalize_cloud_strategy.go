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
	// M3 will populate this from BaseCommit/HeadCommit/DiffStat/Diff on the
	// completion payload. The executor's ChangedFiles is already carried, so a
	// partial answer is better than none: it names the files even before the
	// patch is available.
	if len(s.Completion.ChangedFiles) > 0 {
		return DiffResult{ChangedFiles: s.Completion.ChangedFiles}, nil
	}
	return DiffResult{}, ErrNoDiffSource
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
