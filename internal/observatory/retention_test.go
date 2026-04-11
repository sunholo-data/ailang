package observatory

import (
	"context"
	"database/sql"
	"fmt"
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

// insertTestSpan matches how the production writer stores timestamps: a Go
// time.Time value bound via the go-sqlite3 driver, which serializes to a
// TEXT ISO-8601 string in the spans.start_time column. The previous version
// of this helper passed startTime.UnixNano() directly, which stored numbers
// and hid a retention bug where the production DELETE compared a TEXT column
// against an integer cutoff (matches zero rows per SQLite storage-class
// ordering).
func insertTestSpan(t *testing.T, store *Store, id string, startTime time.Time) {
	t.Helper()
	_, err := store.db.Exec(`INSERT INTO spans (id, trace_id, name, kind, status, start_time, end_time, created_at)
		VALUES (?, ?, 'test-span', 'internal', 'ok', ?, ?, datetime('now'))`,
		id, "trace-"+id, startTime, startTime.Add(time.Second))
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

// TestRunRetention_HandlesTextTimestampsWithTZ is the regression test for the
// TEXT-vs-INTEGER comparison bug. It inserts a span with the exact production
// storage format (ISO-8601 with timezone offset) and asserts retention deletes
// it. Before the fix, the DELETE query compared against a UnixNano cutoff and
// silently matched zero rows.
func TestRunRetention_HandlesTextTimestampsWithTZ(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()

	// Insert old and new spans as literal TEXT, including a TZ offset — this
	// is exactly what the go-sqlite3 driver produces for time.Time values and
	// what we saw in the 1.6GB production DB.
	oldTS := "2026-01-01 10:00:00.000000+02:00"
	newTS := time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05.000000-07:00")

	_, err := store.db.Exec(`INSERT INTO spans (id, trace_id, name, kind, status, start_time, end_time, created_at)
		VALUES ('old', 'trace-old', 'test', 'internal', 'ok', ?, ?, datetime('now'))`, oldTS, oldTS)
	if err != nil {
		t.Fatalf("failed to insert old span: %v", err)
	}
	_, err = store.db.Exec(`INSERT INTO spans (id, trace_id, name, kind, status, start_time, end_time, created_at)
		VALUES ('new', 'trace-new', 'test', 'internal', 'ok', ?, ?, datetime('now'))`, newTS, newTS)
	if err != nil {
		t.Fatalf("failed to insert new span: %v", err)
	}

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.SpansDeleted != 1 {
		t.Errorf("expected 1 span deleted (TEXT timestamp with TZ), got %d — retention is probably still comparing TEXT to INT", stats.SpansDeleted)
	}

	var remaining string
	if err := store.db.QueryRow("SELECT id FROM spans").Scan(&remaining); err != nil {
		t.Fatalf("failed to query remaining span: %v", err)
	}
	if remaining != "new" {
		t.Errorf("expected 'new' span to remain, got %q", remaining)
	}
}

func TestRunRetention_ChatMessages30DayTTL(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// chat_messages.created_at is a TEXT TIMESTAMP column (default
	// CURRENT_TIMESTAMP). Production inserts bind time.Time values which the
	// driver stores as TEXT, so the test fixtures must do the same.
	oldTS := now.Add(-31 * 24 * time.Hour)
	newTS := now.Add(-5 * 24 * time.Hour)
	_, err := store.db.Exec(`INSERT INTO chat_messages (id, session_id, turn_number, role, content_text, timestamp, created_at) VALUES ('m1', 's1', 1, 'user', 'old msg', ?, ?)`,
		oldTS, oldTS)
	if err != nil {
		t.Fatalf("failed to insert old chat: %v", err)
	}
	_, err = store.db.Exec(`INSERT INTO chat_messages (id, session_id, turn_number, role, content_text, timestamp, created_at) VALUES ('m2', 's2', 1, 'user', 'new msg', ?, ?)`,
		newTS, newTS)
	if err != nil {
		t.Fatalf("failed to insert new chat: %v", err)
	}

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.ChatDeleted != 1 {
		t.Errorf("expected 1 chat message deleted, got %d", stats.ChatDeleted)
	}
}

// TestRunRetention_SessionTools verifies retention for session_tools, which
// has no created_at column (the previous code tried to delete by created_at
// and errored silently).
func TestRunRetention_SessionTools(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// Need a parent session row so the FK holds.
	_, err := store.db.Exec(`INSERT INTO sessions (session_id, workspace) VALUES ('s1', '/tmp')`)
	if err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	oldTS := now.Add(-31 * 24 * time.Hour)
	newTS := now.Add(-1 * 24 * time.Hour)
	_, err = store.db.Exec(`INSERT INTO session_tools (tool_use_id, session_id, tool_name, start_time) VALUES ('t-old', 's1', 'Read', ?)`, oldTS)
	if err != nil {
		t.Fatalf("failed to insert old tool: %v", err)
	}
	_, err = store.db.Exec(`INSERT INTO session_tools (tool_use_id, session_id, tool_name, start_time) VALUES ('t-new', 's1', 'Read', ?)`, newTS)
	if err != nil {
		t.Fatalf("failed to insert new tool: %v", err)
	}

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.ToolsDeleted != 1 {
		t.Errorf("expected 1 session_tool deleted, got %d", stats.ToolsDeleted)
	}

	var remaining string
	if err := store.db.QueryRow("SELECT tool_use_id FROM session_tools").Scan(&remaining); err != nil {
		t.Fatalf("failed to query remaining tool: %v", err)
	}
	if remaining != "t-new" {
		t.Errorf("expected 't-new' tool to remain, got %q", remaining)
	}
}

// TestRunRetention_ChunkedDelete inserts more rows than the chunk size so the
// retention loop has to iterate. This is the guardrail against regressions
// where someone removes the chunking — a single monolithic DELETE on 100k+
// rows bloats the WAL past the >2GB threshold when a live reader holds back
// the checkpoint, which in turn historically tripped a destructive auto-delete
// branch in health.go.
func TestRunRetention_ChunkedDelete(t *testing.T) {
	store := openTestDB(t)
	ctx := context.Background()
	old := time.Now().Add(-8 * 24 * time.Hour)

	// Insert 2 * chunk + 17 old rows to force three iterations with a
	// non-round remainder on the last chunk.
	const n = retentionChunkSize*2 + 17
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO spans (id, trace_id, name, kind, status, start_time, end_time, created_at)
		VALUES (?, ?, 'test-span', 'internal', 'ok', ?, ?, datetime('now'))`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("old-%d", i)
		if _, err := stmt.Exec(id, "trace-"+id, old, old.Add(time.Second)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	stats, err := store.RunRetention(ctx)
	if err != nil {
		t.Fatalf("RunRetention failed: %v", err)
	}

	if stats.SpansDeleted != n {
		t.Errorf("expected %d spans deleted, got %d — chunked loop may be exiting early", n, stats.SpansDeleted)
	}

	var remaining int
	store.db.QueryRow("SELECT COUNT(*) FROM spans").Scan(&remaining)
	if remaining != 0 {
		t.Errorf("expected 0 remaining spans, got %d", remaining)
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
