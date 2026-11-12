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

// TestHookEventComplexJSONStructures tests handling of nested JSON in tool input/output
func TestHookEventComplexJSONStructures(t *testing.T) {
	complexInput := json.RawMessage(`{
		"command": "ls",
		"options": {
			"recursive": true,
			"hidden": false,
			"filters": ["*.go", "*.md"]
		},
		"timeout": 30,
		"retries": [
			{"delay": 100, "backoff": 2.0},
			{"delay": 200, "backoff": 2.0}
		]
	}`)

	event := HookEvent{
		Event:     "PreToolUse",
		SessionID: "session-123",
		ToolUseID: "tool-use-456",
		ToolName:  "Bash",
		ToolInput: complexInput,
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(event)
	var parsed HookEvent
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify complex structure is preserved
	var inputObj map[string]interface{}
	if err := json.Unmarshal(parsed.ToolInput, &inputObj); err != nil {
		t.Errorf("Failed to unmarshal ToolInput: %v", err)
	}

	if options, ok := inputObj["options"].(map[string]interface{}); !ok {
		t.Errorf("Expected options to be a map")
	} else if filters, ok := options["filters"].([]interface{}); !ok || len(filters) != 2 {
		t.Errorf("Expected filters array in options")
	}

	if retries, ok := inputObj["retries"].([]interface{}); !ok || len(retries) != 2 {
		t.Errorf("Expected retries array with 2 elements")
	}
}

// TestHookEventNullValues tests handling of null vs empty values
func TestHookEventNullValues(t *testing.T) {
	// Test with explicit null in JSON
	jsonWithNull := []byte(`{
		"event": "PostToolUse",
		"session_id": "session-123",
		"tool_use_id": "tool-use-456",
		"tool_response": null,
		"timestamp": "2024-01-01T00:00:00Z"
	}`)

	var parsed HookEvent
	if err := json.Unmarshal(jsonWithNull, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// null in JSON is preserved as the literal text "null" in RawMessage
	if len(parsed.ToolResponse) > 0 {
		// Verify it's the JSON null value
		if string(parsed.ToolResponse) != "null" {
			t.Errorf("Expected ToolResponse to be 'null' for null JSON value, got %s", string(parsed.ToolResponse))
		}
	}
}

// TestHookEventTimestampPrecision tests timestamp precision handling
func TestHookEventTimestampPrecision(t *testing.T) {
	testTimes := []string{
		"2024-01-15T10:30:45Z",
		"2024-01-15T10:30:45.123Z",
		"2024-01-15T10:30:45.123456Z",
		"2024-01-15T10:30:45.123456789Z",
	}

	for _, timeStr := range testTimes {
		t.Run(timeStr, func(t *testing.T) {
			jsonData := []byte(`{
				"event": "SessionStart",
				"session_id": "session-123",
				"workspace": "/home/user/project",
				"timestamp": "` + timeStr + `"
			}`)

			var parsed HookEvent
			if err := json.Unmarshal(jsonData, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.Event != "SessionStart" {
				t.Errorf("Event mismatch")
			}
		})
	}
}

// TestHookEventUnknownEventType tests handling of unknown event types
func TestHookEventUnknownEventType(t *testing.T) {
	server := &Server{
		obsBackend: nil,
	}

	event := HookEvent{
		Event:     "UnknownEvent",
		SessionID: "session-123",
		Timestamp: time.Now(),
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/observatory/hooks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleObservatoryHooks(w, req)

	// Unknown event types should still get 503 (Observatory not configured)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

// TestHookEventEdgeCaseToolNames tests edge case tool names
func TestHookEventEdgeCaseToolNames(t *testing.T) {
	toolNames := []string{
		"Bash",
		"bash",
		"BASH",
		"Tool-With-Dash",
		"Tool_With_Underscore",
		"Tool.With.Dots",
		"",
	}

	for _, toolName := range toolNames {
		t.Run(toolName, func(t *testing.T) {
			event := HookEvent{
				Event:     "PreToolUse",
				SessionID: "session-123",
				ToolUseID: "tool-use-456",
				ToolName:  toolName,
				Timestamp: time.Now(),
			}

			body, _ := json.Marshal(event)
			var parsed HookEvent
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.ToolName != toolName {
				t.Errorf("Expected ToolName=%q, got %q", toolName, parsed.ToolName)
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

// BenchmarkHookEventWithLargeToolResponse benchmarks handling of large tool responses
func BenchmarkHookEventWithLargeToolResponse(b *testing.B) {
	largeResponse := make([]byte, 100000)
	for i := range largeResponse {
		largeResponse[i] = byte((i % 256))
	}

	event := HookEvent{
		Event:        "PostToolUse",
		SessionID:    "session-123",
		ToolUseID:    "tool-use-456",
		ToolResponse: json.RawMessage(largeResponse),
		Timestamp:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(event)
	}
}

// BenchmarkHookEventWithComplexStructure benchmarks handling of complex nested JSON
func BenchmarkHookEventWithComplexStructure(b *testing.B) {
	complexInput := json.RawMessage(`{
		"command": "ls",
		"options": {
			"recursive": true,
			"hidden": false,
			"filters": ["*.go", "*.md", "*.test"],
			"depth": 5
		},
		"timeout": 30,
		"retries": [
			{"delay": 100, "backoff": 2.0},
			{"delay": 200, "backoff": 2.0},
			{"delay": 400, "backoff": 2.0}
		],
		"metadata": {
			"user": "developer",
			"env": {"PATH": "/usr/bin", "HOME": "/home/dev"}
		}
	}`)

	event := HookEvent{
		Event:     "PreToolUse",
		SessionID: "session-123",
		ToolUseID: "tool-use-456",
		ToolName:  "Bash",
		ToolInput: complexInput,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(event)
	}
}
