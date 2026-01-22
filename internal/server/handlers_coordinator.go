package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/websocket"
)

// CoordinatorApprovalStore provides approval request operations
type CoordinatorApprovalStore interface {
	GetApprovalRequest(ctx context.Context, id string) (*coordinator.ApprovalRequestRecord, error)
	ListPendingApprovals(ctx context.Context) ([]*coordinator.ApprovalRequestRecord, error)
	ListResolvedApprovals(ctx context.Context, limit int) ([]*coordinator.ApprovalRequestRecord, error)
	ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error
}

// SetApprovalStore sets the coordinator approval store
func (s *Server) SetApprovalStore(store CoordinatorApprovalStore) {
	s.approvalStore = store
}

// CoordinatorTaskEventStore provides task event operations for historical replay
type CoordinatorTaskEventStore interface {
	GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*coordinator.TaskEventRecord, error)
	ListTasks(ctx context.Context, filter *coordinator.TaskFilter) ([]*coordinator.TaskRecord, error)
	GetTask(ctx context.Context, id string) (*coordinator.TaskRecord, error)
}

// SetTaskEventStore sets the coordinator task event store
func (s *Server) SetTaskEventStore(store CoordinatorTaskEventStore) {
	s.taskEventStore = store
}

// SetCoordinatorStore sets the coordinator store for statistics
func (s *Server) SetCoordinatorStore(store CoordinatorStore) {
	s.coordStore = store
}

// SetCoordinatorStoreRaw sets the raw coordinator store for Control Plane queries
func (s *Server) SetCoordinatorStoreRaw(store coordinator.Store) {
	s.coordStoreRaw = store
}

// handleCoordinatorRunningTasks returns the list of running/pending tasks
// GET /api/coordinator/running
func (s *Server) handleCoordinatorRunningTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.taskEventStore == nil {
		// Return empty list if no store configured
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]interface{}{}); err != nil {
			log.Printf("Failed to encode empty running tasks: %v", err)
		}
		return
	}

	ctx := r.Context()

	// Fetch running and pending tasks
	filter := &coordinator.TaskFilter{
		Status:    []coordinator.TaskStatus{coordinator.TaskStatusRunning, coordinator.TaskStatusPending, coordinator.TaskStatusQueued},
		OrderBy:   "created_at",
		OrderDesc: true,
		Limit:     50,
	}

	tasks, err := s.taskEventStore.ListTasks(ctx, filter)
	if err != nil {
		log.Printf("Failed to list running tasks: %v", err)
		http.Error(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}

	// Convert to JSON-friendly format
	var result []map[string]interface{}
	for _, t := range tasks {
		task := map[string]interface{}{
			"id":         t.ID,
			"title":      t.Title,
			"status":     t.Status,
			"type":       t.Type,
			"created_at": t.CreatedAt,
		}
		if t.Provider != "" {
			task["provider"] = t.Provider
		}
		if t.ThreadID != "" {
			task["thread_id"] = t.ThreadID
		}
		if t.StartedAt != nil {
			task["started_at"] = t.StartedAt
		}
		result = append(result, task)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode running tasks: %v", err)
	}
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

// handleCoordinatorPendingApprovals lists pending approval requests with enriched task info
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

	// Enrich approvals with task info (worktree_path, files_changed)
	var result []map[string]interface{}
	for _, p := range pending {
		approval := map[string]interface{}{
			"id":          p.ID,
			"task_id":     p.TaskID,
			"type":        p.Type,
			"description": p.Description,
			"status":      p.Status,
			"created_at":  p.CreatedAt,
			"auto_reject": p.AutoReject,
		}
		if p.ContextJSON != "" {
			approval["context_json"] = p.ContextJSON
		}
		if p.TimeoutAt != nil {
			approval["timeout_at"] = p.TimeoutAt
		}

		// Get associated task for worktree info
		if s.taskEventStore != nil {
			task, err := s.taskEventStore.GetTask(ctx, p.TaskID)
			if err == nil && task != nil {
				if task.WorktreePath != "" {
					approval["worktree_path"] = task.WorktreePath
				}
				if task.SessionID != "" {
					approval["session_id"] = task.SessionID
				}
				approval["task_title"] = task.Title
				approval["task_status"] = task.Status
				if task.Provider != "" {
					approval["provider"] = task.Provider
				}
			}
		}

		result = append(result, approval)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
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

	// Debug logging
	log.Printf("[DEBUG] Server received task event: type=%s task=%s (wsClients=%d)",
		streamType, req.TaskID, s.wsServer.GetConnectionCount())

	// Broadcast to all WebSocket clients
	s.wsServer.BroadcastTaskEvent(event)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	}); err != nil {
		log.Printf("Failed to encode event response: %v", err)
	}
}

// handleCoordinatorTaskEvents_ handles task-specific endpoints
// GET /api/coordinator/tasks/{id}/events - fetch historical task events
// GET /api/coordinator/tasks/{id}/diff - fetch git diff for task worktree
func (s *Server) handleCoordinatorTaskEvents_(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/coordinator/tasks/{id}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/api/coordinator/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path: expected /api/coordinator/tasks/{id}/{action}", http.StatusBadRequest)
		return
	}

	action := parts[1]
	if action != "events" && action != "diff" {
		http.Error(w, "Invalid action: expected 'events' or 'diff'", http.StatusBadRequest)
		return
	}

	// Route to specific handler
	if action == "diff" {
		s.handleTaskDiff(w, r, parts[0])
		return
	}

	taskID := parts[0]
	if taskID == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	if s.taskEventStore == nil {
		// Return empty response if no store configured
		w.Header().Set("Content-Type", "application/json")
		resp := &coordinator.EventsResponse{
			TaskID:      taskID,
			TotalEvents: 0,
			TotalTurns:  0,
			Events:      []*coordinator.TaskEventRecord{},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode empty events: %v", err)
		}
		return
	}

	ctx := r.Context()

	// Parse query parameters
	query := r.URL.Query()

	// Limit parameter (default 500)
	limit := 500
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Verify task exists first
	task, err := s.taskEventStore.GetTask(ctx, taskID)
	if err != nil || task == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Fetch events from database
	events, err := s.taskEventStore.GetTaskEvents(ctx, taskID, limit)
	if err != nil {
		log.Printf("Failed to get task events for %s: %v", taskID, err)
		http.Error(w, "Failed to get task events", http.StatusInternalServerError)
		return
	}

	// Build format options from query params
	opts := coordinator.DefaultFormatOptions()

	// Turn filter
	if turnStr := query.Get("turn"); turnStr != "" {
		if turn, err := strconv.Atoi(turnStr); err == nil && turn > 0 {
			opts.TurnFilter = turn
		}
	}

	// Type filter (comma-separated)
	if typeFilter := query.Get("type"); typeFilter != "" {
		opts.TypeFilter = strings.Split(typeFilter, ",")
	}

	// Check format parameter (json, text, summary)
	format := query.Get("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "text":
		// Return formatted text for CLI/human consumption
		text := coordinator.FormatEventsAsText(events, opts)
		resp := map[string]interface{}{
			"task_id":      taskID,
			"format":       "text",
			"content":      text,
			"total_events": len(events),
			"total_turns":  countTurns(events),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode text events: %v", err)
		}
		return

	case "summary":
		// Return compact summary
		// Uses AILANG implementation when AILANG_DASHBOARD=1
		bridge := GetAILANGBridge()
		summary := bridge.SummarizeEvents(events)
		resp := map[string]interface{}{
			"task_id":       taskID,
			"format":        "summary",
			"content":       summary,
			"total_events":  len(events),
			"total_turns":   countTurns(events),
			"ailang_active": bridge.IsEnabled(),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode summary events: %v", err)
		}
		return

	default: // "json" or unrecognized -> default to JSON
		resp, err := coordinator.FormatEventsAsJSON(taskID, events, opts)
		if err != nil {
			log.Printf("Failed to format events for %s: %v", taskID, err)
			http.Error(w, "Failed to format events", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode task events: %v", err)
		}
	}
}

// countTurns counts unique turn numbers in event list.
// Uses AILANG implementation when AILANG_DASHBOARD=1.
func countTurns(events []*coordinator.TaskEventRecord) int {
	bridge := GetAILANGBridge()
	if bridge.IsEnabled() {
		return bridge.CountTurns(events)
	}
	// Go fallback
	turns := make(map[int]bool)
	for _, e := range events {
		if e.TurnNum > 0 {
			turns[e.TurnNum] = true
		}
	}
	return len(turns)
}

// handleTaskDiff returns the git diff for a task's worktree
// GET /api/coordinator/tasks/{id}/diff
func (s *Server) handleTaskDiff(w http.ResponseWriter, r *http.Request, taskID string) {
	if taskID == "" {
		http.Error(w, "Task ID required", http.StatusBadRequest)
		return
	}

	if s.taskEventStore == nil {
		http.Error(w, "Task store not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()

	// Get the task to find worktree path
	task, err := s.taskEventStore.GetTask(ctx, taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if task.WorktreePath == "" {
		http.Error(w, "Task has no worktree", http.StatusNotFound)
		return
	}

	// Check if worktree directory exists
	if _, err := os.Stat(task.WorktreePath); os.IsNotExist(err) {
		// Worktree was deleted - return empty diff with explanation
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"task_id":       taskID,
			"worktree_path": task.WorktreePath,
			"diff":          "",
			"error":         "Worktree has been deleted",
		}); err != nil {
			log.Printf("Failed to encode diff response: %v", err)
		}
		return
	}

	// Get the diff using coordinator's GetWorktreeDiff
	// Pass task.BaseCommit (stable) and task.BaseBranch (fallback)
	// BaseCommit is preferred as the branch may have moved since worktree creation
	diff, err := coordinator.GetWorktreeDiff(ctx, task.WorktreePath, task.BaseBranch, task.BaseCommit)
	if err != nil {
		log.Printf("Failed to get diff for task %s: %v", taskID, err)
		http.Error(w, "Failed to get diff", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id":       taskID,
		"worktree_path": task.WorktreePath,
		"diff":          diff,
	}); err != nil {
		log.Printf("Failed to encode diff response: %v", err)
	}
}

// handleTasksAlias provides a cleaner API path for task operations
// GET /api/tasks/{id}/events -> /api/coordinator/tasks/{id}/events
func (s *Server) handleTasksAlias(w http.ResponseWriter, r *http.Request) {
	// Rewrite path from /api/tasks/{id}/events to /api/coordinator/tasks/{id}/events
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	r.URL.Path = "/api/coordinator/tasks/" + path
	s.handleCoordinatorTaskEvents_(w, r)
}
