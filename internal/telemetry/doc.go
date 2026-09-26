// Package telemetry is AILANG's span API: the instrumentation the language core
// calls (StartSpan, Tracer, Truncate, CategorizeError, LineSnippet,
// NewEffectSpanWrapper, RecordSpan, W3C trace-context propagation across
// subprocesses, and the enable flags that report configuration intent).
//
// It is NOT the OpenTelemetry SDK. It depends on the OTel API only, so with no
// provider registered every span is the API's built-in no-op and the language
// core (`ailang run/check/fmt/repl`) links no exporter, grpc or cloud client.
// The SDK, the OTLP and Google Cloud Trace exporters and the Init* family live
// in internal/platform/otel, which cmd/ailang and internal/server call at
// startup; it installs the global provider that Tracer() then resolves.
//
// Rule: this package stays importable from the language closure, so it must
// never import internal/platform/otel or any go.opentelemetry.io/otel/sdk or
// exporters package (internal/diag/closure_test.go enforces it).
//
// # Configuration
//
// Export is opt-in via standard OpenTelemetry environment variables, read by
// IsEnabled / GoogleCloudProject here and acted on by internal/platform/otel:
//
//	OTEL_SERVICE_NAME=ailang-coordinator
//	OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
//	OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
//	GOOGLE_CLOUD_PROJECT=my-project          # or OTLP_GOOGLE_CLOUD_PROJECT
//
// # Usage
//
//	tracer := telemetry.Tracer("ailang-pipeline")
//	ctx, span := telemetry.StartSpan(ctx, tracer, "compile.parse")
//	defer span.End()
package telemetry
