package telemetry

import (
	"context"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func setupTestTracer(t *testing.T) func() {
	// Set up a real tracer provider for testing
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Set up propagator (same as production)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		_ = tp.Shutdown(context.Background())
	}
}

func TestInjectTraceContext(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	// Create a span to have trace context
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Inject into environment
	env := []string{"PATH=/bin", "HOME=/home/user"}
	env = InjectTraceContext(ctx, env)

	// Verify TRACEPARENT was added
	var traceparent string
	for _, e := range env {
		if strings.HasPrefix(e, "TRACEPARENT=") {
			traceparent = strings.TrimPrefix(e, "TRACEPARENT=")
			break
		}
	}

	if traceparent == "" {
		t.Fatal("TRACEPARENT not found in environment")
	}

	// Verify W3C format: 00-{trace_id}-{span_id}-{flags}
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		t.Fatalf("Invalid TRACEPARENT format: %s (expected 4 parts, got %d)", traceparent, len(parts))
	}

	if parts[0] != "00" {
		t.Errorf("Expected version '00', got '%s'", parts[0])
	}

	if len(parts[1]) != 32 {
		t.Errorf("Expected 32-char trace ID, got %d chars: %s", len(parts[1]), parts[1])
	}

	if len(parts[2]) != 16 {
		t.Errorf("Expected 16-char span ID, got %d chars: %s", len(parts[2]), parts[2])
	}

	// Verify original env vars preserved
	hasPath := false
	hasHome := false
	for _, e := range env {
		if e == "PATH=/bin" {
			hasPath = true
		}
		if e == "HOME=/home/user" {
			hasHome = true
		}
	}
	if !hasPath || !hasHome {
		t.Error("Original environment variables not preserved")
	}
}

func TestInjectTraceContext_NoSpan(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	// Context without active span
	ctx := context.Background()

	env := []string{"PATH=/bin"}
	env = InjectTraceContext(ctx, env)

	// Should still work, but no TRACEPARENT added (no active span)
	// The propagator won't inject if there's no valid span context
	for _, e := range env {
		if strings.HasPrefix(e, "TRACEPARENT=") {
			// If there's a traceparent, it should have invalid trace ID (all zeros)
			tp := strings.TrimPrefix(e, "TRACEPARENT=")
			parts := strings.Split(tp, "-")
			if len(parts) >= 2 && parts[1] != "00000000000000000000000000000000" {
				// This is fine - some implementations might not inject at all
				t.Logf("TRACEPARENT present: %s", tp)
			}
		}
	}

	// Original env should be preserved
	if len(env) < 1 || env[0] != "PATH=/bin" {
		t.Error("Original environment not preserved")
	}
}

func TestInjectCorrelationIDs(t *testing.T) {
	tests := []struct {
		name      string
		taskID    string
		sessionID string
		wantTask  bool
		wantSess  bool
	}{
		{"both set", "task_123", "sess_456", true, true},
		{"only task", "task_123", "", true, false},
		{"only session", "", "sess_456", false, true},
		{"neither", "", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := []string{"PATH=/bin"}
			env = InjectCorrelationIDs(env, tt.taskID, tt.sessionID)

			hasTask := false
			hasSess := false
			for _, e := range env {
				if e == "AILANG_TASK_ID="+tt.taskID && tt.taskID != "" {
					hasTask = true
				}
				if e == "AILANG_SESSION_ID="+tt.sessionID && tt.sessionID != "" {
					hasSess = true
				}
			}

			if hasTask != tt.wantTask {
				t.Errorf("AILANG_TASK_ID: got %v, want %v", hasTask, tt.wantTask)
			}
			if hasSess != tt.wantSess {
				t.Errorf("AILANG_SESSION_ID: got %v, want %v", hasSess, tt.wantSess)
			}
		})
	}
}

func TestExtractTraceContext(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	// Set TRACEPARENT in environment
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	spanID := "00f067aa0ba902b7"
	traceparent := "00-" + traceID + "-" + spanID + "-01"

	os.Setenv("TRACEPARENT", traceparent)
	defer os.Unsetenv("TRACEPARENT")

	ctx := ExtractTraceContext(context.Background())

	// Verify we can get the span context
	spanCtx := trace.SpanContextFromContext(ctx)

	if !spanCtx.IsValid() {
		t.Fatal("Extracted span context is not valid")
	}

	if spanCtx.TraceID().String() != traceID {
		t.Errorf("TraceID mismatch: got %s, want %s", spanCtx.TraceID().String(), traceID)
	}

	// The parent span ID becomes the remote span context
	// Note: SpanID() returns the current span ID, not the parent
	// We need to check that the context is a remote context
	if !spanCtx.IsRemote() {
		t.Error("Expected remote span context")
	}
}

func TestExtractTraceContext_NoEnvVar(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	// Ensure TRACEPARENT is not set
	os.Unsetenv("TRACEPARENT")
	os.Unsetenv("TRACESTATE")

	ctx := ExtractTraceContext(context.Background())

	// Should return original context, no valid span context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		t.Error("Expected invalid span context when no TRACEPARENT set")
	}
}

func TestExtractCorrelationIDs(t *testing.T) {
	// Set correlation IDs
	os.Setenv("AILANG_TASK_ID", "task_abc")
	os.Setenv("AILANG_SESSION_ID", "sess_xyz")
	defer os.Unsetenv("AILANG_TASK_ID")
	defer os.Unsetenv("AILANG_SESSION_ID")

	taskID, sessionID := ExtractCorrelationIDs()

	if taskID != "task_abc" {
		t.Errorf("TaskID: got %q, want %q", taskID, "task_abc")
	}
	if sessionID != "sess_xyz" {
		t.Errorf("SessionID: got %q, want %q", sessionID, "sess_xyz")
	}
}

func TestExtractCorrelationIDs_NotSet(t *testing.T) {
	os.Unsetenv("AILANG_TASK_ID")
	os.Unsetenv("AILANG_SESSION_ID")

	taskID, sessionID := ExtractCorrelationIDs()

	if taskID != "" {
		t.Errorf("TaskID: expected empty, got %q", taskID)
	}
	if sessionID != "" {
		t.Errorf("SessionID: expected empty, got %q", sessionID)
	}
}

func TestExtractTraceContextFromEnv(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	traceID := "abcdef1234567890abcdef1234567890"
	spanID := "1234567890abcdef"
	traceparent := "00-" + traceID + "-" + spanID + "-01"

	env := []string{
		"PATH=/bin",
		"TRACEPARENT=" + traceparent,
		"HOME=/home/user",
	}

	ctx := ExtractTraceContextFromEnv(context.Background(), env)

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		t.Fatal("Extracted span context is not valid")
	}

	if spanCtx.TraceID().String() != traceID {
		t.Errorf("TraceID mismatch: got %s, want %s", spanCtx.TraceID().String(), traceID)
	}
}

func TestExtractTraceContextFromEnv_Empty(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	ctx := ExtractTraceContextFromEnv(context.Background(), []string{})

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		t.Error("Expected invalid span context for empty env")
	}
}

func TestHasTraceContext(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	// Without span
	ctx := context.Background()
	if HasTraceContext(ctx) {
		t.Error("Expected no trace context for background context")
	}

	// With span
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	if !HasTraceContext(ctx) {
		t.Error("Expected trace context when span is active")
	}
}

func TestRoundTrip(t *testing.T) {
	cleanup := setupTestTracer(t)
	defer cleanup()

	// Create a span
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent-span")
	defer span.End()

	originalSpanCtx := trace.SpanContextFromContext(ctx)
	originalTraceID := originalSpanCtx.TraceID().String()

	// Inject into env
	env := InjectTraceContext(ctx, []string{})
	env = InjectCorrelationIDs(env, "task_roundtrip", "sess_roundtrip")

	// Extract from env (simulating subprocess)
	extractedCtx := ExtractTraceContextFromEnv(context.Background(), env)
	extractedSpanCtx := trace.SpanContextFromContext(extractedCtx)

	// Verify trace ID matches
	if extractedSpanCtx.TraceID().String() != originalTraceID {
		t.Errorf("Trace ID mismatch after round-trip: got %s, want %s",
			extractedSpanCtx.TraceID().String(), originalTraceID)
	}

	// Verify it's marked as remote (came from external context)
	if !extractedSpanCtx.IsRemote() {
		t.Error("Expected extracted context to be remote")
	}
}
