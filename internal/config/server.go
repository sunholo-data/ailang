package config

import "strings"

// `ailang server` (the dashboard / collaboration hub) and the commands that
// talk to it.
const (
	EnvFirebaseProject    = "AILANG_FIREBASE_PROJECT"
	EnvHubToken           = "AILANG_HUB_TOKEN"
	EnvIngestAuth         = "AILANG_INGEST_AUTH"
	EnvApprovalSigningKey = "AILANG_APPROVAL_SIGNING_KEY"
	EnvBenchmarksBucket   = "BENCHMARKS_BUCKET"
	EnvDashboardURL       = "AILANG_DASHBOARD_URL"
	EnvDashboardToken     = "AILANG_DASHBOARD_TOKEN"

	EnvApprovalAllowedCallers = "AILANG_APPROVAL_ALLOWED_CALLERS"
	EnvApprovalAudience       = "AILANG_APPROVAL_AUDIENCE"
	EnvApprovalIntakeAuth     = "AILANG_APPROVAL_INTAKE_AUTH"
)

var serverVars = []Var{
	{EnvFirebaseProject, "", AreaServer, "Firebase / Firestore project for the dashboard's auth, workspaces and access control when --firebase-project is not given."},
	{EnvHubToken, "", AreaServer, "Shared secret for the dashboard's hook/exec/telemetry ingest routes (/api/hooks/claude, /api/observatory/hooks, /api/exec/sessions, /api/exec/events, /api/coordinator/events, /api/telemetry), sent as Authorization: Bearer or X-AILANG-Ingest-Token; checked on non-loopback binds per AILANG_INGEST_AUTH."},
	{EnvIngestAuth, "warn", AreaServer, "How a dashboard on a non-loopback bind treats ingest writes (OTLP /v1/* and the hook/exec/telemetry routes) that lack the matching token: warn (default) accepts and logs them, enforce refuses them (401; 503 when the token is unset), off accepts them (rollback lever, logged at startup). Any other value enforces. Loopback binds are always exempt."},
	{EnvApprovalSigningKey, "", AreaServer, "HMAC key that signs the secret-approval action links, so ntfy buttons can POST without IAM; unset disables those endpoints."},
	{EnvBenchmarksBucket, "ailang-multivac-dev-benchmarks", AreaServer, "GCS bucket the benchmarks API reads through."},
	{EnvDashboardURL, "", AreaServer, "Dashboard base URL for commands that print or open links, after the --dashboard flag."},
	{EnvDashboardToken, "", AreaServer, "Bearer token CLI readers (`ailang chains dashboard`, `eval-paired`'s span fetch, eval observatory posts) send to a dashboard: the server's COORDINATOR_API_KEY. A dashboard on a non-loopback bind refuses /api/ reads without it."},
	{EnvApprovalAllowedCallers, "", AreaServer, "Comma-separated service-account emails whose Google-signed ID tokens may create and poll secret approvals (`POST /api/approvals`, `GET /api/approvals/<id>`); unset admits no ID token."},
	{EnvApprovalAudience, "", AreaServer, "Comma-separated audiences an approval caller's ID token may carry; unset uses AILANG_APPROVAL_BASE_URL."},
	{EnvApprovalIntakeAuth, "enforce", AreaServer, "enforce (default) requires an ID token or AILANG_APPROVAL_TOKEN on secret-approval create/poll; off admits anonymous callers (rollback lever, logged loudly). Any other value enforces."},
}

// FirebaseProject returns AILANG_FIREBASE_PROJECT, "" when unset.
func FirebaseProject() string { return get(EnvFirebaseProject) }

// HubToken returns the trimmed AILANG_HUB_TOKEN, "" when unset.
func HubToken() string { return strings.TrimSpace(get(EnvHubToken)) }

// Ingest auth modes (AILANG_INGEST_AUTH).
const (
	IngestAuthOff     = "off"
	IngestAuthWarn    = "warn"
	IngestAuthEnforce = "enforce"
)

// IngestAuthMode returns AILANG_INGEST_AUTH normalised to off, warn or
// enforce; default warn. Any other value — a typo included — enforces: this
// is a security gate, so an unreadable setting fails closed.
func IngestAuthMode() string {
	switch v := strings.ToLower(strings.TrimSpace(getOr(EnvIngestAuth))); v {
	case IngestAuthOff, IngestAuthWarn:
		return v
	default:
		return IngestAuthEnforce
	}
}

// ApprovalSigningKey returns AILANG_APPROVAL_SIGNING_KEY, "" when unset.
func ApprovalSigningKey() string { return get(EnvApprovalSigningKey) }

// BenchmarksBucket returns BENCHMARKS_BUCKET, default the dev bucket.
func BenchmarksBucket() string { return getOr(EnvBenchmarksBucket) }

// DashboardURL returns AILANG_DASHBOARD_URL, "" when unset.
func DashboardURL() string { return get(EnvDashboardURL) }

// DashboardToken returns AILANG_DASHBOARD_TOKEN, "" when unset.
func DashboardToken() string { return strings.TrimSpace(get(EnvDashboardToken)) }

// ApprovalAllowedCallers returns AILANG_APPROVAL_ALLOWED_CALLERS, "" when unset.
func ApprovalAllowedCallers() string { return get(EnvApprovalAllowedCallers) }

// ApprovalAudience returns AILANG_APPROVAL_AUDIENCE, else
// AILANG_APPROVAL_BASE_URL, "" when neither is set.
func ApprovalAudience() string {
	if v := get(EnvApprovalAudience); v != "" {
		return v
	}
	return get(EnvApprovalBaseURL)
}

// ApprovalIntakeAuthOff reports AILANG_APPROVAL_INTAKE_AUTH=off. Every other
// value, including a typo, enforces: this is a security gate.
func ApprovalIntakeAuthOff() bool {
	return strings.EqualFold(strings.TrimSpace(getOr(EnvApprovalIntakeAuth)), "off")
}
