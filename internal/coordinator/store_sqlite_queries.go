package coordinator

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

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
		       worktree_id, worktree_path, base_branch, base_commit, workspace, github_issue, github_repo, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration, chain_id, stage_id,
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
		       worktree_id, worktree_path, base_branch, base_commit, workspace, github_issue, github_repo, stage, design_doc_path, sprint_plan_path,
		       session_id, iteration, chain_id, stage_id,
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
	var provider, agentID, worktreeID, worktreePath, baseBranch, baseCommit, workspace, errStr, output, threadID, parentTaskID, stage sql.NullString
	var designDocPath, sprintPlanPath sql.NullString
	var sessionID sql.NullString
	var iteration sql.NullInt64
	var chainID, stageID sql.NullString // M-CHAINS-SIMPLIFY
	var githubIssue sql.NullInt64
	var githubRepo sql.NullString
	var capsJSON, impactLevel sql.NullString
	var estimatedCost sql.NullFloat64

	err := row.Scan(
		&task.ID, &messageID, &threadID, &parentTaskID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider, &agentID,
		&worktreeID, &worktreePath, &baseBranch, &baseCommit, &workspace, &githubIssue, &githubRepo, &stage, &designDocPath, &sprintPlanPath,
		&sessionID, &iteration, &chainID, &stageID,
		&task.CreatedAt, &startedAt, &completedAt,
		&durationNs, &errStr, &output, &task.Cost, &task.TokensUsed,
		&capsJSON, &impactLevel, &estimatedCost,
	)
	if err != nil {
		return nil, err
	}

	return s.populateTaskFields(task, startedAt, completedAt, durationNs, messageID,
		provider, agentID, worktreeID, worktreePath, baseBranch, baseCommit, workspace,
		errStr, output, threadID, parentTaskID, githubIssue, githubRepo, stage,
		designDocPath, sprintPlanPath, sessionID, iteration, chainID, stageID,
		capsJSON, impactLevel, estimatedCost), nil
}

// Helper to scan a task from rows
func (s *SQLiteStore) scanTaskFromRows(rows *sql.Rows) (*TaskRecord, error) {
	task := &TaskRecord{}
	var startedAt, completedAt sql.NullTime
	var durationNs sql.NullInt64
	var messageID sql.NullString
	var provider, agentID, worktreeID, worktreePath, baseBranch, baseCommit, workspace, errStr, output, threadID, parentTaskID, stage sql.NullString
	var designDocPath, sprintPlanPath sql.NullString
	var sessionID sql.NullString
	var iteration sql.NullInt64
	var chainID, stageID sql.NullString // M-CHAINS-SIMPLIFY
	var githubIssue sql.NullInt64
	var githubRepo sql.NullString
	var capsJSON, impactLevel sql.NullString
	var estimatedCost sql.NullFloat64

	err := rows.Scan(
		&task.ID, &messageID, &threadID, &parentTaskID, &task.Title, &task.Content,
		&task.Type, &task.Priority, &task.Status, &provider, &agentID,
		&worktreeID, &worktreePath, &baseBranch, &baseCommit, &workspace, &githubIssue, &githubRepo, &stage, &designDocPath, &sprintPlanPath,
		&sessionID, &iteration, &chainID, &stageID,
		&task.CreatedAt, &startedAt, &completedAt,
		&durationNs, &errStr, &output, &task.Cost, &task.TokensUsed,
		&capsJSON, &impactLevel, &estimatedCost,
	)
	if err != nil {
		return nil, err
	}

	return s.populateTaskFields(task, startedAt, completedAt, durationNs, messageID,
		provider, agentID, worktreeID, worktreePath, baseBranch, baseCommit, workspace,
		errStr, output, threadID, parentTaskID, githubIssue, githubRepo, stage,
		designDocPath, sprintPlanPath, sessionID, iteration, chainID, stageID,
		capsJSON, impactLevel, estimatedCost), nil
}

// populateTaskFields populates nullable fields on a TaskRecord from scanned values.
// Shared by scanTask and scanTaskFromRows to eliminate duplication.
func (s *SQLiteStore) populateTaskFields(
	task *TaskRecord,
	startedAt, completedAt sql.NullTime,
	durationNs sql.NullInt64,
	messageID sql.NullString,
	provider, agentID, worktreeID, worktreePath, baseBranch, baseCommit, workspace sql.NullString,
	errStr, output sql.NullString,
	threadID, parentTaskID sql.NullString,
	githubIssue sql.NullInt64,
	githubRepo sql.NullString,
	stage sql.NullString,
	designDocPath, sprintPlanPath sql.NullString,
	sessionID sql.NullString,
	iteration sql.NullInt64,
	chainID, stageID sql.NullString,
	capsJSON, impactLevel sql.NullString,
	estimatedCost sql.NullFloat64,
) *TaskRecord {
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
	if baseBranch.Valid {
		task.BaseBranch = baseBranch.String
	}
	if baseCommit.Valid {
		task.BaseCommit = baseCommit.String
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
	if githubRepo.Valid {
		task.GithubRepo = githubRepo.String
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
	// Execution chain fields (M-CHAINS-SIMPLIFY)
	if chainID.Valid {
		task.ChainID = chainID.String
	}
	if stageID.Valid {
		task.StageID = stageID.String
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

	return task
}
