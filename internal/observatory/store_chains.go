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
