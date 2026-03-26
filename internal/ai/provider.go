// Package ai provides a unified interface for AI provider implementations.
//
// This package is designed to be used by:
//   - Eval harness (internal/eval_harness/)
//   - CLI --ai flag (cmd/ailang/)
//   - std/ai effect (internal/effects/)
//
// Each provider (openai, gemini, anthropic) implements the Provider interface
// and can be wrapped with Handler for use with the effects system.
package ai

import (
	"context"
	"fmt"
)

// Request represents a generic AI request.
type Request struct {
	// Model is the model name (e.g., "gemini-2.5-flash", "gpt-5", "claude-sonnet-4-5")
	Model string

	// SystemPrompt contains system/developer instructions
	SystemPrompt string

	// UserPrompt contains the user message
	UserPrompt string

	// MaxTokens is the maximum number of response tokens (0 = provider default)
	MaxTokens int

	// Temperature controls randomness (0.0-2.0, 0 = provider default)
	Temperature float64

	// ResponseFormat controls structured output. Values: "json" or "" (text).
	// When set to "json", providers configure their native structured output.
	ResponseFormat string

	// ResponseSchema is an optional JSON Schema string for structured output.
	// When provided with ResponseFormat="json", providers enforce this schema.
	// When empty with ResponseFormat="json", providers return valid JSON without schema.
	ResponseSchema string

	// ResponseModalities controls output types. Values: ["TEXT"], ["IMAGE"], ["TEXT", "IMAGE"].
	// When set to ["IMAGE"], providers that support image generation will return image data.
	ResponseModalities []string

	// ImageOptions configures image generation parameters (used with ResponseModalities containing "IMAGE").
	ImageOptions *ImageOptions

	// Options contains provider-specific options
	Options map[string]any
}

// ImageOptions configures image generation parameters.
type ImageOptions struct {
	// AspectRatio controls the image aspect ratio (e.g., "1:1", "16:9", "9:16").
	AspectRatio string

	// MIMEType controls the output format (e.g., "image/png", "image/jpeg").
	MIMEType string
}

// Response represents a generic AI response.
type Response struct {
	// Text is the generated text content
	Text string

	// ImageData contains raw image bytes (PNG/JPEG) when the response includes an image.
	// Nil for text-only responses.
	ImageData []byte

	// ImageMIME is the MIME type of ImageData (e.g., "image/png").
	// Empty for text-only responses.
	ImageMIME string

	// InputTokens is the number of prompt/input tokens
	InputTokens int

	// OutputTokens is the number of completion/output tokens
	OutputTokens int

	// TotalTokens is InputTokens + OutputTokens
	TotalTokens int

	// ReasonTokens is the number of reasoning tokens (for o1/codex models)
	ReasonTokens int

	// Model is the model that was actually used (may differ from request)
	Model string
}

// RequestsImage returns true if the request asks for image generation.
func RequestsImage(req *Request) bool {
	for _, m := range req.ResponseModalities {
		if m == "IMAGE" {
			return true
		}
	}
	return false
}

// Provider interface for AI providers.
// Each provider (openai, gemini, anthropic) implements this interface.
type Provider interface {
	// Generate makes a single completion request.
	// The context can be used for cancellation and timeouts.
	Generate(ctx context.Context, req *Request) (*Response, error)

	// Name returns the provider name (e.g., "openai", "gemini", "anthropic")
	Name() string
}

// ProviderError represents an error from an AI provider.
type ProviderError struct {
	Provider   string // Provider name
	StatusCode int    // HTTP status code (0 if not applicable)
	Message    string // Error message
	Err        error  // Underlying error (may be nil)
}

func (e *ProviderError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s error (%d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Provider, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError.
func NewProviderError(provider string, statusCode int, message string, err error) *ProviderError {
	return &ProviderError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}
