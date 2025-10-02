package eval_harness

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// AIAgent generates code using LLM APIs
type AIAgent struct {
	model  string
	apiKey string
	seed   int64
}

// NewAIAgent creates a new AI agent
func NewAIAgent(model string, seed int64) (*AIAgent, error) {
	// Resolve model name to API name and provider
	apiName, provider, err := ResolveModelName(model)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model: %w", err)
	}

	// Get API key from environment based on provider
	var apiKey string
	var envVar string

	switch provider {
	case "openai":
		envVar = "OPENAI_API_KEY"
		apiKey = os.Getenv(envVar)
		if apiKey == "" {
			return nil, fmt.Errorf("%s environment variable not set (required for model: %s)", envVar, model)
		}
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
		apiKey = os.Getenv(envVar)
		if apiKey == "" {
			return nil, fmt.Errorf("%s environment variable not set (required for model: %s)", envVar, model)
		}
	case "google":
		envVar = "GOOGLE_API_KEY"
		apiKey = os.Getenv(envVar)
		// Google supports ADC fallback, so we'll pass empty key and let callGemini handle it
		// Don't error here if key is missing
	default:
		return nil, fmt.Errorf("unsupported provider: %s (model: %s)", provider, model)
	}

	return &AIAgent{
		model:  apiName, // Use resolved API name
		apiKey: apiKey,
		seed:   seed,
	}, nil
}

// GenerateCode generates code using the LLM
func (a *AIAgent) GenerateCode(ctx context.Context, prompt string) (*GenerateResult, error) {
	// Determine provider from model name
	provider := guessProvider(a.model)

	switch provider {
	case "openai":
		return a.generateOpenAI(ctx, prompt)
	case "anthropic":
		return a.generateAnthropic(ctx, prompt)
	case "google":
		return a.generateGemini(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported provider for model: %s", a.model)
	}
}

// GenerateResult contains the result of code generation
type GenerateResult struct {
	Code   string
	Tokens int
	Model  string
}

// generateOpenAI generates code using OpenAI API
func (a *AIAgent) generateOpenAI(ctx context.Context, prompt string) (*GenerateResult, error) {
	return a.callOpenAI(ctx, prompt)
}

// generateAnthropic generates code using Anthropic API
func (a *AIAgent) generateAnthropic(ctx context.Context, prompt string) (*GenerateResult, error) {
	return a.callAnthropic(ctx, prompt)
}

// generateGemini generates code using Google Gemini API
func (a *AIAgent) generateGemini(ctx context.Context, prompt string) (*GenerateResult, error) {
	return a.callGemini(ctx, prompt)
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// GenerateWithRetry generates code with retry logic
func (a *AIAgent) GenerateWithRetry(ctx context.Context, prompt string, cfg RetryConfig) (*GenerateResult, error) {
	var lastErr error
	delay := cfg.BaseDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait with exponential backoff
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			delay *= 2
		}

		result, err := a.GenerateCode(ctx, prompt)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Rate limiting errors
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") {
		return true
	}

	// Temporary network errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") {
		return true
	}

	// Server errors
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") {
		return true
	}

	return false
}

// MockAIAgent is a mock implementation for testing
type MockAIAgent struct {
	model string
	code  string
}

// NewMockAIAgent creates a mock AI agent
func NewMockAIAgent(model, code string) *MockAIAgent {
	return &MockAIAgent{
		model: model,
		code:  code,
	}
}

// GenerateCode returns the pre-configured mock code
func (m *MockAIAgent) GenerateCode(ctx context.Context, prompt string) (*GenerateResult, error) {
	return &GenerateResult{
		Code:   m.code,
		Tokens: len(m.code) / 4, // Rough estimate
		Model:  m.model,
	}, nil
}
