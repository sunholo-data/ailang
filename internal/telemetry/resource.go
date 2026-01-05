package telemetry

import (
	"os"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Version is set at build time via ldflags.
var Version = "dev"

// NewResource creates a resource with service information.
// The serviceName is required; other attributes are automatically populated.
// Includes process.cwd for debugging path-related issues (module resolution, etc.)
// Also parses OTEL_RESOURCE_ATTRIBUTES for task hierarchy context (M-TASK-HIERARCHY).
func NewResource(serviceName string) (*resource.Resource, error) {
	// Get working directory at telemetry init time
	// This is critical for debugging module resolution issues (see M-BUG-LOCAL-IMPORTS)
	cwd, _ := os.Getwd()

	// Base attributes
	attrs := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(Version),
		semconv.DeploymentEnvironment(getEnv("OTEL_ENVIRONMENT", "development")),
		semconv.ProcessRuntimeName("go"),
		semconv.ProcessRuntimeVersion(runtime.Version()),
		attribute.String("process.cwd", cwd), // Working directory for debugging
	}

	// Parse OTEL_RESOURCE_ATTRIBUTES from environment (M-TASK-HIERARCHY)
	// This enables task hierarchy linking when AILANG runs inside a coordinator task.
	// Format: key1=value1,key2=value2
	// Example: ailang.task_id=task-123,ailang.assignment_id=aa_456
	if envAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); envAttrs != "" {
		for _, pair := range strings.Split(envAttrs, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key != "" && value != "" {
					attrs = append(attrs, attribute.String(key, value))
				}
			}
		}
	}

	// Create resource without merging with Default() to avoid schema URL conflicts
	// between different semconv versions in dependencies.
	return resource.NewWithAttributes(semconv.SchemaURL, attrs...), nil
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
