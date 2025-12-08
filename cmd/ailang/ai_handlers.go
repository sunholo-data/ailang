// AI effect handlers for the CLI
//
// Provides real AI handlers for --ai flag
// Supports multiple providers: anthropic, openai, google
// Uses models.yml configuration for model lookup

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval_harness"
)

// AIProvider represents an AI provider type
type AIProvider string

const (
	ProviderAnthropic AIProvider = "anthropic"
	ProviderOpenAI    AIProvider = "openai"
	ProviderGoogle    AIProvider = "google"
)

// AnthropicHandler implements effects.AIHandler for Claude API
type AnthropicHandler struct {
	apiKey string
	model  string // API model name (e.g., "claude-haiku-4-5-20251001")
}

// NewAnthropicHandler creates a handler for Claude API
func NewAnthropicHandler(model, apiKey string) *AnthropicHandler {
	return &AnthropicHandler{apiKey: apiKey, model: model}
}

// Call implements the effects.AIHandler interface
func (h *AnthropicHandler) Call(input string) (string, error) {
	// Claude API request
	reqBody := map[string]interface{}{
		"model":      h.model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": input},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	return result.Content[0].Text, nil
}

// OpenAIHandler implements effects.AIHandler for OpenAI API
type OpenAIHandler struct {
	apiKey string
	model  string // API model name (e.g., "gpt-5-mini")
}

// NewOpenAIHandler creates a handler for OpenAI API
func NewOpenAIHandler(model, apiKey string) *OpenAIHandler {
	return &OpenAIHandler{apiKey: apiKey, model: model}
}

// Call implements the effects.AIHandler interface
func (h *OpenAIHandler) Call(input string) (string, error) {
	// OpenAI API request
	reqBody := map[string]interface{}{
		"model":      h.model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": input},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

// GoogleHandler implements effects.AIHandler for Google Gemini API
type GoogleHandler struct {
	apiKey string
	model  string // API model name (e.g., "gemini-2.5-flash")
}

// NewGoogleHandler creates a handler for Google Gemini API
func NewGoogleHandler(model, apiKey string) *GoogleHandler {
	return &GoogleHandler{apiKey: apiKey, model: model}
}

// Call implements the effects.AIHandler interface
func (h *GoogleHandler) Call(input string) (string, error) {
	// Google Gemini API request
	// Endpoint: https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", h.model, h.apiKey)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": input},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

// setupAIHandler configures the AI effect handler based on CLI flags
func setupAIHandler(effCtx *effects.EffContext, aiStub bool, aiModel string) error {
	if aiStub {
		effCtx.AI = effects.NewAIContext(effects.NewStubAIHandler())
		return nil
	}

	if aiModel == "" {
		// No AI handler configured - will fail at runtime if AI effect used
		return nil
	}

	// Load models config to look up model details
	if err := eval_harness.InitModelsConfig(); err != nil {
		// Config not found - try to use model name directly with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel)
	}

	// Look up model in config
	model, err := eval_harness.GlobalModelsConfig.GetModel(aiModel)
	if err != nil {
		// Model not in config - try direct usage with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel)
	}

	// Get API key from environment
	apiKey := os.Getenv(model.EnvVar)
	if apiKey == "" {
		return fmt.Errorf("%s environment variable required for model %s", model.EnvVar, aiModel)
	}

	// Create handler based on provider
	var handler effects.AIHandler
	switch AIProvider(model.Provider) {
	case ProviderAnthropic:
		handler = NewAnthropicHandler(model.APIName, apiKey)
	case ProviderOpenAI:
		handler = NewOpenAIHandler(model.APIName, apiKey)
	case ProviderGoogle:
		handler = NewGoogleHandler(model.APIName, apiKey)
	default:
		return fmt.Errorf("unsupported AI provider: %s", model.Provider)
	}

	effCtx.AI = effects.NewAIContext(handler)
	return nil
}

// setupAIHandlerDirect creates an AI handler using the model name directly
// (fallback when models.yml is not available)
func setupAIHandlerDirect(effCtx *effects.EffContext, modelName string) error {
	// Guess provider from model name prefix
	provider := guessProvider(modelName)

	var handler effects.AIHandler
	var envVar string

	switch AIProvider(provider) {
	case ProviderAnthropic:
		envVar = "ANTHROPIC_API_KEY"
		apiKey := os.Getenv(envVar)
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required", envVar)
		}
		handler = NewAnthropicHandler(modelName, apiKey)

	case ProviderOpenAI:
		envVar = "OPENAI_API_KEY"
		apiKey := os.Getenv(envVar)
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required", envVar)
		}
		handler = NewOpenAIHandler(modelName, apiKey)

	case ProviderGoogle:
		envVar = "GOOGLE_API_KEY"
		apiKey := os.Getenv(envVar)
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required", envVar)
		}
		handler = NewGoogleHandler(modelName, apiKey)

	default:
		return fmt.Errorf("cannot determine provider for model %s (use models.yml or prefix with claude-/gpt/gemini-)", modelName)
	}

	effCtx.AI = effects.NewAIContext(handler)
	return nil
}

// guessProvider attempts to guess the provider from model name
func guessProvider(modelName string) string {
	if len(modelName) >= 6 && modelName[:6] == "claude" {
		return "anthropic"
	}
	if len(modelName) >= 3 && modelName[:3] == "gpt" {
		return "openai"
	}
	if len(modelName) >= 6 && modelName[:6] == "gemini" {
		return "google"
	}
	return "unknown"
}
