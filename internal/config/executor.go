package config

import (
	"fmt"
	"sort"
	"strings"
)

// Executor selection and the per-executor knobs (claude, codex, motoko).
const (
	EnvExecutor              = "AILANG_EXECUTOR"
	EnvAuthMode              = "AILANG_AUTH_MODE"
	EnvClaudeConfigDir       = "CLAUDE_CONFIG_DIR"
	EnvMotokoRepo            = "MOTOKO_REPO"
	EnvMotokoSystemRole      = "AILANG_MOTOKO_SYSTEM_ROLE"
	EnvMotokoAgentSystemFile = "AILANG_MOTOKO_AGENT_SYSTEM_FILE"
	// M-EXECUTOR-ENV-HARDENING: the operator's extra inherit grants for agent
	// children, and how a cloud job hands the child git credentials.
	EnvExecutorEnvInherit  = "AILANG_EXECUTOR_ENV_INHERIT"
	EnvChildGitCredentials = "AILANG_CHILD_GIT_CREDENTIALS"
)

// Child git credential modes (AILANG_CHILD_GIT_CREDENTIALS).
const (
	ChildGitCredentialsRepo = "repo"
	ChildGitCredentialsNone = "none"
)

var executorVars = []Var{
	{EnvExecutor, "", AreaExecutor, "Executor name that overrides the config file's default_executor."},
	{EnvAuthMode, "", AreaExecutor, "apikey makes the claude executor use ANTHROPIC_API_KEY (billed); anything else writes the OAuth credentials file from CLAUDE_CODE_OAUTH_TOKEN (subscription)."},
	{EnvClaudeConfigDir, "", AreaExecutor, "Claude Code's config dir; when set the executor also writes credentials there, and the cloud job reads session JSONL from it."},
	{EnvMotokoRepo, "", AreaExecutor, "motoko_agent checkout whose .motoko/logfile holds session JSONL; MOTOKO.md says which one evals use."},
	{EnvMotokoSystemRole, "1", AreaExecutor, "0 stops motoko receiving the teaching prompt in the system role (the default sends it there; reverting to gated is a known regression)."},
	{EnvMotokoAgentSystemFile, "", AreaExecutor, "File whose content becomes motoko's system-role prompt for an A/B, keeping the teaching in the user message."},
	{EnvExecutorEnvInherit, "", AreaExecutor, "Comma-separated variable names an agent child may inherit beyond its default-deny profile (M-EXECUTOR-ENV-HARDENING), credentials included; an operator grant, never read from a task. The rollback lever when a lane needs a variable the profile drops."},
	{EnvChildGitCredentials, ChildGitCredentialsRepo, AreaExecutor, "How a cloud job gives the agent child git credentials: repo = a per-task 0600 credential file outside the workspace, scoped by URL to the task's repository (the parent still pushes); none = nothing (the child cannot authenticate git at all). The fleet token never enters the child's environment either way."},
}

// Executor returns AILANG_EXECUTOR, "" when unset.
func Executor() string { return get(EnvExecutor) }

// AuthMode returns AILANG_AUTH_MODE, "" when unset.
func AuthMode() string { return get(EnvAuthMode) }

// ClaudeConfigDir returns CLAUDE_CONFIG_DIR, "" when unset.
func ClaudeConfigDir() string { return get(EnvClaudeConfigDir) }

// MotokoRepo returns MOTOKO_REPO, "" when unset.
func MotokoRepo() string { return get(EnvMotokoRepo) }

// MotokoSystemRole reports whether the teaching goes in motoko's system
// role: anything but AILANG_MOTOKO_SYSTEM_ROLE=0.
func MotokoSystemRole() bool { return getOr(EnvMotokoSystemRole) != "0" }

// MotokoAgentSystemFile returns AILANG_MOTOKO_AGENT_SYSTEM_FILE, "" when unset.
func MotokoAgentSystemFile() string { return get(EnvMotokoAgentSystemFile) }

// ExecutorEnvInherit returns the AILANG_EXECUTOR_ENV_INHERIT names, trimmed
// and sorted; nil when unset.
func ExecutorEnvInherit() []string {
	var out []string
	for _, n := range strings.Split(get(EnvExecutorEnvInherit), ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// ChildGitCredentials returns the AILANG_CHILD_GIT_CREDENTIALS mode, or an
// error naming the variable for a value that is neither mode (no silent
// fallback to either).
func ChildGitCredentials() (string, error) {
	switch v := strings.TrimSpace(getOr(EnvChildGitCredentials)); v {
	case ChildGitCredentialsRepo, ChildGitCredentialsNone:
		return v, nil
	default:
		return "", fmt.Errorf("%s=%q: want %q or %q", EnvChildGitCredentials, v, ChildGitCredentialsRepo, ChildGitCredentialsNone)
	}
}
