package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sunholo-data/ailang/internal/ai"
)

// generateContent uses the generateContent API.
func (c *Client) generateContent(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Build request — detect multimodal JSON input and construct proper parts
	parts := buildParts(req.UserPrompt)
	apiReq := generateRequest{
		Contents: []content{
			{
				Role:  "user",
				Parts: parts,
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
	needsConfig := req.MaxTokens > 0 || req.Temperature > 0 || req.ResponseFormat == "json" || len(req.ResponseModalities) > 0
	if needsConfig {
		apiReq.GenerationConfig = &generationConfig{}
		if req.MaxTokens > 0 {
			apiReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
		}
		if req.Temperature > 0 {
			apiReq.GenerationConfig.Temperature = req.Temperature
		}
		if req.ResponseFormat == "json" {
			apiReq.GenerationConfig.ResponseMimeType = "application/json"
			if req.ResponseSchema != "" {
				raw := json.RawMessage(req.ResponseSchema)
				apiReq.GenerationConfig.ResponseSchema = &raw
			}
		}
		if len(req.ResponseModalities) > 0 {
			apiReq.GenerationConfig.ResponseModalities = req.ResponseModalities
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

	// Extract text and image data from parts
	var text string
	var imageData []byte
	var imageMIME string
	for _, part := range result.Candidates[0].Content.Parts {
		text += part.Text
		if part.InlineData != nil && part.InlineData.Data != "" {
			decoded, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				return nil, ai.NewProviderError("gemini", 0, "failed to decode image data from response", err)
			}
			imageData = decoded
			imageMIME = part.InlineData.MimeType
		}
	}

	// Calculate output tokens
	outputTokens := result.UsageMetadata.CandidatesTokenCount
	reasoningTokens := result.UsageMetadata.ThoughtsTokenCount

	return &ai.Response{
		Text:         text,
		ImageData:    imageData,
		ImageMIME:    imageMIME,
		InputTokens:  result.UsageMetadata.PromptTokenCount,
		OutputTokens: outputTokens,
		TotalTokens:  result.UsageMetadata.TotalTokenCount,
		ReasonTokens: reasoningTokens,
		Model:        req.Model,
	}, nil
}

// buildParts converts a user prompt into Gemini API parts.
// If the prompt is a JSON object with "mode": "multimodal", it constructs
// proper inline_data parts for binary content (PDFs, images, etc.).
//
// Multimodal JSON format:
//
//	{
//	  "mode": "multimodal",
//	  "mimeType": "application/pdf",
//	  "data": "<base64-encoded-content>",
//	  "prompt": "Extract content from this document"
//	}
//
// Plain text prompts are returned as a single text part.
func buildParts(userPrompt string) []part {
	// Try to parse as multimodal JSON
	var obj map[string]string
	if err := json.Unmarshal([]byte(userPrompt), &obj); err == nil {
		if obj["mode"] == "multimodal" && obj["mimeType"] != "" {
			var parts []part

			// fileUri takes precedence over data (avoids redundant base64 for large files)
			if obj["fileUri"] != "" {
				parts = []part{
					{FileData: &fileData{
						MimeType: obj["mimeType"],
						FileUri:  obj["fileUri"],
					}},
				}
			} else if obj["data"] != "" {
				parts = []part{
					{InlineData: &inlineData{
						MimeType: obj["mimeType"],
						Data:     obj["data"],
					}},
				}
			} else {
				// Neither fileUri nor data — fall through to plain text
				return []part{{Text: userPrompt}}
			}

			// Add text prompt if present
			prompt := obj["prompt"]
			if prompt == "" {
				prompt = obj["fileName"] // Fallback to filename as context
			}
			if prompt != "" {
				parts = append(parts, part{Text: prompt})
			}
			return parts
		}
	}

	// Default: plain text
	return []part{{Text: userPrompt}}
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

// buildStreamURL constructs the streaming API URL. Same shape as buildURL
// but swaps :generateContent for :streamGenerateContent and appends
// alt=sse so the response is SSE-framed (text/event-stream of one JSON
// object per data: line) rather than a raw JSON array.
//
// Used by StreamStep (M-AI-STEP-STREAMING v0.18.7).
func (c *Client) buildStreamURL(model string) (string, error) {
	if c.baseURL != "" {
		return fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", c.baseURL, model), nil
	}
	switch c.authType {
	case AuthAPIKey:
		return fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", aiStudioBaseURL, model, c.apiKey), nil
	case AuthADC:
		return fmt.Sprintf("%s/projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent?alt=sse",
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
