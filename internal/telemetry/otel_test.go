package telemetry

import (
	"context"
	"os"
	"testing"
)

func TestInitOTLP_NoEndpoint(t *testing.T) {
	// Ensure no endpoint is set
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	ctx := context.Background()
	shutdown, err := InitOTLP(ctx, "test-service")
	if err != nil {
		t.Fatalf("InitOTLP failed: %v", err)
	}

	// Shutdown should be a no-op
	if err := shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestInitOTLP_WithEndpoint(t *testing.T) {
	// Set endpoint (we won't actually connect, just verify initialization)
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	ctx := context.Background()
	shutdown, err := InitOTLP(ctx, "test-service")
	if err != nil {
		t.Fatalf("InitOTLP failed: %v", err)
	}

	// Shutdown may fail due to connection refused - that's expected in tests
	// since we don't have an actual OTLP collector running
	_ = shutdown(ctx)
	// We just verify that InitOTLP succeeded - connection errors on shutdown are OK
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     bool
	}{
		{"no endpoint", "", false},
		{"with endpoint", "http://localhost:4318", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.endpoint != "" {
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tt.endpoint)
				defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			} else {
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			}

			if got := IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewResource(t *testing.T) {
	res, err := NewResource("test-service")
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	if res == nil {
		t.Fatal("NewResource returned nil resource")
	}

	// Check that the resource has attributes
	attrs := res.Attributes()
	if len(attrs) == 0 {
		t.Error("NewResource returned resource with no attributes")
	}

	// Verify service name is in attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "service.name" && attr.Value.AsString() == "test-service" {
			found = true
			break
		}
	}
	if !found {
		t.Error("service.name attribute not found in resource")
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{"uses default when not set", "TEST_UNSET_VAR", "default", "", "default"},
		{"uses env when set", "TEST_SET_VAR", "default", "custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			if got := getEnv(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
