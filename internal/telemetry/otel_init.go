package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ExporterState describes registration, deliberately not remote delivery health.
type ExporterState string

const (
	ExporterDisabled   ExporterState = "disabled"
	ExporterRegistered ExporterState = "registered"
	ExporterDegraded   ExporterState = "degraded"
)

// InitializationStatus is the result of one initialization, not a process-global
// configuration guess. Warnings explain configured destinations not registered.
type InitializationStatus struct {
	CloudTrace  ExporterState `json:"cloud_trace"`
	OTLPTraces  ExporterState `json:"otlp_traces"`
	OTLPMetrics ExporterState `json:"otlp_metrics"`
	Warnings    []string      `json:"warnings,omitempty"`
}

func (s InitializationStatus) String() string {
	text := fmt.Sprintf("Cloud Trace=%s; OTLP traces=%s; OTLP metrics=%s; delivery unverified", s.CloudTrace, s.OTLPTraces, s.OTLPMetrics)
	if len(s.Warnings) > 0 {
		text += "; " + strings.Join(s.Warnings, "; ")
	}
	return text
}

type initConfig struct {
	otlp             bool
	cloudProject     string
	cloudTimeout     time.Duration
	newCloudExporter func(string) (sdktrace.SpanExporter, error)
}

// initTelemetry constructs everything before installing providers globally.
// Each resource has exactly one shutdown owner, including partial failure paths.
func initTelemetry(ctx context.Context, serviceName string, cfg initConfig) (shutdown ShutdownFunc, status InitializationStatus, err error) {
	status = InitializationStatus{CloudTrace: ExporterDisabled, OTLPTraces: ExporterDisabled, OTLPMetrics: ExporterDisabled}
	if cfg.cloudProject != "" {
		status.CloudTrace = ExporterDegraded
	}
	if cfg.otlp {
		status.OTLPTraces, status.OTLPMetrics = ExporterDegraded, ExporterDegraded
		for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"} {
			if err := validateEndpoint(key); err != nil {
				return nil, status, err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, status, err
	}
	if !cfg.otlp && cfg.cloudProject == "" {
		return boundedShutdown(nil), status, nil
	}
	res, err := NewResource(serviceName)
	if err != nil {
		return nil, status, err
	}
	var owned []func(context.Context) error
	defer func() {
		if err != nil {
			err = errors.Join(err, boundedShutdown(owned)(context.Background()))
		}
	}()
	var exporters []sdktrace.SpanExporter
	cloudRegistered := false
	if cfg.cloudProject != "" {
		exporter, cloudErr := cloudExporter(ctx, cfg)
		if cloudErr != nil {
			if ctx.Err() != nil || !errors.Is(cloudErr, context.DeadlineExceeded) {
				return nil, status, fmt.Errorf("initialize Cloud Trace: %w", cloudErr)
			}
			status.Warnings = append(status.Warnings, "Cloud Trace initialization timed out; exporter not registered")
		} else {
			exporters = append(exporters, exporter)
			owned = append(owned, exporter.Shutdown)
			cloudRegistered = true
		}
	}
	var metricExporter sdkmetric.Exporter
	if cfg.otlp {
		// No network preflight: SDK HTTP transport owns default ports, TLS and
		// retries. Export failure is bounded and later batches can recover.
		exporter, traceErr := otlptracehttp.New(ctx, otlptracehttp.WithTimeout(3*time.Second))
		if traceErr != nil {
			return nil, status, traceErr
		}
		exporters = append(exporters, exporter)
		owned = append(owned, exporter.Shutdown)
		metricExporter, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithTimeout(3*time.Second))
		if err != nil {
			return nil, status, err
		}
		owned = append(owned, metricExporter.Shutdown)
	}
	// Parent cancellation must not publish partially initialized global state.
	if err := ctx.Err(); err != nil {
		return nil, status, err
	}
	if len(exporters) == 0 {
		return boundedShutdown(nil), status, nil
	}
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if cfg.cloudProject != "" {
		opts = append(opts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
	}
	for _, exporter := range exporters {
		opts = append(opts, sdktrace.WithBatcher(exporter, sdktrace.WithExportTimeout(3*time.Second)))
	}
	traces := sdktrace.NewTracerProvider(opts...)
	// Providers now own exporters; do not also shut exporters down directly.
	owned = []func(context.Context) error{traces.Shutdown}
	if metricExporter != nil {
		meters := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithTimeout(3*time.Second))))
		// Reverse-order cleanup flushes traces first within the shared deadline.
		owned = append([]func(context.Context) error{meters.Shutdown}, owned...)
		otel.SetMeterProvider(meters)
		status.OTLPTraces, status.OTLPMetrics = ExporterRegistered, ExporterRegistered
	}
	otel.SetTracerProvider(traces)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	if cloudRegistered {
		status.CloudTrace = ExporterRegistered
	}
	return boundedShutdown(owned), status, nil
}

// Validate URLs before the SDK can log an invalid setting and use its default.
// Diagnostics intentionally name the setting, not credentials in a supplied URL.
func validateEndpoint(key string) error {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Fragment != "" {
		return fmt.Errorf("%s must be an absolute HTTP or HTTPS URL without a fragment", key)
	}
	return nil
}

func boundedShutdown(funcs []func(context.Context) error) ShutdownFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		var errs []error
		for i := len(funcs) - 1; i >= 0; i-- {
			errs = append(errs, funcs[i](ctx))
		}
		return errors.Join(errs...)
	}
}

// cloudExporter bounds credential discovery. Its unbuffered handoff ensures a
// result arriving after timeout/cancellation is disposed instead of leaked.
func cloudExporter(ctx context.Context, cfg initConfig) (sdktrace.SpanExporter, error) {
	if cfg.cloudTimeout == 0 {
		cfg.cloudTimeout = 3 * time.Second
	}
	if cfg.newCloudExporter == nil {
		cfg.newCloudExporter = func(project string) (sdktrace.SpanExporter, error) {
			return cloudtrace.New(cloudtrace.WithProjectID(project))
		}
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.cloudTimeout)
	defer cancel()
	type result struct {
		exporter sdktrace.SpanExporter
		err      error
	}
	results := make(chan result)
	go func() {
		exporter, err := cfg.newCloudExporter(cfg.cloudProject)
		if exporter == nil && err == nil {
			err = errors.New("Cloud Trace constructor returned no exporter")
		}
		if err == nil {
			select {
			case results <- result{exporter: exporter}:
				return
			case <-ctx.Done():
			}
		}
		if exporter != nil {
			if closeErr := boundedShutdown([]func(context.Context) error{exporter.Shutdown})(context.Background()); closeErr != nil {
				log.Printf("Telemetry: disposing unregistered Cloud Trace exporter: %v", closeErr)
			}
		}
		if err != nil {
			select {
			case results <- result{err: err}:
			case <-ctx.Done():
			}
		}
	}()
	select {
	case r := <-results:
		return r.exporter, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
