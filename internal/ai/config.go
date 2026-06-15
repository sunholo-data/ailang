package ai

import (
	"fmt"
	"os"
	"strings"
)

// ProviderType represents an AI provider.
type ProviderType string

const (
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderGoogle     ProviderType = "google"
	ProviderOllama     ProviderType = "ollama"
	ProviderOpenRouter ProviderType = "openrouter"
)

// openrouterVendorPrefixes lists known "vendor/" prefixes that identify a
// model string as being routed via OpenRouter. Detected before the bare
// prefix checks (claude-, gpt-, gemini-) so e.g. "anthropic/claude-..."
// routes to OpenRouter, not the direct Anthropic API.
var openrouterVendorPrefixes = []string{
	"anthropic/", "openai/", "google/", "meta-llama/", "mistralai/",
	"deepseek/", "qwen/", "nvidia/", "x-ai/", "cohere/", "openrouter/",
	"moonshotai/", "microsoft/", "z-ai/", "minimax/", // M-AI-EFFECT-MODES eval-suite expansion (May 2026)
}

// ModelConfig contains provider-specific model configuration.
// This is a minimal struct for the ai package; full config is in eval_harness.
type ModelConfig struct {
	APIName  string       // API name to send to provider
	Provider ProviderType // Provider type
	EnvVar   string       // Environment variable for API key
}

// GuessProvider attempts to determine the provider from a model name.
// This is used when models.yml is not available.
//
// OpenRouter detection precedence: checked BEFORE the bare-prefix checks so
// that vendor/model strings like "anthropic/claude-sonnet-4.5" route to
// OpenRouter rather than the direct Anthropic API. An explicit "openrouter:"
// prefix is also accepted (and stripped at handler-construction time).
func GuessProvider(modelName string) ProviderType {
	lower := strings.ToLower(modelName)

	// Check for an explicit ollama prefix first (highest priority). Both the
	// "ollama:" and "ollama/" forms route to the local Ollama provider — the
	// slash form must be matched HERE, before the generic "vendor/model" →
	// OpenRouter check below, otherwise "ollama/qwen3.5:..." falls through to
	// "cannot determine provider". (Ollama is local; OpenRouter never hosts an
	// "ollama/..." model, so claiming both prefixes is unambiguous.) The
	// provider's bareModel() strips whichever prefix before hitting the API.
	if strings.HasPrefix(lower, "ollama:") || strings.HasPrefix(lower, "ollama/") {
		return ProviderOllama
	}

	// Explicit openrouter: prefix
	if strings.HasPrefix(lower, "openrouter:") {
		return ProviderOpenRouter
	}

	// OpenRouter convention: model strings are "vendor/model" with a "/" in
	// them. Detect this BEFORE the bare-prefix checks so that
	// "anthropic/claude-..." routes to OpenRouter, not the direct Anthropic API.
	if strings.Contains(lower, "/") {
		for _, v := range openrouterVendorPrefixes {
			if strings.HasPrefix(lower, v) {
				return ProviderOpenRouter
			}
		}
	}

	// Check prefixes
	switch {
	case strings.HasPrefix(lower, "gpt"),
		strings.HasPrefix(lower, "o1"),
		strings.HasPrefix(lower, "o3"),
		strings.HasPrefix(lower, "codex"):
		return ProviderOpenAI
	case strings.HasPrefix(lower, "claude"):
		return ProviderAnthropic
	case strings.HasPrefix(lower, "gemini"):
		return ProviderGoogle
	}

	// Check for provider names in the model string
	switch {
	case strings.Contains(lower, "openai"):
		return ProviderOpenAI
	case strings.Contains(lower, "anthropic"):
		return ProviderAnthropic
	case strings.Contains(lower, "google"), strings.Contains(lower, "vertex"):
		return ProviderGoogle
	}

	return ""
}

// EnvVarForProvider returns the environment variable name that holds the
// API key for the given provider. Returns empty string for providers that
// don't need an API key (Google ADC, Ollama local).
func EnvVarForProvider(provider ProviderType) string {
	switch provider {
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ProviderGoogle:
		return "GOOGLE_API_KEY" // ADC also supported separately
	case ProviderOllama:
		return "" // Local, no API key
	case ProviderOpenRouter:
		return "OPENROUTER_API_KEY"
	default:
		return ""
	}
}

// GetAPIKey returns the API key for a provider from environment variables.
func GetAPIKey(provider ProviderType) (string, error) {
	var envVar string
	switch provider {
	case ProviderOpenAI:
		envVar = "OPENAI_API_KEY"
	case ProviderAnthropic:
		envVar = "ANTHROPIC_API_KEY"
	case ProviderGoogle:
		// Google supports multiple auth methods
		// Check for API key first, then ADC will be used
		envVar = "GOOGLE_API_KEY"
		if key := os.Getenv(envVar); key != "" {
			return key, nil
		}
		// No API key, but ADC might work - return empty string
		// Provider implementations should handle ADC separately
		return "", nil
	case ProviderOllama:
		// Ollama is local, no API key needed
		return "", nil
	case ProviderOpenRouter:
		envVar = "OPENROUTER_API_KEY"
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}

	key := os.Getenv(envVar)
	if key == "" {
		return "", fmt.Errorf("environment variable %s not set", envVar)
	}
	return key, nil
}

// ProviderFromString converts a string to ProviderType.
func ProviderFromString(s string) ProviderType {
	switch strings.ToLower(s) {
	case "openai":
		return ProviderOpenAI
	case "anthropic":
		return ProviderAnthropic
	case "google", "gemini", "vertex":
		return ProviderGoogle
	case "ollama":
		return ProviderOllama
	case "openrouter":
		return ProviderOpenRouter
	default:
		return ProviderType(s)
	}
}

// String returns the string representation of a ProviderType.
func (p ProviderType) String() string {
	return string(p)
}
