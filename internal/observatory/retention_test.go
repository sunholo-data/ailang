package observatory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func openTestDB(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_journal_mode=WAL")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if _, err := MigrateWithVersion(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return NewStore(db)
}

func insertTestSpan(t *testing.T, store *Store, id string, startTime time.Time) {
	t.Helper()
	_, err := store.db.Exec(`INSERT INTO spans (id, trace_id, name, kind, status, start_time, end_time, created_at)
		VALUES (?, ?, 'test-span', 'internal', 'ok', ?, ?, datetime('now'))`,
		id, "trace-"+id, startTime.UnixNano(), startTime.Add(time.Second).UnixNano())
	if err != nil {
		t.Fatalf("failed to insert test span %s: %v", id, err)
	}
}

func TestRunRetention_DeletesOldSpans(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	insertTestSpan(t, store, "old-span", now.Add(-8*24*time.Hour))
	insertTestSpan(t, store, "new-span", now.Add(-1*24*time.Hour))

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.SpansDeleted != 1 {
		t.Errorf("expected 1 span deleted, got %d", stats.SpansDeleted)
	}

	var count int
	store.db.QueryRow("SELECT COUNT(*) FROM spans").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 remaining span, got %d", count)
	}
}

func TestRunRetention_PreservesRecentData(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 3; i++ {
		ts := now.Add(-time.Duration(i) * 24 * time.Hour)
		insertTestSpan(t, store, "recent-"+string(rune('a'+i)), ts)
	}

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.SpansDeleted != 0 {
		t.Errorf("expected 0 spans deleted, got %d", stats.SpansDeleted)
	}

	var count int
	store.db.QueryRow("SELECT COUNT(*) FROM spans").Scan(&count)
	if count != 3 {
		t.Errorf("expected 3 remaining spans, got %d", count)
	}
}

func TestRunRetention_ChatMessages30DayTTL(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	oldTS := now.Add(-31 * 24 * time.Hour).Unix()
	newTS := now.Add(-5 * 24 * time.Hour).Unix()
	_, err := store.db.Exec(`INSERT INTO chat_messages (id, session_id, turn_number, role, content_text, timestamp, created_at) VALUES ('m1', 's1', 1, 'user', 'old msg', ?, ?)`,
		oldTS, oldTS)
	if err != nil {
		t.Fatalf("failed to insert old chat: %v", err)
	}
	store.db.Exec(`INSERT INTO chat_messages (id, session_id, turn_number, role, content_text, timestamp, created_at) VALUES ('m2', 's2', 1, 'user', 'new msg', ?, ?)`,
		newTS, newTS)

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.ChatDeleted != 1 {
		t.Errorf("expected 1 chat message deleted, got %d", stats.ChatDeleted)
	}
}

func TestRetentionStats_Total(t *testing.T) {
	stats := RetentionStats{
		SpansDeleted:     100,
		SummariesDeleted: 20,
		MetricsDeleted:   30,
		ChatDeleted:      5,
		ToolsDeleted:     10,
	}
	if stats.Total() != 165 {
		t.Errorf("expected total 165, got %d", stats.Total())
	}
}

func TestRetentionStats_String(t *testing.T) {
	stats := RetentionStats{SpansDeleted: 50}
	s := stats.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}
