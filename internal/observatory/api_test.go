package observatory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestAPI(t *testing.T) (*API, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	backend, err := NewSQLiteBackend(db)
	if err != nil {
		t.Fatalf("NewSQLiteBackend failed: %v", err)
	}
	api := NewAPI(backend)
	cleanup := func() {
		backend.Close()
	}
	return api, cleanup
}

func TestAPI_WorkspaceEndpoints(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create workspace
	workspace := &Workspace{
		ID:   "ws-api-test",
		Name: "API Test Workspace",
		Path: "/tmp/api-test",
	}
	body, _ := json.Marshal(workspace)

	req := httptest.NewRequest("POST", "/api/observatory/workspaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("CreateWorkspace: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get workspace
	req = httptest.NewRequest("GET", "/api/observatory/workspaces/ws-api-test", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetWorkspace: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var retrieved Workspace
	if err := json.NewDecoder(rec.Body).Decode(&retrieved); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if retrieved.Name != "API Test Workspace" {
		t.Errorf("Name mismatch: got %s", retrieved.Name)
	}

	// List workspaces
	req = httptest.NewRequest("GET", "/api/observatory/workspaces", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListWorkspaces: expected 200, got %d", rec.Code)
	}

	var workspaces []*Workspace
	if err := json.NewDecoder(rec.Body).Decode(&workspaces); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(workspaces) != 1 {
		t.Errorf("Expected 1 workspace, got %d", len(workspaces))
	}

	// Delete workspace
	req = httptest.NewRequest("DELETE", "/api/observatory/workspaces/ws-api-test", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("DeleteWorkspace: expected 204, got %d", rec.Code)
	}
}

func TestAPI_TaskEndpoints(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create workspace first
	workspace := &Workspace{ID: "ws-task-api", Name: "Task API Test"}
	if err := api.backend.CreateWorkspace(context.Background(), workspace); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create task
	task := &Task{
		ID:          "task-api-test",
		WorkspaceID: "ws-task-api",
		Title:       "API Test Task",
		SourceType:  TaskSourceMessage,
		Status:      TaskStatusPending,
	}
	body, _ := json.Marshal(task)

	req := httptest.NewRequest("POST", "/api/observatory/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("CreateTask: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get task
	req = httptest.NewRequest("GET", "/api/observatory/tasks/task-api-test", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetTask: expected 200, got %d", rec.Code)
	}

	// List tasks with filter
	req = httptest.NewRequest("GET", "/api/observatory/tasks?workspace_id=ws-task-api&status=pending", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListTasks: expected 200, got %d", rec.Code)
	}

	var tasks []*Task
	if err := json.NewDecoder(rec.Body).Decode(&tasks); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	// Update task
	task.Status = TaskStatusRunning
	body, _ = json.Marshal(task)
	req = httptest.NewRequest("PUT", "/api/observatory/tasks/task-api-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("UpdateTask: expected 200, got %d", rec.Code)
	}
}

func TestAPI_SpanEndpoints(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create span
	now := time.Now()
	span := &Span{
		ID:        "span-api-test",
		TraceID:   "trace-api-test",
		Name:      "api.test",
		Kind:      SpanKindClient,
		Status:    SpanStatusOK,
		StartTime: now,
		Provider:  ProviderClaude,
		TokensIn:  100,
		TokensOut: 50,
	}
	body, _ := json.Marshal(span)

	req := httptest.NewRequest("POST", "/api/observatory/spans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("CreateSpan: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get span
	req = httptest.NewRequest("GET", "/api/observatory/spans/span-api-test", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetSpan: expected 200, got %d", rec.Code)
	}

	// List spans with filters
	req = httptest.NewRequest("GET", "/api/observatory/spans?provider=claude&status=ok", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListSpans: expected 200, got %d", rec.Code)
	}

	// Create span event
	event := &SpanEvent{
		SpanID:    "span-api-test",
		Name:      "tool.call",
		EventType: EventTypeTool,
		ToolName:  "Read",
		Timestamp: now,
	}
	body, _ = json.Marshal(event)

	req = httptest.NewRequest("POST", "/api/observatory/spans/span-api-test/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("CreateSpanEvent: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get span events
	req = httptest.NewRequest("GET", "/api/observatory/spans/span-api-test/events", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetSpanEvents: expected 200, got %d", rec.Code)
	}

	var events []SpanEvent
	if err := json.NewDecoder(rec.Body).Decode(&events); err != nil {
		t.Fatalf("Failed to decode events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestAPI_MessageEndpoints(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create message
	message := &Message{
		ID:          "msg-api-test",
		Inbox:       "user",
		FromAgent:   "agent",
		Title:       "API Test Message",
		Content:     "Test content",
		MessageType: "notification",
		Status:      MessageStatusUnread,
	}
	body, _ := json.Marshal(message)

	req := httptest.NewRequest("POST", "/api/observatory/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("CreateMessage: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get message
	req = httptest.NewRequest("GET", "/api/observatory/messages/msg-api-test", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetMessage: expected 200, got %d", rec.Code)
	}

	// List messages with filter
	req = httptest.NewRequest("GET", "/api/observatory/messages?inbox=user&status=unread", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListMessages: expected 200, got %d", rec.Code)
	}

	var messages []*Message
	if err := json.NewDecoder(rec.Body).Decode(&messages); err != nil {
		t.Fatalf("Failed to decode messages: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Mark as read
	req = httptest.NewRequest("POST", "/api/observatory/messages/msg-api-test/read", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("MarkMessageRead: expected 204, got %d", rec.Code)
	}

	// Verify status changed
	req = httptest.NewRequest("GET", "/api/observatory/messages/msg-api-test", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var updated Message
	json.NewDecoder(rec.Body).Decode(&updated)
	if updated.Status != MessageStatusRead {
		t.Errorf("Status should be read, got %s", updated.Status)
	}

	// Mark as archived
	req = httptest.NewRequest("POST", "/api/observatory/messages/msg-api-test/archive", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("MarkMessageArchived: expected 204, got %d", rec.Code)
	}
}

func TestAPI_TraceEndpoints(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create spans with same trace ID
	now := time.Now()
	spans := []*Span{
		{
			ID:        "span-trace-1",
			TraceID:   "trace-api-123",
			Name:      "root",
			Kind:      SpanKindServer,
			Status:    SpanStatusOK,
			StartTime: now,
			Provider:  ProviderGemini,
			TokensIn:  200,
			TokensOut: 100,
		},
		{
			ID:           "span-trace-2",
			TraceID:      "trace-api-123",
			ParentSpanID: "span-trace-1",
			Name:         "child",
			Kind:         SpanKindClient,
			Status:       SpanStatusOK,
			StartTime:    now.Add(time.Second),
			Provider:     ProviderGemini,
			TokensIn:     100,
			TokensOut:    50,
		},
	}

	for _, span := range spans {
		if err := api.backend.CreateSpan(context.Background(), span); err != nil {
			t.Fatalf("Failed to create span: %v", err)
		}
	}

	// Get trace
	req := httptest.NewRequest("GET", "/api/observatory/traces/trace-api-123", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetTrace: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var trace Trace
	if err := json.NewDecoder(rec.Body).Decode(&trace); err != nil {
		t.Fatalf("Failed to decode trace: %v", err)
	}
	if len(trace.Spans) != 2 {
		t.Errorf("Expected 2 spans in trace, got %d", len(trace.Spans))
	}

	// List traces
	req = httptest.NewRequest("GET", "/api/observatory/traces", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListTraces: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var summaries []*TraceSummary
	if err := json.NewDecoder(rec.Body).Decode(&summaries); err != nil {
		t.Fatalf("Failed to decode summaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("Expected 1 trace summary, got %d", len(summaries))
	}
	if summaries[0].SpanCount != 2 {
		t.Errorf("Expected 2 spans in summary, got %d", summaries[0].SpanCount)
	}
}

func TestAPI_IngestClaude(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	now := time.Now()
	metrics := &ClaudeMetrics{
		SessionID:     "session-ingest-test",
		Model:         "claude-3-sonnet",
		TotalTokensIn: 500,
		TotalTokenOut: 250,
		TotalCostUSD:  0.02,
		DurationMs:    15000,
		TurnCount:     3,
		ToolCalls:     5,
		StartTime:     now,
		EndTime:       now.Add(15 * time.Second),
		Status:        "completed",
		Events: []ClaudeEvent{
			{
				Type:      "tool_call",
				Timestamp: now.Add(5 * time.Second),
				ToolName:  "Read",
			},
		},
	}
	body, _ := json.Marshal(metrics)

	req := httptest.NewRequest("POST", "/api/observatory/ingest/claude", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("IngestClaude: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode result: %v", err)
	}
	if result["span_id"] == "" {
		t.Error("Expected span_id in response")
	}

	// Verify span was created
	span, err := api.backend.GetSpan(context.Background(), result["span_id"])
	if err != nil {
		t.Fatalf("Failed to get created span: %v", err)
	}
	if span.TokensIn != 500 {
		t.Errorf("TokensIn mismatch: got %d", span.TokensIn)
	}
	if span.Provider != ProviderClaude {
		t.Errorf("Provider mismatch: got %s", span.Provider)
	}
}

func TestAPI_MetricsEndpoints(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Create test data - spans for metrics summary
	now := time.Now()
	spans := []*Span{
		{
			ID:        "span-metrics-1",
			TraceID:   "trace-metrics",
			Name:      "test",
			Status:    SpanStatusOK,
			StartTime: now,
			Provider:  ProviderClaude,
			TokensIn:  1000,
			TokensOut: 500,
			CostUSD:   0.05,
		},
		{
			ID:        "span-metrics-2",
			TraceID:   "trace-metrics-2",
			Name:      "test",
			Status:    SpanStatusOK,
			StartTime: now,
			Provider:  ProviderGemini,
			TokensIn:  2000,
			TokensOut: 800,
			CostUSD:   0.03,
		},
	}

	for _, span := range spans {
		if err := api.backend.CreateSpan(context.Background(), span); err != nil {
			t.Fatalf("Failed to create span: %v", err)
		}
	}

	// Get metrics summary
	req := httptest.NewRequest("GET", "/api/observatory/metrics/summary", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetMetricsSummary: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var summary MetricsSummary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatalf("Failed to decode summary: %v", err)
	}
	if summary.TotalSpans != 2 {
		t.Errorf("Expected 2 spans, got %d", summary.TotalSpans)
	}
	if summary.TotalTokensIn != 3000 {
		t.Errorf("Expected 3000 tokens in, got %d", summary.TotalTokensIn)
	}

	// Get provider comparison - returns 200 with empty array when no agent_assignments
	req = httptest.NewRequest("GET", "/api/observatory/metrics/providers", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetProviderComparison: expected 200, got %d", rec.Code)
	}
}

func TestAPI_404(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Non-existent workspace
	req := httptest.NewRequest("GET", "/api/observatory/workspaces/non-existent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent workspace, got %d", rec.Code)
	}

	// Non-existent task
	req = httptest.NewRequest("GET", "/api/observatory/tasks/non-existent", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent task, got %d", rec.Code)
	}
}

func TestAPI_InvalidJSON(t *testing.T) {
	api, cleanup := setupTestAPI(t)
	defer cleanup()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/api/observatory/workspaces", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", rec.Code)
	}
}
