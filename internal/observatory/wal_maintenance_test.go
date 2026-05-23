package observatory

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWALSizeMB_NoFile returns 0 when the WAL file doesn't exist.
func TestWALSizeMB_NoFile(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "nonexistent.db")
	if got := WALSizeMB(dbPath); got != 0 {
		t.Errorf("WALSizeMB(no file) = %d, want 0", got)
	}
}

// TestWALSizeMB_ReadsRealSize seeds a fake WAL file of known byte count
// and asserts the MB rounding matches.
func TestWALSizeMB_ReadsRealSize(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	walPath := dbPath + "-wal"
	// Write a 5 MB sentinel
	data := make([]byte, 5*1024*1024)
	if err := os.WriteFile(walPath, data, 0644); err != nil {
		t.Fatalf("seed WAL: %v", err)
	}
	if got, want := WALSizeMB(dbPath), int64(5); got != want {
		t.Errorf("WALSizeMB(5MB seed) = %d, want %d", got, want)
	}
}

// TestWalCheckpointThreshold_Default returns the constant when env unset.
func TestWalCheckpointThreshold_Default(t *testing.T) {
	prev, hadPrev := os.LookupEnv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB")
	os.Unsetenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB")
	defer func() {
		if hadPrev {
			os.Setenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB", prev)
		}
	}()
	if got, want := walCheckpointThresholdMB(), int64(DefaultWALCheckpointThresholdMB); got != want {
		t.Errorf("walCheckpointThresholdMB() = %d, want %d", got, want)
	}
}

// TestWalCheckpointThreshold_EnvOverride honors a positive integer env value.
func TestWalCheckpointThreshold_EnvOverride(t *testing.T) {
	t.Setenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB", "512")
	if got, want := walCheckpointThresholdMB(), int64(512); got != want {
		t.Errorf("walCheckpointThresholdMB() with env=512 = %d, want %d", got, want)
	}
}

// TestWalCheckpointThreshold_InvalidEnvFallsBack ignores garbage/zero/negative
// values and falls back to the default.
func TestWalCheckpointThreshold_InvalidEnvFallsBack(t *testing.T) {
	cases := []string{"not-a-number", "0", "-100", ""}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			t.Setenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB", v)
			if got, want := walCheckpointThresholdMB(), int64(DefaultWALCheckpointThresholdMB); got != want {
				t.Errorf("threshold with env=%q = %d, want default %d", v, got, want)
			}
		})
	}
}

// TestMaybeCheckpointWAL_BelowThreshold no-ops without opening the DB.
// We seed a small (1MB) WAL and a high (10GB) threshold so the fast path runs.
func TestMaybeCheckpointWAL_BelowThreshold(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	walPath := dbPath + "-wal"
	if err := os.WriteFile(walPath, make([]byte, 1*1024*1024), 0644); err != nil {
		t.Fatalf("seed WAL: %v", err)
	}
	t.Setenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB", "10000")
	before, after, did := MaybeCheckpointWAL(dbPath)
	if did {
		t.Errorf("did checkpoint when below threshold")
	}
	if before != 1 || after != 1 {
		t.Errorf("before=%d after=%d, want 1/1", before, after)
	}
}

// TestMaybeCheckpointWAL_AboveThreshold_NoRealDB attempts the checkpoint
// on a fake DB and asserts graceful behavior. A real WAL needs a real DB
// for sqlite to checkpoint into; here we just confirm the function does
// NOT panic and reports the error path cleanly.
//
// Real end-to-end coverage is by integration: server runs CheckHealth on
// startup, which now invokes MaybeCheckpointWAL. The 2026-05-23 manual
// verification (5.4 GB WAL -> 0 bytes after sqlite3 PRAGMA wal_checkpoint
// TRUNCATE) confirms the underlying SQLite mechanism works as expected.
func TestMaybeCheckpointWAL_AboveThreshold_NoRealDB(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	walPath := dbPath + "-wal"
	// Seed a "5 MB WAL" with no real DB behind it.
	if err := os.WriteFile(walPath, make([]byte, 5*1024*1024), 0644); err != nil {
		t.Fatalf("seed WAL: %v", err)
	}
	t.Setenv("AILANG_OBSERVATORY_WAL_CHECKPOINT_MB", "1") // force trigger
	before, _, _ := MaybeCheckpointWAL(dbPath)
	if before != 5 {
		t.Errorf("before = %d, want 5", before)
	}
	// We don't assert on after / did: behavior depends on whether sqlite
	// will open a missing DB (creates it), and whether the WAL header is
	// valid. The test's job is to confirm no panic and the call returns.
}
