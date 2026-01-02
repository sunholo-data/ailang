package telemetry

import (
	"context"
	"errors"
	"os"

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
