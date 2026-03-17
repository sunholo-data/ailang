package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	obs "github.com/sunholo/ailang/internal/observatory"
)

// --- Chain operations ---

func (s *ObservatoryStore) CreateChain(ctx context.Context, req *obs.ChainCreateRequest) (*obs.ExecutionChain, error) {
	now := time.Now()
	chain := &obs.ExecutionChain{
		ID:                uuid.New().String(),
		SourceType:        req.SourceType,
		SourceRef:         req.SourceRef,
		GitHubRepo:        req.GitHubRepo,
		GitHubIssueNumber: req.GitHubIssueNumber,
		Status:            obs.ChainStatusActive,
		WorkspaceID:       req.WorkspaceID,
		WorkspacePath:     req.WorkspacePath,
		CreatedAt:         now,
	}
	_, err := s.client.Doc(collObsChains, chain.ID).Set(ctx, chainToMap(chain))
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func (s *ObservatoryStore) GetChain(ctx context.Context, id string, opts obs.ChainReadOptions) (*obs.ExecutionChain, error) {
	doc, err := s.client.Doc(collObsChains, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("chain not found: %s", id)
		}
		return nil, err
	}
	chain := mapToChain(doc.Data())

	if opts.IncludeStages {
		stages, err := s.GetChainStages(ctx, id, opts)
		if err != nil {
			return nil, err
		}
		chain.Stages = stages
	}
	return chain, nil
}

func (s *ObservatoryStore) GetChainByMessageID(ctx context.Context, messageID string) (*obs.ExecutionChain, error) {
	return s.findChainByField(ctx, "source_ref", messageID)
}

func (s *ObservatoryStore) GetChainByTaskID(ctx context.Context, taskID string) (*obs.ExecutionChain, error) {
	iter := s.client.Collection(collObsChainStages).
		Where("task_id", "==", taskID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no chain found for task: %s", taskID)
	}
	if err != nil {
		return nil, err
	}
	chainID := getString(doc.Data(), "chain_id")
	return s.GetChain(ctx, chainID, obs.ChainReadOptions{IncludeStages: true})
}

func (s *ObservatoryStore) GetChainByGitHubIssue(ctx context.Context, repo string, issueNumber int) (*obs.ExecutionChain, error) {
	iter := s.client.Collection(collObsChains).
		Where("github_repo", "==", repo).
		Where("github_issue_number", "==", issueNumber).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no chain found for %s#%d", repo, issueNumber)
	}
	if err != nil {
		return nil, err
	}
	return mapToChain(doc.Data()), nil
}

func (s *ObservatoryStore) ListChains(ctx context.Context, opts obs.ChainListOptions) ([]*obs.ChainSummary, error) {
	q := s.client.Collection(collObsChains).Query
	if opts.Status != "" {
		q = q.Where("status", "==", string(opts.Status))
	}
	if opts.SourceType != "" {
		q = q.Where("source_type", "==", opts.SourceType)
	}
	if opts.WorkspaceID != "" {
		q = q.Where("workspace_id", "==", opts.WorkspaceID)
	}
	if opts.GitHubRepo != "" {
		q = q.Where("github_repo", "==", opts.GitHubRepo)
	}
	if opts.CreatedAfter != nil {
		q = q.Where("created_at", ">=", timeToFirestore(*opts.CreatedAfter))
	}
	q = q.OrderBy("created_at", firestore.Desc)
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.ChainSummary
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		result = append(result, &obs.ChainSummary{
			ID:                getString(data, "id"),
			SourceType:        getString(data, "source_type"),
			SourceRef:         getString(data, "source_ref"),
			GitHubRepo:        getString(data, "github_repo"),
			GitHubIssueNumber: getInt(data, "github_issue_number"),
			Status:            obs.ChainStatus(getString(data, "status")),
			CurrentStage:      getInt(data, "current_stage"),
			TotalCost:         getFloat64(data, "total_cost"),
			TotalTokens:       getInt(data, "total_tokens"),
			TotalTurns:        getInt(data, "total_turns"),
			StagesCompleted:   getInt(data, "stages_completed"),
			CreatedAt:         snapshotToTime(data, "created_at"),
			CompletedAt:       snapshotToTimePtr(data, "completed_at"),
		})
	}

	if opts.AgentID != "" {
		result = s.filterChainsByAgent(ctx, result, opts.AgentID)
	}

	return result, nil
}

func (s *ObservatoryStore) UpdateChainStatus(ctx context.Context, chainID string, chainStatus obs.ChainStatus) error {
	updates := []firestore.Update{
		{Path: "status", Value: string(chainStatus)},
		{Path: "updated_at", Value: time.Now()},
	}
	if chainStatus == obs.ChainStatusCompleted || chainStatus == obs.ChainStatusFailed {
		updates = append(updates, firestore.Update{Path: "completed_at", Value: time.Now()})
	}
	_, err := s.client.Doc(collObsChains, chainID).Update(ctx, updates)
	return err
}

func (s *ObservatoryStore) UpdateChainMetrics(ctx context.Context, id string, cost float64, tokens, turns int) error {
	_, err := s.client.Doc(collObsChains, id).Update(ctx, []firestore.Update{
		{Path: "total_cost", Value: firestore.Increment(cost)},
		{Path: "total_tokens", Value: firestore.Increment(tokens)},
		{Path: "total_turns", Value: firestore.Increment(turns)},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}

func (s *ObservatoryStore) GetChainStats(ctx context.Context) (*obs.ChainStats, error) {
	iter := s.client.Collection(collObsChains).Documents(ctx)
	defer iter.Stop()

	stats := &obs.ChainStats{}
	var totalStages int
	var totalDur int64
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		stats.TotalChains++
		stats.TotalCost += getFloat64(data, "total_cost")
		stats.TotalTokens += getInt64(data, "total_tokens")
		totalStages += getInt(data, "stages_completed")

		st := obs.ChainStatus(getString(data, "status"))
		switch st {
		case obs.ChainStatusActive:
			stats.ActiveChains++
		case obs.ChainStatusPendingApproval:
			stats.PendingApprovals++
		case obs.ChainStatusCompleted:
			stats.CompletedChains++
			created := snapshotToTime(data, "created_at")
			completed := snapshotToTimePtr(data, "completed_at")
			if completed != nil {
				totalDur += completed.Sub(created).Milliseconds()
			}
		case obs.ChainStatusFailed:
			stats.FailedChains++
		}
	}

	if stats.TotalChains > 0 {
		stats.AverageStagesCount = float64(totalStages) / float64(stats.TotalChains)
	}
	if stats.CompletedChains > 0 {
		stats.AverageDurationMs = float64(totalDur) / float64(stats.CompletedChains)
	}
	return stats, nil
}

// --- Chain Stage operations ---

func (s *ObservatoryStore) CreateStage(ctx context.Context, req *obs.StageCreateRequest) (*obs.ChainStage, error) {
	stageCount := 0
	existingStages, _ := s.GetChainStages(ctx, req.ChainID, obs.ChainReadOptions{})
	if existingStages != nil {
		stageCount = len(existingStages)
	}

	now := time.Now()
	stage := &obs.ChainStage{
		ID:          uuid.New().String(),
		ChainID:     req.ChainID,
		StageNumber: stageCount + 1,
		AgentID:     req.AgentID,
		Provider:    req.Provider,
		MessageID:   req.MessageID,
		TaskID:      req.TaskID,
		HandoffTo:   req.HandoffTo,
		Status:      obs.StageStatusPending,
		Iteration:   req.Iteration,
		StartedAt:   &now,
	}
	if stage.Iteration == 0 {
		stage.Iteration = 1
	}

	_, err := s.client.Doc(collObsChainStages, stage.ID).Set(ctx, stageToMap(stage))
	if err != nil {
		return nil, err
	}

	s.client.Doc(collObsChains, req.ChainID).Update(ctx, []firestore.Update{
		{Path: "current_stage", Value: stage.StageNumber},
		{Path: "updated_at", Value: now},
	})

	return stage, nil
}

func (s *ObservatoryStore) GetStage(ctx context.Context, id string) (*obs.ChainStage, error) {
	doc, err := s.client.Doc(collObsChainStages, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("stage not found: %s", id)
		}
		return nil, err
	}
	return mapToStage(doc.Data()), nil
}

func (s *ObservatoryStore) GetChainStages(ctx context.Context, chainID string, opts obs.ChainReadOptions) ([]*obs.ChainStage, error) {
	iter := s.client.Collection(collObsChainStages).
		Where("chain_id", "==", chainID).
		OrderBy("stage_number", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var result []*obs.ChainStage
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		stage := mapToStage(doc.Data())

		if opts.IncludeSpans && stage.ID != "" {
			spans, _ := s.GetSpansByStageID(ctx, stage.ID)
			stage.Spans = spans
		}
		if opts.IncludeSessions && stage.SessionID != "" {
			session, _ := s.GetSession(ctx, stage.SessionID)
			stage.Session = session
		}
		result = append(result, stage)
	}
	return result, nil
}

func (s *ObservatoryStore) UpdateStageStatus(ctx context.Context, stageID string, stageStatus obs.ChainStageStatus) error {
	updates := []firestore.Update{
		{Path: "status", Value: string(stageStatus)},
	}
	if stageStatus == obs.StageStatusCompleted || stageStatus == obs.StageStatusFailed {
		updates = append(updates, firestore.Update{Path: "completed_at", Value: time.Now()})
	}
	_, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, updates)
	return err
}

func (s *ObservatoryStore) UpdateStageSession(ctx context.Context, stageID, sessionID string) error {
	_, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, []firestore.Update{
		{Path: "session_id", Value: sessionID},
	})
	return err
}

func (s *ObservatoryStore) UpdateStageApproval(ctx context.Context, stageID string, approvalStatus obs.ApprovalStatus, approvalType obs.ApprovalType, feedback string) error {
	updates := []firestore.Update{
		{Path: "approval_status", Value: string(approvalStatus)},
		{Path: "approval_type", Value: string(approvalType)},
	}
	if feedback != "" {
		updates = append(updates, firestore.Update{Path: "human_feedback", Value: feedback})
	}
	_, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, updates)
	return err
}

func (s *ObservatoryStore) UpdateStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64) error {
	_, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, []firestore.Update{
		{Path: "cost", Value: cost},
		{Path: "tokens_in", Value: tokensIn},
		{Path: "tokens_out", Value: tokensOut},
		{Path: "turns", Value: turns},
		{Path: "tool_calls", Value: toolCalls},
		{Path: "duration_ms", Value: durationMs},
	})
	return err
}

func (s *ObservatoryStore) UpdateStageError(ctx context.Context, stageID, errorMessage string) error {
	_, err := s.client.Doc(collObsChainStages, stageID).Update(ctx, []firestore.Update{
		{Path: "error_message", Value: errorMessage},
		{Path: "error_count", Value: firestore.Increment(1)},
	})
	return err
}

func (s *ObservatoryStore) GetSpansByStageID(ctx context.Context, stageID string) ([]*obs.Span, error) {
	iter := s.client.Collection(collObsSpans).
		Where("stage_id", "==", stageID).
		OrderBy("start_time", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var result []*obs.Span
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToSpan(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) GetChainStatusCounts(ctx context.Context, createdAfter *time.Time) (*obs.ChainStatusCounts, error) {
	q := s.client.Collection(collObsChains).Query
	if createdAfter != nil {
		q = q.Where("created_at", ">=", timeToFirestore(*createdAfter))
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	counts := &obs.ChainStatusCounts{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		counts.Total++
		counts.TotalCost += getFloat64(data, "total_cost")
		counts.TotalTokens += getInt64(data, "total_tokens")
		switch obs.ChainStatus(getString(data, "status")) {
		case obs.ChainStatusCompleted:
			counts.Completed++
		case obs.ChainStatusActive:
			counts.Active++
		case obs.ChainStatusPendingApproval:
			counts.Pending++
		case obs.ChainStatusFailed:
			counts.Failed++
		}
	}
	return counts, nil
}

func (s *ObservatoryStore) GetChainStatsByAgent(ctx context.Context, createdAfter *time.Time) ([]*obs.AgentStatsResult, error) {
	q := s.client.Collection(collObsChainStages).Query
	if createdAfter != nil {
		q = q.Where("started_at", ">=", timeToFirestore(*createdAfter))
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	agentMap := make(map[string]*obs.AgentStatsResult)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		agentID := getString(data, "agent_id")
		if agentID == "" {
			continue
		}
		stats, ok := agentMap[agentID]
		if !ok {
			stats = &obs.AgentStatsResult{AgentID: agentID}
			agentMap[agentID] = stats
		}
		stats.Stages++
		stats.TotalCost += getFloat64(data, "cost")
		stats.TokensIn += getInt(data, "tokens_in")
		stats.TokensOut += getInt(data, "tokens_out")
		switch obs.ChainStageStatus(getString(data, "status")) {
		case obs.StageStatusCompleted:
			stats.Completed++
		case obs.StageStatusFailed:
			stats.Failed++
		}
	}

	result := make([]*obs.AgentStatsResult, 0, len(agentMap))
	for _, s := range agentMap {
		result = append(result, s)
	}
	return result, nil
}

func (s *ObservatoryStore) GetSpanLitesByStageID(ctx context.Context, stageID string, limit, offset int) (*obs.SpanLitePage, error) {
	if limit <= 0 {
		limit = 50
	}

	allIter := s.client.Collection(collObsSpans).
		Where("stage_id", "==", stageID).
		Documents(ctx)
	allDocs, err := collectDocs(allIter)
	total := len(allDocs)
	if err != nil {
		return nil, err
	}

	q := s.client.Collection(collObsSpans).
		Where("stage_id", "==", stageID).
		OrderBy("start_time", firestore.Asc).
		Offset(offset).
		Limit(limit)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var spans []*obs.SpanLite
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		startTime := snapshotToTime(data, "start_time")
		endTime := snapshotToTime(data, "end_time")
		var durationMs int64
		if !endTime.IsZero() && !startTime.IsZero() {
			durationMs = endTime.Sub(startTime).Milliseconds()
		}
		spans = append(spans, &obs.SpanLite{
			ID:            getString(data, "id"),
			TraceID:       getString(data, "trace_id"),
			ParentSpanID:  getString(data, "parent_span_id"),
			ChainID:       getString(data, "chain_id"),
			StageID:       getString(data, "stage_id"),
			Name:          getString(data, "name"),
			Kind:          obs.SpanKind(getString(data, "kind")),
			Status:        getString(data, "status"),
			StatusMessage: getString(data, "status_message"),
			StartTime:     startTime,
			EndTime:       endTime,
			DurationMs:    durationMs,
			TokensIn:      getInt64(data, "tokens_in"),
			TokensOut:     getInt64(data, "tokens_out"),
			CostUSD:       getFloat64(data, "cost_usd"),
			Model:         getString(data, "model"),
			Provider:      getString(data, "provider"),
		})
	}

	return &obs.SpanLitePage{
		Spans:  spans,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *ObservatoryStore) LinkSpanToChain(ctx context.Context, spanID, chainID, stageID string) error {
	_, err := s.client.Doc(collObsSpans, spanID).Update(ctx, []firestore.Update{
		{Path: "chain_id", Value: chainID},
		{Path: "stage_id", Value: stageID},
	})
	return err
}

func (s *ObservatoryStore) ListPendingApprovals(ctx context.Context, limit int) ([]*obs.PendingApprovalInfo, error) {
	q := s.client.Collection(collObsChainStages).
		Where("status", "==", string(obs.StageStatusAwaitingApproval)).
		OrderBy("stage_number", firestore.Asc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.PendingApprovalInfo
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		result = append(result, &obs.PendingApprovalInfo{
			ChainID:        getString(data, "chain_id"),
			StageID:        getString(data, "id"),
			StageNumber:    getInt(data, "stage_number"),
			AgentID:        getString(data, "agent_id"),
			ApprovalStatus: getString(data, "approval_status"),
			ApprovalType:   obs.ApprovalType(getString(data, "approval_type")),
			TaskID:         getString(data, "task_id"),
			SessionID:      getString(data, "session_id"),
			Cost:           getFloat64(data, "cost"),
			Turns:          getInt(data, "turns"),
		})
	}
	return result, nil
}
