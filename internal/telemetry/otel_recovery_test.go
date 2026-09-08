package telemetry

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func isolateExporters(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GOOGLE_CLOUD_PROJECT", "OTLP_GOOGLE_CLOUD_PROJECT", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "OTEL_EXPORTER_OTLP_HEADERS", "OTEL_EXPORTER_OTLP_TRACES_HEADERS", "OTEL_EXPORTER_OTLP_METRICS_HEADERS", "OTEL_RESOURCE_ATTRIBUTES"} {
		t.Setenv(key, "")
	}
	traces, meters, propagator := otel.GetTracerProvider(), otel.GetMeterProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(noop.NewTracerProvider())
	t.Cleanup(func() {
		otel.SetTracerProvider(traces)
		otel.SetMeterProvider(meters)
		otel.SetTextMapPropagator(propagator)
	})
}

func TestOTLPRegistersHTTPSWithoutPort(t *testing.T) {
	isolateExporters(t)
	// No DNS lookup or reachability gate is appropriate at registration time.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector.invalid")
	shutdown, err := InitOTLP(context.Background(), "https-registration")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatal("configured HTTPS exporter was disabled before any delivery attempt")
	}
}

func TestOTLPCollectorStartsAfterInitialization(t *testing.T) {
	for _, init := range []struct {
		name string
		fn   func(context.Context, string) (ShutdownFunc, error)
	}{{"otlp", InitOTLP}, {"dual-otlp-leg", InitDual}} {
		t.Run(init.name, func(t *testing.T) {
			isolateExporters(t)
			reservation, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			address := reservation.Addr().String()
			if err := reservation.Close(); err != nil {
				t.Fatal(err)
			}
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+address)
			shutdown, err := init.fn(context.Background(), "recovering-coordinator")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = shutdown(context.Background()) }()
			provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
			if !ok {
				t.Fatal("collector outage at startup permanently disabled the exporter")
			}
			var received atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/traces" {
					received.Add(1)
				}
				w.Header().Set("Content-Type", "application/x-protobuf")
			}))
			_ = server.Listener.Close()
			server.Listener, err = net.Listen("tcp", address)
			if err != nil {
				t.Fatal(err)
			}
			server.Start()
			defer server.Close()
			_, span := provider.Tracer("recovery-test").Start(context.Background(), "coordinator.task.execute")
			span.End()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := provider.ForceFlush(ctx); err != nil {
				t.Fatal(err)
			}
			if received.Load() != 1 {
				t.Fatalf("received %d trace exports after collector recovery; want 1", received.Load())
			}
		})
	}
}

// Proxy configuration is cached process-wide by net/http. A fresh test process
// keeps this fixture hermetic and verifies the SDK's real CONNECT destination.
func TestOTLPHTTPSUsesDefault443(t *testing.T) {
	if os.Getenv("AILANG_TEST_HTTPS_PROXY_CHILD") == "1" {
		isolateExporters(t)
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector.invalid")
		shutdown, err := InitOTLP(context.Background(), "https-port-test")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = shutdown(context.Background()) }()
		provider := otel.GetTracerProvider().(*sdktrace.TracerProvider)
		_, span := provider.Tracer("fixture").Start(context.Background(), "fixture")
		span.End()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := provider.ForceFlush(ctx); err == nil {
			t.Fatal("proxy rejection should surface")
		}
		return
	}
	hosts := make(chan string, 8)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case hosts <- r.Method + " " + r.Host:
		default:
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("https_proxy", proxy.URL)
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")
	t.Setenv("AILANG_TEST_HTTPS_PROXY_CHILD", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, executable, "-test.run=^TestOTLPHTTPSUsesDefault443$", "-test.count=1").CombinedOutput()
	if err != nil {
		t.Fatalf("child: %v: %s", err, output)
	}
	select {
	case host := <-hosts:
		if host != "CONNECT collector.invalid:443" {
			t.Fatalf("unexpected destination: %s", host)
		}
	default:
		t.Fatal("SDK made no HTTPS CONNECT request")
	}
}

func TestOTLPFailedBatchIsBoundedAndLaterBatchRecovers(t *testing.T) {
	isolateExporters(t)
	var healthy atomic.Bool
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		received.Add(1)
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer server.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	shutdown, err := InitOTLP(context.Background(), "batch-recovery")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	provider := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	emit := func() { _, span := provider.Tracer("fixture").Start(context.Background(), "fixture"); span.End() }
	emit()
	started := time.Now()
	if err := provider.ForceFlush(context.Background()); err == nil {
		t.Fatal("failed batch reported success")
	}
	if time.Since(started) > 4*time.Second {
		t.Fatal("three-second export bound exceeded")
	}
	healthy.Store(true)
	emit()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := provider.ForceFlush(ctx); err != nil {
		t.Fatal(err)
	}
	if received.Load() != 1 {
		t.Fatalf("received %d batches", received.Load())
	}
	healthy.Store(false)
	emit()
	started = time.Now()
	if err := shutdown(context.Background()); err == nil {
		t.Fatal("failed final flush reported success")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("two-second shutdown bound exceeded")
	}
}
