package claudehistory

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create only the tables needed by claudehistory (avoids import cycle with observatory)
	// This is a minimal schema for testing - production uses observatory.MigrateWithVersion
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chat_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			turn_number INTEGER NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
			content_text TEXT,
			content_thinking TEXT,
			content_json TEXT,
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			model TEXT,
			request_id TEXT,
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, turn_number);
		CREATE INDEX IF NOT EXISTS idx_chat_messages_timestamp ON chat_messages(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_chat_messages_request ON chat_messages(request_id);

		CREATE TABLE IF NOT EXISTS chat_import_status (
			session_id TEXT PRIMARY KEY,
			file_path TEXT NOT NULL,
			file_mtime TIMESTAMP NOT NULL,
			message_count INTEGER NOT NULL,
			imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	return db
}

func setupTestData(t *testing.T, baseDir string) string {
	// Create a test project directory
	projectPath := filepath.Join(baseDir, "-test-project")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create a test JSONL file
	sessionID := "test-session-123"
	jsonlPath := filepath.Join(projectPath, sessionID+".jsonl")

	entries := []JSONLEntry{
		{
			SessionID: sessionID,
			Type:      "user",
			Timestamp: "2024-01-15T10:00:00Z",
			UUID:      "uuid-1",
			Message: &struct {
				Model   string `json:"model,omitempty"`
				ID      string `json:"id,omitempty"`
				Role    string `json:"role,omitempty"`
				Content []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				} `json:"content,omitempty"`
				StopReason string `json:"stop_reason,omitempty"`
				Usage      *struct {
					InputTokens          int `json:"input_tokens"`
					OutputTokens         int `json:"output_tokens"`
					CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
					CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
				} `json:"usage,omitempty"`
			}{
				Role: "user",
				Content: []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				}{
					{Type: "text", Text: "Hello, can you help me with Go programming?"},
				},
			},
		},
		{
			SessionID: sessionID,
			Type:      "assistant",
			Timestamp: "2024-01-15T10:00:05Z",
			UUID:      "uuid-2",
			RequestID: "req-123",
			Message: &struct {
				Model   string `json:"model,omitempty"`
				ID      string `json:"id,omitempty"`
				Role    string `json:"role,omitempty"`
				Content []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				} `json:"content,omitempty"`
				StopReason string `json:"stop_reason,omitempty"`
				Usage      *struct {
					InputTokens          int `json:"input_tokens"`
					OutputTokens         int `json:"output_tokens"`
					CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
					CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
				} `json:"usage,omitempty"`
			}{
				Model:      "claude-sonnet-4-5-20250514",
				ID:         "msg-456",
				Role:       "assistant",
				StopReason: "end_turn",
				Content: []struct {
					Type      string      `json:"type"`
					Text      string      `json:"text,omitempty"`
					Thinking  string      `json:"thinking,omitempty"`
					ID        string      `json:"id,omitempty"`
					Name      string      `json:"name,omitempty"`
					Input     interface{} `json:"input,omitempty"`
					ToolUseID string      `json:"tool_use_id,omitempty"`
					Content   string      `json:"content,omitempty"`
					IsError   bool        `json:"is_error,omitempty"`
				}{
					{Type: "thinking", Thinking: "Let me think about how to help with Go programming..."},
					{Type: "text", Text: "Of course! I'd be happy to help with Go programming."},
				},
				Usage: &struct {
					InputTokens          int `json:"input_tokens"`
					OutputTokens         int `json:"output_tokens"`
					CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
					CacheCreationTokens  int `json:"cache_creation_input_tokens,omitempty"`
				}{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
		},
	}

	file, err := os.Create(jsonlPath)
	if err != nil {
		t.Fatalf("failed to create JSONL file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			t.Fatalf("failed to encode entry: %v", err)
		}
	}

	return sessionID
}

func TestImporter_SyncSession(t *testing.T) {
	// Create temp directory for test data
	baseDir, err := os.MkdirTemp("", "claudehistory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	sessionID := setupTestData(t, baseDir)
	db := setupTestDB(t)
	defer db.Close()

	// Create importer with custom reader
	reader := NewReaderWithBase(baseDir)
	importer := NewImporterWithReader(db, reader)

	ctx := context.Background()

	// Import the session
	msgCount, err := importer.SyncSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("SyncSession failed: %v", err)
	}

	if msgCount != 2 {
		t.Errorf("expected 2 messages imported, got %d", msgCount)
	}

	// Verify messages in database
	messages, err := importer.GetChatMessages(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetChatMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages in DB, got %d", len(messages))
	}

	// Check first message (user)
	if messages[0].Role != "user" {
		t.Errorf("expected first message role 'user', got '%s'", messages[0].Role)
	}
	if messages[0].ContentText != "Hello, can you help me with Go programming?" {
		t.Errorf("unexpected content text: %s", messages[0].ContentText)
	}

	// Check second message (assistant)
	if messages[1].Role != "assistant" {
		t.Errorf("expected second message role 'assistant', got '%s'", messages[1].Role)
	}
	if messages[1].Model != "claude-sonnet-4-5-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-5-20250514', got '%s'", messages[1].Model)
	}
	if messages[1].ContentThinking != "Let me think about how to help with Go programming..." {
		t.Errorf("unexpected thinking: %s", messages[1].ContentThinking)
	}
	if messages[1].TokensIn != 100 || messages[1].TokensOut != 50 {
		t.Errorf("unexpected tokens: in=%d out=%d", messages[1].TokensIn, messages[1].TokensOut)
	}

	// Verify import status
	status, err := importer.GetImportStatus(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetImportStatus failed: %v", err)
	}
	if status == nil {
		t.Fatal("expected import status, got nil")
	}
	if status.MessageCount != 2 {
		t.Errorf("expected message count 2, got %d", status.MessageCount)
	}
}

func TestImporter_SyncAll(t *testing.T) {
	// Create temp directory for test data
	baseDir, err := os.MkdirTemp("", "claudehistory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	_ = setupTestData(t, baseDir)
	db := setupTestDB(t)
	defer db.Close()

	// Create importer with custom reader
	reader := NewReaderWithBase(baseDir)
	importer := NewImporterWithReader(db, reader)

	ctx := context.Background()

	// Sync all
	stats, err := importer.SyncAll(ctx)
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if stats.ProjectsScanned != 1 {
		t.Errorf("expected 1 project scanned, got %d", stats.ProjectsScanned)
	}
	if stats.SessionsScanned != 1 {
		t.Errorf("expected 1 session scanned, got %d", stats.SessionsScanned)
	}
	if stats.SessionsImported != 1 {
		t.Errorf("expected 1 session imported, got %d", stats.SessionsImported)
	}
	if stats.MessagesImported != 2 {
		t.Errorf("expected 2 messages imported, got %d", stats.MessagesImported)
	}
	if len(stats.Errors) > 0 {
		t.Errorf("unexpected errors: %v", stats.Errors)
	}
}

func TestImporter_SkipsUnmodified(t *testing.T) {
	// Create temp directory for test data
	baseDir, err := os.MkdirTemp("", "claudehistory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	_ = setupTestData(t, baseDir)
	db := setupTestDB(t)
	defer db.Close()

	// Create importer with custom reader
	reader := NewReaderWithBase(baseDir)
	importer := NewImporterWithReader(db, reader)

	ctx := context.Background()

	// First sync
	stats1, err := importer.SyncAll(ctx)
	if err != nil {
		t.Fatalf("first SyncAll failed: %v", err)
	}
	if stats1.SessionsImported != 1 {
		t.Errorf("first sync: expected 1 session imported, got %d", stats1.SessionsImported)
	}

	// Second sync (should skip)
	stats2, err := importer.SyncAll(ctx)
	if err != nil {
		t.Fatalf("second SyncAll failed: %v", err)
	}
	if stats2.SessionsImported != 0 {
		t.Errorf("second sync: expected 0 sessions imported, got %d", stats2.SessionsImported)
	}
	if stats2.SessionsSkipped != 1 {
		t.Errorf("second sync: expected 1 session skipped, got %d", stats2.SessionsSkipped)
	}
}

func TestImporter_ReimportsModified(t *testing.T) {
	// Create temp directory for test data
	baseDir, err := os.MkdirTemp("", "claudehistory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	sessionID := setupTestData(t, baseDir)
	db := setupTestDB(t)
	defer db.Close()

	// Create importer with custom reader
	reader := NewReaderWithBase(baseDir)
	importer := NewImporterWithReader(db, reader)

	ctx := context.Background()

	// First sync
	stats1, err := importer.SyncAll(ctx)
	if err != nil {
		t.Fatalf("first SyncAll failed: %v", err)
	}
	if stats1.SessionsImported != 1 {
		t.Errorf("first sync: expected 1 session imported, got %d", stats1.SessionsImported)
	}

	// Modify the file (touch it to update mtime)
	jsonlPath := filepath.Join(baseDir, "-test-project", sessionID+".jsonl")

	// Wait a moment so mtime changes
	time.Sleep(100 * time.Millisecond)

	// Touch the file
	now := time.Now()
	if err := os.Chtimes(jsonlPath, now, now); err != nil {
		t.Fatalf("failed to touch file: %v", err)
	}

	// Third sync (should reimport because file was modified)
	stats3, err := importer.SyncAll(ctx)
	if err != nil {
		t.Fatalf("third SyncAll failed: %v", err)
	}
	if stats3.SessionsImported != 1 {
		t.Errorf("third sync: expected 1 session imported after modification, got %d", stats3.SessionsImported)
	}
}

func TestImporter_GetChatMessagesByTimeRange(t *testing.T) {
	// Create temp directory for test data
	baseDir, err := os.MkdirTemp("", "claudehistory-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	sessionID := setupTestData(t, baseDir)
	db := setupTestDB(t)
	defer db.Close()

	// Create importer with custom reader
	reader := NewReaderWithBase(baseDir)
	importer := NewImporterWithReader(db, reader)

	ctx := context.Background()

	// Import the session first
	_, err = importer.SyncSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("SyncSession failed: %v", err)
	}

	// Query with time range that includes only the first message
	start := time.Date(2024, 1, 15, 9, 59, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 10, 0, 3, 0, time.UTC)

	messages, err := importer.GetChatMessagesByTimeRange(ctx, sessionID, start, end)
	if err != nil {
		t.Fatalf("GetChatMessagesByTimeRange failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("expected 1 message in time range, got %d", len(messages))
	}

	// Query with time range that includes both messages
	end = time.Date(2024, 1, 15, 10, 0, 10, 0, time.UTC)
	messages, err = importer.GetChatMessagesByTimeRange(ctx, sessionID, start, end)
	if err != nil {
		t.Fatalf("GetChatMessagesByTimeRange failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages in time range, got %d", len(messages))
	}
}
