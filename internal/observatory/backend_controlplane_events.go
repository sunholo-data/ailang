package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ===== Claude Code Event Types =====

// ClaudeCodeEvent represents a Claude Code API request span formatted as an event
// for the Event Queue. Each api_request span becomes an event that can be clicked
// to show its hierarchy (tool calls correlated by timestamp).
type ClaudeCodeEvent struct {
	ID             string  `json:"id"`           // span_id (also used as task_id for hierarchy lookup)
	CreatedAt      string  `json:"created_at"`   // ISO8601 timestamp (matches inbox message format)
	Type           string  `json:"message_type"` // "claude_code_turn"
	FromAgent      string  `json:"from_agent"`   // Agent ID (e.g., "design-doc-creator") or "claude-code" for user sessions
	ToInbox        string  `json:"to_inbox"`     // Agent inbox (e.g., "design-doc-creator") or "user" for user sessions
	Title          string  `json:"title"`        // "Claude Code Turn ($X.XX)"
	TaskID         string  `json:"task_id"`      // Same as ID for hierarchy lookup
	Status         string  `json:"status"`       // "read" (not actionable like inbox messages)
	CostUSD        float64 `json:"cost_usd"`
	TokensIn       int64   `json:"tokens_in"`
	TokensOut      int64   `json:"tokens_out"`
	DurationMs     int     `json:"duration_ms"`
	Workspace      string  `json:"workspace,omitempty"`       // Working directory (from resource attributes)
	Model          string  `json:"model,omitempty"`           // Model used for this event (e.g., "claude-sonnet-4-5")
	Provider       string  `json:"provider,omitempty"`        // AI provider (e.g., "claude", "gemini")
	Directive      string  `json:"directive,omitempty"`       // Initial user prompt (truncated preview)
	DirectiveFull  string  `json:"directive_full,omitempty"`  // Full directive (for detail views)
	TurnCount      int     `json:"turn_count"`                // Number of turns in session
	MetricsSummary string  `json:"metrics_summary,omitempty"` // "3 turns • $0.42 • 12.5s"
}

// TaskAgentLookup is a callback to resolve coordinator task_id to agent info.
// Returns: agentID (used for FromAgent), inbox (used for ToInbox), title
// If the task_id is not found, returns empty strings (not an error).
type TaskAgentLookup func(ctx context.Context, taskID string) (agentID, inbox, title string, err error)

// GetClaudeCodeEvents returns Claude Code sessions aggregated by session.id.
// Multiple api_request spans (turns) in one session are aggregated into a single event.
// This provides a cleaner Event Queue with one entry per Claude Code conversation.
func (b *SQLiteBackend) GetClaudeCodeEvents(ctx context.Context, limit int) ([]ClaudeCodeEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	// Aggregate by session.id from attributes - sum cost/tokens, get latest timestamp
	// Use MIN(id) as the representative span_id for hierarchy lookup
	query := `
		SELECT
			json_extract(attributes, '$."session.id"') as session_id,
			MIN(id) as first_span_id,
			MAX(start_time) as latest_time,
			COUNT(*) as turn_count,
			SUM(COALESCE(duration_ms, 0)) as total_duration_ms,
			SUM(COALESCE(cost_usd, 0)) as total_cost_usd,
			SUM(COALESCE(tokens_in, 0)) as total_tokens_in,
			SUM(COALESCE(tokens_out, 0)) as total_tokens_out,
			MAX(model) as model,
			MAX(provider) as provider,
			MAX(json_extract(resource_attributes, '$."process.cwd"')) as workspace,
			COALESCE(
				MAX(json_extract(attributes, '$."task.directive"')),
				MAX(json_extract(attributes, '$.prompt')),
				MAX(json_extract(attributes, '$."user.prompt"')),
				MAX(json_extract(attributes, '$."benchmark.name"')),
				MAX(json_extract(attributes, '$."eval.benchmark"'))
			) as directive
		FROM spans
		WHERE name = 'api_request'
		  AND json_extract(resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(attributes, '$."session.id"') IS NOT NULL
		GROUP BY json_extract(attributes, '$."session.id"')
		ORDER BY latest_time DESC
		LIMIT ?
	`

	rows, err := b.store.DB().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ClaudeCodeEvent
	for rows.Next() {
		var sessionID, firstSpanID, latestTimeRaw string
		var turnCount int
		var totalDurationMs int
		var totalCostUSD float64
		var totalTokensIn, totalTokensOut int64
		var model, provider, workspace, directive sql.NullString

		if err := rows.Scan(&sessionID, &firstSpanID, &latestTimeRaw, &turnCount, &totalDurationMs, &totalCostUSD, &totalTokensIn, &totalTokensOut, &model, &provider, &workspace, &directive); err != nil {
			return nil, err
		}

		// Convert SQLite datetime format to RFC3339 (replace space with T)
		latestTime := strings.Replace(latestTimeRaw, " ", "T", 1)

		// Format cost for title
		costStr := fmt.Sprintf("$%.2f", totalCostUSD)
		if totalCostUSD < 0.01 && totalCostUSD > 0 {
			costStr = fmt.Sprintf("$%.4f", totalCostUSD)
		}

		// Format duration for title
		durationStr := fmt.Sprintf("%.1fs", float64(totalDurationMs)/1000)
		if totalDurationMs >= 60000 {
			durationStr = fmt.Sprintf("%.1fm", float64(totalDurationMs)/60000)
		}

		// Build metrics summary for display as badges
		metricsSummary := fmt.Sprintf("%d turns • %s • %s", turnCount, costStr, durationStr)

		// Set default provider if not in DB
		providerVal := "claude"
		if provider.Valid && provider.String != "" {
			providerVal = provider.String
		}

		// Truncate directive for preview (200 chars), keep full for detail view
		directivePreview := directive.String
		directiveFull := directive.String
		if len(directivePreview) > 200 {
			directivePreview = directivePreview[:200] + "..."
		}

		// Title = directive preview (for scanability), fallback to generic description
		title := directivePreview
		if title == "" {
			title = "Claude Code Session"
		}

		events = append(events, ClaudeCodeEvent{
			ID:             sessionID,  // Use session_id as the event ID
			CreatedAt:      latestTime, // Most recent activity
			Type:           "claude_code_session",
			FromAgent:      "claude-code",
			ToInbox:        "user",
			Title:          title,
			TaskID:         sessionID, // Use session_id for hierarchy lookup (aggregates all turns)
			Status:         "read",
			CostUSD:        totalCostUSD,
			TokensIn:       totalTokensIn,
			TokensOut:      totalTokensOut,
			DurationMs:     totalDurationMs,
			Model:          model.String,
			Provider:       providerVal,
			Workspace:      workspace.String,
			Directive:      directivePreview,
			DirectiveFull:  directiveFull,
			TurnCount:      turnCount,
			MetricsSummary: metricsSummary,
		})
	}

	return events, rows.Err()
}

// GetClaudeCodeEventsWithLookup returns Claude Code sessions with agent info resolved.
// For sessions spawned by coordinator tasks, FromAgent and ToInbox are set from agent_assignments.
// For user-initiated sessions (no coordinator task), defaults to "claude-code" / "user".
// Note: The lookup parameter is kept for API compatibility but is no longer used - agent info
// comes directly from observatory.db via JOIN with agent_assignments.
// The sourceType parameter filters by source:
//   - "coordinator": Only sessions with agent_assignment (coordinator-spawned)
//   - "user_session": Only sessions without agent_assignment (user-initiated)
//   - "": No filter (all sessions)
//
// The workspaceFilter parameter filters by workspace path:
//   - Full path: "/Users/mark/dev/TwilightGame" (exact or contains match)
//   - "unknown" or "No Workspace": Sessions with empty workspace
//   - "coordinator_worktrees": Sessions in worktree directories
//   - "": No filter (all workspaces)
func (b *SQLiteBackend) GetClaudeCodeEventsWithLookup(ctx context.Context, limit int, lookup TaskAgentLookup, sourceType, workspaceFilter string, wsConfig WorkspaceMapping) ([]ClaudeCodeEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	// Build source type filter condition
	var sourceFilter string
	switch sourceType {
	case "coordinator":
		// Only sessions with agent_assignment (coordinator-spawned)
		sourceFilter = " AND aa.agent_id IS NOT NULL"
	case "user_session":
		// Only sessions without agent_assignment (user-initiated)
		sourceFilter = " AND (aa.agent_id IS NULL OR aa.agent_id = '')"
	default:
		sourceFilter = "" // No filter
	}

	// Build workspace filter condition using reverse mapping
	// This converts workspace IDs (org/repo) to path patterns that match process.cwd
	var workspaceFilterSQL string
	var workspaceArgs []interface{}
	switch workspaceFilter {
	case "":
		workspaceFilterSQL = ""
	case "unknown", "No Workspace":
		// Match sessions with empty or null workspace
		workspaceFilterSQL = " AND (json_extract(s.resource_attributes, '$.\"process.cwd\"') IS NULL OR json_extract(s.resource_attributes, '$.\"process.cwd\"') = '')"
	case "coordinator_worktrees", "Coordinator Tasks":
		// Match worktree directories
		workspaceFilterSQL = " AND json_extract(s.resource_attributes, '$.\"process.cwd\"') LIKE '%/worktrees/%'"
	default:
		// Use reverse mapping to find path patterns that match the workspace ID
		var patterns []string
		if wsConfig != nil {
			patterns = wsConfig.GetPathPatternsForWorkspace(workspaceFilter)
		}
		if len(patterns) > 0 {
			// Build OR clause for all matching patterns
			var orClauses []string
			for _, p := range patterns {
				orClauses = append(orClauses, `json_extract(s.resource_attributes, '$."process.cwd"') LIKE ?`)
				workspaceArgs = append(workspaceArgs, p)
			}
			workspaceFilterSQL = " AND (" + strings.Join(orClauses, " OR ") + ")"
		} else {
			// Fallback to direct substring match if no mapping found
			workspaceFilterSQL = " AND json_extract(s.resource_attributes, '$.\"process.cwd\"') LIKE ?"
			workspaceArgs = append(workspaceArgs, "%"+workspaceFilter+"%")
		}
	}

	// Aggregate by session.id, join with agent_assignments to get agent info
	// Uses spans.task_id COLUMN (not JSON attribute) to link to agent_assignments
	query := `
		SELECT
			json_extract(s.attributes, '$."session.id"') as session_id,
			MIN(s.id) as first_span_id,
			MAX(s.start_time) as latest_time,
			COUNT(*) as turn_count,
			SUM(COALESCE(s.duration_ms, 0)) as total_duration_ms,
			SUM(COALESCE(s.cost_usd, 0)) as total_cost_usd,
			SUM(COALESCE(s.tokens_in, 0)) as total_tokens_in,
			SUM(COALESCE(s.tokens_out, 0)) as total_tokens_out,
			MAX(s.task_id) as coord_task_id,
			MAX(aa.agent_id) as agent_id,
			MAX(s.model) as model,
			MAX(s.provider) as provider,
			MAX(json_extract(s.resource_attributes, '$."process.cwd"')) as workspace,
			COALESCE(
				MAX(json_extract(s.attributes, '$."task.directive"')),
				MAX(json_extract(s.attributes, '$.prompt')),
				MAX(json_extract(s.attributes, '$."user.prompt"')),
				MAX(json_extract(s.attributes, '$."benchmark.name"')),
				MAX(json_extract(s.attributes, '$."eval.benchmark"'))
			) as directive
		FROM spans s
		LEFT JOIN agent_assignments aa ON s.task_id = aa.task_id
		WHERE s.name = 'api_request'
		  AND json_extract(s.resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(s.attributes, '$."session.id"') IS NOT NULL` + sourceFilter + workspaceFilterSQL + `
		GROUP BY json_extract(s.attributes, '$."session.id"')
		ORDER BY latest_time DESC
		LIMIT ?
	`

	// Build query arguments
	var args []interface{}
	args = append(args, workspaceArgs...)
	args = append(args, limit)

	rows, err := b.store.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ClaudeCodeEvent
	for rows.Next() {
		var sessionID, firstSpanID, latestTimeRaw string
		var turnCount int
		var totalDurationMs int
		var totalCostUSD float64
		var totalTokensIn, totalTokensOut int64
		var coordTaskID, agentID, model, provider, workspace, directive sql.NullString

		if err := rows.Scan(&sessionID, &firstSpanID, &latestTimeRaw, &turnCount, &totalDurationMs, &totalCostUSD, &totalTokensIn, &totalTokensOut, &coordTaskID, &agentID, &model, &provider, &workspace, &directive); err != nil {
			return nil, err
		}

		// Convert SQLite datetime format to RFC3339 (replace space with T)
		latestTime := strings.Replace(latestTimeRaw, " ", "T", 1)

		// Format cost for title
		costStr := fmt.Sprintf("$%.2f", totalCostUSD)
		if totalCostUSD < 0.01 && totalCostUSD > 0 {
			costStr = fmt.Sprintf("$%.4f", totalCostUSD)
		}

		// Format duration for title
		durationStr := fmt.Sprintf("%.1fs", float64(totalDurationMs)/1000)
		if totalDurationMs >= 60000 {
			durationStr = fmt.Sprintf("%.1fm", float64(totalDurationMs)/60000)
		}

		// Build metrics summary for display as badges
		metricsSummary := fmt.Sprintf("%d turns • %s • %s", turnCount, costStr, durationStr)

		// Default values for user-initiated sessions
		fromAgent := "claude-code"
		toInbox := "user"

		// If we have agent_id from agent_assignments, use it
		// (By convention, agent id == inbox in agent config)
		if agentID.Valid && agentID.String != "" {
			fromAgent = agentID.String
			toInbox = agentID.String
		}

		// Set default provider if not in DB
		providerVal := "claude"
		if provider.Valid && provider.String != "" {
			providerVal = provider.String
		}

		// Truncate directive for preview (200 chars), keep full for detail view
		directivePreview := directive.String
		directiveFull := directive.String
		if len(directivePreview) > 200 {
			directivePreview = directivePreview[:200] + "..."
		}

		// Title = directive preview (for scanability), fallback to generic description
		title := directivePreview
		if title == "" {
			title = "Claude Code Session"
		}

		events = append(events, ClaudeCodeEvent{
			ID:             sessionID,  // Use session_id as the event ID
			CreatedAt:      latestTime, // Most recent activity
			Type:           "claude_code_session",
			FromAgent:      fromAgent,
			ToInbox:        toInbox,
			Title:          title,
			TaskID:         sessionID, // Use session_id for hierarchy lookup (aggregates all turns)
			Status:         "read",
			CostUSD:        totalCostUSD,
			TokensIn:       totalTokensIn,
			TokensOut:      totalTokensOut,
			DurationMs:     totalDurationMs,
			Model:          model.String,
			Provider:       providerVal,
			Workspace:      workspace.String,
			Directive:      directivePreview,
			DirectiveFull:  directiveFull,
			TurnCount:      turnCount,
			MetricsSummary: metricsSummary,
		})
	}

	return events, rows.Err()
}

// splitPath splits a file path into components (works for both Unix and Windows paths)
//
//nolint:unused // Utility for path-based span filtering
func splitPath(path string) []string {
	// Remove trailing slashes
	path = strings.TrimRight(path, "/\\")
	if path == "" {
		return nil
	}
	// Split on both / and \
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' || c == '\\' {
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
