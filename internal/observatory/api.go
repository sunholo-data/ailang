// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
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
	mux.HandleFunc("GET /api/observatory/sessions/{id}/metrics", a.handleGetSessionMetrics)

	// OTLP Metrics endpoints (Claude Code telemetry)
	mux.HandleFunc("GET /api/observatory/telemetry/metrics", a.handleListTelemetryMetrics)

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
