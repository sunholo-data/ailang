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
		thread_id TEXT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		type TEXT NOT NULL,
		priority INTEGER DEFAULT 5,
		status TEXT NOT NULL DEFAULT 'pending',
		provider TEXT,
		agent_id TEXT,
		worktree_id TEXT,
		worktree_path TEXT,
		session_id TEXT,
		fingerprint INTEGER,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		duration_ns INTEGER,
		error TEXT,
		output TEXT,
		cost REAL DEFAULT 0,
		tokens_used INTEGER DEFAULT 0,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		peak_cpu REAL DEFAULT 0,
		peak_memory_mb REAL DEFAULT 0,
		workspace TEXT,
		github_issue INTEGER,
		stage TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
	CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
	CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
	CREATE INDEX IF NOT EXISTS idx_tasks_fingerprint ON tasks(fingerprint);
	CREATE INDEX IF NOT EXISTS idx_tasks_message_id ON tasks(message_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_thread_id ON tasks(thread_id);
	-- Note: workspace, github_issue, stage indexes created after ALTER TABLE (for existing DBs)

	-- Approval requests for human-in-the-loop checkpoints
	CREATE TABLE IF NOT EXISTS approval_requests (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		type TEXT NOT NULL,
		description TEXT NOT NULL,
		context_json TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		resolved_by TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME,
		timeout_at DATETIME,
		auto_reject INTEGER DEFAULT 0,
		FOREIGN KEY (task_id) REFERENCES tasks(id)
	);

	CREATE INDEX IF NOT EXISTS idx_approvals_task_id ON approval_requests(task_id);
	CREATE INDEX IF NOT EXISTS idx_approvals_status ON approval_requests(status);

	-- Task streaming events for replay/history
	CREATE TABLE IF NOT EXISTS task_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		thread_id TEXT,
		stream_type TEXT NOT NULL,
		turn_num INTEGER DEFAULT 0,
		text TEXT,
		tool_name TEXT,
		tool_input TEXT,
		tool_output TEXT,
		error_msg TEXT,
		status TEXT,
		tokens_in INTEGER DEFAULT 0,
		tokens_out INTEGER DEFAULT 0,
		cost REAL DEFAULT 0,
		duration_sec INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id)
	);

	CREATE INDEX IF NOT EXISTS idx_task_events_task_id ON task_events(task_id);
	CREATE INDEX IF NOT EXISTS idx_task_events_created_at ON task_events(created_at);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Add new columns if they don't exist (for existing databases)
	alterQueries := []string{
		"ALTER TABLE tasks ADD COLUMN input_tokens INTEGER DEFAULT 0",
		"ALTER TABLE tasks ADD COLUMN output_tokens INTEGER DEFAULT 0",
		"ALTER TABLE tasks ADD COLUMN peak_cpu REAL DEFAULT 0",
		"ALTER TABLE tasks ADD COLUMN peak_memory_mb REAL DEFAULT 0",
		"ALTER TABLE tasks ADD COLUMN workspace TEXT",
		"ALTER TABLE tasks ADD COLUMN worktree_path TEXT",
		"ALTER TABLE tasks ADD COLUMN session_id TEXT",
		// M-COORD-GITHUB-AUTO-ROUTING: GitHub integration columns
		"ALTER TABLE tasks ADD COLUMN github_issue INTEGER",
		"ALTER TABLE tasks ADD COLUMN stage TEXT",
		"ALTER TABLE tasks ADD COLUMN design_doc_path TEXT",
		"ALTER TABLE tasks ADD COLUMN sprint_plan_path TEXT",
		// Generic agent tracking
		"ALTER TABLE tasks ADD COLUMN agent_id TEXT",
	}
	for _, q := range alterQueries {
		_, _ = s.db.Exec(q) // Ignore errors - columns may already exist
	}

	// Add new indexes if they don't exist
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace)")
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_github_issue ON tasks(github_issue)")
	_, _ = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_stage ON tasks(stage)")

	return nil
}

// CreateTask creates a new task
func (s *SQLiteStore) CreateTask(ctx context.Context, task *TaskRecord) error {
	query := `
		INSERT INTO tasks (id, message_id, thread_id, title, content, type, priority, status, workspace, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.MessageID, task.ThreadID, task.Title, task.Content,
		task.Type, task.Priority, task.Status, task.Workspace, task.CreatedAt,
	)
	return err
}

// GetTask retrieves a task by ID
func (s *SQLiteStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	query := `
		SELECT id, message_id, thread_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       created_at, started_at, completed_at, duration_ns,
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
			provider = ?, worktree_id = ?, thread_id = ?, workspace = ?,
			started_at = ?, completed_at = ?, duration_ns = ?,
			error = ?, output = ?, cost = ?, tokens_used = ?
		WHERE id = ?
	`
	var durationNs int64
	if task.Duration > 0 {
		durationNs = int64(task.Duration)
	}

	_, err := s.db.ExecContext(ctx, query,
		task.Title, task.Content, task.Type, task.Priority, task.Status,
		task.Provider, task.WorktreeID, task.ThreadID, task.Workspace,
		task.StartedAt, task.CompletedAt, durationNs,
		task.Error, task.Output, task.Cost, task.TokensUsed,
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
		SELECT id, message_id, thread_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       created_at, started_at, completed_at, duration_ns,
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

	if filter.Workspace != "" {
		query.WriteString(" AND workspace = ?")
		args = append(args, filter.Workspace)
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
		ByType:      make(map[string]int),
		ByProvider:  make(map[string]int),
		ByWorkspace: make(map[string]int),
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
		case TaskStatusPendingApproval:
			stats.PendingApprovals += count
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

	// Count by workspace
	rows4, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(workspace, 'unknown'), COUNT(*) FROM tasks WHERE workspace IS NOT NULL AND workspace != '' GROUP BY workspace
	`)
	if err != nil {
		return nil, err
	}
	defer rows4.Close()

	for rows4.Next() {
		var workspace string
		var count int
		if err := rows4.Scan(&workspace, &count); err != nil {
			return nil, err
		}
		stats.ByWorkspace[workspace] = count
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

// MarkTaskPendingApproval marks a task as awaiting human approval
func (s *SQLiteStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch string, result *ExecuteResult) error {
	now := time.Now()
	// Store status, worktree path, branch, AND execution metrics (cost, tokens) to avoid race condition
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET
			status = ?, completed_at = ?, worktree_path = ?, worktree_id = ?,
			duration_ns = ?, output = ?, cost = ?, tokens_used = ?
		WHERE id = ?`,
		TaskStatusPendingApproval, now, worktreePath, worktreeBranch,
		int64(result.Duration), result.Output, result.Cost, result.TokensUsed, id,
	)
	return err
}

// MarkTaskRejected marks a task as rejected by human
func (s *SQLiteStore) MarkTaskRejected(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?",
		TaskStatusRejected, now, id,
	)
	return err
}

// RequeueTask resets a task to pending status for re-execution.
// Used by approval handlers to trigger the next pipeline stage.
func (s *SQLiteStore) RequeueTask(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, started_at = NULL, completed_at = NULL WHERE id = ?",
		TaskStatusPending, id,
	)
	return err
}

// FindDuplicateTask finds a similar task by fingerprint
func (s *SQLiteStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	// For now, exact match only (SimHash comparison would require custom SQLite function)
	// In practice, you'd compute hamming distance in Go after fetching candidates
	row := s.db.QueryRowContext(ctx,
		`SELECT id, message_id, thread_id, title, content, type, priority, status, provider, agent_id,
		        worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		        created_at, started_at, completed_at, duration_ns,
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

// SetTaskThreadID links a task to a thread in collaboration.db for dashboard visibility
func (s *SQLiteStore) SetTaskThreadID(ctx context.Context, id string, threadID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET thread_id = ? WHERE id = ?",
		threadID, id,
	)
	return err
}

// SetTaskGithubIssue links a task to a GitHub issue number
func (s *SQLiteStore) SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET github_issue = ? WHERE id = ?",
		issueNum, id,
	)
	return err
}

// SetTaskStage sets the pipeline stage for a task
func (s *SQLiteStore) SetTaskStage(ctx context.Context, id string, stage TaskStage) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET stage = ? WHERE id = ?",
		string(stage), id,
	)
	return err
}

// SetTaskDesignDocPath stores the design doc path for a task
func (s *SQLiteStore) SetTaskDesignDocPath(ctx context.Context, id string, path string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET design_doc_path = ? WHERE id = ?",
		path, id,
	)
	return err
}

// SetTaskSprintPlanPath stores the sprint plan path for a task
func (s *SQLiteStore) SetTaskSprintPlanPath(ctx context.Context, id string, path string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET sprint_plan_path = ? WHERE id = ?",
		path, id,
	)
	return err
}

// GetTasksByGithubIssue retrieves all tasks linked to a GitHub issue
func (s *SQLiteStore) GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*TaskRecord, error) {
	query := `
		SELECT id, message_id, thread_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used
		FROM tasks WHERE github_issue = ?
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, issueNum)
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

// GetTasksByStage retrieves all tasks in a specific pipeline stage
func (s *SQLiteStore) GetTasksByStage(ctx context.Context, stage TaskStage) ([]*TaskRecord, error) {
	query := `
		SELECT id, message_id, thread_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used
		FROM tasks WHERE stage = ?
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, string(stage))
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

// UpdateTaskMetrics updates peak resource metrics for a task
func (s *SQLiteStore) UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET peak_cpu = ?, peak_memory_mb = ? WHERE id = ?",
		peakCPU, peakMemory, id,
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

// RecoverStaleTasks marks stale running/queued tasks as cancelled on daemon startup.
// This handles tasks that were running when the daemon crashed or was killed.
func (s *SQLiteStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) {
	cutoff := time.Now().Add(-staleThreshold)
	now := time.Now()

	// Cancel tasks that are running/queued but started more than staleThreshold ago
	result, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, completed_at = ?, error = ?
		 WHERE status IN (?, ?) AND (started_at < ? OR (started_at IS NULL AND created_at < ?))`,
		TaskStatusCancelled, now, "Recovered: task was stale after daemon restart",
		TaskStatusRunning, TaskStatusQueued, cutoff, cutoff,
	)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// RetryAllFailedTasks resets all failed tasks to pending so they will be retried.
func (s *SQLiteStore) RetryAllFailedTasks(ctx context.Context) (int, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, error = NULL, started_at = NULL, completed_at = NULL
		 WHERE status = ?`,
		TaskStatusPending, TaskStatusFailed,
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
	var provider, agentID, worktreeID, worktreePath, workspace, errStr, output, threadID, stage sql.NullString
	var designDocPath, sprintPlanPath sql.NullString
	var githubIssue sql.NullInt64

	err := row.Scan(
		&task.ID, &task.MessageID, &threadID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider, &agentID,
		&worktreeID, &worktreePath, &workspace, &githubIssue, &stage, &designDocPath, &sprintPlanPath,
		&task.CreatedAt, &startedAt, &completedAt,
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
	if agentID.Valid {
		task.AgentID = agentID.String
	}
	if worktreeID.Valid {
		task.WorktreeID = worktreeID.String
	}
	if worktreePath.Valid {
		task.WorktreePath = worktreePath.String
	}
	if workspace.Valid {
		task.Workspace = workspace.String
	}
	if errStr.Valid {
		task.Error = errStr.String
	}
	if output.Valid {
		task.Output = output.String
	}
	if threadID.Valid {
		task.ThreadID = threadID.String
	}
	if githubIssue.Valid {
		task.GithubIssue = int(githubIssue.Int64)
	}
	if stage.Valid {
		task.Stage = TaskStage(stage.String)
	}
	if designDocPath.Valid {
		task.DesignDocPath = designDocPath.String
	}
	if sprintPlanPath.Valid {
		task.SprintPlanPath = sprintPlanPath.String
	}

	return task, nil
}

// Helper to scan a task from rows
func (s *SQLiteStore) scanTaskFromRows(rows *sql.Rows) (*TaskRecord, error) {
	task := &TaskRecord{}
	var startedAt, completedAt sql.NullTime
	var durationNs sql.NullInt64
	var provider, agentID, worktreeID, worktreePath, workspace, errStr, output, threadID, stage sql.NullString
	var designDocPath, sprintPlanPath sql.NullString
	var githubIssue sql.NullInt64

	err := rows.Scan(
		&task.ID, &task.MessageID, &threadID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider, &agentID,
		&worktreeID, &worktreePath, &workspace, &githubIssue, &stage, &designDocPath, &sprintPlanPath,
		&task.CreatedAt, &startedAt, &completedAt,
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
	if agentID.Valid {
		task.AgentID = agentID.String
	}
	if worktreeID.Valid {
		task.WorktreeID = worktreeID.String
	}
	if worktreePath.Valid {
		task.WorktreePath = worktreePath.String
	}
	if workspace.Valid {
		task.Workspace = workspace.String
	}
	if errStr.Valid {
		task.Error = errStr.String
	}
	if output.Valid {
		task.Output = output.String
	}
	if threadID.Valid {
		task.ThreadID = threadID.String
	}
	if githubIssue.Valid {
		task.GithubIssue = int(githubIssue.Int64)
	}
	if stage.Valid {
		task.Stage = TaskStage(stage.String)
	}
	if designDocPath.Valid {
		task.DesignDocPath = designDocPath.String
	}
	if sprintPlanPath.Valid {
		task.SprintPlanPath = sprintPlanPath.String
	}

	return task, nil
}

// ApprovalRequestRecord is the database record for an approval request
type ApprovalRequestRecord struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	ContextJSON string     `json:"context_json,omitempty"`
	Status      string     `json:"status"`
	ResolvedBy  string     `json:"resolved_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	TimeoutAt   *time.Time `json:"timeout_at,omitempty"`
	AutoReject  bool       `json:"auto_reject"`
}

// CreateApprovalRequest creates a new approval request in the database
func (s *SQLiteStore) CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error {
	query := `
		INSERT INTO approval_requests (id, task_id, type, description, context_json, status, timeout_at, auto_reject, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		req.ID, req.TaskID, req.Type, req.Description, req.ContextJSON,
		req.Status, req.TimeoutAt, req.AutoReject, req.CreatedAt,
	)
	return err
}

// GetApprovalRequest retrieves an approval request by ID
func (s *SQLiteStore) GetApprovalRequest(ctx context.Context, id string) (*ApprovalRequestRecord, error) {
	query := `
		SELECT id, task_id, type, description, context_json, status, resolved_by, created_at, resolved_at, timeout_at, auto_reject
		FROM approval_requests WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanApprovalRequest(row)
}

// GetApprovalRequestByTask retrieves a pending approval request for a task
func (s *SQLiteStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	query := `
		SELECT id, task_id, type, description, context_json, status, resolved_by, created_at, resolved_at, timeout_at, auto_reject
		FROM approval_requests WHERE task_id = ? AND status = 'pending'
		ORDER BY created_at DESC LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, query, taskID)
	req, err := s.scanApprovalRequest(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return req, err
}

// ListPendingApprovals retrieves all pending approval requests
func (s *SQLiteStore) ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	query := `
		SELECT id, task_id, type, description, context_json, status, resolved_by, created_at, resolved_at, timeout_at, auto_reject
		FROM approval_requests WHERE status = 'pending'
		ORDER BY created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ApprovalRequestRecord
	for rows.Next() {
		req, err := s.scanApprovalRequestFromRows(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

// ResolveApprovalRequest marks an approval request as approved or rejected
func (s *SQLiteStore) ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE approval_requests SET status = ?, resolved_by = ?, resolved_at = ? WHERE id = ?",
		status, resolvedBy, now, id,
	)
	return err
}

// ResolveApprovalRequestByTask marks an approval request for a task as approved or rejected
func (s *SQLiteStore) ResolveApprovalRequestByTask(ctx context.Context, taskID string, status string, resolvedBy string) error {
	now := time.Now()
	result, err := s.db.ExecContext(ctx,
		"UPDATE approval_requests SET status = ?, resolved_by = ?, resolved_at = ? WHERE task_id = ? AND status = 'pending'",
		status, resolvedBy, now, taskID,
	)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return fmt.Errorf("no pending approval request found for task: %s", taskID)
	}

	// Also update the task status to match the approval decision
	var taskStatus TaskStatus
	switch status {
	case "rejected":
		taskStatus = TaskStatusRejected
	case "approved":
		taskStatus = TaskStatusCompleted
	default:
		return nil // Unknown status, don't update task
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?",
		taskStatus, now, taskID,
	)
	return err
}

// ReopenTask moves a rejected/cancelled task back to pending_approval status
func (s *SQLiteStore) ReopenTask(ctx context.Context, taskID string) error {
	// First check if the task exists and is in a reopenable state
	var currentStatus string
	err := s.db.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return err
	}

	// Only allow reopening rejected or cancelled tasks
	if currentStatus != string(TaskStatusRejected) && currentStatus != string(TaskStatusCancelled) {
		return fmt.Errorf("cannot reopen task with status %q (only rejected or cancelled tasks can be reopened)", currentStatus)
	}

	// Update task status back to pending_approval
	_, err = s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?",
		TaskStatusPendingApproval, taskID,
	)
	if err != nil {
		return err
	}

	// Reset the approval request back to pending (or create one if missing)
	result, err := s.db.ExecContext(ctx,
		"UPDATE approval_requests SET status = 'pending', resolved_by = NULL, resolved_at = NULL WHERE task_id = ?",
		taskID,
	)
	if err != nil {
		return err
	}

	// If no approval request existed, create one
	count, _ := result.RowsAffected()
	if count == 0 {
		approvalID := fmt.Sprintf("apr-%s", taskID[5:]) // apr-<hash> from task-<hash>
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO approval_requests (id, task_id, type, description, status, created_at)
			 VALUES (?, ?, 'merge', 'Task reopened for approval', 'pending', ?)`,
			approvalID, taskID, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to create approval request: %w", err)
		}
	}

	return nil
}

// DeleteOldApprovals removes approval requests older than the specified duration
func (s *SQLiteStore) DeleteOldApprovals(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM approval_requests WHERE created_at < ? AND status != 'pending'",
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// scanApprovalRequest scans a single approval request from a row
func (s *SQLiteStore) scanApprovalRequest(row *sql.Row) (*ApprovalRequestRecord, error) {
	req := &ApprovalRequestRecord{}
	var resolvedBy, contextJSON sql.NullString
	var resolvedAt, timeoutAt sql.NullTime

	err := row.Scan(
		&req.ID, &req.TaskID, &req.Type, &req.Description, &contextJSON,
		&req.Status, &resolvedBy, &req.CreatedAt, &resolvedAt, &timeoutAt, &req.AutoReject,
	)
	if err != nil {
		return nil, err
	}

	if resolvedBy.Valid {
		req.ResolvedBy = resolvedBy.String
	}
	if contextJSON.Valid {
		req.ContextJSON = contextJSON.String
	}
	if resolvedAt.Valid {
		req.ResolvedAt = &resolvedAt.Time
	}
	if timeoutAt.Valid {
		req.TimeoutAt = &timeoutAt.Time
	}

	return req, nil
}

// scanApprovalRequestFromRows scans an approval request from rows
func (s *SQLiteStore) scanApprovalRequestFromRows(rows *sql.Rows) (*ApprovalRequestRecord, error) {
	req := &ApprovalRequestRecord{}
	var resolvedBy, contextJSON sql.NullString
	var resolvedAt, timeoutAt sql.NullTime

	err := rows.Scan(
		&req.ID, &req.TaskID, &req.Type, &req.Description, &contextJSON,
		&req.Status, &resolvedBy, &req.CreatedAt, &resolvedAt, &timeoutAt, &req.AutoReject,
	)
	if err != nil {
		return nil, err
	}

	if resolvedBy.Valid {
		req.ResolvedBy = resolvedBy.String
	}
	if contextJSON.Valid {
		req.ContextJSON = contextJSON.String
	}
	if resolvedAt.Valid {
		req.ResolvedAt = &resolvedAt.Time
	}
	if timeoutAt.Valid {
		req.TimeoutAt = &timeoutAt.Time
	}

	return req, nil
}

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
