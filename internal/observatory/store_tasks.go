// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"fmt"
)

// TaskListOptions configures task listing.
type TaskListOptions struct {
	WorkspaceID string
	Status      TaskStatus
	SourceType  TaskSourceType
	Limit       int
	Offset      int
}

// CreateTask inserts a new task.
func (s *Store) CreateTask(t *Task) error {
	_, err := s.db.Exec(`
		INSERT INTO tasks (id, workspace_id, parent_task_id, title, description, source_type, source_ref,
		                   status, priority, created_at, started_at, completed_at,
		                   total_duration_ms, total_tokens_in, total_tokens_out,
		                   total_cost_usd, agent_count, span_count, error_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.WorkspaceID, t.ParentTaskID, t.Title, t.Description, t.SourceType, t.SourceRef,
		t.Status, t.Priority, t.CreatedAt, t.StartedAt, t.CompletedAt,
		t.TotalDurationMs, t.TotalTokensIn, t.TotalTokensOut,
		t.TotalCostUSD, t.AgentCount, t.SpanCount, t.ErrorCount)
	return err
}

// GetTask retrieves a task by ID.
func (s *Store) GetTask(id string) (*Task, error) {
	t := &Task{}
	var parentTaskID, desc, sourceRef sql.NullString
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRow(`
		SELECT id, workspace_id, parent_task_id, title, description, source_type, source_ref,
		       status, priority, created_at, started_at, completed_at,
		       total_duration_ms, total_tokens_in, total_tokens_out,
		       total_cost_usd, agent_count, span_count, error_count
		FROM tasks WHERE id = ?
	`, id).Scan(&t.ID, &t.WorkspaceID, &parentTaskID, &t.Title, &desc, &t.SourceType, &sourceRef,
		&t.Status, &t.Priority, &t.CreatedAt, &startedAt, &completedAt,
		&t.TotalDurationMs, &t.TotalTokensIn, &t.TotalTokensOut,
		&t.TotalCostUSD, &t.AgentCount, &t.SpanCount, &t.ErrorCount)
	if err != nil {
		return nil, err
	}
	if parentTaskID.Valid {
		t.ParentTaskID = parentTaskID.String
	}
	if desc.Valid {
		t.Description = desc.String
	}
	if sourceRef.Valid {
		t.SourceRef = sourceRef.String
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return t, nil
}

// ListTasks returns tasks with optional filtering.
func (s *Store) ListTasks(opts TaskListOptions) ([]*Task, error) {
	query := `
		SELECT id, workspace_id, parent_task_id, title, description, source_type, source_ref,
		       status, priority, created_at, started_at, completed_at,
		       total_duration_ms, total_tokens_in, total_tokens_out,
		       total_cost_usd, agent_count, span_count, error_count
		FROM tasks WHERE 1=1
	`
	var args []interface{}

	if opts.WorkspaceID != "" {
		query += " AND workspace_id = ?"
		args = append(args, opts.WorkspaceID)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.SourceType != "" {
		query += " AND source_type = ?"
		args = append(args, opts.SourceType)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		var parentTaskID, desc, sourceRef sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &parentTaskID, &t.Title, &desc, &t.SourceType, &sourceRef,
			&t.Status, &t.Priority, &t.CreatedAt, &startedAt, &completedAt,
			&t.TotalDurationMs, &t.TotalTokensIn, &t.TotalTokensOut,
			&t.TotalCostUSD, &t.AgentCount, &t.SpanCount, &t.ErrorCount); err != nil {
			return nil, err
		}
		if parentTaskID.Valid {
			t.ParentTaskID = parentTaskID.String
		}
		if desc.Valid {
			t.Description = desc.String
		}
		if sourceRef.Valid {
			t.SourceRef = sourceRef.String
		}
		if startedAt.Valid {
			t.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// UpdateTask updates an existing task.
func (s *Store) UpdateTask(t *Task) error {
	_, err := s.db.Exec(`
		UPDATE tasks SET parent_task_id = ?, title = ?, description = ?, source_type = ?, source_ref = ?,
		                 status = ?, priority = ?, started_at = ?, completed_at = ?,
		                 total_duration_ms = ?, total_tokens_in = ?, total_tokens_out = ?,
		                 total_cost_usd = ?, agent_count = ?, span_count = ?, error_count = ?
		WHERE id = ?
	`, t.ParentTaskID, t.Title, t.Description, t.SourceType, t.SourceRef,
		t.Status, t.Priority, t.StartedAt, t.CompletedAt,
		t.TotalDurationMs, t.TotalTokensIn, t.TotalTokensOut,
		t.TotalCostUSD, t.AgentCount, t.SpanCount, t.ErrorCount, t.ID)
	return err
}

// DeleteTask removes a task by ID.
func (s *Store) DeleteTask(id string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}
