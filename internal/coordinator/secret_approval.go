package coordinator

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
)

// approvalRequester is the subset of an approval checkpoint the secret gate
// needs. Both ApprovalCheckpoint and StoreBackedApprovalCheckpoint satisfy it.
type approvalRequester interface {
	RequestApproval(ctx context.Context, request *ApprovalRequest) (ApprovalStatus, error)
}

// SecretApprovalGate adapts an approval checkpoint to effects.SecretApprover so
// the Secret effect (internal/effects) can block on a human decision without
// importing the coordinator. The runtime injects an instance into
// EffContext.Secret.Approver for coordinator-run tasks.
//
// Only the op:// reference and a human-readable purpose cross this boundary —
// the resolved secret value is never part of an approval request.
type SecretApprovalGate struct {
	checkpoint approvalRequester
	agentID    string
	taskID     string
}

// NewSecretApprovalGate builds a gate bound to a checkpoint and the requesting
// agent/task (used for the audit trail and the notification payload).
func NewSecretApprovalGate(cp approvalRequester, agentID, taskID string) *SecretApprovalGate {
	return &SecretApprovalGate{checkpoint: cp, agentID: agentID, taskID: taskID}
}

// Approve implements effects.SecretApprover. It blocks until the operator
// approves or denies the request (or it times out). It fails closed: any
// non-approved outcome returns an error so the secret is never resolved.
func (g *SecretApprovalGate) Approve(ctx context.Context, ref, purpose string) error {
	req := &ApprovalRequest{
		TaskID:        g.taskID,
		Type:          ApprovalTypeSecret,
		Title:         fmt.Sprintf("Secret requested: %s", ref),
		Description:   fmt.Sprintf("Agent %q requests %s — %s", g.agentID, ref, purpose),
		SecretRef:     ref,
		SecretPurpose: purpose,
		AgentID:       g.agentID,
		AutoReject:    true, // fail closed: deny on timeout
	}
	status, err := g.checkpoint.RequestApproval(ctx, req)
	if err != nil {
		return fmt.Errorf("approval failed: %w", err)
	}
	switch status {
	case ApprovalStatusApproved:
		return nil
	case ApprovalStatusRejected:
		return fmt.Errorf("secret access rejected for %s", ref)
	case ApprovalStatusTimeout:
		return fmt.Errorf("secret access timed out for %s", ref)
	default:
		return fmt.Errorf("secret access not approved (%s) for %s", status, ref)
	}
}

// Compile-time check that the gate satisfies the effects-layer interface.
var _ effects.SecretApprover = (*SecretApprovalGate)(nil)
