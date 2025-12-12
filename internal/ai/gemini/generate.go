package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sunholo/ailang/internal/ai"
)

// generateContent uses the generateContent API.
func (c *Client) generateContent(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Build request
	apiReq := generateRequest{
		Contents: []content{
			{
				Role: "user",
				Parts: []part{
					{Text: req.UserPrompt},
				},
			},
		},
	}

	// Add system instruction if provided
	if req.SystemPrompt != "" {
		apiReq.SystemInstruction = &content{
			Parts: []part{
				{Text: req.SystemPrompt},
			},
		}
	}

	// Add generation config if needed
	if req.MaxTokens > 0 || req.Temperature > 0 {
		apiReq.GenerationConfig = &generationConfig{}
		if req.MaxTokens > 0 {
			apiReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
		}
		if req.Temperature > 0 {
			apiReq.GenerationConfig.Temperature = req.Temperature
		}
	}

	// Build URL based on auth type
	url, err := c.buildURL(req.Model)
	if err != nil {
		return nil, err
	}

	// Marshal request
	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, ai.NewProviderError("gemini", 0, "failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, ai.NewProviderError("gemini", 0, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add authentication
	if err := c.addAuth(httpReq); err != nil {
		return nil, err
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ai.NewProviderError("gemini", 0, "request failed", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ai.NewProviderError("gemini", resp.StatusCode, "failed to read response", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, ai.NewProviderError("gemini", resp.StatusCode, errResp.Error.Message, nil)
		}
		return nil, ai.NewProviderError("gemini", resp.StatusCode, string(body), nil)
	}

	// Parse successful response
	var result generateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, ai.NewProviderError("gemini", 0, "failed to parse response", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, ai.NewProviderError("gemini", 0, "no content in response", nil)
	}

	// Extract text from parts
	var text string
	for _, part := range result.Candidates[0].Content.Parts {
		text += part.Text
	}

	// Calculate output tokens
	outputTokens := result.UsageMetadata.CandidatesTokenCount
	reasoningTokens := result.UsageMetadata.ThoughtsTokenCount

	return &ai.Response{
		Text:         text,
		InputTokens:  result.UsageMetadata.PromptTokenCount,
		OutputTokens: outputTokens,
		TotalTokens:  result.UsageMetadata.TotalTokenCount,
		ReasonTokens: reasoningTokens,
		Model:        req.Model,
	}, nil
}

// buildURL constructs the API URL based on auth type.
func (c *Client) buildURL(model string) (string, error) {
	// Use custom base URL if set (for testing)
	if c.baseURL != "" {
		return fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, model), nil
	}

	switch c.authType {
	case AuthAPIKey:
		// AI Studio: https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent?key={key}
		return fmt.Sprintf("%s/models/%s:generateContent?key=%s", aiStudioBaseURL, model, c.apiKey), nil

	case AuthADC:
		// Vertex AI: https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/publishers/google/models/{model}:generateContent
		return fmt.Sprintf("%s/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
			vertexAIBaseURL, c.projectID, c.location, model), nil

	default:
		return "", ai.NewProviderError("gemini", 0, "unknown auth type", nil)
	}
}

// addAuth adds authentication headers to the request.
func (c *Client) addAuth(req *http.Request) error {
	switch c.authType {
	case AuthAPIKey:
		// API key is in URL query param, no header needed
		return nil

	case AuthADC:
		// Get access token from gcloud ADC
		token, err := getAccessToken()
		if err != nil {
			return ai.NewProviderError("gemini", 0, "failed to get access token (run 'gcloud auth application-default login')", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil

	default:
		return ai.NewProviderError("gemini", 0, "unknown auth type", nil)
	}
}
