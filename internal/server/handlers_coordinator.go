package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/websocket"
)

// CoordinatorApprovalStore provides approval request operations
type CoordinatorApprovalStore interface {
	// CreateApprovalRequest persists a new pending request. Used by the secret
	// approval intake (M-SECRET-REMOTE-APPROVAL-WIRING). Both the SQLite and
	// Firestore coordinator stores already implement this.
	CreateApprovalRequest(ctx context.Context, req *coordinator.ApprovalRequestRecord) error
	GetApprovalRequest(ctx context.Context, id string) (*coordinator.ApprovalRequestRecord, error)
	ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error
}

// SetApprovalStore sets the coordinator approval store
func (s *Server) SetApprovalStore(store CoordinatorApprovalStore) {
	s.approvalStore = store
}

// CoordinatorTaskEventStore provides task event operations for historical replay
type CoordinatorTaskEventStore interface {
	GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*coordinator.TaskEventRecord, error)
	GetTask(ctx context.Context, id string) (*coordinator.TaskRecord, error)
}

// SetTaskEventStore sets the coordinator task event store
func (s *Server) SetTaskEventStore(store CoordinatorTaskEventStore) {
	s.taskEventStore = store
}

// SetCoordinatorStoreRaw sets the raw coordinator store for Control Plane queries
func (s *Server) SetCoordinatorStoreRaw(store coordinator.Store) {
	s.coordStoreRaw = store
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
		coordinator.SanitizeLog(streamType), coordinator.SanitizeLog(req.TaskID), s.wsServer.GetConnectionCount())

	// Broadcast to all WebSocket clients
	s.wsServer.BroadcastTaskEvent(event)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	}); err != nil {
		log.Printf("Failed to encode event response: %v", err)
	}
}

// handleCoordinatorTaskEvents_ serves a task's recorded events.
// GET /api/coordinator/tasks/{id}/events[?format=json|text|summary]
// Read by `ailang coordinator` browse (cmd/ailang/coordinator_browse.go). The
// .../diff action the UI's approval card used is gone; `ailang coordinator
// approve <id>` shows the diff.
func (s *Server) handleCoordinatorTaskEvents_(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/coordinator/tasks/{id}/events
	path := strings.TrimPrefix(r.URL.Path, "/api/coordinator/tasks/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "events" {
		http.Error(w, "Invalid path: expected /api/coordinator/tasks/{id}/events", http.StatusNotFound)
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
		turns, err := countTurns(events)
		if err != nil {
			log.Printf("count turns for %s: %v", taskID, err)
			http.Error(w, "turn-count transform unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp := map[string]interface{}{
			"task_id":      taskID,
			"format":       "text",
			"content":      text,
			"total_events": len(events),
			"total_turns":  turns,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode text events: %v", err)
		}
		return

	case "summary":
		// Return compact summary
		// The summary transform is AILANG, and it is the only implementation.
		bridge := GetAILANGBridge()
		summary, err := bridge.SummarizeEvents(events)
		if err != nil {
			log.Printf("summarize events for %s: %v", taskID, err)
			http.Error(w, "summary transform unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		turns, err := countTurns(events)
		if err != nil {
			log.Printf("count turns for %s: %v", taskID, err)
			http.Error(w, "summary transform unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		resp := map[string]interface{}{
			"task_id":       taskID,
			"format":        "summary",
			"content":       summary,
			"total_events":  len(events),
			"total_turns":   turns,
			"ailang_active": bridge.Ready() == nil,
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

// countTurns counts unique turn numbers in the event list, in AILANG.
//
// The inline Go loop that used to sit here was the THIRD copy of this rule —
// alongside coordinator.CountTurns and event_formatter.ail — reachable whenever
// AILANG_DASHBOARD was unset, which in production was always.
func countTurns(events []*coordinator.TaskEventRecord) (int, error) {
	return GetAILANGBridge().CountTurns(events)
}
