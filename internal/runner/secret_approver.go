package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/secrets"
)

// EnvApprovalURL names the service that serves /api/approvals (the dashboard)
// for the networked secret gate. On the shared storage plane it is what makes
// secret() gated at all.
const EnvApprovalURL = "AILANG_APPROVAL_URL"

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
// Optional env: AILANG_APPROVAL_TOKEN (authenticates the request),
// AILANG_AGENT_ID / AILANG_TASK_ID (label the approval request).
//
// Absent BOTH URLs on the shared plane, the run used to be silently un-gated
// — the M-SECRET-REMOTE-APPROVAL-WIRING M2 note. That is now the deprecated
// default (M-V1-SIMPLIFY-S4 M1): still un-gated, but config.DeprecatedDefault
// warns once per process naming AILANG_APPROVAL_URL, and under
// AILANG_STRICT_CONFIG=1 a denying approver is installed instead, so every
// secret() on that plane fails with config.ErrDeprecatedDefault. The denial is
// deferred into the approver for the same reason as the plane error below:
// a program that never calls secret() must not be broken by it.
//
// A plane that does not resolve (a retired selector, an unknown value) is an
// ERROR, not "not cloud": this is a security gate, and skipping it because
// the environment was malformed would be fail-open. But it is an error for
// the PROGRAM THAT CALLS secret(), not for every program: a shell that still
// exports a retired ops selector must not break `ailang run hello.ail`
// (measured 2026-09-15 — the old CLAUDE.md recipe exported one in every
// session). So the error is deferred into an approver that denies the first
// secret() with the exact resolution error; programs that never touch a
// secret never see it. Fail-closed where it matters, quiet where it does not.
func attachCloudSecretApprover(effCtx *effects.EffContext) error {
	if effCtx == nil || effCtx.Secret == nil {
		return nil
	}
	plane, err := config.StoragePlane()
	if err != nil {
		effCtx.Secret.Approver = deferredPlaneError{err: fmt.Errorf("secret approver: %w", err)}
		return nil
	}
	if !plane.Shared() {
		return nil
	}
	// The approver POSTs to the service that serves /api/approvals — the
	// dashboard. AILANG_APPROVAL_URL names it explicitly; fall back to
	// AILANG_COORDINATOR_URL for compatibility.
	approvalURL := os.Getenv(EnvApprovalURL)
	if approvalURL == "" {
		approvalURL = os.Getenv("AILANG_COORDINATOR_URL")
	}
	if approvalURL == "" {
		if _, err := config.DeprecatedDefault(EnvApprovalURL, "<none: secret() un-gated on the shared plane>"); err != nil {
			effCtx.Secret.Approver = deferredPlaneError{err: fmt.Errorf("secret approver: shared storage plane with no approval endpoint: %w", err)}
		}
		return nil
	}
	effCtx.Secret.Approver = secrets.NewCloudSecretApprover(
		approvalURL,
		secrets.WithApproverIdentity(os.Getenv("AILANG_AGENT_ID"), os.Getenv("AILANG_TASK_ID")),
		secrets.WithApproverAuthToken(os.Getenv("AILANG_APPROVAL_TOKEN")),
	)
	return nil
}

// deferredPlaneError is the approver installed when the storage plane could
// not be resolved at startup: every secret() is denied with that error, so a
// malformed environment is loud exactly when a secret is at stake and silent
// otherwise.
type deferredPlaneError struct{ err error }

func (d deferredPlaneError) Approve(context.Context, string, string) error { return d.err }
