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

	// SQLite is single-writer; limit to 1 connection to serialize writes at
	// the Go pool level instead of contending on the SQLite file lock.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

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
		base_branch TEXT,
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
		github_repo TEXT,
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
		"ALTER TABLE tasks ADD COLUMN base_branch TEXT",
		"ALTER TABLE tasks ADD COLUMN base_commit TEXT",
		"ALTER TABLE tasks ADD COLUMN session_id TEXT",
		// M-COORD-GITHUB-AUTO-ROUTING: GitHub integration columns
		"ALTER TABLE tasks ADD COLUMN github_issue INTEGER",
		"ALTER TABLE tasks ADD COLUMN github_repo TEXT",
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
		// Handoff tracking - to detect missed handoffs on daemon startup
		"ALTER TABLE approval_requests ADD COLUMN handoffs_triggered INTEGER DEFAULT 0",
		// Execution chain tracking (M-CHAINS-SIMPLIFY)
		"ALTER TABLE tasks ADD COLUMN chain_id TEXT",
		"ALTER TABLE tasks ADD COLUMN stage_id TEXT",
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
		                   agent_id, capabilities_json, impact_level, estimated_cost, github_issue, github_repo,
		                   chain_id, stage_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.MessageID, task.ThreadID, task.ParentTaskID, task.Title, task.Content,
		task.Type, task.Priority, task.Status, task.Workspace,
		task.AgentID, string(capsJSON), task.ImpactLevel, task.EstimatedCost,
		task.GithubIssue, task.GithubRepo,
		task.ChainID, task.StageID, task.CreatedAt,
	)
	return err
}

// GetTask retrieves a task by ID
func (s *SQLiteStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	query := `
		SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		       worktree_id, worktree_path, base_branch, base_commit, workspace, github_issue, github_repo, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration, chain_id, stage_id,
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
		       worktree_id, worktree_path, base_branch, base_commit, workspace, github_issue, github_repo, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration, chain_id, stage_id,
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

// GetCostByProvider returns total cost per provider for budget tracking.
func (s *SQLiteStore) GetCostByProvider() (map[string]float64, error) {
	result := make(map[string]float64)

	rows, err := s.db.Query(`
		SELECT
			COALESCE(provider, 'unknown') as provider,
			COALESCE(SUM(cost), 0) as total_cost
		FROM tasks
		WHERE provider IS NOT NULL
		GROUP BY provider
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var cost float64
		if err := rows.Scan(&provider, &cost); err != nil {
			return nil, err
		}
		result[provider] = cost
	}

	return result, nil
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
func (s *SQLiteStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *ExecuteResult) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET
			status = ?, completed_at = ?, worktree_path = ?, worktree_id = ?, base_branch = ?, base_commit = ?,
			duration_ns = ?, output = ?, cost = ?, tokens_used = ?, session_id = ?
		WHERE id = ?`,
		TaskStatusPendingApproval, now, worktreePath, worktreeBranch, baseBranch, baseCommit,
		int64(result.Duration), result.Output, result.Cost, result.TokensUsed, result.SessionID, id,
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
func (s *SQLiteStore) RequeueTask(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, started_at = NULL, completed_at = NULL WHERE id = ?",
		TaskStatusPending, id,
	)
	return err
}

// ResetTaskToPending resets a running task back to pending state.
func (s *SQLiteStore) ResetTaskToPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, started_at = NULL WHERE id = ?",
		TaskStatusPending, id,
	)
	return err
}

// FindDuplicateTask finds a similar task by fingerprint
func (s *SQLiteStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, message_id, thread_id, parent_task_id, title, content, type, priority, status, provider, agent_id,
		        worktree_id, worktree_path, base_branch, base_commit, workspace, github_issue, github_repo, stage, design_doc_path, sprint_plan_path,
		        session_id, iteration, chain_id, stage_id,
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

// UpdateTaskChainInfo updates the chain_id and stage_id for a task (M-CHAINS-SIMPLIFY)
func (s *SQLiteStore) UpdateTaskChainInfo(ctx context.Context, id, chainID, stageID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET chain_id = ?, stage_id = ? WHERE id = ?",
		chainID, stageID, id,
	)
	return err
}

// GetTaskAgentInfo returns agent info for a task (for cross-db correlation in Control Plane)
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
	inbox = agentID
	return agentID, inbox, title, nil
}
