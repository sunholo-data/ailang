// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// UpdateTaskAggregates updates a task's aggregate metrics based on a new span.
// This should be called within a transaction when creating a span with a task_id.
func UpdateTaskAggregates(ctx context.Context, tx *sql.Tx, span *Span) error {
	if span.TaskID == "" {
		return nil // No task to update
	}

	// Increment span count and sum metrics
	errorIncrement := 0
	if span.Status == SpanStatusError {
		errorIncrement = 1
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE tasks SET
			span_count = span_count + 1,
			total_duration_ms = total_duration_ms + ?,
			total_tokens_in = total_tokens_in + ?,
			total_tokens_out = total_tokens_out + ?,
			total_cost_usd = total_cost_usd + ?,
			error_count = error_count + ?
		WHERE id = ?
	`, span.DurationMs, span.TokensIn, span.TokensOut, span.CostUSD, errorIncrement, span.TaskID)

	if err != nil {
		return fmt.Errorf("update task aggregates: %w", err)
	}
	return nil
}

// UpdateAgentAssignmentAggregates updates an agent assignment's metrics based on a new span.
// This should be called within a transaction when creating a span with an agent_assignment_id.
func UpdateAgentAssignmentAggregates(ctx context.Context, tx *sql.Tx, span *Span) error {
	if span.AgentAssignmentID == "" {
		return nil // No assignment to update
	}

	// Determine if this is a tool call based on span name
	toolCallIncrement := 0
	if isToolCallSpan(span.Name) {
		toolCallIncrement = 1
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE agent_assignments SET
			duration_ms = duration_ms + ?,
			tokens_in = tokens_in + ?,
			tokens_out = tokens_out + ?,
			cost_usd = cost_usd + ?,
			tool_calls = tool_calls + ?
		WHERE id = ?
	`, span.DurationMs, span.TokensIn, span.TokensOut, span.CostUSD, toolCallIncrement, span.AgentAssignmentID)

	if err != nil {
		return fmt.Errorf("update agent assignment aggregates: %w", err)
	}
	return nil
}

// isToolCallSpan checks if a span name represents a tool call.
func isToolCallSpan(name string) bool {
	// Claude Code tool calls: claude_code.tool.write, claude_code.tool.read, etc.
	if strings.HasPrefix(name, "claude_code.tool.") {
		return true
	}
	// Gemini tool calls: gemini.tool.read, gemini.tool.execute, etc.
	if strings.HasPrefix(name, "gemini.tool.") {
		return true
	}
	// Generic tool patterns: tool.execute, tool.call, etc.
	if strings.HasPrefix(name, "tool.") {
		return true
	}
	return false
}

// RecalculateTaskAggregates recalculates all aggregate metrics for a task from its spans.
// Use this for backfill operations or to fix inconsistent aggregates.
// Deprecated: Use Store.RecalculateTaskAggregates instead for encapsulation.
func RecalculateTaskAggregates(ctx context.Context, db *sql.DB, taskID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE tasks SET
			span_count = COALESCE((SELECT COUNT(*) FROM spans WHERE task_id = ?), 0),
			total_duration_ms = COALESCE((SELECT SUM(duration_ms) FROM spans WHERE task_id = ?), 0),
			total_tokens_in = COALESCE((SELECT SUM(tokens_in) FROM spans WHERE task_id = ?), 0),
			total_tokens_out = COALESCE((SELECT SUM(tokens_out) FROM spans WHERE task_id = ?), 0),
			total_cost_usd = COALESCE((SELECT SUM(cost_usd) FROM spans WHERE task_id = ?), 0),
			error_count = COALESCE((SELECT COUNT(*) FROM spans WHERE task_id = ? AND status = 'error'), 0)
		WHERE id = ?
	`, taskID, taskID, taskID, taskID, taskID, taskID, taskID)

	if err != nil {
		return fmt.Errorf("recalculate task aggregates: %w", err)
	}
	return nil
}

// RecalculateAgentAssignmentAggregates recalculates all aggregate metrics for an agent assignment.
// Use this for backfill operations or to fix inconsistent aggregates.
// Deprecated: Use Store.RecalculateAgentAssignmentAggregates instead for encapsulation.
func RecalculateAgentAssignmentAggregates(ctx context.Context, db *sql.DB, assignmentID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE agent_assignments SET
			duration_ms = COALESCE((SELECT SUM(duration_ms) FROM spans WHERE agent_assignment_id = ?), 0),
			tokens_in = COALESCE((SELECT SUM(tokens_in) FROM spans WHERE agent_assignment_id = ?), 0),
			tokens_out = COALESCE((SELECT SUM(tokens_out) FROM spans WHERE agent_assignment_id = ?), 0),
			cost_usd = COALESCE((SELECT SUM(cost_usd) FROM spans WHERE agent_assignment_id = ?), 0),
			tool_calls = COALESCE((SELECT COUNT(*) FROM spans WHERE agent_assignment_id = ? AND name LIKE '%tool.%'), 0)
		WHERE id = ?
	`, assignmentID, assignmentID, assignmentID, assignmentID, assignmentID, assignmentID)

	if err != nil {
		return fmt.Errorf("recalculate agent assignment aggregates: %w", err)
	}
	return nil
}
