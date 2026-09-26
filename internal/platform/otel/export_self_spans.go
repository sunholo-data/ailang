package otelplatform

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// exportSelfSpanNames are the RPCs an exporter makes in order to SHIP telemetry.
// A span describing one of them is not a record of the service doing work; it is
// a record of the telemetry pipeline running, and it feeds itself: the span is
// queued, the next export ships it, and shipping it emits another. The loop
// sustains itself at the batch interval no matter how idle the service is.
//
// Dropping them at the sampler is the guard, not the cure. The cure is refusing
// the instrumentation at the client (see newCloudExporter); this stays because
// any future exporter that arrives already instrumented would restart the loop,
// and a loop is invisible in the thing it costs — a scale-to-zero service that
// never scales to zero.
var exportSelfSpanNames = map[string]struct{}{
	"google.devtools.cloudtrace.v2.TraceService/BatchWriteSpans": {},
	"google.monitoring.v3.MetricService/CreateTimeSeries":        {},
}

// dropExportSelfSpans wraps base so that telemetry-export RPCs are never
// recorded. Every other sampling decision is base's.
func dropExportSelfSpans(base sdktrace.Sampler) sdktrace.Sampler {
	return exportSelfSpanSampler{base: base}
}

type exportSelfSpanSampler struct{ base sdktrace.Sampler }

func (s exportSelfSpanSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if _, self := exportSelfSpanNames[p.Name]; self {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.Drop,
			Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
		}
	}
	return s.base.ShouldSample(p)
}

func (s exportSelfSpanSampler) Description() string {
	return "DropExportSelfSpans(" + s.base.Description() + ")"
}
