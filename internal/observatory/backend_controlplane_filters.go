package observatory

import "strings"

// ===== Helper Functions =====

// buildSourceTypeCondition returns SQL condition for source type filter.
// Uses trace_summaries.root_span_name via subquery for performance (M-PERF-OBSERVATORY).
// Avoids json_extract on resource_attributes (3.9GB column).
//
// NOTE: Queries using this condition should alias the spans table as 's' if using
// table-qualified columns, or leave unqualified for simple queries on 'spans' table.
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
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'claude_code.%')`
	case "eval":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'eval.%')`
	case "coordinator":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'coordinator.%' OR root_span_name LIKE 'claude.execute%' OR root_span_name LIKE 'exec.%' OR root_span_name LIKE 'ailang.exec%')`
	case "github":
		// GitHub has no spans - this condition matches nothing intentionally
		// GitHub messages are filtered in handlers_inbox.go via InferInboxSourceType
		return "1=0"
	case "messaging":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'messages.%')`
	case "cli":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'ailang.%' OR root_span_name LIKE 'ailang %' OR root_span_name LIKE 'compile%' OR root_span_name LIKE 'check.%')`
	case "direct_api":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'anthropic.%' OR root_span_name LIKE 'gemini.%' OR root_span_name LIKE 'openai.%' OR root_span_name IN ('api_request', 'api_error', 'call_llm', 'invocation'))`
	case "exec":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'ailang.exec%')`
	case "local":
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE root_span_name LIKE 'ailang.%' OR root_span_name LIKE 'ailang %')`
	case "other":
		// Exclude all known categories via trace_summaries
		return `trace_id IN (SELECT trace_id FROM trace_summaries WHERE
			root_span_name NOT LIKE 'eval.%'
			AND root_span_name NOT LIKE 'coordinator.%'
			AND root_span_name NOT LIKE 'claude.execute%'
			AND root_span_name NOT LIKE 'exec.%'
			AND root_span_name NOT LIKE 'ailang.exec%'
			AND root_span_name NOT LIKE 'messages.%'
			AND root_span_name NOT LIKE 'ailang-%'
			AND root_span_name NOT LIKE 'ailang.%'
			AND root_span_name NOT LIKE 'ailang %'
			AND root_span_name NOT LIKE 'compile%'
			AND root_span_name NOT LIKE 'check.%'
			AND root_span_name NOT LIKE 'claude_code.%'
			AND root_span_name NOT LIKE 'anthropic.%'
			AND root_span_name NOT LIKE 'gemini.%'
			AND root_span_name NOT LIKE 'openai.%'
			AND root_span_name NOT IN ('api_request', 'api_error', 'call_llm', 'invocation')
		)`
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
