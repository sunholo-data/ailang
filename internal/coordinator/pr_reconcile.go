package coordinator

// Deciding what should happen to the pull request a task left behind.
//
// Kept pure and separate from the GitHub calls, because the interesting part is
// the POLICY and the policy is where this can do damage. Merging a PR is the
// one irreversible thing in this file.

import (
	"encoding/json"
	"fmt"
	"sort"
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
	TaskStatusBlocked:         PRClose, // never started; the branch is the clone and nothing else
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
//
// cardFiles is what the approval card SAID would land, read back from the
// approval record. Passing it is not optional politeness: measured 2026-09-15,
// all eleven reopened ailang-core-triage approvals named
// `design_docs/planned/ailang-core-backlog.md` while every branch carried a
// different per-report file, because a re-run replaced the work and left the
// card alone. One of them was approved and merged a file its card never named.
// An empty cardFiles means "no evidence recorded" and skips the check; a
// non-empty one that disagrees with the branch refuses the merge.
func DecidePR(task *TaskRecord, agent *AgentConfig, changedFiles, cardFiles []string) PRDecision {
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
	// The card and the branch must be describing the same change. Refusing here
	// is the only place that can catch an approval made on superseded evidence,
	// because by then the decision is already recorded.
	if missing, extra := CardBranchDisagreement(cardFiles, changedFiles); len(cardFiles) > 0 && (len(missing) > 0 || len(extra) > 0) {
		var b strings.Builder
		b.WriteString("the approval card and the branch describe DIFFERENT changes, so the recorded decision was made on evidence that is not what would land")
		if len(missing) > 0 {
			fmt.Fprintf(&b, "; on the card but NOT on the branch: %s", strings.Join(missing, ", "))
		}
		if len(extra) > 0 {
			fmt.Fprintf(&b, "; on the branch but NOT on the card: %s", strings.Join(extra, ", "))
		}
		b.WriteString(". Re-run the task (finalisation now refreshes a pending card) or merge by hand if the branch is what you meant to approve")
		return PRDecision{PRRefuse, b.String()}
	}
	reason := fmt.Sprintf("task completed and all %d file(s) are inside %s",
		len(changedFiles), strings.Join(agent.ArtifactPatterns, ", "))
	if len(cardFiles) == 0 {
		// Never silent: proceeding without the check is a fact about this merge.
		reason += " (the approval recorded no file list, so the card could not be checked against the branch)"
	}
	return PRDecision{PRMerge, reason}
}

// CardBranchDisagreement compares two file lists as SETS, ignoring order.
//
// Order varies between the approval's recorded diff and GitHub's PR listing,
// and a check that treats that as disagreement would refuse every merge —
// which is how a safety check gets switched off.
func CardBranchDisagreement(card, branch []string) (missing, extra []string) {
	inBranch := make(map[string]bool, len(branch))
	for _, f := range branch {
		inBranch[f] = true
	}
	inCard := make(map[string]bool, len(card))
	for _, f := range card {
		inCard[f] = true
	}
	for f := range inCard {
		if !inBranch[f] {
			missing = append(missing, f)
		}
	}
	for f := range inBranch {
		if !inCard[f] {
			extra = append(extra, f)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}

// CardFilesFromContext reads the file list the approval card showed.
//
// Returns nil when the approval recorded no diff — which must read as "no
// evidence", never as "no files": an approval with no recorded diff and a
// branch with three files are not in disagreement, they are simply not
// comparable.
func CardFilesFromContext(contextJSON string) []string {
	if strings.TrimSpace(contextJSON) == "" {
		return nil
	}
	var obj struct {
		ChangedFiles []string `json:"changed_files"`
	}
	if err := json.Unmarshal([]byte(contextJSON), &obj); err != nil {
		return nil
	}
	return obj.ChangedFiles
}

// BranchForTask is the branch the cloud wrapper creates.
func BranchForTask(taskID string) string { return "coordinator/" + taskID }
