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

// Tracer returns a tracer from the process-global OTel API provider
// (otel.GetTracerProvider().Tracer(name)). With nothing registered that is the
// API's built-in no-op; once internal/platform/otel has installed the SDK
// provider from cmd, spans from the same call export. The lookup is per call,
// so tracers obtained before registration still resolve to the real provider
// afterwards (the global is a delegating provider).
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
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
