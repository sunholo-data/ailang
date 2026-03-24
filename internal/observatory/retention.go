package observatory

import (
	"context"
	"fmt"
	"time"
)

// RetentionStats reports what was cleaned up by RunRetention.
type RetentionStats struct {
	SpansDeleted     int64
	SummariesDeleted int64
	MetricsDeleted   int64
	ChatDeleted      int64
	ToolsDeleted     int64
}

// String formats retention stats for logging.
func (rs RetentionStats) String() string {
	return fmt.Sprintf("spans=%d summaries=%d metrics=%d chat=%d tools=%d",
		rs.SpansDeleted, rs.SummariesDeleted, rs.MetricsDeleted, rs.ChatDeleted, rs.ToolsDeleted)
}

// Total returns the total number of rows deleted.
func (rs RetentionStats) Total() int64 {
	return rs.SpansDeleted + rs.SummariesDeleted + rs.MetricsDeleted + rs.ChatDeleted + rs.ToolsDeleted
}

// RunRetention deletes rows older than their TTL and checkpoints the WAL.
//
// TTLs:
//   - spans, trace_summaries, metrics: 7 days
//   - chat_messages, session_tools: 30 days
//
// Called periodically by the coordinator daemon and on startup if the DB is oversized.
func (s *Store) RunRetention(ctx context.Context) (RetentionStats, error) {
	var stats RetentionStats
	now := time.Now()

	spanCutoff := now.Add(-7 * 24 * time.Hour).UnixNano()
	summaryCutoff := now.Add(-7 * 24 * time.Hour).Unix()
	metricCutoff := now.Add(-7 * 24 * time.Hour).Unix()
	chatCutoff := now.Add(-30 * 24 * time.Hour).Unix()
	toolsCutoff := now.Add(-30 * 24 * time.Hour).Unix()

	// Spans: 7-day TTL (largest table — start_time is nanoseconds)
	if res, err := s.db.ExecContext(ctx, "DELETE FROM spans WHERE start_time < ?", spanCutoff); err == nil {
		stats.SpansDeleted, _ = res.RowsAffected()
	}

	// Trace summaries: 7-day TTL (derived from spans)
	if res, err := s.db.ExecContext(ctx, "DELETE FROM trace_summaries WHERE start_time < ?", summaryCutoff); err == nil {
		stats.SummariesDeleted, _ = res.RowsAffected()
	}

	// Metrics: 7-day TTL
	if res, err := s.db.ExecContext(ctx, "DELETE FROM metrics WHERE timestamp < ?", metricCutoff); err == nil {
		stats.MetricsDeleted, _ = res.RowsAffected()
	}

	// Chat messages: 30-day TTL
	if res, err := s.db.ExecContext(ctx, "DELETE FROM chat_messages WHERE created_at < ?", chatCutoff); err == nil {
		stats.ChatDeleted, _ = res.RowsAffected()
	}

	// Session tools: 30-day TTL
	if res, err := s.db.ExecContext(ctx, "DELETE FROM session_tools WHERE created_at < ?", toolsCutoff); err == nil {
		stats.ToolsDeleted, _ = res.RowsAffected()
	}

	// Checkpoint WAL to reclaim disk space
	s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)") //nolint:errcheck

	return stats, nil
}
