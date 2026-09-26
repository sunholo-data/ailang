package coordinator

// Approval cards whose pull request has ALREADY merged.
//
// pr_reconcile.go runs card → PR: a decision is made, the PR follows it. Nothing
// ran PR → card, so when the PR merged first — by hand, by a repository's own
// docs-only auto-merge, by another session — the card stayed pending forever.
// Measured 2026-09-23: 57 of 77 pending cards on the prod plane had a merged
// coordinator PR, most merged the day the card was created. The queue an
// operator reads as "what needs me" was three-quarters ledger.
//
// Kept pure, like DecidePR: the policy is the part that can do damage.

import "fmt"

// DecideLandedCard reports whether task's pending card was decided by the merge
// of pr. It requires all of: the task is still pending_approval, the approval
// record is still pending, and the PR is merged from exactly this task's branch.
//
// The branch match is exact on purpose. Head names are the only link between a
// task and a PR, and a prefix or fuzzy match would let one task's merge resolve
// another's card.
func DecideLandedCard(task *TaskRecord, approval *ApprovalRequestRecord, pr *LandedPR) (bool, string) {
	switch {
	case task == nil || pr == nil:
		return false, "no task or no PR"
	case task.Status != TaskStatusPendingApproval:
		return false, fmt.Sprintf("task is %s, not pending_approval", task.Status)
	case approval == nil:
		return false, "no approval record"
	case approval.Status != "pending":
		return false, fmt.Sprintf("approval is already %s", approval.Status)
	case pr.State != "MERGED":
		return false, fmt.Sprintf("PR #%d is %s, not merged", pr.Number, pr.State)
	case pr.HeadRefName != BranchForTask(task.ID):
		return false, fmt.Sprintf("PR #%d head %q is not %q", pr.Number, pr.HeadRefName, BranchForTask(task.ID))
	}
	return true, fmt.Sprintf("PR #%d merged %s by %s", pr.Number, pr.MergedAt, orUnknown(pr.MergedBy))
}

// LandedPR is the subset of a merged PR DecideLandedCard needs; it keeps this
// package free of the messaging client's types.
type LandedPR struct {
	Number      int
	State       string
	HeadRefName string
	MergedAt    string
	MergedBy    string
}

// LandedApprover is the ApprovedBy recorded on a card resolved by its merge, so
// the audit trail names the merge — not an operator who never clicked.
func LandedApprover(pr *LandedPR) string {
	return fmt.Sprintf("pr-merge #%d (%s)", pr.Number, orUnknown(pr.MergedBy))
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// ApprovalHandoffTargets exposes the edges an approval of task's agent would
// fire, for reporting before anything is dispatched.
func ApprovalHandoffTargets(registry *AgentRegistry, task *TaskRecord) []string {
	if registry == nil || task == nil {
		return nil
	}
	a := registry.GetAgentByID(task.AgentID)
	if a == nil {
		return nil
	}
	return approvalHandoffTargets(a)
}
