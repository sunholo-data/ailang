// Package observatory provides session tracking for Claude Code hooks.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Session represents a Claude Code session with workspace metadata.
type Session struct {
	SessionID     string     `json:"session_id"`
	Workspace     string     `json:"workspace"`
	ClaudeVersion string     `json:"claude_version,omitempty"`
	Source        string     `json:"source"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	TurnCount     int        `json:"turn_count"`
	ToolCount     int        `json:"tool_count"`
}

// SessionTool represents a tool call within a Claude Code session.
type SessionTool struct {
	ToolUseID    string          `json:"tool_use_id"`
	SessionID    string          `json:"session_id"`
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time,omitempty"`
	Success      *bool           `json:"success,omitempty"`
}

// UpsertSession inserts or updates a session record.
// If the session exists, updates workspace and claude_version (for reconnection scenarios).
func (s *Store) UpsertSession(ctx context.Context, sessionID, workspace, version, source string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}
	if source == "" {
		source = "hook"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (session_id, workspace, claude_version, source, started_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(session_id) DO UPDATE SET
			workspace = excluded.workspace,
			claude_version = COALESCE(excluded.claude_version, sessions.claude_version)
	`, sessionID, workspace, version, source)
	if err != nil {
		return fmt.Errorf("failed to upsert session: %w", err)
	}
	return nil
}

// GetSessionWorkspace returns the workspace for a session.
// Returns empty string if session not found.
func (s *Store) GetSessionWorkspace(sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}

	var workspace string
	err := s.db.QueryRow(`
		SELECT workspace FROM sessions WHERE session_id = ?
	`, sessionID).Scan(&workspace)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get session workspace: %w", err)
	}
	return workspace, nil
}

// GetSession returns full session details.
func (s *Store) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	var session Session
	var endedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT session_id, workspace, claude_version, source, started_at, ended_at, turn_count
		FROM sessions WHERE session_id = ?
	`, sessionID).Scan(
		&session.SessionID,
		&session.Workspace,
		&session.ClaudeVersion,
		&session.Source,
		&session.StartedAt,
		&endedAt,
		&session.TurnCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	return &session, nil
}

// UpdateSessionEnded marks a session as ended.
func (s *Store) UpdateSessionEnded(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET ended_at = CURRENT_TIMESTAMP WHERE session_id = ?
	`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session ended: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return nil
}

// IncrementSessionTurns increments the turn count for a session.
func (s *Store) IncrementSessionTurns(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET turn_count = turn_count + 1 WHERE session_id = ?
	`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to increment session turns: %w", err)
	}
	return nil
}

// InsertToolStart records the start of a tool call.
func (s *Store) InsertToolStart(ctx context.Context, sessionID, toolUseID, toolName, toolInput string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if toolUseID == "" {
		return fmt.Errorf("tool_use_id is required")
	}
	if toolName == "" {
		return fmt.Errorf("tool_name is required")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO session_tools (tool_use_id, session_id, tool_name, tool_input, start_time)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(tool_use_id) DO UPDATE SET
			tool_name = excluded.tool_name,
			tool_input = excluded.tool_input
	`, toolUseID, sessionID, toolName, toolInput)
	if err != nil {
		return fmt.Errorf("failed to insert tool start: %w", err)
	}
	return nil
}

// FindLatestUnfinishedTool finds the most recent tool call that hasn't completed yet.
// Used to correlate PostToolUse with PreToolUse when tool_use_id is not provided.
func (s *Store) FindLatestUnfinishedTool(ctx context.Context, sessionID, toolName string) (string, error) {
	if sessionID == "" || toolName == "" {
		return "", fmt.Errorf("session_id and tool_name are required")
	}

	var toolUseID string
	err := s.db.QueryRowContext(ctx, `
		SELECT tool_use_id FROM session_tools
		WHERE session_id = ? AND tool_name = ? AND end_time IS NULL
		ORDER BY start_time DESC
		LIMIT 1
	`, sessionID, toolName).Scan(&toolUseID)
	if err != nil {
		return "", err // Includes sql.ErrNoRows
	}
	return toolUseID, nil
}

// UpdateToolEnd records the completion of a tool call.
func (s *Store) UpdateToolEnd(ctx context.Context, toolUseID, toolResponse string, success bool) error {
	if toolUseID == "" {
		return fmt.Errorf("tool_use_id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE session_tools
		SET end_time = CURRENT_TIMESTAMP, tool_response = ?, success = ?
		WHERE tool_use_id = ?
	`, toolResponse, success, toolUseID)
	if err != nil {
		return fmt.Errorf("failed to update tool end: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("tool not found: %s", toolUseID)
	}
	return nil
}

// GetSessionTools returns all tool calls for a session.
func (s *Store) GetSessionTools(ctx context.Context, sessionID string) ([]SessionTool, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT tool_use_id, session_id, tool_name, tool_input, tool_response, start_time, end_time, success
		FROM session_tools
		WHERE session_id = ?
		ORDER BY start_time ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session tools: %w", err)
	}
	defer rows.Close()

	var tools []SessionTool
	for rows.Next() {
		var tool SessionTool
		var endTime sql.NullTime
		var success sql.NullBool
		var toolInput, toolResponse sql.NullString
		err := rows.Scan(
			&tool.ToolUseID,
			&tool.SessionID,
			&tool.ToolName,
			&toolInput,
			&toolResponse,
			&tool.StartTime,
			&endTime,
			&success,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool row: %w", err)
		}
		if endTime.Valid {
			tool.EndTime = &endTime.Time
		}
		if success.Valid {
			tool.Success = &success.Bool
		}
		if toolInput.Valid {
			tool.ToolInput = json.RawMessage(toolInput.String)
		}
		if toolResponse.Valid {
			tool.ToolResponse = json.RawMessage(toolResponse.String)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// GetToolsForSessions returns tools for multiple session IDs, grouped by session.
// This is used for enriching spans with tool metadata.
func (s *Store) GetToolsForSessions(ctx context.Context, sessionIDs []string) (map[string][]SessionTool, error) {
	if len(sessionIDs) == 0 {
		return make(map[string][]SessionTool), nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(sessionIDs))
	args := make([]interface{}, len(sessionIDs))
	for i, id := range sessionIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT tool_use_id, session_id, tool_name, tool_input, tool_response, start_time, end_time, success
		FROM session_tools
		WHERE session_id IN (%s)
		ORDER BY start_time ASC
	`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query session tools: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]SessionTool)
	for rows.Next() {
		var tool SessionTool
		var endTime sql.NullTime
		var success sql.NullBool
		var toolInput, toolResponse sql.NullString
		err := rows.Scan(
			&tool.ToolUseID,
			&tool.SessionID,
			&tool.ToolName,
			&toolInput,
			&toolResponse,
			&tool.StartTime,
			&endTime,
			&success,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool row: %w", err)
		}
		if endTime.Valid {
			tool.EndTime = &endTime.Time
		}
		if success.Valid {
			tool.Success = &success.Bool
		}
		if toolInput.Valid {
			tool.ToolInput = json.RawMessage(toolInput.String)
		}
		if toolResponse.Valid {
			tool.ToolResponse = json.RawMessage(toolResponse.String)
		}
		result[tool.SessionID] = append(result[tool.SessionID], tool)
	}
	return result, nil
}

// BackfillSpansWorkspace updates existing spans that have the given session.id
// but are missing workspace information. Returns the number of spans updated.
func (s *Store) BackfillSpansWorkspace(ctx context.Context, sessionID, workspace string) (int64, error) {
	if sessionID == "" || workspace == "" {
		return 0, nil
	}

	// Update spans where session.id matches but process.cwd is missing
	// The session.id is stored in attributes or resource_attributes as JSON
	result, err := s.db.ExecContext(ctx, `
		UPDATE spans
		SET resource_attributes = json_set(
			COALESCE(resource_attributes, '{}'),
			'$."process.cwd"',
			?
		)
		WHERE (
			json_extract(attributes, '$."session.id"') = ?
			OR json_extract(resource_attributes, '$."session.id"') = ?
		)
		AND (
			json_extract(resource_attributes, '$."process.cwd"') IS NULL
			OR json_extract(resource_attributes, '$."process.cwd"') = ''
		)
	`, workspace, sessionID, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to backfill spans workspace: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// GetToolsByTimestampRange returns tools that started within a time range.
// Used for cross-system correlation when session IDs don't match.
func (s *Store) GetToolsByTimestampRange(ctx context.Context, start, end time.Time, toolName string) ([]SessionTool, error) {
	var query string
	var args []interface{}

	// Format times in SQLite datetime format for BETWEEN query
	// Convert to UTC first: session_tools stores UTC (via CURRENT_TIMESTAMP)
	// Input times may have timezone offsets that need normalizing
	startStr := start.UTC().Format("2006-01-02 15:04:05")
	endStr := end.UTC().Format("2006-01-02 15:04:05")

	if toolName != "" {
		query = `
			SELECT tool_use_id, session_id, tool_name, tool_input, tool_response, start_time, end_time, success
			FROM session_tools
			WHERE start_time BETWEEN ? AND ? AND tool_name = ?
			ORDER BY start_time ASC
		`
		args = []interface{}{startStr, endStr, toolName}
	} else {
		query = `
			SELECT tool_use_id, session_id, tool_name, tool_input, tool_response, start_time, end_time, success
			FROM session_tools
			WHERE start_time BETWEEN ? AND ?
			ORDER BY start_time ASC
		`
		args = []interface{}{startStr, endStr}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tools by timestamp: %w", err)
	}
	defer rows.Close()

	var tools []SessionTool
	for rows.Next() {
		var tool SessionTool
		var success sql.NullBool
		var toolInput, toolResponse sql.NullString
		var startTimeRaw string
		var endTimeRaw sql.NullString
		err := rows.Scan(
			&tool.ToolUseID,
			&tool.SessionID,
			&tool.ToolName,
			&toolInput,
			&toolResponse,
			&startTimeRaw,
			&endTimeRaw,
			&success,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool row: %w", err)
		}

		// Parse start_time - SQLite driver returns ISO format with Z suffix
		// The stored time is from CURRENT_TIMESTAMP (UTC), but the driver converts it
		// Parse as RFC3339 and convert to local timezone for comparison with spans
		parsedTime, parseErr := time.Parse(time.RFC3339, startTimeRaw)
		if parseErr != nil {
			// Try space format as fallback (raw SQLite format)
			parsedTime, _ = time.ParseInLocation("2006-01-02 15:04:05", startTimeRaw, time.Local)
		}
		// Convert UTC to local for comparison with spans (which use local time with TZ offset)
		tool.StartTime = parsedTime.In(time.Local)

		if endTimeRaw.Valid {
			endT, _ := time.Parse(time.RFC3339, endTimeRaw.String)
			if endT.IsZero() {
				endT, _ = time.ParseInLocation("2006-01-02 15:04:05", endTimeRaw.String, time.Local)
			}
			endT = endT.In(time.Local)
			tool.EndTime = &endT
		}
		if success.Valid {
			tool.Success = &success.Bool
		}
		if toolInput.Valid {
			tool.ToolInput = json.RawMessage(toolInput.String)
		}
		if toolResponse.Valid {
			tool.ToolResponse = json.RawMessage(toolResponse.String)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// ListRecentSessions returns recent sessions ordered by start time.
func (s *Store) ListRecentSessions(ctx context.Context, limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.session_id, s.workspace, s.claude_version, s.source, s.started_at, s.ended_at, s.turn_count,
		       COALESCE((SELECT COUNT(*) FROM session_tools t WHERE t.session_id = s.session_id), 0) as tool_count
		FROM sessions s
		ORDER BY s.started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		var endedAt sql.NullTime
		err := rows.Scan(
			&session.SessionID,
			&session.Workspace,
			&session.ClaudeVersion,
			&session.Source,
			&session.StartedAt,
			&endedAt,
			&session.TurnCount,
			&session.ToolCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// GetToolForSpan finds the session_tool that best matches a span by timestamp + tool name.
// Uses a ±10 second window to handle clock drift and hook execution delay.
// Returns nil if no matching tool is found.
func (s *Store) GetToolForSpan(ctx context.Context, sessionID, toolName string, spanTime time.Time) (*SessionTool, error) {
	if sessionID == "" || toolName == "" {
		return nil, nil // No match possible without session ID and tool name
	}

	// UTC normalize for SQLite BETWEEN query
	startStr := spanTime.Add(-10 * time.Second).UTC().Format("2006-01-02 15:04:05")
	endStr := spanTime.Add(10 * time.Second).UTC().Format("2006-01-02 15:04:05")
	centerStr := spanTime.UTC().Format("2006-01-02 15:04:05")

	// Query with session_id, tool_name, and time window
	// Order by closest to span time (ABS distance)
	row := s.db.QueryRowContext(ctx, `
		SELECT tool_use_id, session_id, tool_name, tool_input, tool_response, start_time, end_time, success
		FROM session_tools
		WHERE session_id = ? AND tool_name = ?
		AND start_time BETWEEN ? AND ?
		ORDER BY ABS(julianday(start_time) - julianday(?))
		LIMIT 1
	`, sessionID, toolName, startStr, endStr, centerStr)

	var tool SessionTool
	var success sql.NullBool
	var toolInput, toolResponse sql.NullString
	var startTimeRaw string
	var endTimeRaw sql.NullString

	err := row.Scan(
		&tool.ToolUseID,
		&tool.SessionID,
		&tool.ToolName,
		&toolInput,
		&toolResponse,
		&startTimeRaw,
		&endTimeRaw,
		&success,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No matching tool found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query tool for span: %w", err)
	}

	// Parse start_time
	if t, err := time.Parse("2006-01-02 15:04:05", startTimeRaw); err == nil {
		tool.StartTime = t
	} else if t, err := time.Parse(time.RFC3339, startTimeRaw); err == nil {
		tool.StartTime = t
	}

	// Parse end_time if present
	if endTimeRaw.Valid && endTimeRaw.String != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", endTimeRaw.String); err == nil {
			tool.EndTime = &t
		} else if t, err := time.Parse(time.RFC3339, endTimeRaw.String); err == nil {
			tool.EndTime = &t
		}
	}

	// Handle nullable fields
	if success.Valid {
		tool.Success = &success.Bool
	}
	if toolInput.Valid {
		tool.ToolInput = json.RawMessage(toolInput.String)
	}
	if toolResponse.Valid {
		tool.ToolResponse = json.RawMessage(toolResponse.String)
	}

	return &tool, nil
}
