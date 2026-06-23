package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// secretApprovalTokenTTL bounds how long an Approve/Deny action token is valid.
// Slightly longer than the executor's default 5-minute poll deadline so the
// token never expires before the request itself times out.
const secretApprovalTokenTTL = 10 * time.Minute

// ApprovalPublisher publishes a secret-approval notification to the approvals
// topic. *pubsub.Publisher satisfies it structurally; it is injected in cloud
// mode so this package need not import internal/pubsub.
type ApprovalPublisher interface {
	PublishApproval(ctx context.Context, approvalID, approvalType, agentID string, notificationJSON []byte) error
}

// SetApprovalPublisher wires the approvals-topic publisher (cloud mode only).
func (s *Server) SetApprovalPublisher(p ApprovalPublisher) { s.approvalPublisher = p }

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
	// Terminal-state lock: a secret approval resolves exactly once. The Approve
	// and Deny buttons carry different single-use tokens, so without this a
	// second tap (the *other* button) would flip an already-decided request —
	// corrupting the audit record. Reject resolution of a non-pending request.
	// (Read-then-write isn't atomic, but human taps are seconds apart and the
	// per-token single-use guard serialises same-button retries.)
	if rec.Status != "" && rec.Status != "pending" {
		writeJSONStatus(w, http.StatusConflict, map[string]string{
			"status":      "already_resolved",
			"resolution":  rec.Status,
			"resolved_by": rec.ResolvedBy,
		})
		return true
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
	// Confirmation push: ntfy action buttons can't reflect their own outcome, so
	// send a follow-up so the operator sees the decision landed.
	s.publishSecretApprovalResolved(r.Context(), rec, status)
	writeJSONStatus(w, http.StatusOK, map[string]string{"status": "success", "action": action})
	return true
}

// publishSecretApprovalResolved sends the value-free confirmation push after a
// secret approval is decided. Best-effort: no publisher → no-op.
func (s *Server) publishSecretApprovalResolved(ctx context.Context, rec *coordinator.ApprovalRequestRecord, decision string) {
	if s.approvalPublisher == nil {
		return
	}
	var sc secretApprovalContext
	_ = json.Unmarshal([]byte(rec.ContextJSON), &sc)
	payload, err := json.Marshal(coordinator.BuildSecretApprovalResolvedNotification(sc.Ref, decision))
	if err != nil {
		return
	}
	if err := s.approvalPublisher.PublishApproval(ctx, rec.ID, "secret-resolved", sc.Agent, payload); err != nil {
		log.Printf("secret approval: confirmation publish for %s failed: %v", rec.ID, err)
	}
}

// publishSecretApprovalRequested builds the value-free approval notification
// (with signed Approve/Deny action URLs) and publishes it to the approvals
// topic, where the coordinator's /pubsub/push bridge forwards it to ntfy.
//
// It is best-effort: with no publisher (not cloud mode), no signer (token auth
// off), or no AILANG_APPROVAL_BASE_URL, it no-ops — the executor still learns
// the decision by polling, the phone push is simply skipped.
func (s *Server) publishSecretApprovalRequested(ctx context.Context, rec *coordinator.ApprovalRequestRecord) {
	if s.approvalPublisher == nil || s.secretTokenSigner == nil {
		return
	}
	baseURL := os.Getenv("AILANG_APPROVAL_BASE_URL")
	if baseURL == "" {
		log.Printf("secret approval: AILANG_APPROVAL_BASE_URL unset — skipping push for %s", rec.ID)
		return
	}

	var sc secretApprovalContext
	_ = json.Unmarshal([]byte(rec.ContextJSON), &sc)

	notif, err := coordinator.BuildSecretApprovalNotification(
		&coordinator.ApprovalRequest{
			ID:            rec.ID,
			TaskID:        rec.TaskID,
			Type:          coordinator.ApprovalTypeSecret,
			SecretRef:     sc.Ref,
			SecretPurpose: sc.Purpose,
			AgentID:       sc.Agent,
		},
		baseURL, s.secretTokenSigner, secretApprovalTokenTTL,
	)
	if err != nil {
		log.Printf("secret approval: build notification for %s failed: %v", rec.ID, err)
		return
	}
	payload, err := json.Marshal(notif)
	if err != nil {
		return
	}
	if err := s.approvalPublisher.PublishApproval(ctx, rec.ID, rec.Type, sc.Agent, payload); err != nil {
		log.Printf("secret approval: publish for %s failed: %v", rec.ID, err)
	}
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
