package coordinator

// Deciding what should happen to the pull request a task left behind.
//
// Kept pure and separate from the GitHub calls, because the interesting part is
// the POLICY and the policy is where this can do damage. Merging a PR is the
// one irreversible thing in this file.

import (
	"fmt"
	"strings"
)

// PRVerdict is what a task's status says should happen to its PR.
type PRVerdict string

const (
	PRMerge  PRVerdict = "merge"  // the work was approved: land it
	PRClose  PRVerdict = "close"  // the branch will never merge
	PRLeave  PRVerdict = "leave"  // still undecided, or not ours to judge
	PRRefuse PRVerdict = "refuse" // approved, but outside the agent's declared scope
)

// PRDecision is a verdict with the reason, which is never optional: an
// unexplained merge or close is exactly as untrustworthy as an abandoned PR.
type PRDecision struct {
	Verdict PRVerdict
	Reason  string
}

// prStatusVerdict maps a task status to what its PR deserves.
//
// A table rather than a condition, for the same reason as the dedup policy: the
// exhaustiveness test then catches the next status someone adds, instead of it
// defaulting into a silent answer. And the silent answer here would be "leave",
// which is how 22 PRs accumulated.
var prStatusVerdict = map[TaskStatus]PRVerdict{
	TaskStatusCompleted:       PRMerge,
	TaskStatusRejected:        PRClose,
	TaskStatusFailed:          PRClose,
	TaskStatusCancelled:       PRClose,
	TaskStatusDuplicate:       PRClose,
	TaskStatusNoChanges:       PRClose, // produced nothing; any branch is empty
	TaskStatusPendingApproval: PRLeave, // the decision has not been made yet
	TaskStatusPending:         PRLeave,
	TaskStatusQueued:          PRLeave,
	TaskStatusRunning:         PRLeave,
}

// DecidePR decides what to do with the PR a task left behind.
//
// changedFiles is what the PR actually touches. It is checked against the
// agent's DECLARED artifact_patterns before any merge — not the effective ones,
// which fall back to `**/*` and would bound nothing. An operator approving a
// task consented to the diff on the card; this is the second, mechanical check
// that the branch still matches what that agent is allowed to land.
func DecidePR(task *TaskRecord, agent *AgentConfig, changedFiles []string) PRDecision {
	if task == nil {
		return PRDecision{PRLeave, "no task record"}
	}
	verdict, known := prStatusVerdict[task.Status]
	if !known {
		// An unknown status leaves the PR alone. Safe direction: a stale open PR
		// is visible and cheap, a wrong merge is neither.
		return PRDecision{PRLeave, fmt.Sprintf("unknown task status %q", task.Status)}
	}
	if verdict != PRMerge {
		return PRDecision{verdict, fmt.Sprintf("task is %s", task.Status)}
	}

	// From here the task is completed and the PR would be MERGED.
	if agent == nil {
		return PRDecision{PRRefuse, "no agent config: cannot check the branch against a declared scope"}
	}
	if len(agent.ArtifactPatterns) == 0 {
		return PRDecision{PRRefuse, fmt.Sprintf(
			"agent %q declares no artifact_patterns, so nothing bounds what this merge would land", agent.ID)}
	}
	if len(changedFiles) == 0 {
		return PRDecision{PRRefuse, "the PR reports no changed files: nothing to check, so nothing to trust"}
	}
	var outside []string
	for _, f := range changedFiles {
		if !MatchesArtifactPattern(agent.ArtifactPatterns, f) {
			outside = append(outside, f)
		}
	}
	if len(outside) > 0 {
		return PRDecision{PRRefuse, fmt.Sprintf(
			"%d file(s) outside %q's declared artifact_patterns: %s",
			len(outside), agent.ID, strings.Join(outside, ", "))}
	}
	return PRDecision{PRMerge, fmt.Sprintf("task completed and all %d file(s) are inside %s",
		len(changedFiles), strings.Join(agent.ArtifactPatterns, ", "))}
}

// BranchForTask is the branch the cloud wrapper creates.
func BranchForTask(taskID string) string { return "coordinator/" + taskID }
