package coordinator

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
)

func TestBuildSecretApprovalNotification(t *testing.T) {
	signer, err := approvaltoken.NewSigner([]byte("test-key-abcdefghijklmnop"))
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	req := &ApprovalRequest{
		ID:            "appr-123",
		TaskID:        "task-42",
		Type:          ApprovalTypeSecret,
		SecretRef:     "op://Prod/stripe/api-key",
		SecretPurpose: "charge a card",
		AgentID:       "agent-x",
	}
	n, err := BuildSecretApprovalNotification(req, "https://coord.run.app", signer, time.Hour)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if n.EventType != "pending_approval" {
		t.Fatalf("EventType = %q", n.EventType)
	}
	// Richer body: who, what, why, which task, and the decision window.
	for _, want := range []string{"Requested by: agent-x", "op://Prod/stripe/api-key", "Purpose: charge a card", "Task: task-42", "Decide within"} {
		if !strings.Contains(n.Body, want) {
			t.Fatalf("body missing %q: %q", want, n.Body)
		}
	}

	// When purpose defaults to the ref, don't render a redundant Purpose line.
	plain := &ApprovalRequest{ID: "a", Type: ApprovalTypeSecret, SecretRef: "op://V/i/f", SecretPurpose: "op://V/i/f", AgentID: "a"}
	pn, _ := BuildSecretApprovalNotification(plain, "https://c", signer, time.Hour)
	if strings.Contains(pn.Body, "Purpose:") {
		t.Errorf("should omit Purpose when it equals the ref: %q", pn.Body)
	}
	if len(n.Actions) != 2 {
		t.Fatalf("want 2 actions, got %d", len(n.Actions))
	}

	// Each action URL must carry a token that verifies and matches its action.
	for _, a := range n.Actions {
		idx := strings.Index(a.URL, "token=")
		if idx < 0 {
			t.Fatalf("action %q has no token", a.Label)
		}
		tok := a.URL[idx+len("token="):]
		// URL-decode the single token (QueryEscape may have encoded it).
		tok = strings.ReplaceAll(tok, "%3D", "=")
		claims, err := signer.Verify(tok)
		if err != nil {
			t.Fatalf("token for %q failed verify: %v", a.Label, err)
		}
		if claims.ApprovalID != "appr-123" {
			t.Fatalf("token approvalID = %q", claims.ApprovalID)
		}
		wantAction := "approve"
		if a.Label == "Deny" {
			wantAction = "reject"
		}
		if claims.Action != wantAction {
			t.Fatalf("action %q token action = %q, want %q", a.Label, claims.Action, wantAction)
		}
	}
}

func TestBuildSecretApprovalNotification_RejectsNonSecret(t *testing.T) {
	signer, _ := approvaltoken.NewSigner([]byte("k0123456789abcdef"))
	req := &ApprovalRequest{ID: "x", Type: ApprovalTypeMerge}
	if _, err := BuildSecretApprovalNotification(req, "https://x", signer, time.Hour); err == nil {
		t.Fatal("expected error for non-secret approval type")
	}
}
