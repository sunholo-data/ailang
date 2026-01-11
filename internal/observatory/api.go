// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	mux.HandleFunc("POST /api/observatory/spans", a.handleCreateSpan)
	mux.HandleFunc("GET /api/observatory/spans/{id}", a.handleGetSpan)
	mux.HandleFunc("PUT /api/observatory/spans/{id}", a.handleUpdateSpan)
	mux.HandleFunc("DELETE /api/observatory/spans/{id}", a.handleDeleteSpan)
	mux.HandleFunc("GET /api/observatory/spans/{id}/events", a.handleGetSpanEvents)
	mux.HandleFunc("POST /api/observatory/spans/{id}/events", a.handleCreateSpanEvent)

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
