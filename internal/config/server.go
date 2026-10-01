package config

import "strings"

// `ailang server` (the dashboard / collaboration hub) and the commands that
// talk to it.
const (
	EnvFirebaseProject    = "AILANG_FIREBASE_PROJECT"
	EnvHubToken           = "AILANG_HUB_TOKEN"
	EnvApprovalSigningKey = "AILANG_APPROVAL_SIGNING_KEY"
	EnvBenchmarksBucket   = "BENCHMARKS_BUCKET"
	EnvDashboardURL       = "AILANG_DASHBOARD_URL"

	EnvApprovalAllowedCallers = "AILANG_APPROVAL_ALLOWED_CALLERS"
	EnvApprovalAudience       = "AILANG_APPROVAL_AUDIENCE"
	EnvApprovalIntakeAuth     = "AILANG_APPROVAL_INTAKE_AUTH"
)

var serverVars = []Var{
	{EnvFirebaseProject, "", AreaServer, "Firebase / Firestore project for the dashboard's auth, workspaces and access control when --firebase-project is not given."},
	{EnvHubToken, "", AreaServer, "Bearer token the hub's /api/hooks/* routes require; unset leaves them open (local use)."},
	{EnvApprovalSigningKey, "", AreaServer, "HMAC key that signs the secret-approval action links, so ntfy buttons can POST without IAM; unset disables those endpoints."},
	{EnvBenchmarksBucket, "ailang-multivac-dev-benchmarks", AreaServer, "GCS bucket the benchmarks API reads through."},
	{EnvDashboardURL, "", AreaServer, "Dashboard base URL for commands that print or open links, after the --dashboard flag."},
	{EnvApprovalAllowedCallers, "", AreaServer, "Comma-separated service-account emails whose Google-signed ID tokens may create and poll secret approvals (POST /api/approvals, GET /api/approvals/{id}); unset admits no ID token."},
	{EnvApprovalAudience, "", AreaServer, "Comma-separated audiences an approval caller's ID token may carry; unset uses AILANG_APPROVAL_BASE_URL."},
	{EnvApprovalIntakeAuth, "enforce", AreaServer, "enforce (default) requires an ID token or AILANG_APPROVAL_TOKEN on secret-approval create/poll; off admits anonymous callers (rollback lever, logged loudly). Any other value enforces."},
}

// FirebaseProject returns AILANG_FIREBASE_PROJECT, "" when unset.
func FirebaseProject() string { return get(EnvFirebaseProject) }

// HubToken returns AILANG_HUB_TOKEN, "" when unset.
func HubToken() string { return get(EnvHubToken) }

// ApprovalSigningKey returns AILANG_APPROVAL_SIGNING_KEY, "" when unset.
func ApprovalSigningKey() string { return get(EnvApprovalSigningKey) }

// BenchmarksBucket returns BENCHMARKS_BUCKET, default the dev bucket.
func BenchmarksBucket() string { return getOr(EnvBenchmarksBucket) }

// DashboardURL returns AILANG_DASHBOARD_URL, "" when unset.
func DashboardURL() string { return get(EnvDashboardURL) }

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
