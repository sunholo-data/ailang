package coordinator

import (
	"context"
	"database/sql"
	"encoding/json"
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
		// M-DEPRECATE-AILANG-AGENT: Capability detection columns
		"ALTER TABLE tasks ADD COLUMN capabilities_json TEXT",
		"ALTER TABLE tasks ADD COLUMN impact_level TEXT",
		"ALTER TABLE tasks ADD COLUMN estimated_cost REAL DEFAULT 0",
		// Hierarchy tracking for handoffs
		"ALTER TABLE tasks ADD COLUMN parent_task_id TEXT",
		// Iteration tracking for feedback loops (M-TRANSCRIPT)
		"ALTER TABLE tasks ADD COLUMN iteration INTEGER DEFAULT 0",
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
	// Serialize capabilities to JSON
	var capsJSON []byte
	if len(task.Capabilities) > 0 {
		capsJSON, _ = json.Marshal(task.Capabilities)
	}

	query := `
		INSERT INTO tasks (id, message_id, thread_id, parent_task_id, title, content, type, priority, status, workspace,
		                   agent_id, capabilities_json, impact_level, estimated_cost, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.MessageID, task.ThreadID, task.ParentTaskID, task.Title, task.Content,
		task.Type, task.Priority, task.Status, task.Workspace,
		task.AgentID, string(capsJSON), task.ImpactLevel, task.EstimatedCost, task.CreatedAt,
	)
	return err
}

// GetTask retrieves a task by ID
func (s *SQLiteStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	query := `
		SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration,
		       created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used,
		       capabilities_json, impact_level, estimated_cost
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
			session_id = ?, iteration = ?,
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
		task.SessionID, task.Iteration,
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
		SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration,
		       created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used,
		       capabilities_json, impact_level, estimated_cost
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
		ByProvider:  make(map[string]*DetailedStats),
		ByWorkspace: make(map[string]*DetailedStats),
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

	// Count ALL pending approval_requests (includes both merge approvals and handoffs)
	// This replaces the task status count to avoid double-counting
	// (tasks with status=pending_approval also have approval_requests)
	var pendingApprovalRequests int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM approval_requests WHERE status = 'pending'
	`).Scan(&pendingApprovalRequests)
	if err == nil {
		stats.PendingApprovals = pendingApprovalRequests // Replace, don't add
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

	// Count by provider with cost/token breakdown
	rows3, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(provider, 'unknown') as provider,
			COUNT(*) as task_count,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens
		FROM tasks
		WHERE provider IS NOT NULL
		GROUP BY provider
	`)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()

	for rows3.Next() {
		var provider string
		var count int
		var cost float64
		var inputTokens, outputTokens int
		if err := rows3.Scan(&provider, &count, &cost, &inputTokens, &outputTokens); err != nil {
			return nil, err
		}
		stats.ByProvider[provider] = &DetailedStats{
			Count:        count,
			CostUSD:      cost,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		}
	}

	// Count by workspace with cost/token breakdown
	rows4, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(workspace, 'unknown') as workspace,
			COUNT(*) as task_count,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens
		FROM tasks
		WHERE workspace IS NOT NULL AND workspace != ''
		GROUP BY workspace
	`)
	if err != nil {
		return nil, err
	}
	defer rows4.Close()

	for rows4.Next() {
		var workspace string
		var count int
		var cost float64
		var inputTokens, outputTokens int
		if err := rows4.Scan(&workspace, &count, &cost, &inputTokens, &outputTokens); err != nil {
			return nil, err
		}
		stats.ByWorkspace[workspace] = &DetailedStats{
			Count:        count,
			CostUSD:      cost,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		}
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
			output = ?, cost = ?, tokens_used = ?,
			session_id = ?
		WHERE id = ?`,
		TaskStatusCompleted, now, int64(result.Duration),
		result.Output, result.Cost, result.TokensUsed,
		result.SessionID, id,
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

// ResetTaskToPending resets a running task back to pending state.
// Used when worktree limit is reached - task will be retried on next poll.
// CRITICAL: Prevents tasks from being stuck in "running" state forever.
func (s *SQLiteStore) ResetTaskToPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, started_at = NULL WHERE id = ?",
		TaskStatusPending, id,
	)
	return err
}

// FindDuplicateTask finds a similar task by fingerprint
func (s *SQLiteStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	// For now, exact match only (SimHash comparison would require custom SQLite function)
	// In practice, you'd compute hamming distance in Go after fetching candidates
	row := s.db.QueryRowContext(ctx,
		`SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		        worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		        session_id, iteration,
		        created_at, started_at, completed_at, duration_ns,
		        error, output, cost, tokens_used,
		        capabilities_json, impact_level, estimated_cost
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

// GetTaskAgentInfo returns agent info for a task (for cross-db correlation in Control Plane)
// Returns: agentID (used as FromAgent), inbox (used as ToInbox), title
// Note: By convention, agent id == inbox in the agent config (e.g., "design-doc-creator")
func (s *SQLiteStore) GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT agent_id, title FROM tasks WHERE id = ?
	`, taskID).Scan(&agentID, &title)
	if err == sql.ErrNoRows {
		return "", "", "", nil // Not a coordinator task
	}
	if err != nil {
		return "", "", "", err
	}
	// By convention, agent id == inbox in the agent config
	// (e.g., id: "design-doc-creator" → inbox: "design-doc-creator")
	inbox = agentID
	return agentID, inbox, title, nil
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
		SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration,
		       created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used,
		       capabilities_json, impact_level, estimated_cost
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
		SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, workspace, github_issue, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration,
		       created_at, started_at, completed_at, duration_ns,
		       error, output, cost, tokens_used,
		       capabilities_json, impact_level, estimated_cost
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
	var messageID sql.NullString
	var provider, agentID, worktreeID, worktreePath, workspace, errStr, output, threadID, parentTaskID, stage sql.NullString
	var designDocPath, sprintPlanPath sql.NullString
	var sessionID sql.NullString
	var iteration sql.NullInt64
	var githubIssue sql.NullInt64
	var capsJSON, impactLevel sql.NullString
	var estimatedCost sql.NullFloat64

	err := row.Scan(
		&task.ID, &messageID, &threadID, &parentTaskID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider, &agentID,
		&worktreeID, &worktreePath, &workspace, &githubIssue, &stage, &designDocPath, &sprintPlanPath,
		&sessionID, &iteration,
		&task.CreatedAt, &startedAt, &completedAt,
		&durationNs, &errStr, &output, &task.Cost, &task.TokensUsed,
		&capsJSON, &impactLevel, &estimatedCost,
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
	if messageID.Valid {
		task.MessageID = messageID.String
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
	if parentTaskID.Valid {
		task.ParentTaskID = parentTaskID.String
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
	if sessionID.Valid {
		task.SessionID = sessionID.String
	}
	if iteration.Valid {
		task.Iteration = int(iteration.Int64)
	}
	// Capability detection fields (M-DEPRECATE-AILANG-AGENT)
	if capsJSON.Valid && capsJSON.String != "" {
		_ = json.Unmarshal([]byte(capsJSON.String), &task.Capabilities)
	}
	if impactLevel.Valid {
		task.ImpactLevel = impactLevel.String
	}
	if estimatedCost.Valid {
		task.EstimatedCost = estimatedCost.Float64
	}

	return task, nil
}

// Helper to scan a task from rows
func (s *SQLiteStore) scanTaskFromRows(rows *sql.Rows) (*TaskRecord, error) {
	task := &TaskRecord{}
	var startedAt, completedAt sql.NullTime
	var durationNs sql.NullInt64
	var messageID sql.NullString
	var provider, agentID, worktreeID, worktreePath, workspace, errStr, output, threadID, parentTaskID, stage sql.NullString
	var designDocPath, sprintPlanPath sql.NullString
	var sessionID sql.NullString
	var iteration sql.NullInt64
	var githubIssue sql.NullInt64
	var capsJSON, impactLevel sql.NullString
	var estimatedCost sql.NullFloat64

	err := rows.Scan(
		&task.ID, &messageID, &threadID, &parentTaskID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider, &agentID,
		&worktreeID, &worktreePath, &workspace, &githubIssue, &stage, &designDocPath, &sprintPlanPath,
		&sessionID, &iteration,
		&task.CreatedAt, &startedAt, &completedAt,
		&durationNs, &errStr, &output, &task.Cost, &task.TokensUsed,
		&capsJSON, &impactLevel, &estimatedCost,
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
	if messageID.Valid {
		task.MessageID = messageID.String
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
	if parentTaskID.Valid {
		task.ParentTaskID = parentTaskID.String
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
	if sessionID.Valid {
		task.SessionID = sessionID.String
	}
	if iteration.Valid {
		task.Iteration = int(iteration.Int64)
	}
	// Capability detection fields (M-DEPRECATE-AILANG-AGENT)
	if capsJSON.Valid && capsJSON.String != "" {
		_ = json.Unmarshal([]byte(capsJSON.String), &task.Capabilities)
	}
	if impactLevel.Valid {
		task.ImpactLevel = impactLevel.String
	}
	if estimatedCost.Valid {
		task.EstimatedCost = estimatedCost.Float64
	}

	return task, nil
}
