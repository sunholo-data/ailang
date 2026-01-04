// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ProviderNormalizer normalizes telemetry from different providers into Spans.
type ProviderNormalizer struct{}

// NewProviderNormalizer creates a new normalizer.
func NewProviderNormalizer() *ProviderNormalizer {
	return &ProviderNormalizer{}
}

// ClaudeMetrics represents metrics exported by Claude Code CLI.
// Claude exports metrics/events only, not full OTEL traces.
type ClaudeMetrics struct {
	SessionID     string    `json:"session_id"`
	TaskID        string    `json:"task_id,omitempty"`
	Model         string    `json:"model"`
	TotalTokensIn int64     `json:"total_input_tokens"`
	TotalTokenOut int64     `json:"total_output_tokens"`
	TotalCostUSD  float64   `json:"total_cost_usd"`
	DurationMs    int64     `json:"duration_ms"`
	TurnCount     int       `json:"turn_count"`
	ToolCalls     int       `json:"tool_calls"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status"` // "completed", "failed", "cancelled"
	ErrorMessage  string    `json:"error_message,omitempty"`

	// Events captured during session
	Events []ClaudeEvent `json:"events,omitempty"`
}

// ClaudeEvent represents an event from Claude Code CLI.
type ClaudeEvent struct {
	Type         string         `json:"type"` // "tool_call", "approval", "error"
	Timestamp    time.Time      `json:"timestamp"`
	ToolName     string         `json:"tool_name,omitempty"`
	ToolInput    map[string]any `json:"tool_input,omitempty"`
	ToolResult   string         `json:"tool_result,omitempty"`
	Approved     *bool          `json:"approved,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

// NormalizeClaudeMetrics converts Claude metrics into a Span with events.
func (n *ProviderNormalizer) NormalizeClaudeMetrics(metrics *ClaudeMetrics) (*Span, error) {
	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	// Generate span ID from session ID
	spanID := fmt.Sprintf("claude-%s", metrics.SessionID)

	// Determine status
	var status SpanStatus
	switch metrics.Status {
	case "completed":
		status = SpanStatusOK
	case "failed", "cancelled":
		status = SpanStatusError
	default:
		status = SpanStatusUnset
	}

	span := &Span{
		ID:         spanID,
		TraceID:    spanID, // Claude doesn't have traces, use span ID as trace ID
		TaskID:     metrics.TaskID,
		Name:       "claude.session",
		Kind:       SpanKindClient,
		Status:     status,
		StartTime:  metrics.StartTime,
		EndTime:    &metrics.EndTime,
		DurationMs: metrics.DurationMs,
		TokensIn:   metrics.TotalTokensIn,
		TokensOut:  metrics.TotalTokenOut,
		CostUSD:    metrics.TotalCostUSD,
		Model:      metrics.Model,
		Provider:   ProviderClaude,
		Attributes: map[string]any{
			"claude.session_id": metrics.SessionID,
			"claude.turn_count": metrics.TurnCount,
			"claude.tool_calls": metrics.ToolCalls,
		},
		CreatedAt: time.Now(),
	}

	if metrics.ErrorMessage != "" {
		span.StatusMessage = metrics.ErrorMessage
		span.Attributes["error.message"] = metrics.ErrorMessage
	}

	// Convert events to SpanEvents
	for _, evt := range metrics.Events {
		spanEvent := n.normalizeClaudeEvent(&evt, spanID)
		span.Events = append(span.Events, spanEvent)
	}

	return span, nil
}

func (n *ProviderNormalizer) normalizeClaudeEvent(evt *ClaudeEvent, spanID string) SpanEvent {
	se := SpanEvent{
		SpanID:     spanID,
		Timestamp:  evt.Timestamp,
		Attributes: make(map[string]any),
	}

	switch evt.Type {
	case "tool_call":
		se.Name = "tool.call"
		se.EventType = EventTypeTool
		se.ToolName = evt.ToolName
		if evt.ToolInput != nil {
			se.Attributes["tool.input"] = evt.ToolInput
		}
		if evt.ToolResult != "" {
			se.Attributes["tool.result"] = evt.ToolResult
		}

	case "approval":
		se.Name = "approval.request"
		se.EventType = EventTypeApproval
		if evt.Approved != nil {
			if *evt.Approved {
				se.ApprovalStatus = ApprovalStatusApproved
			} else {
				se.ApprovalStatus = ApprovalStatusRejected
			}
		} else {
			se.ApprovalStatus = ApprovalStatusPending
		}

	case "error":
		se.Name = "error"
		se.EventType = EventTypeError
		se.ErrorMessage = evt.ErrorMessage

	default:
		se.Name = evt.Type
		se.EventType = EventTypeCustom
	}

	return se
}

// GeminiOTELSpan represents an OTEL span from Gemini CLI.
// Gemini exports full OTEL traces with parent-child relationships.
type GeminiOTELSpan struct {
	TraceID           string        `json:"traceId"`
	SpanID            string        `json:"spanId"`
	ParentSpanID      string        `json:"parentSpanId,omitempty"`
	Name              string        `json:"name"`
	Kind              int           `json:"kind"` // OTEL span kind enum
	StartTimeUnixNano int64         `json:"startTimeUnixNano"`
	EndTimeUnixNano   int64         `json:"endTimeUnixNano"`
	Attributes        []KeyValue    `json:"attributes,omitempty"`
	Status            *OTELStatus   `json:"status,omitempty"`
	Events            []OTELEvent   `json:"events,omitempty"`
	Resource          *OTELResource `json:"resource,omitempty"`
}

// KeyValue represents an OTEL attribute.
type KeyValue struct {
	Key   string     `json:"key"`
	Value ValueUnion `json:"value"`
}

// ValueUnion represents an OTEL attribute value.
type ValueUnion struct {
	StringValue *string  `json:"stringValue,omitempty"`
	IntValue    *int64   `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

// OTELStatus represents OTEL span status.
type OTELStatus struct {
	Code    int    `json:"code"` // 0=Unset, 1=OK, 2=Error
	Message string `json:"message,omitempty"`
}

// OTELEvent represents an OTEL span event.
type OTELEvent struct {
	Name         string     `json:"name"`
	TimeUnixNano int64      `json:"timeUnixNano"`
	Attributes   []KeyValue `json:"attributes,omitempty"`
}

// OTELResource represents OTEL resource attributes.
type OTELResource struct {
	Attributes []KeyValue `json:"attributes,omitempty"`
}

// NormalizeGeminiSpan converts a Gemini OTEL span into our normalized Span.
func (n *ProviderNormalizer) NormalizeGeminiSpan(otelSpan *GeminiOTELSpan, taskID string) (*Span, error) {
	if otelSpan == nil {
		return nil, fmt.Errorf("otelSpan cannot be nil")
	}

	startTime := time.Unix(0, otelSpan.StartTimeUnixNano)
	endTime := time.Unix(0, otelSpan.EndTimeUnixNano)
	durationMs := (otelSpan.EndTimeUnixNano - otelSpan.StartTimeUnixNano) / 1_000_000

	span := &Span{
		ID:           fmt.Sprintf("gemini-%s", otelSpan.SpanID),
		TraceID:      otelSpan.TraceID,
		ParentSpanID: otelSpan.ParentSpanID,
		TaskID:       taskID,
		Name:         otelSpan.Name,
		Kind:         normalizeSpanKind(otelSpan.Kind),
		Status:       normalizeOTELStatus(otelSpan.Status),
		StartTime:    startTime,
		EndTime:      &endTime,
		DurationMs:   durationMs,
		Provider:     ProviderGemini,
		Attributes:   make(map[string]any),
		CreatedAt:    time.Now(),
	}

	if otelSpan.Status != nil {
		span.StatusMessage = otelSpan.Status.Message
	}

	// Extract normalized attributes and resource attributes
	for _, kv := range otelSpan.Attributes {
		val := extractValue(kv.Value)
		span.Attributes[kv.Key] = val

		// Extract well-known attributes
		switch kv.Key {
		case "gen_ai.usage.input_tokens", "llm.token_count.input":
			if v, ok := val.(int64); ok {
				span.TokensIn = v
			}
		case "gen_ai.usage.output_tokens", "llm.token_count.output":
			if v, ok := val.(int64); ok {
				span.TokensOut = v
			}
		case "gen_ai.response.model", "llm.model":
			if v, ok := val.(string); ok {
				span.Model = v
			}
		case "llm.cost":
			if v, ok := val.(float64); ok {
				span.CostUSD = v
			}
		}
	}

	// Extract resource attributes
	if otelSpan.Resource != nil {
		span.ResourceAttributes = make(map[string]any)
		for _, kv := range otelSpan.Resource.Attributes {
			span.ResourceAttributes[kv.Key] = extractValue(kv.Value)
		}
	}

	// Convert OTEL events
	for _, otelEvt := range otelSpan.Events {
		spanEvent := n.normalizeGeminiEvent(&otelEvt, span.ID)
		span.Events = append(span.Events, spanEvent)
	}

	return span, nil
}

func (n *ProviderNormalizer) normalizeGeminiEvent(evt *OTELEvent, spanID string) SpanEvent {
	se := SpanEvent{
		SpanID:     spanID,
		Name:       evt.Name,
		Timestamp:  time.Unix(0, evt.TimeUnixNano),
		Attributes: make(map[string]any),
	}

	// Extract attributes
	for _, kv := range evt.Attributes {
		se.Attributes[kv.Key] = extractValue(kv.Value)
	}

	// Detect event type from name
	nameLower := strings.ToLower(evt.Name)
	switch {
	case strings.Contains(nameLower, "tool"):
		se.EventType = EventTypeTool
		if tn, ok := se.Attributes["tool.name"].(string); ok {
			se.ToolName = tn
		}
	case strings.Contains(nameLower, "error") || strings.Contains(nameLower, "exception"):
		se.EventType = EventTypeError
		if msg, ok := se.Attributes["error.message"].(string); ok {
			se.ErrorMessage = msg
		} else if msg, ok := se.Attributes["exception.message"].(string); ok {
			se.ErrorMessage = msg
		}
	case strings.Contains(nameLower, "approval"):
		se.EventType = EventTypeApproval
		if status, ok := se.Attributes["approval.status"].(string); ok {
			switch status {
			case "approved":
				se.ApprovalStatus = ApprovalStatusApproved
			case "rejected":
				se.ApprovalStatus = ApprovalStatusRejected
			default:
				se.ApprovalStatus = ApprovalStatusPending
			}
		}
	default:
		se.EventType = EventTypeCustom
	}

	return se
}

// NormalizeGeminiTrace converts a complete Gemini OTEL trace into normalized Spans.
func (n *ProviderNormalizer) NormalizeGeminiTrace(otelSpans []*GeminiOTELSpan, taskID string) ([]*Span, error) {
	var spans []*Span
	for _, otelSpan := range otelSpans {
		span, err := n.NormalizeGeminiSpan(otelSpan, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize span %s: %w", otelSpan.SpanID, err)
		}
		spans = append(spans, span)
	}
	return spans, nil
}

// normalizeSpanKind converts OTEL span kind to our SpanKind.
func normalizeSpanKind(kind int) SpanKind {
	switch kind {
	case 1:
		return SpanKindInternal
	case 2:
		return SpanKindServer
	case 3:
		return SpanKindClient
	case 4:
		return SpanKindProducer
	case 5:
		return SpanKindConsumer
	default:
		return SpanKindInternal
	}
}

// normalizeOTELStatus converts OTEL status to our SpanStatus.
func normalizeOTELStatus(status *OTELStatus) SpanStatus {
	if status == nil {
		return SpanStatusUnset
	}
	switch status.Code {
	case 1:
		return SpanStatusOK
	case 2:
		return SpanStatusError
	default:
		return SpanStatusUnset
	}
}

// extractValue extracts the actual value from an OTEL ValueUnion.
func extractValue(v ValueUnion) any {
	if v.StringValue != nil {
		return *v.StringValue
	}
	if v.IntValue != nil {
		return *v.IntValue
	}
	if v.DoubleValue != nil {
		return *v.DoubleValue
	}
	if v.BoolValue != nil {
		return *v.BoolValue
	}
	return nil
}

// ParseClaudeMetricsJSON parses Claude metrics from JSON.
func ParseClaudeMetricsJSON(data []byte) (*ClaudeMetrics, error) {
	var metrics ClaudeMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse Claude metrics: %w", err)
	}
	return &metrics, nil
}

// ParseGeminiTraceJSON parses Gemini OTEL trace from JSON.
func ParseGeminiTraceJSON(data []byte) ([]*GeminiOTELSpan, error) {
	var spans []*GeminiOTELSpan
	if err := json.Unmarshal(data, &spans); err != nil {
		// Try parsing as single span
		var span GeminiOTELSpan
		if err2 := json.Unmarshal(data, &span); err2 == nil {
			return []*GeminiOTELSpan{&span}, nil
		}
		return nil, fmt.Errorf("failed to parse Gemini trace: %w", err)
	}
	return spans, nil
}
