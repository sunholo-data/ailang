// Package anthropic provides an Anthropic Claude API client implementing the ai.Provider interface.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var anthropicTracer = telemetry.Tracer("ai.anthropic")

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
	Tools       []toolDef        `json:"tools,omitempty"`
	ToolChoice  *toolChoice      `json:"tool_choice,omitempty"`
}

type messageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// toolDef represents a tool definition for structured output via tool_use.
type toolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// toolChoice forces the model to use a specific tool.
type toolChoice struct {
	Type string `json:"type"` // "tool"
	Name string `json:"name"` // tool name
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
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`    // tool_use block ID
	Name  string          `json:"name,omitempty"`  // tool_use tool name
	Input json.RawMessage `json:"input,omitempty"` // tool_use input (the structured output)
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
	if ai.RequestsImage(req) {
		return nil, ai.NewProviderError("anthropic", 0, "image generation not supported by provider \"anthropic\" (model: "+req.Model+")", nil)
	}
	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, anthropicTracer, "anthropic.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "anthropic"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
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

	// Add structured output via tool_use pattern
	if req.ResponseFormat == "json" {
		schema := json.RawMessage(`{"type": "object"}`)
		if req.ResponseSchema != "" {
			schema = json.RawMessage(req.ResponseSchema)
		}
		apiReq.Tools = []toolDef{
			{
				Name:        "respond",
				Description: "Return the structured JSON response",
				InputSchema: schema,
			},
		}
		apiReq.ToolChoice = &toolChoice{
			Type: "tool",
			Name: "respond",
		}
	}

	// Marshal request
	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal request")
		return nil, ai.NewProviderError("anthropic", 0, "failed to marshal request", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
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
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "request failed")
		return nil, ai.NewProviderError("anthropic", 0, "request failed", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read response")
		return nil, ai.NewProviderError("anthropic", resp.StatusCode, "failed to read response", err)
	}

	// Handle errors
	if resp.StatusCode != 200 {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			span.SetAttributes(
				attribute.String("error.message", telemetry.Truncate(errResp.Error.Message, 200)),
				attribute.String("error.category", "api_error"),
			)
			span.SetStatus(codes.Error, errResp.Error.Message)
			return nil, ai.NewProviderError("anthropic", resp.StatusCode, errResp.Error.Message, nil)
		}
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(string(body), 200)),
			attribute.String("error.category", "api_error"),
		)
		span.SetStatus(codes.Error, string(body))
		return nil, ai.NewProviderError("anthropic", resp.StatusCode, string(body), nil)
	}

	// Parse successful response
	var result messagesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse response")
		return nil, ai.NewProviderError("anthropic", 0, "failed to parse response", err)
	}

	// Extract text from content blocks
	// For structured output (tool_use), extract the tool input as JSON string
	var text string
	if req.ResponseFormat == "json" {
		for _, block := range result.Content {
			if block.Type == "tool_use" && block.Name == "respond" {
				text = string(block.Input)
				break
			}
		}
	} else {
		for _, block := range result.Content {
			if block.Type == "text" {
				text += block.Text
			}
		}
	}

	if text == "" {
		span.SetAttributes(
			attribute.String("error.message", "empty response from Claude"),
			attribute.String("error.category", "api_error"),
		)
		span.SetStatus(codes.Error, "empty response")
		return nil, ai.NewProviderError("anthropic", 0, "empty response from Claude", nil)
	}

	// Record success metrics on span
	span.SetAttributes(
		attribute.Int("ai.tokens_in", result.Usage.InputTokens),
		attribute.Int("ai.tokens_out", result.Usage.OutputTokens),
		attribute.Int("ai.tokens_total", result.Usage.InputTokens+result.Usage.OutputTokens),
		attribute.String("ai.response_preview", telemetry.Truncate(text, 100)),
		attribute.String("ai.finish_reason", result.StopReason),
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
