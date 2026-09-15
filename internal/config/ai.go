package config

import "strings"

// The in-process AI clients (internal/ai): endpoint overrides for the
// OpenAI-compatible providers, OpenRouter's attribution headers and the raw
// HTTP wire log. Credentials are in providers.go; the Ollama knobs in
// ollama.go. A provider's credential variable is chosen at runtime by
// ai.EnvVarForProvider (or a --api-key-env flag) and read through Raw.
const (
	EnvOpenAIBaseURL      = "OPENAI_BASE_URL"
	EnvLyceumBaseURL      = "LYCEUM_BASE_URL"
	EnvZAIBaseURL         = "ZAI_BASE_URL"
	EnvOpenRouterReferer  = "OPENROUTER_HTTP_REFERER"
	EnvOpenRouterTitle    = "OPENROUTER_X_TITLE"
	EnvOpenRouterCategory = "OPENROUTER_CATEGORIES"
	EnvAIHTTPLog          = "AILANG_AI_HTTP_LOG"
	EnvBrowserbaseAPIKey  = "BROWSERBASE_API_KEY"
	EnvBrowserbaseProject = "BROWSERBASE_PROJECT_ID"
)

// DefaultLyceumBaseURL is the EU-hosted Lyceum OpenAI-compatible endpoint
// (M-LYCEUM-PROVIDER D2: constant + env override, not a models.yml field).
const DefaultLyceumBaseURL = "https://api.lyceum.technology/openai/v1"

// DefaultZAIBaseURL is z.ai's first-party OpenAI-compatible PAYG endpoint.
// NOT the coding-plan endpoint (/api/coding/paas/v4), which is contractually
// restricted to officially supported tools (M-ZAI-WINDOW-ROUTING V5).
const DefaultZAIBaseURL = "https://api.z.ai/api/paas/v4"

var aiVars = []Var{
	{EnvOpenAIBaseURL, "", AreaAI, "Base URL for the OpenAI provider when the caller gives none; set alone it allows an unauthenticated custom endpoint."},
	{EnvLyceumBaseURL, DefaultLyceumBaseURL, AreaAI, "Overrides the Lyceum OpenAI-compatible endpoint, for tests and proxies."},
	{EnvZAIBaseURL, DefaultZAIBaseURL, AreaAI, "Overrides z.ai's PAYG OpenAI-compatible endpoint, for tests and proxies; pointing it at the coding-plan endpoint is a usage-policy violation."},
	{EnvOpenRouterReferer, "", AreaAI, "HTTP-Referer attribution header sent to OpenRouter; unset uses the built-in default, and a per-request Attribution overrides both."},
	{EnvOpenRouterTitle, "", AreaAI, "X-Title attribution header sent to OpenRouter; same precedence as the referer."},
	{EnvOpenRouterCategory, "", AreaAI, "X-OpenRouter-Categories header sent to OpenRouter; same precedence as the referer."},
	{EnvAIHTTPLog, "", AreaAI, "Path the OpenAI-compatible clients append their raw HTTP wire log to; unset falls back to the ai-http-log sentinel file under the state dir, and no sentinel means logging off."},
	{EnvBrowserbaseAPIKey, "", AreaAI, "Browserbase API key for browser eval sessions when the benchmark's browser config does not name another variable (read through Raw, since the name is configurable)."},
	{EnvBrowserbaseProject, "", AreaAI, "Browserbase project id for browser eval sessions; same rule as the API key."},
}

// OpenAIBaseURL returns the trimmed OPENAI_BASE_URL, "" when unset.
func OpenAIBaseURL() string { return strings.TrimSpace(get(EnvOpenAIBaseURL)) }

// LyceumBaseURL returns the trimmed LYCEUM_BASE_URL, else DefaultLyceumBaseURL.
func LyceumBaseURL() string {
	if v := strings.TrimSpace(get(EnvLyceumBaseURL)); v != "" {
		return v
	}
	return defaultOf(EnvLyceumBaseURL)
}

// ZAIBaseURL returns the trimmed ZAI_BASE_URL, else DefaultZAIBaseURL.
func ZAIBaseURL() string {
	if v := strings.TrimSpace(get(EnvZAIBaseURL)); v != "" {
		return v
	}
	return defaultOf(EnvZAIBaseURL)
}

// OpenRouterAttribution is the three OPENROUTER_* attribution overrides,
// each "" when unset.
type OpenRouterAttribution struct {
	HTTPReferer, XTitle, Categories string
}

// OpenRouterAttributionConfig returns the OPENROUTER_* attribution values verbatim.
func OpenRouterAttributionConfig() OpenRouterAttribution {
	return OpenRouterAttribution{
		HTTPReferer: get(EnvOpenRouterReferer),
		XTitle:      get(EnvOpenRouterTitle),
		Categories:  get(EnvOpenRouterCategory),
	}
}

// AIHTTPLog returns the trimmed AILANG_AI_HTTP_LOG, "" when unset.
func AIHTTPLog() string { return strings.TrimSpace(get(EnvAIHTTPLog)) }
