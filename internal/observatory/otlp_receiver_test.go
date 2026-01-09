package observatory

import (
	"context"
	"testing"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
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

// TestExtractTaskIDFromCwd tests task ID extraction from worktree paths.
// This is CRITICAL for M-TASK-HIERARCHY - Claude Code doesn't pass env vars to subprocesses,
// so we extract task ID from the worktree cwd path.
func TestExtractTaskIDFromCwd(t *testing.T) {
	tests := []struct {
		name     string
		cwd      string
		expected string
	}{
		// Standard coordinator worktree paths
		{
			name:     "coordinator worktree path",
			cwd:      "/Users/mark/.ailang/state/worktrees/coordinator/task-ea0363c8",
			expected: "task-ea0363c8",
		},
		{
			name:     "coordinator worktree with subdir",
			cwd:      "/Users/mark/.ailang/state/worktrees/coordinator/task-ea0363c8/internal/parser",
			expected: "task-ea0363c8",
		},
		{
			name:     "agent-specific worktree",
			cwd:      "/Users/mark/.ailang/state/worktrees/sprint-executor/task-b734d0f2",
			expected: "task-b734d0f2",
		},
		{
			name:     "design-doc-creator worktree",
			cwd:      "/home/dev/.ailang/state/worktrees/design-doc-creator/task-12345678/docs",
			expected: "task-12345678",
		},

		// Edge cases
		{
			name:     "no worktrees in path",
			cwd:      "/Users/mark/dev/sunholo/ailang",
			expected: "",
		},
		{
			name:     "worktrees but no task",
			cwd:      "/Users/mark/.ailang/state/worktrees/coordinator",
			expected: "",
		},
		{
			name:     "task prefix without full ID - returns prefix",
			cwd:      "/Users/mark/.ailang/state/worktrees/coordinator/task-",
			expected: "task-", // Empty hex chars but still valid prefix
		},
		{
			name:     "empty cwd",
			cwd:      "",
			expected: "",
		},
		{
			name:     "task in different context (not worktrees)",
			cwd:      "/tmp/task-ea0363c8",
			expected: "",
		},
		{
			name:     "multiple worktrees segments",
			cwd:      "/data/worktrees/backup/worktrees/coordinator/task-abcd1234",
			expected: "task-abcd1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := map[string]any{
				"process.cwd": tt.cwd,
			}
			got := extractTaskIDFromCwd(attrs)
			if got != tt.expected {
				t.Errorf("extractTaskIDFromCwd(%q) = %q, want %q", tt.cwd, got, tt.expected)
			}
		})
	}
}

// TestExtractTokensMultipleNamingConventions tests that tokens are extracted from various attribute names.
// Different AI providers use different naming conventions:
// - gen_ai.* (OpenTelemetry semantic conventions)
// - ailang.* (AILANG custom)
// - ai.* (internal/ai/ providers)
func TestExtractTokensMultipleNamingConventions(t *testing.T) {
	tests := []struct {
		name            string
		attrs           map[string]any
		expectedIn      int
		expectedOut     int
		expectedCostUSD float64
	}{
		{
			name: "gen_ai convention (OTEL standard)",
			attrs: map[string]any{
				"gen_ai.usage.input_tokens":  500,
				"gen_ai.usage.output_tokens": 200,
				"gen_ai.usage.cost":          0.05,
			},
			expectedIn:      500,
			expectedOut:     200,
			expectedCostUSD: 0.05,
		},
		{
			name: "ai.* convention (internal/ai/ providers)",
			attrs: map[string]any{
				"ai.tokens_in":  int64(1000),
				"ai.tokens_out": int64(400),
				"ai.cost_usd":   0.10,
			},
			expectedIn:      1000,
			expectedOut:     400,
			expectedCostUSD: 0.10,
		},
		{
			name: "ailang.* convention",
			attrs: map[string]any{
				"ailang.tokens.input":  750,
				"ailang.tokens.output": 300,
				"ailang.cost.usd":      0.08,
			},
			expectedIn:      750,
			expectedOut:     300,
			expectedCostUSD: 0.08,
		},
		{
			name: "task.* convention (coordinator executor spans)",
			attrs: map[string]any{
				"task.tokens_in":  int64(2000),
				"task.tokens_out": int64(800),
				"task.cost_usd":   0.25,
			},
			expectedIn:      2000,
			expectedOut:     800,
			expectedCostUSD: 0.25,
		},
		{
			name: "mixed conventions (first one wins)",
			attrs: map[string]any{
				"gen_ai.usage.input_tokens": 100,
				"ai.tokens_in":              999, // Should be ignored
			},
			expectedIn:  100, // gen_ai wins (first in key list)
			expectedOut: 0,
		},
		{
			name: "string values (Claude Code sends numbers as strings)",
			attrs: map[string]any{
				"ai.tokens_in":  "1234",
				"ai.tokens_out": "567",
				"ai.cost_usd":   "0.123",
			},
			expectedIn:      1234,
			expectedOut:     567,
			expectedCostUSD: 0.123,
		},
		{
			name: "float values",
			attrs: map[string]any{
				"ai.tokens_in":  float64(800),
				"ai.tokens_out": float64(350),
			},
			expectedIn:  800,
			expectedOut: 350,
		},
		{
			name:            "no token attributes",
			attrs:           map[string]any{},
			expectedIn:      0,
			expectedOut:     0,
			expectedCostUSD: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Must match the key list in otlp_receiver.go convertSpan()
			gotIn := extractInt(tt.attrs, "gen_ai.usage.input_tokens", "ailang.tokens.input", "ai.tokens_in", "task.tokens_in")
			gotOut := extractInt(tt.attrs, "gen_ai.usage.output_tokens", "ailang.tokens.output", "ai.tokens_out", "task.tokens_out")
			gotCost := extractFloat(tt.attrs, "gen_ai.usage.cost", "ailang.cost.usd", "ai.cost_usd", "task.cost_usd")

			if gotIn != tt.expectedIn {
				t.Errorf("tokensIn = %d, want %d", gotIn, tt.expectedIn)
			}
			if gotOut != tt.expectedOut {
				t.Errorf("tokensOut = %d, want %d", gotOut, tt.expectedOut)
			}
			if gotCost != tt.expectedCostUSD {
				t.Errorf("costUSD = %f, want %f", gotCost, tt.expectedCostUSD)
			}
		})
	}
}

// TestConvertSpan_TokensFromAIProvider tests that ai.tokens_in/out attributes are properly extracted.
// This verifies the fix for M-TASK-HIERARCHY token display issue.
func TestConvertSpan_TokensFromAIProvider(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "anthropic.generate",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(500 * time.Millisecond).UnixNano()),
		// Span attributes with ai.* naming convention (from internal/ai/ providers)
		Attributes: []*commonpb.KeyValue{
			{Key: "ai.tokens_in", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 1500}}},
			{Key: "ai.tokens_out", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 750}}},
			{Key: "ai.cost_usd", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 0.15}}},
			{Key: "ailang.provider", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "anthropic"}}},
		},
	}

	resourceAttrs := map[string]any{
		"service.name": "ailang-cli",
		"process.cwd":  "/Users/mark/.ailang/state/worktrees/coordinator/task-12345678",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Verify tokens were extracted from ai.* attributes
	if span.TokensIn != 1500 {
		t.Errorf("TokensIn = %d, want 1500", span.TokensIn)
	}
	if span.TokensOut != 750 {
		t.Errorf("TokensOut = %d, want 750", span.TokensOut)
	}
	if span.CostUSD != 0.15 {
		t.Errorf("CostUSD = %f, want 0.15", span.CostUSD)
	}

	// Verify task ID extracted from cwd
	if span.TaskID != "task-12345678" {
		t.Errorf("TaskID = %q, want %q", span.TaskID, "task-12345678")
	}

	// Verify provider mapping
	if span.Provider != ProviderClaude {
		t.Errorf("Provider = %v, want %v", span.Provider, ProviderClaude)
	}
}

// TestConvertSpan_TaskIDFromCwdFallback tests that task ID is extracted from cwd when not in resource attrs.
// This is the M-TASK-HIERARCHY workaround for Claude Code not passing env vars to subprocesses.
func TestConvertSpan_TaskIDFromCwdFallback(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "compile.typecheck", // Use non-background operation (messages.send is now filtered)
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(100 * time.Millisecond).UnixNano()),
	}

	// Resource attributes with cwd in worktree but NO explicit ailang.task_id
	resourceAttrs := map[string]any{
		"service.name": "ailang-compiler",
		"process.cwd":  "/Users/mark/.ailang/state/worktrees/sprint-executor/task-abcd1234/internal",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Task ID should be extracted from cwd path
	if span.TaskID != "task-abcd1234" {
		t.Errorf("TaskID = %q, want %q (should be extracted from cwd)", span.TaskID, "task-abcd1234")
	}
}

// TestConvertSpan_ExplicitTaskIDTakesPrecedence tests that explicit ailang.task_id beats cwd fallback.
func TestConvertSpan_ExplicitTaskIDTakesPrecedence(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "compile.typecheck",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(50 * time.Millisecond).UnixNano()),
	}

	// Resource attributes with BOTH explicit task_id AND cwd with different task
	resourceAttrs := map[string]any{
		"service.name":   "ailang-check",
		"ailang.task_id": "task-explicit", // Explicit - should win
		"process.cwd":    "/Users/mark/.ailang/state/worktrees/coordinator/task-from-cwd",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Explicit ailang.task_id should take precedence
	if span.TaskID != "task-explicit" {
		t.Errorf("TaskID = %q, want %q (explicit should beat cwd fallback)", span.TaskID, "task-explicit")
	}
}

// TestConvertSpan_TaskIDFromTaskIDAttribute tests that task ID is extracted from task.id span attribute.
// This is critical for coordinator.task.execute spans where the coordinator sets task.id directly.
func TestConvertSpan_TaskIDFromTaskIDAttribute(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "coordinator.task.execute",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(60 * time.Second).UnixNano()),
		// Span attributes with task.id (set by coordinator daemon)
		Attributes: []*commonpb.KeyValue{
			{Key: "task.id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "task-12345678"}}},
			{Key: "task.type", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "bug"}}},
			{Key: "task.title", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "Fix the parser"}}},
		},
	}

	// Resource attributes - coordinator runs in main directory, no task ID in cwd
	resourceAttrs := map[string]any{
		"service.name": "ailang-coordinator",
		"process.cwd":  "/Users/mark/dev/sunholo/ailang", // Main dir, no task ID here
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Task ID should be extracted from task.id span attribute
	if span.TaskID != "task-12345678" {
		t.Errorf("TaskID = %q, want %q (should be extracted from task.id)", span.TaskID, "task-12345678")
	}
}

// TestConvertSpan_TaskIDFromWorkspaceAttribute tests that task ID is extracted from task.workspace span attribute.
// This is critical for coordinator executor spans where the coordinator process runs in the main directory
// but the Claude Code subprocess runs in the worktree. The executor sets task.workspace to the worktree path.
func TestConvertSpan_TaskIDFromWorkspaceAttribute(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "claude.execute",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(30 * time.Second).UnixNano()),
		// Span attributes with task.workspace (set by coordinator executor)
		Attributes: []*commonpb.KeyValue{
			{Key: "executor.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "claude"}}},
			{Key: "task.workspace", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "/Users/mark/.ailang/state/worktrees/coordinator/task-abcd1234"}}},
			{Key: "task.tokens_in", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 500}}},
			{Key: "task.tokens_out", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 2000}}},
			{Key: "task.cost_usd", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 0.15}}},
		},
	}

	// Resource attributes with cwd in MAIN directory (NOT worktree)
	// This simulates the coordinator daemon which runs in the main repo
	resourceAttrs := map[string]any{
		"service.name": "ailang-coordinator",
		"process.cwd":  "/Users/mark/dev/sunholo/ailang", // Main dir, no task ID here
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Task ID should be extracted from task.workspace span attribute
	if span.TaskID != "task-abcd1234" {
		t.Errorf("TaskID = %q, want %q (should be extracted from task.workspace)", span.TaskID, "task-abcd1234")
	}

	// Verify tokens were extracted from task.* attributes
	if span.TokensIn != 500 {
		t.Errorf("TokensIn = %d, want 500", span.TokensIn)
	}
	if span.TokensOut != 2000 {
		t.Errorf("TokensOut = %d, want 2000", span.TokensOut)
	}
	if span.CostUSD != 0.15 {
		t.Errorf("CostUSD = %f, want 0.15", span.CostUSD)
	}
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

// TestConvertSpan_CalculatesCostFromTokens tests that cost is calculated when not provided.
// This is M6 from M-TASK-HIERARCHY-FOLLOWUPS: AI providers emit tokens but not cost.
func TestConvertSpan_CalculatesCostFromTokens(t *testing.T) {
	// Reset pricing config for clean test
	ResetPricingConfig()

	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "anthropic.generate",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(2 * time.Second).UnixNano()),
		// Span attributes with tokens but NO cost (like AI providers emit)
		Attributes: []*commonpb.KeyValue{
			{Key: "gen_ai.usage.input_tokens", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 1000}}},
			{Key: "gen_ai.usage.output_tokens", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 1000}}},
			// NO gen_ai.usage.cost - should be calculated from tokens
			{Key: "gen_ai.request.model", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "claude-sonnet-4-5"}}},
		},
	}

	resourceAttrs := map[string]any{
		"service.name": "ailang-eval",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Verify tokens were extracted
	if span.TokensIn != 1000 {
		t.Errorf("TokensIn = %d, want 1000", span.TokensIn)
	}
	if span.TokensOut != 1000 {
		t.Errorf("TokensOut = %d, want 1000", span.TokensOut)
	}

	// Cost should be calculated from tokens using models.yml pricing
	// Claude Sonnet 4.5: $0.003/1K input + $0.015/1K output = $0.018 for 1000+1000 tokens
	// If models.yml is not available, cost will be 0 (no fallback)
	if pricingConfig := GetPricingConfig(); pricingConfig != nil {
		expectedCost := 0.018 // $0.003 + $0.015 = $0.018
		if span.CostUSD < expectedCost*0.99 || span.CostUSD > expectedCost*1.01 {
			t.Errorf("CostUSD = %f, want ~%f (calculated from tokens)", span.CostUSD, expectedCost)
		}
	} else {
		// If models.yml not available, cost should be 0 (no silent fallbacks)
		t.Logf("models.yml not available, skipping cost calculation verification")
	}
}

// TestConvertSpan_DoesNotOverwriteExistingCost tests that existing cost is not overwritten.
func TestConvertSpan_DoesNotOverwriteExistingCost(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "coordinator.task.execute",
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(5 * time.Minute).UnixNano()),
		// Span attributes with tokens AND cost (coordinator aggregates these)
		Attributes: []*commonpb.KeyValue{
			{Key: "task.tokens_in", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 5000}}},
			{Key: "task.tokens_out", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 2000}}},
			{Key: "task.cost_usd", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: 0.25}}}, // Explicit cost
			{Key: "gen_ai.request.model", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "claude-sonnet-4-5"}}},
		},
	}

	resourceAttrs := map[string]any{
		"service.name": "ailang-coordinator",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Explicit cost should be preserved (not overwritten with calculated value)
	if span.CostUSD != 0.25 {
		t.Errorf("CostUSD = %f, want 0.25 (should preserve explicit cost, not calculate)", span.CostUSD)
	}
}

// =============================================================================
// Background Operation Filtering Tests (Phase 10.1)
// =============================================================================
// These tests verify that background operations (messages.list, github_sync, etc.)
// do NOT inherit task_id from CWD extraction. This prevents task hierarchy
// contamination when background operations run in a worktree directory.

// TestBackgroundOperationSpans_MapContents verifies the exclusion map has expected entries.
func TestBackgroundOperationSpans_MapContents(t *testing.T) {
	expected := []string{
		"messages.list",
		"messages.github_sync",
		"messages.ack",
		"messages.send",
		"messages.search",
		"messages.import-github",
	}

	for _, spanName := range expected {
		if !backgroundOperationSpans[spanName] {
			t.Errorf("backgroundOperationSpans should contain %q", spanName)
		}
	}

	// Verify non-background operations are NOT in the map
	nonBackground := []string{
		"compile.typecheck",
		"anthropic.generate",
		"coordinator.task.execute",
		"claude.execute",
	}

	for _, spanName := range nonBackground {
		if backgroundOperationSpans[spanName] {
			t.Errorf("backgroundOperationSpans should NOT contain %q", spanName)
		}
	}
}

// TestConvertSpan_BackgroundOperationNoCwdTaskID verifies that background operations
// do NOT get task_id from CWD path, even when running in a worktree directory.
func TestConvertSpan_BackgroundOperationNoCwdTaskID(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	// Test each background operation span name
	backgroundSpans := []string{
		"messages.list",
		"messages.github_sync",
		"messages.ack",
		"messages.send",
		"messages.search",
		"messages.import-github",
	}

	for _, spanName := range backgroundSpans {
		t.Run(spanName, func(t *testing.T) {
			spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
			traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

			now := time.Now()
			otlpSpan := &tracepb.Span{
				SpanId:            spanID,
				TraceId:           traceID,
				Name:              spanName,
				StartTimeUnixNano: uint64(now.UnixNano()),
				EndTimeUnixNano:   uint64(now.Add(100 * time.Millisecond).UnixNano()),
			}

			// Resource attributes with CWD in worktree but NO explicit ailang.task_id
			// This simulates background operations running in a task's worktree directory
			resourceAttrs := map[string]any{
				"service.name": "ailang-coordinator",
				"process.cwd":  "/Users/mark/.ailang/state/worktrees/coordinator/task-contaminated/internal",
			}

			span := receiver.convertSpan(otlpSpan, resourceAttrs)

			// Task ID should be EMPTY - background operations should NOT inherit from CWD
			if span.TaskID != "" {
				t.Errorf("Background operation %q got TaskID=%q, want empty (should NOT inherit from CWD)", spanName, span.TaskID)
			}
		})
	}
}

// TestConvertSpan_RegularSpanGetsCwdTaskID verifies that regular (non-background) spans
// still get task_id from CWD path when other extraction methods fail.
func TestConvertSpan_RegularSpanGetsCwdTaskID(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	// Regular spans that SHOULD get task_id from CWD
	regularSpans := []string{
		"compile.typecheck",
		"compile.parse",
		"anthropic.generate",
		"ailang.run",
		"ailang.check",
		"coordinator.task.execute",
		"claude.execute",
	}

	for _, spanName := range regularSpans {
		t.Run(spanName, func(t *testing.T) {
			spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
			traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

			now := time.Now()
			otlpSpan := &tracepb.Span{
				SpanId:            spanID,
				TraceId:           traceID,
				Name:              spanName,
				StartTimeUnixNano: uint64(now.UnixNano()),
				EndTimeUnixNano:   uint64(now.Add(100 * time.Millisecond).UnixNano()),
			}

			// Resource attributes with CWD in worktree but NO explicit ailang.task_id
			// Note: task ID must be hex chars (task-XXXXXXXX format)
			resourceAttrs := map[string]any{
				"service.name": "ailang-cli",
				"process.cwd":  "/Users/mark/.ailang/state/worktrees/sprint-executor/task-beef1234",
			}

			span := receiver.convertSpan(otlpSpan, resourceAttrs)

			// Task ID SHOULD be extracted from CWD for regular operations
			if span.TaskID != "task-beef1234" {
				t.Errorf("Regular operation %q got TaskID=%q, want 'task-beef1234' (should inherit from CWD)", spanName, span.TaskID)
			}
		})
	}
}

// TestConvertSpan_BackgroundOperationWithExplicitTaskID verifies that background operations
// CAN have a task_id if it's explicitly set (not from CWD fallback).
func TestConvertSpan_BackgroundOperationWithExplicitTaskID(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "messages.send", // Background operation
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(100 * time.Millisecond).UnixNano()),
		// Span attribute with explicit task.id
		Attributes: []*commonpb.KeyValue{
			{Key: "task.id", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "task-explicit-set"}}},
		},
	}

	// Resource attributes with CWD in worktree (but explicit task.id should win)
	resourceAttrs := map[string]any{
		"service.name": "ailang-coordinator",
		"process.cwd":  "/Users/mark/.ailang/state/worktrees/coordinator/task-from-cwd",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Explicit task.id should be used (not blocked by background operation filter)
	if span.TaskID != "task-explicit-set" {
		t.Errorf("messages.send with explicit task.id got TaskID=%q, want 'task-explicit-set'", span.TaskID)
	}
}

// TestConvertSpan_BackgroundOperationWithResourceTaskID verifies that background operations
// CAN have a task_id if it's in resource attributes (not from CWD fallback).
func TestConvertSpan_BackgroundOperationWithResourceTaskID(t *testing.T) {
	backend := newTestBackend(t)
	receiver := NewOTLPReceiver(backend)

	spanID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	traceID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}

	now := time.Now()
	otlpSpan := &tracepb.Span{
		SpanId:            spanID,
		TraceId:           traceID,
		Name:              "messages.list", // Background operation
		StartTimeUnixNano: uint64(now.UnixNano()),
		EndTimeUnixNano:   uint64(now.Add(50 * time.Millisecond).UnixNano()),
	}

	// Resource attributes with EXPLICIT ailang.task_id (should still work)
	resourceAttrs := map[string]any{
		"service.name":   "ailang-coordinator",
		"ailang.task_id": "task-resource-explicit",
		"process.cwd":    "/Users/mark/.ailang/state/worktrees/coordinator/task-from-cwd",
	}

	span := receiver.convertSpan(otlpSpan, resourceAttrs)

	// Explicit ailang.task_id from resource attrs should be used
	// (background operation filter only affects CWD fallback)
	if span.TaskID != "task-resource-explicit" {
		t.Errorf("messages.list with explicit resource task_id got TaskID=%q, want 'task-resource-explicit'", span.TaskID)
	}
}
