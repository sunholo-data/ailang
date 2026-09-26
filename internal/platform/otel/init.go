package otelplatform

import (
	"context"
	"log"

	"github.com/sunholo-data/ailang/internal/telemetry"
)

// ShutdownFunc flushes and shuts down telemetry providers within a bounded deadline.
type ShutdownFunc func(context.Context) error

// InitWithStatus initializes the configured exporters and reports what was actually
// registered. Registration is not proof of delivery; inspect receiver records and
// export errors before drawing a conclusion about capture health.
//
// Configuration intent (which destinations are requested) is read through
// internal/telemetry's enable flags so the core and the platform agree on it.
func InitWithStatus(ctx context.Context, serviceName string) (ShutdownFunc, InitializationStatus, error) {
	return initTelemetry(ctx, serviceName, initConfig{otlp: telemetry.IsEnabled(), cloudProject: telemetry.GoogleCloudProject()})
}

// Init preserves the original initialization API. Call InitWithStatus when showing
// startup status instead of inferring registration from environment variables.
func Init(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	return legacyInitialization(InitWithStatus(ctx, serviceName))
}

// InitOTLP initializes configured OTLP HTTP trace and metric exporters. Transport
// owns endpoint parsing and retries; collector outages do not disable registration.
func InitOTLP(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	return legacyInitialization(initTelemetry(ctx, serviceName, initConfig{otlp: telemetry.IsEnabled()}))
}

// InitGoogleCloudTrace initializes the configured Google Cloud Trace exporter.
// Credential initialization is bounded. A timeout is reported as degradation.
func InitGoogleCloudTrace(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	return legacyInitialization(initTelemetry(ctx, serviceName, initConfig{cloudProject: telemetry.GoogleCloudProject()}))
}

// InitDual initializes both configured destinations, retaining the original API.
func InitDual(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	return legacyInitialization(InitWithStatus(ctx, serviceName))
}

func legacyInitialization(shutdown ShutdownFunc, status InitializationStatus, err error) (ShutdownFunc, error) {
	for _, warning := range status.Warnings {
		log.Printf("Telemetry: %s", warning)
	}
	return shutdown, err
}
