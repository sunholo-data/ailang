package observatory

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

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

	// Verify workspace ownership if workspace filter is specified
	if requestedWorkspace := r.URL.Query().Get("workspace"); requestedWorkspace != "" {
		sessionWorkspace, err := sqliteBackend.store.GetSessionWorkspace(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if sessionWorkspace != "" && sessionWorkspace != requestedWorkspace {
			writeError(w, http.StatusForbidden, "session belongs to different workspace")
			return
		}
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

	// Verify workspace ownership if workspace filter is specified
	if requestedWorkspace := r.URL.Query().Get("workspace"); requestedWorkspace != "" {
		sessionWorkspace, err := sqliteBackend.store.GetSessionWorkspace(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if sessionWorkspace != "" && sessionWorkspace != requestedWorkspace {
			writeError(w, http.StatusForbidden, "session belongs to different workspace")
			return
		}
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

// ===== Session Metrics Handlers (Claude Code telemetry) =====

// handleGetSessionMetrics returns aggregated metrics for a session.
// GET /api/observatory/sessions/{id}/metrics
func (a *API) handleGetSessionMetrics(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session ID required")
		return
	}

	summary, err := a.backend.GetSessionMetricsSummary(r.Context(), sessionID)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

// handleListTelemetryMetrics returns OTLP metrics with optional filtering.
// GET /api/observatory/telemetry/metrics?session_id=...&name=...&workspace=...&limit=100
func (a *API) handleListTelemetryMetrics(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	opts := MetricListOptions{
		SessionID: query.Get("session_id"),
		Workspace: query.Get("workspace"),
		Name:      query.Get("name"),
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			opts.Limit = limit
		}
	}
	if opts.Limit == 0 {
		opts.Limit = 100 // Default limit
	}

	// Parse offset
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}

	// Parse time range
	if sinceStr := query.Get("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			if opts.TimeRange == nil {
				opts.TimeRange = &TimeRange{}
			}
			opts.TimeRange.Start = since
		}
	}
	if untilStr := query.Get("until"); untilStr != "" {
		if until, err := time.Parse(time.RFC3339, untilStr); err == nil {
			if opts.TimeRange == nil {
				opts.TimeRange = &TimeRange{}
			}
			opts.TimeRange.End = until
		}
	}

	metrics, err := a.backend.ListMetrics(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"metrics": metrics,
		"count":   len(metrics),
		"limit":   opts.Limit,
		"offset":  opts.Offset,
	})
}
