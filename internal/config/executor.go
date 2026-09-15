package config

// Executor selection and the per-executor knobs (claude, codex, motoko).
const (
	EnvExecutor              = "AILANG_EXECUTOR"
	EnvAuthMode              = "AILANG_AUTH_MODE"
	EnvClaudeConfigDir       = "CLAUDE_CONFIG_DIR"
	EnvMotokoRepo            = "MOTOKO_REPO"
	EnvMotokoSystemRole      = "AILANG_MOTOKO_SYSTEM_ROLE"
	EnvMotokoAgentSystemFile = "AILANG_MOTOKO_AGENT_SYSTEM_FILE"
)

var executorVars = []Var{
	{EnvExecutor, "", AreaExecutor, "Executor name that overrides the config file's default_executor."},
	{EnvAuthMode, "", AreaExecutor, "apikey makes the claude executor use ANTHROPIC_API_KEY (billed); anything else writes the OAuth credentials file from CLAUDE_CODE_OAUTH_TOKEN (subscription)."},
	{EnvClaudeConfigDir, "", AreaExecutor, "Claude Code's config dir; when set the executor also writes credentials there, and the cloud job reads session JSONL from it."},
	{EnvMotokoRepo, "", AreaExecutor, "motoko_agent checkout whose .motoko/logfile holds session JSONL; MOTOKO.md says which one evals use."},
	{EnvMotokoSystemRole, "1", AreaExecutor, "0 stops motoko receiving the teaching prompt in the system role (the default sends it there; reverting to gated is a known regression)."},
	{EnvMotokoAgentSystemFile, "", AreaExecutor, "File whose content becomes motoko's system-role prompt for an A/B, keeping the teaching in the user message."},
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
