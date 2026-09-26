package config

import "strings"

// Provider credentials. The getters return the value so the caller can hand
// it to a client or a child process; nothing here logs or prints one. A
// provider whose credential variable is chosen at runtime
// (ai.EnvVarForProvider, a --api-key-env flag) reads it through Raw.
const (
	EnvAnthropicAPIKey      = "ANTHROPIC_API_KEY"
	EnvAnthropicAuthToken   = "ANTHROPIC_AUTH_TOKEN"
	EnvClaudeCodeOAuthToken = "CLAUDE_CODE_OAUTH_TOKEN"
	EnvOpenAIAPIKey         = "OPENAI_API_KEY"
	EnvOpenRouterAPIKey     = "OPENROUTER_API_KEY"
	EnvTypeSafeAPIKey       = "TYPESAFE_API_KEY"
	EnvOllamaAPIKey         = "OLLAMA_API_KEY"
	EnvGeminiAPIKey         = "GEMINI_API_KEY"
	EnvGoogleAPIKey         = "GOOGLE_API_KEY"
	EnvGitHubToken          = "GITHUB_TOKEN"
)

var providerVars = []Var{
	{EnvAnthropicAPIKey, "", AreaProviders, "Anthropic API key (METERED); the in-process client resolves it first, and the claude executor requires it under AILANG_AUTH_MODE=apikey and decrypts an ENC:-prefixed value with AILANG_KMS_KEY."},
	{EnvAnthropicAuthToken, "", AreaProviders, "Anthropic OAuth access token from a Claude subscription profile (SUBSCRIPTION QUOTA); the in-process client resolves it after ANTHROPIC_API_KEY, matching the official SDKs."},
	{EnvClaudeCodeOAuthToken, "", AreaProviders, "Claude Code subscription token: a JSON credential blob in cloud containers, which the in-process client resolves third; the claude executor writes it to the credentials file, and the mission loop's Anthropic quota reader uses it (an empty-but-set value deliberately bypasses the keychain)."},
	{EnvOpenAIAPIKey, "", AreaProviders, "OpenAI API key; the codex executor bootstraps auth.json from it when the file is missing."},
	{EnvOpenRouterAPIKey, "", AreaProviders, "OpenRouter API key; required by motoko smoke runs and the OpenRouter quota observer."},
	{EnvTypeSafeAPIKey, "", AreaProviders, "TypeSafe direct API key (System One decision model, Jev): read by the sunholo/decisions package's TypeSafeDirect transport via std/env; the OpenRouter transport uses OPENROUTER_API_KEY instead. No Go code reads it — the row exists so the variable is documented and gated like every other provider key."},
	{EnvOllamaAPIKey, "", AreaProviders, "Ollama Cloud API key; the mission admission policy observes quota with it."},
	{EnvGeminiAPIKey, "", AreaProviders, "Gemini API key the factory falls back to when GOOGLE_API_KEY is unset, Vertex ADC is unavailable and no key was given."},
	{EnvGoogleAPIKey, "", AreaProviders, "Google API key: the Gemini provider's credential variable and the Gemini embedder's key."},
	{EnvGitHubToken, "", AreaProviders, "GitHub token for PR creation, docs search and read-only API calls; falls back to `gh auth token` where a command can shell out."},
}

// AnthropicAPIKey returns ANTHROPIC_API_KEY, "" when unset.
func AnthropicAPIKey() string { return get(EnvAnthropicAPIKey) }

// AnthropicAuthToken returns ANTHROPIC_AUTH_TOKEN, "" when unset.
func AnthropicAuthToken() string { return get(EnvAnthropicAuthToken) }

// GeminiAPIKey returns GEMINI_API_KEY, "" when unset.
func GeminiAPIKey() string { return get(EnvGeminiAPIKey) }

// GoogleAPIKey returns GOOGLE_API_KEY, "" when unset.
func GoogleAPIKey() string { return get(EnvGoogleAPIKey) }

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

// TypeSafeAPIKey returns TYPESAFE_API_KEY, "" when unset.
func TypeSafeAPIKey() string { return get(EnvTypeSafeAPIKey) }

// OllamaAPIKey returns OLLAMA_API_KEY, "" when unset.
func OllamaAPIKey() string { return get(EnvOllamaAPIKey) }

// GitHubToken returns the trimmed GITHUB_TOKEN, "" when unset. Secret
// Manager values carry trailing newlines that net/http rejects in a header.
func GitHubToken() string { return strings.TrimSpace(get(EnvGitHubToken)) }
