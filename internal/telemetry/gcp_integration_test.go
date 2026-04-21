//go:build integration

package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TestGoogleCloudTrace_Integration tests Google Cloud Trace export.
//
// Run with either environment variable (matching Gemini CLI convention):
//
//	OTLP_GOOGLE_CLOUD_PROJECT=multivac-internal-dev go test -tags=integration -run TestGoogleCloudTrace ./internal/telemetry/...
//	GOOGLE_CLOUD_PROJECT=multivac-internal-dev go test -tags=integration -run TestGoogleCloudTrace ./internal/telemetry/...
//
// Then view traces at:
//
//	https://console.cloud.google.com/traces/explorer?project=multivac-internal-dev
func TestGoogleCloudTrace_Integration(t *testing.T) {
	projectID := telemetry.GoogleCloudProject()
	if projectID == "" {
		t.Skip("OTLP_GOOGLE_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT not set, skipping integration test")
	}

	ctx := context.Background()

	// Initialize Google Cloud Trace
	shutdown, err := telemetry.InitGoogleCloudTrace(ctx, "ailang-integration-test")
	if err != nil {
		t.Fatalf("Failed to initialize Google Cloud Trace: %v", err)
	}
	defer shutdown(ctx)

	// Get a tracer
	tracer := otel.Tracer("integration-test")

	// Create a parent span
	ctx, parentSpan := tracer.Start(ctx, "integration-test.parent",
		trace.WithAttributes(
			attribute.String("test.name", "TestGoogleCloudTrace_Integration"),
			attribute.String("test.project", projectID),
		),
	)

	// Simulate some work with child spans
	for i := 0; i < 3; i++ {
		_, childSpan := tracer.Start(ctx, "integration-test.child",
			trace.WithAttributes(
				attribute.Int("child.index", i),
			),
		)
		// Simulate work
		time.Sleep(100 * time.Millisecond)
		childSpan.End()
	}

	// Add some attributes to parent span
	parentSpan.SetAttributes(
		attribute.Int("children.count", 3),
		attribute.String("status", "completed"),
	)

	parentSpan.End()

	// Give the batcher time to export
	t.Logf("Trace created. Waiting for export to Google Cloud...")
	time.Sleep(2 * time.Second)

	t.Logf("✓ Trace exported to project: %s", projectID)
	t.Logf("  View at: https://console.cloud.google.com/traces/explorer?project=%s", projectID)
}

// TestAIProviderTrace_Integration simulates an AI provider call with tracing.
func TestAIProviderTrace_Integration(t *testing.T) {
	projectID := telemetry.GoogleCloudProject()
	if projectID == "" {
		t.Skip("OTLP_GOOGLE_CLOUD_PROJECT or GOOGLE_CLOUD_PROJECT not set, skipping integration test")
	}

	ctx := context.Background()

	// Initialize Google Cloud Trace
	shutdown, err := telemetry.InitGoogleCloudTrace(ctx, "ailang-ai-provider-test")
	if err != nil {
		t.Fatalf("Failed to initialize Google Cloud Trace: %v", err)
	}
	defer shutdown(ctx)

	// Simulate an AI provider call
	tracer := otel.Tracer("ai.test")

	ctx, span := tracer.Start(ctx, "ai.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "test-provider"),
			attribute.String("ai.model", "test-model"),
		),
	)

	// Simulate API call latency
	time.Sleep(200 * time.Millisecond)

	// Record results
	span.SetAttributes(
		attribute.Int("ai.tokens_in", 150),
		attribute.Int("ai.tokens_out", 450),
		attribute.Int("ai.tokens_total", 600),
		attribute.Float64("ai.cost_usd", 0.015),
	)

	span.End()

	// Give the batcher time to export
	t.Logf("AI Provider trace created. Waiting for export...")
	time.Sleep(2 * time.Second)

	t.Logf("✓ Trace exported to project: %s", projectID)
	t.Logf("  View at: https://console.cloud.google.com/traces/explorer?project=%s", projectID)
}
