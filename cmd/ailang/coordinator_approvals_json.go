package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// Machine-readable pending approvals, for the SessionStart hook and any agent
// that wants to act on the queue rather than read a card.
//
// The queue is what the plane is FOR. Delivery, execution and handoff all work
// (measured 2026-09-07/08: 26 of 27 job executions succeeded, and the
// design-doc-creator -> sprint-planner -> sprint-executor chain fired end to
// end), so the binding constraint is no longer "did the message arrive" but
// "is anyone going to decide". A decision queue that only exists behind a
// command nobody runs is a queue that silently grows: 6 approvals were pending
// in prod when this was written, the oldest 8h, and no session had been told.
//
// Two rules this output has to keep:
//
//   - An UNREACHABLE queue must never render as an EMPTY one. Every consumer
//     here reports the error in-band (`error` field, non-zero exit) rather
//     than emitting `[]`. This is the exact shape of failure the message plane
//     kept producing — a green banner over a broken read.
//   - `agent_actionable` is ADVICE, not a control. An agent session and a human
//     drive the identical CLI against the identical store, so nothing here can
//     enforce who approves; pretending otherwise would be a safeguard that does
//     not exist. It reports whether the configured policy COVERS the row, and
//     the deciding caller remains responsible.

// approvalPolicy names who may resolve a pending approval from an agent session.
type approvalPolicy string

const (
	// approvalPolicyNever is the default: every approval is the operator's.
	approvalPolicyNever approvalPolicy = "never"
	// approvalPolicyEvaluated lets an agent resolve rows an independent
	// evaluator already passed AND whose diff is visible. A row with no diff
	// is never covered — approving what you cannot see is the failure #921
	// was filed for, and an agent cannot "look at the PR" to compensate.
	approvalPolicyEvaluated approvalPolicy = "evaluated"
	// approvalPolicyAlways covers every pending row.
	approvalPolicyAlways approvalPolicy = "always"
)

// resolveApprovalPolicy reads AILANG_APPROVAL_POLICY, defaulting to never.
//
// Env rather than the agent registry because the policy is a property of THIS
// session (attended, unattended loop, package agent), not of the cloud
// coordinator's config — the same registry serves every caller.
func resolveApprovalPolicy() approvalPolicy {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AILANG_APPROVAL_POLICY"))) {
	case string(approvalPolicyEvaluated):
		return approvalPolicyEvaluated
	case string(approvalPolicyAlways):
		return approvalPolicyAlways
	default:
		return approvalPolicyNever
	}
}

// pendingApprovalJSON is one row of the decision queue.
type pendingApprovalJSON struct {
	ID          string  `json:"id"`
	TaskID      string  `json:"task_id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
	AgeHours    float64 `json:"age_hours"`

	// Evaluation is the sprint-evaluator verdict when one has reported:
	// PASS/FAIL/UNAVAILABLE, or empty if the stage has not run.
	Evaluation string `json:"evaluation,omitempty"`

	// SourceAgent and HandoffTargets say what approving STARTS. On a
	// merge_handoff this dispatches the next agent in the chain, which is the
	// difference between an approval and a merge.
	SourceAgent    string   `json:"source_agent,omitempty"`
	HandoffTargets []string `json:"handoff_targets,omitempty"`

	ChangedFiles    int    `json:"changed_files"`
	DiffAvailable   bool   `json:"diff_available"`
	DiffUnavailable string `json:"diff_unavailable,omitempty"`

	// AgentActionable reports whether the configured policy covers this row.
	// Advice, not a control — see the file comment.
	AgentActionable bool   `json:"agent_actionable"`
	PolicyReason    string `json:"policy_reason"`
}

// approvalsJSONOutput is the whole queue plus the things that make a zero
// trustworthy: which plane it was read from, and how many stuck rows exist that
// the pending list structurally cannot show.
type approvalsJSONOutput struct {
	Store     string                `json:"store"`
	Policy    string                `json:"policy"`
	Pending   []pendingApprovalJSON `json:"pending"`
	Orphans   int                   `json:"orphans"`
	OrphanErr string                `json:"orphan_error,omitempty"`
}

// approvalContext is the reviewable half of an approval, as the executor recorded it.
type approvalContext struct {
	HandoffTargets  []string `json:"handoff_targets"`
	SourceAgent     string   `json:"source_agent"`
	ChangedFiles    []string `json:"changed_files"`
	DiffStat        string   `json:"diff_stat"`
	Diff            string   `json:"diff"`
	DiffUnavailable string   `json:"diff_unavailable"`
}

// parseApprovalContext decodes ContextJSON, reporting unparseable context as
// absent rather than as empty — a row whose context will not decode is exactly
// as unreviewable as one that has none, and must not render as a clean zero.
func parseApprovalContext(raw string) (approvalContext, bool) {
	var out approvalContext
	if strings.TrimSpace(raw) == "" {
		return out, false
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return approvalContext{DiffUnavailable: fmt.Sprintf("context could not be parsed: %v", err)}, false
	}
	return out, true
}

// buildPendingApprovalJSON projects one record through the policy.
func buildPendingApprovalJSON(req *coordinator.ApprovalRequestRecord, policy approvalPolicy, now time.Time) pendingApprovalJSON {
	ctxData, ok := parseApprovalContext(req.ContextJSON)

	row := pendingApprovalJSON{
		ID:              req.ID,
		TaskID:          req.TaskID,
		Type:            req.Type,
		Description:     req.Description,
		CreatedAt:       req.CreatedAt.UTC().Format(time.RFC3339),
		AgeHours:        now.Sub(req.CreatedAt).Hours(),
		Evaluation:      req.Evaluation,
		SourceAgent:     ctxData.SourceAgent,
		HandoffTargets:  ctxData.HandoffTargets,
		ChangedFiles:    len(ctxData.ChangedFiles),
		DiffUnavailable: ctxData.DiffUnavailable,
	}
	row.DiffAvailable = ok && ctxData.DiffUnavailable == "" && len(ctxData.ChangedFiles) > 0

	row.AgentActionable, row.PolicyReason = agentMayApprove(row, policy)
	return row
}

// agentMayApprove applies the policy to one row and RETURNS ITS REASON, so a
// banner can say why a row is the operator's rather than just that it is.
func agentMayApprove(row pendingApprovalJSON, policy approvalPolicy) (bool, string) {
	switch policy {
	case approvalPolicyAlways:
		return true, "policy=always"
	case approvalPolicyEvaluated:
		if !row.DiffAvailable {
			return false, "no visible diff — cannot approve what cannot be reviewed"
		}
		if !strings.EqualFold(row.Evaluation, "PASS") {
			if row.Evaluation == "" {
				return false, "no evaluator verdict yet"
			}
			return false, "evaluator verdict " + row.Evaluation
		}
		return true, "policy=evaluated, verdict PASS, diff visible"
	default:
		return false, "policy=never — approvals are the operator's (set AILANG_APPROVAL_POLICY to change)"
	}
}

// collectPendingApprovals reads the queue for a resolved plane.
//
// It returns an error rather than an empty queue whenever the read itself
// failed. Orphans are counted alongside because a pending list of zero with
// stuck rows behind it is the misleading case, not the healthy one.
func collectPendingApprovals(ctx context.Context, bundle *coordinatorStoreBundle, policy approvalPolicy, now time.Time) (*approvalsJSONOutput, error) {
	pending, err := bundle.Store.ListPendingApprovals(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending approvals: %w", err)
	}

	out := &approvalsJSONOutput{
		Store:   bundle.Mode,
		Policy:  string(policy),
		Pending: make([]pendingApprovalJSON, 0, len(pending)),
	}
	for _, req := range pending {
		out.Pending = append(out.Pending, buildPendingApprovalJSON(req, policy, now))
	}

	orphans, oErr := findOrphanedApprovals(ctx, bundle.Store)
	if oErr != nil {
		out.OrphanErr = oErr.Error()
	}
	out.Orphans = len(orphans)

	return out, nil
}

// printApprovalsJSON writes the queue as JSON on stdout.
func printApprovalsJSON(out *approvalsJSONOutput) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
