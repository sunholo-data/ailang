package firestore

import (
	"context"
	"encoding/json"
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
	// Check stages for task_id link
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

	// Apply agent_id filter client-side if needed (requires checking stages)
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
			// Calculate duration for completed chains
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
	// Get current stage count for chain
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

	// Update chain's current_stage and stages_completed
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
	return s.ListSpans(ctx, obs.SpanListOptions{}) // Filter by stage_id would need custom query
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

// --- Chat message operations ---

func (s *ObservatoryStore) GetChatMessagesByTaskID(ctx context.Context, taskID string) ([]*obs.ChatMessage, error) {
	iter := s.client.Collection(collObsChatMessages).
		Where("task_id", "==", taskID).
		OrderBy("timestamp", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var result []*obs.ChatMessage
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToChatMessage(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) GetChatMessagesBySession(ctx context.Context, sessionID string, startTime, endTime time.Time) ([]*obs.ChatMessage, error) {
	q := s.client.Collection(collObsChatMessages).
		Where("session_id", "==", sessionID)
	if !startTime.IsZero() {
		q = q.Where("timestamp", ">=", timeToFirestore(startTime))
	}
	if !endTime.IsZero() {
		q = q.Where("timestamp", "<=", timeToFirestore(endTime))
	}
	q = q.OrderBy("timestamp", firestore.Asc)

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.ChatMessage
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToChatMessage(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) CountChatMessages(ctx context.Context, q obs.ChatMessageQuery) (total int, withTaskID int, err error) {
	fq := s.client.Collection(collObsChatMessages).Query
	if q.SessionID != "" {
		fq = fq.Where("session_id", "==", q.SessionID)
	}
	if q.TaskID != "" {
		fq = fq.Where("task_id", "==", q.TaskID)
	}
	if q.Limit > 0 {
		fq = fq.Limit(q.Limit)
	}

	iter := fq.Documents(ctx)
	defer iter.Stop()

	for {
		doc, iterErr := iter.Next()
		if iterErr == iterator.Done {
			break
		}
		if iterErr != nil {
			return total, withTaskID, iterErr
		}
		total++
		if getString(doc.Data(), "task_id") != "" {
			withTaskID++
		}
	}
	return total, withTaskID, nil
}

// --- Helper methods ---

func (s *ObservatoryStore) findChainByField(ctx context.Context, field, value string) (*obs.ExecutionChain, error) {
	iter := s.client.Collection(collObsChains).
		Where(field, "==", value).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("chain not found: %s=%s", field, value)
	}
	if err != nil {
		return nil, err
	}
	return mapToChain(doc.Data()), nil
}

func (s *ObservatoryStore) filterChainsByAgent(ctx context.Context, chains []*obs.ChainSummary, agentID string) []*obs.ChainSummary {
	// Build set of chain IDs that have a stage with this agent
	chainIDsWithAgent := make(map[string]bool)
	iter := s.client.Collection(collObsChainStages).
		Where("agent_id", "==", agentID).
		Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		chainIDsWithAgent[getString(doc.Data(), "chain_id")] = true
	}

	var filtered []*obs.ChainSummary
	for _, c := range chains {
		if chainIDsWithAgent[c.ID] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// --- Conversion helpers ---

func chainToMap(c *obs.ExecutionChain) map[string]interface{} {
	return map[string]interface{}{
		"id":                  c.ID,
		"source_type":         string(c.SourceType),
		"source_ref":          c.SourceRef,
		"github_repo":         c.GitHubRepo,
		"github_issue_number": c.GitHubIssueNumber,
		"status":              string(c.Status),
		"current_stage":       c.CurrentStage,
		"workspace_id":        c.WorkspaceID,
		"workspace_path":      c.WorkspacePath,
		"created_at":          timeToFirestore(c.CreatedAt),
		"updated_at":          timePtrToFirestore(c.UpdatedAt),
		"completed_at":        timePtrToFirestore(c.CompletedAt),
		"total_cost":          c.TotalCost,
		"total_tokens":        c.TotalTokens,
		"total_turns":         c.TotalTurns,
		"stages_completed":    c.StagesCompleted,
	}
}

func mapToChain(data map[string]interface{}) *obs.ExecutionChain {
	return &obs.ExecutionChain{
		ID:                getString(data, "id"),
		SourceType:        obs.ChainSourceType(getString(data, "source_type")),
		SourceRef:         getString(data, "source_ref"),
		GitHubRepo:        getString(data, "github_repo"),
		GitHubIssueNumber: getInt(data, "github_issue_number"),
		Status:            obs.ChainStatus(getString(data, "status")),
		CurrentStage:      getInt(data, "current_stage"),
		WorkspaceID:       getString(data, "workspace_id"),
		WorkspacePath:     getString(data, "workspace_path"),
		CreatedAt:         snapshotToTime(data, "created_at"),
		UpdatedAt:         snapshotToTimePtr(data, "updated_at"),
		CompletedAt:       snapshotToTimePtr(data, "completed_at"),
		TotalCost:         getFloat64(data, "total_cost"),
		TotalTokens:       getInt(data, "total_tokens"),
		TotalTurns:        getInt(data, "total_turns"),
		StagesCompleted:   getInt(data, "stages_completed"),
	}
}

func stageToMap(st *obs.ChainStage) map[string]interface{} {
	m := map[string]interface{}{
		"id":              st.ID,
		"chain_id":        st.ChainID,
		"stage_number":    st.StageNumber,
		"agent_id":        st.AgentID,
		"provider":        string(st.Provider),
		"message_id":      st.MessageID,
		"task_id":         st.TaskID,
		"session_id":      st.SessionID,
		"status":          string(st.Status),
		"approval_status": string(st.ApprovalStatus),
		"approval_type":   string(st.ApprovalType),
		"handoff_to":      st.HandoffTo,
		"iteration":       st.Iteration,
		"human_feedback":  st.HumanFeedback,
		"started_at":      timePtrToFirestore(st.StartedAt),
		"completed_at":    timePtrToFirestore(st.CompletedAt),
		"cost":            st.Cost,
		"tokens_in":       st.TokensIn,
		"tokens_out":      st.TokensOut,
		"turns":           st.Turns,
		"tool_calls":      st.ToolCalls,
		"duration_ms":     st.DurationMs,
		"error_message":   st.ErrorMessage,
		"error_count":     st.ErrorCount,
	}
	if st.EvalAssessment != nil {
		if b, err := json.Marshal(st.EvalAssessment); err == nil {
			m["eval_assessment"] = string(b)
		}
	}
	return m
}

func mapToStage(data map[string]interface{}) *obs.ChainStage {
	st := &obs.ChainStage{
		ID:             getString(data, "id"),
		ChainID:        getString(data, "chain_id"),
		StageNumber:    getInt(data, "stage_number"),
		AgentID:        getString(data, "agent_id"),
		Provider:       obs.Provider(getString(data, "provider")),
		MessageID:      getString(data, "message_id"),
		TaskID:         getString(data, "task_id"),
		SessionID:      getString(data, "session_id"),
		Status:         obs.ChainStageStatus(getString(data, "status")),
		ApprovalStatus: obs.ApprovalStatus(getString(data, "approval_status")),
		ApprovalType:   obs.ApprovalType(getString(data, "approval_type")),
		HandoffTo:      getString(data, "handoff_to"),
		Iteration:      getInt(data, "iteration"),
		HumanFeedback:  getString(data, "human_feedback"),
		StartedAt:      snapshotToTimePtr(data, "started_at"),
		CompletedAt:    snapshotToTimePtr(data, "completed_at"),
		Cost:           getFloat64(data, "cost"),
		TokensIn:       getInt(data, "tokens_in"),
		TokensOut:      getInt(data, "tokens_out"),
		Turns:          getInt(data, "turns"),
		ToolCalls:      getInt(data, "tool_calls"),
		DurationMs:     getInt64(data, "duration_ms"),
		ErrorMessage:   getString(data, "error_message"),
		ErrorCount:     getInt(data, "error_count"),
	}
	if evalStr := getString(data, "eval_assessment"); evalStr != "" {
		var ea obs.EvalAssessment
		if err := json.Unmarshal([]byte(evalStr), &ea); err == nil {
			st.EvalAssessment = &ea
		}
	}
	return st
}

func mapToChatMessage(data map[string]interface{}) *obs.ChatMessage {
	return &obs.ChatMessage{
		ID:          getString(data, "id"),
		SessionID:   getString(data, "session_id"),
		TurnNumber:  getInt(data, "turn_number"),
		Role:        getString(data, "role"),
		ContentJSON: getString(data, "content_json"),
		TokensIn:    getInt(data, "tokens_in"),
		TokensOut:   getInt(data, "tokens_out"),
		Model:       getString(data, "model"),
		Timestamp:   snapshotToTime(data, "timestamp"),
		TaskID:      getString(data, "task_id"),
		ChainID:     getString(data, "chain_id"),
	}
}
