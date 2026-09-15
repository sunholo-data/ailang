package telemetry

import "os"

// The four functions here report CONFIGURATION INTENT — which export
// destinations the environment asks for — not registration or delivery. They
// read the environment only, so the language core can consult them (see
// NewEffectSpanWrapper) without linking the SDK. Registration itself lives in
// internal/platform/otel, which reads the same intent through these.

// IsEnabled reports whether an OTLP endpoint is configured.
func IsEnabled() bool { return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" }

// IsGoogleCloudEnabled reports whether a Cloud Trace project is configured.
func IsGoogleCloudEnabled() bool { return GoogleCloudProject() != "" }

// GoogleCloudProject honors telemetry-specific project configuration first.
func GoogleCloudProject() string {
	if p := os.Getenv("OTLP_GOOGLE_CLOUD_PROJECT"); p != "" {
		return p
	}
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}

// IsDualExportEnabled reports whether both destinations are configured.
func IsDualExportEnabled() bool { return IsGoogleCloudEnabled() && IsEnabled() }
