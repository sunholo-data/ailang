package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/claudehistory"
)

// ===== Span Handlers =====

func (a *API) handleListSpans(w http.ResponseWriter, r *http.Request) {
	opts := SpanListOptions{
		TaskID:            r.URL.Query().Get("task_id"),
		TraceID:           r.URL.Query().Get("trace_id"),
		AgentAssignmentID: r.URL.Query().Get("agent_assignment_id"),
		Provider:          r.URL.Query().Get("provider"),
		Model:             r.URL.Query().Get("model"),
		Status:            r.URL.Query().Get("status"),
		Workspace:         r.URL.Query().Get("workspace"),
	}

	if startTime := r.URL.Query().Get("start_after"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			opts.StartAfter = t
		}
	}
	if endTime := r.URL.Query().Get("start_before"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			opts.StartBefore = t
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}

	spans, err := a.backend.ListSpans(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich spans with chat context if requested
	if r.URL.Query().Get("include_chat") == "true" {
		spans = a.enrichSpansWithChat(r.Context(), spans)
	}

	writeJSON(w, http.StatusOK, spans)
}

func (a *API) handleCreateSpan(w http.ResponseWriter, r *http.Request) {
	var span Span
	if err := json.NewDecoder(r.Body).Decode(&span); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateSpan(r.Context(), &span); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, span)
}

func (a *API) handleGetSpan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	span, err := a.backend.GetSpan(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "span not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, span)
}

// handleGetEnrichedSpan returns a single span with tool metadata from session_tools.
// GET /api/observatory/spans/{id}/enriched
func (a *API) handleGetEnrichedSpan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()

	// Get base span
	span, err := a.backend.GetSpan(ctx, id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "span not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Prepare enriched response with additional fields
	type EnrichedSpan struct {
		*Span
		DisplayName  string          `json:"display_name,omitempty"`
		ToolInput    json.RawMessage `json:"tool_input,omitempty"`
		ToolResponse json.RawMessage `json:"tool_response,omitempty"`
		ToolSuccess  *bool           `json:"tool_success,omitempty"`
	}
	enriched := EnrichedSpan{Span: span}

	// Try to correlate with session_tools
	sessionID := ""
	toolName := ""

	// Extract session_id from span attributes
	if span.Attributes != nil {
		if sid, ok := span.Attributes["session.id"]; ok {
			sessionID = fmt.Sprintf("%v", sid)
		}
	}

	// Extract tool name from span name (e.g., "claude_code.tool.Read" -> "Read")
	if strings.HasPrefix(span.Name, "claude_code.tool.") {
		toolName = strings.TrimPrefix(span.Name, "claude_code.tool.")
	}

	// Query session_tools for matching tool
	if sessionID != "" && toolName != "" && !span.StartTime.IsZero() {
		tool, err := a.backend.GetToolForSpan(ctx, sessionID, toolName, span.StartTime)
		if err == nil && tool != nil {
			// Generate display name
			enriched.DisplayName = generateDisplayName(span.Name, tool.ToolInput)
			enriched.ToolInput = tool.ToolInput
			enriched.ToolResponse = tool.ToolResponse
			enriched.ToolSuccess = tool.Success
		}
	}

	// Fallback display name if no tool correlation
	if enriched.DisplayName == "" {
		enriched.DisplayName = span.Name
	}

	writeJSON(w, http.StatusOK, enriched)
}

func (a *API) handleUpdateSpan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var span Span
	if err := json.NewDecoder(r.Body).Decode(&span); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	span.ID = id
	if err := a.backend.UpdateSpan(r.Context(), &span); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, span)
}

func (a *API) handleDeleteSpan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteSpan(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetSpanEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	events, err := a.backend.GetSpanEvents(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *API) handleCreateSpanEvent(w http.ResponseWriter, r *http.Request) {
	spanID := r.PathValue("id")
	var event SpanEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	event.SpanID = spanID
	if err := a.backend.CreateSpanEvent(r.Context(), &event); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

// enrichSpansWithChat populates ChatContext for spans that have session.id attributes.
// Uses the chat_messages table to retrieve user prompts and assistant responses.
func (a *API) enrichSpansWithChat(ctx context.Context, spans []*Span) []*Span {
	// Get SQLite backend for database access
	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok || sqliteBackend == nil {
		return spans // Can't enrich without SQLite backend
	}

	db := sqliteBackend.DB()
	if db == nil {
		return spans
	}

	importer := claudehistory.NewImporter(db)

	for _, span := range spans {
		// Extract session.id from span attributes
		if span.Attributes == nil {
			continue
		}
		sessionID, ok := span.Attributes["session.id"].(string)
		if !ok || sessionID == "" {
			continue
		}

		// Get chat messages for this span's time window
		var messages []*claudehistory.ChatMessage
		var err error

		if !span.StartTime.IsZero() && span.EndTime != nil && !span.EndTime.IsZero() {
			// Query within span's time window with buffer
			start := span.StartTime.Add(-2 * time.Minute)
			end := span.EndTime.Add(2 * time.Minute)
			messages, err = importer.GetChatMessagesByTimeRange(ctx, sessionID, start, end)
		} else {
			// Fallback: get all messages for session (limited)
			messages, err = importer.GetChatMessages(ctx, sessionID)
			if len(messages) > 10 {
				messages = messages[:10]
			}
		}

		if err != nil || len(messages) == 0 {
			continue
		}

		// Build ChatContext from messages
		chatCtx := &ChatContext{
			FullChatURL: "/api/claude-history/db/session/" + sessionID,
		}

		for _, msg := range messages {
			if msg.Role == "user" && chatCtx.UserPrompt == "" {
				chatCtx.UserPrompt = truncateChatString(msg.ContentText, 500)
				chatCtx.TurnNumber = msg.TurnNumber
			} else if msg.Role == "assistant" {
				if chatCtx.AssistantResponse == "" {
					chatCtx.AssistantResponse = truncateChatString(msg.ContentText, 500)
				}
				if msg.ContentThinking != "" {
					chatCtx.HasThinking = true
				}
			}
		}

		span.ChatContext = chatCtx
	}

	return spans
}

// truncateChatString truncates a string to maxLen characters, adding "..." if truncated.
func truncateChatString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
