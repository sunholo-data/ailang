// AI effect handlers for the CLI
//
// Provides real AI handlers for --ai-anthropic flag

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sunholo/ailang/internal/effects"
)

// AnthropicHandler implements effects.AIHandler for Claude API
type AnthropicHandler struct {
	apiKey string
	model  string
}

// NewAnthropicHandler creates a handler for Claude API
// Uses ANTHROPIC_API_KEY from environment
func NewAnthropicHandler(model string) (*AnthropicHandler, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable required")
	}
	return &AnthropicHandler{apiKey: apiKey, model: model}, nil
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

// setupAIHandler configures the AI effect handler based on CLI flags
func setupAIHandler(effCtx *effects.EffContext, aiStub bool, aiAnthropic string) error {
	if aiStub {
		effCtx.AI = effects.NewAIContext(effects.NewStubAIHandler())
		return nil
	}

	if aiAnthropic != "" {
		handler, err := NewAnthropicHandler(aiAnthropic)
		if err != nil {
			return err
		}
		effCtx.AI = effects.NewAIContext(handler)
		return nil
	}

	// No AI handler configured - will fail at runtime if AI effect used
	return nil
}
