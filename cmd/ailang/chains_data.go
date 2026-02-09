package main

import (
	"context"
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

// chatMessageExport represents a chat message with full tool data (CLI presentation type)
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

// --- Coordinator.db queries (kept as direct SQL - separate database) ---

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

// --- Observatory.db queries (via Backend Store methods) ---

// getChatMessagesForTask fetches chat messages by task_id directly (M-DETERMINISTIC-CHAT-LINKING)
// Uses observatory.Backend Store methods instead of raw SQL.
func getChatMessagesForTask(taskID string) []chatMessageExport {
	if taskID == "" {
		return nil
	}

	backend, err := getObservatoryBackend()
	if err != nil {
		return nil
	}
	defer backend.Close()

	msgs, err := backend.GetChatMessagesByTaskID(context.Background(), taskID)
	if err != nil {
		return nil
	}
	return convertChatMessages(msgs)
}

// getChatMessages fetches chat messages from observatory.db with full content
func getChatMessages(sessionID string) []chatMessageExport {
	return getChatMessagesInRange(sessionID, time.Time{}, time.Time{})
}

// getChatMessagesInRange fetches chat messages within a time range
func getChatMessagesInRange(sessionID string, startTime, endTime time.Time) []chatMessageExport {
	backend, err := getObservatoryBackend()
	if err != nil {
		return nil
	}
	defer backend.Close()

	msgs, err := backend.GetChatMessagesBySession(context.Background(), sessionID, startTime, endTime)
	if err != nil {
		return nil
	}
	return convertChatMessages(msgs)
}

// getObservatoryBackend opens the observatory backend (caller must Close)
func getObservatoryBackend() (observatory.Backend, error) {
	dbPath := observatory.DefaultDatabasePath()
	return observatory.NewSQLiteBackendFromPath(dbPath)
}

// convertChatMessages converts Store ChatMessage types to CLI export types
func convertChatMessages(msgs []*observatory.ChatMessage) []chatMessageExport {
	if len(msgs) == 0 {
		return nil
	}
	exports := make([]chatMessageExport, 0, len(msgs))
	for _, msg := range msgs {
		export := chatMessageExport{
			ID:         msg.ID,
			SessionID:  msg.SessionID,
			TurnNumber: msg.TurnNumber,
			Role:       msg.Role,
			Model:      msg.Model,
			TokensIn:   msg.TokensIn,
			TokensOut:  msg.TokensOut,
			Timestamp:  msg.Timestamp.Format(time.RFC3339),
		}
		if msg.ContentJSON != "" {
			export.Content = parseContentJSON(msg.ContentJSON)
		}
		exports = append(exports, export)
	}
	return exports
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
