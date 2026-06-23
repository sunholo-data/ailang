package main

import (
	"os"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/secrets"
)

// attachCloudSecretApprover wires a networked secret-approval gate onto the
// effect context when running in cloud mode (M-SECRET-REMOTE-APPROVAL-WIRING).
// With it attached, secret() blocks on a human approval from the coordinator
// before the value is resolved.
//
// Cloud mode = AILANG_STORAGE=gcp AND an approval-API URL set. The approver
// POSTs to the service that serves /api/approvals (the dashboard):
// AILANG_APPROVAL_URL names it, falling back to AILANG_COORDINATOR_URL.
// Absent either, the approver stays nil and runs are un-gated — identical to
// local CLI today. Optional env: AILANG_APPROVAL_TOKEN (authenticates the
// request), AILANG_AGENT_ID / AILANG_TASK_ID (label the approval request).
//
// NOTE (M2 follow-up): in gcp mode WITHOUT a coordinator URL a secret() call is
// currently un-gated. Promoting that to a fail-closed policy error is tracked in
// the M-SECRET-REMOTE-APPROVAL-WIRING M2 milestone.
func attachCloudSecretApprover(effCtx *effects.EffContext) {
	if effCtx == nil || effCtx.Secret == nil {
		return
	}
	if os.Getenv("AILANG_STORAGE") != "gcp" {
		return
	}
	// The approver POSTs to the service that serves /api/approvals — the
	// dashboard. AILANG_APPROVAL_URL names it explicitly; fall back to
	// AILANG_COORDINATOR_URL for compatibility.
	approvalURL := os.Getenv("AILANG_APPROVAL_URL")
	if approvalURL == "" {
		approvalURL = os.Getenv("AILANG_COORDINATOR_URL")
	}
	if approvalURL == "" {
		return
	}
	effCtx.Secret.Approver = secrets.NewCloudSecretApprover(
		approvalURL,
		secrets.WithApproverIdentity(os.Getenv("AILANG_AGENT_ID"), os.Getenv("AILANG_TASK_ID")),
		secrets.WithApproverAuthToken(os.Getenv("AILANG_APPROVAL_TOKEN")),
	)
}
