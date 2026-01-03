// Package telemetry provides OpenTelemetry instrumentation for AILANG.
//
// This package enables OTLP export of traces, metrics, and logs to any
// OpenTelemetry-compatible backend (ai-observer, Grafana, Honeycomb, etc.).
//
// # Configuration
//
// Telemetry is opt-in and configured via standard OpenTelemetry environment variables:
//
//	OTEL_SERVICE_NAME=ailang-coordinator
//	OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
//	OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
//	OTEL_TRACES_EXPORTER=otlp
//	OTEL_METRICS_EXPORTER=otlp
//
// # Usage
//
//	ctx := context.Background()
//	shutdown, err := telemetry.InitOTLP(ctx, "ailang-server")
//	if err != nil {
//	    log.Printf("OTEL init failed: %v", err)
//	} else {
//	    defer shutdown(ctx)
//	}
//
// If environment variables are not set, InitOTLP returns a no-op shutdown
// function and nil error, allowing the application to run without telemetry.
package telemetry
