package observatory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeClaudeMetrics(t *testing.T) {
	normalizer := NewProviderNormalizer()

	now := time.Now().Truncate(time.Second)
	endTime := now.Add(30 * time.Second)

	metrics := &ClaudeMetrics{
		SessionID:     "session-123",
		TaskID:        "task-456",
		Model:         "claude-3-sonnet",
		TotalTokensIn: 1000,
		TotalTokenOut: 500,
		TotalCostUSD:  0.05,
		DurationMs:    30000,
		TurnCount:     5,
		ToolCalls:     10,
		StartTime:     now,
		EndTime:       endTime,
		Status:        "completed",
		Events: []ClaudeEvent{
			{
				Type:      "tool_call",
				Timestamp: now.Add(10 * time.Second),
				ToolName:  "Read",
				ToolInput: map[string]any{"path": "/foo.txt"},
			},
			{
				Type:      "approval",
				Timestamp: now.Add(20 * time.Second),
				Approved:  ptrBool(true),
			},
		},
	}

	span, err := normalizer.NormalizeClaudeMetrics(metrics)
	if err != nil {
		t.Fatalf("NormalizeClaudeMetrics failed: %v", err)
	}

	// Check span properties
	if span.ID != "claude-session-123" {
		t.Errorf("ID mismatch: got %s", span.ID)
	}
	if span.TaskID != "task-456" {
		t.Errorf("TaskID mismatch: got %s", span.TaskID)
	}
	if span.Model != "claude-3-sonnet" {
		t.Errorf("Model mismatch: got %s", span.Model)
	}
	if span.TokensIn != 1000 {
		t.Errorf("TokensIn mismatch: got %d", span.TokensIn)
	}
	if span.TokensOut != 500 {
		t.Errorf("TokensOut mismatch: got %d", span.TokensOut)
	}
	if span.CostUSD != 0.05 {
		t.Errorf("CostUSD mismatch: got %f", span.CostUSD)
	}
	if span.Status != SpanStatusOK {
		t.Errorf("Status mismatch: got %s", span.Status)
	}
	if span.Provider != ProviderClaude {
		t.Errorf("Provider mismatch: got %s", span.Provider)
	}

	// Check events
	if len(span.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(span.Events))
	}

	// Check tool call event
	if span.Events[0].EventType != EventTypeTool {
		t.Errorf("event 0 type mismatch: got %s", span.Events[0].EventType)
	}
	if span.Events[0].ToolName != "Read" {
		t.Errorf("event 0 tool name mismatch: got %s", span.Events[0].ToolName)
	}

	// Check approval event
	if span.Events[1].EventType != EventTypeApproval {
		t.Errorf("event 1 type mismatch: got %s", span.Events[1].EventType)
	}
	if span.Events[1].ApprovalStatus != ApprovalStatusApproved {
		t.Errorf("event 1 approval status mismatch: got %s", span.Events[1].ApprovalStatus)
	}
}

func TestNormalizeClaudeMetrics_Failed(t *testing.T) {
	normalizer := NewProviderNormalizer()

	now := time.Now()
	metrics := &ClaudeMetrics{
		SessionID:    "session-123",
		Status:       "failed",
		ErrorMessage: "Task cancelled by user",
		StartTime:    now,
		EndTime:      now.Add(time.Second),
	}

	span, err := normalizer.NormalizeClaudeMetrics(metrics)
	if err != nil {
		t.Fatalf("NormalizeClaudeMetrics failed: %v", err)
	}

	if span.Status != SpanStatusError {
		t.Errorf("Status should be error for failed task")
	}
	if span.StatusMessage != "Task cancelled by user" {
		t.Errorf("StatusMessage mismatch: got %s", span.StatusMessage)
	}
}

func TestNormalizeClaudeMetrics_Nil(t *testing.T) {
	normalizer := NewProviderNormalizer()
	_, err := normalizer.NormalizeClaudeMetrics(nil)
	if err == nil {
		t.Error("expected error for nil metrics")
	}
}

func TestNormalizeGeminiSpan(t *testing.T) {
	normalizer := NewProviderNormalizer()

	now := time.Now()
	startNano := now.UnixNano()
	endNano := now.Add(5 * time.Second).UnixNano()

	inputTokens := int64(500)
	outputTokens := int64(200)
	model := "gemini-2.0-pro"

	otelSpan := &GeminiOTELSpan{
		TraceID:           "trace-abc",
		SpanID:            "span-123",
		ParentSpanID:      "span-parent",
		Name:              "gemini.generate",
		Kind:              3, // Client
		StartTimeUnixNano: startNano,
		EndTimeUnixNano:   endNano,
		Attributes: []KeyValue{
			{Key: "gen_ai.usage.input_tokens", Value: ValueUnion{IntValue: &inputTokens}},
			{Key: "gen_ai.usage.output_tokens", Value: ValueUnion{IntValue: &outputTokens}},
			{Key: "gen_ai.response.model", Value: ValueUnion{StringValue: &model}},
		},
		Status: &OTELStatus{Code: 1, Message: ""},
		Events: []OTELEvent{
			{
				Name:         "tool.call",
				TimeUnixNano: startNano + 1_000_000_000,
				Attributes: []KeyValue{
					{Key: "tool.name", Value: ValueUnion{StringValue: ptrString("search")}},
				},
			},
		},
	}

	span, err := normalizer.NormalizeGeminiSpan(otelSpan, "task-789")
	if err != nil {
		t.Fatalf("NormalizeGeminiSpan failed: %v", err)
	}

	// Check span properties
	if span.ID != "gemini-span-123" {
		t.Errorf("ID mismatch: got %s", span.ID)
	}
	if span.TraceID != "trace-abc" {
		t.Errorf("TraceID mismatch: got %s", span.TraceID)
	}
	if span.ParentSpanID != "span-parent" {
		t.Errorf("ParentSpanID mismatch: got %s", span.ParentSpanID)
	}
	if span.TaskID != "task-789" {
		t.Errorf("TaskID mismatch: got %s", span.TaskID)
	}
	if span.Name != "gemini.generate" {
		t.Errorf("Name mismatch: got %s", span.Name)
	}
	if span.Kind != SpanKindClient {
		t.Errorf("Kind mismatch: got %s", span.Kind)
	}
	if span.Status != SpanStatusOK {
		t.Errorf("Status mismatch: got %s", span.Status)
	}
	if span.TokensIn != 500 {
		t.Errorf("TokensIn mismatch: got %d", span.TokensIn)
	}
	if span.TokensOut != 200 {
		t.Errorf("TokensOut mismatch: got %d", span.TokensOut)
	}
	if span.Model != "gemini-2.0-pro" {
		t.Errorf("Model mismatch: got %s", span.Model)
	}
	if span.Provider != ProviderGemini {
		t.Errorf("Provider mismatch: got %s", span.Provider)
	}

	// Check events
	if len(span.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(span.Events))
	}
	if span.Events[0].EventType != EventTypeTool {
		t.Errorf("event type mismatch: got %s", span.Events[0].EventType)
	}
	if span.Events[0].ToolName != "search" {
		t.Errorf("tool name mismatch: got %s", span.Events[0].ToolName)
	}
}

func TestNormalizeGeminiSpan_Error(t *testing.T) {
	normalizer := NewProviderNormalizer()

	now := time.Now()
	otelSpan := &GeminiOTELSpan{
		TraceID:           "trace-abc",
		SpanID:            "span-123",
		Name:              "gemini.generate",
		Kind:              3,
		StartTimeUnixNano: now.UnixNano(),
		EndTimeUnixNano:   now.Add(time.Second).UnixNano(),
		Status:            &OTELStatus{Code: 2, Message: "rate limit exceeded"},
	}

	span, err := normalizer.NormalizeGeminiSpan(otelSpan, "")
	if err != nil {
		t.Fatalf("NormalizeGeminiSpan failed: %v", err)
	}

	if span.Status != SpanStatusError {
		t.Errorf("Status should be error")
	}
	if span.StatusMessage != "rate limit exceeded" {
		t.Errorf("StatusMessage mismatch: got %s", span.StatusMessage)
	}
}

func TestNormalizeGeminiTrace(t *testing.T) {
	normalizer := NewProviderNormalizer()

	now := time.Now()
	otelSpans := []*GeminiOTELSpan{
		{
			TraceID:           "trace-abc",
			SpanID:            "span-1",
			Name:              "root",
			Kind:              2,
			StartTimeUnixNano: now.UnixNano(),
			EndTimeUnixNano:   now.Add(10 * time.Second).UnixNano(),
			Status:            &OTELStatus{Code: 1},
		},
		{
			TraceID:           "trace-abc",
			SpanID:            "span-2",
			ParentSpanID:      "span-1",
			Name:              "child",
			Kind:              1,
			StartTimeUnixNano: now.Add(time.Second).UnixNano(),
			EndTimeUnixNano:   now.Add(5 * time.Second).UnixNano(),
			Status:            &OTELStatus{Code: 1},
		},
	}

	spans, err := normalizer.NormalizeGeminiTrace(otelSpans, "task-123")
	if err != nil {
		t.Fatalf("NormalizeGeminiTrace failed: %v", err)
	}

	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	if spans[0].TraceID != spans[1].TraceID {
		t.Error("spans should share same trace ID")
	}
	if spans[1].ParentSpanID != "span-1" {
		t.Errorf("child should reference parent, got %s", spans[1].ParentSpanID)
	}
}

func TestParseClaudeMetricsJSON(t *testing.T) {
	jsonData := `{
		"session_id": "sess-abc",
		"model": "claude-3-haiku",
		"total_input_tokens": 100,
		"total_output_tokens": 50,
		"total_cost_usd": 0.001,
		"duration_ms": 5000,
		"turn_count": 2,
		"tool_calls": 3,
		"status": "completed"
	}`

	metrics, err := ParseClaudeMetricsJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseClaudeMetricsJSON failed: %v", err)
	}

	if metrics.SessionID != "sess-abc" {
		t.Errorf("SessionID mismatch: got %s", metrics.SessionID)
	}
	if metrics.TotalTokensIn != 100 {
		t.Errorf("TotalTokensIn mismatch: got %d", metrics.TotalTokensIn)
	}
	if metrics.TotalCostUSD != 0.001 {
		t.Errorf("TotalCostUSD mismatch: got %f", metrics.TotalCostUSD)
	}
}

func TestParseGeminiTraceJSON(t *testing.T) {
	jsonData := `[{
		"traceId": "trace-123",
		"spanId": "span-456",
		"name": "test.span",
		"kind": 1,
		"startTimeUnixNano": 1000000000,
		"endTimeUnixNano": 2000000000
	}]`

	spans, err := ParseGeminiTraceJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseGeminiTraceJSON failed: %v", err)
	}

	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].TraceID != "trace-123" {
		t.Errorf("TraceID mismatch: got %s", spans[0].TraceID)
	}
}

func TestParseGeminiTraceJSON_SingleSpan(t *testing.T) {
	jsonData := `{
		"traceId": "trace-123",
		"spanId": "span-456",
		"name": "single.span",
		"kind": 1,
		"startTimeUnixNano": 1000000000,
		"endTimeUnixNano": 2000000000
	}`

	spans, err := ParseGeminiTraceJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseGeminiTraceJSON failed: %v", err)
	}

	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
}

func TestNormalizeSpanKind(t *testing.T) {
	tests := []struct {
		input    int
		expected SpanKind
	}{
		{0, SpanKindInternal},
		{1, SpanKindInternal},
		{2, SpanKindServer},
		{3, SpanKindClient},
		{4, SpanKindProducer},
		{5, SpanKindConsumer},
		{99, SpanKindInternal},
	}

	for _, tc := range tests {
		result := normalizeSpanKind(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeSpanKind(%d): got %s, want %s", tc.input, result, tc.expected)
		}
	}
}

func TestNormalizeOTELStatus(t *testing.T) {
	tests := []struct {
		input    *OTELStatus
		expected SpanStatus
	}{
		{nil, SpanStatusUnset},
		{&OTELStatus{Code: 0}, SpanStatusUnset},
		{&OTELStatus{Code: 1}, SpanStatusOK},
		{&OTELStatus{Code: 2}, SpanStatusError},
		{&OTELStatus{Code: 99}, SpanStatusUnset},
	}

	for _, tc := range tests {
		result := normalizeOTELStatus(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeOTELStatus(%+v): got %s, want %s", tc.input, result, tc.expected)
		}
	}
}

func TestExtractValue(t *testing.T) {
	str := "test"
	i := int64(42)
	f := 3.14
	b := true

	tests := []struct {
		input    ValueUnion
		expected any
	}{
		{ValueUnion{StringValue: &str}, "test"},
		{ValueUnion{IntValue: &i}, int64(42)},
		{ValueUnion{DoubleValue: &f}, 3.14},
		{ValueUnion{BoolValue: &b}, true},
		{ValueUnion{}, nil},
	}

	for _, tc := range tests {
		result := extractValue(tc.input)
		if result != tc.expected {
			t.Errorf("extractValue(%+v): got %v, want %v", tc.input, result, tc.expected)
		}
	}
}

// Test JSON round-trip for metrics
func TestClaudeMetrics_JSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := &ClaudeMetrics{
		SessionID:     "sess-123",
		TaskID:        "task-456",
		Model:         "claude-3-opus",
		TotalTokensIn: 1000,
		TotalTokenOut: 500,
		TotalCostUSD:  0.05,
		DurationMs:    30000,
		TurnCount:     5,
		ToolCalls:     10,
		StartTime:     now,
		EndTime:       now.Add(30 * time.Second),
		Status:        "completed",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, err := ParseClaudeMetricsJSON(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch after round-trip")
	}
	if parsed.TotalTokensIn != original.TotalTokensIn {
		t.Errorf("TotalTokensIn mismatch after round-trip")
	}
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}
