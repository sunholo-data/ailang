package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/websocket"
)

// CoordinatorApprovalStore provides approval request operations
type CoordinatorApprovalStore interface {
	GetApprovalRequest(ctx context.Context, id string) (*coordinator.ApprovalRequestRecord, error)
	ListPendingApprovals(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error)
	ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error
}

// SetApprovalStore sets the coordinator approval store
func (s *Server) SetApprovalStore(store CoordinatorApprovalStore) {
	s.approvalStore = store
}

// handleCoordinatorApproval handles approve/reject requests for coordinator tasks
// POST /api/coordinator/approve/{id}
// POST /api/coordinator/reject/{id}
func (s *Server) handleCoordinatorApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/coordinator/{action}/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/coordinator/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid path: expected /api/coordinator/{action}/{id}", http.StatusBadRequest)
		return
	}

	action := parts[0]
	id := parts[1]

	if action != "approve" && action != "reject" {
		http.Error(w, "Invalid action: expected 'approve' or 'reject'", http.StatusBadRequest)
		return
	}

	if s.approvalStore == nil {
		http.Error(w, "Coordinator approval store not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()

	// Check if approval request exists
	req, err := s.approvalStore.GetApprovalRequest(ctx, id)
	if err != nil {
		log.Printf("Failed to get approval request %s: %v", id, err)
		http.Error(w, "Approval request not found", http.StatusNotFound)
		return
	}

	if req.Status != "pending" {
		http.Error(w, "Approval request already resolved", http.StatusConflict)
		return
	}

	// Resolve the request
	status := "approved"
	if action == "reject" {
		status = "rejected"
	}

	if err := s.approvalStore.ResolveApprovalRequest(ctx, id, status, "dashboard-user"); err != nil {
		log.Printf("Failed to resolve approval request %s: %v", id, err)
		http.Error(w, "Failed to resolve approval request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"action":    action,
		"id":        id,
		"timestamp": time.Now().Unix(),
	}); err != nil {
		log.Printf("Failed to encode approval response: %v", err)
	}
}

// handleCoordinatorPendingApprovals lists pending approval requests
// GET /api/coordinator/pending
func (s *Server) handleCoordinatorPendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.approvalStore == nil {
		// Return empty list if no store configured
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]interface{}{}); err != nil {
			log.Printf("Failed to encode empty pending approvals: %v", err)
		}
		return
	}

	ctx := r.Context()
	pending, err := s.approvalStore.ListPendingApprovals(ctx)
	if err != nil {
		log.Printf("Failed to list pending approvals: %v", err)
		http.Error(w, "Failed to list pending approvals", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pending); err != nil {
		log.Printf("Failed to encode pending approvals: %v", err)
	}
}

// TaskStreamEventRequest is the JSON body for task stream events
type TaskStreamEventRequest struct {
	TaskID      string  `json:"task_id"`
	ThreadID    string  `json:"thread_id,omitempty"`
	StreamType  string  `json:"stream_type"`
	TurnNum     int     `json:"turn_num,omitempty"`
	Text        string  `json:"text,omitempty"`
	ToolName    string  `json:"tool_name,omitempty"`
	ToolInput   string  `json:"tool_input,omitempty"`
	ToolOutput  string  `json:"tool_output,omitempty"`
	Status      string  `json:"status,omitempty"`
	TokensIn    int     `json:"tokens_in,omitempty"`
	TokensOut   int     `json:"tokens_out,omitempty"`
	Cost        float64 `json:"cost,omitempty"`
	DurationSec int     `json:"duration_sec,omitempty"`
	ErrorMsg    string  `json:"error_msg,omitempty"`
}

// handleCoordinatorTaskEvents receives task stream events from the daemon
// POST /api/coordinator/events
func (s *Server) handleCoordinatorTaskEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TaskStreamEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Convert to WebSocket event format
	streamType := websocket.TaskStreamEventType(req.StreamType)

	event := &websocket.TaskStreamEvent{
		TaskID:      req.TaskID,
		ThreadID:    req.ThreadID,
		StreamType:  streamType,
		TurnNum:     req.TurnNum,
		Text:        req.Text,
		ToolName:    req.ToolName,
		ToolInput:   req.ToolInput,
		ToolOutput:  req.ToolOutput,
		Status:      req.Status,
		TokensIn:    req.TokensIn,
		TokensOut:   req.TokensOut,
		Cost:        req.Cost,
		DurationSec: req.DurationSec,
		ErrorMsg:    req.ErrorMsg,
	}

	// Broadcast to all WebSocket clients
	s.wsServer.BroadcastTaskEvent(event)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	}); err != nil {
		log.Printf("Failed to encode event response: %v", err)
	}
}
