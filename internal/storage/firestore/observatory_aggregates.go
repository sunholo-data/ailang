package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	obs "github.com/sunholo/ailang/internal/observatory"
)

// --- Aggregate operations ---

func (s *ObservatoryStore) GetMetricsSummary(ctx context.Context) (*obs.MetricsSummary, error) {
	summary := &obs.MetricsSummary{}

	// Count workspaces
	if docs, err := collectDocs(s.client.Collection(collObsWorkspaces).Documents(ctx)); err == nil {
		summary.TotalWorkspaces = len(docs)
	}
	// Count tasks
	if docs, err := collectDocs(s.client.Collection(collObsTasks).Documents(ctx)); err == nil {
		summary.TotalTasks = len(docs)
	}

	// Aggregate spans
	spanIter := s.client.Collection(collObsSpans).Documents(ctx)
	defer spanIter.Stop()
	var totalSpans, errorCount int
	for {
		doc, err := spanIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		totalSpans++
		summary.TotalTokensIn += getInt64(data, "tokens_in")
		summary.TotalTokensOut += getInt64(data, "tokens_out")
		summary.TotalCostUSD += getFloat64(data, "cost_usd")
		summary.TotalCacheReadTokens += getInt64(data, "cache_read_tokens")
		summary.TotalCacheCreationTokens += getInt64(data, "cache_creation_tokens")
		if getString(data, "status") == "error" {
			errorCount++
		}
	}
	summary.TotalSpans = totalSpans
	if totalSpans > 0 {
		summary.SuccessRate = float64(totalSpans-errorCount) / float64(totalSpans)
	}

	// Count unique agents
	agentMap := make(map[string]bool)
	aaIter := s.client.Collection(collObsAgentAssignments).Documents(ctx)
	defer aaIter.Stop()
	for {
		doc, err := aaIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		agentMap[getString(doc.Data(), "agent_id")] = true
	}
	summary.TotalAgents = len(agentMap)

	return summary, nil
}

func (s *ObservatoryStore) GetProviderComparison(ctx context.Context) ([]*obs.ProviderComparison, error) {
	iter := s.client.Collection(collObsSpans).Documents(ctx)
	defer iter.Stop()

	providerStats := make(map[string]*obs.ProviderComparison)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		p := getString(data, "provider")
		if p == "" {
			continue
		}
		pc, ok := providerStats[p]
		if !ok {
			pc = &obs.ProviderComparison{Provider: obs.Provider(p)}
			providerStats[p] = pc
		}
		pc.TotalExecutions++
		pc.TotalTokensIn += getInt64(data, "tokens_in")
		pc.TotalTokensOut += getInt64(data, "tokens_out")
		pc.TotalCost += getFloat64(data, "cost_usd")
		dur := getInt64(data, "duration_ms")
		pc.AvgDurationMs += float64(dur)
		if getString(data, "status") != "error" {
			pc.SuccessRate += 1.0
		}
	}

	result := make([]*obs.ProviderComparison, 0, len(providerStats))
	for _, pc := range providerStats {
		if pc.TotalExecutions > 0 {
			pc.AvgDurationMs /= float64(pc.TotalExecutions)
			pc.SuccessRate /= float64(pc.TotalExecutions)
		}
		result = append(result, pc)
	}
	return result, nil
}

func (s *ObservatoryStore) GetTaskTimeline(ctx context.Context, taskID string) ([]*obs.TaskTimeline, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	spans, err := s.ListSpans(ctx, obs.SpanListOptions{TaskID: taskID})
	if err != nil {
		return nil, err
	}

	var timeline []*obs.TaskTimeline
	for _, sp := range spans {
		timeline = append(timeline, &obs.TaskTimeline{
			TaskID:     taskID,
			Title:      task.Title,
			Status:     task.Status,
			SpanID:     sp.ID,
			SpanName:   sp.Name,
			StartTime:  &sp.StartTime,
			EndTime:    sp.EndTime,
			DurationMs: sp.DurationMs,
			SpanStatus: sp.Status,
			TokensIn:   sp.TokensIn,
			TokensOut:  sp.TokensOut,
			CostUSD:    sp.CostUSD,
			Provider:   sp.Provider,
		})
	}
	return timeline, nil
}

func (s *ObservatoryStore) GetExecTaskHierarchy(ctx context.Context, limit int) ([]*obs.ExecTaskNode, error) {
	// Find root exec spans (those without parent)
	q := s.client.Collection(collObsSpans).
		Where("parent_span_id", "==", "").
		OrderBy("start_time", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var nodes []*obs.ExecTaskNode
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
		nodes = append(nodes, &obs.ExecTaskNode{
			TaskID:     getString(data, "task_id"),
			Command:    getString(data, "name"),
			Provider:   getString(data, "provider"),
			Status:     getString(data, "status"),
			StartTime:  &startTime,
			DurationMs: int(getInt64(data, "duration_ms")),
		})
	}
	return nodes, nil
}

// --- Metric operations ---

func (s *ObservatoryStore) CreateMetric(ctx context.Context, m *obs.Metric) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	id := fmt.Sprintf("metric_%d_%s", time.Now().UnixMilli(), generateShortID())
	_, err := s.client.Doc(collObsMetrics, id).Set(ctx, obsMetricToMap(m))
	return err
}

func (s *ObservatoryStore) ListMetrics(ctx context.Context, opts obs.MetricListOptions) ([]*obs.Metric, error) {
	q := s.client.Collection(collObsMetrics).Query
	if opts.SessionID != "" {
		q = q.Where("session_id", "==", opts.SessionID)
	}
	if opts.Name != "" {
		q = q.Where("name", "==", opts.Name)
	}
	if opts.Workspace != "" {
		q = q.Where("workspace", "==", opts.Workspace)
	}
	q = q.OrderBy("timestamp", firestore.Desc)
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.Metric
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToObsMetric(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) GetSessionMetricsSummary(ctx context.Context, sessionID string) (*obs.SessionMetricsSummary, error) {
	summary := &obs.SessionMetricsSummary{SessionID: sessionID}

	// Aggregate from spans
	spanIter := s.client.Collection(collObsSpans).
		Where("session_id", "==", sessionID).
		Documents(ctx)
	defer spanIter.Stop()

	for {
		doc, err := spanIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		summary.TokensIn += getInt64(data, "tokens_in")
		summary.TokensOut += getInt64(data, "tokens_out")
		summary.CacheReadTokens += getInt64(data, "cache_read_tokens")
		summary.CacheCreationTokens += getInt64(data, "cache_creation_tokens")
		summary.TotalCostUSD += getFloat64(data, "cost_usd")
		summary.DurationMs += getInt64(data, "duration_ms")
		summary.SpanCount++
		if getString(data, "status") == "error" {
			summary.ErrorCount++
		}
	}

	if summary.SpanCount > 0 {
		summary.SuccessRate = float64(summary.SpanCount-summary.ErrorCount) / float64(summary.SpanCount)
	}
	return summary, nil
}

// --- Session detail operations ---

func (s *ObservatoryStore) GetSession(ctx context.Context, sessionID string) (*obs.Session, error) {
	doc, err := s.client.Doc(collObsSessions, sessionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, err
	}
	data := doc.Data()
	return &obs.Session{
		SessionID:     getString(data, "session_id"),
		Workspace:     getString(data, "workspace"),
		ClaudeVersion: getString(data, "claude_version"),
		Source:        getString(data, "source"),
		StartedAt:     snapshotToTime(data, "started_at"),
		EndedAt:       snapshotToTimePtr(data, "ended_at"),
		TaskID:        getString(data, "task_id"),
		ChainID:       getString(data, "chain_id"),
		StageID:       getString(data, "stage_id"),
		MessageID:     getString(data, "message_id"),
	}, nil
}

func (s *ObservatoryStore) GetSessionTools(ctx context.Context, sessionID string) ([]obs.SessionTool, error) {
	iter := s.client.Collection(collObsSessionTools).
		Where("session_id", "==", sessionID).
		OrderBy("start_time", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var result []obs.SessionTool
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		tool := obs.SessionTool{
			ToolUseID: getString(data, "tool_use_id"),
			SessionID: getString(data, "session_id"),
			ToolName:  getString(data, "tool_name"),
			StartTime: snapshotToTime(data, "start_time"),
			EndTime:   snapshotToTimePtr(data, "end_time"),
		}
		if input := getString(data, "tool_input"); input != "" {
			tool.ToolInput = json.RawMessage(input)
		}
		if resp := getString(data, "tool_response"); resp != "" {
			tool.ToolResponse = json.RawMessage(resp)
		}
		if v, ok := data["success"]; ok && v != nil {
			b := getBool(data, "success")
			tool.Success = &b
		}
		result = append(result, tool)
	}
	return result, nil
}

// --- Metric conversion helpers ---

func obsMetricToMap(m *obs.Metric) map[string]interface{} {
	data := map[string]interface{}{
		"name":           m.Name,
		"metric_type":    m.Type,
		"session_id":     m.SessionID,
		"workspace":      m.Workspace,
		"provider":       m.Provider,
		"label_type":     m.LabelType,
		"label_tool":     m.LabelTool,
		"label_decision": m.LabelDecision,
		"label_language": m.LabelLanguage,
		"label_model":    m.LabelModel,
		"value_int":      m.ValueInt,
		"value_float":    m.ValueFloat,
		"timestamp":      timeToFirestore(m.Timestamp),
		"created_at":     timeToFirestore(m.CreatedAt),
	}
	if m.Labels != nil {
		if b, err := json.Marshal(m.Labels); err == nil {
			data["labels"] = string(b)
		}
	}
	if m.ResourceAttributes != nil {
		if b, err := json.Marshal(m.ResourceAttributes); err == nil {
			data["resource_attributes"] = string(b)
		}
	}
	return data
}

func mapToObsMetric(data map[string]interface{}) *obs.Metric {
	m := &obs.Metric{
		Name:          getString(data, "name"),
		Type:          getString(data, "metric_type"),
		SessionID:     getString(data, "session_id"),
		Workspace:     getString(data, "workspace"),
		Provider:      getString(data, "provider"),
		LabelType:     getString(data, "label_type"),
		LabelTool:     getString(data, "label_tool"),
		LabelDecision: getString(data, "label_decision"),
		LabelLanguage: getString(data, "label_language"),
		LabelModel:    getString(data, "label_model"),
		ValueInt:      getInt64(data, "value_int"),
		ValueFloat:    getFloat64(data, "value_float"),
		Timestamp:     snapshotToTime(data, "timestamp"),
		CreatedAt:     snapshotToTime(data, "created_at"),
	}
	if labelsStr := getString(data, "labels"); labelsStr != "" {
		_ = json.Unmarshal([]byte(labelsStr), &m.Labels)
	}
	if raStr := getString(data, "resource_attributes"); raStr != "" {
		_ = json.Unmarshal([]byte(raStr), &m.ResourceAttributes)
	}
	return m
}
