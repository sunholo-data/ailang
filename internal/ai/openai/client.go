package openai

import (
	"context"
	"net/http"
	"strings"

	"github.com/sunholo/ailang/internal/ai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var openaiTracer = otel.Tracer("ai.openai")

const (
	defaultBaseURL = "https://api.openai.com/v1"
)

// Client implements ai.Provider for OpenAI's APIs.
// It supports both Chat Completions and Responses APIs.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	apiType    APIType // Override API type (empty = auto-detect)
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

// WithAPIType forces a specific API type (chat or responses).
func WithAPIType(apiType APIType) ClientOption {
	return func(c *Client) {
		c.apiType = apiType
	}
}

// NewClient creates a new OpenAI client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Generate implements ai.Provider.
// It routes to either Chat Completions or Responses API based on the model.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	apiType := c.detectAPIType(req.Model)

	// Start OTEL span
	ctx, span := openaiTracer.Start(ctx, "openai.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.api_type", string(apiType)),
		),
	)
	defer span.End()

	var resp *ai.Response
	var err error

	switch apiType {
	case APIResponses:
		resp, err = c.generateResponses(ctx, req)
	default:
		resp, err = c.generateChat(ctx, req)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Record success metrics on span
	span.SetAttributes(
		attribute.Int("ai.tokens_in", resp.InputTokens),
		attribute.Int("ai.tokens_out", resp.OutputTokens),
		attribute.Int("ai.tokens_total", resp.TotalTokens),
	)

	return resp, nil
}

// detectAPIType determines which API to use based on the model name.
func (c *Client) detectAPIType(model string) APIType {
	// If API type is explicitly set, use it
	if c.apiType != "" {
		return c.apiType
	}

	// Codex models use Responses API
	lower := strings.ToLower(model)
	if strings.Contains(lower, "codex") {
		return APIResponses
	}

	// All other models use Chat Completions
	return APIChatCompletions
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "openai"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}

// usesMaxCompletionTokens returns true if the model uses max_completion_tokens.
// All GPT-5+ models and o-series reasoning models require this instead of max_tokens.
func usesMaxCompletionTokens(model string) bool {
	lower := strings.ToLower(model)
	return strings.HasPrefix(lower, "gpt-5") || // All GPT-5 models
		strings.Contains(lower, "o1") ||
		strings.Contains(lower, "o3")
}
