package telemetry

import (
	"github.com/sunholo-data/ailang/internal/config"
)

// The four functions here report CONFIGURATION INTENT — which export
// destinations the environment asks for — not registration or delivery. They
// read the environment only, so the language core can consult them (see
// NewEffectSpanWrapper) without linking the SDK. Registration itself lives in
// internal/platform/otel, which reads the same intent through these.

// IsEnabled reports whether an OTLP endpoint is configured.
func IsEnabled() bool { return config.OTLPEndpoint() != "" }

// IsGoogleCloudEnabled reports whether a Cloud Trace project is configured.
func IsGoogleCloudEnabled() bool { return GoogleCloudProject() != "" }

// GoogleCloudProject honors telemetry-specific project configuration first.
// Env-only on purpose (see config.TraceProjectFromEnv): a non-empty value is
// what enables Cloud Trace export.
func GoogleCloudProject() string { return config.TraceProjectFromEnv() }

// IsDualExportEnabled reports whether both destinations are configured.
func IsDualExportEnabled() bool { return IsGoogleCloudEnabled() && IsEnabled() }
