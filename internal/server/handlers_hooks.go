// Package server provides HTTP handlers for the collaboration hub.
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// HookEvent represents a Claude Code hook event from the telemetry script.
// These events are sent by ~/.claude/hooks/claude_telemetry.sh.
type HookEvent struct {
	Event         string          `json:"event"`
	SessionID     string          `json:"session_id"`
	Workspace     string          `json:"workspace,omitempty"`
	ClaudeVersion string          `json:"claude_version,omitempty"`
	ToolName      string          `json:"tool_name,omitempty"`
	ToolUseID     string          `json:"tool_use_id,omitempty"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse  json.RawMessage `json:"tool_response,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// handleObservatoryHooks handles POST /api/observatory/hooks
// This endpoint receives Claude Code hook events and stores them in the sessions table.
// Events are used to enrich OTEL spans with workspace information.
func (s *Server) handleObservatoryHooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory not configured", http.StatusServiceUnavailable)
		return
	}

	var event HookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("hooks: failed to parse event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch event.Event {
	case "SessionStart":
		// Upsert session with workspace information
		if event.SessionID == "" {
			log.Printf("hooks: SessionStart missing session_id")
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}
		if event.Workspace == "" {
			log.Printf("hooks: SessionStart missing workspace")
			http.Error(w, "workspace required", http.StatusBadRequest)
			return
		}

		if err := s.obsBackend.UpsertSession(ctx, event.SessionID, event.Workspace, event.ClaudeVersion, "hook"); err != nil {
			log.Printf("hooks: SessionStart upsert error: %v", err)
			// Continue anyway - don't fail the hook
		} else {
			log.Printf("hooks: SessionStart session=%s workspace=%s", event.SessionID, event.Workspace)
		}

		// Backfill any spans that arrived before this hook (race condition handling)
		backfilled, err := s.obsBackend.BackfillSpansWorkspace(ctx, event.SessionID, event.Workspace)
		if err != nil {
			log.Printf("hooks: SessionStart backfill error: %v", err)
		} else if backfilled > 0 {
			log.Printf("hooks: SessionStart backfilled %d spans with workspace", backfilled)
		}

	case "PreToolUse":
		// Record tool call start
		if event.SessionID == "" || event.ToolUseID == "" || event.ToolName == "" {
			log.Printf("hooks: PreToolUse missing required fields")
			http.Error(w, "session_id, tool_use_id, tool_name required", http.StatusBadRequest)
			return
		}

		toolInput := ""
		if event.ToolInput != nil {
			toolInput = string(event.ToolInput)
		}

		if err := s.obsBackend.InsertToolStart(ctx, event.SessionID, event.ToolUseID, event.ToolName, toolInput); err != nil {
			log.Printf("hooks: PreToolUse error: %v", err)
		} else {
			log.Printf("hooks: PreToolUse session=%s tool=%s", event.SessionID, event.ToolName)
		}

	case "PostToolUse":
		// Record tool call completion
		if event.ToolUseID == "" {
			log.Printf("hooks: PostToolUse missing tool_use_id")
			http.Error(w, "tool_use_id required", http.StatusBadRequest)
			return
		}

		toolResponse := ""
		if event.ToolResponse != nil {
			toolResponse = string(event.ToolResponse)
			// Truncate very long responses
			if len(toolResponse) > 10000 {
				toolResponse = toolResponse[:10000] + "...[truncated]"
			}
		}

		// Determine success from response (could parse for error indicators)
		success := true

		if err := s.obsBackend.UpdateToolEnd(ctx, event.ToolUseID, toolResponse, success); err != nil {
			log.Printf("hooks: PostToolUse error: %v", err)
		} else {
			log.Printf("hooks: PostToolUse tool_use_id=%s success=%v", event.ToolUseID, success)
		}

	case "Stop":
		// Mark session as ended
		if event.SessionID == "" {
			log.Printf("hooks: Stop missing session_id")
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}

		if err := s.obsBackend.UpdateSessionEnded(ctx, event.SessionID); err != nil {
			log.Printf("hooks: Stop error: %v", err)
		} else {
			log.Printf("hooks: Stop session=%s ended", event.SessionID)
		}

	default:
		log.Printf("hooks: unknown event type: %s", event.Event)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
