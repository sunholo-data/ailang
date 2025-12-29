package eval_harness

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AIAgent generates code using LLM APIs.
// Uses the unified internal/ai/ providers via providerAdapter.
type AIAgent struct {
	friendlyName string           // Friendly name (e.g., "claude-sonnet-4-5") - used for cost lookups
	model        string           // API model name (e.g., "claude-sonnet-4-5-20250929") - used for API calls
	adapter      *providerAdapter // Unified provider adapter
	seed         int64
}

// NewAIAgent creates a new AI agent using unified providers.
func NewAIAgent(model string, seed int64) (*AIAgent, error) {
	// Resolve model name to API name and provider
	apiName, provider, err := ResolveModelName(model)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model: %w", err)
	}

	// Get API key for provider
	apiKey, err := getAPIKeyForProvider(provider, model)
	if err != nil {
		return nil, err
	}

	// Create unified provider adapter
	adapter, err := newProviderAdapter(apiName, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &AIAgent{
		friendlyName: model,   // Store original friendly name for cost lookups
		model:        apiName, // Use resolved API name for API calls
		adapter:      adapter,
		seed:         seed,
	}, nil
}

// GenerateCode generates code using the unified provider.
func (a *AIAgent) GenerateCode(ctx context.Context, prompt string) (*GenerateResult, error) {
	return a.adapter.generate(ctx, prompt)
}

// GenerateResult contains the result of code generation
type GenerateResult struct {
	Code         string
	InputTokens  int // Prompt tokens (input to LLM)
	OutputTokens int // Completion tokens (generated code)
	TotalTokens  int // Total tokens (for billing)
	Model        string
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
	outputTokens := len(m.code) / 4 // Rough estimate: ~4 chars per token
	inputTokens := len(prompt) / 4  // Rough estimate: ~4 chars per token
	return &GenerateResult{
		Code:         m.code,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        m.model,
	}, nil
}
