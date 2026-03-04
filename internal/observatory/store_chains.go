// Package observatory provides execution chain CRUD operations.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateChain creates a new execution chain.
// Returns the chain with its generated ID populated.
func (s *Store) CreateChain(ctx context.Context, req *ChainCreateRequest) (*ExecutionChain, error) {
	if req.SourceType == "" {
		return nil, fmt.Errorf("source_type is required")
	}

	chain := &ExecutionChain{
		ID:                uuid.New().String(),
		SourceType:        req.SourceType,
		SourceRef:         req.SourceRef,
		GitHubRepo:        req.GitHubRepo,
		GitHubIssueNumber: req.GitHubIssueNumber,
		Status:            ChainStatusActive,
		CurrentStage:      0,
		WorkspaceID:       req.WorkspaceID,
		WorkspacePath:     req.WorkspacePath,
		CreatedAt:         time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO execution_chains (
			id, source_type, source_ref, github_repo, github_issue_number,
			status, current_stage, workspace_id, workspace_path, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		chain.ID, chain.SourceType, chain.SourceRef, chain.GitHubRepo, chain.GitHubIssueNumber,
		chain.Status, chain.CurrentStage, chain.WorkspaceID, chain.WorkspacePath, chain.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}

	return chain, nil
}

// GetChain retrieves a chain by ID with optional related data.
func (s *Store) GetChain(ctx context.Context, id string, opts ChainReadOptions) (*ExecutionChain, error) {
	if id == "" {
		return nil, fmt.Errorf("chain id is required")
	}

	chain := &ExecutionChain{}
	var updatedAt, completedAt sql.NullTime

	// Support short ID prefix matching (like git)
	whereClause := "id = ?"
	if len(id) < 36 {
		whereClause = "id LIKE ?"
		id = id + "%"
	}

	err := s.db.QueryRowContext(ctx, `
		SELECT id, source_type, source_ref, github_repo, github_issue_number,
		       status, current_stage, workspace_id, workspace_path,
		       created_at, updated_at, completed_at,
		       total_cost, total_tokens, total_turns, stages_completed
		FROM execution_chains WHERE `+whereClause+`
	`, id).Scan(
		&chain.ID, &chain.SourceType, &chain.SourceRef, &chain.GitHubRepo, &chain.GitHubIssueNumber,
		&chain.Status, &chain.CurrentStage, &chain.WorkspaceID, &chain.WorkspacePath,
		&chain.CreatedAt, &updatedAt, &completedAt,
		&chain.TotalCost, &chain.TotalTokens, &chain.TotalTurns, &chain.StagesCompleted,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chain: %w", err)
	}

	if updatedAt.Valid {
		chain.UpdatedAt = &updatedAt.Time
	}
	if completedAt.Valid {
		chain.CompletedAt = &completedAt.Time
	}

	// Load stages if requested
	if opts.IncludeStages {
		stages, err := s.GetChainStages(ctx, id, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to load stages: %w", err)
		}
		chain.Stages = stages
	}

	return chain, nil
}

// ListChains returns chains matching the given options.
func (s *Store) ListChains(ctx context.Context, opts ChainListOptions) ([]*ChainSummary, error) {
	var conditions []string
	var args []interface{}

	if opts.Status != "" {
		conditions = append(conditions, "c.status = ?")
		args = append(args, opts.Status)
	}
	if opts.SourceType != "" {
		conditions = append(conditions, "c.source_type = ?")
		args = append(args, opts.SourceType)
	}
	if opts.WorkspaceID != "" {
		conditions = append(conditions, "c.workspace_id = ?")
		args = append(args, opts.WorkspaceID)
	}
	if opts.GitHubRepo != "" {
		conditions = append(conditions, "c.github_repo = ?")
		args = append(args, opts.GitHubRepo)
	}
	if opts.AgentID != "" {
		conditions = append(conditions, "s.agent_id = ?")
		args = append(args, opts.AgentID)
	}
	if opts.CreatedAfter != nil {
		conditions = append(conditions, "c.created_at > ?")
		args = append(args, *opts.CreatedAfter)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.source_type, c.source_ref, c.github_repo, c.github_issue_number,
		       c.status, c.current_stage, c.total_cost, c.total_tokens, c.total_turns,
		       c.stages_completed, c.created_at, c.completed_at,
		       COUNT(s.id) as stage_count,
		       COALESCE(MAX(s.stage_number), 0) as max_stage,
		       GROUP_CONCAT(s.agent_id, ' -> ') as agent_flow
		FROM execution_chains c
		LEFT JOIN chain_stages s ON s.chain_id = c.id
		%s
		GROUP BY c.id
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list chains: %w", err)
	}
	defer rows.Close()

	var chains []*ChainSummary
	for rows.Next() {
		chain := &ChainSummary{}
		var completedAt sql.NullTime
		var agentFlow sql.NullString

		err := rows.Scan(
			&chain.ID, &chain.SourceType, &chain.SourceRef, &chain.GitHubRepo, &chain.GitHubIssueNumber,
			&chain.Status, &chain.CurrentStage, &chain.TotalCost, &chain.TotalTokens, &chain.TotalTurns,
			&chain.StagesCompleted, &chain.CreatedAt, &completedAt,
			&chain.StageCount, &chain.MaxStage, &agentFlow,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chain row: %w", err)
		}

		if completedAt.Valid {
			chain.CompletedAt = &completedAt.Time
		}
		if agentFlow.Valid {
			chain.AgentFlow = agentFlow.String
		}

		chains = append(chains, chain)
	}

	return chains, nil
}

// UpdateChainStatus updates the status of a chain.
func (s *Store) UpdateChainStatus(ctx context.Context, id string, status ChainStatus) error {
	if id == "" {
		return fmt.Errorf("chain id is required")
	}

	now := time.Now()
	var completedAt *time.Time
	if status == ChainStatusCompleted || status == ChainStatusFailed {
		completedAt = &now
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE execution_chains
		SET status = ?, updated_at = ?, completed_at = COALESCE(?, completed_at)
		WHERE id = ?
	`, status, now, completedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update chain status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("chain not found: %s", id)
	}

	return nil
}

// UpdateChainMetrics updates the denormalized metrics on a chain.
func (s *Store) UpdateChainMetrics(ctx context.Context, id string, cost float64, tokens, turns int) error {
	if id == "" {
		return fmt.Errorf("chain id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE execution_chains
		SET total_cost = total_cost + ?,
		    total_tokens = total_tokens + ?,
		    total_turns = total_turns + ?,
		    updated_at = ?
		WHERE id = ?
	`, cost, tokens, turns, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update chain metrics: %w", err)
	}

	return nil
}

// DeleteChain removes a chain and all its stages (CASCADE).
func (s *Store) DeleteChain(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("chain id is required")
	}

	result, err := s.db.ExecContext(ctx, "DELETE FROM execution_chains WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete chain: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("chain not found: %s", id)
	}

	return nil
}

// ===== Chain Stage Operations =====

// CreateStage creates a new stage within a chain.
// Returns the stage with its generated ID populated.
func (s *Store) CreateStage(ctx context.Context, req *StageCreateRequest) (*ChainStage, error) {
	if req.ChainID == "" {
		return nil, fmt.Errorf("chain_id is required")
	}
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	iteration := req.Iteration
	if iteration == 0 {
		iteration = 1
	}

	stage := &ChainStage{
		ID:        uuid.New().String(),
		ChainID:   req.ChainID,
		AgentID:   req.AgentID,
		Provider:  req.Provider,
		MessageID: req.MessageID,
		TaskID:    req.TaskID,
		HandoffTo: req.HandoffTo,
		Iteration: iteration,
		Status:    StageStatusPending,
	}

	// Atomic INSERT with computed stage_number to avoid TOCTOU race
	// under concurrent stage creation (e.g., parallel eval benchmarks).
	// Retry up to 5 times on UNIQUE constraint violations.
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO chain_stages (
				id, chain_id, stage_number, agent_id, provider,
				message_id, task_id, handoff_to, iteration, status
			) VALUES (
				?, ?,
				(SELECT COALESCE(MAX(stage_number), 0) + 1 FROM chain_stages WHERE chain_id = ?),
				?, ?, ?, ?, ?, ?, ?
			)
			RETURNING stage_number
		`,
			stage.ID, stage.ChainID, stage.ChainID,
			stage.AgentID, stage.Provider,
			stage.MessageID, stage.TaskID, stage.HandoffTo, stage.Iteration, stage.Status,
		).Scan(&stage.StageNumber)
		if err == nil {
			break
		}
		// Retry on UNIQUE constraint violation (concurrent inserts)
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			stage.ID = uuid.New().String() // New ID for retry
			continue
		}
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create stage after retries: %w", err)
	}

	// Update chain's current stage
	_, err = s.db.ExecContext(ctx, `
		UPDATE execution_chains
		SET current_stage = ?, updated_at = ?
		WHERE id = ?
	`, stage.StageNumber, time.Now(), req.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to update chain current_stage: %w", err)
	}

	return stage, nil
}

// GetChainStages returns all stages for a chain.
func (s *Store) GetChainStages(ctx context.Context, chainID string, opts ChainReadOptions) ([]*ChainStage, error) {
	if chainID == "" {
		return nil, fmt.Errorf("chain_id is required")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, chain_id, stage_number, agent_id, provider,
		       message_id, task_id, session_id,
		       status, approval_status, approval_type,
		       handoff_to, iteration, human_feedback,
		       started_at, completed_at,
		       cost, tokens_in, tokens_out, turns, tool_calls, duration_ms,
		       error_message, error_count,
		       eval_assessment
		FROM chain_stages
		WHERE chain_id = ?
		ORDER BY stage_number ASC
	`, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to query stages: %w", err)
	}
	defer rows.Close()

	var stages []*ChainStage
	for rows.Next() {
		stage := &ChainStage{}
		var startedAt, completedAt sql.NullTime
		var messageID, taskID, sessionID, handoffTo sql.NullString
		var approvalStatus, approvalType, humanFeedback, errorMessage sql.NullString
		var provider sql.NullString
		var evalAssessmentJSON sql.NullString

		err := rows.Scan(
			&stage.ID, &stage.ChainID, &stage.StageNumber, &stage.AgentID, &provider,
			&messageID, &taskID, &sessionID,
			&stage.Status, &approvalStatus, &approvalType,
			&handoffTo, &stage.Iteration, &humanFeedback,
			&startedAt, &completedAt,
			&stage.Cost, &stage.TokensIn, &stage.TokensOut, &stage.Turns, &stage.ToolCalls, &stage.DurationMs,
			&errorMessage, &stage.ErrorCount,
			&evalAssessmentJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stage row: %w", err)
		}

		if provider.Valid {
			stage.Provider = Provider(provider.String)
		}
		if messageID.Valid {
			stage.MessageID = messageID.String
		}
		if taskID.Valid {
			stage.TaskID = taskID.String
		}
		if sessionID.Valid {
			stage.SessionID = sessionID.String
		}
		if handoffTo.Valid {
			stage.HandoffTo = handoffTo.String
		}
		if startedAt.Valid {
			stage.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			stage.CompletedAt = &completedAt.Time
		}
		if approvalStatus.Valid {
			stage.ApprovalStatus = ApprovalStatus(approvalStatus.String)
		}
		if approvalType.Valid {
			stage.ApprovalType = ApprovalType(approvalType.String)
		}
		if humanFeedback.Valid {
			stage.HumanFeedback = humanFeedback.String
		}
		if errorMessage.Valid {
			stage.ErrorMessage = errorMessage.String
		}
		if evalAssessmentJSON.Valid && evalAssessmentJSON.String != "" {
			var ea EvalAssessment
			if jsonErr := json.Unmarshal([]byte(evalAssessmentJSON.String), &ea); jsonErr == nil {
				stage.EvalAssessment = &ea
			}
		}

		// Load session data if requested
		if opts.IncludeSessions && stage.SessionID != "" {
			session, err := s.GetSession(ctx, stage.SessionID)
			if err == nil && session != nil {
				stage.Session = session
			}
		}

		// Load spans if requested
		if opts.IncludeSpans {
			spans, err := s.GetSpansByStageID(ctx, stage.ID)
			if err == nil {
				stage.Spans = spans
			}
		}

		stages = append(stages, stage)
	}

	return stages, nil
}

// GetStage retrieves a single stage by ID.
func (s *Store) GetStage(ctx context.Context, id string) (*ChainStage, error) {
	if id == "" {
		return nil, fmt.Errorf("stage id is required")
	}

	stage := &ChainStage{}
	var startedAt, completedAt sql.NullTime
	var messageID, taskID, sessionID, handoffTo sql.NullString
	var approvalStatus, approvalType, humanFeedback, errorMessage sql.NullString
	var provider sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, chain_id, stage_number, agent_id, provider,
		       message_id, task_id, session_id,
		       status, approval_status, approval_type,
		       handoff_to, iteration, human_feedback,
		       started_at, completed_at,
		       cost, tokens_in, tokens_out, turns, tool_calls, duration_ms,
		       error_message, error_count
		FROM chain_stages WHERE id = ?
	`, id).Scan(
		&stage.ID, &stage.ChainID, &stage.StageNumber, &stage.AgentID, &provider,
		&messageID, &taskID, &sessionID,
		&stage.Status, &approvalStatus, &approvalType,
		&handoffTo, &stage.Iteration, &humanFeedback,
		&startedAt, &completedAt,
		&stage.Cost, &stage.TokensIn, &stage.TokensOut, &stage.Turns, &stage.ToolCalls, &stage.DurationMs,
		&errorMessage, &stage.ErrorCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stage: %w", err)
	}

	if provider.Valid {
		stage.Provider = Provider(provider.String)
	}
	if messageID.Valid {
		stage.MessageID = messageID.String
	}
	if taskID.Valid {
		stage.TaskID = taskID.String
	}
	if sessionID.Valid {
		stage.SessionID = sessionID.String
	}
	if handoffTo.Valid {
		stage.HandoffTo = handoffTo.String
	}
	if startedAt.Valid {
		stage.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		stage.CompletedAt = &completedAt.Time
	}
	if approvalStatus.Valid {
		stage.ApprovalStatus = ApprovalStatus(approvalStatus.String)
	}
	if approvalType.Valid {
		stage.ApprovalType = ApprovalType(approvalType.String)
	}
	if humanFeedback.Valid {
		stage.HumanFeedback = humanFeedback.String
	}
	if errorMessage.Valid {
		stage.ErrorMessage = errorMessage.String
	}

	return stage, nil
}

// UpdateStageStatus updates a stage's execution status.
func (s *Store) UpdateStageStatus(ctx context.Context, id string, status ChainStageStatus) error {
	if id == "" {
		return fmt.Errorf("stage id is required")
	}

	now := time.Now()
	var startedAt, completedAt *time.Time

	if status == StageStatusRunning {
		startedAt = &now
	}
	if status == StageStatusCompleted || status == StageStatusFailed {
		completedAt = &now
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET status = ?,
		    started_at = COALESCE(?, started_at),
		    completed_at = COALESCE(?, completed_at)
		WHERE id = ?
	`, status, startedAt, completedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update stage status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("stage not found: %s", id)
	}

	// If stage completed, increment chain's stages_completed counter
	if status == StageStatusCompleted {
		stage, _ := s.GetStage(ctx, id)
		if stage != nil {
			_, _ = s.db.ExecContext(ctx, `
				UPDATE execution_chains
				SET stages_completed = stages_completed + 1, updated_at = ?
				WHERE id = ?
			`, now, stage.ChainID)
		}
	}

	return nil
}

// UpdateStageSession links a session to a stage.
func (s *Store) UpdateStageSession(ctx context.Context, stageID, sessionID string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages SET session_id = ? WHERE id = ?
	`, sessionID, stageID)
	if err != nil {
		return fmt.Errorf("failed to update stage session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("stage not found: %s", stageID)
	}

	return nil
}

// UpdateStageApproval updates the approval status of a stage.
func (s *Store) UpdateStageApproval(ctx context.Context, stageID string, status ApprovalStatus, approvalType ApprovalType, feedback string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	// Determine stage status based on approval
	var stageStatus ChainStageStatus
	switch status {
	case ApprovalStatusPending:
		stageStatus = StageStatusAwaitingApproval
	case ApprovalStatusApproved:
		stageStatus = StageStatusCompleted
	case ApprovalStatusRejected:
		stageStatus = StageStatusFailed
	}

	now := time.Now()
	var completedAt *time.Time
	if status == ApprovalStatusApproved || status == ApprovalStatusRejected {
		completedAt = &now
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET approval_status = ?, approval_type = ?, human_feedback = ?,
		    status = ?, completed_at = COALESCE(?, completed_at)
		WHERE id = ?
	`, status, approvalType, feedback, stageStatus, completedAt, stageID)
	if err != nil {
		return fmt.Errorf("failed to update stage approval: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("stage not found: %s", stageID)
	}

	return nil
}

// UpdateStageMetrics updates the denormalized metrics on a stage.
func (s *Store) UpdateStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET cost = cost + ?,
		    tokens_in = tokens_in + ?,
		    tokens_out = tokens_out + ?,
		    turns = turns + ?,
		    tool_calls = tool_calls + ?,
		    duration_ms = duration_ms + ?
		WHERE id = ?
	`, cost, tokensIn, tokensOut, turns, toolCalls, durationMs, stageID)
	if err != nil {
		return fmt.Errorf("failed to update stage metrics: %w", err)
	}

	return nil
}

// UpdateStageError records an error on a stage.
func (s *Store) UpdateStageError(ctx context.Context, stageID, errorMessage string) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE chain_stages
		SET error_message = ?, error_count = error_count + 1, status = ?
		WHERE id = ?
	`, errorMessage, StageStatusFailed, stageID)
	if err != nil {
		return fmt.Errorf("failed to update stage error: %w", err)
	}

	return nil
}

// ===== Eval Assessment Methods (M-EVAL-CHAINS) =====

// UpdateStageEvalAssessment stores eval assessment data as JSON on a chain stage.
func (s *Store) UpdateStageEvalAssessment(ctx context.Context, stageID string, assessment *EvalAssessment) error {
	if stageID == "" {
		return fmt.Errorf("stage_id is required")
	}
	if assessment == nil {
		return fmt.Errorf("assessment is required")
	}

	data, err := json.Marshal(assessment)
	if err != nil {
		return fmt.Errorf("failed to marshal eval assessment: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE chain_stages SET eval_assessment = ? WHERE id = ?
	`, string(data), stageID)
	if err != nil {
		return fmt.Errorf("failed to update eval assessment: %w", err)
	}

	return nil
}

// GetStageEvalAssessment reads eval assessment data from a chain stage.
// Returns nil, nil if the stage exists but has no eval assessment.
func (s *Store) GetStageEvalAssessment(ctx context.Context, stageID string) (*EvalAssessment, error) {
	if stageID == "" {
		return nil, fmt.Errorf("stage_id is required")
	}

	var rawJSON sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT eval_assessment FROM chain_stages WHERE id = ?
	`, stageID).Scan(&rawJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stage not found: %s", stageID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query eval assessment: %w", err)
	}
	if !rawJSON.Valid || rawJSON.String == "" {
		return nil, nil
	}

	var assessment EvalAssessment
	if err := json.Unmarshal([]byte(rawJSON.String), &assessment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal eval assessment: %w", err)
	}

	return &assessment, nil
}

// QueryEvalResults returns chain stages that have eval assessments, with optional filters.
// Uses json_extract() for filtering on assessment fields.
func (s *Store) QueryEvalResults(ctx context.Context, opts EvalQueryOptions) ([]*ChainStage, error) {
	query := `
		SELECT id, chain_id, stage_number, agent_id, provider,
		       message_id, task_id, session_id,
		       status, approval_status, approval_type,
		       handoff_to, iteration, human_feedback,
		       started_at, completed_at,
		       cost, tokens_in, tokens_out, turns, tool_calls, duration_ms,
		       error_message, error_count,
		       eval_assessment
		FROM chain_stages
		WHERE eval_assessment IS NOT NULL
	`
	var args []interface{}

	if opts.ChainID != "" {
		// Support short ID prefix matching (like git)
		if len(opts.ChainID) < 36 {
			query += " AND chain_id LIKE ?"
			args = append(args, opts.ChainID+"%")
		} else {
			query += " AND chain_id = ?"
			args = append(args, opts.ChainID)
		}
	}
	if opts.Model != "" {
		query += " AND json_extract(eval_assessment, '$.model') = ?"
		args = append(args, opts.Model)
	}
	if opts.Language != "" {
		query += " AND json_extract(eval_assessment, '$.language') = ?"
		args = append(args, opts.Language)
	}
	if opts.BenchmarkID != "" {
		query += " AND json_extract(eval_assessment, '$.benchmark_id') = ?"
		args = append(args, opts.BenchmarkID)
	}
	if opts.Condition != "" {
		query += " AND json_extract(eval_assessment, '$.condition') = ?"
		args = append(args, opts.Condition)
	}
	if opts.EvalMode != "" {
		query += " AND json_extract(eval_assessment, '$.eval_mode') = ?"
		args = append(args, opts.EvalMode)
	}
	if opts.SuccessOnly {
		query += " AND json_extract(eval_assessment, '$.stdout_ok') = 1"
	}
	if opts.FailureOnly {
		query += " AND json_extract(eval_assessment, '$.stdout_ok') = 0"
	}

	query += " ORDER BY stage_number ASC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query eval results: %w", err)
	}
	defer rows.Close()

	var stages []*ChainStage
	for rows.Next() {
		stage, err := s.scanStageWithAssessment(rows)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}

	return stages, nil
}

// scanStageWithAssessment scans a chain stage row including the eval_assessment column.
func (s *Store) scanStageWithAssessment(rows *sql.Rows) (*ChainStage, error) {
	stage := &ChainStage{}
	var startedAt, completedAt sql.NullTime
	var messageID, taskID, sessionID, handoffTo sql.NullString
	var approvalStatus, approvalType, humanFeedback, errorMessage sql.NullString
	var provider sql.NullString
	var evalAssessmentJSON sql.NullString

	err := rows.Scan(
		&stage.ID, &stage.ChainID, &stage.StageNumber, &stage.AgentID, &provider,
		&messageID, &taskID, &sessionID,
		&stage.Status, &approvalStatus, &approvalType,
		&handoffTo, &stage.Iteration, &humanFeedback,
		&startedAt, &completedAt,
		&stage.Cost, &stage.TokensIn, &stage.TokensOut, &stage.Turns, &stage.ToolCalls, &stage.DurationMs,
		&errorMessage, &stage.ErrorCount,
		&evalAssessmentJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan stage row: %w", err)
	}

	if provider.Valid {
		stage.Provider = Provider(provider.String)
	}
	if messageID.Valid {
		stage.MessageID = messageID.String
	}
	if taskID.Valid {
		stage.TaskID = taskID.String
	}
	if sessionID.Valid {
		stage.SessionID = sessionID.String
	}
	if handoffTo.Valid {
		stage.HandoffTo = handoffTo.String
	}
	if startedAt.Valid {
		stage.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		stage.CompletedAt = &completedAt.Time
	}
	if approvalStatus.Valid {
		stage.ApprovalStatus = ApprovalStatus(approvalStatus.String)
	}
	if approvalType.Valid {
		stage.ApprovalType = ApprovalType(approvalType.String)
	}
	if humanFeedback.Valid {
		stage.HumanFeedback = humanFeedback.String
	}
	if errorMessage.Valid {
		stage.ErrorMessage = errorMessage.String
	}
	if evalAssessmentJSON.Valid && evalAssessmentJSON.String != "" {
		var assessment EvalAssessment
		if err := json.Unmarshal([]byte(evalAssessmentJSON.String), &assessment); err == nil {
			stage.EvalAssessment = &assessment
		}
	}

	return stage, nil
}

// ListEvalChains returns chains with source_type = "eval_suite".
func (s *Store) ListEvalChains(ctx context.Context, limit int) ([]*ChainSummary, error) {
	return s.ListChains(ctx, ChainListOptions{
		SourceType: string(ChainSourceEvalSuite),
		Limit:      limit,
	})
}

// ===== Query Helpers =====

// GetChainByTaskID finds the chain containing a given task ID.
func (s *Store) GetChainByTaskID(ctx context.Context, taskID string) (*ExecutionChain, error) {
	if taskID == "" {
		return nil, nil
	}

	var chainID string
	err := s.db.QueryRowContext(ctx, `
		SELECT chain_id FROM chain_stages WHERE task_id = ? LIMIT 1
	`, taskID).Scan(&chainID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find chain by task: %w", err)
	}

	return s.GetChain(ctx, chainID, DefaultChainReadOptions())
}

// GetChainByMessageID finds the chain containing a given message ID.
func (s *Store) GetChainByMessageID(ctx context.Context, messageID string) (*ExecutionChain, error) {
	if messageID == "" {
		return nil, nil
	}

	var chainID string
	err := s.db.QueryRowContext(ctx, `
		SELECT chain_id FROM chain_stages WHERE message_id = ? LIMIT 1
	`, messageID).Scan(&chainID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find chain by message: %w", err)
	}

	return s.GetChain(ctx, chainID, DefaultChainReadOptions())
}

// GetChainByGitHubIssue finds the chain for a GitHub issue.
func (s *Store) GetChainByGitHubIssue(ctx context.Context, repo string, issueNumber int) (*ExecutionChain, error) {
	if repo == "" || issueNumber == 0 {
		return nil, nil
	}

	var chainID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM execution_chains
		WHERE github_repo = ? AND github_issue_number = ?
		ORDER BY created_at DESC LIMIT 1
	`, repo, issueNumber).Scan(&chainID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find chain by github issue: %w", err)
	}

	return s.GetChain(ctx, chainID, DefaultChainReadOptions())
}

// ListPendingApprovals returns all stages awaiting approval.
func (s *Store) ListPendingApprovals(ctx context.Context, limit int) ([]*PendingApprovalInfo, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.source_type, c.source_ref,
		       s.id, s.stage_number, s.agent_id, s.approval_status, s.approval_type,
		       s.task_id, s.session_id, s.cost, s.turns, s.started_at
		FROM execution_chains c
		JOIN chain_stages s ON s.chain_id = c.id
		WHERE s.approval_status = 'pending'
		ORDER BY s.started_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending approvals: %w", err)
	}
	defer rows.Close()

	var approvals []*PendingApprovalInfo
	for rows.Next() {
		info := &PendingApprovalInfo{}
		var stageCreated sql.NullTime
		var sourceRef, approvalStatus, approvalType, taskID, sessionID sql.NullString

		err := rows.Scan(
			&info.ChainID, &info.SourceType, &sourceRef,
			&info.StageID, &info.StageNumber, &info.AgentID, &approvalStatus, &approvalType,
			&taskID, &sessionID, &info.Cost, &info.Turns, &stageCreated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval row: %w", err)
		}

		if sourceRef.Valid {
			info.SourceRef = sourceRef.String
		}
		if approvalStatus.Valid {
			info.ApprovalStatus = approvalStatus.String
		}
		if approvalType.Valid {
			info.ApprovalType = ApprovalType(approvalType.String)
		}
		if taskID.Valid {
			info.TaskID = taskID.String
		}
		if sessionID.Valid {
			info.SessionID = sessionID.String
		}
		if stageCreated.Valid {
			info.StageCreated = stageCreated.Time
		}

		approvals = append(approvals, info)
	}

	return approvals, nil
}

// GetChainStats returns aggregate statistics about chains.
func (s *Store) GetChainStats(ctx context.Context) (*ChainStats, error) {
	stats := &ChainStats{}

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total_chains,
			SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active_chains,
			SUM(CASE WHEN status = 'pending_approval' THEN 1 ELSE 0 END) as pending_approvals,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_chains,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_chains,
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(AVG(stages_completed), 0) as avg_stages,
			COALESCE(AVG(
				CASE WHEN completed_at IS NOT NULL
				THEN (julianday(completed_at) - julianday(created_at)) * 86400000
				ELSE NULL END
			), 0) as avg_duration_ms
		FROM execution_chains
	`).Scan(
		&stats.TotalChains,
		&stats.ActiveChains,
		&stats.PendingApprovals,
		&stats.CompletedChains,
		&stats.FailedChains,
		&stats.TotalCost,
		&stats.TotalTokens,
		&stats.AverageStagesCount,
		&stats.AverageDurationMs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain stats: %w", err)
	}

	return stats, nil
}

// GetChainStatusCounts returns chain counts grouped by status in a single query.
// Replaces the pattern of fetching all chains and counting in Go (M-PERF-OBSERVATORY).
func (s *Store) GetChainStatusCounts(ctx context.Context, createdAfter *time.Time) (*ChainStatusCounts, error) {
	counts := &ChainStatusCounts{}

	var args []interface{}
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'pending_approval' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END),
			COALESCE(SUM(total_cost), 0),
			COALESCE(SUM(total_tokens), 0)
		FROM execution_chains
	`
	if createdAfter != nil {
		query += ` WHERE created_at > ?`
		args = append(args, *createdAfter)
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&counts.Total,
		&counts.Completed,
		&counts.Active,
		&counts.Pending,
		&counts.Failed,
		&counts.TotalCost,
		&counts.TotalTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain status counts: %w", err)
	}

	return counts, nil
}

// GetChainStatsByAgent returns per-agent aggregated stats in a single SQL query.
// Replaces the N+1 pattern of fetching all chains then querying stages per chain (M-PERF-OBSERVATORY).
func (s *Store) GetChainStatsByAgent(ctx context.Context, createdAfter *time.Time) ([]*AgentStatsResult, error) {
	var args []interface{}
	query := `
		SELECT cs.agent_id,
		       COUNT(cs.id) as stages,
		       SUM(CASE WHEN cs.status = 'completed' THEN 1 ELSE 0 END) as completed,
		       SUM(CASE WHEN cs.status = 'failed' THEN 1 ELSE 0 END) as failed,
		       COALESCE(SUM(cs.cost), 0) as total_cost,
		       COALESCE(SUM(cs.tokens_in), 0) as total_tokens_in,
		       COALESCE(SUM(cs.tokens_out), 0) as total_tokens_out
		FROM chain_stages cs
		JOIN execution_chains c ON cs.chain_id = c.id
	`
	if createdAfter != nil {
		query += ` WHERE c.created_at > ?`
		args = append(args, *createdAfter)
	}
	query += ` GROUP BY cs.agent_id ORDER BY total_cost DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain stats by agent: %w", err)
	}
	defer rows.Close()

	var results []*AgentStatsResult
	for rows.Next() {
		r := &AgentStatsResult{}
		if err := rows.Scan(&r.AgentID, &r.Stages, &r.Completed, &r.Failed, &r.TotalCost, &r.TokensIn, &r.TokensOut); err != nil {
			return nil, fmt.Errorf("failed to scan agent stats row: %w", err)
		}
		results = append(results, r)
	}

	return results, nil
}

// GetSpanLitesByStageID returns lightweight spans for a stage without the heavy attributes columns.
// This avoids reading the 3.9GB attributes data when only metadata is needed (M-PERF-OBSERVATORY).
func (s *Store) GetSpanLitesByStageID(ctx context.Context, stageID string, limit, offset int) (*SpanLitePage, error) {
	if stageID == "" {
		return &SpanLitePage{}, nil
	}
	if limit <= 0 {
		limit = 200
	}

	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM spans WHERE stage_id = ?`, stageID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count spans for stage: %w", err)
	}

	// Fetch spans without attributes columns
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trace_id, parent_span_id,
		       COALESCE(chain_id, ''), COALESCE(stage_id, ''),
		       name, kind, status, status_message,
		       start_time, end_time, duration_ms,
		       tokens_in, tokens_out, cost_usd,
		       model, provider
		FROM spans
		WHERE stage_id = ?
		ORDER BY start_time ASC
		LIMIT ? OFFSET ?
	`, stageID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query lite spans: %w", err)
	}
	defer rows.Close()

	var spans []*SpanLite
	for rows.Next() {
		sl := &SpanLite{}
		var parentSpanID, statusMessage, model, provider sql.NullString
		var endTime sql.NullTime

		err := rows.Scan(
			&sl.ID, &sl.TraceID, &parentSpanID,
			&sl.ChainID, &sl.StageID,
			&sl.Name, &sl.Kind, &sl.Status, &statusMessage,
			&sl.StartTime, &endTime, &sl.DurationMs,
			&sl.TokensIn, &sl.TokensOut, &sl.CostUSD,
			&model, &provider,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lite span row: %w", err)
		}

		if parentSpanID.Valid {
			sl.ParentSpanID = parentSpanID.String
		}
		if statusMessage.Valid {
			sl.StatusMessage = statusMessage.String
		}
		if endTime.Valid {
			sl.EndTime = endTime.Time
		}
		if model.Valid {
			sl.Model = model.String
		}
		if provider.Valid {
			sl.Provider = provider.String
		}

		spans = append(spans, sl)
	}

	return &SpanLitePage{
		Spans:  spans,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetSpansByStageID returns spans linked to a stage via stage_id column.
func (s *Store) GetSpansByStageID(ctx context.Context, stageID string) ([]*Span, error) {
	if stageID == "" {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trace_id, parent_span_id, task_id, agent_assignment_id,
		       name, kind, status, status_message, start_time, end_time,
		       duration_ms, tokens_in, tokens_out,
		       COALESCE(cache_read_tokens, 0), COALESCE(cache_creation_tokens, 0),
		       cost_usd, model, provider,
		       attributes, resource_attributes, created_at
		FROM spans
		WHERE stage_id = ?
		ORDER BY start_time ASC
	`, stageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query spans by stage: %w", err)
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		span := &Span{}
		var parentSpanID, taskID, agentAssignmentID, statusMessage, model sql.NullString
		var provider sql.NullString
		var endTime sql.NullTime
		var cacheReadTokens, cacheCreationTokens sql.NullInt64
		var attributesJSON, resourceAttributesJSON string

		err := rows.Scan(
			&span.ID, &span.TraceID, &parentSpanID, &taskID, &agentAssignmentID,
			&span.Name, &span.Kind, &span.Status, &statusMessage, &span.StartTime, &endTime,
			&span.DurationMs, &span.TokensIn, &span.TokensOut,
			&cacheReadTokens, &cacheCreationTokens,
			&span.CostUSD, &model, &provider,
			&attributesJSON, &resourceAttributesJSON, &span.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan span row: %w", err)
		}

		if endTime.Valid {
			span.EndTime = &endTime.Time
		}
		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if taskID.Valid {
			span.TaskID = taskID.String
		}
		if agentAssignmentID.Valid {
			span.AgentAssignmentID = agentAssignmentID.String
		}
		if statusMessage.Valid {
			span.StatusMessage = statusMessage.String
		}
		if model.Valid {
			span.Model = model.String
		}
		if provider.Valid {
			span.Provider = Provider(provider.String)
		}
		if cacheReadTokens.Valid {
			span.CacheReadTokens = cacheReadTokens.Int64
		}
		if cacheCreationTokens.Valid {
			span.CacheCreationTokens = cacheCreationTokens.Int64
		}

		// Parse JSON attributes
		if attributesJSON != "" {
			_ = json.Unmarshal([]byte(attributesJSON), &span.Attributes)
		}
		if resourceAttributesJSON != "" {
			_ = json.Unmarshal([]byte(resourceAttributesJSON), &span.ResourceAttributes)
		}

		spans = append(spans, span)
	}

	return spans, nil
}

// LinkSpanToChain updates a span's chain_id and stage_id.
// Called during OTEL ingest when AILANG_CHAIN_ID/AILANG_STAGE_ID env vars are present.
func (s *Store) LinkSpanToChain(ctx context.Context, spanID, chainID, stageID string) error {
	if spanID == "" {
		return fmt.Errorf("span_id is required")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE spans SET chain_id = ?, stage_id = ? WHERE id = ?
	`, chainID, stageID, spanID)
	if err != nil {
		return fmt.Errorf("failed to link span to chain: %w", err)
	}

	return nil
}
