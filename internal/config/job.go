package config

import "strings"

// Variables the cloud dispatcher sets on a Cloud Run Job (`ailang coordinator
// execute-job`). They describe ONE task; nothing outside the job reads them.
const (
	EnvAgentID            = "AILANG_AGENT_ID"
	EnvMaxCostUSD         = "AILANG_MAX_COST_USD"
	EnvCascadeRootPackage = "AILANG_CASCADE_ROOT_PACKAGE"
	EnvDirective          = "AILANG_DIRECTIVE"
	EnvAutoMerge          = "AILANG_AUTO_MERGE"
	EnvArtifactPatterns   = "AILANG_ARTIFACT_PATTERNS"
	EnvTaskTitle          = "AILANG_TASK_TITLE"
	EnvImageProvider      = "AILANG_IMAGE_PROVIDER"
	EnvSSHKeySecret       = "AILANG_SSH_KEY_SECRET"
	EnvSSHHostAlias       = "AILANG_SSH_HOST_ALIAS"
)

var jobVars = []Var{
	{EnvAgentID, "", AreaJob, "Agent the job runs as; recorded on spans and completions."},
	{EnvMaxCostUSD, "", AreaJob, "Per-task cost budget in USD; unset or malformed means no cap (malformed is logged and ignored)."},
	{EnvCascadeRootPackage, "", AreaJob, "Root package of a package cascade; when set the PR is labelled and titled as a cascade."},
	{EnvDirective, "", AreaJob, "The task directive text, used to derive the PR title and body."},
	{EnvAutoMerge, "0", AreaJob, "1 lets the job enable GitHub auto-merge on a docs-only PR that matches the artifact patterns."},
	{EnvArtifactPatterns, "", AreaJob, "Newline-separated path patterns the dispatcher declared as the task's artifacts; the auto-merge scope guard."},
	{EnvTaskTitle, "", AreaJob, "Human-written task title used as the message subject; unset derives one from the directive."},
	{EnvImageProvider, "", AreaJob, "Which provider image the job believes it runs in; printed by preflight diagnostics."},
	{EnvSSHKeySecret, "", AreaJob, "Secret Manager secret NAME holding a deploy key (never the key itself); set means the job installs it."},
	{EnvSSHHostAlias, "agent-repo", AreaJob, "SSH host alias the deploy key is installed under."},
}

// AgentID returns AILANG_AGENT_ID, "" when unset.
func AgentID() string { return get(EnvAgentID) }

// MaxCostUSD returns AILANG_MAX_COST_USD verbatim, "" when unset; the job
// parses and reports a malformed value itself.
func MaxCostUSD() string { return get(EnvMaxCostUSD) }

// CascadeRootPackage returns AILANG_CASCADE_ROOT_PACKAGE, "" when unset.
func CascadeRootPackage() string { return get(EnvCascadeRootPackage) }

// Directive returns AILANG_DIRECTIVE, "" when unset.
func Directive() string { return get(EnvDirective) }

// AutoMerge reports AILANG_AUTO_MERGE=1.
func AutoMerge() bool { return getOr(EnvAutoMerge) == "1" }

// ArtifactPatterns returns the trimmed AILANG_ARTIFACT_PATTERNS, "" when unset.
func ArtifactPatterns() string { return strings.TrimSpace(get(EnvArtifactPatterns)) }

// TaskTitle returns the trimmed AILANG_TASK_TITLE, "" when unset.
func TaskTitle() string { return strings.TrimSpace(get(EnvTaskTitle)) }

// ImageProvider returns the trimmed AILANG_IMAGE_PROVIDER, "" when unset.
func ImageProvider() string { return strings.TrimSpace(get(EnvImageProvider)) }

// SSHKeySecret returns the trimmed AILANG_SSH_KEY_SECRET, "" when unset.
func SSHKeySecret() string { return strings.TrimSpace(get(EnvSSHKeySecret)) }

// SSHHostAlias returns the trimmed AILANG_SSH_HOST_ALIAS, default agent-repo.
func SSHHostAlias() string {
	if v := strings.TrimSpace(get(EnvSSHHostAlias)); v != "" {
		return v
	}
	return defaultOf(EnvSSHHostAlias)
}
