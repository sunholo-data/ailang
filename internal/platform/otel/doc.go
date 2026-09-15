// Package otelplatform is the OpenTelemetry SDK and its exporters (OTLP HTTP,
// Google Cloud Trace), registered as the process-global providers from cmd/ailang
// and internal/server at startup.
//
// It is NOT the span API: instrumentation (StartSpan, Tracer, Truncate,
// CategorizeError, trace-context propagation, the enable flags) lives in
// internal/telemetry, which depends on the OTel API only and is a no-op until
// this package installs a provider.
//
// Rule: nothing under the language closure (pipeline, eval, effects, builtins,
// format, repl, prompt, loader, link, lsp, vm, gen/golang, smt) may import this
// package — internal/diag/closure_test.go enforces it. The directory is named
// otel after the design doc; the package is otelplatform so call sites that
// also import go.opentelemetry.io/otel need no alias.
package otelplatform
