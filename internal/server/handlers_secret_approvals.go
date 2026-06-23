package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// handlers_secret_approvals.go — the coordinator-side intake/status/resolve for
// secret approvals (M-SECRET-REMOTE-APPROVAL-WIRING M2).
//
// A CloudSecretApprover in an agent-executor job calls POST /api/approvals when
// secret() is invoked, then polls GET /api/approvals/{id} for the decision. The
// operator's phone (M4) resolves it via POST /api/approvals/{id}/approve|reject
// with a signed single-use token. Secret approvals are deliberately resolved
// without the task merge/handoff machinery (ProcessApprovalRequest).
//
// Nothing here ever carries a resolved secret value — only the reference,
// purpose, and requesting agent.

// secretApprovalContext is JSON-serialized into ApprovalRequestRecord.ContextJSON
// for Type=="secret" records. Value-free by construction.
type secretApprovalContext struct {
	Ref     string `json:"ref"`
	Purpose string `json:"purpose,omitempty"`
	Agent   string `json:"agent,omitempty"`
}

// secretIntakeRequest is the POST /api/approvals body from CloudSecretApprover.
type secretIntakeRequest struct {
	Ref     string `json:"ref"`
	Purpose string `json:"purpose"`
	Agent   string `json:"agent"`
	Task    string `json:"task"`
}

// handleSecretApprovalIntake creates a pending secret-approval request and
// returns its id. Reached via POST /api/approvals (GET on that path lists
// approvals — see handleApprovals).
func (s *Server) handleSecretApprovalIntake(w http.ResponseWriter, r *http.Request) {
	if s.approvalStore == nil {
		http.Error(w, "approval store not configured", http.StatusServiceUnavailable)
		return
	}
	var body secretIntakeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Ref == "" {
		http.Error(w, "ref is required", http.StatusBadRequest)
		return
	}

	ctxJSON, _ := json.Marshal(secretApprovalContext{Ref: body.Ref, Purpose: body.Purpose, Agent: body.Agent})
	agent := body.Agent
	if agent == "" {
		agent = "unknown agent"
	}
	desc := fmt.Sprintf("%s requests %s", agent, body.Ref)
	if body.Purpose != "" && body.Purpose != body.Ref {
		desc += " — " + body.Purpose
	}

	rec := &coordinator.ApprovalRequestRecord{
		ID:          fmt.Sprintf("secret-%d", time.Now().UnixNano()),
		TaskID:      body.Task,
		Type:        "secret",
		Description: desc,
		ContextJSON: string(ctxJSON),
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	if err := s.approvalStore.CreateApprovalRequest(r.Context(), rec); err != nil {
		http.Error(w, fmt.Sprintf("could not create approval: %v", err), http.StatusInternalServerError)
		return
	}

	s.publishSecretApprovalRequested(r.Context(), rec)

	writeJSONStatus(w, http.StatusOK, map[string]string{"id": rec.ID})
}

// handleSecretApprovalStatus returns the current status of one approval.
// GET /api/approvals/{id} — polled by CloudSecretApprover until a decision.
func (s *Server) handleSecretApprovalStatus(w http.ResponseWriter, r *http.Request, id string) {
	if s.approvalStore == nil {
		http.Error(w, "approval store not configured", http.StatusServiceUnavailable)
		return
	}
	rec, err := s.approvalStore.GetApprovalRequest(r.Context(), id)
	if err != nil || rec == nil {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}
	writeJSONStatus(w, http.StatusOK, map[string]string{
		"id":     rec.ID,
		"status": rec.Status, // pending | approved | rejected | timeout
		"type":   rec.Type,
	})
}

// resolveSecretApproval resolves a Type=="secret" approval directly via the
// store, bypassing the task merge/handoff path. It returns true when it has
// handled the request (i.e. rec is a secret approval); callers fall through to
// the task path when it returns false.
func (s *Server) resolveSecretApproval(w http.ResponseWriter, r *http.Request, rec *coordinator.ApprovalRequestRecord, action, resolvedBy string) bool {
	if rec == nil || rec.Type != "secret" {
		return false
	}
	status := "approved"
	if action == "reject" {
		status = "rejected"
	}
	if resolvedBy == "" {
		resolvedBy = "operator"
	}
	if err := s.approvalStore.ResolveApprovalRequest(r.Context(), rec.ID, status, resolvedBy); err != nil {
		http.Error(w, fmt.Sprintf("could not %s: %v", action, err), http.StatusInternalServerError)
		return true
	}
	writeJSONStatus(w, http.StatusOK, map[string]string{"status": "success", "action": action})
	return true
}

// publishSecretApprovalRequested is the seam where a kind=approval Pub/Sub event
// will be published to drive the iPhone ntfy push. It is intentionally a no-op
// until the dedicated ${prefix}-approvals topic + publisher land in M3, so
// intake works today (the executor polls for the decision) without it.
func (s *Server) publishSecretApprovalRequested(_ context.Context, _ *coordinator.ApprovalRequestRecord) {
	// M3: publish {kind:"approval", approval_id, approval_type:"secret"} plus the
	// BuildSecretApprovalNotification payload to the ${prefix}-approvals topic.
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
