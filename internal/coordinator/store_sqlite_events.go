package coordinator

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// StoreTaskEvent saves a task streaming event to the database
func (s *SQLiteStore) StoreTaskEvent(ctx context.Context, event *TaskEventRecord) error {
	query := `
		INSERT INTO task_events (task_id, thread_id, stream_type, turn_num, text, tool_name, tool_input, tool_output, error_msg, status, tokens_in, tokens_out, cost, duration_sec, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		event.TaskID, event.ThreadID, event.StreamType, event.TurnNum,
		event.Text, event.ToolName, event.ToolInput, event.ToolOutput,
		event.ErrorMsg, event.Status, event.TokensIn, event.TokensOut,
		event.Cost, event.DurationSec, time.Now(),
	)
	return err
}

// GetTaskEvents retrieves all events for a task
func (s *SQLiteStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*TaskEventRecord, error) {
	query := `
		SELECT id, task_id, thread_id, stream_type, turn_num, text, tool_name, tool_input, tool_output, error_msg, status, tokens_in, tokens_out, cost, duration_sec, created_at
		FROM task_events WHERE task_id = ?
		ORDER BY id ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*TaskEventRecord
	for rows.Next() {
		event := &TaskEventRecord{}
		var threadID, text, toolName, toolInput, toolOutput, errorMsg, status sql.NullString

		err := rows.Scan(
			&event.ID, &event.TaskID, &threadID, &event.StreamType, &event.TurnNum,
			&text, &toolName, &toolInput, &toolOutput, &errorMsg, &status,
			&event.TokensIn, &event.TokensOut, &event.Cost, &event.DurationSec, &event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if threadID.Valid {
			event.ThreadID = threadID.String
		}
		if text.Valid {
			event.Text = text.String
		}
		if toolName.Valid {
			event.ToolName = toolName.String
		}
		if toolInput.Valid {
			event.ToolInput = toolInput.String
		}
		if toolOutput.Valid {
			event.ToolOutput = toolOutput.String
		}
		if errorMsg.Valid {
			event.ErrorMsg = errorMsg.String
		}
		if status.Valid {
			event.Status = status.String
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// DeleteTaskEvents removes all events for a task
func (s *SQLiteStore) DeleteTaskEvents(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM task_events WHERE task_id = ?", taskID)
	return err
}

// DeleteOldTaskEvents removes events older than the specified duration
func (s *SQLiteStore) DeleteOldTaskEvents(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.ExecContext(ctx, "DELETE FROM task_events WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}
