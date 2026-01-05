package observatory

import (
	"context"
	"testing"
	"time"

	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// newTestBackend creates an in-memory backend for testing.
func newTestBackend(t *testing.T) Backend {
	t.Helper()
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test backend: %v", err)
	}
	return backend
}

// TestConvertSpan_ExtractsTaskHierarchy tests that task context is extracted from resource attributes.
func TestConvertSpan_ExtractsTaskHierarchy(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	// Create test span with resource attributes containing task context
	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	startNano := uint64(now.UnixNano())
	endNano := uint64(now.Add(100 * time.Millisecond).UnixNano())

	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "test-operation",
		StartTimeUnixNano: startNano,
		EndTimeUnixNano:   endNano,
	}

	// Resource attributes with task hierarchy context
	resourceAttrs := map[string]any{
		"ailang.task_id":       "task-123",
		"ailang.assignment_id": "aa_abc123",
		"ailang.agent_id":      "sprint-executor",
		"ailang.workspace_id":  "ws_xyz789",
		"ailang.source":        "coordinator",
	}

	// Convert span
	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Verify task context was extracted
	if span.TaskID != "task-123" {
		t.Errorf("Expected TaskID='task-123', got '%s'", span.TaskID)
	}
	if span.AgentAssignmentID != "aa_abc123" {
		t.Errorf("Expected AgentAssignmentID='aa_abc123', got '%s'", span.AgentAssignmentID)
	}

	// Verify resource attributes are preserved
	if span.ResourceAttributes["ailang.agent_id"] != "sprint-executor" {
		t.Errorf("Expected resource attr ailang.agent_id='sprint-executor', got '%v'", span.ResourceAttributes["ailang.agent_id"])
	}
}

// TestConvertSpan_NoTaskHierarchy tests spans without task context.
func TestConvertSpan_NoTaskHierarchy(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "standalone-operation",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(50 * time.Millisecond).UnixNano()),
	}

	// Resource attributes without task context
	resourceAttrs := map[string]any{
		"service.name": "ailang-cli",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Verify task context is empty (not set)
	if span.TaskID != "" {
		t.Errorf("Expected empty TaskID, got '%s'", span.TaskID)
	}
	if span.AgentAssignmentID != "" {
		t.Errorf("Expected empty AgentAssignmentID, got '%s'", span.AgentAssignmentID)
	}
}

// TestValidateTaskHierarchy_ValidReferences tests validation with existing task and assignment.
func TestValidateTaskHierarchy_ValidReferences(t *testing.T) {
	backend := newTestBackend(t)
	ctx := context.Background()

	// Create task and assignment
	ws := &Workspace{ID: "ws_test", Name: "test", Path: "/tmp/test", CreatedAt: time.Now()}
	if err := backend.CreateWorkspace(ctx, ws); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	task := &Task{
		ID:          "task-valid",
		WorkspaceID: "ws_test",
		Title:       "Test Task",
		Status:      TaskStatusRunning,
		CreatedAt:   time.Now(),
	}
	if err := backend.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	assignment := &AgentAssignment{
		ID:         "aa_valid",
		TaskID:     "task-valid",
		AgentID:    "test-agent",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	if err := backend.CreateAgentAssignment(ctx, assignment); err != nil {
		t.Fatalf("Failed to create assignment: %v", err)
	}

	// Create receiver and span with valid references
	receiver := NewOTLPReceiver(backend)
	span := &Span{
		ID:                "span-123",
		TaskID:            "task-valid",
		AgentAssignmentID: "aa_valid",
	}

	// Validate - should not log warnings (we can't capture printf in test, but at least verify no panic)
	receiver.validateTaskHierarchy(ctx, span)

	// If we get here without panic, the test passes
}

// TestValidateTaskHierarchy_InvalidReferences tests validation with non-existent references.
func TestValidateTaskHierarchy_InvalidReferences(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)
	ctx := context.Background()

	// Span with references to non-existent task and assignment
	span := &Span{
		ID:                "span-456",
		TaskID:            "nonexistent-task",
		AgentAssignmentID: "aa_nonexistent",
	}

	// Validate - should log warnings but not fail
	receiver.validateTaskHierarchy(ctx, span)

	// If we get here without panic, the test passes
	// In production, warnings would be logged
}

// TestShouldFilterSpan tests the span filtering logic.
func TestShouldFilterSpan(t *testing.T) {
	tests := []struct {
		name          string
		spanName      string
		resourceAttrs map[string]any
		shouldFilter  bool
	}{
		{
			name:         "GCP Trace internal span",
			spanName:     "google.devtools.cloudtrace.v2.TraceService.BatchWriteSpans",
			shouldFilter: true,
		},
		{
			name:         "OTEL SDK internal span",
			spanName:     "opentelemetry.sdk.trace.SpanProcessor",
			shouldFilter: true,
		},
		{
			name:         "Health check endpoint",
			spanName:     "/health",
			shouldFilter: true,
		},
		{
			name:         "Static assets",
			spanName:     "/assets/main.js",
			shouldFilter: true,
		},
		{
			name:         "Polling endpoint",
			spanName:     "/api/observatory/traces",
			shouldFilter: true,
		},
		{
			name:          "Coordinator polling",
			spanName:      "messages.list",
			resourceAttrs: map[string]any{"service.name": "ailang-coordinator"},
			shouldFilter:  true,
		},
		{
			name:         "Normal user operation",
			spanName:     "compile.typecheck",
			shouldFilter: false,
		},
		{
			name:         "AI generation span",
			spanName:     "anthropic.generate",
			shouldFilter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resourceAttrs == nil {
				tt.resourceAttrs = map[string]any{}
			}
			got := shouldFilterSpan(tt.spanName, tt.resourceAttrs)
			if got != tt.shouldFilter {
				t.Errorf("shouldFilterSpan(%q) = %v, want %v", tt.spanName, got, tt.shouldFilter)
			}
		})
	}
}
