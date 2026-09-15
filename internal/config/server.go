package config

// `ailang server` (the dashboard / collaboration hub) and the commands that
// talk to it.
const (
	EnvFirebaseProject    = "AILANG_FIREBASE_PROJECT"
	EnvHubToken           = "AILANG_HUB_TOKEN"
	EnvApprovalSigningKey = "AILANG_APPROVAL_SIGNING_KEY"
	EnvBenchmarksBucket   = "BENCHMARKS_BUCKET"
	EnvDashboardURL       = "AILANG_DASHBOARD_URL"
)

var serverVars = []Var{
	{EnvFirebaseProject, "", AreaServer, "Firebase / Firestore project for the dashboard's auth, workspaces and access control when --firebase-project is not given."},
	{EnvHubToken, "", AreaServer, "Bearer token the hub's /api/hooks/* routes require; unset leaves them open (local use)."},
	{EnvApprovalSigningKey, "", AreaServer, "HMAC key that signs the secret-approval action links, so ntfy buttons can POST without IAM; unset disables those endpoints."},
	{EnvBenchmarksBucket, "ailang-multivac-dev-benchmarks", AreaServer, "GCS bucket the benchmarks API reads through."},
	{EnvDashboardURL, "", AreaServer, "Dashboard base URL for commands that print or open links, after the --dashboard flag."},
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
