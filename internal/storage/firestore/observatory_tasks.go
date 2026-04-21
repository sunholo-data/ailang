package firestore

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// --- Task operations ---

func (s *ObservatoryStore) CreateTask(ctx context.Context, t *obs.Task) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := s.client.Doc(collObsTasks, t.ID).Set(ctx, obsTaskToMap(t))
	return err
}

func (s *ObservatoryStore) GetTask(ctx context.Context, id string) (*obs.Task, error) {
	doc, err := s.client.Doc(collObsTasks, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, err
	}
	return mapToObsTask(doc.Data()), nil
}

func (s *ObservatoryStore) ListTasks(ctx context.Context, opts obs.TaskListOptions) ([]*obs.Task, error) {
	q := s.client.Collection(collObsTasks).Query
	if opts.WorkspaceID != "" {
		q = q.Where("workspace_id", "==", opts.WorkspaceID)
	}
	if opts.ParentTaskID != "" {
		q = q.Where("parent_task_id", "==", opts.ParentTaskID)
	}
	if opts.Status != "" {
		q = q.Where("status", "==", string(opts.Status))
	}
	if opts.SourceType != "" {
		q = q.Where("source_type", "==", string(opts.SourceType))
	}
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.Task
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToObsTask(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) UpdateTask(ctx context.Context, t *obs.Task) error {
	_, err := s.client.Doc(collObsTasks, t.ID).Set(ctx, obsTaskToMap(t))
	return err
}

func (s *ObservatoryStore) DeleteTask(ctx context.Context, id string) error {
	_, err := s.client.Doc(collObsTasks, id).Delete(ctx)
	return err
}

// --- Agent Assignment operations ---

func (s *ObservatoryStore) CreateAgentAssignment(ctx context.Context, a *obs.AgentAssignment) error {
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now()
	}
	_, err := s.client.Doc(collObsAgentAssignments, a.ID).Set(ctx, agentAssignmentToMap(a))
	return err
}

func (s *ObservatoryStore) GetAgentAssignment(ctx context.Context, id string) (*obs.AgentAssignment, error) {
	doc, err := s.client.Doc(collObsAgentAssignments, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("agent assignment not found: %s", id)
		}
		return nil, err
	}
	return mapToAgentAssignment(doc.Data()), nil
}

func (s *ObservatoryStore) ListAgentAssignments(ctx context.Context, taskID string) ([]*obs.AgentAssignment, error) {
	iter := s.client.Collection(collObsAgentAssignments).
		Where("task_id", "==", taskID).
		Documents(ctx)
	defer iter.Stop()

	var result []*obs.AgentAssignment
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToAgentAssignment(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) UpdateAgentAssignment(ctx context.Context, a *obs.AgentAssignment) error {
	_, err := s.client.Doc(collObsAgentAssignments, a.ID).Set(ctx, agentAssignmentToMap(a))
	return err
}

func (s *ObservatoryStore) DeleteAgentAssignment(ctx context.Context, id string) error {
	_, err := s.client.Doc(collObsAgentAssignments, id).Delete(ctx)
	return err
}

func (s *ObservatoryStore) GetAgentStats(ctx context.Context, agentID string) (*obs.AgentStats, error) {
	iter := s.client.Collection(collObsAgentAssignments).
		Where("agent_id", "==", agentID).
		Documents(ctx)
	defer iter.Stop()

	stats := &obs.AgentStats{AgentID: agentID}
	var completed int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		stats.ExecutionCount++
		stats.TotalDurationMs += getInt64(data, "duration_ms")
		stats.TotalTokensIn += getInt64(data, "tokens_in")
		stats.TotalTokensOut += getInt64(data, "tokens_out")
		stats.TotalCost += getFloat64(data, "cost_usd")
		stats.TotalToolCalls += getInt(data, "tool_calls")
		if getString(data, "status") == "completed" {
			completed++
		}
		if p := getString(data, "provider"); p != "" && stats.Provider == "" {
			stats.Provider = obs.Provider(p)
		}
	}
	if stats.ExecutionCount > 0 {
		stats.AvgDurationMs = float64(stats.TotalDurationMs) / float64(stats.ExecutionCount)
		stats.SuccessRate = float64(completed) / float64(stats.ExecutionCount)
	}
	return stats, nil
}

// --- Conversion helpers ---

func obsTaskToMap(t *obs.Task) map[string]interface{} {
	m := map[string]interface{}{
		"id":                t.ID,
		"workspace_id":      t.WorkspaceID,
		"parent_task_id":    t.ParentTaskID,
		"title":             t.Title,
		"description":       t.Description,
		"source_type":       string(t.SourceType),
		"source_ref":        t.SourceRef,
		"status":            string(t.Status),
		"priority":          t.Priority,
		"created_at":        timeToFirestore(t.CreatedAt),
		"started_at":        timePtrToFirestore(t.StartedAt),
		"completed_at":      timePtrToFirestore(t.CompletedAt),
		"total_duration_ms": t.TotalDurationMs,
		"total_tokens_in":   t.TotalTokensIn,
		"total_tokens_out":  t.TotalTokensOut,
		"total_cost_usd":    t.TotalCostUSD,
		"agent_count":       t.AgentCount,
		"span_count":        t.SpanCount,
		"error_count":       t.ErrorCount,
	}
	return m
}

func mapToObsTask(data map[string]interface{}) *obs.Task {
	return &obs.Task{
		ID:              getString(data, "id"),
		WorkspaceID:     getString(data, "workspace_id"),
		ParentTaskID:    getString(data, "parent_task_id"),
		Title:           getString(data, "title"),
		Description:     getString(data, "description"),
		SourceType:      obs.TaskSourceType(getString(data, "source_type")),
		SourceRef:       getString(data, "source_ref"),
		Status:          obs.TaskStatus(getString(data, "status")),
		Priority:        getString(data, "priority"),
		CreatedAt:       snapshotToTime(data, "created_at"),
		StartedAt:       snapshotToTimePtr(data, "started_at"),
		CompletedAt:     snapshotToTimePtr(data, "completed_at"),
		TotalDurationMs: getInt64(data, "total_duration_ms"),
		TotalTokensIn:   getInt64(data, "total_tokens_in"),
		TotalTokensOut:  getInt64(data, "total_tokens_out"),
		TotalCostUSD:    getFloat64(data, "total_cost_usd"),
		AgentCount:      getInt(data, "agent_count"),
		SpanCount:       getInt(data, "span_count"),
		ErrorCount:      getInt(data, "error_count"),
	}
}

func agentAssignmentToMap(a *obs.AgentAssignment) map[string]interface{} {
	return map[string]interface{}{
		"id":                   a.ID,
		"task_id":              a.TaskID,
		"agent_id":             a.AgentID,
		"provider":             string(a.Provider),
		"status":               string(a.Status),
		"assigned_at":          timeToFirestore(a.AssignedAt),
		"started_at":           timePtrToFirestore(a.StartedAt),
		"completed_at":         timePtrToFirestore(a.CompletedAt),
		"parent_assignment_id": a.ParentAssignmentID,
		"duration_ms":          a.DurationMs,
		"tokens_in":            a.TokensIn,
		"tokens_out":           a.TokensOut,
		"cost_usd":             a.CostUSD,
		"tool_calls":           a.ToolCalls,
		"turns":                a.Turns,
	}
}

func mapToAgentAssignment(data map[string]interface{}) *obs.AgentAssignment {
	return &obs.AgentAssignment{
		ID:                 getString(data, "id"),
		TaskID:             getString(data, "task_id"),
		AgentID:            getString(data, "agent_id"),
		Provider:           obs.Provider(getString(data, "provider")),
		Status:             obs.AgentAssignmentStatus(getString(data, "status")),
		AssignedAt:         snapshotToTime(data, "assigned_at"),
		StartedAt:          snapshotToTimePtr(data, "started_at"),
		CompletedAt:        snapshotToTimePtr(data, "completed_at"),
		ParentAssignmentID: getString(data, "parent_assignment_id"),
		DurationMs:         getInt64(data, "duration_ms"),
		TokensIn:           getInt64(data, "tokens_in"),
		TokensOut:          getInt64(data, "tokens_out"),
		CostUSD:            getFloat64(data, "cost_usd"),
		ToolCalls:          getInt(data, "tool_calls"),
		Turns:              getInt(data, "turns"),
	}
}
