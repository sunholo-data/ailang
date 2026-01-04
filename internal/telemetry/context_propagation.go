package telemetry

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InjectTraceContext adds W3C trace context to environment variables.
// Returns the updated environment slice with TRACEPARENT and TRACESTATE added.
//
// This enables distributed tracing across subprocess boundaries. When AILANG
// spawns external CLI tools (Claude Code, Gemini CLI), those tools receive
// the trace context via environment variables. Even if the CLI tools don't
// use TRACEPARENT directly, they pass it through to their child processes
// (like `ailang run`), enabling end-to-end trace linking.
//
// Injected variables:
//   - TRACEPARENT: W3C trace context (00-{trace_id}-{span_id}-{flags})
//   - TRACESTATE: Vendor-specific state (if present)
//
// Example:
//
//	env := os.Environ()
//	env = telemetry.InjectTraceContext(ctx, env)
//	cmd.Env = env
func InjectTraceContext(ctx context.Context, env []string) []string {
	carrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))

	for key, value := range carrier {
		// W3C spec uses lowercase, but env vars conventionally uppercase
		// traceparent -> TRACEPARENT, tracestate -> TRACESTATE
		envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		env = append(env, envKey+"="+value)
	}

	return env
}

// InjectCorrelationIDs adds AILANG-specific correlation IDs to environment.
// These serve as fallback correlation when TRACEPARENT isn't supported by
// intermediate processes.
//
// Injected variables:
//   - AILANG_TASK_ID: Coordinator task identifier
//   - AILANG_SESSION_ID: Executor session identifier
//
// These IDs can be used to query related traces even when parent-child
// relationships aren't preserved.
func InjectCorrelationIDs(env []string, taskID, sessionID string) []string {
	if taskID != "" {
		env = append(env, "AILANG_TASK_ID="+taskID)
	}
	if sessionID != "" {
		env = append(env, "AILANG_SESSION_ID="+sessionID)
	}
	return env
}

// ExtractTraceContext reads W3C trace context from environment variables.
// Returns a context with the extracted trace context, or the original context
// if no trace context is found in the environment.
//
// This enables `ailang run` and other commands to participate in distributed
// traces when spawned as subprocesses. The function reads TRACEPARENT and
// TRACESTATE from the process environment.
//
// Example:
//
//	ctx := context.Background()
//	ctx = telemetry.ExtractTraceContext(ctx)
//	// ctx now contains parent trace context if TRACEPARENT was set
func ExtractTraceContext(ctx context.Context) context.Context {
	carrier := make(map[string]string)

	// Read trace context from environment
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		// Only extract known propagation headers
		// TRACEPARENT -> traceparent, TRACESTATE -> tracestate
		if key == "traceparent" || key == "tracestate" {
			carrier[key] = parts[1]
		}
	}

	if len(carrier) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
}

// ExtractCorrelationIDs reads AILANG-specific correlation IDs from environment.
// Returns the task ID and session ID if present, empty strings otherwise.
//
// These IDs can be recorded as span attributes for post-run trace correlation.
func ExtractCorrelationIDs() (taskID, sessionID string) {
	return os.Getenv("AILANG_TASK_ID"), os.Getenv("AILANG_SESSION_ID")
}

// ExtractTraceContextFromEnv reads W3C trace context from a provided environment slice.
// This is useful when you have an explicit environment rather than os.Environ().
//
// Returns a context with the extracted trace context, or the original context
// if no trace context is found.
func ExtractTraceContextFromEnv(ctx context.Context, env []string) context.Context {
	carrier := make(map[string]string)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		if key == "traceparent" || key == "tracestate" {
			carrier[key] = parts[1]
		}
	}

	if len(carrier) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
}

// HasTraceContext checks if the current context has valid trace context.
// Returns true if a trace ID is present in the context.
func HasTraceContext(ctx context.Context) bool {
	carrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))
	_, hasTraceparent := carrier["traceparent"]
	return hasTraceparent
}
