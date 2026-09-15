package config

import "strings"

// Provider credentials. The getters return the value so the caller can hand
// it to a client or a child process; nothing here logs or prints one. The
// AI-provider clients under internal/ai read their own keys today and will
// route through here when that package's Truncate migration lands.
const (
	EnvAnthropicAPIKey      = "ANTHROPIC_API_KEY"
	EnvClaudeCodeOAuthToken = "CLAUDE_CODE_OAUTH_TOKEN"
	EnvOpenAIAPIKey         = "OPENAI_API_KEY"
	EnvOpenRouterAPIKey     = "OPENROUTER_API_KEY"
	EnvOllamaAPIKey         = "OLLAMA_API_KEY"
	EnvGitHubToken          = "GITHUB_TOKEN"
)

var providerVars = []Var{
	{EnvAnthropicAPIKey, "", AreaProviders, "Anthropic API key; the claude executor requires it under AILANG_AUTH_MODE=apikey and decrypts an ENC:-prefixed value with AILANG_KMS_KEY."},
	{EnvClaudeCodeOAuthToken, "", AreaProviders, "Claude Code subscription token; the claude executor writes it to the credentials file, and the mission loop's Anthropic quota reader uses it (an empty-but-set value deliberately bypasses the keychain)."},
	{EnvOpenAIAPIKey, "", AreaProviders, "OpenAI API key; the codex executor bootstraps auth.json from it when the file is missing."},
	{EnvOpenRouterAPIKey, "", AreaProviders, "OpenRouter API key; required by motoko smoke runs and the OpenRouter quota observer."},
	{EnvOllamaAPIKey, "", AreaProviders, "Ollama Cloud API key; the mission admission policy observes quota with it."},
	{EnvGitHubToken, "", AreaProviders, "GitHub token for PR creation, docs search and read-only API calls; falls back to `gh auth token` where a command can shell out."},
}

// AnthropicAPIKey returns ANTHROPIC_API_KEY, "" when unset.
func AnthropicAPIKey() string { return get(EnvAnthropicAPIKey) }

// ClaudeCodeOAuthToken returns CLAUDE_CODE_OAUTH_TOKEN and whether it is
// present at all — an empty-but-set token is a deliberate bypass of the
// keychain lookup, so presence matters.
func ClaudeCodeOAuthToken() (string, bool) {
	if !isSet(EnvClaudeCodeOAuthToken) {
		return "", false
	}
	return get(EnvClaudeCodeOAuthToken), true
}

// OpenAIAPIKey returns OPENAI_API_KEY, "" when unset.
func OpenAIAPIKey() string { return get(EnvOpenAIAPIKey) }

// OpenRouterAPIKey returns OPENROUTER_API_KEY, "" when unset.
func OpenRouterAPIKey() string { return get(EnvOpenRouterAPIKey) }

// OllamaAPIKey returns OLLAMA_API_KEY, "" when unset.
func OllamaAPIKey() string { return get(EnvOllamaAPIKey) }

// GitHubToken returns the trimmed GITHUB_TOKEN, "" when unset. Secret
// Manager values carry trailing newlines that net/http rejects in a header.
func GitHubToken() string { return strings.TrimSpace(get(EnvGitHubToken)) }
