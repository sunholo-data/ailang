package config

import "strings"

// Variables the cloud dispatcher sets on a Cloud Run Job (`ailang coordinator
// execute-job`). They describe ONE task; nothing outside the job reads them.
const (
	EnvAgentID            = "AILANG_AGENT_ID"
	EnvMaxCostUSD         = "AILANG_MAX_COST_USD"
	EnvCascadeRootPackage = "AILANG_CASCADE_ROOT_PACKAGE"
	EnvCascadeChangeClass = "AILANG_CASCADE_CHANGE_CLASS"
	EnvCascadeToVersion   = "AILANG_CASCADE_TO_VERSION"
	EnvDirective          = "AILANG_DIRECTIVE"
	EnvAutoMerge          = "AILANG_AUTO_MERGE"
	EnvArtifactPatterns   = "AILANG_ARTIFACT_PATTERNS"
	EnvTaskTitle          = "AILANG_TASK_TITLE"
	EnvImageProvider      = "AILANG_IMAGE_PROVIDER"
	EnvProvider           = "AILANG_PROVIDER"
	EnvBranch             = "AILANG_BRANCH"
	EnvPushBranch         = "AILANG_PUSH_BRANCH"
	EnvPluginRepo         = "AILANG_PLUGIN_REPO"
	EnvModel              = "AILANG_MODEL"
	EnvTimeout            = "AILANG_TIMEOUT"
	EnvAcknowledgeOnly    = "AILANG_ACKNOWLEDGE_ONLY"
	EnvSubdirectory       = "AILANG_SUBDIRECTORY"
	EnvGitMode            = "AILANG_GIT_MODE"
	EnvSiteSlug           = "AILANG_SITE_SLUG"
	EnvBriefID            = "AILANG_BRIEF_ID"
	EnvGitAuthorName      = "AILANG_GIT_AUTHOR_NAME"
	EnvGitAuthorEmail     = "AILANG_GIT_AUTHOR_EMAIL"
	EnvSSHKeySecret       = "AILANG_SSH_KEY_SECRET"
	EnvSSHHostAlias       = "AILANG_SSH_HOST_ALIAS"
)

// DefaultJobBranch is the branch a job clones when AILANG_BRANCH is unset.
const DefaultJobBranch = "dev"

var jobVars = []Var{
	{EnvAgentID, "", AreaJob, "Agent the job runs as; recorded on spans and completions."},
	{EnvMaxCostUSD, "", AreaJob, "Per-task cost budget in USD; unset or malformed means no cap (malformed is logged and ignored)."},
	{EnvCascadeRootPackage, "", AreaJob, "Root package of a package cascade; when set the job tries the deterministic bump first and the PR is labelled and titled as a cascade."},
	{EnvCascadeChangeClass, "", AreaJob, "Change class of the cascade (A content-only, B additive, ...); decides whether the deterministic path applies."},
	{EnvCascadeToVersion, "", AreaJob, "Version the cascade bumps the dependency to."},
	{EnvDirective, "", AreaJob, "The task directive text handed to the executor and used to derive the PR title and body; unset derives one from the task and agent ids."},
	{EnvAutoMerge, "0", AreaJob, "1 lets the job enable GitHub auto-merge on a docs-only PR that matches the artifact patterns."},
	{EnvArtifactPatterns, "", AreaJob, "Newline-separated path patterns the dispatcher declared as the task's artifacts; the auto-merge scope guard."},
	{EnvTaskTitle, "", AreaJob, "Human-written task title used as the message subject; unset derives one from the directive."},
	{EnvImageProvider, "", AreaJob, "Which provider image the job believes it runs in; verified against AILANG_PROVIDER and printed by preflight diagnostics."},
	{EnvProvider, "", AreaJob, "Provider the dispatcher requested for the task; deliberately not defaulted, the job resolves and verifies it against the image."},
	{EnvBranch, DefaultJobBranch, AreaJob, "Branch the job clones and branches from."},
	{EnvPushBranch, "", AreaJob, "Branch the job commits to and pushes directly (skip_approval agents); set, it also replaces the clone branch."},
	{EnvPluginRepo, "", AreaJob, "Repository of shared skills cloned into the job's plugin directory."},
	{EnvModel, "", AreaJob, "Model the executor runs; there is no default (an empty value fails at the point of use), and it names the commit co-author."},
	{EnvTimeout, "", AreaJob, "Executor wall-clock as a Go duration; unset means the coordinator's default task timeout (2h)."},
	{EnvAcknowledgeOnly, "false", AreaJob, "Exactly true declares the task acknowledge-only (no file changes expected); anything else means changes were expected, so an older dispatcher fails loud rather than lenient."},
	{EnvSubdirectory, "", AreaJob, "Monorepo subdirectory the executor is scoped to, relative to the clone."},
	{EnvGitMode, "", AreaJob, "Git mode the executor's children run under; unset, the job exports guardrails before starting the executor."},
	{EnvSiteSlug, "", AreaJob, "Site slug that turns the job's commit into a structured Build: <slug> message."},
	{EnvBriefID, "", AreaJob, "Brief id appended to the structured commit subject when AILANG_SITE_SLUG is set."},
	{EnvGitAuthorName, "", AreaJob, "git user.name for the job's commits; applied only together with AILANG_GIT_AUTHOR_EMAIL, since a half-configured identity is worse than the container default."},
	{EnvGitAuthorEmail, "", AreaJob, "git user.email for the job's commits; see AILANG_GIT_AUTHOR_NAME."},
	{EnvSSHKeySecret, "", AreaJob, "Secret Manager secret NAME holding a deploy key (never the key itself); set means the job installs it."},
	{EnvSSHHostAlias, "agent-repo", AreaJob, "SSH host alias the deploy key is installed under."},
}

// AgentID returns AILANG_AGENT_ID, "" when unset.
func AgentID() string { return get(EnvAgentID) }

// CascadeChangeClass returns AILANG_CASCADE_CHANGE_CLASS, "" when unset.
func CascadeChangeClass() string { return get(EnvCascadeChangeClass) }

// CascadeToVersion returns AILANG_CASCADE_TO_VERSION, "" when unset.
func CascadeToVersion() string { return get(EnvCascadeToVersion) }

// Provider returns AILANG_PROVIDER, "" when unset.
func Provider() string { return get(EnvProvider) }

// Branch returns AILANG_BRANCH, default dev.
func Branch() string { return getOr(EnvBranch) }

// PushBranch returns AILANG_PUSH_BRANCH, "" when unset.
func PushBranch() string { return get(EnvPushBranch) }

// PluginRepo returns AILANG_PLUGIN_REPO, "" when unset.
func PluginRepo() string { return get(EnvPluginRepo) }

// Model returns AILANG_MODEL, "" when unset.
func Model() string { return get(EnvModel) }

// Timeout returns AILANG_TIMEOUT verbatim, "" when unset; the job applies
// the coordinator's default and parses it.
func Timeout() string { return get(EnvTimeout) }

// AcknowledgeOnly reports AILANG_ACKNOWLEDGE_ONLY=true exactly.
func AcknowledgeOnly() bool { return get(EnvAcknowledgeOnly) == "true" }

// Subdirectory returns AILANG_SUBDIRECTORY, "" when unset.
func Subdirectory() string { return get(EnvSubdirectory) }

// GitModeSet reports whether AILANG_GIT_MODE is set to anything.
func GitModeSet() bool { return get(EnvGitMode) != "" }

// SiteSlug returns AILANG_SITE_SLUG, "" when unset.
func SiteSlug() string { return get(EnvSiteSlug) }

// BriefID returns AILANG_BRIEF_ID, "" when unset.
func BriefID() string { return get(EnvBriefID) }

// GitAuthor returns the trimmed AILANG_GIT_AUTHOR_NAME and
// AILANG_GIT_AUTHOR_EMAIL, each "" when unset.
func GitAuthor() (name, email string) {
	return strings.TrimSpace(get(EnvGitAuthorName)), strings.TrimSpace(get(EnvGitAuthorEmail))
}

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
