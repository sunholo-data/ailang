// Package observatory provides chain query helpers, stats, and journey operations.
package observatory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

// agentActionLabel maps agent IDs to human-readable action descriptions.
var agentActionLabel = map[string]string{
	"design-doc-creator": "Created design doc",
	"sprint-planner":     "Planned sprint",
	"sprint-executor":    "Implemented changes",
	"eval-standard":      "Ran evaluations",
	"eval-runner":        "Ran evaluations",
	"eval-agent":         "Ran agent evaluations",
}

// GetChainJourney computes a narrative journey for a chain from its stages.
func (s *Store) GetChainJourney(ctx context.Context, chainID string) (*JourneyResponse, error) {
	stages, err := s.GetChainStages(ctx, chainID, ChainReadOptions{IncludeStages: true})
	if err != nil {
		return nil, err
	}
	if len(stages) == 0 {
		return &JourneyResponse{ChainID: chainID}, nil
	}

	resp := &JourneyResponse{
		ChainID: chainID,
		Steps:   make([]JourneyStep, 0, len(stages)),
	}

	var summaryParts []string

	for _, stage := range stages {
		action, ok := agentActionLabel[stage.AgentID]
		if !ok {
			action = stage.AgentID
		}

		step := JourneyStep{
			StageNumber:    stage.StageNumber,
			AgentID:        stage.AgentID,
			Action:         action,
			Status:         string(stage.Status),
			ApprovalStatus: string(stage.ApprovalStatus),
			Iteration:      stage.Iteration,
			Cost:           stage.Cost,
			DurationMs:     stage.DurationMs,
		}

		if stage.HumanFeedback != "" {
			fb := stage.HumanFeedback
			if len(fb) > 120 {
				fb = fb[:120] + "..."
			}
			step.Feedback = fb
		}
		if stage.ErrorMessage != "" {
			em := stage.ErrorMessage
			if idx := strings.Index(em, "\n"); idx > 0 {
				em = em[:idx]
			}
			if len(em) > 120 {
				em = em[:120] + "..."
			}
			step.ErrorExcerpt = em
		}

		resp.Steps = append(resp.Steps, step)

		// Build summary fragment
		fragment := action
		switch stage.Status {
		case "completed":
			if stage.ApprovalStatus == "approved" {
				fragment += " (approved)"
			}
		case "failed":
			fragment += " (failed)"
		case "running":
			fragment += " (in progress)"
		case "awaiting_approval":
			fragment += " (awaiting approval)"
		}
		summaryParts = append(summaryParts, fragment)
	}

	resp.Summary = strings.Join(summaryParts, " -> ")
	return resp, nil
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
