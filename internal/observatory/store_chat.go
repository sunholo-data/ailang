// Package observatory provides chat message query methods for the Store.
package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ChatMessage represents a stored chat message from a Claude/Gemini session.
type ChatMessage struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	TurnNumber  int       `json:"turn_number"`
	Role        string    `json:"role"`
	ContentJSON string    `json:"content_json,omitempty"` // Raw JSON content blocks
	TokensIn    int       `json:"tokens_in,omitempty"`
	TokensOut   int       `json:"tokens_out,omitempty"`
	Model       string    `json:"model,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	TaskID      string    `json:"task_id,omitempty"`
	ChainID     string    `json:"chain_id,omitempty"`
}

// ChatMessageQuery options for filtering chat messages.
type ChatMessageQuery struct {
	TaskID    string
	SessionID string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// GetChatMessagesByTaskID fetches chat messages linked to a task via task_id.
// This is the preferred deterministic query (M-DETERMINISTIC-CHAT-LINKING).
func (s *Store) GetChatMessagesByTaskID(ctx context.Context, taskID string) ([]*ChatMessage, error) {
	if taskID == "" {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, turn_number, role, content_json,
		       tokens_in, tokens_out, model, timestamp,
		       COALESCE(task_id, ''), COALESCE(chain_id, '')
		FROM chat_messages
		WHERE task_id = ?
		ORDER BY turn_number, timestamp
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat messages by task_id: %w", err)
	}
	defer rows.Close()

	return scanChatMessages(rows)
}

// GetChatMessagesBySession fetches chat messages for a session, optionally filtered by time range.
// Pass zero-value times to skip time filtering.
func (s *Store) GetChatMessagesBySession(ctx context.Context, sessionID string, startTime, endTime time.Time) ([]*ChatMessage, error) {
	if sessionID == "" {
		return nil, nil
	}

	query := `
		SELECT id, session_id, turn_number, role, content_json,
		       tokens_in, tokens_out, model, timestamp,
		       COALESCE(task_id, ''), COALESCE(chain_id, '')
		FROM chat_messages
		WHERE session_id = ?
	`
	args := []interface{}{sessionID}

	if !startTime.IsZero() {
		// Buffer 1 min before to catch setup messages
		bufferedStart := startTime.Add(-1 * time.Minute)
		query += " AND timestamp >= ?"
		args = append(args, bufferedStart)
	}
	if !endTime.IsZero() {
		// Buffer 1 min after to catch completion messages
		bufferedEnd := endTime.Add(1 * time.Minute)
		query += " AND timestamp <= ?"
		args = append(args, bufferedEnd)
	}

	query += " ORDER BY turn_number, timestamp"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat messages by session: %w", err)
	}
	defer rows.Close()

	return scanChatMessages(rows)
}

// CountChatMessages returns total and linked message counts for health diagnostics.
func (s *Store) CountChatMessages(ctx context.Context, q ChatMessageQuery) (total int, withTaskID int, err error) {
	query := `SELECT COUNT(*), COUNT(CASE WHEN task_id IS NOT NULL AND task_id <> '' THEN 1 END) FROM chat_messages WHERE 1=1`
	args := []interface{}{}

	if q.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, q.SessionID)
	}
	if !q.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, q.StartTime)
	}
	if !q.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, q.EndTime)
	}

	err = s.db.QueryRowContext(ctx, query, args...).Scan(&total, &withTaskID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count chat messages: %w", err)
	}
	return total, withTaskID, nil
}

// scanChatMessages scans rows into ChatMessage structs.
func scanChatMessages(rows *sql.Rows) ([]*ChatMessage, error) {
	var messages []*ChatMessage
	for rows.Next() {
		var msg ChatMessage
		var contentJSON, model sql.NullString

		if err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.TurnNumber, &msg.Role,
			&contentJSON, &msg.TokensIn, &msg.TokensOut, &model,
			&msg.Timestamp, &msg.TaskID, &msg.ChainID,
		); err != nil {
			continue
		}

		msg.Model = model.String
		if contentJSON.Valid {
			msg.ContentJSON = contentJSON.String
		}

		messages = append(messages, &msg)
	}
	return messages, rows.Err()
}
