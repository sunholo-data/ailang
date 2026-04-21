package telemetry_test

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/telemetry"
)

// TestTraceRecording_Integration verifies that the telemetry bridge works:
// when trace recording is enabled, spans created via telemetry.StartSpan
// are recorded to the TraceRegistry.
func TestTraceRecording_Integration(t *testing.T) {
	// Enable trace recording
	telemetry.SetTraceRecordingEnabled(true)
	defer telemetry.SetTraceRecordingEnabled(false)
	effects.ClearGlobalTraces()

	// Get a tracer (this is what the pipeline files use)
	tracer := telemetry.Tracer("integration.test")

	// Create spans using the bridged StartSpan function
	ctx := context.Background()
	ctx, span1 := telemetry.StartSpan(ctx, tracer, "integration.test.phase1")
	span1.End()

	_, span2 := telemetry.StartSpan(ctx, tracer, "integration.test.phase2")
	span2.End()

	// Verify spans were recorded to TraceRegistry
	registry := effects.GlobalTraceRegistry()

	if !registry.Exists("integration.test.phase1") {
		t.Error("expected integration.test.phase1 to be recorded")
	}
	if !registry.Exists("integration.test.phase2") {
		t.Error("expected integration.test.phase2 to be recorded")
	}

	// Verify prefix matching works
	if !registry.Exists("integration.test") {
		t.Error("expected 'integration.test' prefix to match spans")
	}
	if !registry.Exists("integration") {
		t.Error("expected 'integration' prefix to match spans")
	}

	// Verify non-existent spans return false
	if registry.Exists("nonexistent.span") {
		t.Error("expected nonexistent.span to NOT exist")
	}
}

// TestTraceRecording_DisabledByDefault verifies that traces are NOT recorded
// when trace recording is disabled.
func TestTraceRecording_DisabledByDefault(t *testing.T) {
	// Ensure trace recording is disabled
	telemetry.SetTraceRecordingEnabled(false)
	effects.ClearGlobalTraces()

	// Get a tracer and create spans
	tracer := telemetry.Tracer("disabled.test")
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, tracer, "disabled.test.span")
	span.End()

	// Verify spans were NOT recorded
	registry := effects.GlobalTraceRegistry()

	if registry.Exists("disabled.test.span") {
		t.Error("expected disabled.test.span to NOT be recorded when disabled")
	}
	if registry.Exists("disabled.test") {
		t.Error("expected 'disabled.test' prefix to NOT match when disabled")
	}
}
