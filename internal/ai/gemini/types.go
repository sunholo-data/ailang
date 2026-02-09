// Package gemini provides a Google Gemini API client implementing the ai.Provider interface.
// It supports both AI Studio (API key) and Vertex AI (ADC) authentication.
package gemini

import "encoding/json"

// generateRequest represents the request body for generateContent API.
type generateRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

// content represents a content block with role and parts.
type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

// part represents a content part (text, inline_data, etc).
type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inlineData,omitempty"` // For multimodal (images, PDFs, etc.)
}

// inlineData represents inline binary data for multimodal requests.
type inlineData struct {
	MimeType string `json:"mimeType"` // e.g., "application/pdf", "image/png"
	Data     string `json:"data"`     // Base64-encoded content
}

// generationConfig represents generation parameters.
type generationConfig struct {
	MaxOutputTokens  int              `json:"maxOutputTokens,omitempty"`
	Temperature      float64          `json:"temperature,omitempty"`
	TopP             float64          `json:"topP,omitempty"`
	TopK             int              `json:"topK,omitempty"`
	ResponseMimeType string           `json:"responseMimeType,omitempty"` // "application/json" for structured output
	ResponseSchema   *json.RawMessage `json:"responseSchema,omitempty"`   // JSON Schema for structured output
}

// generateResponse represents the response from generateContent API.
type generateResponse struct {
	Candidates    []candidate   `json:"candidates"`
	UsageMetadata usageMetadata `json:"usageMetadata"`
}

// candidate represents a generation candidate.
type candidate struct {
	Content       content `json:"content"`
	FinishReason  string  `json:"finishReason"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

// usageMetadata represents token usage.
type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount,omitempty"` // Reasoning tokens (Gemini 3+)
}

// errorResponse represents an error response from the API.
type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// AuthType indicates the authentication method.
type AuthType string

const (
	// AuthAPIKey uses API key authentication (AI Studio).
	AuthAPIKey AuthType = "apikey"

	// AuthADC uses Application Default Credentials (Vertex AI).
	AuthADC AuthType = "adc"
)
