package observatory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

// retentionChunkSize controls how many rows a single DELETE statement removes
// before retention issues a WAL checkpoint. Chunking keeps the WAL from
// ballooning when another process (e.g. 'ailang serve') holds a reader
// connection that pins the oldest WAL snapshot. A single DELETE of 100k rows
// in WAL mode can produce a 1GB+ WAL file that can't be drained until the
// reader goes away; chunked deletes with periodic checkpoints avoid that.
const retentionChunkSize = 5000

// RunRetention deletes rows older than their TTL and checkpoints the WAL.
//
// TTLs:
//   - spans, trace_summaries, metrics: 7 days
//   - chat_messages, session_tools:    30 days
//
// Time columns are stored as TEXT (ISO-8601 via the go-sqlite3 driver), so
// comparisons go through SQLite's datetime() function on both sides. Comparing
// a TEXT column directly against an int64 Unix/UnixNano cutoff silently matches
// zero rows — SQLite's storage-class ordering always ranks INTEGER < TEXT.
//
// Deletions are chunked to avoid WAL bloat when another connection holds back
// the checkpoint. VACUUM is not attempted automatically: it requires an
// exclusive lock that the running server process blocks, and the failure modes
// are ugly (temp file growth, SQLITE_BUSY loops). VACUUM should be run
// manually during a maintenance window when 'ailang serve' is stopped.
//
// Called periodically by the coordinator daemon and on startup if the DB is
// oversized.
func (s *Store) RunRetention(ctx context.Context) (RetentionStats, error) {
	var stats RetentionStats
	var errs []error

	// Each entry: stats pointer, DELETE template (with %d chunk limit), label.
	// The LIMIT clause requires SQLite built with SQLITE_ENABLE_UPDATE_DELETE_LIMIT
	// (which go-sqlite3 enables by default).
	steps := []struct {
		target    *int64
		queryTmpl string
		label     string
	}{
		{
			&stats.SpansDeleted,
			`DELETE FROM spans
			   WHERE rowid IN (
			     SELECT rowid FROM spans
			      WHERE datetime(start_time) < datetime('now', '-7 days')
			      LIMIT %d
			   )`,
			"spans",
		},
		{
			&stats.SummariesDeleted,
			`DELETE FROM trace_summaries
			   WHERE trace_id IN (
			     SELECT trace_id FROM trace_summaries
			      WHERE datetime(start_time) < datetime('now', '-7 days')
			      LIMIT %d
			   )`,
			"trace_summaries",
		},
		{
			&stats.MetricsDeleted,
			`DELETE FROM metrics
			   WHERE rowid IN (
			     SELECT rowid FROM metrics
			      WHERE datetime(timestamp) < datetime('now', '-7 days')
			      LIMIT %d
			   )`,
			"metrics",
		},
		{
			&stats.ChatDeleted,
			`DELETE FROM chat_messages
			   WHERE rowid IN (
			     SELECT rowid FROM chat_messages
			      WHERE datetime(created_at) < datetime('now', '-30 days')
			      LIMIT %d
			   )`,
			"chat_messages",
		},
		{
			// session_tools has no created_at column — use start_time.
			&stats.ToolsDeleted,
			`DELETE FROM session_tools
			   WHERE rowid IN (
			     SELECT rowid FROM session_tools
			      WHERE datetime(start_time) < datetime('now', '-30 days')
			      LIMIT %d
			   )`,
			"session_tools",
		},
	}

	for _, step := range steps {
		n, err := deleteInChunks(ctx, s.db, step.queryTmpl, retentionChunkSize)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", step.label, err))
			continue
		}
		*step.target = n
	}

	// Final passive checkpoint. Best-effort: will do nothing if a reader is
	// holding back the WAL, but that's fine — the per-chunk checkpoints in
	// deleteInChunks did the real work.
	if _, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)"); err != nil {
		errs = append(errs, fmt.Errorf("wal_checkpoint: %w", err))
	}

	if len(errs) > 0 {
		return stats, errors.Join(errs...)
	}
	return stats, nil
}

// deleteInChunks runs a DELETE template with a chunk limit, looping until no
// more rows match. Issues a PASSIVE WAL checkpoint between chunks so the WAL
// file doesn't grow unbounded while long-running readers are attached.
func deleteInChunks(ctx context.Context, db *sql.DB, queryTmpl string, chunk int) (int64, error) {
	query := fmt.Sprintf(queryTmpl, chunk)
	var total int64
	// Upper bound on iterations: avoids a pathological infinite loop if
	// something causes the DELETE to always report > 0 but never shrink.
	// 10,000 iterations * 5,000 chunk = 50M rows, which is 100x larger than
	// any observed observatory DB.
	const maxIters = 10000
	for i := 0; i < maxIters; i++ {
		res, err := db.ExecContext(ctx, query)
		if err != nil {
			// Check for "no such column"-type errors that indicate a schema
			// mismatch (e.g. session_tools.created_at from the old retention
			// code). Those should not be retried.
			if strings.Contains(err.Error(), "no such column") ||
				strings.Contains(err.Error(), "no such table") {
				return total, err
			}
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		if n == 0 {
			break
		}
		total += n

		// Drain the WAL between chunks. PASSIVE won't block on readers;
		// it drains as much as it can and returns.
		_, _ = db.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)")
	}
	return total, nil
}
