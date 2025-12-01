package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// GET /api/approvals?status={status} - Get approvals by status
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	approvals, err := s.store.GetApprovalsByStatus(status, 50)
	if err != nil {
		http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(approvals); err != nil {
		log.Printf("Failed to encode approvals response: %v", err)
	}
}

// POST /api/approvals/{id}/approve - Approve an approval request
// POST /api/approvals/{id}/reject - Reject an approval request
func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract approval ID and action from path
	path := r.URL.Path[len("/api/approvals/"):]
	var approvalID, action string

	// Parse path: {id}/approve or {id}/reject
	for i, ch := range path {
		if ch == '/' {
			approvalID = path[:i]
			action = path[i+1:]
			break
		}
	}

	if approvalID == "" || (action != "approve" && action != "reject") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Parse request body for review notes
	var body struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform action
	var err error
	if action == "approve" {
		err = s.store.ApproveApproval(approvalID, "user", body.Notes, 24*time.Hour)
	} else {
		err = s.store.RejectApproval(approvalID, "user", body.Notes)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to %s approval: %v", action, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"action":  action,
		"message": fmt.Sprintf("Approval %s successfully", action+"d"),
	}); err != nil {
		log.Printf("Failed to encode approval response: %v", err)
	}
}

// handleApprovalHistory returns approval history entries
// GET /api/approvals/history?thread_id={id}&limit={n}
func (s *Server) handleApprovalHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	threadID := r.URL.Query().Get("thread_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	entries, err := s.store.GetApprovalHistory(threadID, limit)
	if err != nil {
		log.Printf("Failed to get approval history: %v", err)
		http.Error(w, "Failed to get approval history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Printf("Failed to encode approval history response: %v", err)
	}
}
