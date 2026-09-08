package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type countedExporter struct{ closed atomic.Int32 }

func (*countedExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }
func (e *countedExporter) Shutdown(context.Context) error                           { e.closed.Add(1); return nil }

func TestInitializationReportsRegisteredNotDelivered(t *testing.T) {
	isolateExporters(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector.invalid")
	shutdown, status, err := InitWithStatus(context.Background(), "report-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	if status.OTLPTraces != ExporterRegistered || status.OTLPMetrics != ExporterRegistered || status.CloudTrace != ExporterDisabled {
		t.Fatalf("incorrect registration report: %+v", status)
	}
	if !strings.Contains(status.String(), "delivery unverified") {
		t.Fatal(status.String())
	}
}

func TestInitializationCloudTimeoutIsDegradedAndLateExporterClosed(t *testing.T) {
	isolateExporters(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	exporter := &countedExporter{}
	release := make(chan struct{})
	defer close(release)
	shutdown, status, err := initTelemetry(context.Background(), "degraded-test", initConfig{
		otlp: true, cloudProject: "fixture-project", cloudTimeout: 10 * time.Millisecond,
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) { <-release; return exporter, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	if status.CloudTrace != ExporterDegraded || status.OTLPTraces != ExporterRegistered || len(status.Warnings) != 1 {
		t.Fatalf("timeout misreported as registration: %+v", status)
	}
	// Release the constructor while this test still owns the exporter, then
	// wait with a deadline for asynchronous disposal of its late result.
	release <- struct{}{}
	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for exporter.closed.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("late exporter leaked")
		case <-tick.C:
		}
	}
	if exporter.closed.Load() != 1 {
		t.Fatal("late exporter closed more than once")
	}
}

func TestInitializationCloudRegisteredAndClosedOnce(t *testing.T) {
	isolateExporters(t)
	exporter := &countedExporter{}
	shutdown, status, err := initTelemetry(context.Background(), "cloud-test", initConfig{
		cloudProject: "fixture-project", cloudTimeout: time.Second,
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) { return exporter, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.CloudTrace != ExporterRegistered {
		t.Fatalf("%+v", status)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if exporter.closed.Load() != 1 {
		t.Fatalf("exporter shutdown called %d times", exporter.closed.Load())
	}
}

func TestInitializationCloudFailureDoesNotReportRegistration(t *testing.T) {
	isolateExporters(t)
	want := errors.New("fixture credentials unavailable")
	_, status, err := initTelemetry(context.Background(), "failure-test", initConfig{
		cloudProject: "fixture-project", cloudTimeout: time.Second,
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) { return nil, want },
	})
	if !errors.Is(err, want) || status.CloudTrace == ExporterRegistered {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}

func TestInitializationNoDestinations(t *testing.T) {
	isolateExporters(t)
	shutdown, status, err := InitWithStatus(context.Background(), "disabled-test")
	if err != nil {
		t.Fatal(err)
	}
	if status.CloudTrace != ExporterDisabled || status.OTLPTraces != ExporterDisabled || status.OTLPMetrics != ExporterDisabled {
		t.Fatalf("%+v", status)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInitializationInvalidEndpointFailsWithoutFallback(t *testing.T) {
	for _, endpoint := range []string{"localhost:4318", "ftp://collector.example", "http://%invalid", "https://collector.example/#fragment"} {
		t.Run(endpoint, func(t *testing.T) {
			isolateExporters(t)
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)
			_, status, err := InitWithStatus(context.Background(), "invalid-endpoint")
			if err == nil || status.OTLPTraces == ExporterRegistered {
				t.Fatalf("invalid endpoint reported registered: %+v, %v", status, err)
			}
		})
	}
}

func TestInitializationCloudErrorDisposesReturnedExporter(t *testing.T) {
	isolateExporters(t)
	exporter := &countedExporter{}
	want := errors.New("fixture constructor failed after allocation")
	_, _, err := initTelemetry(context.Background(), "partial-init", initConfig{
		cloudProject: "fixture", cloudTimeout: time.Second,
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) { return exporter, want },
	})
	if !errors.Is(err, want) || exporter.closed.Load() != 1 {
		t.Fatalf("err=%v shutdowns=%d", err, exporter.closed.Load())
	}
}

func TestInitializationDualRegistration(t *testing.T) {
	isolateExporters(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "application/x-protobuf") }))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	exporter := &countedExporter{}
	shutdown, status, err := initTelemetry(context.Background(), "dual-test", initConfig{
		otlp: true, cloudProject: "fixture",
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) { return exporter, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.CloudTrace != ExporterRegistered || status.OTLPTraces != ExporterRegistered || status.OTLPMetrics != ExporterRegistered {
		t.Fatalf("%+v", status)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if exporter.closed.Load() != 1 {
		t.Fatalf("shutdowns=%d", exporter.closed.Load())
	}
}

func TestInitializationCancellationDisposesAllocation(t *testing.T) {
	isolateExporters(t)
	original := otel.GetTracerProvider()
	exporter := &countedExporter{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, status, err := initTelemetry(ctx, "cancel-test", initConfig{
		cloudProject:     "fixture",
		newCloudExporter: func(string) (sdktrace.SpanExporter, error) { cancel(); return exporter, nil },
	})
	if !errors.Is(err, context.Canceled) || status.CloudTrace == ExporterRegistered || otel.GetTracerProvider() != original {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	deadline := time.Now().Add(time.Second)
	for exporter.closed.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if exporter.closed.Load() != 1 {
		t.Fatalf("shutdowns=%d", exporter.closed.Load())
	}
}
