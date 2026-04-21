package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// ExecSessionRequest represents a request to create an exec session
type ExecSessionRequest struct {
	SessionID string `json:"session_id"`
	Workspace string `json:"workspace,omitempty"`
	Provider  string `json:"provider,omitempty"`
}

// ExecEventRequest represents a request to store an exec event
type ExecEventRequest struct {
	SessionID  string `json:"session_id"`
	StreamType string `json:"stream_type"` // text, tool_use, tool_result, turn_start, turn_end, error
	TurnNum    int    `json:"turn_num,omitempty"`
	Text       string `json:"text,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolOutput string `json:"tool_output,omitempty"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

// handleExecSessions creates an exec session record
// POST /api/exec/sessions
func (s *Server) handleExecSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.coordStoreRaw == nil {
		http.Error(w, "Store not configured", http.StatusServiceUnavailable)
		return
	}

	var req ExecSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Prefix session ID to make it a valid task ID
	taskID := "exec-" + req.SessionID
	if len(req.SessionID) > 8 {
		taskID = "exec-" + req.SessionID[:8]
	}

	ctx := r.Context()

	// Create a task record for this exec session
	task := &coordinator.TaskRecord{
		ID:        taskID,
		Status:    coordinator.TaskStatusRunning,
		Workspace: req.Workspace,
		Provider:  req.Provider,
		CreatedAt: time.Now(),
	}

	if err := s.coordStoreRaw.CreateTask(ctx, task); err != nil {
		// Log but don't fail - session might already exist
		log.Printf("Warning: failed to create exec task record: %v", err)
	}

	// Store a session_start event
	event := &coordinator.TaskEventRecord{
		TaskID:     taskID,
		StreamType: "session_start",
		Text:       "Session started",
		CreatedAt:  time.Now(),
	}

	if err := s.coordStoreRaw.StoreTaskEvent(ctx, event); err != nil {
		log.Printf("Warning: failed to store session_start event: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"task_id":    taskID,
		"session_id": req.SessionID,
		"status":     "created",
	})
}

// handleExecEvents stores an exec event
// POST /api/exec/events
func (s *Server) handleExecEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.coordStoreRaw == nil {
		http.Error(w, "Store not configured", http.StatusServiceUnavailable)
		return
	}

	var req ExecEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Build task ID from session ID
	taskID := "exec-" + req.SessionID
	if len(req.SessionID) > 8 {
		taskID = "exec-" + req.SessionID[:8]
	}

	ctx := r.Context()

	// Build event record
	event := &coordinator.TaskEventRecord{
		TaskID:     taskID,
		StreamType: req.StreamType,
		TurnNum:    req.TurnNum,
		Text:       req.Text,
		ToolName:   req.ToolName,
		ToolInput:  req.ToolInput,
		ToolOutput: req.ToolOutput,
		ErrorMsg:   req.ErrorMsg,
		CreatedAt:  time.Now(),
	}

	if err := s.coordStoreRaw.StoreTaskEvent(ctx, event); err != nil {
		log.Printf("Failed to store exec event: %v", err)
		http.Error(w, "Failed to store event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "stored",
		"task_id": taskID,
	})
}

// handleExecSessionEvents retrieves events for an exec session
// GET /api/exec/sessions/{session_id}/events
func (s *Server) handleExecSessionEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.coordStoreRaw == nil {
		http.Error(w, "Store not configured", http.StatusServiceUnavailable)
		return
	}

	// Extract session ID from path
	// Expected: /api/exec/sessions/{session_id}/events
	path := r.URL.Path
	const prefix = "/api/exec/sessions/"
	const suffix = "/events"

	if !hasPrefix(path, prefix) || !hasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	sessionID := path[len(prefix) : len(path)-len(suffix)]
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Build task ID from session ID
	taskID := "exec-" + sessionID
	if len(sessionID) > 8 {
		taskID = "exec-" + sessionID[:8]
	}

	ctx := r.Context()

	events, err := s.coordStoreRaw.GetTaskEvents(ctx, taskID, 1000)
	if err != nil {
		log.Printf("Failed to get exec events: %v", err)
		http.Error(w, "Failed to get events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id": taskID,
		"events":  events,
		"count":   len(events),
	})
}

// hasPrefix is a simple string prefix check
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// hasSuffix is a simple string suffix check
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// ExecEventStorer is a helper interface for storing exec events
// Can be used by the Claude Code hooks endpoint
type ExecEventStorer interface {
	StoreTaskEvent(ctx context.Context, event *coordinator.TaskEventRecord) error
}

// GetExecEventStorer returns the event storer for external use
func (s *Server) GetExecEventStorer() ExecEventStorer {
	return s.coordStoreRaw
}
