package ai

import "testing"

// TestGuessProvider_OpenRouter ensures vendor/model strings route to
// OpenRouter even when the bare prefix would otherwise map to a direct
// provider (e.g., "anthropic/claude-..." must NOT return ProviderAnthropic).
// This is the M-AI-OPENROUTER follow-up: prevent the eval-harness vs handler
// drift that left `ailang run --ai openrouter/auto` failing with "cannot
// determine provider".
func TestGuessProvider_OpenRouter(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  ProviderType
	}{
		{"openrouter auto-router", "openrouter/auto", ProviderOpenRouter},
		{"explicit openrouter prefix", "openrouter:openrouter/auto", ProviderOpenRouter},
		{"anthropic via openrouter", "anthropic/claude-sonnet-4.5", ProviderOpenRouter},
		{"openai via openrouter", "openai/gpt-5-mini", ProviderOpenRouter},
		{"google via openrouter", "google/gemini-2.5-flash", ProviderOpenRouter},
		{"meta-llama via openrouter", "meta-llama/llama-3.1-405b", ProviderOpenRouter},
		{"mistralai via openrouter", "mistralai/mistral-large", ProviderOpenRouter},
		{"deepseek via openrouter", "deepseek/deepseek-v3", ProviderOpenRouter},
		{"qwen via openrouter", "qwen/qwen-2.5-72b", ProviderOpenRouter},
		{"x-ai via openrouter", "x-ai/grok-2", ProviderOpenRouter},
		{"cohere via openrouter", "cohere/command-r-plus", ProviderOpenRouter},
		{"nvidia via openrouter", "nvidia/llama-3.1-nemotron-70b-instruct", ProviderOpenRouter},
		// Case-insensitive
		{"uppercase vendor", "Anthropic/Claude-Sonnet-4.5", ProviderOpenRouter},

		// Direct providers (must NOT be openrouter)
		{"plain claude", "claude-sonnet-4-5", ProviderAnthropic},
		{"plain gpt", "gpt-5-mini", ProviderOpenAI},
		{"plain gemini", "gemini-2-5-flash", ProviderGoogle},
		{"ollama prefix", "ollama:llama3", ProviderOllama},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GuessProvider(tt.model)
			if got != tt.want {
				t.Errorf("GuessProvider(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

// TestEnvVarForProvider locks the provider→env-var mapping so all callers
// (cmd/ailang, eval_harness) read from a single source of truth.
func TestEnvVarForProvider(t *testing.T) {
	tests := []struct {
		provider ProviderType
		want     string
	}{
		{ProviderOpenAI, "OPENAI_API_KEY"},
		{ProviderAnthropic, "ANTHROPIC_API_KEY"},
		{ProviderGoogle, "GOOGLE_API_KEY"},
		{ProviderOllama, ""}, // local, no key
		{ProviderOpenRouter, "OPENROUTER_API_KEY"},
		{ProviderType("unknown"), ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := EnvVarForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("EnvVarForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

// TestProviderFromString_OpenRouter ensures the string→ProviderType
// converter accepts "openrouter".
func TestProviderFromString_OpenRouter(t *testing.T) {
	if got := ProviderFromString("openrouter"); got != ProviderOpenRouter {
		t.Errorf("ProviderFromString(\"openrouter\") = %q, want %q", got, ProviderOpenRouter)
	}
	if got := ProviderFromString("OpenRouter"); got != ProviderOpenRouter {
		t.Errorf("ProviderFromString(\"OpenRouter\") = %q, want %q (case-insensitive)", got, ProviderOpenRouter)
	}
}
