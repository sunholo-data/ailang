package telemetry

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

func TestStartSpan_RecordsWhenEnabled(t *testing.T) {
	// Enable trace recording
	SetTraceRecordingEnabled(true)
	defer SetTraceRecordingEnabled(false)
	effects.ClearGlobalTraces()

	tracer := Tracer("test.tracer")
	ctx := context.Background()

	// Create some spans using StartSpan
	ctx, span1 := StartSpan(ctx, tracer, "compile.parse")
	span1.End()

	_, span2 := StartSpan(ctx, tracer, "compile.typecheck")
	span2.End()

	_, span3 := StartSpan(ctx, tracer, "compile.lower")
	span3.End()

	// Verify spans were recorded to TraceRegistry
	registry := effects.GlobalTraceRegistry()

	if !registry.Exists("compile.parse") {
		t.Error("expected compile.parse to be recorded")
	}
	if !registry.Exists("compile.typecheck") {
		t.Error("expected compile.typecheck to be recorded")
	}
	if !registry.Exists("compile.lower") {
		t.Error("expected compile.lower to be recorded")
	}

	// Verify prefix matching works
	if !registry.Exists("compile") {
		t.Error("expected 'compile' prefix to match compile.* spans")
	}

	// Verify non-existent spans return false
	if registry.Exists("eval.start") {
		t.Error("expected eval.start to NOT exist")
	}
}

func TestStartSpan_NoRecordingWhenDisabled(t *testing.T) {
	// Ensure trace recording is disabled
	SetTraceRecordingEnabled(false)
	effects.ClearGlobalTraces()

	tracer := Tracer("test.tracer")
	ctx := context.Background()

	// Create a span using StartSpan
	_, span := StartSpan(ctx, tracer, "should.not.record")
	span.End()

	// Verify span was NOT recorded
	registry := effects.GlobalTraceRegistry()
	if registry.Exists("should.not.record") {
		t.Error("expected span to NOT be recorded when disabled")
	}
}

func TestRecordSpan_RecordsWhenEnabled(t *testing.T) {
	// Enable trace recording
	SetTraceRecordingEnabled(true)
	defer SetTraceRecordingEnabled(false)
	effects.ClearGlobalTraces()

	// Direct call to RecordSpan
	RecordSpan("manual.span")

	registry := effects.GlobalTraceRegistry()
	if !registry.Exists("manual.span") {
		t.Error("expected manual.span to be recorded")
	}
}

func TestRecordSpan_NoRecordingWhenDisabled(t *testing.T) {
	// Ensure trace recording is disabled
	SetTraceRecordingEnabled(false)
	effects.ClearGlobalTraces()

	// Direct call to RecordSpan
	RecordSpan("manual.span")

	registry := effects.GlobalTraceRegistry()
	if registry.Exists("manual.span") {
		t.Error("expected manual.span to NOT be recorded when disabled")
	}
}

func TestIsTraceRecordingEnabled(t *testing.T) {
	// Test the getter
	SetTraceRecordingEnabled(true)
	if !IsTraceRecordingEnabled() {
		t.Error("expected IsTraceRecordingEnabled to return true")
	}

	SetTraceRecordingEnabled(false)
	if IsTraceRecordingEnabled() {
		t.Error("expected IsTraceRecordingEnabled to return false")
	}
}

func TestTracer_ReturnsValidTracer(t *testing.T) {
	tracer := Tracer("test")
	if tracer == nil {
		t.Error("expected non-nil tracer")
	}

	// Should be able to create spans
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test.span")
	span.End()
}
