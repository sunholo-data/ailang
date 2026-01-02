package telemetry

import (
	"context"
	"errors"
	"os"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ShutdownFunc is a function that shuts down telemetry providers.
type ShutdownFunc func(context.Context) error

// InitOTLP initializes OpenTelemetry with OTLP exporters for traces and metrics.
//
// It returns a shutdown function that should be called on application exit
// to flush pending telemetry data.
//
// If the OTEL_EXPORTER_OTLP_ENDPOINT environment variable is not set,
// this function returns a no-op shutdown and nil error, allowing the
// application to run without telemetry.
//
// Example:
//
//	shutdown, err := telemetry.InitOTLP(ctx, "ailang-coordinator")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(ctx)
func InitOTLP(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	// Check if OTLP endpoint is configured
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		// No endpoint configured - return no-op
		return func(context.Context) error { return nil }, nil
	}

	var shutdownFuncs []func(context.Context) error

	// Create resource
	res, err := NewResource(serviceName)
	if err != nil {
		return nil, err
	}

	// Setup trace exporter
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, traceExporter.Shutdown)

	// Create trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Setup metric exporter
	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, metricExporter.Shutdown)

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Set text map propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return combined shutdown function
	return func(ctx context.Context) error {
		var errs []error
		// Shutdown in reverse order
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			if err := shutdownFuncs[i](ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, nil
}

// IsEnabled returns true if OTLP telemetry is configured.
func IsEnabled() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != ""
}

// IsGoogleCloudEnabled returns true if Google Cloud Trace is configured.
// Matches Gemini CLI convention: OTLP_GOOGLE_CLOUD_PROJECT takes precedence.
func IsGoogleCloudEnabled() bool {
	return os.Getenv("OTLP_GOOGLE_CLOUD_PROJECT") != "" || os.Getenv("GOOGLE_CLOUD_PROJECT") != ""
}

// GoogleCloudProject returns the configured Google Cloud project for telemetry.
// Priority: OTLP_GOOGLE_CLOUD_PROJECT > GOOGLE_CLOUD_PROJECT
func GoogleCloudProject() string {
	if p := os.Getenv("OTLP_GOOGLE_CLOUD_PROJECT"); p != "" {
		return p
	}
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}

// InitGoogleCloudTrace initializes OpenTelemetry with Google Cloud Trace exporter.
//
// This uses Application Default Credentials (ADC) for authentication.
// Environment variables (matching Gemini CLI convention):
//   - OTLP_GOOGLE_CLOUD_PROJECT: Telemetry-specific project (takes precedence)
//   - GOOGLE_CLOUD_PROJECT: General GCP project (fallback)
//
// Traces will appear in:
// https://console.cloud.google.com/traces/explorer?project=YOUR_PROJECT
//
// Example:
//
//	# Option 1: Same project for inference and telemetry
//	export GOOGLE_CLOUD_PROJECT=multivac-internal-dev
//
//	# Option 2: Separate telemetry project
//	export OTLP_GOOGLE_CLOUD_PROJECT=my-telemetry-project
//
//	shutdown, err := telemetry.InitGoogleCloudTrace(ctx, "ailang-coordinator")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(ctx)
func InitGoogleCloudTrace(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	projectID := GoogleCloudProject()
	if projectID == "" {
		// No project configured - return no-op
		return func(context.Context) error { return nil }, nil
	}

	var shutdownFuncs []func(context.Context) error

	// Create resource
	res, err := NewResource(serviceName)
	if err != nil {
		return nil, err
	}

	// Create Google Cloud Trace exporter (uses ADC automatically)
	traceExporter, err := cloudtrace.New(cloudtrace.WithProjectID(projectID))
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, traceExporter.Shutdown)

	// Create trace provider with synced exporter for immediate visibility
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		// Sample all traces for debugging
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set text map propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return combined shutdown function
	return func(ctx context.Context) error {
		var errs []error
		// Shutdown in reverse order
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			if err := shutdownFuncs[i](ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, nil
}

// Init initializes telemetry based on environment configuration.
// It checks for Google Cloud first, then falls back to generic OTLP.
//
// Priority:
// 1. GOOGLE_CLOUD_PROJECT → Google Cloud Trace
// 2. OTEL_EXPORTER_OTLP_ENDPOINT → Generic OTLP
// 3. Neither → No-op (telemetry disabled)
func Init(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	if IsGoogleCloudEnabled() {
		return InitGoogleCloudTrace(ctx, serviceName)
	}
	return InitOTLP(ctx, serviceName)
}

// Tracer returns a tracer for the given instrumentation scope.
// This is a convenience wrapper around otel.Tracer.
func Tracer(name string) interface{ Tracer() } {
	return tracerWrapper{name: name}
}

type tracerWrapper struct {
	name string
}

func (t tracerWrapper) Tracer() {
	// This is just for documentation - use otel.Tracer(name) directly
}
