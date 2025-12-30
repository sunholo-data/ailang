package messaging

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// MessageExecutionStats represents metrics extracted from a single result message
type MessageExecutionStats struct {
	DurationMS   int      `json:"duration_ms"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	CostCents    int      `json:"cost_cents"`
	FilesCreated []string `json:"files_created"`
}

// AggregatedMetrics represents pre-computed metrics for a scope
type AggregatedMetrics struct {
	ScopeType     string  `json:"scope_type"`
	ScopeID       string  `json:"scope_id"`
	TotalRuns     int     `json:"total_runs"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"` // Dollars (cents / 100)
	TotalDuration int     `json:"total_duration_ms"`
	TotalFiles    int     `json:"total_files_modified"`
	AvgTokens     float64 `json:"avg_tokens_per_run"`
	AvgCost       float64 `json:"avg_cost_per_run"`
	AvgDuration   float64 `json:"avg_duration_per_run"`
	PendingTasks  int     `json:"pending_tasks"` // Number of currently running/pending tasks
}

// ParseMessageExecutionStats extracts execution stats from message metadata JSON
func ParseMessageExecutionStats(metadataJSON string) (*MessageExecutionStats, error) {
	if metadataJSON == "" {
		return nil, nil
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	execStats, ok := metadata["execution_stats"].(map[string]interface{})
	if !ok {
		return nil, nil // No execution stats in metadata
	}

	stats := &MessageExecutionStats{}

	// Parse numeric fields
	if v, ok := execStats["duration_ms"].(float64); ok {
		stats.DurationMS = int(v)
	}
	if v, ok := execStats["input_tokens"].(float64); ok {
		stats.InputTokens = int(v)
	}
	if v, ok := execStats["output_tokens"].(float64); ok {
		stats.OutputTokens = int(v)
	}
	// Cost might be in dollars (float) - convert to cents
	if v, ok := execStats["cost"].(float64); ok {
		stats.CostCents = int(v * 100)
	}

	// Parse files_created array
	if files, ok := execStats["files_created"].([]interface{}); ok {
		for _, f := range files {
			if s, ok := f.(string); ok {
				stats.FilesCreated = append(stats.FilesCreated, s)
			}
		}
	}

	return stats, nil
}

// RecordMetrics updates aggregated metrics when a result message is created
// This should be called after creating a result message
func (s *Store) RecordMetrics(threadID, agentID string, stats *MessageExecutionStats) error {
	if stats == nil {
		return nil
	}

	now := time.Now()
	nowMS := now.UnixMilli()

	// Calculate period starts for minute, hour, day
	minuteStart := now.Truncate(time.Minute).UnixMilli()
	hourStart := now.Truncate(time.Hour).UnixMilli()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()

	totalTokens := stats.InputTokens + stats.OutputTokens
	filesCount := len(stats.FilesCreated)

	// Upsert metrics for all scopes and periods
	scopes := []struct {
		scopeType string
		scopeID   string
	}{
		{"global", ""},
		{"agent", agentID},
		{"thread", threadID},
	}

	periods := []struct {
		name  string
		start int64
	}{
		{"minute", minuteStart},
		{"hour", hourStart},
		{"day", dayStart},
	}

	for _, scope := range scopes {
		for _, period := range periods {
			id := generateID(fmt.Sprintf("%s_%s_%s_%d", scope.scopeType, scope.scopeID, period.name, period.start))

			// Use UPSERT to atomically update metrics
			_, err := s.db.Exec(`
				INSERT INTO metrics_aggregates (
					id, scope_type, scope_id, period, period_start,
					total_runs, total_tokens, total_cost_cents, total_duration_ms, total_files_modified,
					created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(scope_type, scope_id, period, period_start) DO UPDATE SET
					total_runs = total_runs + 1,
					total_tokens = total_tokens + excluded.total_tokens,
					total_cost_cents = total_cost_cents + excluded.total_cost_cents,
					total_duration_ms = total_duration_ms + excluded.total_duration_ms,
					total_files_modified = total_files_modified + excluded.total_files_modified,
					avg_tokens_per_run = CAST((total_tokens + excluded.total_tokens) AS REAL) / (total_runs + 1),
					avg_cost_per_run = CAST((total_cost_cents + excluded.total_cost_cents) AS REAL) / 100.0 / (total_runs + 1),
					avg_duration_per_run = CAST((total_duration_ms + excluded.total_duration_ms) AS REAL) / (total_runs + 1),
					updated_at = excluded.updated_at
			`, id, scope.scopeType, scope.scopeID, period.name, period.start,
				totalTokens, stats.CostCents, stats.DurationMS, filesCount,
				nowMS, nowMS)

			if err != nil {
				return fmt.Errorf("failed to upsert metrics for %s/%s: %w", scope.scopeType, scope.scopeID, err)
			}
		}
	}

	return nil
}

// GetMetrics returns aggregated metrics for a given scope
func (s *Store) GetMetrics(scopeType, scopeID string) (*AggregatedMetrics, error) {
	// Sum across all periods for the current day
	dayStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location()).UnixMilli()

	row := s.db.QueryRow(`
		SELECT
			COALESCE(SUM(total_runs), 0),
			COALESCE(SUM(total_tokens), 0),
			COALESCE(SUM(total_cost_cents), 0),
			COALESCE(SUM(total_duration_ms), 0),
			COALESCE(SUM(total_files_modified), 0)
		FROM metrics_aggregates
		WHERE scope_type = ? AND scope_id = ? AND period = 'day' AND period_start >= ?
	`, scopeType, scopeID, dayStart)

	var totalRuns, totalTokens, totalCostCents, totalDuration, totalFiles int
	if err := row.Scan(&totalRuns, &totalTokens, &totalCostCents, &totalDuration, &totalFiles); err != nil {
		if err == sql.ErrNoRows {
			return &AggregatedMetrics{
				ScopeType: scopeType,
				ScopeID:   scopeID,
			}, nil
		}
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics := &AggregatedMetrics{
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		TotalRuns:     totalRuns,
		TotalTokens:   totalTokens,
		TotalCost:     float64(totalCostCents) / 100.0,
		TotalDuration: totalDuration,
		TotalFiles:    totalFiles,
	}

	if totalRuns > 0 {
		metrics.AvgTokens = float64(totalTokens) / float64(totalRuns)
		metrics.AvgCost = metrics.TotalCost / float64(totalRuns)
		metrics.AvgDuration = float64(totalDuration) / float64(totalRuns)
	}

	return metrics, nil
}

// GetGlobalMetrics returns global aggregated metrics
func (s *Store) GetGlobalMetrics() (*AggregatedMetrics, error) {
	return s.GetMetrics("global", "")
}

// GetAgentMetrics returns metrics for a specific agent
func (s *Store) GetAgentMetrics(agentID string) (*AggregatedMetrics, error) {
	return s.GetMetrics("agent", agentID)
}

// GetThreadMetrics returns metrics for a specific thread
func (s *Store) GetThreadMetrics(threadID string) (*AggregatedMetrics, error) {
	return s.GetMetrics("thread", threadID)
}

// GetMetricsTrends returns time-series metrics for a given scope and period
func (s *Store) GetMetricsTrends(scopeType, scopeID, period string, limit int) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`
		SELECT period_start, total_runs, total_tokens, total_cost_cents, total_duration_ms
		FROM metrics_aggregates
		WHERE scope_type = ? AND scope_id = ? AND period = ?
		ORDER BY period_start DESC
		LIMIT ?
	`, scopeType, scopeID, period, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}
	defer rows.Close()

	var trends []map[string]interface{}
	for rows.Next() {
		var periodStart int64
		var runs, tokens, costCents, duration int
		if err := rows.Scan(&periodStart, &runs, &tokens, &costCents, &duration); err != nil {
			return nil, fmt.Errorf("failed to scan trend row: %w", err)
		}
		trends = append(trends, map[string]interface{}{
			"period_start": periodStart,
			"runs":         runs,
			"tokens":       tokens,
			"cost":         float64(costCents) / 100.0,
			"duration_ms":  duration,
		})
	}

	// Reverse to chronological order
	for i, j := 0, len(trends)-1; i < j; i, j = i+1, j-1 {
		trends[i], trends[j] = trends[j], trends[i]
	}

	return trends, nil
}

// Helper to generate a unique ID
func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read always returns nil error on modern systems
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
