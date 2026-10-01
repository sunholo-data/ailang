package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/policy"
)

// EnvPolicy is the resolved inheritance rule for one agent child
// (M-EXECUTOR-ENV-HARDENING D1/D2): which host variables the model-facing CLI
// may see. It is default-deny in two ways —
//
//   - a host variable is inherited only when its name is on the inherit
//     allowlist (locale, PATH/HOME/TMPDIR, toolchain dirs, the harness's own
//     AILANG_*/OTEL_*/CLAUDE_*/PI_*/MOTOKO_* configuration, …); and
//   - a credential-shaped name (…_KEY, …_TOKEN, …SECRET…, SSH_AUTH_SOCK,
//     AWS_*, …) is withheld even when its family is allowlisted, unless the
//     policy GRANTS it by name.
//
// Grants are derived from the executor and task (the lane's inference
// credential, the credentials an attached program policy's worker may
// select) plus the operator's AILANG_EXECUTOR_ENV_INHERIT list. Nothing a task
// supplies can add a grant: Task.ExtraEnv sets values, it does not unlock
// inheritance.
//
// Same-UID caveat: the child runs as the same user as the parent, so this is
// defence in depth (no secret in `printenv`, in tool/MCP subprocess
// inheritance, or in a crash dump), not a boundary against a determined
// process — that is the UID split (audit H-6) plus the egress lock.
type EnvPolicy struct {
	Executor string
	// Grants are the credential-shaped names this child may inherit, sorted.
	Grants []string
	// Extra are the operator's AILANG_EXECUTOR_ENV_INHERIT names, sorted.
	Extra []string
}

// Executor names the policy knows. They match each executor's Name().
const (
	envExecClaude   = "claude"
	envExecCodex    = "codex"
	envExecPi       = "pi"
	envExecOpenCode = "opencode"
	envExecMotoko   = "motoko"
)

// inheritExact are inherited by (case-insensitive) exact name.
var inheritExact = toSet(
	// Process basics and locale.
	"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TERM", "COLORTERM", "TERM_PROGRAM",
	"LANG", "LANGUAGE", "TZ", "TMPDIR", "TMP", "TEMP", "PWD", "HOSTNAME", "DISPLAY",
	"EDITOR", "VISUAL", "PAGER", "CI", "NO_COLOR", "FORCE_COLOR", "DEBUG_AGENT",
	"__CF_USER_TEXT_ENCODING",
	// TLS trust and operator-set proxies (the host's own network setup; a task
	// cannot set these — see ValidateExtraEnv).
	"SSL_CERT_FILE", "SSL_CERT_DIR", "NODE_EXTRA_CA_CERTS", "REQUESTS_CA_BUNDLE", "CURL_CA_BUNDLE",
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY",
	// Toolchains the agents build and test with.
	"GOPATH", "GOROOT", "GOCACHE", "GOMODCACHE", "GOFLAGS", "GOPROXY", "GOPRIVATE",
	"GONOSUMDB", "GONOPROXY", "GOSUMDB", "GOTOOLCHAIN", "GOBIN", "GOOS", "GOARCH", "CGO_ENABLED",
	"GOGC", "GOMEMLIMIT",
	"JAVA_HOME", "CARGO_HOME", "RUSTUP_HOME", "PNPM_HOME", "BUN_INSTALL", "VIRTUAL_ENV",
	"PYENV_ROOT", "SDKROOT", "DEVELOPER_DIR",
	// Commit identity (not transport: GIT_SSH_COMMAND/GIT_ASKPASS/GIT_CONFIG_* stay out).
	"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
	"GIT_TERMINAL_PROMPT",
	// GCP identity hints (no credentials: Cloud Run uses the metadata server).
	"GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_LOCATION", "GOOGLE_GENAI_USE_VERTEXAI",
	"OTLP_GOOGLE_CLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT", "GOOGLE_CLOUD_REGION",
	"OBSERVATORY_ENDPOINT", "CONTROLLER_ID",
	"CLOUD_RUN_JOB", "CLOUD_RUN_EXECUTION", "CLOUD_RUN_TASK_INDEX", "CLOUD_RUN_TASK_COUNT",
	"CLOUD_RUN_TASK_ATTEMPT", "K_SERVICE", "K_REVISION", "K_CONFIGURATION",
	// Windows process basics.
	"SYSTEMROOT", "SYSTEMDRIVE", "COMSPEC", "PATHEXT", "WINDIR", "APPDATA", "LOCALAPPDATA",
	"USERPROFILE", "PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)", "NUMBER_OF_PROCESSORS",
	"PROCESSOR_ARCHITECTURE", "HOMEDRIVE", "HOMEPATH", "USERNAME",
)

// inheritPrefixes are inherited by (case-insensitive) prefix — still subject
// to the credential rule.
var inheritPrefixes = []string{
	"LC_", "XDG_",
	"AILANG_", "OTEL_",
	"CLAUDE_", "ANTHROPIC_", "CODEX_", "PI_", "MOTOKO_", "OPENCODE_", "OLLAMA_", "GEMINI_",
	"OPENROUTER_", "OPENAI_", "ZAI_", "LYCEUM_", // provider routing (BASE_URL, headers); keys are credentials
	"MISSION_", "RIG_", // mission-loop and rig-lock configuration the stage's hooks and extensions read
	"NVM_", "NODE_", "NPM_CONFIG_", "HOMEBREW_", "PLAYWRIGHT_",
}

// credentialSegments mark a name as a credential when any "_"-separated
// segment equals one of them (so MAX_OUTPUT_TOKENS is not a token, but
// GITHUB_TOKEN and AILANG_KMS_KEY are).
var credentialSegments = toSet(
	"KEY", "TOKEN", "SECRET", "SECRETS", "PASSWORD", "PASSWD", "PASS", "CREDENTIAL",
	"CREDENTIALS", "APIKEY", "PAT", "DSN", "WEBHOOK", "COOKIE", "PRIVATE",
)

// credentialExact and credentialPrefixes cover credentials whose names have
// no tell-tale segment.
var credentialExact = toSet("SSH_AUTH_SOCK", "GIT_ASKPASS", "SSH_ASKPASS", "KUBECONFIG", "DOCKER_CONFIG", "NETRC")
var credentialPrefixes = []string{"AWS_", "AZURE_", "ARM_CLIENT", "BROWSERBASE_"}

// IsCredentialName reports whether an environment variable name is
// credential-shaped and therefore withheld from an agent child unless granted.
func IsCredentialName(name string) bool {
	upper := strings.ToUpper(name)
	if credentialExact[upper] {
		return true
	}
	for _, p := range credentialPrefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	for _, seg := range strings.Split(upper, "_") {
		if credentialSegments[seg] {
			return true
		}
	}
	return false
}

// inheritAllowed reports whether a non-credential name is on the allowlist.
func inheritAllowed(name string) bool {
	upper := strings.ToUpper(name)
	if inheritExact[upper] {
		return true
	}
	for _, p := range inheritPrefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	return false
}

// Allows reports whether the child may inherit name from the host.
func (p EnvPolicy) Allows(name string) bool {
	if containsFold(p.Grants, name) || containsFold(p.Extra, name) {
		return true
	}
	return inheritAllowed(name) && !IsCredentialName(name)
}

// ResolveEnvPolicy derives the child-env policy for one executor and task.
// model is the model the executor will actually run (its own default when
// the task names none). An error means a grant could not be derived — an
// unreadable program policy — and the run must not start with a guessed env.
func ResolveEnvPolicy(executorName, model string, task *Task) (EnvPolicy, error) {
	grants := map[string]bool{}
	for _, n := range executorCredentialGrants(executorName, model) {
		grants[n] = true
	}
	if task != nil && task.PolicyPath != "" {
		res, _, err := policy.LoadResolved(task.PolicyPath)
		if err != nil {
			return EnvPolicy{}, fmt.Errorf("agent env: cannot derive the program policy's credential grants: %w", err)
		}
		for _, n := range PolicyCredentialVars(res) {
			grants[n] = true
		}
	}
	return EnvPolicy{
		Executor: executorName,
		Grants:   sortedKeys(grants),
		Extra:    config.ExecutorEnvInherit(),
	}, nil
}

// executorCredentialGrants is each executor's inference credential — the
// one thing a model-facing CLI genuinely needs from the environment.
func executorCredentialGrants(executorName, model string) []string {
	switch executorName {
	case envExecClaude:
		// OAuth lanes authenticate from ~/.claude/.credentials.json, written by
		// the parent; CLAUDE_CODE_OAUTH_TOKEN never reaches the child (the
		// env var also crashes Claude Code). The metered key only under
		// AILANG_AUTH_MODE=apikey, where the parent has KMS-decrypted it.
		if config.AuthMode() == "apikey" {
			return []string{config.EnvAnthropicAPIKey}
		}
		return nil
	case envExecCodex:
		// Cloud codex reads ~/.codex/auth.json installed by the parent;
		// the key covers local and the eval jobs.
		return []string{config.EnvOpenAIAPIKey, "CODEX_API_KEY"}
	case envExecPi, envExecOpenCode:
		return modelProviderCredentialVars(model)
	case envExecMotoko:
		// The ratified motoko set (motoko.go, EXECUTOR_SHAPE.md §8): never
		// ANTHROPIC_API_KEY — Claude routes via OpenRouter.
		return []string{config.EnvOpenRouterAPIKey, config.EnvOpenAIAPIKey, config.EnvGeminiAPIKey}
	}
	return nil
}

// modelProviderCredentialVars maps a pi/opencode "provider/model" string to
// the credential variable(s) of that provider. An unknown provider (a custom
// models.json entry) gets none; the operator grants it with
// AILANG_EXECUTOR_ENV_INHERIT.
func modelProviderCredentialVars(model string) []string {
	provider, _, ok := strings.Cut(strings.ToLower(strings.TrimSpace(model)), "/")
	if !ok {
		return nil
	}
	switch provider {
	case "openrouter":
		return []string{config.EnvOpenRouterAPIKey}
	case "google", "gemini":
		return []string{config.EnvGeminiAPIKey, config.EnvGoogleAPIKey}
	case "google-vertex", "vertex":
		return []string{"GOOGLE_APPLICATION_CREDENTIALS"}
	case "openai", "openai-codex":
		return []string{config.EnvOpenAIAPIKey}
	case "anthropic":
		return []string{config.EnvAnthropicAPIKey}
	case "ollama", "ollama-cloud":
		return []string{config.EnvOllamaAPIKey}
	case "groq":
		return []string{"GROQ_API_KEY"}
	case "xai":
		return []string{"XAI_API_KEY"}
	case "mistral":
		return []string{"MISTRAL_API_KEY"}
	case "deepseek":
		return []string{"DEEPSEEK_API_KEY"}
	case "cerebras":
		return []string{"CEREBRAS_API_KEY"}
	case "zai":
		return []string{"ZAI_API_KEY"}
	}
	return nil
}

// PolicyCredentialVars names the environment variables a program policy's
// worker may select: the pinned AI provider's credentials when the policy
// admits AI, and std/web's backend key when it admits Net to that host. It is
// the one list shared by the restricted worker (`ailang run --policy`) and the
// agent child that launches it — a key the child does not hold cannot reach
// the worker.
func PolicyCredentialVars(res *policy.Resolved) []string {
	if res == nil {
		return nil
	}
	var vars []string
	if res.Admits("AI") {
		vars = append(vars, ProviderCredentialVars(res.AIProvider)...)
	}
	if res.Admits("Net") {
		vars = append(vars, effects.WebCredentialVars(res.NetAllow)...)
	}
	return vars
}

// ProviderCredentialVars names the environment variables the pinned AI
// provider reads. "stub" needs none.
func ProviderCredentialVars(aiProvider string) []string {
	if aiProvider == "" || aiProvider == "stub" {
		return nil
	}
	provider := ai.GuessProvider(aiProvider)
	var vars []string
	if v := ai.EnvVarForProvider(provider); v != "" {
		vars = append(vars, v)
	}
	switch provider {
	case ai.ProviderGoogle:
		vars = append(vars, config.EnvGeminiAPIKey, "GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_LOCATION", "GOOGLE_GENAI_USE_VERTEXAI")
	case ai.ProviderOllama:
		vars = append(vars, "OLLAMA_HOST", config.EnvOllamaAPIKey)
	}
	return vars
}

// filterInherited applies the policy to the host environment and returns the
// entries the child may inherit, plus the credential-shaped names withheld
// (names only, for the one-line operator notice).
func (p EnvPolicy) filterInherited(environ []string) (kept []string, withheldCredentials []string) {
	for _, kv := range environ {
		name, _, ok := splitEnvEntry(kv)
		if !ok {
			continue
		}
		if strings.HasPrefix(name, "=") { // Windows per-drive cwd entries
			kept = append(kept, kv)
			continue
		}
		if p.Allows(name) {
			kept = append(kept, kv)
			continue
		}
		if IsCredentialName(name) {
			withheldCredentials = append(withheldCredentials, name)
		}
	}
	sort.Strings(withheldCredentials)
	return kept, withheldCredentials
}

// EnvNamesDigest is the sha256 over the sorted, de-duplicated NAMES of a
// child environment (M-EXECUTOR-ENV-HARDENING D6). Values never enter it, so
// it is safe to bank; identical name sets give identical digests.
func EnvNamesDigest(env []string) string {
	names := make([]string, 0, len(env))
	seen := map[string]bool{}
	for _, kv := range env {
		name, _, ok := splitEnvEntry(kv)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	sum := sha256.Sum256([]byte(strings.Join(names, "\n")))
	return hex.EncodeToString(sum[:])
}

func toSet(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[strings.ToUpper(n)] = true
	}
	return m
}

func containsFold(list []string, name string) bool {
	for _, n := range list {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
