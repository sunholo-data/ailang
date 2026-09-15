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

	"github.com/sunholo-data/ailang/internal/mapval"
	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// --- Aggregate operations ---

func (s *ObservatoryStore) GetMetricsSummary(ctx context.Context) (*obs.MetricsSummary, error) {
	if cached, ok := s.metricsSummaryCache.get(); ok {
		return &cached, nil
	}

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
		summary.TotalTokensIn += mapval.Int64(data, "tokens_in")
		summary.TotalTokensOut += mapval.Int64(data, "tokens_out")
		summary.TotalCostUSD += mapval.Float(data, "cost_usd")
		summary.TotalCacheReadTokens += mapval.Int64(data, "cache_read_tokens")
		summary.TotalCacheCreationTokens += mapval.Int64(data, "cache_creation_tokens")
		if mapval.String(data, "status") == "error" {
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
		agentMap[mapval.String(doc.Data(), "agent_id")] = true
	}
	summary.TotalAgents = len(agentMap)

	s.metricsSummaryCache.set(*summary)
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
		p := mapval.String(data, "provider")
		if p == "" {
			continue
		}
		pc, ok := providerStats[p]
		if !ok {
			pc = &obs.ProviderComparison{Provider: obs.Provider(p)}
			providerStats[p] = pc
		}
		pc.TotalExecutions++
		pc.TotalTokensIn += mapval.Int64(data, "tokens_in")
		pc.TotalTokensOut += mapval.Int64(data, "tokens_out")
		pc.TotalCost += mapval.Float(data, "cost_usd")
		dur := mapval.Int64(data, "duration_ms")
		pc.AvgDurationMs += float64(dur)
		if mapval.String(data, "status") != "error" {
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
			TaskID:     mapval.String(data, "task_id"),
			Command:    mapval.String(data, "name"),
			Provider:   mapval.String(data, "provider"),
			Status:     mapval.String(data, "status"),
			StartTime:  &startTime,
			DurationMs: int(mapval.Int64(data, "duration_ms")),
		})
	}
	return nodes, nil
}

func (s *ObservatoryStore) GetExecTaskHierarchyWithMessages(ctx context.Context, limit int) (*obs.ExecHierarchyWithMessages, error) {
	// Get flat exec hierarchy first
	execs, err := s.GetExecTaskHierarchy(ctx, limit)
	if err != nil {
		return nil, err
	}
	if len(execs) == 0 {
		return &obs.ExecHierarchyWithMessages{Count: 0}, nil
	}

	// Collect unique task IDs and batch-fetch them with GetAll (1 round-trip instead of N).
	taskIDSet := make(map[string]bool)
	var refs []*firestore.DocumentRef
	for _, exec := range execs {
		if exec.TaskID == "" || taskIDSet[exec.TaskID] {
			continue
		}
		taskIDSet[exec.TaskID] = true
		refs = append(refs, s.client.Doc(collObsTasks, exec.TaskID))
	}

	taskToMessage := make(map[string]string)
	if len(refs) > 0 {
		docs, err := s.client.GetAll(ctx, refs)
		if err != nil {
			// Fall back gracefully — hierarchy still works, just without message grouping.
			docs = nil
		}
		for _, doc := range docs {
			if doc == nil || !doc.Exists() {
				continue
			}
			data := doc.Data()
			taskID := mapval.String(data, "id")
			if taskID == "" {
				taskID = doc.Ref.ID
			}
			if mapval.String(data, "source_type") == "message" {
				if ref := mapval.String(data, "source_ref"); ref != "" {
					taskToMessage[taskID] = ref
				}
			}
		}
	}

	// Group execs by message
	messageToExecs := make(map[string][]*obs.ExecTaskNode)
	var orphans []*obs.ExecTaskNode
	for _, exec := range execs {
		if msgID, ok := taskToMessage[exec.TaskID]; ok {
			messageToExecs[msgID] = append(messageToExecs[msgID], exec)
		} else {
			orphans = append(orphans, exec)
		}
	}

	// Fetch message details
	var messages []*obs.MessageNode
	for msgID, msgExecs := range messageToExecs {
		msgNode := &obs.MessageNode{
			MessageID: msgID,
			Execs:     msgExecs,
		}
		if msg, err := s.GetMessage(ctx, msgID); err == nil && msg != nil {
			msgNode.Title = msg.Title
			msgNode.FromAgent = msg.FromAgent
			msgNode.ToInbox = msg.Inbox
			msgNode.Status = string(msg.Status)
			msgNode.CreatedAt = &msg.CreatedAt
		}
		messages = append(messages, msgNode)
	}

	return &obs.ExecHierarchyWithMessages{
		Messages: messages,
		Orphan:   orphans,
		Count:    len(execs),
	}, nil
}

func (s *ObservatoryStore) GetSpanHierarchy(ctx context.Context, limit int) (*obs.SpanHierarchyResult, error) {
	if limit <= 0 {
		limit = 100
	}

	// Fetch recent spans
	iter := s.client.Collection(collObsSpans).
		OrderBy("start_time", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	spanMap := make(map[string]*obs.SpanHierarchyNode)
	var allNodes []*obs.SpanHierarchyNode
	sessions := make(map[string]int)

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
		node := &obs.SpanHierarchyNode{
			ID:         mapval.String(data, "id"),
			Name:       mapval.String(data, "name"),
			ParentID:   mapval.String(data, "parent_span_id"),
			StartTime:  startTime,
			DurationMs: mapval.Int64(data, "duration_ms"),
			TokensIn:   mapval.Int64(data, "tokens_in"),
			TokensOut:  mapval.Int64(data, "tokens_out"),
			CostUSD:    mapval.Float(data, "cost_usd"),
			SessionID:  mapval.String(data, "session_id"),
			Status:     obs.SpanStatus(mapval.String(data, "status")),
			Provider:   obs.Provider(mapval.String(data, "provider")),
		}
		spanMap[node.ID] = node
		allNodes = append(allNodes, node)

		if node.SessionID != "" {
			sessions[node.SessionID]++
		}
	}

	// Build tree by linking children to parents
	var roots []*obs.SpanHierarchyNode
	for _, node := range allNodes {
		if node.ParentID != "" {
			if parent, ok := spanMap[node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	// Compute stats
	stats := obs.SpanHierarchyStats{TotalSpans: len(allNodes)}
	for _, node := range allNodes {
		stats.TotalCost += node.CostUSD
		stats.TotalTokens.In += node.TokensIn
		stats.TotalTokens.Out += node.TokensOut
	}

	return &obs.SpanHierarchyResult{
		Roots:    roots,
		Sessions: sessions,
		Stats:    stats,
	}, nil
}

func (s *ObservatoryStore) GetToolsByTimestampRange(ctx context.Context, start, end time.Time, toolName string) ([]obs.SessionTool, error) {
	q := s.client.Collection(collObsSessionTools).
		Where("start_time", ">=", timeToFirestore(start)).
		Where("start_time", "<=", timeToFirestore(end)).
		OrderBy("start_time", firestore.Asc)
	if toolName != "" {
		q = q.Where("tool_name", "==", toolName)
	}

	iter := q.Documents(ctx)
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
			ToolUseID: mapval.String(data, "tool_use_id"),
			SessionID: mapval.String(data, "session_id"),
			ToolName:  mapval.String(data, "tool_name"),
			StartTime: snapshotToTime(data, "start_time"),
			EndTime:   snapshotToTimePtr(data, "end_time"),
		}
		if input := mapval.String(data, "tool_input"); input != "" {
			tool.ToolInput = json.RawMessage(input)
		}
		if resp := mapval.String(data, "tool_response"); resp != "" {
			tool.ToolResponse = json.RawMessage(resp)
		}
		if v, ok := data["success"]; ok && v != nil {
			b := mapval.Bool(data, "success")
			tool.Success = &b
		}
		result = append(result, tool)
	}
	return result, nil
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
		summary.TokensIn += mapval.Int64(data, "tokens_in")
		summary.TokensOut += mapval.Int64(data, "tokens_out")
		summary.CacheReadTokens += mapval.Int64(data, "cache_read_tokens")
		summary.CacheCreationTokens += mapval.Int64(data, "cache_creation_tokens")
		summary.TotalCostUSD += mapval.Float(data, "cost_usd")
		summary.DurationMs += mapval.Int64(data, "duration_ms")
		summary.SpanCount++
		if mapval.String(data, "status") == "error" {
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
		SessionID:     mapval.String(data, "session_id"),
		Workspace:     mapval.String(data, "workspace"),
		ClaudeVersion: mapval.String(data, "claude_version"),
		Source:        mapval.String(data, "source"),
		StartedAt:     snapshotToTime(data, "started_at"),
		EndedAt:       snapshotToTimePtr(data, "ended_at"),
		TaskID:        mapval.String(data, "task_id"),
		ChainID:       mapval.String(data, "chain_id"),
		StageID:       mapval.String(data, "stage_id"),
		MessageID:     mapval.String(data, "message_id"),
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
			ToolUseID: mapval.String(data, "tool_use_id"),
			SessionID: mapval.String(data, "session_id"),
			ToolName:  mapval.String(data, "tool_name"),
			StartTime: snapshotToTime(data, "start_time"),
			EndTime:   snapshotToTimePtr(data, "end_time"),
		}
		if input := mapval.String(data, "tool_input"); input != "" {
			tool.ToolInput = json.RawMessage(input)
		}
		if resp := mapval.String(data, "tool_response"); resp != "" {
			tool.ToolResponse = json.RawMessage(resp)
		}
		if v, ok := data["success"]; ok && v != nil {
			b := mapval.Bool(data, "success")
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
		Name:          mapval.String(data, "name"),
		Type:          mapval.String(data, "metric_type"),
		SessionID:     mapval.String(data, "session_id"),
		Workspace:     mapval.String(data, "workspace"),
		Provider:      mapval.String(data, "provider"),
		LabelType:     mapval.String(data, "label_type"),
		LabelTool:     mapval.String(data, "label_tool"),
		LabelDecision: mapval.String(data, "label_decision"),
		LabelLanguage: mapval.String(data, "label_language"),
		LabelModel:    mapval.String(data, "label_model"),
		ValueInt:      mapval.Int64(data, "value_int"),
		ValueFloat:    mapval.Float(data, "value_float"),
		Timestamp:     snapshotToTime(data, "timestamp"),
		CreatedAt:     snapshotToTime(data, "created_at"),
	}
	if labelsStr := mapval.String(data, "labels"); labelsStr != "" {
		_ = json.Unmarshal([]byte(labelsStr), &m.Labels)
	}
	if raStr := mapval.String(data, "resource_attributes"); raStr != "" {
		_ = json.Unmarshal([]byte(raStr), &m.ResourceAttributes)
	}
	return m
}
