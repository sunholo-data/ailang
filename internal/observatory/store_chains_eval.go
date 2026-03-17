// Package observatory provides eval assessment operations for execution chains.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

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
