package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

type mockApprovalPublisher struct {
	called                            bool
	approvalID, approvalType, agentID string
	payload                           []byte
}

func (m *mockApprovalPublisher) PublishApproval(_ context.Context, approvalID, approvalType, agentID string, notificationJSON []byte) error {
	m.called = true
	m.approvalID, m.approvalType, m.agentID, m.payload = approvalID, approvalType, agentID, notificationJSON
	return nil
}

// TestPublishSecretApprovalRequested_BuildsValueFreePush: intake publishes a
// notification that references the secret and carries signed Approve/Deny action
// URLs — but never a resolved value.
func TestPublishSecretApprovalRequested_BuildsValueFreePush(t *testing.T) {
	t.Setenv("AILANG_APPROVAL_BASE_URL", "https://dash.example")
	signer, err := approvaltoken.NewSigner([]byte("test-signing-key-0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	pub := &mockApprovalPublisher{}
	s := &Server{}
	s.SetSecretApprovalAuth(signer)
	s.SetApprovalPublisher(pub)

	ctxJSON, _ := json.Marshal(secretApprovalContext{Ref: "op://Prod/stripe/key", Purpose: "charge", Agent: "agent-x"})
	rec := &coordinator.ApprovalRequestRecord{ID: "secret-9", Type: "secret", ContextJSON: string(ctxJSON)}

	s.publishSecretApprovalRequested(context.Background(), rec)

	if !pub.called {
		t.Fatal("expected PublishApproval to be called")
	}
	if pub.approvalID != "secret-9" || pub.approvalType != "secret" || pub.agentID != "agent-x" {
		t.Errorf("unexpected attrs: id=%s type=%s agent=%s", pub.approvalID, pub.approvalType, pub.agentID)
	}
	body := string(pub.payload)
	if !strings.Contains(body, "op://Prod/stripe/key") {
		t.Errorf("notification should reference the secret: %s", body)
	}
	if !strings.Contains(body, "/approve?token=") || !strings.Contains(body, "/reject?token=") {
		t.Errorf("notification should carry signed Approve/Deny action URLs: %s", body)
	}
}

// TestPublishSecretApprovalRequested_NoopWithoutConfig: with no publisher/signer/
// base URL, the push is skipped without panicking (the executor still polls).
func TestPublishSecretApprovalRequested_NoopWithoutConfig(t *testing.T) {
	s := &Server{}
	s.publishSecretApprovalRequested(context.Background(), &coordinator.ApprovalRequestRecord{ID: "x", Type: "secret"})
}

func newSecretApprovalServer() (*Server, *MockApprovalStore) {
	store := NewMockApprovalStore()
	s := &Server{}
	s.SetApprovalStore(store)
	return s, store
}

// TestSecretApprovalIntake_CreatesPendingValueFreeRecord: POST /api/approvals
// creates a pending Type=="secret" record whose context carries the ref but no
// value.
func TestSecretApprovalIntake_CreatesPendingValueFreeRecord(t *testing.T) {
	s, store := newSecretApprovalServer()
	body, _ := json.Marshal(secretIntakeRequest{Ref: "op://Prod/stripe/key", Purpose: "charge", Agent: "agent-x"})
	req := httptest.NewRequest(http.MethodPost, "/api/approvals", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handleApprovals(w, req) // POST → intake branch

	if w.Code != http.StatusOK {
		t.Fatalf("intake: expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["id"]
	if !strings.HasPrefix(id, "secret-") {
		t.Fatalf("expected a secret-* id, got %q", id)
	}
	rec := store.approvals[id]
	if rec == nil || rec.Type != "secret" || rec.Status != "pending" {
		t.Fatalf("expected a pending secret record, got %+v", rec)
	}
	if !strings.Contains(rec.ContextJSON, "op://Prod/stripe/key") {
		t.Errorf("context should carry the ref: %s", rec.ContextJSON)
	}
	if strings.Contains(rec.ContextJSON, "\"value\"") {
		t.Errorf("secret record context must be value-free: %s", rec.ContextJSON)
	}
}

func TestSecretApprovalIntake_RejectsMissingRef(t *testing.T) {
	s, _ := newSecretApprovalServer()
	body, _ := json.Marshal(secretIntakeRequest{Purpose: "charge"})
	req := httptest.NewRequest(http.MethodPost, "/api/approvals", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleApprovals(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing ref, got %d", w.Code)
	}
}

// TestSecretApprovalStatus_PendingThenApproved: GET /api/approvals/{id} reflects
// the current decision — what CloudSecretApprover polls.
func TestSecretApprovalStatus_PendingThenApproved(t *testing.T) {
	s, store := newSecretApprovalServer()
	store.approvals["secret-1"] = &coordinator.ApprovalRequestRecord{ID: "secret-1", Type: "secret", Status: "pending"}

	get := func() map[string]string {
		req := httptest.NewRequest(http.MethodGet, "/api/approvals/secret-1", nil)
		w := httptest.NewRecorder()
		s.handleApproval(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status GET: expected 200, got %d (%s)", w.Code, w.Body.String())
		}
		var st map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &st)
		return st
	}

	if st := get(); st["status"] != "pending" {
		t.Fatalf("expected pending, got %q", st["status"])
	}
	_ = store.ResolveApprovalRequest(context.Background(), "secret-1", "approved", "tester")
	if st := get(); st["status"] != "approved" {
		t.Fatalf("expected approved, got %q", st["status"])
	}
}

// TestSecretApprovalApprove_ResolvesWithoutTaskPath: approving a secret request
// resolves it directly via the store. coordStoreRaw is nil, proving the secret
// branch does NOT run the task merge/handoff processor.
func TestSecretApprovalApprove_ResolvesWithoutTaskPath(t *testing.T) {
	s, store := newSecretApprovalServer()
	store.approvals["secret-2"] = &coordinator.ApprovalRequestRecord{ID: "secret-2", Type: "secret", Status: "pending"}

	req := httptest.NewRequest(http.MethodPost, "/api/approvals/secret-2/approve", nil)
	w := httptest.NewRecorder()
	s.handleApproval(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("approve: expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if got := store.approvals["secret-2"].Status; got != "approved" {
		t.Fatalf("expected approved, got %q", got)
	}
}

// TestSecretApprovalTerminalLock_NoReflip: once a secret approval is resolved,
// tapping the other button (a different token) must NOT flip the decision —
// the audit record is immutable. Returns 409 and leaves the status unchanged.
func TestSecretApprovalTerminalLock_NoReflip(t *testing.T) {
	s, store := newSecretApprovalServer()
	store.approvals["secret-3"] = &coordinator.ApprovalRequestRecord{
		ID: "secret-3", Type: "secret", Status: "approved", ResolvedBy: "operator",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/approvals/secret-3/reject", nil)
	w := httptest.NewRecorder()
	s.handleApproval(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict re-resolving a decided approval, got %d (%s)", w.Code, w.Body.String())
	}
	if got := store.approvals["secret-3"].Status; got != "approved" {
		t.Fatalf("status must stay approved (not flip to rejected), got %q", got)
	}
}

// TestSecretApprovalConfirmationPush: resolving a secret approval sends a
// value-free "secret-resolved" confirmation push so the operator sees the
// decision landed (ntfy buttons can't reflect their own outcome).
func TestSecretApprovalConfirmationPush(t *testing.T) {
	s, store := newSecretApprovalServer()
	pub := &mockApprovalPublisher{}
	s.SetApprovalPublisher(pub)
	ctxJSON, _ := json.Marshal(secretApprovalContext{Ref: "op://Prod/k", Purpose: "x", Agent: "a"})
	store.approvals["secret-4"] = &coordinator.ApprovalRequestRecord{
		ID: "secret-4", Type: "secret", Status: "pending", ContextJSON: string(ctxJSON),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/approvals/secret-4/approve", nil)
	w := httptest.NewRecorder()
	s.handleApproval(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if store.approvals["secret-4"].Status != "approved" {
		t.Fatalf("expected approved, got %q", store.approvals["secret-4"].Status)
	}
	if !pub.called || pub.approvalType != "secret-resolved" {
		t.Fatalf("expected a 'secret-resolved' confirmation publish, got called=%v type=%q", pub.called, pub.approvalType)
	}
	if !strings.Contains(string(pub.payload), "Approved") || strings.Contains(string(pub.payload), "op://Prod/k") == false {
		t.Errorf("confirmation should say Approved + the ref: %s", pub.payload)
	}
}

// TestResolveSecretApproval_IgnoresNonSecret guards the live task-approval path:
// resolveSecretApproval must decline anything that is not a secret request so it
// falls through to ProcessApprovalRequest unchanged.
func TestResolveSecretApproval_IgnoresNonSecret(t *testing.T) {
	s, _ := newSecretApprovalServer()
	rec := &coordinator.ApprovalRequestRecord{ID: "task-1", Type: "merge", Status: "pending"}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/approvals/task-1/approve", nil)
	if s.resolveSecretApproval(w, req, rec, "approve", "operator") {
		t.Fatal("resolveSecretApproval must NOT handle a non-secret (task) approval")
	}
}
