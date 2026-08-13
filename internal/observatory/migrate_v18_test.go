package observatory

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

// M-OPENROUTER-BROADCAST-INGEST M2.
//
// The pair below was read off the PRODUCTION observatory on 2026-08-13, so
// these tests are anchored to real corrupted data rather than to a value
// synthesized from the same code that repairs it.
const (
	prodCorruptedTraceID = "6bb6b86b6d3ad37e1bd9fe1f6b5736ebd6b9f766f8e1a6bd" // 48 chars
	prodRecoveredTraceID = "a7a4a206034b2f4fa1c269a592b44aa9"                 // 32 chars

	// corruptedSpanIDColliding repairs to knownSpanIDHex ("051581bf3cb55c13").
	corruptedSpanIDColliding = "d39d79f356dfddc6f9e5cd77" // 24 chars
	// corruptedSpanIDNoCollision repairs to "aabbccddeeff0011".
	corruptedSpanIDNoCollision = "69a6db71c75d79e7dfd34d75" // 24 chars
)

// openMigratedDB returns an in-memory DB migrated to the current schema.
func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteBackendFromPath: %v", err)
	}
	t.Cleanup(func() { _ = backend.Close() })
	return backend.DB()
}

// insertSpanRow writes a span row directly, bypassing the receiver, so a
// corrupted ID can be seeded exactly as production holds it.
func insertSpanRow(t *testing.T, db *sql.DB, id, traceID, parentID string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO spans (id, trace_id, parent_span_id, name, kind, status, start_time)
		 VALUES (?, ?, ?, ?, 'internal', 'unset', ?)`,
		id, traceID, parentID, "seeded.span", time.Now().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("insert span %q: %v", id, err)
	}
}

// spansChecksum hashes the full spans table so idempotence can be asserted by
// comparison rather than by inspection.
func spansChecksum(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT id, trace_id, COALESCE(parent_span_id,''), name FROM spans ORDER BY id`)
	if err != nil {
		t.Fatalf("checksum query: %v", err)
	}
	defer rows.Close()

	h := sha256.New()
	for rows.Next() {
		var id, traceID, parentID, name string
		if err := rows.Scan(&id, &traceID, &parentID, &name); err != nil {
			t.Fatalf("checksum scan: %v", err)
		}
		fmt.Fprintf(h, "%s|%s|%s|%s\n", id, traceID, parentID, name)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func spanCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM spans").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// TestMigrateV18_RecoversProductionTraceID is the anchor test: the exact
// corrupted value production held must recover to the exact original.
func TestMigrateV18_RecoversProductionTraceID(t *testing.T) {
	db := openMigratedDB(t)
	insertSpanRow(t, db, "d39d79f356dfddc6f9e5cd77", prodCorruptedTraceID, "")

	repaired, _, err := repairCorruptedSpanIDs(db)
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if repaired != 1 {
		t.Errorf("repaired %d rows, want 1", repaired)
	}

	var gotID, gotTrace string
	if err := db.QueryRow("SELECT id, trace_id FROM spans").Scan(&gotID, &gotTrace); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if gotTrace != prodRecoveredTraceID {
		t.Errorf("trace_id = %s, want %s", gotTrace, prodRecoveredTraceID)
	}
	if len(gotID) != CorrectSpanIDHexLen {
		t.Errorf("span id = %s (%d chars), want %d", gotID, len(gotID), CorrectSpanIDHexLen)
	}
}

// TestMigrateV18_Idempotent asserts the second run is a no-op, by checksum.
func TestMigrateV18_Idempotent(t *testing.T) {
	db := openMigratedDB(t)
	insertSpanRow(t, db, "d39d79f356dfddc6f9e5cd77", prodCorruptedTraceID, "")

	if _, _, err := repairCorruptedSpanIDs(db); err != nil {
		t.Fatalf("first repair: %v", err)
	}
	afterFirst := spansChecksum(t, db)

	repaired, _, err := repairCorruptedSpanIDs(db)
	if err != nil {
		t.Fatalf("second repair: %v", err)
	}
	if repaired != 0 {
		t.Errorf("second run repaired %d rows, want 0 — the migration is not idempotent", repaired)
	}
	if afterSecond := spansChecksum(t, db); afterSecond != afterFirst {
		t.Errorf("checksum changed on second run:\n  first  = %s\n  second = %s", afterFirst, afterSecond)
	}
}

// TestMigrateV18_LeavesCorrectRowsAlone is the guard that matters most: a row
// already in the right encoding must be provably untouched.
func TestMigrateV18_LeavesCorrectRowsAlone(t *testing.T) {
	db := openMigratedDB(t)

	// One correct row, one corrupted row. The corrupted span id repairs to
	// "aabbccddeeff0011", deliberately NOT the correct row's id — collision is
	// covered separately by TestMigrateV18_SkipsIDCollision.
	insertSpanRow(t, db, knownSpanIDHex, knownTraceIDHex, "")
	insertSpanRow(t, db, corruptedSpanIDNoCollision, prodCorruptedTraceID, "")

	repaired, skipped, err := repairCorruptedSpanIDs(db)
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if repaired != 1 {
		t.Errorf("repaired %d rows, want exactly 1 (only the corrupted one)", repaired)
	}
	if skipped != 0 {
		t.Errorf("skipped %d rows, want 0", skipped)
	}

	var gotTrace string
	if err := db.QueryRow("SELECT trace_id FROM spans WHERE id = ?", knownSpanIDHex).Scan(&gotTrace); err != nil {
		t.Fatalf("correct row vanished: %v", err)
	}
	if gotTrace != knownTraceIDHex {
		t.Errorf("already-correct trace_id was modified: %s -> %s", knownTraceIDHex, gotTrace)
	}
}

// TestMigrateV18_SkipsIDCollision covers the case where the same span was
// ingested under BOTH encodings: the corrupted row's repaired id already
// exists.
//
// Rewriting it would violate the primary key and roll back every other repair,
// so the row must be skipped and counted — not dropped, and not allowed to
// abort the migration. corruptedSpanIDColliding repairs to exactly
// knownSpanIDHex, which the first row already occupies.
func TestMigrateV18_SkipsIDCollision(t *testing.T) {
	db := openMigratedDB(t)

	insertSpanRow(t, db, knownSpanIDHex, knownTraceIDHex, "")
	insertSpanRow(t, db, corruptedSpanIDColliding, prodCorruptedTraceID, "")

	before := spanCount(t, db)

	repaired, skipped, err := repairCorruptedSpanIDs(db)
	if err != nil {
		t.Fatalf("a collision must be skipped, not returned as an error: %v", err)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
	if repaired != 0 {
		t.Errorf("repaired = %d, want 0 (the only candidate collided)", repaired)
	}
	if after := spanCount(t, db); after != before {
		t.Errorf("span count went %d -> %d; a skipped row must not be deleted", before, after)
	}

	// The other row's repairs must have survived the skip.
	var gotTrace string
	if err := db.QueryRow("SELECT trace_id FROM spans WHERE id = ?", knownSpanIDHex).Scan(&gotTrace); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if gotTrace != knownTraceIDHex {
		t.Errorf("trace_id = %s, want %s", gotTrace, knownTraceIDHex)
	}
}

// TestMigrateV18_PreservesSpanCount pins that this is a repair, never a delete.
func TestMigrateV18_PreservesSpanCount(t *testing.T) {
	db := openMigratedDB(t)
	insertSpanRow(t, db, "d39d79f356dfddc6f9e5cd77", prodCorruptedTraceID, "")
	insertSpanRow(t, db, "aabbccddeeff0011", knownTraceIDHex, "")
	insertSpanRow(t, db, "d39d79f356dfddc6f9e5cd78", prodCorruptedTraceID, "")

	before := spanCount(t, db)
	if _, _, err := repairCorruptedSpanIDs(db); err != nil {
		t.Fatalf("repair: %v", err)
	}
	if after := spanCount(t, db); after != before {
		t.Errorf("span count went %d -> %d; the migration must never delete", before, after)
	}
}

// TestMigrateV18_NoCorruptedRowsIsNoOp covers the clean-DB path.
func TestMigrateV18_NoCorruptedRowsIsNoOp(t *testing.T) {
	db := openMigratedDB(t)
	insertSpanRow(t, db, knownSpanIDHex, knownTraceIDHex, "")

	repaired, _, err := repairCorruptedSpanIDs(db)
	if err != nil {
		t.Fatalf("repair on a clean DB must not error: %v", err)
	}
	if repaired != 0 {
		t.Errorf("repaired %d rows on a clean DB, want 0", repaired)
	}
}

// TestMigrateV18_RepairsParentSpanID covers the third ID column.
func TestMigrateV18_RepairsParentSpanID(t *testing.T) {
	db := openMigratedDB(t)
	corruptedParent := "d39d79f356dfddc6f9e5cd77" // 24 chars
	insertSpanRow(t, db, knownSpanIDHex, knownTraceIDHex, corruptedParent)

	if _, _, err := repairCorruptedSpanIDs(db); err != nil {
		t.Fatalf("repair: %v", err)
	}

	var gotParent string
	if err := db.QueryRow("SELECT parent_span_id FROM spans WHERE id = ?", knownSpanIDHex).Scan(&gotParent); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(gotParent) != CorrectSpanIDHexLen {
		t.Errorf("parent_span_id = %s (%d chars), want %d", gotParent, len(gotParent), CorrectSpanIDHexLen)
	}
}
