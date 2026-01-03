package telemetry

import (
	"os"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Version is set at build time via ldflags.
var Version = "dev"

// NewResource creates a resource with service information.
// The serviceName is required; other attributes are automatically populated.
// Includes process.cwd for debugging path-related issues (module resolution, etc.)
func NewResource(serviceName string) (*resource.Resource, error) {
	// Get working directory at telemetry init time
	// This is critical for debugging module resolution issues (see M-BUG-LOCAL-IMPORTS)
	cwd, _ := os.Getwd()

	// Create resource without merging with Default() to avoid schema URL conflicts
	// between different semconv versions in dependencies.
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(Version),
		semconv.DeploymentEnvironment(getEnv("OTEL_ENVIRONMENT", "development")),
		semconv.ProcessRuntimeName("go"),
		semconv.ProcessRuntimeVersion(runtime.Version()),
		attribute.String("process.cwd", cwd), // Working directory for debugging
	), nil
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
