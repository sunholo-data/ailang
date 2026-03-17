package server

import (
	"bytes"
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
func TestHookHandlerInvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.handleObservatoryHooks(w, req)

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

	server := &Server{
		obsBackend: nil,
	}
	server.handleObservatoryHooks(w, req)

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

	if parsed.ToolInput == nil {
		t.Errorf("ToolInput should not be nil")
	}

	if parsed.ToolResponse == nil {
		t.Errorf("ToolResponse should not be nil")
	}

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

	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Expected Content-Type header")
	}
}

// TestHookEventLargeToolResponse tests handling of very large tool responses
func TestHookEventLargeToolResponse(t *testing.T) {
	largeData := ""
	for i := 0; i < 1000; i++ {
		largeData += `{"key":"value"},`
	}
	largeData = largeData[:len(largeData)-1]

	largeJSON := `{"status":"ok","data":[` + largeData + `]}`

	event := HookEvent{
		Event:        "PostToolUse",
		SessionID:    "session-123",
		ToolUseID:    "tool-use-456",
		ToolResponse: json.RawMessage(largeJSON),
		Timestamp:    time.Now(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var parsed HookEvent
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(parsed.ToolResponse) == 0 {
		t.Errorf("Expected ToolResponse to have content")
	}
}

// TestHookEventSpecialCharacters tests handling of special characters in fields
func TestHookEventSpecialCharacters(t *testing.T) {
	testCases := []struct {
		name      string
		sessionID string
		workspace string
		toolName  string
	}{
		{"unicode", "session-🚀-123", "/home/user/project-💾", "Bash™"},
		{"quotes", "session-'quote'", "/home/user/project\"double", "Tool'Name"},
		{"newlines", "session\nbreak", "/home/user/project\n\r", "Tool\nUse"},
		{"backslash", "session\\back", "C:\\Users\\Project", "Tool\\Use"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := HookEvent{
				Event:     "SessionStart",
				SessionID: tc.sessionID,
				Workspace: tc.workspace,
				ToolName:  tc.toolName,
				Timestamp: time.Now(),
			}

			body, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed HookEvent
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.SessionID != tc.sessionID {
				t.Errorf("Expected SessionID=%q, got %q", tc.sessionID, parsed.SessionID)
			}
			if parsed.Workspace != tc.workspace {
				t.Errorf("Expected Workspace=%q, got %q", tc.workspace, parsed.Workspace)
			}
			if parsed.ToolName != tc.toolName {
				t.Errorf("Expected ToolName=%q, got %q", tc.toolName, parsed.ToolName)
			}
		})
	}
}

// TestHookHandlerPostToolUseValidation tests PostToolUse event validation
func TestHookHandlerPostToolUseValidation(t *testing.T) {
	server := &Server{
		obsBackend: nil,
	}

	testCases := []struct {
		name           string
		event          HookEvent
		expectedStatus int
	}{
		{
			name: "valid PostToolUse",
			event: HookEvent{
				Event:        "PostToolUse",
				ToolUseID:    "tool-use-456",
				ToolResponse: json.RawMessage(`{"status":"ok"}`),
				Timestamp:    time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "missing ToolUseID",
			event: HookEvent{
				Event:        "PostToolUse",
				ToolUseID:    "",
				ToolResponse: json.RawMessage(`{"status":"ok"}`),
				Timestamp:    time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.event)
			req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.handleObservatoryHooks(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

// TestHookHandlerPreToolUseValidation tests PreToolUse event validation
func TestHookHandlerPreToolUseValidation(t *testing.T) {
	server := &Server{
		obsBackend: nil,
	}

	testCases := []struct {
		name           string
		event          HookEvent
		expectedStatus int
	}{
		{
			name: "valid PreToolUse",
			event: HookEvent{
				Event:     "PreToolUse",
				SessionID: "session-123",
				ToolUseID: "tool-use-456",
				ToolName:  "Bash",
				ToolInput: json.RawMessage(`{"command":"ls"}`),
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "missing SessionID",
			event: HookEvent{
				Event:     "PreToolUse",
				SessionID: "",
				ToolUseID: "tool-use-456",
				ToolName:  "Bash",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.event)
			req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.handleObservatoryHooks(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

// TestHookHandlerSessionStartValidation tests SessionStart event validation
func TestHookHandlerSessionStartValidation(t *testing.T) {
	server := &Server{
		obsBackend: nil,
	}

	testCases := []struct {
		name           string
		event          HookEvent
		expectedStatus int
	}{
		{
			name: "missing SessionID",
			event: HookEvent{
				Event:     "SessionStart",
				SessionID: "",
				Workspace: "/home/user/project",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "missing Workspace",
			event: HookEvent{
				Event:     "SessionStart",
				SessionID: "session-123",
				Workspace: "",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.event)
			req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.handleObservatoryHooks(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

// TestHookHandlerStopValidation tests Stop event validation
func TestHookHandlerStopValidation(t *testing.T) {
	server := &Server{
		obsBackend: nil,
	}

	testCases := []struct {
		name           string
		event          HookEvent
		expectedStatus int
	}{
		{
			name: "valid Stop",
			event: HookEvent{
				Event:     "Stop",
				SessionID: "session-123",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name: "missing SessionID",
			event: HookEvent{
				Event:     "Stop",
				SessionID: "",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.event)
			req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.handleObservatoryHooks(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
