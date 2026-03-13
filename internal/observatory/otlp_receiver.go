// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// backgroundOperationSpans contains span names that should NOT inherit task_id from CWD.
// These are coordinator background operations that may run in a worktree directory
// but are not part of the task execution. Without this filter, operations like
// messages.list or github_sync that run in a worktree directory would incorrectly
// get tagged with the task ID, contaminating the task hierarchy.
var backgroundOperationSpans = map[string]bool{
	"messages.list":          true,
	"messages.github_sync":   true,
	"messages.ack":           true,
	"messages.send":          true,
	"messages.search":        true,
	"messages.import-github": true,
}

// FilterPattern defines a single span filter rule.
type FilterPattern struct {
	Type    string // "prefix", "exact", "suffix"
	Pattern string // The pattern to match against span name
	Service string // Optional: only match spans from this service (service.name)
}

// SpanFilterConfig holds configurable span filtering rules.
// Loaded once at startup from environment variables; immutable after creation.
type SpanFilterConfig struct {
	AllowPatterns []FilterPattern // Allow takes priority — matching spans are ALWAYS kept
	DenyPatterns  []FilterPattern // Matching spans are dropped (unless allow-listed)
	DisableAll    bool            // Bypass all filtering (debug mode)
}

// DefaultSpanFilterConfig returns the default deny rules matching the original hard-coded behavior.
func DefaultSpanFilterConfig() *SpanFilterConfig {
	return &SpanFilterConfig{
		DenyPatterns: []FilterPattern{
			// GCP Trace exporter internals
			{Type: "prefix", Pattern: "google.devtools.cloudtrace"},
			// OTEL SDK internals
			{Type: "prefix", Pattern: "opentelemetry."},
			// Health checks
			{Type: "exact", Pattern: "/health"},
			{Type: "exact", Pattern: "health.check"},
			{Type: "exact", Pattern: "/api/health"},
			// Static assets
			{Type: "prefix", Pattern: "/assets/"},
			{Type: "suffix", Pattern: ".js"},
			{Type: "suffix", Pattern: ".css"},
			{Type: "suffix", Pattern: ".png"},
			{Type: "suffix", Pattern: ".ico"},
			{Type: "suffix", Pattern: ".svg"},
			// UI polling endpoints
			{Type: "prefix", Pattern: "/api/approvals"},
			{Type: "prefix", Pattern: "/api/hierarchy"},
			{Type: "prefix", Pattern: "/api/statistics"},
			{Type: "prefix", Pattern: "/api/version"},
			{Type: "prefix", Pattern: "/api/monitor"},
			{Type: "prefix", Pattern: "/api/telemetry/config"},
			{Type: "prefix", Pattern: "/api/metrics"},
			{Type: "prefix", Pattern: "/api/observatory/traces"},
			{Type: "prefix", Pattern: "/api/observatory/metrics"},
			{Type: "prefix", Pattern: "/api/inbox"},
			{Type: "prefix", Pattern: "/api/budget/"},
			// Control plane polling
			{Type: "prefix", Pattern: "/api/controlplane/"},
			// Coordinator events SSE
			{Type: "exact", Pattern: "/api/coordinator/events"},
			// Service-scoped: coordinator polling
			{Type: "exact", Pattern: "messages.list", Service: "ailang-coordinator"},
			{Type: "exact", Pattern: "messages.count", Service: "ailang-coordinator"},
			{Type: "exact", Pattern: "inbox.poll", Service: "ailang-coordinator"},
			{Type: "exact", Pattern: "agent.heartbeat", Service: "ailang-coordinator"},
			// Service-scoped: server polling
			{Type: "exact", Pattern: "messages.list", Service: "ailang-server"},
			{Type: "exact", Pattern: "messages.count", Service: "ailang-server"},
		},
	}
}

// parseFilterPattern parses a single pattern string into a FilterPattern.
// Formats: "name" (exact), "name*" (prefix), "*name" (suffix), "service:name" (service-scoped).
func parseFilterPattern(raw string) FilterPattern {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FilterPattern{}
	}

	// Check for service-scoped pattern: "service:pattern"
	var service string
	if idx := strings.Index(raw, ":"); idx > 0 && !strings.HasPrefix(raw, "/") {
		service = raw[:idx]
		raw = raw[idx+1:]
	}

	var patternType, pattern string
	switch {
	case strings.HasPrefix(raw, "*") && strings.HasSuffix(raw, "*"):
		// *contains* — treat as prefix+suffix not supported, use prefix for simplicity
		patternType = "prefix"
		pattern = strings.TrimPrefix(raw, "*")
		pattern = strings.TrimSuffix(pattern, "*")
	case strings.HasSuffix(raw, "*"):
		patternType = "prefix"
		pattern = strings.TrimSuffix(raw, "*")
	case strings.HasPrefix(raw, "*"):
		patternType = "suffix"
		pattern = strings.TrimPrefix(raw, "*")
	default:
		patternType = "exact"
		pattern = raw
	}

	return FilterPattern{Type: patternType, Pattern: pattern, Service: service}
}

// LoadSpanFilterConfig creates a SpanFilterConfig from environment variables,
// merging with defaults. Exported for testing.
func LoadSpanFilterConfig() *SpanFilterConfig {
	config := DefaultSpanFilterConfig()

	if os.Getenv("AILANG_SPAN_FILTER_DISABLE") == "true" {
		config.DisableAll = true
	}

	if allowEnv := os.Getenv("AILANG_SPAN_FILTER_ALLOW"); allowEnv != "" {
		for _, raw := range strings.Split(allowEnv, ",") {
			if p := parseFilterPattern(raw); p.Pattern != "" {
				config.AllowPatterns = append(config.AllowPatterns, p)
			}
		}
	}

	if denyEnv := os.Getenv("AILANG_SPAN_FILTER_DENY"); denyEnv != "" {
		for _, raw := range strings.Split(denyEnv, ",") {
			if p := parseFilterPattern(raw); p.Pattern != "" {
				config.DenyPatterns = append(config.DenyPatterns, p)
			}
		}
	}

	fmt.Printf("observatory: span filter config loaded (allow=%d, deny=%d, disable=%v)\n",
		len(config.AllowPatterns), len(config.DenyPatterns), config.DisableAll)
	return config
}

// OTLPReceiver receives spans via the standard OTLP HTTP protocol.
// This allows any OTEL-compatible exporter to send traces to the observatory.
type OTLPReceiver struct {
	backend      Backend
	filterConfig *SpanFilterConfig
}

// NewOTLPReceiver creates a new OTLP receiver that stores spans in the backend.
func NewOTLPReceiver(backend Backend) *OTLPReceiver {
	return &OTLPReceiver{
		backend:      backend,
		filterConfig: LoadSpanFilterConfig(),
	}
}

// RegisterRoutes registers the OTLP HTTP endpoints on the given mux.
// Implements the OTLP/HTTP specification:
// - POST /v1/traces - Receive trace data (protobuf or JSON)
// - POST /v1/logs - Receive logs/events (for Claude Code events)
// - POST /v1/metrics - Receive metrics (for Claude Code metrics)
func (r *OTLPReceiver) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/traces", r.handleTraces)
	mux.HandleFunc("POST /v1/logs", r.handleLogs)
	mux.HandleFunc("POST /v1/metrics", r.handleMetrics)
}

// handleMetrics handles the OTLP metrics export endpoint.
// Claude Code sends metrics like:
// - claude_code.lines_of_code.count (type: added/removed)
// - claude_code.commit.count
// - claude_code.pull_request.count
// - claude_code.session.count
// - claude_code.active_time.total
// - claude_code.code_edit_tool.decision
func (r *OTLPReceiver) handleMetrics(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var exportReq colmetricspb.ExportMetricsServiceRequest

	// Parse based on content type
	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "application/x-protobuf":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	default:
		// Try protobuf first, fall back to JSON
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := protojson.Unmarshal(body, &exportReq); jsonErr != nil {
				http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
				return
			}
		}
	}

	// Process and store metrics
	ctx := req.Context()
	var storedCount int
	for _, resourceMetrics := range exportReq.ResourceMetrics {
		count, err := r.processResourceMetrics(ctx, resourceMetrics)
		if err != nil {
			fmt.Printf("observatory: failed to process resource metrics: %v\n", err)
		}
		storedCount += count
	}

	if storedCount > 0 {
		fmt.Printf("observatory: stored %d metrics from OTLP\n", storedCount)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"partialSuccess": map[string]any{},
	})
}

// processResourceMetrics processes a single ResourceMetrics message.
func (r *OTLPReceiver) processResourceMetrics(ctx context.Context, rm *metricspb.ResourceMetrics) (int, error) {
	// Extract resource attributes
	resourceAttrs := make(map[string]any)
	if rm.Resource != nil {
		for _, kv := range rm.Resource.Attributes {
			resourceAttrs[kv.Key] = anyValueToGo(kv.Value)
		}
	}

	sessionID := extractString(resourceAttrs, "session.id")
	workspace := extractString(resourceAttrs, "process.cwd")
	provider := extractString(resourceAttrs, "service.name")

	var storedCount int
	for _, scopeMetrics := range rm.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			count, err := r.processMetric(ctx, metric, sessionID, workspace, provider, resourceAttrs)
			if err != nil {
				fmt.Printf("observatory: failed to process metric %s: %v\n", metric.Name, err)
				continue
			}
			storedCount += count
		}
	}
	return storedCount, nil
}

// processMetric processes a single metric and stores it.
func (r *OTLPReceiver) processMetric(ctx context.Context, m *metricspb.Metric, sessionID, workspace, provider string, resourceAttrs map[string]any) (int, error) {
	metricName := m.Name

	// Only capture Claude Code metrics (filter out other sources)
	if !strings.HasPrefix(metricName, "claude_code.") {
		return 0, nil
	}

	switch data := m.Data.(type) {
	case *metricspb.Metric_Sum:
		return r.processSumMetric(ctx, metricName, data.Sum, sessionID, workspace, provider, resourceAttrs)
	case *metricspb.Metric_Gauge:
		return r.processGaugeMetric(ctx, metricName, data.Gauge, sessionID, workspace, provider, resourceAttrs)
	default:
		// Skip histogram/summary for now
		return 0, nil
	}
}

// processSumMetric processes Sum (counter) metrics.
func (r *OTLPReceiver) processSumMetric(ctx context.Context, name string, sum *metricspb.Sum, sessionID, workspace, provider string, resourceAttrs map[string]any) (int, error) {
	var storedCount int
	for _, dp := range sum.DataPoints {
		metric := r.buildMetricFromDataPoint(name, "counter", dp, sessionID, workspace, provider, resourceAttrs)
		if err := r.backend.CreateMetric(ctx, metric); err != nil {
			return storedCount, fmt.Errorf("store metric: %w", err)
		}
		storedCount++
	}
	return storedCount, nil
}

// processGaugeMetric processes Gauge metrics.
func (r *OTLPReceiver) processGaugeMetric(ctx context.Context, name string, gauge *metricspb.Gauge, sessionID, workspace, provider string, resourceAttrs map[string]any) (int, error) {
	var storedCount int
	for _, dp := range gauge.DataPoints {
		metric := r.buildMetricFromDataPoint(name, "gauge", dp, sessionID, workspace, provider, resourceAttrs)
		if err := r.backend.CreateMetric(ctx, metric); err != nil {
			return storedCount, fmt.Errorf("store metric: %w", err)
		}
		storedCount++
	}
	return storedCount, nil
}

// buildMetricFromDataPoint creates a Metric from an OTLP NumberDataPoint.
func (r *OTLPReceiver) buildMetricFromDataPoint(name, metricType string, dp *metricspb.NumberDataPoint, sessionID, workspace, provider string, resourceAttrs map[string]any) *Metric {
	metric := &Metric{
		Name:               name,
		Type:               metricType,
		SessionID:          sessionID,
		Workspace:          workspace,
		Provider:           provider,
		Timestamp:          time.Unix(0, int64(dp.TimeUnixNano)),
		ResourceAttributes: resourceAttrs,
		Labels:             make(map[string]any),
	}

	// Extract labels from data point attributes
	for _, kv := range dp.Attributes {
		key := kv.Key
		val := anyValueToGo(kv.Value)
		metric.Labels[key] = val

		// Denormalize common labels for efficient queries
		if s, ok := val.(string); ok {
			switch key {
			case "type":
				metric.LabelType = s
			case "tool":
				metric.LabelTool = s
			case "decision":
				metric.LabelDecision = s
			case "language":
				metric.LabelLanguage = s
			case "model":
				metric.LabelModel = s
			}
		}
	}

	// Extract value
	switch v := dp.Value.(type) {
	case *metricspb.NumberDataPoint_AsInt:
		metric.ValueInt = v.AsInt
	case *metricspb.NumberDataPoint_AsDouble:
		metric.ValueFloat = v.AsDouble
	}

	return metric
}

// handleTraces handles the OTLP trace export endpoint.
func (r *OTLPReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var exportReq coltracepb.ExportTraceServiceRequest

	// Parse based on content type
	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "application/x-protobuf":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	default:
		// Try protobuf first, fall back to JSON
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := protojson.Unmarshal(body, &exportReq); jsonErr != nil {
				http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
				return
			}
		}
	}

	// Convert and store spans
	ctx := req.Context()
	fmt.Printf("observatory: received traces request with %d resource spans\n", len(exportReq.ResourceSpans))
	for i, resourceSpans := range exportReq.ResourceSpans {
		scopeCount := 0
		spanCount := 0
		if resourceSpans.ScopeSpans != nil {
			scopeCount = len(resourceSpans.ScopeSpans)
			for _, scope := range resourceSpans.ScopeSpans {
				if scope.Spans != nil {
					spanCount += len(scope.Spans)
				}
			}
		}
		fmt.Printf("observatory: resource[%d] has %d scope spans, %d total spans\n", i, scopeCount, spanCount)
		if err := r.processResourceSpans(ctx, resourceSpans); err != nil {
			// Log but continue processing other spans
			fmt.Printf("observatory: failed to process resource spans: %v\n", err)
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"partialSuccess": map[string]any{},
	})
}

// handleLogs handles the OTLP logs export endpoint.
// This receives Claude Code events (api_request, tool_result, etc.) and converts them to spans.
func (r *OTLPReceiver) handleLogs(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	fmt.Printf("observatory: received logs request, body length: %d, content-type: %s\n", len(body), req.Header.Get("Content-Type"))

	var exportReq collogspb.ExportLogsServiceRequest

	// Parse based on content type
	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "application/x-protobuf":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			fmt.Printf("observatory: failed to parse protobuf logs: %v\n", err)
			http.Error(w, fmt.Sprintf("failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			fmt.Printf("observatory: failed to parse JSON logs: %v\n", err)
			http.Error(w, fmt.Sprintf("failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	default:
		// Try protobuf first, fall back to JSON
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := protojson.Unmarshal(body, &exportReq); jsonErr != nil {
				fmt.Printf("observatory: failed to parse logs (tried both): proto=%v, json=%v\n", err, jsonErr)
				http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
				return
			}
		}
	}

	fmt.Printf("observatory: parsed %d resource logs\n", len(exportReq.ResourceLogs))

	// Convert and store logs as spans
	ctx := req.Context()
	for _, resourceLogs := range exportReq.ResourceLogs {
		if err := r.processResourceLogs(ctx, resourceLogs); err != nil {
			fmt.Printf("observatory: failed to process resource logs: %v\n", err)
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"partialSuccess": map[string]any{},
	})
}

// processResourceLogs converts Claude Code events to spans.
func (r *OTLPReceiver) processResourceLogs(ctx context.Context, rl *logspb.ResourceLogs) error {
	// Extract resource attributes
	resourceAttrs := make(map[string]any)
	if rl.Resource != nil {
		for _, kv := range rl.Resource.Attributes {
			resourceAttrs[kv.Key] = anyValueToGo(kv.Value)
		}
	}

	fmt.Printf("observatory: processing resource logs with %d scope logs, resource attrs: %v\n", len(rl.ScopeLogs), resourceAttrs)

	for _, scopeLogs := range rl.ScopeLogs {
		fmt.Printf("observatory: scope has %d log records\n", len(scopeLogs.LogRecords))
		for _, logRecord := range scopeLogs.LogRecords {
			span := r.convertLogToSpan(logRecord, resourceAttrs)
			if span != nil {
				// Validate task hierarchy references (M-TASK-HIERARCHY)
				r.validateTaskHierarchy(ctx, span)

				fmt.Printf("observatory: converted log to span: name=%s, tokens=%d/%d\n", span.Name, span.TokensIn, span.TokensOut)
				if err := r.backend.CreateSpan(ctx, span); err != nil {
					return fmt.Errorf("store span from log: %w", err)
				}
			} else {
				fmt.Printf("observatory: log record skipped (no event.name)\n")
			}
		}
	}
	return nil
}

// convertLogToSpan converts a Claude Code event (OTEL log record) to a span.
// Claude Code events: claude_code.api_request, claude_code.tool_result, etc.
func (r *OTLPReceiver) convertLogToSpan(log *logspb.LogRecord, resourceAttrs map[string]any) *Span {
	// Extract log attributes
	attrs := make(map[string]any)
	for _, kv := range log.Attributes {
		attrs[kv.Key] = anyValueToGo(kv.Value)
	}

	// Get event name
	eventName := extractString(attrs, "event.name")
	if eventName == "" {
		// Not a Claude Code event, skip
		return nil
	}

	// Generate IDs
	spanID := generateSpanID()
	traceID := extractString(resourceAttrs, "ailang.trace_id")
	if traceID == "" {
		traceID = generateTraceID()
	}

	// Extract timing
	timestamp := time.Unix(0, int64(log.TimeUnixNano))
	durationMs := int64(extractInt(attrs, "duration_ms"))
	var endTime *time.Time
	if durationMs > 0 {
		t := timestamp.Add(time.Duration(durationMs) * time.Millisecond)
		endTime = &t
	} else {
		endTime = &timestamp
	}

	// Extract Claude Code specific metrics
	tokensIn := int64(extractInt(attrs, "input_tokens"))
	tokensOut := int64(extractInt(attrs, "output_tokens"))
	cacheReadTokens := int64(extractInt(attrs, "cache_read_tokens"))
	cacheCreationTokens := int64(extractInt(attrs, "cache_creation_tokens"))
	costUSD := extractFloat(attrs, "cost_usd")
	model := extractString(attrs, "model")

	// Extract session ID - check both resource and span attributes
	// Claude Code sends session.id in span attributes, not resource attributes
	sessionID := extractString(resourceAttrs, "session.id")
	if sessionID == "" {
		sessionID = extractString(attrs, "session.id")
	}

	// Workspace enrichment (M-SESSION-WORKSPACE-HOOKS)
	// If session.id is present but process.cwd is missing, look up workspace from sessions table
	if sessionID != "" && extractString(resourceAttrs, "process.cwd") == "" {
		if workspace, err := r.backend.GetSessionWorkspace(sessionID); err == nil && workspace != "" {
			resourceAttrs["process.cwd"] = workspace
		}
	}

	// Calculate cost from tokens if not provided (M-TASK-HIERARCHY-FOLLOWUPS M6)
	// Use cache-aware pricing when cache tokens are present
	if costUSD == 0 && (tokensIn > 0 || tokensOut > 0 || cacheReadTokens > 0 || cacheCreationTokens > 0) && model != "" {
		costUSD = CalculateCostFromTokensWithCache(model, tokensIn, tokensOut, cacheReadTokens, cacheCreationTokens)
	}

	// Determine status
	status := SpanStatusOK
	statusMsg := ""
	if eventName == "claude_code.api_error" {
		status = SpanStatusError
		statusMsg = extractString(attrs, "error")
	} else if success := extractString(attrs, "success"); success == "false" {
		status = SpanStatusError
		statusMsg = extractString(attrs, "error")
	}

	// Build span name
	spanName := eventName
	if toolName := extractString(attrs, "tool_name"); toolName != "" {
		spanName = fmt.Sprintf("claude_code.tool.%s", toolName)
	}

	// Add session ID to attributes
	if sessionID != "" {
		attrs["claude.session_id"] = sessionID
	}

	// Extract task hierarchy context (M-TASK-HIERARCHY)
	// Priority order:
	// 1. ailang.task_id from resource attributes (OTEL_RESOURCE_ATTRIBUTES)
	// 2. task.id span attribute (coordinator sets this on task execution spans)
	// 3. exec.parent_task_id span attribute (ailang exec sets this for hierarchy)
	// 4. task.workspace span attribute (executor sets this for coordinator tasks)
	// 5. process.cwd worktree path fallback (Claude Code subprocesses)
	// 6. Session-based correlation: look up parent claude.execute span by session.id
	taskID := extractString(resourceAttrs, "ailang.task_id")
	if taskID == "" {
		// Check task.id span attribute (coordinator sets this)
		taskID = extractString(attrs, "task.id")
	}
	if taskID == "" {
		// Check exec.parent_task_id span attribute (ailang exec sets this for hierarchy)
		// This links child ailang.exec spans to their parent coordinator task
		taskID = extractString(attrs, "exec.parent_task_id")
	}
	if taskID == "" {
		// Check task.workspace span attribute (executor sets this)
		if workspace := extractString(attrs, "task.workspace"); workspace != "" {
			taskID = extractTaskIDFromPath(workspace)
		}
	}
	if taskID == "" && !backgroundOperationSpans[spanName] {
		taskID = extractTaskIDFromCwd(resourceAttrs)
	}
	assignmentID := extractString(resourceAttrs, "ailang.assignment_id")

	// Extract chain context (M-CHAINS-SIMPLIFY)
	// Chain IDs are passed via OTEL_RESOURCE_ATTRIBUTES from the executor
	chainID := extractString(resourceAttrs, "ailang.chain_id")
	if chainID == "" {
		chainID = extractString(attrs, "chain_id")
	}
	stageID := extractString(resourceAttrs, "ailang.stage_id")
	if stageID == "" {
		stageID = extractString(attrs, "stage_id")
	}

	// Session-based correlation (M-TASK-HIERARCHY-SESSION-LINKING)
	// Claude Code internal events have session.id but not task_id.
	// Look up the parent claude.execute span which has both session.id AND task_id.
	if taskID == "" && sessionID != "" {
		parentTaskID, parentAssignmentID, parentTraceID := r.backend.LookupTaskBySessionID(context.Background(), sessionID)
		if parentTaskID != "" {
			taskID = parentTaskID
			assignmentID = parentAssignmentID
			// Also link to parent trace for proper hierarchy
			if parentTraceID != "" {
				traceID = parentTraceID
			}
		} else {
			// No coordinator task found - use session_id as the task_id
			// This allows local Claude Code sessions to be queryable by standard hierarchy code
			taskID = sessionID
		}
	}

	return &Span{
		ID:                  spanID,
		TraceID:             traceID,
		TaskID:              taskID,
		AgentAssignmentID:   assignmentID,
		ChainID:             chainID,
		StageID:             stageID,
		Name:                spanName,
		Kind:                SpanKindInternal,
		Status:              status,
		StatusMessage:       statusMsg,
		StartTime:           timestamp,
		EndTime:             endTime,
		DurationMs:          durationMs,
		TokensIn:            tokensIn,
		TokensOut:           tokensOut,
		CacheReadTokens:     cacheReadTokens,
		CacheCreationTokens: cacheCreationTokens,
		CostUSD:             costUSD,
		Model:               model,
		Provider:            ProviderClaude,
		Attributes:          attrs,
		ResourceAttributes:  resourceAttrs,
	}
}

// generateSpanID generates a random 16-character hex span ID.
func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateTraceID generates a random 32-character hex trace ID.
func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// matchesPattern checks if a span name matches a single filter pattern,
// optionally scoped to a specific service.
func matchesPattern(name string, resourceAttrs map[string]any, p FilterPattern) bool {
	// Check service scope first
	if p.Service != "" {
		serviceName, _ := resourceAttrs["service.name"].(string)
		if serviceName != p.Service {
			return false
		}
	}

	switch p.Type {
	case "prefix":
		return strings.HasPrefix(name, p.Pattern)
	case "suffix":
		return strings.HasSuffix(name, p.Pattern)
	case "exact":
		return name == p.Pattern
	default:
		return name == p.Pattern
	}
}

// shouldFilterSpan returns true if the span should be filtered out (not stored).
// Uses the receiver's SpanFilterConfig: allow-list takes priority over deny-list.
func (r *OTLPReceiver) shouldFilterSpan(name string, resourceAttrs map[string]any) bool {
	if r.filterConfig.DisableAll {
		return false
	}

	// Allow-list takes priority: if any allow pattern matches, keep the span
	for _, p := range r.filterConfig.AllowPatterns {
		if matchesPattern(name, resourceAttrs, p) {
			return false
		}
	}

	// Deny-list: if any deny pattern matches, filter the span
	for _, p := range r.filterConfig.DenyPatterns {
		if matchesPattern(name, resourceAttrs, p) {
			return true
		}
	}

	// Default: keep the span
	return false
}

// processResourceSpans converts and stores spans from a resource.
func (r *OTLPReceiver) processResourceSpans(ctx context.Context, rs *tracepb.ResourceSpans) error {
	// Extract resource attributes
	resourceAttrs := make(map[string]any)
	if rs.Resource != nil {
		for _, kv := range rs.Resource.Attributes {
			resourceAttrs[kv.Key] = anyValueToGo(kv.Value)
		}
	}
	fmt.Printf("observatory: processing resource with attrs: %v\n", resourceAttrs)

	for _, scopeSpans := range rs.ScopeSpans {
		for _, span := range scopeSpans.Spans {
			// Check if span should be filtered out (pass resource attrs for service-based filtering)
			if r.shouldFilterSpan(span.Name, resourceAttrs) {
				fmt.Printf("observatory: filtered span name=%s (internal/noise)\n", span.Name)
				continue
			}

			normalized := r.convertSpan(span, resourceAttrs)

			// Validate task hierarchy references (M-TASK-HIERARCHY)
			r.validateTaskHierarchy(ctx, normalized)

			fmt.Printf("observatory: storing span name=%s, id=%s\n", normalized.Name, normalized.ID)
			if err := r.backend.CreateSpan(ctx, normalized); err != nil {
				return fmt.Errorf("store span %s: %w", span.SpanId, err)
			}
			fmt.Printf("observatory: stored span successfully\n")

			// Post-processing: Link orphaned Claude Code events to this span's task hierarchy
			// (M-TASK-HIERARCHY-SESSION-LINKING)
			// When claude.execute span arrives (delayed by OTEL batching), retroactively link
			// any Claude Code internal events that arrived earlier via OTLP logs.
			if normalized.Name == "claude.execute" && normalized.TaskID != "" {
				sessionID := extractString(normalized.Attributes, "session.id")
				if sessionID != "" {
					linked, err := r.backend.LinkOrphanedSpansBySession(ctx, sessionID, normalized.TaskID, normalized.AgentAssignmentID)
					if err != nil {
						fmt.Printf("observatory: warning: failed to link orphaned spans for session %s: %v\n", sessionID, err)
					} else if linked > 0 {
						fmt.Printf("observatory: linked %d orphaned Claude Code events to task %s (session %s)\n", linked, normalized.TaskID, sessionID)
						// Recalculate task aggregates to reflect newly linked spans
						if err := r.backend.RecalculateTaskAggregates(ctx, normalized.TaskID); err != nil {
							fmt.Printf("observatory: warning: failed to recalculate aggregates for task %s: %v\n", normalized.TaskID, err)
						}
					}
				}
			}
		}
	}
	return nil
}

// convertSpan converts an OTLP span to the observatory model.
func (r *OTLPReceiver) convertSpan(span *tracepb.Span, resourceAttrs map[string]any) *Span {
	// Extract span attributes
	attrs := make(map[string]any)
	for _, kv := range span.Attributes {
		attrs[kv.Key] = anyValueToGo(kv.Value)
	}

	// Convert times
	startTime := time.Unix(0, int64(span.StartTimeUnixNano))
	endTime := time.Unix(0, int64(span.EndTimeUnixNano))
	durationMs := endTime.Sub(startTime).Milliseconds()

	// Convert status
	status := SpanStatusUnset
	statusMsg := ""
	if span.Status != nil {
		switch span.Status.Code {
		case tracepb.Status_STATUS_CODE_OK:
			status = SpanStatusOK
		case tracepb.Status_STATUS_CODE_ERROR:
			status = SpanStatusError
		}
		statusMsg = span.Status.Message
	}

	// Convert kind
	kind := SpanKindInternal
	switch span.Kind {
	case tracepb.Span_SPAN_KIND_CLIENT:
		kind = SpanKindClient
	case tracepb.Span_SPAN_KIND_SERVER:
		kind = SpanKindServer
	case tracepb.Span_SPAN_KIND_PRODUCER:
		kind = SpanKindProducer
	case tracepb.Span_SPAN_KIND_CONSUMER:
		kind = SpanKindConsumer
	}

	// Extract parent span ID
	parentSpanID := ""
	if len(span.ParentSpanId) > 0 {
		parentSpanID = fmt.Sprintf("%x", span.ParentSpanId)
	}

	// Extract normalized metrics from attributes
	// Support multiple naming conventions:
	// - gen_ai.* (OpenTelemetry semantic conventions)
	// - ailang.* (AILANG custom)
	// - ai.* (used by internal/ai/ providers)
	// - task.* (used by coordinator executor spans)
	tokensIn := int64(extractInt(attrs, "gen_ai.usage.input_tokens", "ailang.tokens.input", "ai.tokens_in", "task.tokens_in"))
	tokensOut := int64(extractInt(attrs, "gen_ai.usage.output_tokens", "ailang.tokens.output", "ai.tokens_out", "task.tokens_out"))
	costUSD := extractFloat(attrs, "gen_ai.usage.cost", "ailang.cost.usd", "ai.cost_usd", "task.cost_usd")
	model := extractString(attrs, "gen_ai.request.model", "ailang.model", "ai.model")

	// Calculate cost from tokens if not provided (M-TASK-HIERARCHY-FOLLOWUPS M6)
	// AI providers emit tokens but not cost, so we calculate from models.yml pricing
	if costUSD == 0 && (tokensIn > 0 || tokensOut > 0) && model != "" {
		costUSD = CalculateCostFromTokens(model, tokensIn, tokensOut)
	}
	providerStr := extractString(attrs, "ailang.provider", "gen_ai.system")

	// Map provider string to Provider type
	var provider Provider
	switch providerStr {
	case "claude", "anthropic":
		provider = ProviderClaude
	case "gemini", "google":
		provider = ProviderGemini
	case "ollama":
		provider = ProviderOllama
	default:
		provider = Provider(providerStr)
	}

	// Workspace enrichment (M-SESSION-WORKSPACE-HOOKS)
	// If session.id is present but process.cwd is missing, look up workspace from sessions table
	sessionID := extractString(resourceAttrs, "session.id")
	if sessionID == "" {
		sessionID = extractString(attrs, "session.id")
	}
	if sessionID != "" && extractString(resourceAttrs, "process.cwd") == "" {
		if workspace, err := r.backend.GetSessionWorkspace(sessionID); err == nil && workspace != "" {
			resourceAttrs["process.cwd"] = workspace
		}
	}

	// Extract task hierarchy context (M-TASK-HIERARCHY)
	// Priority order:
	// 1. ailang.task_id from resource attributes (OTEL_RESOURCE_ATTRIBUTES)
	// 2. task.id span attribute (coordinator sets this on task execution spans)
	// 3. exec.parent_task_id span attribute (ailang exec sets this for hierarchy)
	// 4. task.workspace span attribute (executor sets this for coordinator tasks)
	// 5. process.cwd worktree path fallback (Claude Code subprocesses)
	taskID := extractString(resourceAttrs, "ailang.task_id")
	if taskID == "" {
		// Check task.id span attribute (coordinator sets this)
		taskID = extractString(attrs, "task.id")
	}
	if taskID == "" {
		// Check exec.parent_task_id span attribute (ailang exec sets this for hierarchy)
		// This links child ailang.exec spans to their parent coordinator task
		taskID = extractString(attrs, "exec.parent_task_id")
	}
	if taskID == "" {
		// Check task.workspace span attribute (executor sets this)
		if workspace := extractString(attrs, "task.workspace"); workspace != "" {
			taskID = extractTaskIDFromPath(workspace)
		}
	}
	if taskID == "" && !backgroundOperationSpans[span.Name] {
		taskID = extractTaskIDFromCwd(resourceAttrs)
	}
	assignmentID := extractString(resourceAttrs, "ailang.assignment_id")

	// Extract chain context (M-CHAINS-SIMPLIFY)
	// Chain IDs are passed via OTEL_RESOURCE_ATTRIBUTES from the executor
	chainID := extractString(resourceAttrs, "ailang.chain_id")
	if chainID == "" {
		chainID = extractString(attrs, "chain_id")
	}
	stageID := extractString(resourceAttrs, "ailang.stage_id")
	if stageID == "" {
		stageID = extractString(attrs, "stage_id")
	}

	return &Span{
		ID:                 fmt.Sprintf("%x", span.SpanId),
		TraceID:            fmt.Sprintf("%x", span.TraceId),
		ParentSpanID:       parentSpanID,
		TaskID:             taskID,
		AgentAssignmentID:  assignmentID,
		ChainID:            chainID,
		StageID:            stageID,
		Name:               span.Name,
		Kind:               kind,
		Status:             status,
		StatusMessage:      statusMsg,
		StartTime:          startTime,
		EndTime:            &endTime,
		DurationMs:         durationMs,
		TokensIn:           tokensIn,
		TokensOut:          tokensOut,
		CostUSD:            costUSD,
		Model:              model,
		Provider:           provider,
		Attributes:         attrs,
		ResourceAttributes: resourceAttrs,
	}
}

// validateTaskHierarchy validates that task_id and assignment_id references exist.
// Logs warnings if they don't - spans are still stored but with a warning.
func (r *OTLPReceiver) validateTaskHierarchy(ctx context.Context, span *Span) {
	// Validate task_id if present
	if span.TaskID != "" {
		task, err := r.backend.GetTask(ctx, span.TaskID)
		if err != nil || task == nil {
			fmt.Printf("observatory: WARNING: span %s has task_id=%s but task not found\n", span.ID, span.TaskID)
		}
	}

	// Validate assignment_id if present
	if span.AgentAssignmentID != "" {
		assignment, err := r.backend.GetAgentAssignment(ctx, span.AgentAssignmentID)
		if err != nil || assignment == nil {
			fmt.Printf("observatory: WARNING: span %s has assignment_id=%s but assignment not found\n", span.ID, span.AgentAssignmentID)
		}
	}
}

// anyValueToGo converts an OTLP AnyValue to a Go value.
func anyValueToGo(v *commonpb.AnyValue) any {
	if v == nil {
		return nil
	}
	switch val := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return val.StringValue
	case *commonpb.AnyValue_IntValue:
		return val.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return val.DoubleValue
	case *commonpb.AnyValue_BoolValue:
		return val.BoolValue
	case *commonpb.AnyValue_ArrayValue:
		arr := make([]any, len(val.ArrayValue.Values))
		for i, elem := range val.ArrayValue.Values {
			arr[i] = anyValueToGo(elem)
		}
		return arr
	case *commonpb.AnyValue_KvlistValue:
		m := make(map[string]any)
		for _, kv := range val.KvlistValue.Values {
			m[kv.Key] = anyValueToGo(kv.Value)
		}
		return m
	default:
		return nil
	}
}

// Helper functions
func extractInt(attrs map[string]any, keys ...string) int {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case string:
				// Claude Code sends numbers as strings
				if i, err := strconv.Atoi(val); err == nil {
					return i
				}
				// Try parsing as float then convert
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return int(f)
				}
			}
		}
	}
	return 0
}

func extractFloat(attrs map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case string:
				// Claude Code sends numbers as strings
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func extractString(attrs map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// extractTaskIDFromCwd extracts task ID from worktree path in process.cwd attribute.
// Claude Code CLI doesn't pass OTEL_RESOURCE_ATTRIBUTES to subprocesses,
// but the worktree path contains the task ID.
func extractTaskIDFromCwd(attrs map[string]any) string {
	cwd := extractString(attrs, "process.cwd")
	if cwd == "" {
		return ""
	}
	return extractTaskIDFromPath(cwd)
}

// extractTaskIDFromPath extracts task ID from a file path.
// Path format: .../worktrees/.../task-XXXXXXXX/...
// Returns empty string if no task ID found.
func extractTaskIDFromPath(path string) string {
	if path == "" {
		return ""
	}

	// Look for task ID pattern in the path
	const taskPrefix = "task-"
	idx := strings.Index(path, "/worktrees/")
	if idx == -1 {
		return ""
	}

	// Find task-XXXXXXXX in the path after /worktrees/
	remainder := path[idx:]
	taskIdx := strings.Index(remainder, taskPrefix)
	if taskIdx == -1 {
		return ""
	}

	// Extract task ID (task-XXXXXXXX format, 8 hex chars after prefix)
	start := taskIdx
	end := start + len(taskPrefix) + 8 // task- + 8 hex chars
	if end > len(remainder) {
		// Try to find next path separator
		nextSlash := strings.Index(remainder[start:], "/")
		if nextSlash > 0 {
			end = start + nextSlash
		} else {
			end = len(remainder)
		}
	}

	taskID := remainder[start:end]
	if strings.HasPrefix(taskID, taskPrefix) {
		return taskID
	}
	return ""
}
