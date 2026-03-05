package observatory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// ===== Enriched Spans Handler =====

// handleGetEnrichedSpans returns spans with enriched display_name from session tool metadata.
// GET /api/observatory/spans/enriched?trace_id=X&task_id=Y&limit=100&hierarchical=true
//
// When hierarchical=true, returns spans as a tree structure with children[] arrays.
// This eliminates client-side hierarchy building (~50 lines of frontend code).
func (a *API) handleGetEnrichedSpans(w http.ResponseWriter, r *http.Request) {
	opts := SpanListOptions{
		TaskID:            r.URL.Query().Get("task_id"),
		TraceID:           r.URL.Query().Get("trace_id"),
		AgentAssignmentID: r.URL.Query().Get("agent_assignment_id"),
		Provider:          r.URL.Query().Get("provider"),
		Model:             r.URL.Query().Get("model"),
		Status:            r.URL.Query().Get("status"),
	}

	// Parse hierarchical flag (default: false for backwards compatibility)
	hierarchical := r.URL.Query().Get("hierarchical") == "true"

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

	// Use lightweight query (skips 3.9GB attributes/resource_attributes columns)
	// when SQLite backend is available. Falls back to full ListSpans for other backends.
	var spans []*Span
	var err error
	sqliteBackend, isSQLite := a.backend.(*SQLiteBackend)
	if isSQLite {
		spans, err = sqliteBackend.ListSpansLightweight(r.Context(), opts)
	} else {
		spans, err = a.backend.ListSpans(r.Context(), opts)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich spans with chat context if requested
	if r.URL.Query().Get("include_chat") == "true" {
		spans = a.enrichSpansWithChat(r.Context(), spans)
	}

	if !isSQLite {
		// No enrichment available, return spans as-is
		writeJSON(w, http.StatusOK, map[string]any{"spans": spans, "enriched": false, "hierarchical": false})
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

	// Build display names map for all spans
	displayNames := make(map[string]string, len(spans))
	for _, span := range spans {
		var displayName string

		// Try timestamp + tool name correlation (cross-system correlation)
		toolNameFromSpan := extractToolNameFromSpan(span)
		if toolNameFromSpan != "" {
			displayName = findMatchingToolByTimestamp(span, allToolsInRange, toolNameFromSpan)
		}

		// Fallback: extract display name from span attributes or span name
		if displayName == "" {
			displayName = extractDisplayNameFromSpan(span)
		}

		if displayName != "" {
			displayNames[span.ID] = displayName
		}
	}

	// Return hierarchical format if requested
	if hierarchical {
		hierarchicalSpans := buildHierarchicalSpans(spans, displayNames)
		writeJSON(w, http.StatusOK, map[string]any{
			"spans":        hierarchicalSpans,
			"enriched":     true,
			"hierarchical": true,
		})
		return
	}

	// Return flat format (default, backwards compatible)
	enrichedSpans := make([]map[string]any, 0, len(spans))
	for _, span := range spans {
		enriched := spanToMap(span)
		if dn, ok := displayNames[span.ID]; ok {
			enriched["display_name"] = dn
		}
		enrichedSpans = append(enrichedSpans, enriched)
	}

	writeJSON(w, http.StatusOK, map[string]any{"spans": enrichedSpans, "enriched": true, "hierarchical": false})
}

// ===== Display Name Extraction Helpers =====

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
//
//nolint:unused // Scaffolded for session grouping feature
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
//
//nolint:unused // Scaffolded for session grouping feature
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
//
//nolint:unused // Scaffolded for tool display enhancement
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
//
//nolint:unused // Helper for findMatchingToolDisplayName
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

		// OTEL tool_result events are logged when the tool COMPLETES,
		// so span.StartTime often matches tool.EndTime rather than StartTime.
		// Check both and use whichever has the better match.
		diffStart := span.StartTime.Sub(tool.StartTime)
		if diffStart < 0 {
			diffStart = -diffStart
		}

		diffEnd := 24 * time.Hour // Default to large diff if no EndTime
		if tool.EndTime != nil {
			diffEnd = span.StartTime.Sub(*tool.EndTime)
			if diffEnd < 0 {
				diffEnd = -diffEnd
			}
		}

		// Use the better of the two diffs
		diff := diffStart
		if diffEnd < diffStart {
			diff = diffEnd
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
	case "Read", "Write":
		if path, ok := data["file_path"].(string); ok {
			return fmt.Sprintf("%s: %s", toolName, shortenPath(path))
		}
	case "Edit":
		if path, ok := data["file_path"].(string); ok {
			// Show what's being changed if available
			if oldStr, ok := data["old_string"].(string); ok {
				preview := truncateString(oldStr, 25)
				return fmt.Sprintf("Edit: %s (%q)", shortenPath(path), preview)
			}
			return fmt.Sprintf("Edit: %s", shortenPath(path))
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
	case "TodoWrite":
		// Show todo count and status breakdown
		if todos, ok := data["todos"].([]any); ok {
			total := len(todos)
			if total == 0 {
				return "TodoWrite: Clear todos"
			}
			// Count by status
			completed, inProgress, pending := 0, 0, 0
			for _, t := range todos {
				if todo, ok := t.(map[string]any); ok {
					if status, ok := todo["status"].(string); ok {
						switch status {
						case "completed":
							completed++
						case "in_progress":
							inProgress++
						case "pending":
							pending++
						}
					}
				}
			}
			// Build summary
			parts := []string{}
			if completed > 0 {
				parts = append(parts, fmt.Sprintf("%d done", completed))
			}
			if inProgress > 0 {
				parts = append(parts, fmt.Sprintf("%d active", inProgress))
			}
			if pending > 0 {
				parts = append(parts, fmt.Sprintf("%d pending", pending))
			}
			if len(parts) > 0 {
				return fmt.Sprintf("TodoWrite: %d todos (%s)", total, joinStrings(parts, ", "))
			}
			return fmt.Sprintf("TodoWrite: %d todos", total)
		}
	case "TaskOutput":
		// Show which task we're getting output from
		if taskID, ok := data["task_id"].(string); ok {
			return fmt.Sprintf("TaskOutput: %s", truncateString(taskID, 20))
		}
	case "KillShell":
		// Show which shell is being killed
		if shellID, ok := data["shell_id"].(string); ok {
			return fmt.Sprintf("KillShell: %s", truncateString(shellID, 12))
		}
	case "BashOutput":
		// Show which bash output is being read
		if bashID, ok := data["bash_id"].(string); ok {
			return fmt.Sprintf("BashOutput: %s", truncateString(bashID, 12))
		}
	case "NotebookEdit":
		// Show notebook being edited
		if path, ok := data["notebook_path"].(string); ok {
			if editMode, ok := data["edit_mode"].(string); ok {
				return fmt.Sprintf("NotebookEdit: %s (%s)", shortenPath(path), editMode)
			}
			return fmt.Sprintf("NotebookEdit: %s", shortenPath(path))
		}
	}

	return toolName
}

// ===== String Utility Functions =====

// joinStrings joins strings with a separator (avoiding strings.Join import).
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
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

// ===== Span Conversion and Hierarchy =====

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
	// DurationMs is calculated, include if available
	if span.DurationMs > 0 {
		m["duration_ms"] = span.DurationMs
	}
	// Chat context (populated when include_chat=true)
	if span.ChatContext != nil {
		m["chat_context"] = span.ChatContext
	}
	return m
}

// HierarchicalSpan represents a span with children for hierarchical JSON response.
// This type is used by the /spans/enriched?hierarchical=true endpoint.
type HierarchicalSpan struct {
	ID           string              `json:"id"`
	TraceID      string              `json:"trace_id"`
	ParentSpanID string              `json:"parent_span_id,omitempty"`
	Name         string              `json:"name"`
	DisplayName  string              `json:"display_name,omitempty"`
	StartTime    time.Time           `json:"start_time"`
	EndTime      *time.Time          `json:"end_time,omitempty"`
	DurationMs   int64               `json:"duration_ms"`
	Status       SpanStatus          `json:"status"`
	Attributes   map[string]any      `json:"attributes,omitempty"`
	Provider     Provider            `json:"provider,omitempty"`
	Model        string              `json:"model,omitempty"`
	TokensIn     int64               `json:"tokens_in,omitempty"`
	TokensOut    int64               `json:"tokens_out,omitempty"`
	CostUSD      float64             `json:"cost_usd,omitempty"`
	TaskID       string              `json:"task_id,omitempty"`
	ChatContext  *ChatContext        `json:"chat_context,omitempty"`
	Children     []*HierarchicalSpan `json:"children,omitempty"`
}

// buildHierarchicalSpans transforms flat spans into a tree structure.
// Returns only root spans (no parent_span_id) with their children populated.
func buildHierarchicalSpans(spans []*Span, displayNames map[string]string) []*HierarchicalSpan {
	if len(spans) == 0 {
		return nil
	}

	// Build lookup maps
	spanMap := make(map[string]*HierarchicalSpan, len(spans))
	childMap := make(map[string][]*HierarchicalSpan)

	// Convert all spans to HierarchicalSpan
	for _, span := range spans {
		hs := &HierarchicalSpan{
			ID:           span.ID,
			TraceID:      span.TraceID,
			ParentSpanID: span.ParentSpanID,
			Name:         span.Name,
			DisplayName:  displayNames[span.ID],
			StartTime:    span.StartTime,
			EndTime:      span.EndTime,
			DurationMs:   span.DurationMs,
			Status:       span.Status,
			Attributes:   span.Attributes,
			Provider:     span.Provider,
			Model:        span.Model,
			TokensIn:     span.TokensIn,
			TokensOut:    span.TokensOut,
			CostUSD:      span.CostUSD,
			TaskID:       span.TaskID,
			ChatContext:  span.ChatContext,
			Children:     nil,
		}
		if hs.DisplayName == "" {
			hs.DisplayName = span.Name
		}
		spanMap[span.ID] = hs

		if span.ParentSpanID != "" {
			childMap[span.ParentSpanID] = append(childMap[span.ParentSpanID], hs)
		}
	}

	// Link children to parents
	for parentID, children := range childMap {
		if parent, ok := spanMap[parentID]; ok {
			parent.Children = children
			// Sort children by start time
			sort.Slice(parent.Children, func(i, j int) bool {
				return parent.Children[i].StartTime.Before(parent.Children[j].StartTime)
			})
		}
	}

	// Collect root spans (no parent or parent not in this set)
	var roots []*HierarchicalSpan
	for _, span := range spans {
		if span.ParentSpanID == "" {
			roots = append(roots, spanMap[span.ID])
		} else if _, hasParent := spanMap[span.ParentSpanID]; !hasParent {
			// Parent not in result set, treat as root
			roots = append(roots, spanMap[span.ID])
		}
	}

	// Sort roots by start time
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].StartTime.Before(roots[j].StartTime)
	})

	return roots
}
