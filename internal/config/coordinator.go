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
)

var coordinatorVars = []Var{
	{EnvCoordinatorAPIKey, "", AreaCoordinator, "Shared secret for the coordinator's HTTP API and the dashboard's WebSocket; the daemon rejects every request while it is unset (fail-closed since S3 M5) and `ailang coordinator` commands discover it from here first."},
	{EnvCoordinatorBindAddr, "", AreaCoordinator, "Host the daemon's HTTP server binds; unset is 127.0.0.1 locally and 0.0.0.0 in cloud mode."},
	{EnvPort, "", AreaCoordinator, "Cloud Run's port convention: when set, the daemon starts its HTTP server on it, `ailang server` binds it on 0.0.0.0, and the registry validator listens on it (default 8080 there)."},
	{EnvCoordHTTPPort, "", AreaCoordinator, "Port the coordinator's HTTP API is on, for commands that must reach a running daemon; falls back to PORT."},
	{EnvGitHubWebhookSecret, "", AreaCoordinator, "HMAC secret for the /github/webhook route; unset means the route is not served."},
	{EnvRepoURL, "", AreaCoordinator, "Repository URL for task worktrees when the task's workspace does not name one."},
	{EnvKMSKey, "", AreaCoordinator, "Cloud KMS key resource that encrypts stored secrets and decrypts an ENC:-prefixed ANTHROPIC_API_KEY; unset means plaintext passthrough."},
	{EnvBackstopSweep, "report", AreaCoordinator, "Backstop sweep mode: dispatch runs stranded work, off disables the sweep, report (and any other value) only reports."},
	{EnvFeedbackGateMode, "", AreaCoordinator, "Overrides the feedback gate's configured mode (operator kill-switch)."},
	{EnvFeedbackGateDryRun, "", AreaCoordinator, "1, true, yes or on forces the feedback gate into dry-run."},
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
}

// CoordinatorAPIKey returns COORDINATOR_API_KEY, "" when unset.
func CoordinatorAPIKey() string { return get(EnvCoordinatorAPIKey) }

// CoordinatorBindAddr returns COORDINATOR_BIND_ADDR, "" when unset.
func CoordinatorBindAddr() string { return get(EnvCoordinatorBindAddr) }

// Port returns PORT, "" when unset.
func Port() string { return get(EnvPort) }

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
