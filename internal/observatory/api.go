// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// API provides HTTP handlers for the observatory REST API.
type API struct {
	backend Backend
}

// NewAPI creates a new API handler.
func NewAPI(backend Backend) *API {
	return &API{backend: backend}
}

// RegisterRoutes registers all observatory routes on the given mux.
// All routes are prefixed with /api/observatory/
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	// Workspace endpoints
	mux.HandleFunc("GET /api/observatory/workspaces", a.handleListWorkspaces)
	mux.HandleFunc("POST /api/observatory/workspaces", a.handleCreateWorkspace)
	mux.HandleFunc("GET /api/observatory/workspaces/{id}", a.handleGetWorkspace)
	mux.HandleFunc("PUT /api/observatory/workspaces/{id}", a.handleUpdateWorkspace)
	mux.HandleFunc("DELETE /api/observatory/workspaces/{id}", a.handleDeleteWorkspace)
	mux.HandleFunc("GET /api/observatory/workspaces/{id}/stats", a.handleGetWorkspaceStats)

	// Task endpoints
	mux.HandleFunc("GET /api/observatory/tasks", a.handleListTasks)
	mux.HandleFunc("POST /api/observatory/tasks", a.handleCreateTask)
	mux.HandleFunc("GET /api/observatory/tasks/{id}", a.handleGetTask)
	mux.HandleFunc("PUT /api/observatory/tasks/{id}", a.handleUpdateTask)
	mux.HandleFunc("DELETE /api/observatory/tasks/{id}", a.handleDeleteTask)

	// Agent Assignment endpoints
	mux.HandleFunc("GET /api/observatory/tasks/{taskId}/agents", a.handleListAgentAssignments)
	mux.HandleFunc("POST /api/observatory/agents", a.handleCreateAgentAssignment)
	mux.HandleFunc("GET /api/observatory/agents/{id}", a.handleGetAgentAssignment)
	mux.HandleFunc("PUT /api/observatory/agents/{id}", a.handleUpdateAgentAssignment)
	mux.HandleFunc("DELETE /api/observatory/agents/{id}", a.handleDeleteAgentAssignment)
	mux.HandleFunc("GET /api/observatory/agents/{id}/stats", a.handleGetAgentStats)

	// Span endpoints
	mux.HandleFunc("GET /api/observatory/spans", a.handleListSpans)
	mux.HandleFunc("GET /api/observatory/spans/enriched", a.handleGetEnrichedSpans)
	mux.HandleFunc("POST /api/observatory/spans", a.handleCreateSpan)
	mux.HandleFunc("GET /api/observatory/spans/{id}", a.handleGetSpan)
	mux.HandleFunc("PUT /api/observatory/spans/{id}", a.handleUpdateSpan)
	mux.HandleFunc("DELETE /api/observatory/spans/{id}", a.handleDeleteSpan)
	mux.HandleFunc("GET /api/observatory/spans/{id}/events", a.handleGetSpanEvents)
	mux.HandleFunc("POST /api/observatory/spans/{id}/events", a.handleCreateSpanEvent)
	mux.HandleFunc("GET /api/observatory/spans/{id}/enriched", a.handleGetEnrichedSpan)

	// Trace endpoints
	mux.HandleFunc("GET /api/observatory/traces", a.handleListTraces)
	mux.HandleFunc("GET /api/observatory/traces/{id}", a.handleGetTrace)

	// Message endpoints
	mux.HandleFunc("GET /api/observatory/messages", a.handleListMessages)
	mux.HandleFunc("POST /api/observatory/messages", a.handleCreateMessage)
	mux.HandleFunc("GET /api/observatory/messages/{id}", a.handleGetMessage)
	mux.HandleFunc("PUT /api/observatory/messages/{id}", a.handleUpdateMessage)
	mux.HandleFunc("DELETE /api/observatory/messages/{id}", a.handleDeleteMessage)
	mux.HandleFunc("POST /api/observatory/messages/{id}/read", a.handleMarkMessageRead)
	mux.HandleFunc("POST /api/observatory/messages/{id}/archive", a.handleMarkMessageArchived)

	// Aggregate endpoints
	mux.HandleFunc("GET /api/observatory/metrics/summary", a.handleGetMetricsSummary)
	mux.HandleFunc("GET /api/observatory/metrics/providers", a.handleGetProviderComparison)
	mux.HandleFunc("GET /api/observatory/tasks/{id}/timeline", a.handleGetTaskTimeline)

	// Hierarchy endpoint (M-TASK-HIERARCHY)
	mux.HandleFunc("GET /api/observatory/tasks/{id}/hierarchy", a.handleGetTaskHierarchy)

	// Session endpoints (M-SESSION-WORKSPACE-HOOKS)
	mux.HandleFunc("GET /api/observatory/sessions", a.handleListSessions)
	mux.HandleFunc("GET /api/observatory/sessions/{id}", a.handleGetSession)
	mux.HandleFunc("GET /api/observatory/sessions/{id}/tools", a.handleGetSessionTools)
	mux.HandleFunc("GET /api/observatory/sessions/{id}/tools/summary", a.handleGetSessionToolsSummary)

	// Telemetry ingest endpoint (for receiving OTEL/Claude data)
	mux.HandleFunc("POST /api/observatory/ingest/claude", a.handleIngestClaude)
	mux.HandleFunc("POST /api/observatory/ingest/otel", a.handleIngestOTEL)
}

// ===== Response Helpers =====

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// isNotFoundError checks if an error indicates a record was not found.
func isNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// ===== Workspace Handlers =====

func (a *API) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := a.backend.ListWorkspaces(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workspaces)
}

func (a *API) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var workspace Workspace
	if err := json.NewDecoder(r.Body).Decode(&workspace); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateWorkspace(r.Context(), &workspace); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

func (a *API) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	workspace, err := a.backend.GetWorkspace(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "workspace not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (a *API) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var workspace Workspace
	if err := json.NewDecoder(r.Body).Decode(&workspace); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	workspace.ID = id
	if err := a.backend.UpdateWorkspace(r.Context(), &workspace); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workspace)
}

func (a *API) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteWorkspace(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetWorkspaceStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stats, err := a.backend.GetWorkspaceStats(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ===== Task Handlers =====

func (a *API) handleListTasks(w http.ResponseWriter, r *http.Request) {
	opts := TaskListOptions{
		WorkspaceID: r.URL.Query().Get("workspace_id"),
		Status:      TaskStatus(r.URL.Query().Get("status")),
		SourceType:  TaskSourceType(r.URL.Query().Get("source_type")),
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

	tasks, err := a.backend.ListTasks(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (a *API) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateTask(r.Context(), &task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (a *API) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := a.backend.GetTask(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "task not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	task.ID = id
	if err := a.backend.UpdateTask(r.Context(), &task); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ===== Agent Assignment Handlers =====

func (a *API) handleListAgentAssignments(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	agents, err := a.backend.ListAgentAssignments(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

func (a *API) handleCreateAgentAssignment(w http.ResponseWriter, r *http.Request) {
	var agent AgentAssignment
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateAgentAssignment(r.Context(), &agent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (a *API) handleGetAgentAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, err := a.backend.GetAgentAssignment(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "agent assignment not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *API) handleUpdateAgentAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var agent AgentAssignment
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	agent.ID = id
	if err := a.backend.UpdateAgentAssignment(r.Context(), &agent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *API) handleDeleteAgentAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteAgentAssignment(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetAgentStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stats, err := a.backend.GetAgentStats(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ===== Span Handlers =====

func (a *API) handleListSpans(w http.ResponseWriter, r *http.Request) {
	opts := SpanListOptions{
		TaskID:            r.URL.Query().Get("task_id"),
		TraceID:           r.URL.Query().Get("trace_id"),
		AgentAssignmentID: r.URL.Query().Get("agent_assignment_id"),
		Provider:          r.URL.Query().Get("provider"),
		Model:             r.URL.Query().Get("model"),
		Status:            r.URL.Query().Get("status"),
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

// ===== Enriched Spans Handler =====

// handleGetEnrichedSpans returns spans with enriched display_name from session tool metadata.
// GET /api/observatory/spans/enriched?trace_id=X&task_id=Y&limit=100
func (a *API) handleGetEnrichedSpans(w http.ResponseWriter, r *http.Request) {
	opts := SpanListOptions{
		TaskID:            r.URL.Query().Get("task_id"),
		TraceID:           r.URL.Query().Get("trace_id"),
		AgentAssignmentID: r.URL.Query().Get("agent_assignment_id"),
		Provider:          r.URL.Query().Get("provider"),
		Model:             r.URL.Query().Get("model"),
		Status:            r.URL.Query().Get("status"),
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

	// Default limit for enriched queries
	if opts.Limit == 0 {
		opts.Limit = 100
	}

	spans, err := a.backend.ListSpans(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Try to get SQLite backend for session enrichment
	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		// No enrichment available, return spans as-is
		writeJSON(w, http.StatusOK, map[string]any{"spans": spans, "enriched": false})
		return
	}

	// Find time range of all spans for timestamp-based correlation
	var minTime, maxTime time.Time
	for _, span := range spans {
		if minTime.IsZero() || span.StartTime.Before(minTime) {
			minTime = span.StartTime
		}
		endTime := span.StartTime.Add(time.Duration(span.DurationMs) * time.Millisecond)
		if maxTime.IsZero() || endTime.After(maxTime) {
			maxTime = endTime
		}
	}

	// Expand time window by 30 seconds on each side for:
	// - Clock drift between OTEL spans and hook-captured tools
	// - Hook execution delay (hooks may fire before or after span creation)
	// - Network latency differences
	if !minTime.IsZero() {
		minTime = minTime.Add(-30 * time.Second)
		maxTime = maxTime.Add(30 * time.Second)
	}

	// Get all tools in the time range for timestamp-based correlation
	// This works across different session ID systems (OTEL vs hooks)
	var allToolsInRange []SessionTool
	if !minTime.IsZero() {
		allToolsInRange, _ = sqliteBackend.store.GetToolsByTimestampRange(r.Context(), minTime, maxTime, "")
	}

	// Enrich spans with tool metadata
	enrichedSpans := make([]map[string]any, 0, len(spans))
	for _, span := range spans {
		enriched := spanToMap(span)

		var displayName string

		// Try timestamp + tool name correlation (cross-system correlation)
		// This works even when OTEL spans and hook session_tools use different session IDs
		toolNameFromSpan := extractToolNameFromSpan(span)
		if toolNameFromSpan != "" {
			displayName = findMatchingToolByTimestamp(span, allToolsInRange, toolNameFromSpan)
		}

		// Fallback: extract display name from span attributes or span name
		if displayName == "" {
			displayName = extractDisplayNameFromSpan(span)
		}

		if displayName != "" {
			enriched["display_name"] = displayName
		}

		enrichedSpans = append(enrichedSpans, enriched)
	}

	writeJSON(w, http.StatusOK, map[string]any{"spans": enrichedSpans, "enriched": true})
}

// extractDisplayNameFromSpan extracts a display name from span attributes or name.
// Used as fallback when session_tools don't have matching data.
func extractDisplayNameFromSpan(span *Span) string {
	// Try to get tool_name from attributes
	if span.Attributes != nil {
		if toolName, ok := span.Attributes["tool_name"].(string); ok && toolName != "" {
			return toolName
		}
	}

	// Parse "claude_code.tool.X" pattern from span name
	const prefix = "claude_code.tool."
	if len(span.Name) > len(prefix) && span.Name[:len(prefix)] == prefix {
		return span.Name[len(prefix):]
	}

	// Parse "api_request" with model info
	if span.Name == "api_request" && span.Model != "" {
		// Truncate model name if too long
		model := span.Model
		if len(model) > 25 {
			model = model[:25] + "..."
		}
		return "API: " + model
	}

	return ""
}

// extractSessionIDsFromSpans extracts unique session IDs from span attributes.
func extractSessionIDsFromSpans(spans []*Span) []string {
	seen := make(map[string]bool)
	var ids []string

	for _, span := range spans {
		if id := extractSessionIDFromSpan(span); id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	return ids
}

// extractSessionIDFromSpan extracts session.id from span attributes or resource attributes.
func extractSessionIDFromSpan(span *Span) string {
	// Try span attributes first
	if span.Attributes != nil {
		if id, ok := span.Attributes["session.id"].(string); ok {
			return id
		}
	}
	// Try resource attributes
	if span.ResourceAttributes != nil {
		if id, ok := span.ResourceAttributes["session.id"].(string); ok {
			return id
		}
	}
	return ""
}

// findMatchingToolDisplayName finds a tool that matches the span by timestamp and generates display name.
func findMatchingToolDisplayName(span *Span, tools []SessionTool) string {
	// Look for tool with matching timestamp (within 100ms window)
	const timestampTolerance = 100 * time.Millisecond

	for _, tool := range tools {
		// Check timestamp proximity
		diff := span.StartTime.Sub(tool.StartTime)
		if diff < 0 {
			diff = -diff
		}
		if diff > timestampTolerance {
			continue
		}

		// Found matching tool - generate display name
		return generateDisplayName(tool.ToolName, tool.ToolInput)
	}

	// No matching tool found - try to match by span name containing tool name
	for _, tool := range tools {
		if containsToolName(span.Name, tool.ToolName) {
			return generateDisplayName(tool.ToolName, tool.ToolInput)
		}
	}

	return ""
}

// containsToolName checks if span name contains the tool name.
func containsToolName(spanName, toolName string) bool {
	// Simple substring match
	return len(spanName) >= len(toolName) && spanName[:len(toolName)] == toolName
}

// extractToolNameFromSpan extracts the tool name from a span (e.g., "Bash" from "claude_code.tool.Bash").
func extractToolNameFromSpan(span *Span) string {
	// Try to get tool_name from attributes first
	if span.Attributes != nil {
		if toolName, ok := span.Attributes["tool_name"].(string); ok && toolName != "" {
			return toolName
		}
	}

	// Parse "claude_code.tool.X" pattern from span name
	const prefix = "claude_code.tool."
	if len(span.Name) > len(prefix) && span.Name[:len(prefix)] == prefix {
		return span.Name[len(prefix):]
	}

	return ""
}

// findMatchingToolByTimestamp finds a tool that matches the span by timestamp and tool name.
// Uses a tolerance window for clock drift between OTEL spans and hook-captured tools.
func findMatchingToolByTimestamp(span *Span, tools []SessionTool, expectedToolName string) string {
	// Tolerance window: 10 seconds to account for:
	// - Hook execution delay (hooks run after tool completion)
	// - Clock drift between OTEL and hook systems
	// - Network latency for OTEL export vs immediate hook POST
	const timestampTolerance = 10 * time.Second

	var bestMatch *SessionTool
	var bestDiff time.Duration = 24 * time.Hour // Start with large diff

	for i := range tools {
		tool := &tools[i]

		// Must match tool name
		if tool.ToolName != expectedToolName {
			continue
		}

		// Calculate timestamp difference
		diff := span.StartTime.Sub(tool.StartTime)
		if diff < 0 {
			diff = -diff
		}

		// Check if within tolerance and is better than current best
		if diff <= timestampTolerance && diff < bestDiff {
			bestMatch = tool
			bestDiff = diff
		}
	}

	if bestMatch != nil {
		return generateDisplayName(bestMatch.ToolName, bestMatch.ToolInput)
	}

	return ""
}

// generateDisplayName creates a human-readable display name from tool metadata.
func generateDisplayName(toolName string, input json.RawMessage) string {
	var data map[string]any
	if err := json.Unmarshal(input, &data); err != nil {
		return toolName
	}

	switch toolName {
	case "Read", "Write", "Edit":
		if path, ok := data["file_path"].(string); ok {
			return fmt.Sprintf("%s: %s", toolName, shortenPath(path))
		}
	case "Grep":
		if pattern, ok := data["pattern"].(string); ok {
			truncated := truncateString(pattern, 30)
			if path, ok := data["path"].(string); ok {
				return fmt.Sprintf("Grep: %q in %s", truncated, shortenPath(path))
			}
			return fmt.Sprintf("Grep: %q", truncated)
		}
	case "Glob":
		if pattern, ok := data["pattern"].(string); ok {
			if path, ok := data["path"].(string); ok {
				return fmt.Sprintf("Glob: %s in %s", pattern, shortenPath(path))
			}
			return fmt.Sprintf("Glob: %s", pattern)
		}
	case "Bash":
		if cmd, ok := data["command"].(string); ok {
			return fmt.Sprintf("Bash: %s", truncateString(cmd, 40))
		}
	case "WebFetch":
		if url, ok := data["url"].(string); ok {
			return fmt.Sprintf("WebFetch: %s", truncateURL(url))
		}
	case "Task":
		// Prefer description (short summary), then subagent_type, then prompt
		if desc, ok := data["description"].(string); ok && desc != "" {
			return fmt.Sprintf("Task: %s", truncateString(desc, 50))
		}
		if agentType, ok := data["subagent_type"].(string); ok {
			return fmt.Sprintf("Task: %s agent", agentType)
		}
		if prompt, ok := data["prompt"].(string); ok {
			return fmt.Sprintf("Task: %s", truncateString(prompt, 40))
		}
	case "Skill":
		// Show which skill was invoked
		if skill, ok := data["skill"].(string); ok {
			return fmt.Sprintf("Skill: %s", skill)
		}
	case "WebSearch":
		if query, ok := data["query"].(string); ok {
			return fmt.Sprintf("WebSearch: %q", truncateString(query, 40))
		}
	case "AskUserQuestion":
		if questions, ok := data["questions"].([]any); ok && len(questions) > 0 {
			if q, ok := questions[0].(map[string]any); ok {
				if header, ok := q["header"].(string); ok {
					return fmt.Sprintf("AskUser: %s", header)
				}
			}
		}
		return "AskUser"
	}

	return toolName
}

// shortenPath truncates a file path to show only the last 2-3 components.
func shortenPath(path string) string {
	parts := splitPathForDisplay(path)
	if len(parts) <= 3 {
		return path
	}
	return ".../" + parts[len(parts)-3] + "/" + parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

// splitPathForDisplay splits a path into components for display purposes.
func splitPathForDisplay(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// truncateString truncates a string to maxLen with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// truncateURL truncates a URL to show host + truncated path.
func truncateURL(url string) string {
	// Find host start (after ://)
	hostStart := 0
	if idx := findSubstring(url, "://"); idx >= 0 {
		hostStart = idx + 3
	}

	// Find path start (after host)
	pathStart := hostStart
	for i := hostStart; i < len(url); i++ {
		if url[i] == '/' {
			pathStart = i
			break
		}
	}

	// If URL is short enough, return as-is
	if len(url) <= 50 {
		return url
	}

	// Return host + truncated path
	host := url[hostStart:pathStart]
	if pathStart < len(url) {
		path := url[pathStart:]
		if len(path) > 20 {
			path = path[:20] + "..."
		}
		return host + path
	}
	return host
}

// findSubstring finds the index of substr in s, or -1 if not found.
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// spanToMap converts a Span to a map for JSON serialization with additional fields.
func spanToMap(span *Span) map[string]any {
	m := map[string]any{
		"id":         span.ID,
		"trace_id":   span.TraceID,
		"name":       span.Name,
		"start_time": span.StartTime,
		"end_time":   span.EndTime,
		"status":     span.Status,
	}
	if span.ParentSpanID != "" {
		m["parent_span_id"] = span.ParentSpanID
	}
	if span.TaskID != "" {
		m["task_id"] = span.TaskID
	}
	if span.AgentAssignmentID != "" {
		m["agent_assignment_id"] = span.AgentAssignmentID
	}
	if span.Attributes != nil {
		m["attributes"] = span.Attributes
	}
	if span.ResourceAttributes != nil {
		m["resource_attributes"] = span.ResourceAttributes
	}
	if span.Provider != "" {
		m["provider"] = span.Provider
	}
	if span.Model != "" {
		m["model"] = span.Model
	}
	if span.TokensIn > 0 {
		m["tokens_in"] = span.TokensIn
	}
	if span.TokensOut > 0 {
		m["tokens_out"] = span.TokensOut
	}
	if span.CostUSD > 0 {
		m["cost_usd"] = span.CostUSD
	}
	if len(span.Events) > 0 {
		m["events"] = span.Events
	}
	return m
}

// ===== Trace Handlers =====

func (a *API) handleListTraces(w http.ResponseWriter, r *http.Request) {
	opts := TraceQuery{
		TaskID:  r.URL.Query().Get("task_id"),
		TraceID: r.URL.Query().Get("trace_id"),
	}

	// Parse time range
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	if startTimeStr != "" || endTimeStr != "" {
		opts.TimeRange = &TimeRange{}
		if startTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				opts.TimeRange.Start = t
			}
		}
		if endTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
				opts.TimeRange.End = t
			}
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

	traces, err := a.backend.ListTraces(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, traces)
}

func (a *API) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	trace, err := a.backend.GetTrace(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "trace not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

// ===== Message Handlers =====

func (a *API) handleListMessages(w http.ResponseWriter, r *http.Request) {
	opts := MessageListOptions{
		Inbox:     r.URL.Query().Get("inbox"),
		Status:    MessageStatus(r.URL.Query().Get("status")),
		FromAgent: r.URL.Query().Get("from"),
		TaskID:    r.URL.Query().Get("task_id"),
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

	messages, err := a.backend.ListMessages(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (a *API) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	var message Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateMessage(r.Context(), &message); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, message)
}

func (a *API) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	message, err := a.backend.GetMessage(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "message not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, message)
}

func (a *API) handleUpdateMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var message Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	message.ID = id
	if err := a.backend.UpdateMessage(r.Context(), &message); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, message)
}

func (a *API) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteMessage(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMarkMessageRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.MarkMessageRead(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMarkMessageArchived(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.MarkMessageArchived(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ===== Aggregate Handlers =====

// handleGetMetricsSummary returns global aggregates.
// DEPRECATED: Use /api/controlplane/stats instead, which provides
// unified Observatory + Coordinator stats with filtering options.
// This endpoint will be removed in a future version.
func (a *API) handleGetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := a.backend.GetMetricsSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) handleGetProviderComparison(w http.ResponseWriter, r *http.Request) {
	comparison, err := a.backend.GetProviderComparison(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

func (a *API) handleGetTaskTimeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	timeline, err := a.backend.GetTaskTimeline(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}

// ===== Hierarchy Handler (M-TASK-HIERARCHY) =====

// enrichHierarchySpans adds display_name to spans in a hierarchy using session_tools data.
// This correlates OTEL spans with hook-captured tool metadata for richer display.
func (a *API) enrichHierarchySpans(ctx context.Context, hierarchy *TaskHierarchy) {
	if hierarchy == nil {
		return
	}

	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		return
	}

	// Collect all spans from the hierarchy structure
	// TaskHierarchy -> AgentHierarchy -> TraceHierarchy -> SpanNode
	var allSpans []*Span
	var collectFromSpanNodes func(nodes []*SpanNode)
	collectFromSpanNodes = func(nodes []*SpanNode) {
		for _, node := range nodes {
			if node != nil && node.Span != nil {
				allSpans = append(allSpans, node.Span)
			}
			if node != nil && len(node.Children) > 0 {
				collectFromSpanNodes(node.Children)
			}
		}
	}

	// Traverse the hierarchy
	for _, agent := range hierarchy.Agents {
		if agent == nil {
			continue
		}
		for _, trace := range agent.Traces {
			if trace == nil {
				continue
			}
			// Collect from flat spans list
			collectFromSpanNodes(trace.Spans)
			// Also collect from root span if present
			if trace.RootSpan != nil {
				collectFromSpanNodes([]*SpanNode{trace.RootSpan})
			}
		}
	}

	if len(allSpans) == 0 {
		return
	}

	// Find time range
	var minTime, maxTime time.Time
	for _, span := range allSpans {
		if minTime.IsZero() || span.StartTime.Before(minTime) {
			minTime = span.StartTime
		}
		endTime := span.StartTime.Add(time.Duration(span.DurationMs) * time.Millisecond)
		if maxTime.IsZero() || endTime.After(maxTime) {
			maxTime = endTime
		}
	}

	// Expand time window
	minTime = minTime.Add(-30 * time.Second)
	maxTime = maxTime.Add(30 * time.Second)

	// Fetch all tools in range
	tools, _ := sqliteBackend.store.GetToolsByTimestampRange(ctx, minTime, maxTime, "")

	// Enrich spans with display_name
	for _, span := range allSpans {
		if span == nil || span.DisplayName != "" {
			continue
		}

		toolName := extractToolNameFromSpan(span)
		if toolName == "" {
			// Try fallback display name extraction
			span.DisplayName = extractDisplayNameFromSpan(span)
			continue
		}

		// Find matching tool
		displayName := findMatchingToolByTimestamp(span, tools, toolName)
		if displayName != "" {
			span.DisplayName = displayName
		} else {
			span.DisplayName = extractDisplayNameFromSpan(span)
		}
	}
}

func (a *API) handleGetTaskHierarchy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Parse query parameters
	opts := DefaultHierarchyOptions()

	if depth := r.URL.Query().Get("depth"); depth != "" {
		if d, err := strconv.Atoi(depth); err == nil && d > 0 {
			opts.MaxDepth = d
		}
	}

	if includeSpans := r.URL.Query().Get("include_spans"); includeSpans == "false" {
		opts.IncludeSpans = false
	}

	hierarchy, err := GetTaskHierarchy(r.Context(), a.backend, id, opts)
	if err != nil {
		if isNotFoundError(err) {
			// Fallback: Try Claude Code hierarchy if task not found
			// This handles the case where "id" is a Claude Code span_id rather than a task_id
			if sqliteBackend, ok := a.backend.(*SQLiteBackend); ok {
				ccHierarchy, ccErr := sqliteBackend.GetClaudeCodeHierarchy(r.Context(), id)
				if ccErr == nil {
					a.enrichHierarchySpans(r.Context(), ccHierarchy)
					writeJSON(w, http.StatusOK, ccHierarchy)
					return
				}
			}
			writeError(w, http.StatusNotFound, "task not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	a.enrichHierarchySpans(r.Context(), hierarchy)
	writeJSON(w, http.StatusOK, hierarchy)
}

// ===== Telemetry Ingest Handlers =====

func (a *API) handleIngestClaude(w http.ResponseWriter, r *http.Request) {
	var metrics ClaudeMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	normalizer := NewProviderNormalizer()
	span, err := normalizer.NormalizeClaudeMetrics(&metrics)
	if err != nil {
		writeError(w, http.StatusBadRequest, "normalization failed: "+err.Error())
		return
	}

	if err := a.backend.CreateSpan(r.Context(), span); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create span events
	for _, event := range span.Events {
		if err := a.backend.CreateSpanEvent(r.Context(), &event); err != nil {
			// Log but don't fail - span was created
			fmt.Printf("warning: failed to create span event: %v\n", err)
		}
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"span_id":  span.ID,
		"trace_id": span.TraceID,
	})
}

func (a *API) handleIngestOTEL(w http.ResponseWriter, r *http.Request) {
	// Parse OTEL spans (could be single or array)
	body, err := json.Marshal(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	spans, err := ParseGeminiTraceJSON(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid OTEL JSON: "+err.Error())
		return
	}

	// Get task ID from query param or header
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		taskID = r.Header.Get("X-Task-ID")
	}

	normalizer := NewProviderNormalizer()
	normalized, err := normalizer.NormalizeGeminiTrace(spans, taskID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "normalization failed: "+err.Error())
		return
	}

	var spanIDs []string
	for _, span := range normalized {
		if err := a.backend.CreateSpan(r.Context(), span); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		spanIDs = append(spanIDs, span.ID)

		// Create span events
		for _, event := range span.Events {
			if err := a.backend.CreateSpanEvent(r.Context(), &event); err != nil {
				fmt.Printf("warning: failed to create span event: %v\n", err)
			}
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"span_ids": spanIDs,
		"count":    len(spanIDs),
	})
}

// ===== Session Handlers (M-SESSION-WORKSPACE-HOOKS) =====

func (a *API) handleListSessions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Access Store directly via SQLiteBackend
	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		writeError(w, http.StatusNotImplemented, "session queries require SQLite backend")
		return
	}

	sessions, err := sqliteBackend.store.ListRecentSessions(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (a *API) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		writeError(w, http.StatusNotImplemented, "session queries require SQLite backend")
		return
	}

	session, err := sqliteBackend.store.GetSession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found: "+id)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (a *API) handleGetSessionTools(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		writeError(w, http.StatusNotImplemented, "session queries require SQLite backend")
		return
	}

	tools, err := sqliteBackend.store.GetSessionTools(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich tool data with parsed metadata
	enrichedTools := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		enriched := map[string]any{
			"tool_use_id": tool.ToolUseID,
			"session_id":  tool.SessionID,
			"tool_name":   tool.ToolName,
			"start_time":  tool.StartTime,
		}
		if tool.EndTime != nil {
			enriched["end_time"] = tool.EndTime
			enriched["duration_ms"] = tool.EndTime.Sub(tool.StartTime).Milliseconds()
		}
		if tool.Success != nil {
			enriched["success"] = *tool.Success
		}
		if len(tool.ToolInput) > 0 {
			// Validate JSON before including - malformed data can cause marshal errors
			if json.Valid(tool.ToolInput) {
				enriched["tool_input"] = tool.ToolInput
			} else {
				enriched["tool_input"] = string(tool.ToolInput) // Include as string fallback
			}
			// Parse rich metadata from tool input
			if metadata := parseToolMetadata(tool.ToolName, tool.ToolInput); metadata != nil {
				enriched["metadata"] = metadata
			}
		}
		if len(tool.ToolResponse) > 0 {
			// Validate JSON before including
			if json.Valid(tool.ToolResponse) {
				enriched["tool_response"] = tool.ToolResponse
			} else {
				enriched["tool_response"] = string(tool.ToolResponse)
			}
		}
		enrichedTools = append(enrichedTools, enriched)
	}

	writeJSON(w, http.StatusOK, enrichedTools)
}

func (a *API) handleGetSessionToolsSummary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sqliteBackend, ok := a.backend.(*SQLiteBackend)
	if !ok {
		writeError(w, http.StatusNotImplemented, "session queries require SQLite backend")
		return
	}

	tools, err := sqliteBackend.store.GetSessionTools(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Aggregate by tool name with rich metadata
	summary := make(map[string]map[string]any)
	filesByTool := make(map[string][]string)

	for _, tool := range tools {
		if _, exists := summary[tool.ToolName]; !exists {
			summary[tool.ToolName] = map[string]any{
				"count":       0,
				"success":     0,
				"failed":      0,
				"total_ms":    int64(0),
				"files":       []string{},
				"directories": []string{},
				"patterns":    []string{},
				"commands":    []string{},
			}
		}

		s := summary[tool.ToolName]
		s["count"] = s["count"].(int) + 1

		if tool.Success != nil {
			if *tool.Success {
				s["success"] = s["success"].(int) + 1
			} else {
				s["failed"] = s["failed"].(int) + 1
			}
		}

		if tool.EndTime != nil {
			duration := tool.EndTime.Sub(tool.StartTime).Milliseconds()
			s["total_ms"] = s["total_ms"].(int64) + duration
		}

		// Extract rich metadata
		if len(tool.ToolInput) > 0 {
			if metadata := parseToolMetadata(tool.ToolName, tool.ToolInput); metadata != nil {
				if file, ok := metadata["file_path"].(string); ok {
					filesByTool[tool.ToolName] = append(filesByTool[tool.ToolName], file)
				}
				if pattern, ok := metadata["pattern"].(string); ok {
					patterns := s["patterns"].([]string)
					s["patterns"] = appendUnique(patterns, pattern)
				}
				if cmd, ok := metadata["command"].(string); ok {
					commands := s["commands"].([]string)
					// Truncate long commands
					if len(cmd) > 60 {
						cmd = cmd[:60] + "..."
					}
					s["commands"] = appendUnique(commands, cmd)
				}
			}
		}
	}

	// Add deduplicated files to summary
	for toolName, files := range filesByTool {
		summary[toolName]["files"] = uniqueStrings(files)
	}

	// Convert map to array format for frontend
	toolsArray := make([]map[string]any, 0, len(summary))
	for toolName, data := range summary {
		// Combine details: files, patterns, commands
		details := make([]string, 0)
		if files, ok := data["files"].([]string); ok {
			details = append(details, files...)
		}
		if patterns, ok := data["patterns"].([]string); ok {
			details = append(details, patterns...)
		}
		if commands, ok := data["commands"].([]string); ok {
			details = append(details, commands...)
		}

		toolsArray = append(toolsArray, map[string]any{
			"tool_name":     toolName,
			"count":         data["count"],
			"success_count": data["success"],
			"details":       details,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"tools": toolsArray})
}

// parseToolMetadata extracts structured metadata from tool input JSON.
func parseToolMetadata(toolName string, input json.RawMessage) map[string]any {
	var data map[string]any
	if err := json.Unmarshal(input, &data); err != nil {
		return nil
	}

	metadata := make(map[string]any)

	switch toolName {
	case "Read", "Write", "Edit":
		if path, ok := data["file_path"].(string); ok {
			metadata["file_path"] = path
		}
	case "Grep":
		if pattern, ok := data["pattern"].(string); ok {
			metadata["pattern"] = pattern
		}
		if path, ok := data["path"].(string); ok {
			metadata["search_path"] = path
		}
	case "Glob":
		if pattern, ok := data["pattern"].(string); ok {
			metadata["pattern"] = pattern
		}
		if path, ok := data["path"].(string); ok {
			metadata["search_path"] = path
		}
	case "Bash":
		if cmd, ok := data["command"].(string); ok {
			metadata["command"] = cmd
		}
	case "Task":
		if desc, ok := data["description"].(string); ok {
			metadata["description"] = desc
		}
		if prompt, ok := data["prompt"].(string); ok {
			// Truncate long prompts
			if len(prompt) > 100 {
				prompt = prompt[:100] + "..."
			}
			metadata["prompt"] = prompt
		}
		if agentType, ok := data["subagent_type"].(string); ok {
			metadata["subagent_type"] = agentType
		}
	case "Skill":
		if skill, ok := data["skill"].(string); ok {
			metadata["skill"] = skill
		}
		if args, ok := data["args"].(string); ok {
			metadata["args"] = args
		}
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

// appendUnique appends value to slice if not already present.
func appendUnique(slice []string, value string) []string {
	for _, s := range slice {
		if s == value {
			return slice
		}
	}
	return append(slice, value)
}

// uniqueStrings returns a deduplicated slice.
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
