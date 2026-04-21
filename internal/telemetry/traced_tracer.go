package telemetry

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/sunholo-data/ailang/internal/effects"
)

// traceRecordingEnabled controls whether span names are recorded to TraceRegistry.
// Set AILANG_TRACE_RECORDING=1 to enable.
var traceRecordingEnabled = os.Getenv("AILANG_TRACE_RECORDING") == "1"

// SetTraceRecordingEnabled allows programmatic control of trace recording.
// Primarily used for testing.
func SetTraceRecordingEnabled(enabled bool) {
	traceRecordingEnabled = enabled
}

// IsTraceRecordingEnabled returns whether trace recording is enabled.
func IsTraceRecordingEnabled() bool {
	return traceRecordingEnabled
}

// Tracer returns an OTEL tracer for the given name.
// This is a simple wrapper around otel.Tracer for consistency.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// RecordSpan records a span name to TraceRegistry if recording is enabled.
// Call this after tracer.Start() to bridge OTEL spans to TraceRegistry.
//
// Example:
//
//	ctx, span := tracer.Start(ctx, "compile.parse")
//	telemetry.RecordSpan("compile.parse")
//	defer span.End()
func RecordSpan(spanName string) {
	if traceRecordingEnabled {
		effects.RecordTrace(spanName)
	}
}

// StartSpan creates a span and records it to TraceRegistry if enabled.
// This is the recommended way to create spans when trace recording may be enabled.
//
// Example:
//
//	ctx, span := telemetry.StartSpan(ctx, tracer, "compile.parse")
//	defer span.End()
func StartSpan(ctx context.Context, tracer trace.Tracer, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	RecordSpan(spanName)
	return tracer.Start(ctx, spanName, opts...)
}
