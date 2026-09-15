package runner

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/secrets"
)

// attachCloudSecretApprover wires a networked secret-approval gate onto the
// effect context when running in cloud mode (M-SECRET-REMOTE-APPROVAL-WIRING).
// With it attached, secret() blocks on a human approval from the coordinator
// before the value is resolved.
//
// Cloud mode = the SHARED storage plane (AILANG_STORAGE=gcp or hybrid, per
// config.StoragePlane — hybrid silently lacked the approver before
// M-V1-SIMPLIFY-S3 M3) AND an approval-API URL set. The approver POSTs to
// the service that serves /api/approvals (the dashboard):
// AILANG_APPROVAL_URL names it, falling back to AILANG_COORDINATOR_URL.
// Absent either, the approver stays nil and runs are un-gated — identical to
// local CLI today. Optional env: AILANG_APPROVAL_TOKEN (authenticates the
// request), AILANG_AGENT_ID / AILANG_TASK_ID (label the approval request).
//
// NOTE (M2 follow-up): in gcp mode WITHOUT a coordinator URL a secret() call is
// currently un-gated. Promoting that to a fail-closed policy error is tracked in
// the M-SECRET-REMOTE-APPROVAL-WIRING M2 milestone.
//
// A plane that does not resolve (a retired selector, an unknown value) is an
// ERROR, not "not cloud": this is a security gate, and skipping it because
// the environment was malformed would be fail-open.
func attachCloudSecretApprover(effCtx *effects.EffContext) error {
	if effCtx == nil || effCtx.Secret == nil {
		return nil
	}
	plane, err := config.StoragePlane()
	if err != nil {
		return fmt.Errorf("secret approver: %w", err)
	}
	if !plane.Shared() {
		return nil
	}
	// The approver POSTs to the service that serves /api/approvals — the
	// dashboard. AILANG_APPROVAL_URL names it explicitly; fall back to
	// AILANG_COORDINATOR_URL for compatibility.
	approvalURL := os.Getenv("AILANG_APPROVAL_URL")
	if approvalURL == "" {
		approvalURL = os.Getenv("AILANG_COORDINATOR_URL")
	}
	if approvalURL == "" {
		return nil
	}
	effCtx.Secret.Approver = secrets.NewCloudSecretApprover(
		approvalURL,
		secrets.WithApproverIdentity(os.Getenv("AILANG_AGENT_ID"), os.Getenv("AILANG_TASK_ID")),
		secrets.WithApproverAuthToken(os.Getenv("AILANG_APPROVAL_TOKEN")),
	)
	return nil
}
