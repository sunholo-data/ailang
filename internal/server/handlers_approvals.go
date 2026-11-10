package server

import (
	"net/http"
	"strings"
)

// The approval routes the dashboard keeps are the SECRET-approval ones
// (M-SECRET-REMOTE-APPROVAL-WIRING): cloud jobs file a request, poll it, and
// the operator decides from the ntfy push.
//
//	POST /api/approvals                    intake  (internal/secrets/cloud_approver.go)
//	GET  /api/approvals/{id}               poll    (same client)
//	POST /api/approvals/{id}/approve|reject decision (ntfy action: signed ?token=)
//
// Coordinator TASK approvals are not decided over this API any more: the
// approval-card UI was the only HTTP caller. `ailang coordinator approve|reject`
// (add --remote gcp for the cloud plane) runs the same processor
// (coordinator.ProcessApprovalRequest) against the store directly. The list
// (GET /api/approvals) and history (GET /api/approvals/history) the UI read are
// gone with it (m-dashboard-headless-api).

// handleApprovals serves POST /api/approvals — secret-approval intake.
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handleSecretApprovalIntake(w, r)
}

// handleApproval serves GET /api/approvals/{id} (status poll) and
// POST /api/approvals/{id}/approve|reject (the operator's decision).
func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/approvals/")

	if r.Method == http.MethodGet {
		if path != "" && !strings.Contains(path, "/") {
			s.handleSecretApprovalStatus(w, r, path)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	approvalID, action, _ := strings.Cut(path, "/")
	if approvalID == "" || (action != "approve" && action != "reject") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Secret-approval token auth (M-SECRET-EFFECT): the iPhone ntfy action
	// buttons authenticate with a signed single-use ?token=. If a token is
	// present (and token auth is enabled) it must be valid for this exact
	// approval+action; a present-but-invalid token is rejected rather than
	// falling through to another auth path. Without a token the decision needs
	// an authenticated Approver session (#920) — on a non-loopback bind the
	// dashboard gate (read_auth.go) has already demanded a credential too.
	if handled, ok := s.checkSecretApprovalToken(r, approvalID, action); handled && !ok {
		http.Error(w, "Invalid or expired approval token", http.StatusUnauthorized)
		return
	} else if !handled {
		if !s.approverSessionAuthorized(w, r) {
			return
		}
	}

	if s.approvalStore == nil {
		http.Error(w, "approval store not configured", http.StatusServiceUnavailable)
		return
	}
	rec, err := s.approvalStore.GetApprovalRequest(r.Context(), approvalID)
	if err != nil || rec == nil {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}
	if !s.resolveSecretApproval(w, r, rec, action, "operator") {
		// A task (merge/handoff) approval: decided with the CLI, which runs the
		// merge + handoff processor. Say so rather than half-doing it here.
		http.Error(w, "not a secret approval: decide task approvals with `ailang coordinator "+action+" <task-id>`", http.StatusConflict)
	}
}
