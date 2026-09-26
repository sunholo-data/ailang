package otelplatform

import (
	"context"
	"reflect"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// cloudTraceExportSpan is the span name measured arriving at every dashboard on
// 2026-09-22, at a steady 12/min (the 5s batch delay) from an otherwise idle
// coordinator.
const cloudTraceExportSpan = "google.devtools.cloudtrace.v2.TraceService/BatchWriteSpans"

// The loop is only broken if NOTHING reaches the exporter. Assert on exported
// spans, not on the sampler's return value.
func TestTracerProvider_ExportRPCSpansNeverReachTheExporter(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(dropExportSelfSpans(sdktrace.AlwaysSample())),
		sdktrace.WithSpanProcessor(recorder),
	)
	tracer := tp.Tracer("test")

	_, span := tracer.Start(context.Background(), cloudTraceExportSpan)
	span.End()
	if got := len(recorder.Ended()); got != 0 {
		t.Fatalf("export-RPC span reached the exporter: %d spans, want 0 — the feedback loop is still live", got)
	}

	// Control: without this the test would pass even if the sampler dropped
	// every span, which would be a silent telemetry outage rather than a fix.
	_, real := tracer.Start(context.Background(), "coordinator.DispatchTask")
	real.End()
	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ordinary span count = %d, want 1 (sampler is dropping real work)", len(ended))
	}
	if ended[0].Name() != "coordinator.DispatchTask" {
		t.Fatalf("exported span = %q, want the ordinary one", ended[0].Name())
	}
}

func TestDropExportSelfSpans_DropsEveryKnownExportRPC(t *testing.T) {
	for name := range exportSelfSpanNames {
		t.Run(name, func(t *testing.T) {
			got := dropExportSelfSpans(sdktrace.AlwaysSample()).
				ShouldSample(sdktrace.SamplingParameters{Name: name})
			if got.Decision != sdktrace.Drop {
				t.Errorf("decision = %v, want Drop", got.Decision)
			}
		})
	}
}

// The wrapper must not become a sampler in its own right: every non-export
// decision stays the base sampler's.
func TestDropExportSelfSpans_DelegatesEveryOtherDecisionToBase(t *testing.T) {
	sampler := dropExportSelfSpans(sdktrace.NeverSample())
	if got := sampler.ShouldSample(sdktrace.SamplingParameters{Name: "some.work"}); got.Decision != sdktrace.Drop {
		t.Errorf("NeverSample base: decision = %v, want Drop (not delegating)", got.Decision)
	}
	sampler = dropExportSelfSpans(sdktrace.AlwaysSample())
	if got := sampler.ShouldSample(sdktrace.SamplingParameters{Name: "some.work"}); got.Decision != sdktrace.RecordAndSample {
		t.Errorf("AlwaysSample base: decision = %v, want RecordAndSample (not delegating)", got.Decision)
	}
	if desc := sampler.Description(); !strings.Contains(desc, "AlwaysOnSampler") {
		t.Errorf("Description() = %q, should name the base sampler it wraps", desc)
	}
}

// The cure, as opposed to the guard: the client must not be instrumented at all.
// Pinned by type because the option's effect (internal.DialSettings) is not
// reachable from here.
func TestCloudTraceClientOptions_DisablesDefaultClientTelemetry(t *testing.T) {
	opts := cloudTraceClientOptions()
	if len(opts) == 0 {
		t.Fatal("no client options: the Cloud Trace gRPC client would be instrumented by default")
	}
	for _, o := range opts {
		if strings.Contains(reflect.TypeOf(o).String(), "TelemetryDisabled") {
			return
		}
	}
	t.Fatalf("option.WithTelemetryDisabled() absent from %v — otelgrpc will trace BatchWriteSpans and restart the loop", opts)
}
