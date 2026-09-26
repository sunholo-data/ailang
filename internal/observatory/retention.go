package observatory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// RetentionStats reports what was cleaned up by RunRetention.
type RetentionStats struct {
	SpansDeleted     int64
	SummariesDeleted int64
	MetricsDeleted   int64
	ChatDeleted      int64
	ToolsDeleted     int64
	StagesDeleted    int64
	ChainsDeleted    int64
	SessionsDeleted  int64
}

// String formats retention stats for logging.
func (rs RetentionStats) String() string {
	return fmt.Sprintf("spans=%d summaries=%d metrics=%d chat=%d tools=%d stages=%d chains=%d sessions=%d",
		rs.SpansDeleted, rs.SummariesDeleted, rs.MetricsDeleted, rs.ChatDeleted, rs.ToolsDeleted,
		rs.StagesDeleted, rs.ChainsDeleted, rs.SessionsDeleted)
}

// Total returns the total number of rows deleted.
func (rs RetentionStats) Total() int64 {
	return rs.SpansDeleted + rs.SummariesDeleted + rs.MetricsDeleted + rs.ChatDeleted + rs.ToolsDeleted +
		rs.StagesDeleted + rs.ChainsDeleted + rs.SessionsDeleted
}

// RetentionStampSuffix names the file RunRetention touches next to the DB
// when a pass completes without error. CheckHealth reads its mtime on the
// stat-only fast path to decide whether a CLI start needs to run retention
// itself or whether the coordinator daemon's hourly tick already did — a
// 7-day span window plus a 30-day tool window sits at ~580MB on this rig,
// permanently above the cleanup threshold, and DELETE never shrinks the
// file, so without the stamp every platform command re-ran the full pass.
const RetentionStampSuffix = ".retention-stamp"

// RetentionFreshFor is how long a stamp counts as "retention already ran":
// the daemon ticks hourly, so a stamp older than this means the daemon is
// down and the CLI start should run the pass itself.
const RetentionFreshFor = time.Hour

// RetentionRanWithin reports whether a retention pass on dbPath completed
// within d. Pure stat call — no DB open.
func RetentionRanWithin(dbPath string, d time.Duration) bool {
	info, err := os.Stat(dbPath + RetentionStampSuffix)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < d
}

// stampRetention records a completed pass for RetentionRanWithin. Best
// effort: a store opened on ":memory:" or without a path has nothing to stamp.
func stampRetention(dbPath string) {
	if dbPath == "" || strings.HasPrefix(dbPath, ":memory:") {
		return
	}
	stamp := dbPath + RetentionStampSuffix
	now := time.Now()
	if err := os.Chtimes(stamp, now, now); err == nil {
		return
	}
	if f, err := os.OpenFile(stamp, os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		f.Close()
	}
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
//   - spans, trace_summaries, metrics:      7 days
//   - chat_messages, session_tools:         30 days
//   - execution_chains (+ chain_stages),
//     sessions (+ their session_tools):     90 days
//
// Chains and sessions had no TTL until 2026-09-21: chain_stages carried every
// row since May (72MB, 45MB of it eval_assessment) and sessions 24k rows. The
// 90-day window is the trade-off that `ailang chains stats --mission` with no
// --since now means "the last 90 days", not all time. Stages are deleted by
// their chain's age (not their own started_at, which is NULL on 1,351 rows)
// and before the chain row so the pass does not depend on foreign_keys=ON.
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
		{
			&stats.StagesDeleted,
			`DELETE FROM chain_stages
			   WHERE id IN (
			     SELECT cs.id FROM chain_stages cs
			       JOIN execution_chains c ON c.id = cs.chain_id
			      WHERE datetime(c.created_at) < datetime('now', '-90 days')
			      LIMIT %d
			   )`,
			"chain_stages",
		},
		{
			&stats.ChainsDeleted,
			`DELETE FROM execution_chains
			   WHERE id IN (
			     SELECT id FROM execution_chains
			      WHERE datetime(created_at) < datetime('now', '-90 days')
			      LIMIT %d
			   )`,
			"execution_chains",
		},
		{
			// A session's tools normally age out at 30 days above; this
			// catches rows whose start_time is NULL or unparseable, which the
			// 30-day step never matches, so a deleted session leaves no orphans
			// whether or not the connection enforces foreign keys.
			&stats.ToolsDeleted,
			`DELETE FROM session_tools
			   WHERE tool_use_id IN (
			     SELECT st.tool_use_id FROM session_tools st
			       JOIN sessions s ON s.session_id = st.session_id
			      WHERE datetime(s.started_at) < datetime('now', '-90 days')
			      LIMIT %d
			   )`,
			"session_tools(old sessions)",
		},
		{
			&stats.SessionsDeleted,
			`DELETE FROM sessions
			   WHERE session_id IN (
			     SELECT session_id FROM sessions
			      WHERE datetime(started_at) < datetime('now', '-90 days')
			      LIMIT %d
			   )`,
			"sessions",
		},
	}

	for _, step := range steps {
		n, err := deleteInChunks(ctx, s.db, step.queryTmpl, retentionChunkSize)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", step.label, err))
			continue
		}
		*step.target += n
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
	stampRetention(s.dbPath)
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
