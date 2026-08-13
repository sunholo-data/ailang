package observatory

import (
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// migrateV18 repairs trace and span IDs corrupted by the OTLP/JSON decode
// defect fixed in M-OPENROUTER-BROADCAST-INGEST M1.
//
// Before that fix, protojson base64-decoded the hex ID strings the OTLP/JSON
// spec mandates, so every JSON-ingested row stored 1.5x the bytes it should:
//
//	trace_id       32 hex chars (16 bytes)  ->  48 hex chars (24 bytes)
//	id / parent    16 hex chars (8 bytes)   ->  24 hex chars (12 bytes)
//
// The transform is a pure function and LOSSLESS, because base64 decoding is
// injective on fixed-length input. Re-encoding the stored bytes as base64
// recovers the original hex string exactly. Verified against production rows on
// 2026-08-13:
//
//	6bb6b86b6d3ad37e1bd9fe1f6b5736ebd6b9f766f8e1a6bd  ->  a7a4a206034b2f4fa1c269a592b44aa9
//
// That is why this is a repair rather than a quarantine: no information was
// destroyed, only re-encoded.
//
// The migration is guarded strictly on length, so it is idempotent and cannot
// touch a correctly-encoded row. It never deletes: the span count before and
// after is identical.
func migrateV18(db *sql.DB, currentVersion int) (int, error) {
	repaired, skipped, err := repairCorruptedSpanIDs(db)
	if err != nil {
		return currentVersion, fmt.Errorf("v18 repair corrupted span IDs: %w", err)
	}
	if repaired > 0 {
		fmt.Printf("observatory: v18 repaired %d span row(s) corrupted by the OTLP/JSON ID defect\n", repaired)
	}
	if skipped > 0 {
		// Loud, not silent: these rows keep their corrupted IDs, so anyone
		// joining on them needs to know they exist.
		fmt.Printf("observatory: v18 SKIPPED %d span row(s) whose repaired id already exists "+
			"(the same span was ingested under both encodings); they retain their corrupted ids\n", skipped)
	}

	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (18)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 18: %w", err)
	}
	return 18, nil
}

// corruptedIDLength maps an ID's correct hex length to the length it has when
// corrupted. A 16-byte ID stored as base64-decoded-hex occupies 24 bytes, and a
// 8-byte ID occupies 12 — in both cases exactly 1.5x, rendering as 1.5x the hex
// characters.
const (
	correctTraceIDHexLen   = 32
	corruptedTraceIDHexLen = 48
	correctSpanIDHexLen    = 16
	corruptedSpanIDHexLen  = 24
)

// repairCorruptedSpanIDs rewrites every corrupted ID in the spans table and
// reports how many rows were repaired and how many were skipped.
//
// A row is SKIPPED when its repaired span id already exists — which happens if
// the same span was ingested under both encodings, e.g. an exporter retried a
// failed JSON post as protobuf. Repairing it would collide with the primary key
// and abort the whole migration, so the row keeps its corrupted id and the
// caller reports it. Deleting the duplicate is not this migration's call to
// make: it repairs, it never destroys.
//
// All updates run in one transaction: a partial repair would leave the table
// with IDs in two encodings and no way to tell them apart.
func repairCorruptedSpanIDs(db *sql.DB) (repairedCount int, skippedCount int, err error) {
	rows, err := db.Query(`
		SELECT id, trace_id, COALESCE(parent_span_id, '')
		FROM spans
		WHERE LENGTH(trace_id) = ? OR LENGTH(id) = ? OR LENGTH(parent_span_id) = ?`,
		corruptedTraceIDHexLen, corruptedSpanIDHexLen, corruptedSpanIDHexLen)
	if err != nil {
		return 0, 0, fmt.Errorf("select corrupted rows: %w", err)
	}

	type repair struct {
		oldID                      string
		newID, newTrace, newParent string
		setID, setTrace, setParent bool
	}
	var repairs []repair

	for rows.Next() {
		var id, traceID, parentID string
		if err := rows.Scan(&id, &traceID, &parentID); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan corrupted row: %w", err)
		}

		r := repair{oldID: id}
		if r.newID, r.setID, err = recoverCorruptedID(id, correctSpanIDHexLen, corruptedSpanIDHexLen); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("span id %q: %w", id, err)
		}
		if r.newTrace, r.setTrace, err = recoverCorruptedID(traceID, correctTraceIDHexLen, corruptedTraceIDHexLen); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("trace id %q: %w", traceID, err)
		}
		if r.newParent, r.setParent, err = recoverCorruptedID(parentID, correctSpanIDHexLen, corruptedSpanIDHexLen); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("parent span id %q: %w", parentID, err)
		}
		if r.setID || r.setTrace || r.setParent {
			repairs = append(repairs, r)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, fmt.Errorf("iterate corrupted rows: %w", err)
	}
	rows.Close()

	if len(repairs) == 0 {
		return 0, 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range repairs {
		// A repaired span id that already exists means this span is present
		// under both encodings. Rewriting it would violate the primary key and
		// roll back every other repair, so skip the row and let the caller
		// report it.
		if r.setID {
			var collisions int
			if err := tx.QueryRow("SELECT COUNT(*) FROM spans WHERE id = ?", r.newID).Scan(&collisions); err != nil {
				return 0, 0, fmt.Errorf("check id collision: %w", err)
			}
			if collisions > 0 {
				skippedCount++
				continue
			}
			if _, err := tx.Exec("UPDATE spans SET id = ? WHERE id = ?", r.newID, r.oldID); err != nil {
				return 0, 0, fmt.Errorf("update span id: %w", err)
			}
		}

		key := r.oldID
		if r.setID {
			key = r.newID
		}
		if r.setTrace {
			if _, err := tx.Exec("UPDATE spans SET trace_id = ? WHERE id = ?", r.newTrace, key); err != nil {
				return 0, 0, fmt.Errorf("update trace id: %w", err)
			}
		}
		if r.setParent {
			if _, err := tx.Exec("UPDATE spans SET parent_span_id = ? WHERE id = ?", r.newParent, key); err != nil {
				return 0, 0, fmt.Errorf("update parent span id: %w", err)
			}
		}
		repairedCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit: %w", err)
	}
	return repairedCount, skippedCount, nil
}

// recoverCorruptedID inverts the base64-decoded-hex corruption for one ID.
//
// It reports whether a repair applied. A value that is not exactly
// corruptedLen hex characters is left ALONE — that guard is what makes the
// migration idempotent and keeps it off correctly-encoded rows.
func recoverCorruptedID(value string, correctLen, corruptedLen int) (string, bool, error) {
	if len(value) != corruptedLen {
		return value, false, nil
	}

	raw, err := hex.DecodeString(value)
	if err != nil {
		// Right length but not hex: not ours to touch, and guessing would be
		// exactly the silent repair this milestone exists to remove.
		return value, false, nil
	}

	recovered := base64.StdEncoding.EncodeToString(raw)
	if len(recovered) != correctLen {
		return value, false, fmt.Errorf("recovered %d chars, want %d", len(recovered), correctLen)
	}
	if _, err := hex.DecodeString(recovered); err != nil {
		// The recovered string must itself be a valid hex ID. If it is not,
		// this row was never corrupted the way we assume, so leave it.
		return value, false, nil
	}
	return recovered, true, nil
}
