// Package eval_harness provides AI code generation benchmarking.
// This file provides the adapter layer between unified ai.Provider and eval harness.
package eval_harness

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// providerAdapter wraps ai.Provider for eval harness use.
type providerAdapter struct {
	provider ai.Provider
	model    string
}

// newProviderAdapter creates a provider adapter for the given model.
func newProviderAdapter(model string, apiKey string) (*providerAdapter, error) {
	providerType := guessProvider(model)

	var provider ai.Provider
	switch providerType {
	case "openai":
		provider = openai.NewClient(apiKey)
	case "anthropic":
		provider = anthropic.NewClient(apiKey)
	case "google":
		// Gemini uses ADC (Application Default Credentials) via Vertex AI
		client, err := gemini.NewVertexAIClient("")
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		provider = client
	case "ollama":
		// Ollama is local, strip the "ollama:" prefix for the model name
		client, err := ollama.NewClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama client: %w", err)
		}
		// Strip ollama: prefix if present
		model = strings.TrimPrefix(model, "ollama:")
		provider = client
	case "openrouter":
		// Strip optional explicit "openrouter:" prefix; the model name itself
		// is "vendor/model" (e.g., "anthropic/claude-sonnet-4.5").
		model = strings.TrimPrefix(model, "openrouter:")
		provider = openrouter.NewClient(apiKey)
	default:
		return nil, fmt.Errorf("unsupported provider for model: %s", model)
	}

	return &providerAdapter{
		provider: provider,
		model:    model,
	}, nil
}

// generate calls the unified provider and converts to GenerateResult.
func (p *providerAdapter) generate(ctx context.Context, prompt string) (*GenerateResult, error) {
	systemPrompt := "You are a code generation engine. Output ONLY a complete, runnable program that solves the given task. " +
		"Do NOT output explanations, markdown formatting, or conversational text. " +
		"Do NOT output placeholder or stub code — implement the full solution. " +
		"Output raw code only, no code fences."

	req := &ai.Request{
		Model:        p.model,
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
		MaxTokens:    4096,
	}

	resp, err := p.provider.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &GenerateResult{
		Code:         extractCodeFromMarkdown(resp.Text),
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		TotalTokens:  resp.TotalTokens,
		Model:        resp.Model,
	}, nil
}

// guessProvider determines the provider from a model name.
// This duplicates logic from ai.GuessProvider but returns string for compatibility.
func guessProvider(model string) string {
	lower := strings.ToLower(model)

	// Check for explicit ollama: prefix first (highest priority)
	if strings.HasPrefix(lower, "ollama:") {
		return "ollama"
	}

	// Explicit openrouter: prefix
	if strings.HasPrefix(lower, "openrouter:") {
		return "openrouter"
	}

	// OpenRouter convention: model strings are "vendor/model" with a "/" in
	// them. Detect this BEFORE the bare-prefix checks so that
	// "anthropic/claude-..." routes to OpenRouter, not the direct Anthropic API.
	if strings.Contains(lower, "/") {
		orVendors := []string{
			"anthropic/", "openai/", "google/", "meta-llama/", "mistralai/",
			"deepseek/", "qwen/", "nvidia/", "x-ai/", "cohere/", "openrouter/",
		}
		for _, v := range orVendors {
			if strings.HasPrefix(lower, v) {
				return "openrouter"
			}
		}
	}

	// Check common prefixes
	switch {
	case strings.HasPrefix(lower, "gpt"),
		strings.HasPrefix(lower, "o1"),
		strings.HasPrefix(lower, "o3"),
		strings.HasPrefix(lower, "codex"):
		return "openai"
	case strings.HasPrefix(lower, "claude"):
		return "anthropic"
	case strings.HasPrefix(lower, "gemini"):
		return "google"
	}

	// Check for provider keywords
	switch {
	case strings.Contains(lower, "openai"):
		return "openai"
	case strings.Contains(lower, "anthropic"):
		return "anthropic"
	case strings.Contains(lower, "google"), strings.Contains(lower, "vertex"):
		return "google"
	case strings.Contains(lower, "ollama"):
		return "ollama"
	}

	return "unknown"
}

// getAPIKeyForProvider returns the API key for the given provider.
func getAPIKeyForProvider(provider string, model string) (string, error) {
	var envVar string
	switch provider {
	case "openai":
		envVar = "OPENAI_API_KEY"
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
	case "google":
		// Google uses ADC, no API key required
		return "", nil
	case "ollama":
		// Ollama is local, no API key required
		return "", nil
	case "openrouter":
		envVar = "OPENROUTER_API_KEY"
	default:
		return "", fmt.Errorf("unsupported provider: %s (model: %s)", provider, model)
	}

	key := os.Getenv(envVar)
	if key == "" {
		return "", fmt.Errorf("%s environment variable not set (required for model: %s)", envVar, model)
	}
	return key, nil
}

// extractCodeFromMarkdown strips markdown code fences if present.
// This is eval-specific logic that doesn't belong in the unified ai package.
func extractCodeFromMarkdown(text string) string {
	// Trim leading/trailing whitespace first
	text = strings.TrimSpace(text)
	lines := []byte(text)

	// Check if starts with ``` (after trimming)
	if len(lines) > 3 && lines[0] == '`' && lines[1] == '`' && lines[2] == '`' {
		// Find first newline (end of opening fence)
		start := 0
		for i, b := range lines {
			if b == '\n' {
				start = i + 1
				break
			}
		}

		// Find last ``` working backwards
		end := len(lines)
		for i := len(lines) - 1; i >= 2; i-- {
			if lines[i] == '`' && lines[i-1] == '`' && lines[i-2] == '`' {
				// Check if this is at start of line or has newline before it
				end = i - 2
				// Trim trailing newline before closing fence
				if end > 0 && lines[end-1] == '\n' {
					end--
				}
				break
			}
		}

		if start < end {
			return string(lines[start:end])
		}
	}

	return text
}
