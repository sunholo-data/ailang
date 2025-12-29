package coordinator

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if dbPath == "" {
		homeDir, _ := os.UserHomeDir()
		dbPath = filepath.Join(homeDir, ".ailang", "state", "coordinator.db")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// migrate creates the necessary tables
func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		message_id TEXT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		type TEXT NOT NULL,
		priority INTEGER DEFAULT 5,
		status TEXT NOT NULL DEFAULT 'pending',
		provider TEXT,
		worktree_id TEXT,
		fingerprint INTEGER,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		duration_ns INTEGER,
		error TEXT,
		output TEXT,
		cost REAL DEFAULT 0,
		tokens_used INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
	CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
	CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
	CREATE INDEX IF NOT EXISTS idx_tasks_fingerprint ON tasks(fingerprint);
	CREATE INDEX IF NOT EXISTS idx_tasks_message_id ON tasks(message_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateTask creates a new task
func (s *SQLiteStore) CreateTask(ctx context.Context, task *TaskRecord) error {
	query := `
		INSERT INTO tasks (id, message_id, title, content, type, priority, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.MessageID, task.Title, task.Content,
		task.Type, task.Priority, task.Status, task.CreatedAt,
	)
	return err
}

// GetTask retrieves a task by ID
func (s *SQLiteStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	query := `
		SELECT id, message_id, title, content, type, priority, status, provider,
		       worktree_id, created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used
		FROM tasks WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanTask(row)
}

// UpdateTask updates an existing task
func (s *SQLiteStore) UpdateTask(ctx context.Context, task *TaskRecord) error {
	query := `
		UPDATE tasks SET
			title = ?, content = ?, type = ?, priority = ?, status = ?,
			provider = ?, worktree_id = ?, started_at = ?, completed_at = ?,
			duration_ns = ?, error = ?, output = ?, cost = ?, tokens_used = ?
		WHERE id = ?
	`
	var durationNs int64
	if task.Duration > 0 {
		durationNs = int64(task.Duration)
	}

	_, err := s.db.ExecContext(ctx, query,
		task.Title, task.Content, task.Type, task.Priority, task.Status,
		task.Provider, task.WorktreeID, task.StartedAt, task.CompletedAt,
		durationNs, task.Error, task.Output, task.Cost, task.TokensUsed,
		task.ID,
	)
	return err
}

// DeleteTask deletes a task
func (s *SQLiteStore) DeleteTask(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	return err
}

// ListTasks retrieves tasks matching the filter
func (s *SQLiteStore) ListTasks(ctx context.Context, filter *TaskFilter) ([]*TaskRecord, error) {
	query := strings.Builder{}
	query.WriteString(`
		SELECT id, message_id, title, content, type, priority, status, provider,
		       worktree_id, created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used
		FROM tasks WHERE 1=1
	`)

	var args []interface{}

	if len(filter.Status) > 0 {
		placeholders := make([]string, len(filter.Status))
		for i, s := range filter.Status {
			placeholders[i] = "?"
			args = append(args, s)
		}
		query.WriteString(" AND status IN (" + strings.Join(placeholders, ",") + ")")
	}

	if len(filter.Type) > 0 {
		placeholders := make([]string, len(filter.Type))
		for i, t := range filter.Type {
			placeholders[i] = "?"
			args = append(args, t)
		}
		query.WriteString(" AND type IN (" + strings.Join(placeholders, ",") + ")")
	}

	if filter.Provider != "" {
		query.WriteString(" AND provider = ?")
		args = append(args, filter.Provider)
	}

	if filter.Since != nil {
		query.WriteString(" AND created_at >= ?")
		args = append(args, filter.Since)
	}

	if filter.Until != nil {
		query.WriteString(" AND created_at <= ?")
		args = append(args, filter.Until)
	}

	// Order by
	orderBy := "created_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	order := "ASC"
	if filter.OrderDesc {
		order = "DESC"
	}
	query.WriteString(fmt.Sprintf(" ORDER BY %s %s", orderBy, order))

	// Limit and offset
	if filter.Limit > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query.WriteString(" OFFSET ?")
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRecord
	for rows.Next() {
		task, err := s.scanTaskFromRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetTaskStats returns aggregate statistics
func (s *SQLiteStore) GetTaskStats(ctx context.Context) (*TaskStats, error) {
	stats := &TaskStats{
		ByType:     make(map[string]int),
		ByProvider: make(map[string]int),
	}

	// Count by status
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM tasks GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.TotalTasks += count
		switch TaskStatus(status) {
		case TaskStatusPending, TaskStatusQueued:
			stats.PendingTasks += count
		case TaskStatusRunning:
			stats.RunningTasks += count
		case TaskStatusCompleted:
			stats.CompletedTasks += count
		case TaskStatusFailed:
			stats.FailedTasks += count
		}
	}

	// Count by type
	rows2, err := s.db.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM tasks GROUP BY type
	`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var taskType string
		var count int
		if err := rows2.Scan(&taskType, &count); err != nil {
			return nil, err
		}
		stats.ByType[taskType] = count
	}

	// Count by provider
	rows3, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(provider, 'unknown'), COUNT(*) FROM tasks WHERE provider IS NOT NULL GROUP BY provider
	`)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()

	for rows3.Next() {
		var provider string
		var count int
		if err := rows3.Scan(&provider, &count); err != nil {
			return nil, err
		}
		stats.ByProvider[provider] = count
	}

	// Totals
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cost), 0), COALESCE(SUM(tokens_used), 0), COALESCE(AVG(duration_ns), 0)
		FROM tasks WHERE status = 'completed'
	`)
	var avgDurationNs float64
	if err := row.Scan(&stats.TotalCost, &stats.TotalTokens, &avgDurationNs); err != nil {
		return nil, err
	}
	stats.AvgDuration = time.Duration(avgDurationNs)

	return stats, nil
}

// MarkTaskQueued marks a task as queued
func (s *SQLiteStore) MarkTaskQueued(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ? WHERE id = ?",
		TaskStatusQueued, id,
	)
	return err
}

// MarkTaskRunning marks a task as running
func (s *SQLiteStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, provider = ?, worktree_id = ?, started_at = ? WHERE id = ?",
		TaskStatusRunning, provider, worktreeID, time.Now(), id,
	)
	return err
}

// MarkTaskCompleted marks a task as completed with results
func (s *SQLiteStore) MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET
			status = ?, completed_at = ?, duration_ns = ?,
			output = ?, cost = ?, tokens_used = ?
		WHERE id = ?`,
		TaskStatusCompleted, now, int64(result.Duration),
		result.Output, result.Cost, result.TokensUsed, id,
	)
	return err
}

// MarkTaskFailed marks a task as failed
func (s *SQLiteStore) MarkTaskFailed(ctx context.Context, id string, taskErr error) error {
	now := time.Now()
	errMsg := ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ?, error = ? WHERE id = ?",
		TaskStatusFailed, now, errMsg, id,
	)
	return err
}

// MarkTaskCancelled marks a task as cancelled
func (s *SQLiteStore) MarkTaskCancelled(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?",
		TaskStatusCancelled, now, id,
	)
	return err
}

// FindDuplicateTask finds a similar task by fingerprint
func (s *SQLiteStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	// For now, exact match only (SimHash comparison would require custom SQLite function)
	// In practice, you'd compute hamming distance in Go after fetching candidates
	row := s.db.QueryRowContext(ctx,
		`SELECT id, message_id, title, content, type, priority, status, provider,
		        worktree_id, created_at, started_at, completed_at, duration_ns,
		        error, output, cost, tokens_used
		FROM tasks WHERE fingerprint = ? AND status != 'cancelled' LIMIT 1`,
		fingerprint,
	)
	task, err := s.scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return task, err
}

// SetTaskFingerprint sets the fingerprint for duplicate detection
func (s *SQLiteStore) SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET fingerprint = ? WHERE id = ?",
		fingerprint, id,
	)
	return err
}

// DeleteOldTasks removes tasks older than the specified duration
func (s *SQLiteStore) DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM tasks WHERE created_at < ? AND status IN (?, ?, ?)",
		cutoff, TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled,
	)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Helper to scan a single task from a row
func (s *SQLiteStore) scanTask(row *sql.Row) (*TaskRecord, error) {
	task := &TaskRecord{}
	var startedAt, completedAt sql.NullTime
	var durationNs sql.NullInt64
	var provider, worktreeID, errStr, output sql.NullString

	err := row.Scan(
		&task.ID, &task.MessageID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider,
		&worktreeID, &task.CreatedAt, &startedAt, &completedAt,
		&durationNs, &errStr, &output, &task.Cost, &task.TokensUsed,
	)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if durationNs.Valid {
		task.Duration = time.Duration(durationNs.Int64)
	}
	if provider.Valid {
		task.Provider = provider.String
	}
	if worktreeID.Valid {
		task.WorktreeID = worktreeID.String
	}
	if errStr.Valid {
		task.Error = errStr.String
	}
	if output.Valid {
		task.Output = output.String
	}

	return task, nil
}

// Helper to scan a task from rows
func (s *SQLiteStore) scanTaskFromRows(rows *sql.Rows) (*TaskRecord, error) {
	task := &TaskRecord{}
	var startedAt, completedAt sql.NullTime
	var durationNs sql.NullInt64
	var provider, worktreeID, errStr, output sql.NullString

	err := rows.Scan(
		&task.ID, &task.MessageID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider,
		&worktreeID, &task.CreatedAt, &startedAt, &completedAt,
		&durationNs, &errStr, &output, &task.Cost, &task.TokensUsed,
	)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if durationNs.Valid {
		task.Duration = time.Duration(durationNs.Int64)
	}
	if provider.Valid {
		task.Provider = provider.String
	}
	if worktreeID.Valid {
		task.WorktreeID = worktreeID.String
	}
	if errStr.Valid {
		task.Error = errStr.String
	}
	if output.Valid {
		task.Output = output.String
	}

	return task, nil
}
