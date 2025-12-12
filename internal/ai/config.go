package ai

import (
	"fmt"
	"os"
	"strings"
)

// ProviderType represents an AI provider.
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderGoogle    ProviderType = "google"
)

// ModelConfig contains provider-specific model configuration.
// This is a minimal struct for the ai package; full config is in eval_harness.
type ModelConfig struct {
	APIName  string       // API name to send to provider
	Provider ProviderType // Provider type
	EnvVar   string       // Environment variable for API key
}

// GuessProvider attempts to determine the provider from a model name.
// This is used when models.yml is not available.
func GuessProvider(modelName string) ProviderType {
	lower := strings.ToLower(modelName)

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
	default:
		return ProviderType(s)
	}
}

// String returns the string representation of a ProviderType.
func (p ProviderType) String() string {
	return string(p)
}
