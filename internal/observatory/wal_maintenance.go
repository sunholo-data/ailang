// Package observatory: WAL maintenance for unbounded-WAL prevention.
//
// M-EVAL-OS-LONGITUDINAL M0 (2026-05-23): the existing CheckHealth path
// detects bloated databases but only WARNS at the dangerous threshold,
// telling the user to manually run `PRAGMA wal_checkpoint(TRUNCATE)`.
// During a 12-hour overnight eval the WAL ballooned to 5.4 GB because no
// process was reading the suggestion at 02:00 in the morning.
//
// This file adds a non-destructive auto-checkpoint that fires whenever the
// WAL alone exceeds a configurable threshold (default 1 GB). The checkpoint
// merges WAL pages into the main DB file and truncates the WAL — purely
// reclamation, no data loss. Distinct from retention cleanup (which deletes
// old rows): this just flushes pending writes that are already committed.
package observatory

import (
	"database/sql"
	"log"
	"os"
	"strconv"
)

// DefaultWALCheckpointThresholdMB is the WAL size at which auto-checkpoint
// triggers. Tuned for the eval rig: well below the 2GB total-DB warning
// threshold so we never reach the manual-intervention state under normal
// rotation load.
//
// Override at runtime via AILANG_OBSERVATORY_WAL_CHECKPOINT_MB env var.
const DefaultWALCheckpointThresholdMB = 1024

// walCheckpointThresholdMB returns the configured WAL checkpoint threshold,
// honoring the env override. Falls back to DefaultWALCheckpointThresholdMB
// if the env var is unset, malformed, or non-positive.
func walCheckpointThresholdMB() int64 {
	if v := os.Getenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return DefaultWALCheckpointThresholdMB
}

// WALSizeMB returns the size of the WAL file in MB, or 0 if the file
// doesn't exist or is unreadable. Pure stat call — no DB open.
func WALSizeMB(dbPath string) int64 {
	info, err := os.Stat(dbPath + "-wal")
	if err != nil {
		return 0
	}
	return info.Size() / (1024 * 1024)
}

// MaybeCheckpointWAL inspects the WAL file size and runs PRAGMA
// wal_checkpoint(TRUNCATE) when it exceeds the configured threshold.
//
// Non-destructive: the checkpoint merges already-committed WAL pages into
// the main DB and truncates the WAL to zero. No row data is lost.
//
// Returns (sizeBefore, sizeAfter, didCheckpoint). When didCheckpoint is
// false, sizeAfter == sizeBefore and no DB connection was opened (fast path).
func MaybeCheckpointWAL(dbPath string) (sizeBeforeMB, sizeAfterMB int64, didCheckpoint bool) {
	sizeBeforeMB = WALSizeMB(dbPath)
	threshold := walCheckpointThresholdMB()
	if sizeBeforeMB < threshold {
		return sizeBeforeMB, sizeBeforeMB, false
	}

	log.Printf("Observatory WAL: %dMB exceeds checkpoint threshold (%dMB) — running PRAGMA wal_checkpoint(TRUNCATE)",
		sizeBeforeMB, threshold)

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Printf("Observatory WAL: failed to open DB for checkpoint: %v", err)
		return sizeBeforeMB, sizeBeforeMB, false
	}
	defer db.Close()

	// PRAGMA wal_checkpoint(TRUNCATE) returns (busy, log, checkpointed).
	// busy=1 means a reader prevented full truncation; we proceed anyway
	// because some checkpoint progress is still better than none.
	var busy, logFrames, checkpointed int
	if err := db.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
		log.Printf("Observatory WAL: checkpoint failed: %v", err)
		return sizeBeforeMB, sizeBeforeMB, false
	}

	sizeAfterMB = WALSizeMB(dbPath)
	log.Printf("Observatory WAL: checkpoint complete (busy=%d, log_frames=%d, checkpointed=%d, size %dMB -> %dMB)",
		busy, logFrames, checkpointed, sizeBeforeMB, sizeAfterMB)

	if busy != 0 {
		log.Printf("Observatory WAL: checkpoint reported busy=%d — a reader is holding the WAL. Subsequent ticks will retry.", busy)
	}

	return sizeBeforeMB, sizeAfterMB, true
}
