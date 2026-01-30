package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sunholo/ailang/internal/observatory"
)

// taskEvent represents an event from coordinator task_events table (basic)
type taskEvent struct {
	StreamType string
	TurnNum    int
	Text       string
	ToolName   string
	ToolInput  string
}

// taskSessionInfo contains session_id and time range for filtering
type taskSessionInfo struct {
	SessionID   string
	StartedAt   time.Time
	CompletedAt time.Time
}

// chatMessageExport represents a chat message with full tool data
type chatMessageExport struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"session_id"`
	TurnNumber int            `json:"turn_number"`
	Role       string         `json:"role"`
	Model      string         `json:"model,omitempty"`
	TokensIn   int            `json:"tokens_in,omitempty"`
	TokensOut  int            `json:"tokens_out,omitempty"`
	Timestamp  string         `json:"timestamp"`
	Content    []contentBlock `json:"content"` // Parsed content blocks with tools
}

// contentBlock represents a content block from Claude Code
type contentBlock struct {
	Type string `json:"type"` // "text", "thinking", "tool_use", "tool_result"

	// For type="text"
	Text string `json:"text,omitempty"`

	// For type="thinking"
	Thinking string `json:"thinking,omitempty"`

	// For type="tool_use"
	ToolUse *toolUseBlock `json:"tool_use,omitempty"`

	// For type="tool_result"
	ToolResult *toolResultBlock `json:"tool_result,omitempty"`
}

type toolUseBlock struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"` // Full tool input (varies by tool)
}

type toolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"` // Full tool output
	IsError   bool   `json:"is_error"`
}

// getTaskEvents queries coordinator.db for basic task events (for tree display)
func getTaskEvents(taskID string) []taskEvent {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dbPath := filepath.Join(homeDir, ".ailang", "state", "coordinator.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT stream_type, turn_num, COALESCE(text, ''), COALESCE(tool_name, ''), COALESCE(tool_input, '')
		FROM task_events
		WHERE task_id = ?
		ORDER BY id ASC
	`, taskID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var events []taskEvent
	for rows.Next() {
		var e taskEvent
		if err := rows.Scan(&e.StreamType, &e.TurnNum, &e.Text, &e.ToolName, &e.ToolInput); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events
}

// getSessionInfoFromTask looks up session_id and time range from coordinator.db tasks table
func getSessionInfoFromTask(taskID string) *taskSessionInfo {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dbPath := filepath.Join(homeDir, ".ailang", "state", "coordinator.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	var sessionID sql.NullString
	var startedAt, completedAt sql.NullTime
	err = db.QueryRow(`
		SELECT session_id, started_at, completed_at
		FROM tasks WHERE id = ?
	`, taskID).Scan(&sessionID, &startedAt, &completedAt)
	if err != nil || !sessionID.Valid {
		return nil
	}

	info := &taskSessionInfo{SessionID: sessionID.String}
	if startedAt.Valid {
		info.StartedAt = startedAt.Time
	}
	if completedAt.Valid {
		info.CompletedAt = completedAt.Time
	}
	return info
}

// getSessionIDFromTask looks up session_id from coordinator.db tasks table (legacy helper)
func getSessionIDFromTask(taskID string) string {
	info := getSessionInfoFromTask(taskID)
	if info == nil {
		return ""
	}
	return info.SessionID
}

// getChatMessages fetches chat messages from observatory.db with full content
func getChatMessages(sessionID string) []chatMessageExport {
	return getChatMessagesInRange(sessionID, time.Time{}, time.Time{})
}

// getChatMessagesForTask fetches chat messages by task_id directly (M-DETERMINISTIC-CHAT-LINKING)
// This is the preferred method when task_id is known - no timestamp filtering needed
func getChatMessagesForTask(taskID string) []chatMessageExport {
	if taskID == "" {
		return nil
	}

	dbPath := observatory.DefaultDatabasePath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	query := `
		SELECT id, session_id, turn_number, role, content_json,
		       tokens_in, tokens_out, model, timestamp
		FROM chat_messages
		WHERE task_id = ?
		ORDER BY turn_number, timestamp
	`

	rows, err := db.Query(query, taskID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []chatMessageExport
	for rows.Next() {
		var msg chatMessageExport
		var contentJSON, model sql.NullString
		var timestamp time.Time

		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.TurnNumber, &msg.Role,
			&contentJSON, &msg.TokensIn, &msg.TokensOut, &model, &timestamp); err != nil {
			continue
		}

		msg.Model = model.String
		msg.Timestamp = timestamp.Format(time.RFC3339)

		// Parse content_json to get full tool data
		if contentJSON.Valid && contentJSON.String != "" {
			msg.Content = parseContentJSON(contentJSON.String)
		}

		messages = append(messages, msg)
	}

	return messages
}

// getChatMessagesInRange fetches chat messages within a time range
func getChatMessagesInRange(sessionID string, startTime, endTime time.Time) []chatMessageExport {
	dbPath := observatory.DefaultDatabasePath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	// Build query with optional time filtering
	query := `
		SELECT id, session_id, turn_number, role, content_json,
		       tokens_in, tokens_out, model, timestamp
		FROM chat_messages
		WHERE session_id = ?
	`
	args := []interface{}{sessionID}

	// Add time range filter if provided
	if !startTime.IsZero() {
		// Add a buffer before start time (1 min) to catch setup messages
		bufferedStart := startTime.Add(-1 * time.Minute)
		query += " AND timestamp >= ?"
		args = append(args, bufferedStart)
	}
	if !endTime.IsZero() {
		// Add a buffer after end time (1 min) to catch completion messages
		bufferedEnd := endTime.Add(1 * time.Minute)
		query += " AND timestamp <= ?"
		args = append(args, bufferedEnd)
	}

	query += " ORDER BY turn_number, timestamp"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []chatMessageExport
	for rows.Next() {
		var msg chatMessageExport
		var contentJSON, model sql.NullString
		var timestamp time.Time

		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.TurnNumber, &msg.Role,
			&contentJSON, &msg.TokensIn, &msg.TokensOut, &model, &timestamp); err != nil {
			continue
		}

		msg.Model = model.String
		msg.Timestamp = timestamp.Format(time.RFC3339)

		// Parse content_json to get full tool data
		if contentJSON.Valid && contentJSON.String != "" {
			msg.Content = parseContentJSON(contentJSON.String)
		}

		messages = append(messages, msg)
	}

	return messages
}

// parseContentJSON parses the content_json into structured blocks
func parseContentJSON(jsonStr string) []contentBlock {
	// The actual structure has nested tool_use/tool_result objects
	var rawBlocks []struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		Thinking string `json:"thinking,omitempty"`
		// Nested structure for tool_use
		ToolUse *struct {
			ID    string      `json:"id"`
			Name  string      `json:"name"`
			Input interface{} `json:"input"`
		} `json:"tool_use,omitempty"`
		// Nested structure for tool_result
		ToolResult *struct {
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
			IsError   bool   `json:"is_error"`
		} `json:"tool_result,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawBlocks); err != nil {
		return nil
	}

	blocks := make([]contentBlock, 0, len(rawBlocks))
	for _, raw := range rawBlocks {
		block := contentBlock{Type: raw.Type}

		switch raw.Type {
		case "text":
			block.Text = raw.Text
		case "thinking":
			block.Thinking = raw.Thinking
		case "tool_use":
			if raw.ToolUse != nil {
				block.ToolUse = &toolUseBlock{
					ID:    raw.ToolUse.ID,
					Name:  raw.ToolUse.Name,
					Input: raw.ToolUse.Input,
				}
			}
		case "tool_result":
			if raw.ToolResult != nil {
				block.ToolResult = &toolResultBlock{
					ToolUseID: raw.ToolResult.ToolUseID,
					Content:   raw.ToolResult.Content,
					IsError:   raw.ToolResult.IsError,
				}
			}
		}

		blocks = append(blocks, block)
	}

	return blocks
}
