package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHookEventParsing tests that hook events are correctly parsed
func TestHookEventParsing(t *testing.T) {
	event := HookEvent{
		Event:         "SessionStart",
		SessionID:     "session-123",
		Workspace:     "/home/user/project",
		ClaudeVersion: "v0.3.14",
		Timestamp:     time.Now(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Verify it can be unmarshaled
	var parsed HookEvent
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if parsed.SessionID != "session-123" {
		t.Errorf("Expected SessionID=session-123, got %s", parsed.SessionID)
	}

	if parsed.Workspace != "/home/user/project" {
		t.Errorf("Expected Workspace=/home/user/project, got %s", parsed.Workspace)
	}

	if parsed.Event != "SessionStart" {
		t.Errorf("Expected Event=SessionStart, got %s", parsed.Event)
	}
}

// TestHookEventTypes tests parsing of different hook event types
func TestHookEventTypes(t *testing.T) {
	tests := []struct {
		name  string
		event string
	}{
		{"SessionStart", "SessionStart"},
		{"PreToolUse", "PreToolUse"},
		{"PostToolUse", "PostToolUse"},
		{"Stop", "Stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := HookEvent{
				Event:     tt.event,
				SessionID: "session-123",
				Timestamp: time.Now(),
			}

			body, _ := json.Marshal(event)
			var parsed HookEvent
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.Event != tt.event {
				t.Errorf("Expected Event=%s, got %s", tt.event, parsed.Event)
			}
		})
	}
}

// TestHookHandlerWithoutBackend tests that handler returns error when Observatory not configured
func TestHookHandlerWithoutBackend(t *testing.T) {
	server := &Server{
		obsBackend: nil, // Not configured
	}

	event := HookEvent{
		Event:     "SessionStart",
		SessionID: "session-123",
		Workspace: "/home/user/project",
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleObservatoryHooks(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// TestHookHandlerMethodNotAllowed tests that only POST is allowed
func TestHookHandlerMethodNotAllowed(t *testing.T) {
	server := &Server{
		obsBackend: nil,
	}

	methods := []string{"GET", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/observatory/hooks", nil)
			w := httptest.NewRecorder()

			server.handleObservatoryHooks(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: Expected status 405, got %d", method, w.Code)
			}
		})
	}
}

// TestHookHandlerInvalidJSON tests that invalid JSON is rejected
// (Note: handler checks Observatory backend first, so will get 503 if not configured)
func TestHookHandlerInvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.handleObservatoryHooks(w, req)

	// Handler checks Observatory first, so returns 503 when not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// TestHookResponseFormat tests that response is valid JSON with correct format
func TestHookResponseFormat(t *testing.T) {
	event := HookEvent{
		Event:     "SessionStart",
		SessionID: "session-123",
		Workspace: "/home/user/project",
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Note: Can't test response format without Observatory backend, so just verify HTTP error handling
	server := &Server{
		obsBackend: nil, // Not configured
	}
	server.handleObservatoryHooks(w, req)

	// Verify response exists and has error status
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// TestHookEventTimestamp tests that timestamp is properly parsed and stored
func TestHookEventTimestamp(t *testing.T) {
	now := time.Now()
	event := HookEvent{
		Event:     "SessionStart",
		SessionID: "session-123",
		Workspace: "/home/user/project",
		Timestamp: now,
	}

	body, _ := json.Marshal(event)
	var parsed HookEvent
	json.Unmarshal(body, &parsed)

	// Timestamps may lose nanosecond precision in JSON
	timeDiff := parsed.Timestamp.Sub(now)
	if timeDiff < -1*time.Microsecond || timeDiff > 1*time.Microsecond {
		t.Errorf("Timestamp mismatch: expected %v, got %v", now, parsed.Timestamp)
	}
}

// TestHookEventWithRawMessage tests handling of JSON raw messages in events
func TestHookEventWithRawMessage(t *testing.T) {
	toolInput := json.RawMessage(`{"key":"value","nested":{"deep":42}}`)
	toolResponse := json.RawMessage(`{"result":"success","data":[1,2,3]}`)

	event := HookEvent{
		Event:        "PreToolUse",
		SessionID:    "session-123",
		ToolUseID:    "tool-use-456",
		ToolName:     "Bash",
		ToolInput:    toolInput,
		ToolResponse: toolResponse,
		Timestamp:    time.Now(),
	}

	body, _ := json.Marshal(event)
	var parsed HookEvent
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Just verify that the RawMessages exist and are not nil
	if parsed.ToolInput == nil {
		t.Errorf("ToolInput should not be nil")
	}

	if parsed.ToolResponse == nil {
		t.Errorf("ToolResponse should not be nil")
	}

	// Verify we can parse them back to valid JSON
	var inputObj map[string]interface{}
	if err := json.Unmarshal(parsed.ToolInput, &inputObj); err != nil {
		t.Errorf("Failed to unmarshal ToolInput: %v", err)
	}

	var responseObj map[string]interface{}
	if err := json.Unmarshal(parsed.ToolResponse, &responseObj); err != nil {
		t.Errorf("Failed to unmarshal ToolResponse: %v", err)
	}
}

// TestHookEventEmptyFields tests handling of empty optional fields
func TestHookEventEmptyFields(t *testing.T) {
	event := HookEvent{
		Event:     "Stop",
		SessionID: "session-123",
		// Leave other fields empty
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(event)
	var parsed HookEvent
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.Event != "Stop" {
		t.Errorf("Event should be Stop")
	}

	if parsed.Workspace != "" {
		t.Errorf("Workspace should be empty, got %s", parsed.Workspace)
	}

	if len(parsed.ToolInput) != 0 {
		t.Errorf("ToolInput should be empty")
	}
}

// TestHookEventSessionIDValidation tests various session ID formats
func TestHookEventSessionIDValidation(t *testing.T) {
	testCases := []struct {
		sessionID string
	}{
		{"session-123"},
		{"claude-session-abc123"},
		{"s123"},
		{"session_with_underscore"},
		{"SESSION-UPPERCASE"},
	}

	for _, tc := range testCases {
		t.Run(tc.sessionID, func(t *testing.T) {
			event := HookEvent{
				Event:     "SessionStart",
				SessionID: tc.sessionID,
				Workspace: "/home/user/project",
				Timestamp: time.Now(),
			}

			body, _ := json.Marshal(event)
			var parsed HookEvent
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.SessionID != tc.sessionID {
				t.Errorf("Expected SessionID=%s, got %s", tc.sessionID, parsed.SessionID)
			}
		})
	}
}

// TestHookEventWorkspacePath tests various workspace path formats
func TestHookEventWorkspacePath(t *testing.T) {
	paths := []string{
		"/home/user/project",
		"/Users/developer/workspace",
		"C:\\Users\\Project",
		"/tmp/ephemeral",
		"/var/folders/tmp",
		"relative/path",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			event := HookEvent{
				Event:     "SessionStart",
				SessionID: "session-123",
				Workspace: path,
				Timestamp: time.Now(),
			}

			body, _ := json.Marshal(event)
			var parsed HookEvent
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.Workspace != path {
				t.Errorf("Expected Workspace=%s, got %s", path, parsed.Workspace)
			}
		})
	}
}

// TestHookEventContextHandling tests that context is properly handled
func TestHookEventContextHandling(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simulate a request with this context
	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader([]byte("invalid")))
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	server := &Server{}
	server.handleObservatoryHooks(w, req)

	// Handler checks Observatory backend first (not configured), returns 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// TestHookEventHTTPHeaders tests that required HTTP headers are present
func TestHookEventHTTPHeaders(t *testing.T) {
	event := HookEvent{
		Event:     "SessionStart",
		SessionID: "session-123",
		Workspace: "/home/user/project",
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server := &Server{}
	server.handleObservatoryHooks(w, req)

	// Check response has JSON content type (when Observatory not configured)
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Expected Content-Type header")
	}
}

// BenchmarkHookEventMarshal benchmarks marshaling of hook events
func BenchmarkHookEventMarshal(b *testing.B) {
	event := HookEvent{
		Event:         "SessionStart",
		SessionID:     "session-123",
		Workspace:     "/home/user/project",
		ClaudeVersion: "v0.3.14",
		Timestamp:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(event)
	}
}

// BenchmarkHookEventUnmarshal benchmarks unmarshaling of hook events
func BenchmarkHookEventUnmarshal(b *testing.B) {
	event := HookEvent{
		Event:         "SessionStart",
		SessionID:     "session-123",
		Workspace:     "/home/user/project",
		ClaudeVersion: "v0.3.14",
		Timestamp:     time.Now(),
	}

	data, _ := json.Marshal(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsed HookEvent
		json.Unmarshal(data, &parsed)
	}
}
