package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/simhash"
)

// retiredASCIISimhash is the coordinator's pre-S3 fingerprint, copied from
// analyzer.go as deleted by 5731a1f38: ASCII-only tokens, words shorter than
// two bytes dropped, FNV-1a 64. It exists here ONLY so the migration test can
// write rows in the hash space a real pre-migration coordinator.db holds.
func retiredASCIISimhash(content string) int64 {
	words := strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	var v [64]int
	for _, word := range words {
		if len(word) < 2 {
			continue
		}
		h := uint64(14695981039346656037)
		for i := 0; i < len(word); i++ {
			h ^= uint64(word[i])
			h *= 1099511628211
		}
		for i := uint(0); i < 64; i++ {
			if (h>>i)&1 == 1 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}
	var fp uint64
	for i := uint(0); i < 64; i++ {
		if v[i] > 0 {
			fp |= 1 << i
		}
	}
	return int64(fp)
}

func TestReindexFingerprints_RecomputesOnlyDivergentRowsInsideWindow(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()
	now := time.Now()

	const (
		ascii    = "fix the parser panic on an empty match arm"
		nonASCII = "Zürich café — naïve résumé 日本語 テスト parser panic"
		oneChar  = "I have a 1 x parser panic" // ASCII, but the retired variant dropped 1-char words
	)
	// Instrument check: the two hash spaces must actually differ where the
	// test claims they do, and agree where it claims they agree.
	if retiredASCIISimhash(ascii) != simhash.Hash(ascii) {
		t.Fatalf("instrument: ASCII multi-char content must hash identically in both spaces")
	}
	if retiredASCIISimhash(nonASCII) == simhash.Hash(nonASCII) {
		t.Fatalf("instrument: non-ASCII content must hash differently in the two spaces")
	}
	if retiredASCIISimhash(oneChar) == simhash.Hash(oneChar) {
		t.Fatalf("instrument: one-char-token content must hash differently in the two spaces")
	}

	rows := []struct {
		id      string
		content string
		created time.Time
	}{
		{"ascii-recent", ascii, now.Add(-time.Hour)},
		{"nonascii-recent", nonASCII, now.Add(-time.Hour)},
		{"onechar-recent", oneChar, now.Add(-time.Hour)},
		{"nonascii-old", nonASCII, now.Add(-DedupWindow - time.Hour)},
	}
	for _, r := range rows {
		task := &TaskRecord{ID: r.id, Title: "t", Content: r.content, Type: TaskTypeBugFix, Status: TaskStatusPending, CreatedAt: r.created}
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatal(err)
		}
		if err := store.SetTaskFingerprint(ctx, r.id, uint64(retiredASCIISimhash(r.content))); err != nil {
			t.Fatal(err)
		}
	}
	// Bring the store back to "never migrated": createTestStore already ran
	// migrateData on an empty table and stamped user_version.
	if _, err := store.db.Exec(`PRAGMA user_version = 0`); err != nil {
		t.Fatal(err)
	}

	if err := store.migrateData(now); err != nil {
		t.Fatalf("migrateData: %v", err)
	}

	got := func(id string) int64 {
		var fp int64
		if err := store.db.QueryRow(`SELECT fingerprint FROM tasks WHERE id = ?`, id).Scan(&fp); err != nil {
			t.Fatal(err)
		}
		return fp
	}
	if fp := got("ascii-recent"); fp != retiredASCIISimhash(ascii) {
		t.Errorf("ASCII fingerprint changed by the migration: %d", fp)
	}
	if fp := got("nonascii-recent"); fp != simhash.Hash(nonASCII) {
		t.Errorf("non-ASCII fingerprint not recomputed: %d, want %d", fp, simhash.Hash(nonASCII))
	}
	if fp := got("onechar-recent"); fp != simhash.Hash(oneChar) {
		t.Errorf("one-char-token fingerprint not recomputed: %d, want %d", fp, simhash.Hash(oneChar))
	}
	if fp := got("nonascii-old"); fp != retiredASCIISimhash(nonASCII) {
		t.Errorf("row outside DedupWindow was rewritten: %d", fp)
	}

	// The recomputed row now suppresses a duplicate hashed by the survivor.
	dup, err := store.FindDuplicateTask(ctx, uint64(simhash.Hash(nonASCII)), DedupScope{Since: DedupSince(now)})
	if err != nil {
		t.Fatal(err)
	}
	if dup == nil || dup.ID != "nonascii-recent" {
		t.Errorf("recomputed fingerprint does not match the survivor's hash: %+v", dup)
	}

	var version int
	if err := store.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != dataMigrations[len(dataMigrations)-1].version {
		t.Errorf("user_version = %d after migration", version)
	}
}

func TestMigrateData_RunsOnceAndSurvivesReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "coordinator.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const content = "naïve résumé"
	task := &TaskRecord{ID: "t1", Title: "t", Content: content, Type: TaskTypeBugFix, Status: TaskStatusPending, CreatedAt: time.Now()}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	// A row written AFTER the migration ran (the store is already at the
	// current version) in the retired space must stay as written: the
	// migration is not re-run on reopen.
	if err := store.SetTaskFingerprint(ctx, "t1", uint64(retiredASCIISimhash(content))); err != nil {
		t.Fatal(err)
	}
	store.Close()

	store, err = NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var fp int64
	if err := store.db.QueryRow(`SELECT fingerprint FROM tasks WHERE id = 't1'`).Scan(&fp); err != nil {
		t.Fatal(err)
	}
	if fp != retiredASCIISimhash(content) {
		t.Errorf("migration re-ran on reopen: fingerprint = %d", fp)
	}
}
