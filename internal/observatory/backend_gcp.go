// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	trace "cloud.google.com/go/trace/apiv1"
	"cloud.google.com/go/trace/apiv1/tracepb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Default cache TTL for GCP trace results (to avoid quota exhaustion)
const defaultGCPCacheTTL = 60 * time.Second

// gcpTraceCache holds cached trace results with TTL
type gcpTraceCache struct {
	mu       sync.RWMutex
	traces   []*TraceSummary
	expiry   time.Time
	cacheKey string // Cache key based on query params
}

// GCPTraceBackend implements Backend using Google Cloud Trace.
// This is a read-only backend for querying traces stored in GCP.
// Results are cached for CacheTTL duration to avoid quota exhaustion.
type GCPTraceBackend struct {
	projectID string
	client    *trace.Client
	cacheTTL  time.Duration
	cache     gcpTraceCache
}

// GCPConfig contains configuration for GCP Trace backend.
type GCPConfig struct {
	ProjectID       string
	CredentialsPath string        // optional - uses default credentials if empty
	CacheTTL        time.Duration // Cache TTL for trace results (default: 60s)
}

// NewGCPTraceBackend creates a new GCP Trace backend.
func NewGCPTraceBackend(config GCPConfig) (*GCPTraceBackend, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}

	ctx := context.Background()
	client, err := trace.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create GCP trace client: %w", err)
	}

	cacheTTL := config.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = defaultGCPCacheTTL
	}

	return &GCPTraceBackend{
		projectID: config.ProjectID,
		client:    client,
		cacheTTL:  cacheTTL,
	}, nil
}

// Close closes the GCP client.
func (b *GCPTraceBackend) Close() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

// errNotSupported returns an error for operations not supported by this backend.
func errNotSupported(op string) error {
	return fmt.Errorf("operation %s not supported by GCP Trace backend (read-only)", op)
}

// ===== Span Operations (read-only supported) =====

func (b *GCPTraceBackend) CreateSpan(ctx context.Context, span *Span) error {
	return errNotSupported("CreateSpan")
}

func (b *GCPTraceBackend) GetSpan(ctx context.Context, id string) (*Span, error) {
	// GCP Trace API doesn't support fetching individual spans directly
	// Need to fetch the trace and find the span within it
	return nil, fmt.Errorf("GetSpan requires trace_id; use GetTrace instead")
}

func (b *GCPTraceBackend) ListSpans(ctx context.Context, opts SpanListOptions) ([]*Span, error) {
	// List all traces and flatten to spans
	// This is expensive but necessary since GCP doesn't have a span-level API
	query := TraceQuery{
		Limit: opts.Limit,
	}

	// Convert SpanListOptions time filters to TraceQuery TimeRange
	if !opts.StartAfter.IsZero() || !opts.StartBefore.IsZero() {
		query.TimeRange = &TimeRange{}
		if !opts.StartAfter.IsZero() {
			query.TimeRange.Start = opts.StartAfter
		}
		if !opts.StartBefore.IsZero() {
			query.TimeRange.End = opts.StartBefore
		}
	}

	traces, err := b.ListTraces(ctx, query)
	if err != nil {
		return nil, err
	}

	var spans []*Span
	for _, summary := range traces {
		traceData, err := b.GetTrace(ctx, summary.TraceID)
		if err != nil {
			continue // Skip traces we can't fetch
		}
		spans = append(spans, traceData.Spans...)
		if opts.Limit > 0 && len(spans) >= opts.Limit {
			break
		}
	}

	if opts.Limit > 0 && len(spans) > opts.Limit {
		spans = spans[:opts.Limit]
	}

	return spans, nil
}

func (b *GCPTraceBackend) UpdateSpan(ctx context.Context, span *Span) error {
	return errNotSupported("UpdateSpan")
}

func (b *GCPTraceBackend) UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error {
	return errNotSupported("UpdateSpanLinks")
}

func (b *GCPTraceBackend) RecalculateTaskAggregates(ctx context.Context, taskID string) error {
	return errNotSupported("RecalculateTaskAggregates")
}

func (b *GCPTraceBackend) DeleteSpan(ctx context.Context, id string) error {
	return errNotSupported("DeleteSpan")
}

func (b *GCPTraceBackend) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	req := &tracepb.GetTraceRequest{
		ProjectId: b.projectID,
		TraceId:   traceID,
	}

	resp, err := b.client.GetTrace(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get trace from GCP: %w", err)
	}

	return b.convertGCPTrace(resp), nil
}

func (b *GCPTraceBackend) ListTraces(ctx context.Context, opts TraceQuery) ([]*TraceSummary, error) {
	// Build cache key from query params
	cacheKey := fmt.Sprintf("list:%d:%s", opts.Limit, opts.TraceID)

	// Check cache first
	b.cache.mu.RLock()
	if b.cache.cacheKey == cacheKey && time.Now().Before(b.cache.expiry) {
		traces := b.cache.traces
		b.cache.mu.RUnlock()
		return traces, nil
	}
	b.cache.mu.RUnlock()

	// Extract time range from opts
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour) // Default: last hour

	if opts.TimeRange != nil {
		if !opts.TimeRange.End.IsZero() {
			endTime = opts.TimeRange.End
		}
		if !opts.TimeRange.Start.IsZero() {
			startTime = opts.TimeRange.Start
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	req := &tracepb.ListTracesRequest{
		ProjectId: b.projectID,
		View:      tracepb.ListTracesRequest_ROOTSPAN,
		StartTime: timestampProto(startTime),
		EndTime:   timestampProto(endTime),
		PageSize:  int32(limit),
	}

	// Filter by trace ID if specified
	if opts.TraceID != "" {
		req.Filter = fmt.Sprintf("traceId:%s", opts.TraceID)
	}

	it := b.client.ListTraces(ctx, req)

	var summaries []*TraceSummary
	count := 0

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list traces from GCP: %w", err)
		}

		// Skip internal OTEL exporter traces
		if len(resp.Spans) > 0 {
			rootName := resp.Spans[0].Name
			if isInternalTrace(rootName) {
				continue
			}
		}

		count++
		if count > limit {
			break
		}

		summary := b.convertGCPTraceSummary(resp)
		summaries = append(summaries, summary)
	}

	// Update cache
	b.cache.mu.Lock()
	b.cache.traces = summaries
	b.cache.expiry = time.Now().Add(b.cacheTTL)
	b.cache.cacheKey = cacheKey
	b.cache.mu.Unlock()

	return summaries, nil
}

// convertGCPTrace converts a GCP Trace to our Trace model.
func (b *GCPTraceBackend) convertGCPTrace(gcpTrace *tracepb.Trace) *Trace {
	if gcpTrace == nil {
		return nil
	}

	result := &Trace{
		TraceID: gcpTrace.TraceId,
		Spans:   make([]*Span, 0, len(gcpTrace.Spans)),
	}

	// Build span lookup for hierarchy
	spanByID := make(map[uint64]*Span)

	// Convert all spans first
	for _, gcpSpan := range gcpTrace.Spans {
		span := b.convertGCPSpan(gcpTrace.TraceId, gcpSpan)
		spanByID[gcpSpan.SpanId] = span
		result.Spans = append(result.Spans, span)
	}

	// Build parent-child relationships
	for _, gcpSpan := range gcpTrace.Spans {
		span := spanByID[gcpSpan.SpanId]
		if gcpSpan.ParentSpanId != 0 {
			if parent, ok := spanByID[gcpSpan.ParentSpanId]; ok {
				parent.Children = append(parent.Children, span)
			}
		}
	}

	// Find root span for trace metadata
	for _, gcpSpan := range gcpTrace.Spans {
		if gcpSpan.ParentSpanId == 0 {
			span := spanByID[gcpSpan.SpanId]
			result.RootSpan = span
			result.StartTime = span.StartTime
			if span.EndTime != nil {
				result.EndTime = *span.EndTime
				result.DurationMs = span.DurationMs
			}
			break
		}
	}

	result.SpanCount = len(result.Spans)
	return result
}

// convertGCPSpan converts a GCP TraceSpan to our Span model.
func (b *GCPTraceBackend) convertGCPSpan(traceID string, gcpSpan *tracepb.TraceSpan) *Span {
	if gcpSpan == nil {
		return nil
	}

	span := &Span{
		ID:         fmt.Sprintf("%x", gcpSpan.SpanId),
		TraceID:    traceID,
		Name:       gcpSpan.Name,
		Kind:       convertGCPSpanKind(gcpSpan.Kind),
		StartTime:  gcpSpan.StartTime.AsTime(),
		Attributes: make(map[string]any),
		CreatedAt:  time.Now(),
	}

	// Set parent span ID
	if gcpSpan.ParentSpanId != 0 {
		span.ParentSpanID = fmt.Sprintf("%x", gcpSpan.ParentSpanId)
	}

	// Calculate duration
	if gcpSpan.EndTime != nil {
		endTime := gcpSpan.EndTime.AsTime()
		span.EndTime = &endTime
		span.DurationMs = endTime.Sub(span.StartTime).Milliseconds()
	}

	// Convert labels to attributes and extract normalized fields
	if gcpSpan.Labels != nil {
		for k, v := range gcpSpan.Labels {
			span.Attributes[k] = v

			// Extract normalized metrics from Gemini CLI labels
			switch k {
			case "gen_ai.usage.input_tokens", "input_tokens":
				if tokens, err := strconv.ParseInt(v, 10, 64); err == nil {
					span.TokensIn = tokens
				}
			case "gen_ai.usage.output_tokens", "output_tokens":
				if tokens, err := strconv.ParseInt(v, 10, 64); err == nil {
					span.TokensOut = tokens
				}
			case "gen_ai.request.model", "model":
				span.Model = v
			case "gen_ai.system", "provider":
				span.Provider = Provider(v)
			case "service.name":
				// Detect provider from service name
				if strings.Contains(v, "gemini") {
					span.Provider = ProviderGemini
				} else if strings.Contains(v, "claude") {
					span.Provider = ProviderClaude
				}
				span.ResourceAttributes = map[string]any{"service.name": v}
			case "session.id":
				span.Attributes["session.id"] = v
			case "ailang.task_id", "task_id":
				span.TaskID = v
			}
		}
	}

	// Default provider to Gemini for GCP traces (since that's our use case)
	if span.Provider == "" {
		span.Provider = ProviderGemini
	}

	// Set status (GCP v1 API doesn't have status, use heuristics)
	span.Status = SpanStatusOK
	if _, hasError := span.Attributes["error"]; hasError {
		span.Status = SpanStatusError
	}

	return span
}

// convertGCPTraceSummary converts a GCP Trace to a TraceSummary.
func (b *GCPTraceBackend) convertGCPTraceSummary(gcpTrace *tracepb.Trace) *TraceSummary {
	summary := &TraceSummary{
		TraceID:   gcpTrace.TraceId,
		SpanCount: len(gcpTrace.Spans),
		Status:    SpanStatusOK,
		Source:    TraceSourceGCP, // Mark as coming from GCP
	}

	if len(gcpTrace.Spans) > 0 {
		rootSpan := gcpTrace.Spans[0]
		summary.RootSpan = rootSpan.Name
		summary.StartTime = rootSpan.StartTime.AsTime()

		if rootSpan.EndTime != nil {
			duration := rootSpan.EndTime.AsTime().Sub(rootSpan.StartTime.AsTime())
			summary.DurationMs = duration.Milliseconds()
		}

		// Extract service name from labels
		if labels := rootSpan.Labels; labels != nil {
			if serviceName, ok := labels["service.name"]; ok {
				summary.ServiceName = serviceName
			}
		}
	}

	// Default service name for GCP traces (likely from Gemini CLI)
	if summary.ServiceName == "" {
		summary.ServiceName = "gemini-cli"
	}

	return summary
}

// convertGCPSpanKind converts GCP span kind to our SpanKind.
func convertGCPSpanKind(kind tracepb.TraceSpan_SpanKind) SpanKind {
	switch kind {
	case tracepb.TraceSpan_RPC_CLIENT:
		return SpanKindClient
	case tracepb.TraceSpan_RPC_SERVER:
		return SpanKindServer
	default:
		return SpanKindInternal
	}
}

// isInternalTrace returns true for OTEL exporter internal traces that should be hidden.
func isInternalTrace(name string) bool {
	if strings.HasPrefix(name, "google.devtools.cloudtrace") {
		return true
	}
	if strings.HasPrefix(name, "opentelemetry.") {
		return true
	}
	if name == "/health" || name == "health.check" {
		return true
	}
	return false
}

// timestampProto converts time.Time to protobuf Timestamp.
func timestampProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// ===== Workspace Operations (not supported) =====

func (b *GCPTraceBackend) CreateWorkspace(ctx context.Context, w *Workspace) error {
	return errNotSupported("CreateWorkspace")
}

func (b *GCPTraceBackend) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	return nil, errNotSupported("GetWorkspace")
}

func (b *GCPTraceBackend) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	return nil, errNotSupported("ListWorkspaces")
}

func (b *GCPTraceBackend) UpdateWorkspace(ctx context.Context, w *Workspace) error {
	return errNotSupported("UpdateWorkspace")
}

func (b *GCPTraceBackend) DeleteWorkspace(ctx context.Context, id string) error {
	return errNotSupported("DeleteWorkspace")
}

func (b *GCPTraceBackend) GetWorkspaceStats(ctx context.Context, id string) (*WorkspaceStats, error) {
	return nil, errNotSupported("GetWorkspaceStats")
}

// ===== Task Operations (not supported) =====

func (b *GCPTraceBackend) CreateTask(ctx context.Context, t *Task) error {
	return errNotSupported("CreateTask")
}

func (b *GCPTraceBackend) GetTask(ctx context.Context, id string) (*Task, error) {
	return nil, errNotSupported("GetTask")
}

func (b *GCPTraceBackend) ListTasks(ctx context.Context, opts TaskListOptions) ([]*Task, error) {
	return nil, errNotSupported("ListTasks")
}

func (b *GCPTraceBackend) UpdateTask(ctx context.Context, t *Task) error {
	return errNotSupported("UpdateTask")
}

func (b *GCPTraceBackend) DeleteTask(ctx context.Context, id string) error {
	return errNotSupported("DeleteTask")
}

// ===== Agent Assignment Operations (not supported) =====

func (b *GCPTraceBackend) CreateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return errNotSupported("CreateAgentAssignment")
}

func (b *GCPTraceBackend) GetAgentAssignment(ctx context.Context, id string) (*AgentAssignment, error) {
	return nil, errNotSupported("GetAgentAssignment")
}

func (b *GCPTraceBackend) ListAgentAssignments(ctx context.Context, taskID string) ([]*AgentAssignment, error) {
	return nil, errNotSupported("ListAgentAssignments")
}

func (b *GCPTraceBackend) UpdateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return errNotSupported("UpdateAgentAssignment")
}

func (b *GCPTraceBackend) DeleteAgentAssignment(ctx context.Context, id string) error {
	return errNotSupported("DeleteAgentAssignment")
}

func (b *GCPTraceBackend) GetAgentStats(ctx context.Context, agentID string) (*AgentStats, error) {
	return nil, errNotSupported("GetAgentStats")
}

// ===== Span Event Operations (not supported) =====

func (b *GCPTraceBackend) CreateSpanEvent(ctx context.Context, e *SpanEvent) error {
	return errNotSupported("CreateSpanEvent")
}

func (b *GCPTraceBackend) GetSpanEvents(ctx context.Context, spanID string) ([]SpanEvent, error) {
	return nil, errNotSupported("GetSpanEvents")
}

func (b *GCPTraceBackend) DeleteSpanEvent(ctx context.Context, id int64) error {
	return errNotSupported("DeleteSpanEvent")
}

// ===== Message Operations (not supported) =====

func (b *GCPTraceBackend) CreateMessage(ctx context.Context, m *Message) error {
	return errNotSupported("CreateMessage")
}

func (b *GCPTraceBackend) GetMessage(ctx context.Context, id string) (*Message, error) {
	return nil, errNotSupported("GetMessage")
}

func (b *GCPTraceBackend) ListMessages(ctx context.Context, opts MessageListOptions) ([]*Message, error) {
	return nil, errNotSupported("ListMessages")
}

func (b *GCPTraceBackend) UpdateMessage(ctx context.Context, m *Message) error {
	return errNotSupported("UpdateMessage")
}

func (b *GCPTraceBackend) DeleteMessage(ctx context.Context, id string) error {
	return errNotSupported("DeleteMessage")
}

func (b *GCPTraceBackend) MarkMessageRead(ctx context.Context, id string) error {
	return errNotSupported("MarkMessageRead")
}

func (b *GCPTraceBackend) MarkMessageArchived(ctx context.Context, id string) error {
	return errNotSupported("MarkMessageArchived")
}

// ===== Aggregate Operations (not supported) =====

func (b *GCPTraceBackend) GetMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	return nil, errNotSupported("GetMetricsSummary")
}

func (b *GCPTraceBackend) GetProviderComparison(ctx context.Context) ([]*ProviderComparison, error) {
	return nil, errNotSupported("GetProviderComparison")
}

func (b *GCPTraceBackend) GetTaskTimeline(ctx context.Context, taskID string) ([]*TaskTimeline, error) {
	return nil, errNotSupported("GetTaskTimeline")
}

func (b *GCPTraceBackend) GetExecTaskHierarchy(ctx context.Context, limit int) ([]*ExecTaskNode, error) {
	return nil, errNotSupported("GetExecTaskHierarchy")
}

// LookupTaskBySessionID is not supported by GCP backend (session correlation is local only).
func (b *GCPTraceBackend) LookupTaskBySessionID(ctx context.Context, sessionID string) (string, string, string) {
	return "", "", ""
}

// LinkOrphanedSpansBySession is not supported by GCP backend (session correlation is local only).
func (b *GCPTraceBackend) LinkOrphanedSpansBySession(ctx context.Context, sessionID, taskID, assignmentID string) (int64, error) {
	return 0, nil
}

// Session operations - not supported by GCP backend (session tracking is local only).

func (b *GCPTraceBackend) GetSessionWorkspace(sessionID string) (string, error) {
	return "", nil
}

func (b *GCPTraceBackend) UpsertSession(ctx context.Context, sessionID, workspace, version, source string) error {
	return nil
}

func (b *GCPTraceBackend) UpdateSessionEnded(ctx context.Context, sessionID string) error {
	return nil
}

func (b *GCPTraceBackend) InsertToolStart(ctx context.Context, sessionID, toolUseID, toolName, toolInput string) error {
	return nil
}

func (b *GCPTraceBackend) UpdateToolEnd(ctx context.Context, toolUseID, toolResponse string, success bool) error {
	return nil
}

func (b *GCPTraceBackend) FindLatestUnfinishedTool(ctx context.Context, sessionID, toolName string) (string, error) {
	return "", nil // Not supported by GCP backend
}

func (b *GCPTraceBackend) BackfillSpansWorkspace(ctx context.Context, sessionID, workspace string) (int64, error) {
	return 0, nil
}

func (b *GCPTraceBackend) GetToolForSpan(ctx context.Context, sessionID, toolName string, spanTime time.Time) (*SessionTool, error) {
	return nil, nil // Not supported by GCP backend
}

// Ensure GCPTraceBackend implements Backend
var _ Backend = (*GCPTraceBackend)(nil)
