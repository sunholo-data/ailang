package observatory

import (
	"context"
	"fmt"
	"log"
	"os"
)

// HealthAction describes what action to take based on DB size.
type HealthAction string

const (
	HealthOK       HealthAction = "ok"
	HealthWarn     HealthAction = "warn"
	HealthCleanup  HealthAction = "cleanup"
	HealthRecreate HealthAction = "recreate"
)

// CheckHealth examines the observatory DB size and takes action.
// Fast path: just os.Stat (no DB open). Only opens DB if cleanup needed.
//
// Thresholds:
//   - < 200MB: ok (log size)
//   - 200-500MB: warn
//   - 500MB-2GB: auto-cleanup (run retention)
//   - > 2GB: recreate (delete DB files)
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
		log.Printf("Observatory: %dMB (>2GB) — recreating database", totalMB)
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")

	case totalMB > 500:
		log.Printf("Observatory: %dMB (>500MB) — running retention cleanup", totalMB)
		store, err := OpenStore(dbPath)
		if err != nil {
			log.Printf("Warning: failed to open observatory for cleanup: %v", err)
			return
		}
		defer store.Close()
		stats, err := store.RunRetention(context.Background())
		if err != nil {
			log.Printf("Warning: retention cleanup failed: %v", err)
		} else if stats.Total() > 0 {
			log.Printf("Observatory cleanup: deleted %s", stats)
		}

	case totalMB > 200:
		log.Printf("Observatory: %dMB (warn threshold: 200MB)", totalMB)

	default:
		// Log size for visibility but don't warn
		if totalMB > 0 {
			fmt.Fprintf(os.Stderr, "Observatory: %dMB\n", totalMB)
		}
	}
}
