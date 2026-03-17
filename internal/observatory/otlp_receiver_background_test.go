package observatory

import (
	"testing"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

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
