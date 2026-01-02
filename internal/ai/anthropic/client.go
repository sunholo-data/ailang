// Package anthropic provides an Anthropic Claude API client implementing the ai.Provider interface.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sunholo/ailang/internal/ai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var anthropicTracer = otel.Tracer("ai.anthropic")

const (
	defaultBaseURL    = "https://api.anthropic.com/v1"
	defaultAPIVersion = "2023-06-01"
)

// Client implements ai.Provider for Anthropic's Claude API.
type Client struct {
	apiKey     string
	baseURL    string
	apiVersion string
	httpClient *http.Client
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithAPIVersion sets the Anthropic API version.
func WithAPIVersion(version string) ClientOption {
	return func(c *Client) {
		c.apiVersion = version
	}
}

// NewClient creates a new Anthropic client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		apiVersion: defaultAPIVersion,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// messagesRequest represents the request body for the Messages API.
type messagesRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	System      string           `json:"system,omitempty"`
	Messages    []messageContent `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
}

type messageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesResponse represents the response from the Messages API.
type messagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Model        string         `json:"model"`
	Content      []contentBlock `json:"content"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// errorResponse represents an error response from the API.
type errorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Generate implements ai.Provider.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Start OTEL span
	ctx, span := anthropicTracer.Start(ctx, "anthropic.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "anthropic"),
			attribute.String("ai.model", req.Model),
		),
	)
	defer span.End()

	// Build request
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	apiReq := messagesRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Messages: []messageContent{
			{Role: "user", Content: req.UserPrompt},
		},
	}

	if req.SystemPrompt != "" {
		apiReq.System = req.SystemPrompt
	}

	if req.Temperature > 0 {
		apiReq.Temperature = req.Temperature
	}

	// Marshal request
	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal request")
		return nil, ai.NewProviderError("anthropic", 0, "failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create request")
		return nil, ai.NewProviderError("anthropic", 0, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", c.apiVersion)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "request failed")
		return nil, ai.NewProviderError("anthropic", 0, "request failed", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read response")
		return nil, ai.NewProviderError("anthropic", resp.StatusCode, "failed to read response", err)
	}

	// Handle errors
	if resp.StatusCode != 200 {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			span.SetStatus(codes.Error, errResp.Error.Message)
			return nil, ai.NewProviderError("anthropic", resp.StatusCode, errResp.Error.Message, nil)
		}
		span.SetStatus(codes.Error, string(body))
		return nil, ai.NewProviderError("anthropic", resp.StatusCode, string(body), nil)
	}

	// Parse successful response
	var result messagesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse response")
		return nil, ai.NewProviderError("anthropic", 0, "failed to parse response", err)
	}

	// Extract text from content blocks
	var text string
	for _, block := range result.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	if text == "" {
		span.SetStatus(codes.Error, "empty response")
		return nil, ai.NewProviderError("anthropic", 0, "empty response from Claude", nil)
	}

	// Record success metrics on span
	span.SetAttributes(
		attribute.Int("ai.tokens_in", result.Usage.InputTokens),
		attribute.Int("ai.tokens_out", result.Usage.OutputTokens),
		attribute.Int("ai.tokens_total", result.Usage.InputTokens+result.Usage.OutputTokens),
	)

	return &ai.Response{
		Text:         text,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		TotalTokens:  result.Usage.InputTokens + result.Usage.OutputTokens,
		Model:        result.Model,
	}, nil
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "anthropic"
}

// NewHandler creates an ai.Handler wrapping this client.
// This is a convenience function for common use cases.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}
