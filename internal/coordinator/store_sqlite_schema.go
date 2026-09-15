package coordinator

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/simhash"
)

// Versioned migrations for coordinator.db.
//
// The additive schema (CREATE TABLE IF NOT EXISTS + ignored ALTER TABLE ADD
// COLUMN) in migrate() is idempotent by construction and needs no version.
// A DATA migration — one that rewrites rows — must run exactly once, so it is
// keyed on SQLite's `PRAGMA user_version`: 0 on every database created before
// this file existed, then the number of the last data migration applied.
//
// Add a migration by appending to dataMigrations; never reorder or edit an
// entry that has shipped — its number is stamped in every rig's database.

type dataMigration struct {
	version int
	name    string
	apply   func(s *SQLiteStore, now time.Time) error
}

var dataMigrations = []dataMigration{
	{1, "reindex tasks.fingerprint into the simhash leaf's hash space", (*SQLiteStore).reindexFingerprints},
}

// migrateData runs every data migration newer than the database's user_version.
func (s *SQLiteStore) migrateData(now time.Time) error {
	var have int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&have); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	for _, m := range dataMigrations {
		if m.version <= have {
			continue
		}
		if err := m.apply(s, now); err != nil {
			return fmt.Errorf("data migration %d (%s): %w", m.version, m.name, err)
		}
		// PRAGMA does not take a bound parameter.
		if _, err := s.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, m.version)); err != nil {
			return fmt.Errorf("stamp schema version %d: %w", m.version, err)
		}
		have = m.version
	}
	// A version newer than this binary knows is not an error: data migrations
	// here rewrite values in place and never change what a column means, so
	// an older binary (a stale install sharing ~/.ailang/coordinator.db with
	// a fresh build) still reads the store correctly.
	return nil
}

// reindexFingerprints (v1, M-V1-SIMPLIFY-S4 M3B) recomputes tasks.fingerprint
// for every row still inside DedupWindow from the row's own content.
//
// Until M-V1-SIMPLIFY-S3 M5 (5731a1f38) the coordinator fingerprinted with a
// private SimHash variant — ASCII-only tokens, words shorter than two bytes
// dropped. The survivor, simhash.Hash, tokenises Unicode letters and digits
// and keeps one-character words, so the two disagree for any content holding
// a non-ASCII rune or a one-character token ("a", "I", "1") and agree
// otherwise. FindDuplicateTask matches by EXACT equality, so a row written by
// the old variant could not suppress a duplicate hashed by the new one.
//
// Only rows newer than the window matter: BlocksDuplicate ignores anything
// older, and a row's content is stored (tasks.content NOT NULL), so the
// fingerprint is recomputed rather than cleared. Rows whose stored value
// already equals the recomputed one — every ASCII-only, multi-character-word
// row — are left untouched.
func (s *SQLiteStore) reindexFingerprints(now time.Time) error {
	rows, err := s.db.Query(
		`SELECT id, content, fingerprint FROM tasks
		 WHERE fingerprint IS NOT NULL AND fingerprint != 0 AND created_at >= ?`,
		DedupSince(now),
	)
	if err != nil {
		return err
	}
	type fix struct {
		id string
		fp int64
	}
	var fixes []fix
	for rows.Next() {
		var id, content string
		var stored sql.NullInt64
		if err := rows.Scan(&id, &content, &stored); err != nil {
			_ = rows.Close()
			return err
		}
		if want := simhash.Hash(content); stored.Int64 != want {
			fixes = append(fixes, fix{id, want})
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, f := range fixes {
		// int64 on both sides — see SetTaskFingerprint.
		if _, err := s.db.Exec(`UPDATE tasks SET fingerprint = ? WHERE id = ?`, f.fp, f.id); err != nil {
			return err
		}
	}
	return nil
}
