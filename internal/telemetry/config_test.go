package telemetry

import "testing"

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
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tt.endpoint)
			if got := IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoogleCloudProject_TelemetryOverrideWins(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "general")
	t.Setenv("OTLP_GOOGLE_CLOUD_PROJECT", "")
	if got := GoogleCloudProject(); got != "general" {
		t.Fatalf("GoogleCloudProject() = %q, want general", got)
	}
	if !IsGoogleCloudEnabled() {
		t.Fatal("IsGoogleCloudEnabled() = false with GOOGLE_CLOUD_PROJECT set")
	}
	t.Setenv("OTLP_GOOGLE_CLOUD_PROJECT", "telemetry-only")
	if got := GoogleCloudProject(); got != "telemetry-only" {
		t.Fatalf("GoogleCloudProject() = %q, want telemetry-only", got)
	}
}

func TestIsDualExportEnabled(t *testing.T) {
	t.Setenv("OTLP_GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	if IsDualExportEnabled() {
		t.Fatal("dual export reported with nothing configured")
	}
	t.Setenv("GOOGLE_CLOUD_PROJECT", "p")
	if IsDualExportEnabled() {
		t.Fatal("dual export reported with only Cloud Trace configured")
	}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	if !IsDualExportEnabled() {
		t.Fatal("dual export not reported with both configured")
	}
}
