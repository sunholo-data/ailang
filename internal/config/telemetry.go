package config

import (
	"strconv"
	"strings"
)

// OTLP export, the observatory's ingest side, and the correlation ids a
// parent process hands its children.
const (
	EnvOTLPEndpoint          = "OTEL_EXPORTER_OTLP_ENDPOINT"
	EnvOTLPProtocol          = "OTEL_EXPORTER_OTLP_PROTOCOL"
	EnvOTLPTracesEndpoint    = "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"
	EnvOTLPMetricsEndpoint   = "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"
	EnvOTELResourceAttrs     = "OTEL_RESOURCE_ATTRIBUTES"
	EnvOTELEnvironment       = "OTEL_ENVIRONMENT"
	EnvGoogleCloudLocation   = "GOOGLE_CLOUD_LOCATION"
	EnvGeminiTelemetryTarget = "GEMINI_TELEMETRY_TARGET"
	EnvTaskID                = "AILANG_TASK_ID"
	EnvSessionID             = "AILANG_SESSION_ID"
	EnvParentTaskID          = "AILANG_PARENT_TASK_ID"
	EnvOTLPIngestToken       = "AILANG_OTLP_INGEST_TOKEN"
	EnvWALCheckpointMB       = "AILANG_OBSERVATORY_WAL_CHECKPOINT_MB"
	EnvSpanFilterDisable     = "AILANG_SPAN_FILTER_DISABLE"
	EnvSpanFilterAllow       = "AILANG_SPAN_FILTER_ALLOW"
	EnvSpanFilterDeny        = "AILANG_SPAN_FILTER_DENY"
)

// DefaultWALCheckpointMB is the observatory WAL size that triggers a
// checkpoint: well below the 2GB total-DB warning so normal rotation load
// never reaches the manual-intervention state.
const DefaultWALCheckpointMB int64 = 1024

var telemetryVars = []Var{
	{EnvOTLPEndpoint, "", AreaTelemetry, "OTLP collector URL; setting it enables OTLP export, and executors default their children to http://localhost:1957 (the local observatory) when it is unset."},
	{EnvOTLPProtocol, "", AreaTelemetry, "OTLP transport (grpc, http/protobuf) passed through to executor children when set."},
	{EnvOTLPTracesEndpoint, "", AreaTelemetry, "Per-signal override of the traces endpoint; validated as a URL when OTLP export is on."},
	{EnvOTLPMetricsEndpoint, "", AreaTelemetry, "Per-signal override of the metrics endpoint; validated as a URL when OTLP export is on."},
	{EnvOTELResourceAttrs, "", AreaTelemetry, "Comma-separated key=value resource attributes merged into every span; executors extend it with task ids for their children."},
	{EnvOTELEnvironment, "development", AreaTelemetry, "deployment.environment resource attribute."},
	{EnvGoogleCloudLocation, "", AreaTelemetry, "GCP location handed to executors (Vertex / managed agents); empty means the executor's own default."},
	{EnvGeminiTelemetryTarget, "", AreaTelemetry, "Where the Gemini CLI sends telemetry; when unset an executor with a cloud project sets gcp."},
	{EnvTaskID, "", AreaTelemetry, "Coordinator task id of this process, recorded on spans for cross-trace correlation."},
	{EnvSessionID, "", AreaTelemetry, "Session id of this process, recorded on spans."},
	{EnvParentTaskID, "", AreaTelemetry, "Task id of the parent that spawned this ailang process; inherited by check, run and exec for hierarchy linking."},
	{EnvOTLPIngestToken, "", AreaTelemetry, "Shared secret the observatory's OTLP receiver requires from callers; empty disables ingest auth."},
	{EnvWALCheckpointMB, strconv.FormatInt(DefaultWALCheckpointMB, 10), AreaTelemetry, "Observatory SQLite WAL size in MB that triggers a checkpoint; non-positive or malformed keeps the default."},
	{EnvSpanFilterDisable, "false", AreaTelemetry, "true disables the observatory's span filter entirely."},
	{EnvSpanFilterAllow, "", AreaTelemetry, "Comma-separated span-name patterns the observatory keeps."},
	{EnvSpanFilterDeny, "", AreaTelemetry, "Comma-separated span-name patterns the observatory drops."},
}

// OTLPEndpoint returns OTEL_EXPORTER_OTLP_ENDPOINT, "" when unset.
func OTLPEndpoint() string { return get(EnvOTLPEndpoint) }

// OTLPProtocol returns OTEL_EXPORTER_OTLP_PROTOCOL, "" when unset.
func OTLPProtocol() string { return get(EnvOTLPProtocol) }

// OTELResourceAttributes returns OTEL_RESOURCE_ATTRIBUTES verbatim.
func OTELResourceAttributes() string { return get(EnvOTELResourceAttrs) }

// OTELEnvironment returns OTEL_ENVIRONMENT, default development.
func OTELEnvironment() string { return getOr(EnvOTELEnvironment) }

// GoogleCloudLocation returns GOOGLE_CLOUD_LOCATION, "" when unset.
func GoogleCloudLocation() string { return get(EnvGoogleCloudLocation) }

// GeminiTelemetryTarget returns GEMINI_TELEMETRY_TARGET, "" when unset.
func GeminiTelemetryTarget() string { return get(EnvGeminiTelemetryTarget) }

// TaskID returns AILANG_TASK_ID, "" when unset.
func TaskID() string { return get(EnvTaskID) }

// SessionID returns AILANG_SESSION_ID, "" when unset.
func SessionID() string { return get(EnvSessionID) }

// ParentTaskID returns AILANG_PARENT_TASK_ID, "" when unset.
func ParentTaskID() string { return get(EnvParentTaskID) }

// OTLPIngestToken returns the trimmed AILANG_OTLP_INGEST_TOKEN, "" when
// ingest auth is off.
func OTLPIngestToken() string { return strings.TrimSpace(get(EnvOTLPIngestToken)) }

// WALCheckpointMB returns AILANG_OBSERVATORY_WAL_CHECKPOINT_MB as a
// positive int64, else the registered default.
func WALCheckpointMB() int64 {
	if v := get(EnvWALCheckpointMB); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return DefaultWALCheckpointMB
}

// SpanFilterDisabled reports AILANG_SPAN_FILTER_DISABLE=true.
func SpanFilterDisabled() bool { return getOr(EnvSpanFilterDisable) == "true" }

// SpanFilterAllow returns AILANG_SPAN_FILTER_ALLOW verbatim.
func SpanFilterAllow() string { return get(EnvSpanFilterAllow) }

// SpanFilterDeny returns AILANG_SPAN_FILTER_DENY verbatim.
func SpanFilterDeny() string { return get(EnvSpanFilterDeny) }
