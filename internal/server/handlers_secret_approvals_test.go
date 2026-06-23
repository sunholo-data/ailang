package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

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
