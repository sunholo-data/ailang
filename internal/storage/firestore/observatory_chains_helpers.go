package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/sunholo-data/ailang/internal/mapval"
	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// --- Conversion helpers ---

func chainToMap(c *obs.ExecutionChain, ttl time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"expire_at":           expireAt(c.CreatedAt, ttl),
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
		ID:                mapval.String(data, "id"),
		SourceType:        obs.ChainSourceType(mapval.String(data, "source_type")),
		SourceRef:         mapval.String(data, "source_ref"),
		GitHubRepo:        mapval.String(data, "github_repo"),
		GitHubIssueNumber: mapval.Int(data, "github_issue_number"),
		Status:            obs.ChainStatus(mapval.String(data, "status")),
		CurrentStage:      mapval.Int(data, "current_stage"),
		WorkspaceID:       mapval.String(data, "workspace_id"),
		WorkspacePath:     mapval.String(data, "workspace_path"),
		CreatedAt:         snapshotToTime(data, "created_at"),
		UpdatedAt:         snapshotToTimePtr(data, "updated_at"),
		CompletedAt:       snapshotToTimePtr(data, "completed_at"),
		TotalCost:         mapval.Float(data, "total_cost"),
		TotalTokens:       mapval.Int(data, "total_tokens"),
		TotalTurns:        mapval.Int(data, "total_turns"),
		StagesCompleted:   mapval.Int(data, "stages_completed"),
	}
}

func stageToMap(st *obs.ChainStage, ttl time.Duration) map[string]interface{} {
	var started time.Time
	if st.StartedAt != nil {
		started = *st.StartedAt
	}
	m := map[string]interface{}{
		"expire_at":             expireAt(started, ttl),
		"id":                    st.ID,
		"chain_id":              st.ChainID,
		"stage_number":          st.StageNumber,
		"agent_id":              st.AgentID,
		"provider":              string(st.Provider),
		"message_id":            st.MessageID,
		"task_id":               st.TaskID,
		"session_id":            st.SessionID,
		"status":                string(st.Status),
		"approval_status":       string(st.ApprovalStatus),
		"approval_type":         string(st.ApprovalType),
		"handoff_to":            st.HandoffTo,
		"iteration":             st.Iteration,
		"human_feedback":        st.HumanFeedback,
		"started_at":            timePtrToFirestore(st.StartedAt),
		"completed_at":          timePtrToFirestore(st.CompletedAt),
		"cost":                  st.Cost,
		"tokens_in":             st.TokensIn,
		"tokens_out":            st.TokensOut,
		"quota_tokens":          st.QuotaTokens,
		"cache_read_tokens":     st.CacheReadTokens,
		"cache_creation_tokens": st.CacheCreationTokens,
		"turns":                 st.Turns,
		"tool_calls":            st.ToolCalls,
		"duration_ms":           st.DurationMs,
		"error_message":         st.ErrorMessage,
		"error_count":           st.ErrorCount,
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
		ID:                  mapval.String(data, "id"),
		ChainID:             mapval.String(data, "chain_id"),
		StageNumber:         mapval.Int(data, "stage_number"),
		AgentID:             mapval.String(data, "agent_id"),
		Provider:            obs.Provider(mapval.String(data, "provider")),
		MessageID:           mapval.String(data, "message_id"),
		TaskID:              mapval.String(data, "task_id"),
		SessionID:           mapval.String(data, "session_id"),
		Status:              obs.ChainStageStatus(mapval.String(data, "status")),
		ApprovalStatus:      obs.ApprovalStatus(mapval.String(data, "approval_status")),
		ApprovalType:        obs.ApprovalType(mapval.String(data, "approval_type")),
		HandoffTo:           mapval.String(data, "handoff_to"),
		Iteration:           mapval.Int(data, "iteration"),
		HumanFeedback:       mapval.String(data, "human_feedback"),
		StartedAt:           snapshotToTimePtr(data, "started_at"),
		CompletedAt:         snapshotToTimePtr(data, "completed_at"),
		Cost:                mapval.Float(data, "cost"),
		TokensIn:            mapval.Int(data, "tokens_in"),
		TokensOut:           mapval.Int(data, "tokens_out"),
		QuotaTokens:         mapval.Int64(data, "quota_tokens"),
		CacheReadTokens:     mapval.Int(data, "cache_read_tokens"),
		CacheCreationTokens: mapval.Int(data, "cache_creation_tokens"),
		Turns:               mapval.Int(data, "turns"),
		ToolCalls:           mapval.Int(data, "tool_calls"),
		DurationMs:          mapval.Int64(data, "duration_ms"),
		ErrorMessage:        mapval.String(data, "error_message"),
		ErrorCount:          mapval.Int(data, "error_count"),
	}
	if evalStr := mapval.String(data, "eval_assessment"); evalStr != "" {
		var ea obs.EvalAssessment
		if err := json.Unmarshal([]byte(evalStr), &ea); err == nil {
			st.EvalAssessment = &ea
		}
	}
	return st
}

func mapToChatMessage(data map[string]interface{}) *obs.ChatMessage {
	return &obs.ChatMessage{
		ID:                  mapval.String(data, "id"),
		SessionID:           mapval.String(data, "session_id"),
		TurnNumber:          mapval.Int(data, "turn_number"),
		Role:                mapval.String(data, "role"),
		ContentJSON:         mapval.String(data, "content_json"),
		TokensIn:            mapval.Int(data, "tokens_in"),
		TokensOut:           mapval.Int(data, "tokens_out"),
		CacheReadTokens:     mapval.Int(data, "cache_read_tokens"),
		CacheCreationTokens: mapval.Int(data, "cache_creation_tokens"),
		Model:               mapval.String(data, "model"),
		Timestamp:           snapshotToTime(data, "timestamp"),
		TaskID:              mapval.String(data, "task_id"),
		ChainID:             mapval.String(data, "chain_id"),
	}
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

func (s *ObservatoryStore) chainIDsForAgent(ctx context.Context, agentID string) (map[string]bool, error) {
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
			return nil, err
		}
		chainIDsWithAgent[mapval.String(doc.Data(), "chain_id")] = true
	}

	return chainIDsWithAgent, nil
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
		if mapval.String(doc.Data(), "task_id") != "" {
			withTaskID++
		}
	}
	return total, withTaskID, nil
}
