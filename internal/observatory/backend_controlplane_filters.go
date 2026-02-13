package observatory

import "strings"

// ===== Helper Functions =====

// buildSourceTypeCondition returns SQL condition for source type filter
// Uses resource_attributes (service.name, ailang.source) first, then falls back to span name patterns
// Note: JSON keys with dots must be quoted in json_extract (e.g., '$."service.name"')
//
// Canonical source types (must match InferInboxSourceType in handlers_inbox.go):
// - user_session: Claude Code user sessions
// - eval: Eval suite runs
// - coordinator: Coordinator-managed agent tasks
// - github: GitHub sync messages (inbox only, no spans)
// - messaging: Agent-to-agent messages
// - cli: CLI tool usage
// - direct_api: Direct API calls
// - other: Catch-all
func buildSourceTypeCondition(sourceType string) string {
	switch sourceType {
	case "user_session", "claude_code":
		// Match Claude Code spans from direct user sessions, but EXCLUDE coordinator-spawned sessions
		// This ensures coordinator tasks get proper cost attribution
		return `(
			(json_extract(resource_attributes, '$."service.name"') = 'claude-code' OR
			 json_extract(resource_attributes, '$."ailang.source"') = 'user' OR
			 name LIKE 'claude_code.%')
			AND COALESCE(json_extract(resource_attributes, '$."ailang.source"'), 'user') != 'coordinator'
		)`
	case "eval":
		return `(json_extract(resource_attributes, '$."service.name"') = 'ailang-eval' OR json_extract(resource_attributes, '$."ailang.source"') = 'eval' OR name LIKE 'eval.%')`
	case "coordinator":
		return `(json_extract(resource_attributes, '$."ailang.source"') = 'coordinator' OR name LIKE 'coordinator.%' OR name LIKE 'claude.execute%')`
	case "github":
		// GitHub has no spans - this condition matches nothing intentionally
		// GitHub messages are filtered in handlers_inbox.go via InferInboxSourceType
		return "1=0"
	case "messaging":
		return "(name LIKE 'messages.%')"
	case "cli":
		return "(name LIKE 'ailang.%' OR name LIKE 'ailang %' OR name LIKE 'compile%' OR name LIKE 'check.%')"
	case "direct_api":
		return "(name LIKE 'anthropic.%' OR name LIKE 'gemini.%' OR name LIKE 'openai.%')"
	case "exec":
		return "(name LIKE 'ailang.exec%')"
	case "local":
		return "(name LIKE 'ailang.%' OR name LIKE 'ailang %')"
	case "other":
		// Exclude all known categories
		return `(
			json_extract(resource_attributes, '$."service.name"') IS NULL OR
			json_extract(resource_attributes, '$."service.name"') NOT IN ('claude-code', 'ailang-eval')
		) AND (
			json_extract(resource_attributes, '$."ailang.source"') IS NULL OR
			json_extract(resource_attributes, '$."ailang.source"') NOT IN ('user', 'coordinator', 'eval')
		) AND name NOT LIKE 'eval.%'
		  AND name NOT LIKE 'coordinator.%'
		  AND name NOT LIKE 'claude.execute%'
		  AND name NOT LIKE 'anthropic.%'
		  AND name NOT LIKE 'gemini.%'
		  AND name NOT LIKE 'openai.%'
		  AND name NOT LIKE 'ailang.%'
		  AND name NOT LIKE 'ailang %'
		  AND name NOT LIKE 'claude_code.%'`
	default:
		return ""
	}
}

// buildFilterConditions builds WHERE clause conditions from a ControlPlaneFilter
// Returns conditions slice, args slice, ready for building WHERE clause
// The wsConfig parameter is used to reverse-map workspace IDs to path patterns for filtering.
func buildFilterConditions(filter *ControlPlaneFilter, wsConfig WorkspaceMapping) ([]string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filter == nil {
		return conditions, args
	}

	if filter.SourceType != "" {
		if cond := buildSourceTypeCondition(filter.SourceType); cond != "" {
			conditions = append(conditions, cond)
		}
	}
	if filter.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, filter.Provider)
	}
	if filter.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, filter.Model)
	}
	if filter.StartDate != "" {
		// start_time is datetime, compare with date string (SQLite handles this)
		conditions = append(conditions, "date(start_time) >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		conditions = append(conditions, "date(start_time) <= ?")
		args = append(args, filter.EndDate)
	}
	if filter.Workspace != "" {
		// Workspace filter: use reverse mapping to find path patterns that match
		// Spans store raw paths in process.cwd, but filters use org/repo workspace IDs
		var patterns []string
		if wsConfig != nil {
			patterns = wsConfig.GetPathPatternsForWorkspace(filter.Workspace)
		}
		if len(patterns) > 0 {
			// Build OR clause for all matching patterns
			var orClauses []string
			for _, p := range patterns {
				orClauses = append(orClauses, `json_extract(resource_attributes, '$."process.cwd"') LIKE ?`)
				args = append(args, p)
			}
			conditions = append(conditions, "("+strings.Join(orClauses, " OR ")+")")
		} else {
			// Fallback: direct match if no mapping found (supports raw path filtering)
			conditions = append(conditions, `json_extract(resource_attributes, '$."process.cwd"') LIKE ?`)
			args = append(args, "%"+filter.Workspace+"%")
		}
	}

	return conditions, args
}
