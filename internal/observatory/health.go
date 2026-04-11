package observatory

import (
	"context"
	"log"
	"os"
)

// HealthAction describes what action to take based on DB size.
type HealthAction string

const (
	HealthOK      HealthAction = "ok"
	HealthWarn    HealthAction = "warn"
	HealthCleanup HealthAction = "cleanup"
	HealthDanger  HealthAction = "danger"
)

// CheckHealth examines the observatory DB size and takes action.
// Fast path: just os.Stat (no DB open). Only opens the DB if cleanup is needed.
//
// Thresholds:
//   - < 200MB:       ok (no output)
//   - 200-500MB:     warn (log only)
//   - 500MB-2GB:     auto-cleanup (run retention)
//   - > 2GB:         loud warning, no destructive action
//
// The >2GB branch intentionally does NOT delete the database. The previous
// implementation called os.Remove(dbPath) on this threshold, which destroyed
// production data if the WAL bloated past 2GB (e.g. when retention issued a
// large DELETE with a long-lived reader holding back the WAL checkpoint).
// Silent data destruction violates the project's "no silent fallbacks" rule.
// If the DB is genuinely >2GB, the user needs to intervene (stop the server,
// VACUUM, etc.) — we surface the problem loudly and leave the data alone.
func CheckHealth(dbPath string) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return // DB doesn't exist yet
	}

	sizeMB := info.Size() / (1024 * 1024)

	// Also check WAL size
	walMB := int64(0)
	if walInfo, err := os.Stat(dbPath + "-wal"); err == nil {
		walMB = walInfo.Size() / (1024 * 1024)
	}

	totalMB := sizeMB + walMB

	switch {
	case totalMB > 2048:
		log.Printf("Observatory: %dMB (DB=%dMB WAL=%dMB) — DB exceeds 2GB threshold.",
			totalMB, sizeMB, walMB)
		log.Printf("Observatory: NOT auto-deleting. To reclaim space manually:")
		log.Printf("  1. Stop 'ailang serve' (releases the WAL reader holding back checkpoints)")
		log.Printf("  2. sqlite3 %s 'PRAGMA wal_checkpoint(TRUNCATE); VACUUM;'", dbPath)
		log.Printf("  3. Or delete the DB files if the data is disposable:")
		log.Printf("     rm %s %s-wal %s-shm", dbPath, dbPath, dbPath)

	case totalMB > 500:
		log.Printf("Observatory: %dMB (DB=%dMB WAL=%dMB) — running retention cleanup",
			totalMB, sizeMB, walMB)
		store, err := OpenStore(dbPath)
		if err != nil {
			log.Printf("Warning: failed to open observatory for cleanup: %v", err)
			return
		}
		defer store.Close()
		stats, err := store.RunRetention(context.Background())
		if err != nil {
			// Log the error but still report stats — RunRetention collects
			// per-table errors and returns whatever succeeded.
			log.Printf("Warning: retention cleanup had errors: %v", err)
		}
		// Always report the outcome, including no-ops, so silent failures
		// stay visible. Also report the new on-disk size.
		newMB := totalMB
		if info2, err := os.Stat(dbPath); err == nil {
			walMB2 := int64(0)
			if walInfo2, err := os.Stat(dbPath + "-wal"); err == nil {
				walMB2 = walInfo2.Size() / (1024 * 1024)
			}
			newMB = info2.Size()/(1024*1024) + walMB2
		}
		log.Printf("Observatory cleanup: deleted %s (size %dMB → %dMB)",
			stats, totalMB, newMB)

	case totalMB > 200:
		log.Printf("Observatory: %dMB (warn threshold: 200MB)", totalMB)

	default:
		// Size is healthy, no output needed
	}
}
