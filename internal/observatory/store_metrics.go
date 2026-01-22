// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"encoding/json"
	"time"
)

// CreateMetric inserts a new metric.
func (s *Store) CreateMetric(m *Metric) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}

	labelsJSON, _ := json.Marshal(m.Labels)
	if m.Labels == nil {
		labelsJSON = []byte("{}")
	}

	resourceAttrsJSON, _ := json.Marshal(m.ResourceAttributes)
	if m.ResourceAttributes == nil {
		resourceAttrsJSON = []byte("{}")
	}

	result, err := s.db.Exec(`
		INSERT INTO metrics (name, metric_type, session_id, workspace, provider,
		                     label_type, label_tool, label_decision, label_language, label_model,
		                     value_int, value_float, labels, resource_attributes,
		                     timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.Name, m.Type, m.SessionID, m.Workspace, m.Provider,
		m.LabelType, m.LabelTool, m.LabelDecision, m.LabelLanguage, m.LabelModel,
		m.ValueInt, m.ValueFloat, string(labelsJSON), string(resourceAttrsJSON),
		m.Timestamp, m.CreatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		m.ID = id
	}
	return nil
}

// ListMetrics retrieves metrics with optional filtering.
func (s *Store) ListMetrics(opts MetricListOptions) ([]*Metric, error) {
	query := `
		SELECT id, name, metric_type, session_id, workspace, provider,
		       label_type, label_tool, label_decision, label_language, label_model,
		       value_int, value_float, labels, resource_attributes,
		       timestamp, created_at
		FROM metrics WHERE 1=1
	`
	var args []interface{}

	if opts.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, opts.SessionID)
	}
	if opts.Workspace != "" {
		query += " AND workspace = ?"
		args = append(args, opts.Workspace)
	}
	if opts.Name != "" {
		query += " AND name = ?"
		args = append(args, opts.Name)
	}
	if opts.TimeRange != nil {
		query += " AND timestamp BETWEEN ? AND ?"
		args = append(args, opts.TimeRange.Start, opts.TimeRange.End)
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*Metric
	for rows.Next() {
		m := &Metric{}
		var sessionID, workspace, provider sql.NullString
		var labelType, labelTool, labelDecision, labelLanguage, labelModel sql.NullString
		var labelsJSON, resourceAttrsJSON string

		err := rows.Scan(
			&m.ID, &m.Name, &m.Type, &sessionID, &workspace, &provider,
			&labelType, &labelTool, &labelDecision, &labelLanguage, &labelModel,
			&m.ValueInt, &m.ValueFloat, &labelsJSON, &resourceAttrsJSON,
			&m.Timestamp, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		m.SessionID = sessionID.String
		m.Workspace = workspace.String
		m.Provider = provider.String
		m.LabelType = labelType.String
		m.LabelTool = labelTool.String
		m.LabelDecision = labelDecision.String
		m.LabelLanguage = labelLanguage.String
		m.LabelModel = labelModel.String

		if labelsJSON != "" && labelsJSON != "{}" {
			json.Unmarshal([]byte(labelsJSON), &m.Labels)
		}
		if resourceAttrsJSON != "" && resourceAttrsJSON != "{}" {
			json.Unmarshal([]byte(resourceAttrsJSON), &m.ResourceAttributes)
		}

		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// GetSessionMetricsSummary aggregates all metrics for a session.
func (s *Store) GetSessionMetricsSummary(sessionID string) (*SessionMetricsSummary, error) {
	summary := &SessionMetricsSummary{SessionID: sessionID}

	// Aggregate token and cost from spans
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(tokens_in), 0),
		       COALESCE(SUM(tokens_out), 0),
		       COALESCE(SUM(cache_read_tokens), 0),
		       COALESCE(SUM(cache_creation_tokens), 0),
		       COALESCE(SUM(cost_usd), 0),
		       COALESCE(SUM(duration_ms), 0),
		       COUNT(*),
		       COUNT(CASE WHEN status = 'error' THEN 1 END)
		FROM spans
		WHERE json_extract(attributes, '$.session.id') = ?
	`, sessionID).Scan(
		&summary.TokensIn,
		&summary.TokensOut,
		&summary.CacheReadTokens,
		&summary.CacheCreationTokens,
		&summary.TotalCostUSD,
		&summary.DurationMs,
		&summary.SpanCount,
		&summary.ErrorCount,
	)
	if err != nil {
		return nil, err
	}

	// Calculate cache savings
	if summary.CacheReadTokens > 0 {
		// Get model from first span to estimate savings
		var model sql.NullString
		s.db.QueryRow(`
			SELECT model FROM spans
			WHERE json_extract(attributes, '$.session.id') = ? AND model IS NOT NULL AND model != ''
			LIMIT 1
		`, sessionID).Scan(&model)

		if model.Valid && model.String != "" {
			summary.CacheSavingsUSD = CalculateCacheSavings(model.String, summary.CacheReadTokens)
		}
	}

	// Count turns and tool calls from spans
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT json_extract(attributes, '$.turn.number')),
		       COUNT(CASE WHEN name LIKE 'exec.tool_use%' THEN 1 END)
		FROM spans
		WHERE json_extract(attributes, '$.session.id') = ?
	`, sessionID).Scan(&summary.TurnCount, &summary.ToolCalls)
	if err != nil {
		return nil, err
	}

	// Aggregate LOC from metrics
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(CASE WHEN label_type = 'added' THEN value_int ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN label_type = 'removed' THEN value_int ELSE 0 END), 0)
		FROM metrics
		WHERE session_id = ? AND name = 'claude_code.lines_of_code.count'
	`, sessionID).Scan(&summary.LinesAdded, &summary.LinesRemoved)
	if err != nil {
		return nil, err
	}

	// Aggregate commit and PR counts
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(CASE WHEN name = 'claude_code.commit.count' THEN value_int ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN name = 'claude_code.pull_request.count' THEN value_int ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN name = 'claude_code.active_time.total' THEN value_int ELSE 0 END), 0)
		FROM metrics
		WHERE session_id = ?
	`, sessionID).Scan(&summary.CommitCount, &summary.PullRequestCount, &summary.ActiveTimeMs)
	if err != nil {
		return nil, err
	}

	// Calculate success rate
	if summary.SpanCount > 0 {
		summary.SuccessRate = float64(summary.SpanCount-summary.ErrorCount) / float64(summary.SpanCount)
	}

	return summary, nil
}
