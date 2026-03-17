// Package observatory provides trace listing and task span summary operations.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// GetTrace retrieves a complete trace with all spans.
func (s *Store) GetTrace(traceID string) (*Trace, error) {
	spans, err := s.ListSpans(SpanListOptions{TraceID: traceID})
	if err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return nil, sql.ErrNoRows
	}

	trace := &Trace{
		TraceID:   traceID,
		Spans:     spans,
		SpanCount: len(spans),
	}

	// Find root span and calculate duration
	for _, span := range spans {
		if span.ParentSpanID == "" {
			trace.RootSpan = span
		}
		if trace.StartTime.IsZero() || span.StartTime.Before(trace.StartTime) {
			trace.StartTime = span.StartTime
		}
		if span.EndTime != nil && span.EndTime.After(trace.EndTime) {
			trace.EndTime = *span.EndTime
		}
	}

	if !trace.EndTime.IsZero() {
		trace.DurationMs = trace.EndTime.Sub(trace.StartTime).Milliseconds()
	}

	return trace, nil
}

// ListTraces returns trace summaries.
// Uses the trace_summaries materialized table (M-PERF-OBSERVATORY v12 migration)
// instead of correlated subqueries on the spans table.
func (s *Store) ListTraces(opts TraceQuery) ([]*TraceSummary, error) {
	// Try fast path: trace_summaries table (no correlated subqueries)
	summaries, err := s.listTracesFromSummaries(opts)
	if err == nil && len(summaries) > 0 {
		return summaries, nil
	}
	// Fallback: trace_summaries table might be empty (pre-v12 migration)
	// Use the original query with correlated subqueries
	return s.listTracesLegacy(opts)
}

// listTracesFromSummaries reads from the pre-computed trace_summaries table.
// O(1) per row — no correlated subqueries.
func (s *Store) listTracesFromSummaries(opts TraceQuery) ([]*TraceSummary, error) {
	query := `
		SELECT trace_id, root_span_name, root_span_status, span_count,
		       total_duration_ms, start_time, task_id, service_name
		FROM trace_summaries
		WHERE 1=1
	`
	var args []interface{}

	if opts.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, opts.TaskID)
	}
	if opts.TraceID != "" {
		query += " AND trace_id = ?"
		args = append(args, opts.TraceID)
	}
	if opts.TimeRange != nil {
		query += " AND start_time >= ? AND start_time <= ?"
		args = append(args, opts.TimeRange.Start, opts.TimeRange.End)
	}

	query += " ORDER BY start_time DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*TraceSummary
	for rows.Next() {
		ts := &TraceSummary{
			Source: TraceSourceLocal,
		}
		var rootSpan, rootStatus, taskID, serviceName sql.NullString
		var startTimeStr string
		if err := rows.Scan(&ts.TraceID, &rootSpan, &rootStatus, &ts.SpanCount,
			&ts.DurationMs, &startTimeStr, &taskID, &serviceName); err != nil {
			return nil, err
		}
		if rootSpan.Valid {
			ts.RootSpan = rootSpan.String
		}
		if rootStatus.Valid {
			ts.Status = SpanStatus(rootStatus.String)
		}
		if taskID.Valid {
			ts.TaskID = taskID.String
		}
		if serviceName.Valid {
			ts.ServiceName = serviceName.String
		}
		ts.StartTime = parseTimeString(startTimeStr)
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// listTracesLegacy is the original query with correlated subqueries.
// Kept as fallback for databases that haven't run the v12 migration yet.
func (s *Store) listTracesLegacy(opts TraceQuery) ([]*TraceSummary, error) {
	query := `
		SELECT trace_id,
		       (SELECT name FROM spans s2 WHERE s2.trace_id = s.trace_id AND s2.parent_span_id IS NULL LIMIT 1) as root_span,
		       COUNT(*) as span_count,
		       COALESCE(SUM(duration_ms), 0) as duration_ms,
		       MIN(start_time) as start_time,
		       (SELECT status FROM spans s3 WHERE s3.trace_id = s.trace_id AND s3.parent_span_id IS NULL LIMIT 1) as status,
		       task_id,
		       (SELECT resource_attributes FROM spans s4 WHERE s4.trace_id = s.trace_id AND s4.parent_span_id IS NULL LIMIT 1) as resource_attrs
		FROM spans s
		WHERE 1=1
	`
	var args []interface{}

	if opts.TaskID != "" {
		query += " AND task_id = ?"
		args = append(args, opts.TaskID)
	}
	if opts.TraceID != "" {
		query += " AND trace_id = ?"
		args = append(args, opts.TraceID)
	}
	if opts.TimeRange != nil {
		query += " AND start_time >= ? AND start_time <= ?"
		args = append(args, opts.TimeRange.Start, opts.TimeRange.End)
	}

	query += " GROUP BY trace_id ORDER BY start_time DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*TraceSummary
	for rows.Next() {
		ts := &TraceSummary{
			Source: TraceSourceLocal,
		}
		var rootSpan, status, taskID, resourceAttrs sql.NullString
		var startTimeStr string
		if err := rows.Scan(&ts.TraceID, &rootSpan, &ts.SpanCount, &ts.DurationMs,
			&startTimeStr, &status, &taskID, &resourceAttrs); err != nil {
			return nil, err
		}
		if rootSpan.Valid {
			ts.RootSpan = rootSpan.String
		}
		if status.Valid {
			ts.Status = SpanStatus(status.String)
		}
		if taskID.Valid {
			ts.TaskID = taskID.String
		}
		if resourceAttrs.Valid && resourceAttrs.String != "" {
			ts.ServiceName = extractServiceName(resourceAttrs.String)
		}
		ts.StartTime = parseTimeString(startTimeStr)
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

// parseTimeString handles the various time formats SQLite may return.
func parseTimeString(s string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// extractServiceName extracts service.name from resource_attributes JSON.
// TaskSpanSummary holds aggregated span statistics for a task_id.
// Used by "ailang chains find --task-id" when no execution_chain exists
// (e.g., Claude Code user sessions that aren't coordinator-managed).
type TaskSpanSummary struct {
	TaskID     string    `json:"task_id"`
	SpanCount  int       `json:"span_count"`
	TraceCount int       `json:"trace_count"`
	TokensIn   int64     `json:"tokens_in"`
	TokensOut  int64     `json:"tokens_out"`
	CostUSD    float64   `json:"cost_usd"`
	DurationMs int64     `json:"duration_ms"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	// Top span names by count (e.g., "api_request: 487, claude_code.tool.Read: 338")
	TopSpanNames []SpanNameCount `json:"top_span_names"`
}

// SpanNameCount holds a span name and its count.
type SpanNameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetTaskSpanSummary returns aggregated span statistics for a task_id.
// Works for ALL task_id formats (coordinator, eval, UUID sessions).
func (s *Store) GetTaskSpanSummary(ctx context.Context, taskID string) (*TaskSpanSummary, error) {
	if taskID == "" {
		return nil, nil
	}

	summary := &TaskSpanSummary{TaskID: taskID}

	// Aggregate metrics (scan times as strings since SQLite returns text)
	var startStr, endStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as span_count,
			COUNT(DISTINCT trace_id) as trace_count,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out,
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(duration_ms), 0) as duration_ms,
			COALESCE(MIN(start_time), '') as start_time,
			COALESCE(MAX(start_time), '') as end_time
		FROM spans WHERE task_id = ?
	`, taskID).Scan(
		&summary.SpanCount, &summary.TraceCount,
		&summary.TokensIn, &summary.TokensOut,
		&summary.CostUSD, &summary.DurationMs,
		&startStr, &endStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get task span summary: %w", err)
	}
	// Parse time strings
	if startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", startStr); err == nil {
			summary.StartTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", startStr); err == nil {
			summary.StartTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05Z07:00", startStr); err == nil {
			summary.StartTime = t
		}
	}
	if endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", endStr); err == nil {
			summary.EndTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", endStr); err == nil {
			summary.EndTime = t
		} else if t, err := time.Parse("2006-01-02 15:04:05Z07:00", endStr); err == nil {
			summary.EndTime = t
		}
	}
	if summary.SpanCount == 0 {
		return nil, nil
	}

	// Top span names
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, COUNT(*) as cnt
		FROM spans WHERE task_id = ?
		GROUP BY name ORDER BY cnt DESC LIMIT 10
	`, taskID)
	if err != nil {
		return summary, nil // Non-fatal
	}
	defer rows.Close()
	for rows.Next() {
		var nc SpanNameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err == nil {
			summary.TopSpanNames = append(summary.TopSpanNames, nc)
		}
	}

	return summary, nil
}

func extractServiceName(jsonStr string) string {
	if jsonStr == "" || jsonStr == "{}" {
		return ""
	}
	var attrs map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &attrs); err != nil {
		return ""
	}
	if sn, ok := attrs["service.name"].(string); ok {
		return sn
	}
	return ""
}
