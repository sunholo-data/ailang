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

// --- Span operations ---

func (s *ObservatoryStore) CreateSpan(ctx context.Context, span *obs.Span) error {
	if span.CreatedAt.IsZero() {
		span.CreatedAt = time.Now()
	}
	_, err := s.client.Doc(collObsSpans, span.ID).Set(ctx, spanToMap(span, s.spanTTL))
	return err
}

func (s *ObservatoryStore) GetSpan(ctx context.Context, id string) (*obs.Span, error) {
	doc, err := s.client.Doc(collObsSpans, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("span not found: %s", id)
		}
		return nil, err
	}
	return mapToSpan(doc.Data()), nil
}

func (s *ObservatoryStore) ListSpans(ctx context.Context, opts obs.SpanListOptions) ([]*obs.Span, error) {
	q := s.client.Collection(collObsSpans).Query
	if opts.TraceID != "" {
		q = q.Where("trace_id", "==", opts.TraceID)
	}
	if opts.TaskID != "" {
		q = q.Where("task_id", "==", opts.TaskID)
	}
	if opts.AgentAssignmentID != "" {
		q = q.Where("agent_assignment_id", "==", opts.AgentAssignmentID)
	}
	if opts.Provider != "" {
		q = q.Where("provider", "==", opts.Provider)
	}
	if opts.Model != "" {
		q = q.Where("model", "==", opts.Model)
	}
	if opts.Status != "" {
		q = q.Where("status", "==", opts.Status)
	}
	if !opts.StartAfter.IsZero() {
		q = q.Where("start_time", ">=", timeToFirestore(opts.StartAfter))
	}
	if !opts.StartBefore.IsZero() {
		q = q.Where("start_time", "<=", timeToFirestore(opts.StartBefore))
	}
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
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

func (s *ObservatoryStore) UpdateSpan(ctx context.Context, span *obs.Span) error {
	_, err := s.client.Doc(collObsSpans, span.ID).Set(ctx, spanToMap(span, s.spanTTL))
	return err
}

func (s *ObservatoryStore) UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error {
	_, err := s.client.Doc(collObsSpans, spanID).Update(ctx, []firestore.Update{
		{Path: "task_id", Value: taskID},
		{Path: "agent_assignment_id", Value: assignmentID},
	})
	return err
}

func (s *ObservatoryStore) RecalculateTaskAggregates(ctx context.Context, taskID string) error {
	iter := s.client.Collection(collObsSpans).
		Where("task_id", "==", taskID).
		Documents(ctx)
	defer iter.Stop()

	var totalDur int64
	var tokIn, tokOut int64
	var cost float64
	var spanCount, errorCount int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		data := doc.Data()
		spanCount++
		totalDur += getInt64(data, "duration_ms")
		tokIn += getInt64(data, "tokens_in")
		tokOut += getInt64(data, "tokens_out")
		cost += getFloat64(data, "cost_usd")
		if getString(data, "status") == "error" {
			errorCount++
		}
	}

	_, err := s.client.Doc(collObsTasks, taskID).Update(ctx, []firestore.Update{
		{Path: "total_duration_ms", Value: totalDur},
		{Path: "total_tokens_in", Value: tokIn},
		{Path: "total_tokens_out", Value: tokOut},
		{Path: "total_cost_usd", Value: cost},
		{Path: "span_count", Value: spanCount},
		{Path: "error_count", Value: errorCount},
	})
	return err
}

func (s *ObservatoryStore) DeleteSpan(ctx context.Context, id string) error {
	_, err := s.client.Doc(collObsSpans, id).Delete(ctx)
	return err
}

func (s *ObservatoryStore) GetTrace(ctx context.Context, traceID string) (*obs.Trace, error) {
	spans, err := s.ListSpans(ctx, obs.SpanListOptions{TraceID: traceID, Limit: 10000})
	if err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	trace := &obs.Trace{
		TraceID:   traceID,
		Spans:     spans,
		SpanCount: len(spans),
		StartTime: spans[0].StartTime,
	}
	for _, sp := range spans {
		if sp.ParentSpanID == "" {
			trace.RootSpan = sp
		}
		if sp.EndTime != nil && sp.EndTime.After(trace.EndTime) {
			trace.EndTime = *sp.EndTime
		}
	}
	if !trace.EndTime.IsZero() {
		trace.DurationMs = trace.EndTime.Sub(trace.StartTime).Milliseconds()
	}
	return trace, nil
}

func (s *ObservatoryStore) ListTraces(ctx context.Context, opts obs.TraceQuery) ([]*obs.TraceSummary, error) {
	q := s.client.Collection(collObsSpans).
		Where("parent_span_id", "==", "")

	if opts.TaskID != "" {
		q = q.Where("task_id", "==", opts.TaskID)
	}
	q = q.OrderBy("start_time", firestore.Desc)
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var result []*obs.TraceSummary
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		result = append(result, &obs.TraceSummary{
			TraceID:    getString(data, "trace_id"),
			RootSpan:   getString(data, "name"),
			DurationMs: getInt64(data, "duration_ms"),
			StartTime:  snapshotToTime(data, "start_time"),
			Status:     obs.SpanStatus(getString(data, "status")),
			TaskID:     getString(data, "task_id"),
		})
	}
	return result, nil
}

func (s *ObservatoryStore) LookupTaskBySessionID(ctx context.Context, sessionID string) (taskID, assignmentID, traceID string) {
	// Check sessions table first
	doc, err := s.client.Doc(collObsSessions, sessionID).Get(ctx)
	if err == nil {
		data := doc.Data()
		if tid := getString(data, "task_id"); tid != "" {
			taskID = tid
		}
	}

	// Look for spans with this session.id in attributes
	iter := s.client.Collection(collObsSpans).
		Where("session_id", "==", sessionID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()
	if d, err := iter.Next(); err == nil {
		data := d.Data()
		if taskID == "" {
			taskID = getString(data, "task_id")
		}
		assignmentID = getString(data, "agent_assignment_id")
		traceID = getString(data, "trace_id")
	}
	return
}

func (s *ObservatoryStore) LinkOrphanedSpansBySession(ctx context.Context, sessionID, taskID, assignmentID string) (int64, error) {
	iter := s.client.Collection(collObsSpans).
		Where("session_id", "==", sessionID).
		Where("task_id", "==", "").
		Documents(ctx)
	defer iter.Stop()

	var count int64
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		updates := []firestore.Update{{Path: "task_id", Value: taskID}}
		if assignmentID != "" {
			updates = append(updates, firestore.Update{Path: "agent_assignment_id", Value: assignmentID})
		}
		if _, err := doc.Ref.Update(ctx, updates); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// --- Session operations ---

func (s *ObservatoryStore) GetSessionWorkspace(sessionID string) (string, error) {
	doc, err := s.client.Doc(collObsSessions, sessionID).Get(context.Background())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", err
	}
	return getString(doc.Data(), "workspace"), nil
}

func (s *ObservatoryStore) UpsertSession(ctx context.Context, sessionID, workspace, version, source string) error {
	return s.UpsertSessionWithCorrelation(ctx, sessionID, workspace, version, source, nil)
}

func (s *ObservatoryStore) UpsertSessionWithCorrelation(ctx context.Context, sessionID, workspace, version, source string, corr *obs.SessionCorrelation) error {
	data := map[string]interface{}{
		"session_id":     sessionID,
		"workspace":      workspace,
		"claude_version": version,
		"source":         source,
		"started_at":     time.Now(),
	}
	if corr != nil {
		data["task_id"] = corr.TaskID
		data["chain_id"] = corr.ChainID
		data["stage_id"] = corr.StageID
		data["message_id"] = corr.MessageID
	}
	_, err := s.client.Doc(collObsSessions, sessionID).Set(ctx, data, firestore.MergeAll)
	return err
}

func (s *ObservatoryStore) UpdateSessionEnded(ctx context.Context, sessionID string) error {
	_, err := s.client.Doc(collObsSessions, sessionID).Update(ctx, []firestore.Update{
		{Path: "ended_at", Value: time.Now()},
	})
	return err
}

func (s *ObservatoryStore) InsertToolStart(ctx context.Context, sessionID, toolUseID, toolName, toolInput string) error {
	now := time.Now()
	_, err := s.client.Doc(collObsSessionTools, toolUseID).Set(ctx, map[string]interface{}{
		"tool_use_id": toolUseID,
		"session_id":  sessionID,
		"tool_name":   toolName,
		"tool_input":  toolInput,
		"start_time":  now,
		"expire_at":   now.Add(s.spanTTL),
	})
	return err
}

func (s *ObservatoryStore) FindLatestUnfinishedTool(ctx context.Context, sessionID, toolName string) (string, error) {
	q := s.client.Collection(collObsSessionTools).
		Where("session_id", "==", sessionID).
		Where("tool_name", "==", toolName).
		OrderBy("start_time", firestore.Desc).
		Limit(10)

	iter := q.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		data := doc.Data()
		// Find one without end_time
		if _, ok := data["end_time"]; !ok || data["end_time"] == nil {
			return getString(data, "tool_use_id"), nil
		}
	}
	return "", fmt.Errorf("no unfinished tool call found for session %s, tool %s", sessionID, toolName)
}

func (s *ObservatoryStore) UpdateToolEnd(ctx context.Context, toolUseID, toolResponse string, success bool) error {
	_, err := s.client.Doc(collObsSessionTools, toolUseID).Update(ctx, []firestore.Update{
		{Path: "tool_response", Value: toolResponse},
		{Path: "end_time", Value: time.Now()},
		{Path: "success", Value: success},
	})
	return err
}

func (s *ObservatoryStore) GetToolForSpan(ctx context.Context, sessionID, toolName string, spanTime time.Time) (*obs.SessionTool, error) {
	q := s.client.Collection(collObsSessionTools).
		Where("session_id", "==", sessionID).
		Where("tool_name", "==", toolName).
		OrderBy("start_time", firestore.Desc).
		Limit(5)

	iter := q.Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		st := snapshotToTime(data, "start_time")
		// Find tool closest to spanTime
		if !st.After(spanTime.Add(5 * time.Second)) {
			tool := &obs.SessionTool{
				ToolUseID: getString(data, "tool_use_id"),
				SessionID: getString(data, "session_id"),
				ToolName:  getString(data, "tool_name"),
				StartTime: st,
			}
			if input := getString(data, "tool_input"); input != "" {
				tool.ToolInput = json.RawMessage(input)
			}
			if resp := getString(data, "tool_response"); resp != "" {
				tool.ToolResponse = json.RawMessage(resp)
			}
			tool.EndTime = snapshotToTimePtr(data, "end_time")
			if v, ok := data["success"]; ok && v != nil {
				b := getBool(data, "success")
				tool.Success = &b
			}
			return tool, nil
		}
	}
	return nil, fmt.Errorf("no matching tool found")
}

func (s *ObservatoryStore) BackfillSpansWorkspace(ctx context.Context, sessionID, workspace string) (int64, error) {
	// Update spans that have this session_id but empty workspace
	iter := s.client.Collection(collObsSpans).
		Where("session_id", "==", sessionID).
		Documents(ctx)
	defer iter.Stop()

	var count int64
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return count, err
		}
		if getString(doc.Data(), "workspace") == "" {
			if _, err := doc.Ref.Update(ctx, []firestore.Update{
				{Path: "workspace", Value: workspace},
			}); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

// --- Span Event operations ---

func (s *ObservatoryStore) CreateSpanEvent(ctx context.Context, e *obs.SpanEvent) error {
	id := fmt.Sprintf("spevt_%d_%s", time.Now().UnixMilli(), generateShortID())
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	data := map[string]interface{}{
		"span_id":         e.SpanID,
		"name":            e.Name,
		"timestamp":       timeToFirestore(e.Timestamp),
		"event_type":      string(e.EventType),
		"approval_status": string(e.ApprovalStatus),
		"tool_name":       e.ToolName,
		"error_message":   e.ErrorMessage,
	}
	if e.Attributes != nil {
		if b, err := json.Marshal(e.Attributes); err == nil {
			data["attributes"] = string(b)
		}
	}
	_, err := s.client.Doc(collObsSpanEvents, id).Set(ctx, data)
	return err
}

func (s *ObservatoryStore) GetSpanEvents(ctx context.Context, spanID string) ([]obs.SpanEvent, error) {
	iter := s.client.Collection(collObsSpanEvents).
		Where("span_id", "==", spanID).
		OrderBy("timestamp", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var result []obs.SpanEvent
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		e := obs.SpanEvent{
			SpanID:         getString(data, "span_id"),
			Name:           getString(data, "name"),
			Timestamp:      snapshotToTime(data, "timestamp"),
			EventType:      obs.EventType(getString(data, "event_type")),
			ApprovalStatus: obs.ApprovalStatus(getString(data, "approval_status")),
			ToolName:       getString(data, "tool_name"),
			ErrorMessage:   getString(data, "error_message"),
		}
		if attrStr := getString(data, "attributes"); attrStr != "" {
			_ = json.Unmarshal([]byte(attrStr), &e.Attributes)
		}
		result = append(result, e)
	}
	return result, nil
}

func (s *ObservatoryStore) DeleteSpanEvent(ctx context.Context, id int64) error {
	// Firestore uses string IDs; convert int64 to string for lookup
	docID := fmt.Sprintf("spevt_%d", id)
	_, err := s.client.Doc(collObsSpanEvents, docID).Delete(ctx)
	return err
}

// --- Span conversion helpers ---

func spanToMap(sp *obs.Span, ttl time.Duration) map[string]interface{} {
	m := map[string]interface{}{
		"id":                    sp.ID,
		"trace_id":              sp.TraceID,
		"parent_span_id":        sp.ParentSpanID,
		"task_id":               sp.TaskID,
		"agent_assignment_id":   sp.AgentAssignmentID,
		"chain_id":              sp.ChainID,
		"stage_id":              sp.StageID,
		"name":                  sp.Name,
		"kind":                  string(sp.Kind),
		"status":                string(sp.Status),
		"status_message":        sp.StatusMessage,
		"start_time":            timeToFirestore(sp.StartTime),
		"end_time":              timePtrToFirestore(sp.EndTime),
		"duration_ms":           sp.DurationMs,
		"tokens_in":             sp.TokensIn,
		"tokens_out":            sp.TokensOut,
		"cache_read_tokens":     sp.CacheReadTokens,
		"cache_creation_tokens": sp.CacheCreationTokens,
		"cost_usd":              sp.CostUSD,
		"model":                 sp.Model,
		"provider":              string(sp.Provider),
		"created_at":            timeToFirestore(sp.CreatedAt),
		"expire_at":             timeToFirestore(sp.CreatedAt.Add(ttl)),
	}

	// Store session_id at top level for efficient queries
	if sp.Attributes != nil {
		if sid, ok := sp.Attributes["session.id"]; ok {
			m["session_id"] = sid
		}
		if ws, ok := sp.Attributes["workspace"]; ok {
			m["workspace"] = ws
		}
	}

	// Store attributes and resource_attributes as JSON strings
	if sp.Attributes != nil {
		if b, err := json.Marshal(sp.Attributes); err == nil {
			m["attributes"] = string(b)
		}
	}
	if sp.ResourceAttributes != nil {
		if b, err := json.Marshal(sp.ResourceAttributes); err == nil {
			m["resource_attributes"] = string(b)
		}
	}
	return m
}

func mapToSpan(data map[string]interface{}) *obs.Span {
	sp := &obs.Span{
		ID:                  getString(data, "id"),
		TraceID:             getString(data, "trace_id"),
		ParentSpanID:        getString(data, "parent_span_id"),
		TaskID:              getString(data, "task_id"),
		AgentAssignmentID:   getString(data, "agent_assignment_id"),
		ChainID:             getString(data, "chain_id"),
		StageID:             getString(data, "stage_id"),
		Name:                getString(data, "name"),
		Kind:                obs.SpanKind(getString(data, "kind")),
		Status:              obs.SpanStatus(getString(data, "status")),
		StatusMessage:       getString(data, "status_message"),
		StartTime:           snapshotToTime(data, "start_time"),
		EndTime:             snapshotToTimePtr(data, "end_time"),
		DurationMs:          getInt64(data, "duration_ms"),
		TokensIn:            getInt64(data, "tokens_in"),
		TokensOut:           getInt64(data, "tokens_out"),
		CacheReadTokens:     getInt64(data, "cache_read_tokens"),
		CacheCreationTokens: getInt64(data, "cache_creation_tokens"),
		CostUSD:             getFloat64(data, "cost_usd"),
		Model:               getString(data, "model"),
		Provider:            obs.Provider(getString(data, "provider")),
		CreatedAt:           snapshotToTime(data, "created_at"),
	}
	if attrStr := getString(data, "attributes"); attrStr != "" {
		_ = json.Unmarshal([]byte(attrStr), &sp.Attributes)
	}
	if raStr := getString(data, "resource_attributes"); raStr != "" {
		_ = json.Unmarshal([]byte(raStr), &sp.ResourceAttributes)
	}
	return sp
}
