package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// GET /api/controlplane/exec-hierarchy - Get exec task hierarchy from span attributes
// Returns tree structure of ailang exec tasks with parent/child relationships
// Query params:
//   - limit: Maximum number of exec spans to query (default: 100)
//   - include_messages: If "true", groups execs by triggering messages (4-level hierarchy)
func (s *Server) handleControlPlaneExecHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 100
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Parse include_messages parameter
	includeMessages := r.URL.Query().Get("include_messages") == "true"

	w.Header().Set("Content-Type", "application/json")

	if includeMessages {
		// Return hierarchy grouped by messages (4-level: Message -> Exec -> Turn -> Tool)
		result, err := s.obsBackend.GetExecTaskHierarchyWithMessages(r.Context(), limit)
		if err != nil {
			log.Printf("Failed to get exec hierarchy with messages: %v", err)
			http.Error(w, "Failed to get exec hierarchy", http.StatusInternalServerError)
			return
		}
		// Enrich all exec hierarchies within messages
		for _, msg := range result.Messages {
			enrichExecHierarchy(r.Context(), s.obsBackend, msg.Execs)
		}
		enrichExecHierarchy(r.Context(), s.obsBackend, result.Orphan)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("Failed to encode exec hierarchy response: %v", err)
		}
		return
	}

	// Return flat hierarchy (backward compatible)
	hierarchy, err := s.obsBackend.GetExecTaskHierarchy(r.Context(), limit)
	if err != nil {
		log.Printf("Failed to get exec hierarchy: %v", err)
		http.Error(w, "Failed to get exec hierarchy", http.StatusInternalServerError)
		return
	}

	// Enrich hierarchy with display names from session_tools
	enrichExecHierarchy(r.Context(), s.obsBackend, hierarchy)

	response := struct {
		Hierarchy []*observatory.ExecTaskNode `json:"hierarchy"`
		Count     int                         `json:"count"`
	}{
		Hierarchy: hierarchy,
		Count:     len(hierarchy),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode exec hierarchy response: %v", err)
	}
}

// GET /api/controlplane/span-hierarchy - Get span hierarchy using parent_span_id relationships
// This works with standard OTEL span parenting, not custom attributes.
// Query params:
//   - limit: Maximum number of root spans to query (default: 100)
func (s *Server) handleSpanHierarchy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse limit parameter
	limit := 100
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	result, err := s.obsBackend.GetSpanHierarchy(r.Context(), limit)
	if err != nil {
		log.Printf("Failed to get span hierarchy: %v", err)
		http.Error(w, "Failed to get span hierarchy", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode span hierarchy response: %v", err)
	}
}

// enrichExecHierarchy adds display_name to tool_use nodes using session_tools data.
// This correlates OTEL spans with hook-captured tool metadata for richer display.
func enrichExecHierarchy(ctx context.Context, backend observatory.Backend, hierarchy []*observatory.ExecTaskNode) {
	if len(hierarchy) == 0 {
		return
	}

	// Collect all tool_use nodes with their timestamps
	var toolNodes []*observatory.ExecTaskNode
	var minTime, maxTime time.Time

	var collect func(nodes []*observatory.ExecTaskNode)
	collect = func(nodes []*observatory.ExecTaskNode) {
		for _, node := range nodes {
			if node.Command == "tool_use" && node.StartTime != nil {
				toolNodes = append(toolNodes, node)
				if minTime.IsZero() || node.StartTime.Before(minTime) {
					minTime = *node.StartTime
				}
				endTime := node.StartTime.Add(time.Duration(node.DurationMs) * time.Millisecond)
				if maxTime.IsZero() || endTime.After(maxTime) {
					maxTime = endTime
				}
			}
			if len(node.Children) > 0 {
				collect(node.Children)
			}
		}
	}
	collect(hierarchy)

	if len(toolNodes) == 0 {
		return
	}

	// Expand time window for matching
	minTime = minTime.Add(-30 * time.Second)
	maxTime = maxTime.Add(30 * time.Second)

	// Fetch session tools in range
	tools, err := backend.GetToolsByTimestampRange(ctx, minTime, maxTime, "")
	if err != nil || len(tools) == 0 {
		return
	}

	// Pre-index tools by name for O(n+m) instead of O(n*m) matching
	const tolerance = 10 * time.Second
	toolsByName := make(map[string][]observatory.SessionTool, len(tools)/4)
	for _, tool := range tools {
		toolsByName[tool.ToolName] = append(toolsByName[tool.ToolName], tool)
	}

	// Enrich each tool node using pre-indexed lookup
	for _, node := range toolNodes {
		if node.ToolName == "" || node.DisplayName != "" {
			continue
		}

		candidates := toolsByName[node.ToolName]
		for _, tool := range candidates {
			toolStart := tool.StartTime
			nodeStart := *node.StartTime
			diff := toolStart.Sub(nodeStart)
			if diff < 0 {
				diff = -diff
			}
			if diff > tolerance {
				continue
			}

			displayName := generateToolDisplayName(tool.ToolName, tool.ToolInput)
			if displayName != "" {
				node.DisplayName = displayName
				break
			}
		}
	}
}

// generateToolDisplayName creates a rich display name from tool name and input.
func generateToolDisplayName(toolName string, toolInput json.RawMessage) string {
	if len(toolInput) == 0 {
		return toolName
	}

	var data map[string]any
	if err := json.Unmarshal(toolInput, &data); err != nil {
		return toolName
	}

	switch toolName {
	case "Read":
		if path, ok := data["file_path"].(string); ok {
			return "Read: " + truncateForDisplay(path, 50)
		}
	case "Edit":
		if path, ok := data["file_path"].(string); ok {
			return "Edit: " + truncateForDisplay(path, 50)
		}
	case "Write":
		if path, ok := data["file_path"].(string); ok {
			return "Write: " + truncateForDisplay(path, 50)
		}
	case "Glob":
		if pattern, ok := data["pattern"].(string); ok {
			return "Glob: " + truncateForDisplay(pattern, 50)
		}
	case "Grep":
		if pattern, ok := data["pattern"].(string); ok {
			return "Grep: " + truncateForDisplay(pattern, 50)
		}
	case "Bash":
		if cmd, ok := data["command"].(string); ok {
			return "Bash: " + truncateForDisplay(cmd, 50)
		}
		if desc, ok := data["description"].(string); ok {
			return "Bash: " + truncateForDisplay(desc, 50)
		}
	case "Task":
		if desc, ok := data["description"].(string); ok && desc != "" {
			return "Task: " + truncateForDisplay(desc, 50)
		}
		if subType, ok := data["subagent_type"].(string); ok {
			return "Task: " + subType
		}
	case "Skill":
		if skill, ok := data["skill"].(string); ok {
			return "Skill: " + skill
		}
	}

	return toolName
}

// truncateForDisplay truncates a string to maxLen characters.
func truncateForDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
