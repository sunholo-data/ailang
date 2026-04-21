package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

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
