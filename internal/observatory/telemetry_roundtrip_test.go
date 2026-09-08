package observatory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Exercise real SDK protobuf transport -> production OTLP auth/conversion ->
// persistent Backend reads. No artificial pass based on an HTTP 200 alone.
func TestTelemetryExporterReceiverPreservesWorkProvenance(t *testing.T) {
	for _, key := range []string{"GOOGLE_CLOUD_PROJECT", "OTLP_GOOGLE_CLOUD_PROJECT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "OTEL_EXPORTER_OTLP_HEADERS", "OTEL_EXPORTER_OTLP_METRICS_HEADERS", "AILANG_OTLP_FILTER_DISABLE", "AILANG_OTLP_FILTER_DENY", "AILANG_OTLP_FILTER_ALLOW"} {
		t.Setenv(key, "")
	}
	t.Setenv(OTLPIngestTokenEnv, "isolated-test-token")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", OTLPIngestTokenHeader+"=isolated-test-token")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "ailang.task_id=task-roundtrip,ailang.chain_id=chain-roundtrip,ailang.stage_id=stage-roundtrip")
	ctx := context.Background()
	backend := newTestBackend(t)
	defer backend.Close()
	if err := backend.CreateWorkspace(ctx, &Workspace{ID: "ws-roundtrip", Name: "fixture", Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := backend.CreateTask(ctx, &Task{ID: "task-roundtrip", WorkspaceID: "ws-roundtrip", Title: "fixture", Status: TaskStatusRunning, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.CreateChain(ctx, &ChainCreateRequest{ID: "chain-roundtrip", SourceType: ChainSourceManual, SourceRef: "isolated-test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.CreateStage(ctx, &StageCreateRequest{ID: "stage-roundtrip", ChainID: "chain-roundtrip", AgentID: "fixture", TaskID: "task-roundtrip"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewOTLPReceiver(backend).RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	oldTraces, oldMeters, oldPropagation := otel.GetTracerProvider(), otel.GetMeterProvider(), otel.GetTextMapPropagator()
	defer func() {
		otel.SetTracerProvider(oldTraces)
		otel.SetMeterProvider(oldMeters)
		otel.SetTextMapPropagator(oldPropagation)
	}()
	shutdown, status, err := telemetry.InitWithStatus(ctx, "ailang-coordinator-fixture")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(ctx) }()
	if status.OTLPTraces != telemetry.ExporterRegistered {
		t.Fatalf("%+v", status)
	}
	provider := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	tracer := provider.Tracer("coordinator-fixture")
	parentCtx, parent := tracer.Start(ctx, "coordinator.task.execute")
	_, child := tracer.Start(parentCtx, "agent.execute", trace.WithAttributes(attribute.Int("task.tokens_in", 12), attribute.Int("task.tokens_out", 3)))
	childID, parentID, traceID := child.SpanContext().SpanID().String(), parent.SpanContext().SpanID().String(), parent.SpanContext().TraceID().String()
	child.End()
	parent.End()
	flushCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := provider.ForceFlush(flushCtx); err != nil {
		t.Fatal(err)
	}
	stored, err := backend.GetSpan(ctx, childID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.TaskID != "task-roundtrip" || stored.ChainID != "chain-roundtrip" || stored.StageID != "stage-roundtrip" || stored.ParentSpanID != parentID || stored.TraceID != traceID {
		t.Fatalf("provenance lost: %+v", stored)
	}
	if stored.TokensIn != 12 || stored.TokensOut != 3 {
		t.Fatalf("accounting lost: %+v", stored)
	}
}
