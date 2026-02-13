package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GetClaudeCodeHierarchy returns the hierarchy for a Claude Code session.
// The sessionID is the session.id from Claude Code telemetry (UUID format).
//
// Claude Code telemetry creates a SEPARATE trace for each span (api_request and tool calls).
// This means we can't use standard trace hierarchy building (which relies on parent_span_id).
// Instead, we build a turn-based hierarchy using timestamp correlation:
//   - api_request spans = "turns" (top level)
//   - tool calls that occur before the next api_request = children of that turn
func (b *SQLiteBackend) GetClaudeCodeHierarchy(ctx context.Context, sessionID string) (*TaskHierarchy, error) {
	// Fetch all spans for this session by session.id attribute
	spans, err := b.getSpansBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list spans for session: %w", err)
	}
	if len(spans) == 0 {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Sort spans by start_time (already sorted by query, but ensure)
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].StartTime.Before(spans[j].StartTime)
	})

	// Separate api_requests (turns) from tool calls
	var apiRequests []*Span
	var toolCalls []*Span
	for _, span := range spans {
		if span.Name == "api_request" {
			apiRequests = append(apiRequests, span)
		} else if strings.HasPrefix(span.Name, "claude_code.tool.") {
			toolCalls = append(toolCalls, span)
		}
	}

	// Build turn-based hierarchy: each api_request is a turn with tool calls as children
	var turnNodes []*SpanNode
	for i, apiReq := range apiRequests {
		// Find the end of this turn's time window
		var turnEnd time.Time
		if i+1 < len(apiRequests) {
			turnEnd = apiRequests[i+1].StartTime
		} else {
			// Last turn: use a far future time
			turnEnd = time.Now().Add(24 * time.Hour)
		}

		// Find tool calls that belong to this turn (started after this api_request, before next)
		var turnChildren []*SpanNode
		for _, tool := range toolCalls {
			if tool.StartTime.After(apiReq.StartTime) && tool.StartTime.Before(turnEnd) {
				turnChildren = append(turnChildren, &SpanNode{Span: tool})
			}
		}

		turnNodes = append(turnNodes, &SpanNode{
			Span:     apiReq,
			Children: turnChildren,
		})
	}

	// Calculate totals for the session
	// NOTE: Only sum api_request durations to avoid double-counting.
	// Tool calls happen INSIDE api_request turns - their duration is already
	// part of the turn's duration. Summing all spans would double-count.
	var totalTokensIn, totalTokensOut int64
	var totalCost float64
	var totalDuration int64
	for _, span := range apiRequests {
		totalTokensIn += span.TokensIn
		totalTokensOut += span.TokensOut
		totalCost += span.CostUSD
		totalDuration += span.DurationMs
	}

	// Create a virtual "session" span as the root containing all turns
	sessionSpan := &SpanNode{
		Span: &Span{
			ID:         sessionID,
			Name:       "claude_code.session",
			StartTime:  spans[0].StartTime,
			DurationMs: totalDuration,
			Provider:   "claude",
			TokensIn:   totalTokensIn,
			TokensOut:  totalTokensOut,
			CostUSD:    totalCost,
		},
		Children: turnNodes,
	}

	// Create a single "session" trace containing all turns
	sessionTrace := &TraceHierarchy{
		TraceID:  sessionID, // Use session ID as trace ID for the virtual trace
		RootSpan: sessionSpan,
		Spans:    []*SpanNode{sessionSpan}, // Include session root so frontend sees hierarchy
		Summary: &HierarchyTraceSummary{
			SpanCount:    len(spans),
			TotalTokens:  totalTokensIn + totalTokensOut,
			TotalCostUSD: totalCost,
			DurationMs:   totalDuration,
		},
	}

	// Build short ID for display
	shortID := sessionID
	if len(sessionID) >= 8 {
		shortID = sessionID[:8]
	}

	return &TaskHierarchy{
		Agents: []*AgentHierarchy{{
			Agent: &AgentAssignment{
				ID:       "claude-code-session-" + shortID,
				AgentID:  "claude-code",
				Provider: "claude",
			},
			Traces: []*TraceHierarchy{sessionTrace},
		}},
	}, nil
}

// getSpansBySessionID retrieves ALL spans for a Claude Code session.
// This queries by session.id attribute to support existing spans that don't have task_id set.
func (b *SQLiteBackend) getSpansBySessionID(ctx context.Context, sessionID string) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans
		WHERE json_extract(resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(attributes, '$."session.id"') = ?
		ORDER BY start_time ASC
		LIMIT 1000
	`

	rows, err := b.store.DB().QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
		var attrs, resourceAttrs sql.NullString

		if err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &span.Name,
			&span.StartTime, &endTime, &span.DurationMs,
			&status, &attrs, &resourceAttrs, &provider, &model,
			&span.TokensIn, &span.TokensOut, &span.CostUSD,
			&taskID, &agentAssignmentID,
		); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if status.Valid {
			span.Status = SpanStatus(status.String)
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if model.Valid {
			span.Model = model.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if attrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(attrs.String), &m) == nil {
				span.Attributes = m
			}
		}
		if resourceAttrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
				span.ResourceAttributes = m
			}
		}

		spans = append(spans, &span)
	}

	return spans, rows.Err()
}

// getSessionSpans retrieves all api_request spans for a Claude Code session.
//
//nolint:unused // Scaffolded for session-level analytics
func (b *SQLiteBackend) getSessionSpans(ctx context.Context, sessionID string) ([]*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans
		WHERE name = 'api_request'
		  AND json_extract(resource_attributes, '$."service.name"') = 'claude-code'
		  AND json_extract(attributes, '$."session.id"') = ?
		ORDER BY start_time ASC
	`

	rows, err := b.store.DB().QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
		var attrs, resourceAttrs sql.NullString

		if err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &span.Name,
			&span.StartTime, &endTime, &span.DurationMs,
			&status, &attrs, &resourceAttrs, &provider, &model,
			&span.TokensIn, &span.TokensOut, &span.CostUSD,
			&taskID, &agentAssignmentID,
		); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if status.Valid {
			span.Status = SpanStatus(status.String)
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if model.Valid {
			span.Model = model.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if attrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(attrs.String), &m) == nil {
				span.Attributes = m
			}
		}
		if resourceAttrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
				span.ResourceAttributes = m
			}
		}

		spans = append(spans, &span)
	}

	return spans, rows.Err()
}

// getSpanByID retrieves a single span by its ID.
//
//nolint:unused // Scaffolded for span detail view
func (b *SQLiteBackend) getSpanByID(ctx context.Context, spanID string) (*Span, error) {
	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans WHERE id = ?
	`
	row := b.store.DB().QueryRowContext(ctx, query, spanID)

	var span Span
	var endTime sql.NullTime
	var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
	var attrs, resourceAttrs sql.NullString

	if err := row.Scan(
		&span.ID, &span.TraceID, &parentSpanID, &span.Name,
		&span.StartTime, &endTime, &span.DurationMs,
		&status, &attrs, &resourceAttrs, &provider, &model,
		&span.TokensIn, &span.TokensOut, &span.CostUSD,
		&taskID, &agentAssignmentID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if parentSpanID.Valid {
		span.ParentSpanID = parentSpanID.String
	}
	if endTime.Valid {
		span.EndTime = &endTime.Time
	}
	if status.Valid {
		span.Status = SpanStatus(status.String)
	}
	if provider.Valid {
		span.Provider = Provider(provider.String)
	}
	if model.Valid {
		span.Model = model.String
	}
	if taskID.Valid {
		span.TaskID = taskID.String
	}
	if agentAssignmentID.Valid {
		span.AgentAssignmentID = agentAssignmentID.String
	}
	if attrs.Valid {
		var m map[string]interface{}
		if json.Unmarshal([]byte(attrs.String), &m) == nil {
			span.Attributes = m
		}
	}
	if resourceAttrs.Valid {
		var m map[string]interface{}
		if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
			span.ResourceAttributes = m
		}
	}

	return &span, nil
}

// getToolCallsInWindow finds tool call spans within a time window.
// These are spans from claude_code.tool.* that started within the given time range.
//
//nolint:unused // Scaffolded for timestamp correlation feature
func (b *SQLiteBackend) getToolCallsInWindow(ctx context.Context, start time.Time, end *time.Time) ([]*Span, error) {
	if end == nil {
		return nil, nil // Can't correlate without end time
	}

	query := `
		SELECT id, trace_id, parent_span_id, name, start_time, end_time, duration_ms,
		       status, attributes, resource_attributes, provider, model,
		       tokens_in, tokens_out, cost_usd, task_id, agent_assignment_id
		FROM spans
		WHERE (
			name LIKE 'claude_code.tool.%'
			OR (json_extract(resource_attributes, '$."service.name"') = 'claude-code' AND name NOT IN ('api_request', 'user_prompt'))
		)
		AND start_time >= ? AND start_time <= ?
		ORDER BY start_time ASC
		LIMIT 100
	`

	rows, err := b.store.DB().QueryContext(ctx, query, start, *end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var parentSpanID, status, provider, model, taskID, agentAssignmentID sql.NullString
		var attrs, resourceAttrs sql.NullString

		if err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &span.Name,
			&span.StartTime, &endTime, &span.DurationMs,
			&status, &attrs, &resourceAttrs, &provider, &model,
			&span.TokensIn, &span.TokensOut, &span.CostUSD,
			&taskID, &agentAssignmentID,
		); err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if status.Valid {
			span.Status = SpanStatus(status.String)
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if model.Valid {
			span.Model = model.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if attrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(attrs.String), &m) == nil {
				span.Attributes = m
			}
		}
		if resourceAttrs.Valid {
			var m map[string]interface{}
			if json.Unmarshal([]byte(resourceAttrs.String), &m) == nil {
				span.ResourceAttributes = m
			}
		}

		spans = append(spans, &span)
	}

	return spans, rows.Err()
}
