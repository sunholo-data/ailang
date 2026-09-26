package config

import "strings"

// The coordinator daemon and the commands that administer it.
const (
	EnvCoordinatorAPIKey       = "COORDINATOR_API_KEY"
	EnvCoordinatorBindAddr     = "COORDINATOR_BIND_ADDR"
	EnvPort                    = "PORT"
	EnvCoordHTTPPort           = "AILANG_COORD_HTTP_PORT"
	EnvGitHubWebhookSecret     = "GITHUB_WEBHOOK_SECRET"
	EnvRepoURL                 = "AILANG_REPO_URL"
	EnvKMSKey                  = "AILANG_KMS_KEY"
	EnvBackstopSweep           = "AILANG_BACKSTOP_SWEEP"
	EnvFeedbackGateMode        = "AILANG_FEEDBACK_GATE_MODE"
	EnvFeedbackGateDryRun      = "AILANG_FEEDBACK_GATE_DRY_RUN"
	EnvFeedbackGateShadow      = "AILANG_FEEDBACK_GATE_SHADOW"
	EnvResidentProject         = "RESIDENT_LIFECYCLE_PROJECT"
	EnvResidentRegion          = "RESIDENT_LIFECYCLE_REGION"
	EnvResidentAudience        = "RESIDENT_LIFECYCLE_AUDIENCE"
	EnvResidentAllowedCallers  = "RESIDENT_LIFECYCLE_ALLOWED_CALLERS"
	EnvCoordinatorService      = "AILANG_COORDINATOR_SERVICE"
	EnvConfigBucket            = "AILANG_CONFIG_BUCKET"
	EnvConfigObject            = "AILANG_CONFIG_OBJECT"
	EnvAgentCheckRepoConfig    = "AILANG_AGENT_CHECK_REPO_CONFIG"
	EnvApprovalPolicy          = "AILANG_APPROVAL_POLICY"
	EnvApprovalAuthorityModels = "AILANG_APPROVAL_AUTHORITY_MODELS"
	EnvApprovalController      = "AILANG_APPROVAL_CONTROLLER"
	EnvApprovalBaseURL         = "AILANG_APPROVAL_BASE_URL"
	EnvTokenSecret             = "AILANG_TOKEN_SECRET"
	EnvMessagesProject         = "AILANG_MESSAGES_PROJECT"
	EnvWorkspace               = "AILANG_WORKSPACE"
	EnvDefaultProvider         = "AILANG_DEFAULT_PROVIDER"
	EnvBudgetUnlimited         = "AILANG_BUDGET_UNLIMITED"
	EnvApprovalTimeout         = "AILANG_APPROVAL_TIMEOUT"
	EnvApprovalURL             = "AILANG_APPROVAL_URL"
	EnvCoordinatorURL          = "AILANG_COORDINATOR_URL"
	EnvApprovalToken           = "AILANG_APPROVAL_TOKEN"
)

var coordinatorVars = []Var{
	{EnvCoordinatorAPIKey, "", AreaCoordinator, "Shared secret for the coordinator's HTTP API and the dashboard's WebSocket; the daemon rejects every request while it is unset (fail-closed since S3 M5) and `ailang coordinator` commands discover it from here first."},
	{EnvCoordinatorBindAddr, "", AreaCoordinator, "Host the daemon's HTTP server binds; unset is 127.0.0.1 locally and 0.0.0.0 in cloud mode."},
	{EnvPort, "", AreaCoordinator, "Cloud Run's port convention: when set, the daemon starts its HTTP server on it, `ailang server` and `ailang serve-api` bind 0.0.0.0 instead of 127.0.0.1 (DefaultBindHost; --bind overrides), and the registry validator listens on it (default 8080 there)."},
	{EnvCoordHTTPPort, "", AreaCoordinator, "Port the coordinator's HTTP API is on, for commands that must reach a running daemon; falls back to PORT."},
	{EnvGitHubWebhookSecret, "", AreaCoordinator, "HMAC secret for the /github/webhook route; unset means the route is not served."},
	{EnvRepoURL, "", AreaCoordinator, "Repository URL for task worktrees when the task's workspace does not name one."},
	{EnvKMSKey, "", AreaCoordinator, "Cloud KMS key resource that encrypts stored secrets and decrypts an ENC:-prefixed ANTHROPIC_API_KEY; unset means plaintext passthrough."},
	{EnvBackstopSweep, "report", AreaCoordinator, "Backstop sweep mode: dispatch runs stranded work, off disables the sweep, report (and any other value) only reports."},
	{EnvFeedbackGateMode, "", AreaCoordinator, "Overrides the feedback gate's configured mode (operator kill-switch)."},
	{EnvFeedbackGateDryRun, "", AreaCoordinator, "1, true, yes or on forces the feedback gate into dry-run."},
	{EnvFeedbackGateShadow, "", AreaCoordinator, "off | openrouter | direct: runs a System One decision model (sunholo/decisions, TypeSafe Jev) beside the feedback-gate classifier and records both verdicts in the audit row; never changes the action. Enabling it sends the submission body to TypeSafe — the operator's data-boundary ruling. Overrides coordinator.feedback_gate.shadow."},
	{EnvResidentProject, "", AreaCoordinator, "Project of the resident Cloud Run instances the lifecycle routes start and stop."},
	{EnvResidentRegion, "", AreaCoordinator, "Region of the resident instances."},
	{EnvResidentAudience, "", AreaCoordinator, "ID-token audience the lifecycle routes verify; unset means nobody may call them."},
	{EnvResidentAllowedCallers, "", AreaCoordinator, "Comma-separated principals allowed to call the lifecycle routes; unset means nobody."},
	{EnvCoordinatorService, "ailang-coordinator", AreaCoordinator, "Cloud Run service name `coordinator config roll` restarts."},
	{EnvConfigBucket, "", AreaCoordinator, "GCS bucket holding the fleet's config.yaml; unset derives <project>-ailang-config from CloudProject."},
	{EnvConfigObject, "config.yaml", AreaCoordinator, "Object name of the fleet config inside AILANG_CONFIG_BUCKET."},
	{EnvAgentCheckRepoConfig, "", AreaCoordinator, "Path of the repo config `coordinator agent check` and `agent set` validate against when --repo-config is not given."},
	{EnvApprovalPolicy, "", AreaCoordinator, "Approval policy override (evaluated, manual, ...); unset reads the coordinator config."},
	{EnvApprovalAuthorityModels, "fable,astra,opus", AreaCoordinator, "Comma-separated models allowed to rule on approvals."},
	{EnvApprovalController, "0", AreaCoordinator, "1 grants an attended session approval authority."},
	{EnvApprovalBaseURL, "", AreaCoordinator, "Public base URL for the secret-approval action links pushed to ntfy; unset skips the push."},
	{EnvTokenSecret, "", AreaCoordinator, "HMAC secret for approval tokens; unset generates one per process."},
	{EnvMessagesProject, "", AreaCoordinator, "Pins the messaging store's Firestore project without moving anything else to the cloud project."},
	{EnvWorkspace, "", AreaCoordinator, "Workspace a cloud process partitions its data under: the daemon's broadcast events and the execute-job's completion; unset serves default as a deprecated default (D3), refused under AILANG_STRICT_CONFIG=1."},
	{EnvDefaultProvider, "", AreaCoordinator, "Provider a task is attributed to for budgeting when neither the task nor its agent names one; unset serves claude as a deprecated default (D3)."},
	{EnvBudgetUnlimited, "0", AreaCoordinator, "1 acknowledges that a task may run with NO spend cap when no budget resolves; unset serves that as a deprecated default (D3), refused under AILANG_STRICT_CONFIG=1."},
	{EnvApprovalTimeout, "", AreaCoordinator, "How long an approval request waits for a human, as a positive Go duration (e.g. 24h); unset serves the built-in wait as a deprecated default (D3)."},
	{EnvApprovalURL, "", AreaCoordinator, "Service that serves /api/approvals (the dashboard), which secret() on the shared storage plane POSTs approval requests to; falls back to AILANG_COORDINATOR_URL, and unset leaves secret() un-gated as a deprecated default (D3)."},
	{EnvCoordinatorURL, "", AreaCoordinator, "Compatibility fallback for AILANG_APPROVAL_URL."},
	{EnvApprovalToken, "", AreaCoordinator, "Bearer token the cloud secret approver authenticates its requests with."},
}

// CoordinatorAPIKey returns COORDINATOR_API_KEY, "" when unset.
func CoordinatorAPIKey() string { return get(EnvCoordinatorAPIKey) }

// CoordinatorBindAddr returns COORDINATOR_BIND_ADDR, "" when unset.
func CoordinatorBindAddr() string { return get(EnvCoordinatorBindAddr) }

// Port returns PORT, "" when unset.
func Port() string { return get(EnvPort) }

// DefaultBindHost is the host an AILANG HTTP listener binds when no --bind
// is given: 0.0.0.0 when PORT is set (Cloud Run injects PORT and requires the
// wildcard), otherwise 127.0.0.1. One rule for `ailang server` and
// `ailang serve-api` (M-SERVEAPI-BIND-HOST-CORS F2).
func DefaultBindHost() string {
	if Port() != "" {
		return "0.0.0.0"
	}
	return "127.0.0.1"
}

// CoordHTTPPort returns AILANG_COORD_HTTP_PORT, "" when unset.
func CoordHTTPPort() string { return get(EnvCoordHTTPPort) }

// GitHubWebhookSecret returns GITHUB_WEBHOOK_SECRET, "" when unset.
func GitHubWebhookSecret() string { return get(EnvGitHubWebhookSecret) }

// RepoURL returns AILANG_REPO_URL, "" when unset.
func RepoURL() string { return get(EnvRepoURL) }

// KMSKey returns AILANG_KMS_KEY, "" when unset.
func KMSKey() string { return get(EnvKMSKey) }

// BackstopSweep returns AILANG_BACKSTOP_SWEEP lower-cased and trimmed, ""
// when unset; the coordinator maps it to a mode.
func BackstopSweep() string { return strings.ToLower(strings.TrimSpace(get(EnvBackstopSweep))) }

// FeedbackGateMode returns the trimmed AILANG_FEEDBACK_GATE_MODE, "" when unset.
func FeedbackGateMode() string { return strings.TrimSpace(get(EnvFeedbackGateMode)) }

// FeedbackGateDryRun returns AILANG_FEEDBACK_GATE_DRY_RUN verbatim; the
// coordinator applies its truthiness rule.
func FeedbackGateDryRun() string { return get(EnvFeedbackGateDryRun) }

// FeedbackGateShadow returns AILANG_FEEDBACK_GATE_SHADOW lower-cased and
// trimmed, "" when unset.
func FeedbackGateShadow() string {
	return strings.ToLower(strings.TrimSpace(get(EnvFeedbackGateShadow)))
}

// ResidentLifecycle is the resident-instance lifecycle configuration.
type ResidentLifecycle struct {
	Project, Region, Audience, AllowedCallers string
}

// ResidentLifecycleConfig returns the four RESIDENT_LIFECYCLE_* values verbatim.
func ResidentLifecycleConfig() ResidentLifecycle {
	return ResidentLifecycle{
		Project:        get(EnvResidentProject),
		Region:         get(EnvResidentRegion),
		Audience:       get(EnvResidentAudience),
		AllowedCallers: get(EnvResidentAllowedCallers),
	}
}

// CoordinatorService returns AILANG_COORDINATOR_SERVICE, default ailang-coordinator.
func CoordinatorService() string { return getOr(EnvCoordinatorService) }

// ConfigBucket returns AILANG_CONFIG_BUCKET, "" when unset.
func ConfigBucket() string { return get(EnvConfigBucket) }

// ConfigObject returns AILANG_CONFIG_OBJECT, default config.yaml.
func ConfigObject() string { return getOr(EnvConfigObject) }

// AgentCheckRepoConfig returns AILANG_AGENT_CHECK_REPO_CONFIG, "" when unset.
func AgentCheckRepoConfig() string { return get(EnvAgentCheckRepoConfig) }

// ApprovalPolicy returns AILANG_APPROVAL_POLICY lower-cased and trimmed.
func ApprovalPolicy() string { return strings.ToLower(strings.TrimSpace(get(EnvApprovalPolicy))) }

// ApprovalAuthorityModels returns the trimmed AILANG_APPROVAL_AUTHORITY_MODELS,
// "" when unset (the caller applies the default list).
func ApprovalAuthorityModels() string { return strings.TrimSpace(get(EnvApprovalAuthorityModels)) }

// ApprovalController reports AILANG_APPROVAL_CONTROLLER=1.
func ApprovalController() bool { return getOr(EnvApprovalController) == "1" }

// ApprovalBaseURL returns AILANG_APPROVAL_BASE_URL, "" when unset.
func ApprovalBaseURL() string { return get(EnvApprovalBaseURL) }

// TokenSecret returns AILANG_TOKEN_SECRET, "" when unset.
func TokenSecret() string { return get(EnvTokenSecret) }

// MessagesProject returns the trimmed AILANG_MESSAGES_PROJECT, "" when unset.
func MessagesProject() string { return strings.TrimSpace(get(EnvMessagesProject)) }

// Workspace returns AILANG_WORKSPACE, "" when unset; the caller serves the
// deprecated default through DeprecatedDefault.
func Workspace() string { return get(EnvWorkspace) }

// DefaultProvider returns the trimmed AILANG_DEFAULT_PROVIDER, "" when unset.
func DefaultProvider() string { return strings.TrimSpace(get(EnvDefaultProvider)) }

// BudgetUnlimited reports AILANG_BUDGET_UNLIMITED=1.
func BudgetUnlimited() bool { return getOr(EnvBudgetUnlimited) == "1" }

// ApprovalTimeout returns the trimmed AILANG_APPROVAL_TIMEOUT verbatim, ""
// when unset; the coordinator parses it so a malformed wait is an error.
func ApprovalTimeout() string { return strings.TrimSpace(get(EnvApprovalTimeout)) }

// ApprovalURL returns AILANG_APPROVAL_URL, else AILANG_COORDINATOR_URL, ""
// when neither is set.
func ApprovalURL() string {
	if v := get(EnvApprovalURL); v != "" {
		return v
	}
	return get(EnvCoordinatorURL)
}

// ApprovalToken returns AILANG_APPROVAL_TOKEN, "" when unset.
func ApprovalToken() string { return get(EnvApprovalToken) }
