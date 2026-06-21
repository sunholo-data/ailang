package coordinator

import (
	"context"
	"testing"
	"time"
)

// resolveAfter approves or rejects the single pending request shortly after it
// is created, simulating an operator tapping Approve/Deny on their phone.
func resolveAfter(t *testing.T, cp *ApprovalCheckpoint, approve bool) {
	t.Helper()
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			pending := cp.GetPendingRequests()
			if len(pending) > 0 {
				if approve {
					_ = cp.Approve(pending[0].ID, "phone")
				} else {
					_ = cp.Reject(pending[0].ID, "phone")
				}
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
}

func TestSecretApprovalGate_Approved(t *testing.T) {
	cp := NewApprovalCheckpoint(2 * time.Second)
	gate := NewSecretApprovalGate(cp, "agent-x", "task-1")
	resolveAfter(t, cp, true)
	if err := gate.Approve(context.Background(), "op://Prod/stripe/api-key", "charge a card"); err != nil {
		t.Fatalf("expected approval, got error: %v", err)
	}
}

func TestSecretApprovalGate_Rejected(t *testing.T) {
	cp := NewApprovalCheckpoint(2 * time.Second)
	gate := NewSecretApprovalGate(cp, "agent-x", "task-1")
	resolveAfter(t, cp, false)
	err := gate.Approve(context.Background(), "op://Prod/stripe/api-key", "charge a card")
	if err == nil {
		t.Fatal("expected rejection error, got nil")
	}
}

func TestSecretApprovalGate_RequestCarriesRefNotValue(t *testing.T) {
	cp := NewApprovalCheckpoint(2 * time.Second)
	gate := NewSecretApprovalGate(cp, "agent-x", "task-1")

	var captured *ApprovalRequest
	done := make(chan struct{})
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			pending := cp.GetPendingRequests()
			if len(pending) > 0 {
				captured = pending[0]
				_ = cp.Approve(pending[0].ID, "phone")
				close(done)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		close(done)
	}()

	_ = gate.Approve(context.Background(), "op://Prod/stripe/api-key", "charge a card")
	<-done

	if captured == nil {
		t.Fatal("no approval request was created")
	}
	if captured.Type != ApprovalTypeSecret {
		t.Fatalf("type = %q, want %q", captured.Type, ApprovalTypeSecret)
	}
	if captured.SecretRef != "op://Prod/stripe/api-key" {
		t.Fatalf("SecretRef = %q", captured.SecretRef)
	}
	if captured.AgentID != "agent-x" {
		t.Fatalf("AgentID = %q", captured.AgentID)
	}
	if !captured.AutoReject {
		t.Fatal("secret approval must fail closed (AutoReject=true)")
	}
}
