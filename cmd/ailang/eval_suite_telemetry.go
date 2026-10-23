package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	otelplatform "github.com/sunholo-data/ailang/internal/platform/otel"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// startEvalSuiteTelemetry preserves ordinary suite correlation and trace setup.
// Maintenance dry-run returns before this helper, including credential discovery.
func startEvalSuiteTelemetry() (context.Context, trace.Span, string, string, func()) {
	ctx := context.Background()

	// Check if we already have a task ID from coordinator/parent process
	taskID, sessionID := telemetry.ExtractCorrelationIDs()

	// If no task ID provided, generate one for this eval run
	// This enables eval runs to appear in the Observatory task hierarchy
	// Generate assignment ID as well - needed for span grouping in UI
	var assignmentID string
	if taskID == "" {
		taskID = fmt.Sprintf("eval-%d", time.Now().UnixNano())
		assignmentID = fmt.Sprintf("aa_%d", time.Now().UnixNano())
		// Set in environment so it gets picked up by otelplatform.NewResource()
		// Both task_id and assignment_id are needed for full hierarchy visibility
		existingAttrs := config.OTELResourceAttributes()
		newAttrs := fmt.Sprintf("ailang.task_id=%s,ailang.assignment_id=%s", taskID, assignmentID)
		if existingAttrs != "" {
			os.Setenv(config.EnvOTELResourceAttrs, existingAttrs+","+newAttrs)
		} else {
			os.Setenv(config.EnvOTELResourceAttrs, newAttrs)
		}
	}

	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	shutdownTelemetry, err := otelplatform.Init(ctx, "ailang-eval")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
		shutdownTelemetry = nil
	}

	// Extract trace context from environment (enables cross-process trace linking)
	// If TRACEPARENT is set (e.g., by coordinator or CI), this eval suite will be
	// a child span in the parent trace
	ctx = telemetry.ExtractTraceContext(ctx)

	// Start parent span for the entire eval suite
	ctx, suiteSpan := evalTracer.Start(ctx, "eval.suite")

	// Record correlation IDs as span attributes for fallback trace linking
	if taskID != "" {
		suiteSpan.SetAttributes(attribute.String("ailang.task_id", taskID))
	}
	if sessionID != "" {
		suiteSpan.SetAttributes(attribute.String("ailang.session_id", sessionID))
	}

	return ctx, suiteSpan, taskID, assignmentID, func() {
		suiteSpan.End()
		if shutdownTelemetry != nil {
			shutdownTelemetry(ctx)
		}
	}
}
