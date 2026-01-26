package claudehistory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// Importer syncs Claude Code JSONL conversation history to observatory.db.
// It tracks which files have been imported and only imports new/modified files.
type Importer struct {
	reader *Reader
	db     *sql.DB
}

// NewImporter creates a new Importer with a database connection.
func NewImporter(db *sql.DB) *Importer {
	return &Importer{
		reader: NewReader(),
		db:     db,
	}
}

// NewImporterWithReader creates an Importer with a custom reader (for testing).
func NewImporterWithReader(db *sql.DB, reader *Reader) *Importer {
	return &Importer{
		reader: reader,
		db:     db,
	}
}

// SyncStats tracks import statistics.
type SyncStats struct {
	ProjectsScanned  int
	SessionsScanned  int
	SessionsImported int
	SessionsSkipped  int
	MessagesImported int
	Errors           []string
}

// SyncAll imports all new or modified JSONL files from ~/.claude/projects/.
func (i *Importer) SyncAll(ctx context.Context) (*SyncStats, error) {
	stats := &SyncStats{}

	// List all projects
	projects, err := i.reader.ListProjects()
	if err != nil {
		return stats, fmt.Errorf("listing projects: %w", err)
	}
	stats.ProjectsScanned = len(projects)

	// Process each project
	for _, project := range projects {
		sessions, err := i.reader.ListSessions(project.Path)
		if err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("listing sessions for %s: %v", project.Path, err))
			continue
		}

		for _, sessionMeta := range sessions {
			stats.SessionsScanned++

			// Check if we need to import this session
			needsImport, err := i.needsImport(ctx, sessionMeta)
			if err != nil {
				stats.Errors = append(stats.Errors, fmt.Sprintf("checking import status for %s: %v", sessionMeta.ID, err))
				continue
			}

			if !needsImport {
				stats.SessionsSkipped++
				continue
			}

			// Import the session
			msgCount, err := i.SyncSession(ctx, sessionMeta.ID)
			if err != nil {
				stats.Errors = append(stats.Errors, fmt.Sprintf("importing session %s: %v", sessionMeta.ID, err))
				continue
			}

			stats.SessionsImported++
			stats.MessagesImported += msgCount
		}
	}

	return stats, nil
}

// SyncSession imports a single session by ID.
// Returns the number of messages imported.
func (i *Importer) SyncSession(ctx context.Context, sessionID string) (int, error) {
	// Get the full session
	session, err := i.reader.GetSession(sessionID)
	if err != nil {
		return 0, fmt.Errorf("getting session: %w", err)
	}

	// Get file info for tracking
	filePath, err := i.reader.findSessionFile(sessionID)
	if err != nil {
		return 0, fmt.Errorf("finding session file: %w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("stat session file: %w", err)
	}

	// Start transaction
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing messages for this session (re-import)
	_, err = tx.ExecContext(ctx, "DELETE FROM chat_messages WHERE session_id = ?", session.ID)
	if err != nil {
		return 0, fmt.Errorf("deleting existing messages: %w", err)
	}

	// Insert messages
	turnNumber := 0
	for _, msg := range session.Messages {
		// Determine turn number (user message starts a new turn)
		if msg.Type == "user" {
			turnNumber++
		}

		// Extract text content
		var contentText, contentThinking string
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				if contentText != "" {
					contentText += "\n\n"
				}
				contentText += block.Text
			case "thinking":
				if contentThinking != "" {
					contentThinking += "\n\n"
				}
				contentThinking += block.Thinking
			}
		}

		// Serialize full content as JSON
		contentJSON, _ := json.Marshal(msg.Content)

		// Generate ID if not present
		msgID := msg.UUID
		if msgID == "" {
			msgID = uuid.New().String()
		}

		// Get token counts
		var tokensIn, tokensOut int
		if msg.Usage != nil {
			tokensIn = msg.Usage.InputTokens
			tokensOut = msg.Usage.OutputTokens
		}

		// Map Type to role
		role := msg.Type
		if role != "user" && role != "assistant" {
			role = "assistant" // Default for unknown types
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO chat_messages (
				id, session_id, turn_number, role, content_text, content_thinking,
				content_json, tokens_in, tokens_out, model, request_id, timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			msgID, session.ID, turnNumber, role, contentText, contentThinking,
			string(contentJSON), tokensIn, tokensOut, msg.Model, msg.RequestID, msg.Timestamp,
		)
		if err != nil {
			return 0, fmt.Errorf("inserting message: %w", err)
		}
	}

	// Update import status
	_, err = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO chat_import_status (
			session_id, file_path, file_mtime, message_count, imported_at
		) VALUES (?, ?, ?, ?, ?)
	`,
		session.ID, filePath, fileInfo.ModTime(), len(session.Messages), time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("updating import status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return len(session.Messages), nil
}

// needsImport checks if a session needs to be imported.
// Returns true if the session hasn't been imported or the file has been modified.
func (i *Importer) needsImport(ctx context.Context, meta SessionMeta) (bool, error) {
	// Get file mtime
	fileInfo, err := os.Stat(meta.FilePath)
	if err != nil {
		return false, fmt.Errorf("stat file: %w", err)
	}
	fileMtime := fileInfo.ModTime()

	// Check import status
	var importedMtime time.Time
	err = i.db.QueryRowContext(ctx, `
		SELECT file_mtime FROM chat_import_status WHERE session_id = ?
	`, meta.ID).Scan(&importedMtime)

	if err == sql.ErrNoRows {
		// Never imported
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("query import status: %w", err)
	}

	// Check if file has been modified since last import
	return fileMtime.After(importedMtime), nil
}

// GetImportStatus returns the import status for a session.
func (i *Importer) GetImportStatus(ctx context.Context, sessionID string) (*ImportStatus, error) {
	var status ImportStatus
	err := i.db.QueryRowContext(ctx, `
		SELECT session_id, file_path, file_mtime, message_count, imported_at
		FROM chat_import_status WHERE session_id = ?
	`, sessionID).Scan(&status.SessionID, &status.FilePath, &status.FileMtime, &status.MessageCount, &status.ImportedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query import status: %w", err)
	}

	return &status, nil
}

// ImportStatus represents the import status of a session.
type ImportStatus struct {
	SessionID    string    `json:"session_id"`
	FilePath     string    `json:"file_path"`
	FileMtime    time.Time `json:"file_mtime"`
	MessageCount int       `json:"message_count"`
	ImportedAt   time.Time `json:"imported_at"`
}

// GetAllImportStatus returns import status for all imported sessions.
func (i *Importer) GetAllImportStatus(ctx context.Context) ([]*ImportStatus, error) {
	rows, err := i.db.QueryContext(ctx, `
		SELECT session_id, file_path, file_mtime, message_count, imported_at
		FROM chat_import_status ORDER BY imported_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query import status: %w", err)
	}
	defer rows.Close()

	var statuses []*ImportStatus
	for rows.Next() {
		var status ImportStatus
		if err := rows.Scan(&status.SessionID, &status.FilePath, &status.FileMtime, &status.MessageCount, &status.ImportedAt); err != nil {
			return nil, fmt.Errorf("scan import status: %w", err)
		}
		statuses = append(statuses, &status)
	}

	return statuses, rows.Err()
}

// GetChatMessages retrieves chat messages for a session from the database.
func (i *Importer) GetChatMessages(ctx context.Context, sessionID string) ([]*ChatMessage, error) {
	rows, err := i.db.QueryContext(ctx, `
		SELECT id, session_id, turn_number, role, content_text, content_thinking,
		       content_json, tokens_in, tokens_out, model, request_id, timestamp, created_at
		FROM chat_messages WHERE session_id = ? ORDER BY turn_number, timestamp
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var messages []*ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var contentText, contentThinking, contentJSON, model, requestID sql.NullString
		if err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.TurnNumber, &msg.Role,
			&contentText, &contentThinking, &contentJSON,
			&msg.TokensIn, &msg.TokensOut, &model, &requestID,
			&msg.Timestamp, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		msg.ContentText = contentText.String
		msg.ContentThinking = contentThinking.String
		msg.ContentJSON = contentJSON.String
		msg.Model = model.String
		msg.RequestID = requestID.String
		messages = append(messages, &msg)
	}

	return messages, rows.Err()
}

// ChatMessage represents a chat message stored in the database.
type ChatMessage struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id"`
	TurnNumber      int       `json:"turn_number"`
	Role            string    `json:"role"`
	ContentText     string    `json:"content_text,omitempty"`
	ContentThinking string    `json:"content_thinking,omitempty"`
	ContentJSON     string    `json:"content_json,omitempty"`
	TokensIn        int       `json:"tokens_in"`
	TokensOut       int       `json:"tokens_out"`
	Model           string    `json:"model,omitempty"`
	RequestID       string    `json:"request_id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetChatMessagesByTimeRange retrieves chat messages within a time range.
// Useful for correlating spans to chat context.
func (i *Importer) GetChatMessagesByTimeRange(ctx context.Context, sessionID string, start, end time.Time) ([]*ChatMessage, error) {
	rows, err := i.db.QueryContext(ctx, `
		SELECT id, session_id, turn_number, role, content_text, content_thinking,
		       content_json, tokens_in, tokens_out, model, request_id, timestamp, created_at
		FROM chat_messages
		WHERE session_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY turn_number, timestamp
	`, sessionID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var messages []*ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var contentText, contentThinking, contentJSON, model, requestID sql.NullString
		if err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.TurnNumber, &msg.Role,
			&contentText, &contentThinking, &contentJSON,
			&msg.TokensIn, &msg.TokensOut, &model, &requestID,
			&msg.Timestamp, &msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		msg.ContentText = contentText.String
		msg.ContentThinking = contentThinking.String
		msg.ContentJSON = contentJSON.String
		msg.Model = model.String
		msg.RequestID = requestID.String
		messages = append(messages, &msg)
	}

	return messages, rows.Err()
}
