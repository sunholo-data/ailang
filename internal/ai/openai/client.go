package openai

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var openaiTracer = telemetry.Tracer("ai.openai")

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
	if req.Routing != nil && (req.Routing.HasRouting() || req.Routing.PriceCapSet()) {
		return nil, ai.NewProviderError("openai", 0,
			"this provider does not support AIRoutingPolicy; use openrouter instead",
			ai.ErrRoutingNotSupported)
	}
	if ai.RequestsImage(req) {
		return nil, ai.NewProviderError("openai", 0, "image generation not supported by provider \"openai\" (model: "+req.Model+") — use a Gemini image model", nil)
	}
	apiType := c.detectAPIType(req.Model)

	// M-AI-REASONING-EFFORT: resolve reasoning controls BEFORE marshaling or
	// dispatch. Fail loud on any invalid/unsupported/conflicting control.
	reasoning, rErr := ai.ResolveReasoning(req, "openai", req.Model)
	if rErr != nil {
		return nil, rErr
	}

	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, openaiTracer, "openai.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.api_type", string(apiType)),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	var resp *ai.Response
	var err error

	switch apiType {
	case APIResponses:
		resp, err = c.generateResponses(ctx, req, reasoning)
	default:
		resp, err = c.generateChat(ctx, req, reasoning)
	}

	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Record success metrics on span
	span.SetAttributes(
		attribute.Int("ai.tokens_in", resp.InputTokens),
		attribute.Int("ai.tokens_out", resp.OutputTokens),
		attribute.Int("ai.tokens_total", resp.TotalTokens),
		attribute.String("ai.response_preview", telemetry.Truncate(resp.Text, 100)),
	)

	return resp, nil
}

// detectAPIType determines which API to use based on the model name.
// Responses API is preferred for all modern models (GPT-5+, o-series).
// Chat Completions is only used for legacy models (GPT-4 and earlier).
func (c *Client) detectAPIType(model string) APIType {
	// If API type is explicitly set, use it
	if c.apiType != "" {
		return c.apiType
	}

	lower := strings.ToLower(model)

	// Modern models use Responses API (better caching, CoT passing, lower latency)
	// GPT-5, GPT-5.1, GPT-5.2, o1, o3, codex all benefit from Responses API
	if strings.HasPrefix(lower, "gpt-5") ||
		strings.Contains(lower, "codex") ||
		strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") {
		return APIResponses
	}

	// Legacy models (GPT-4, GPT-3.5, etc.) use Chat Completions
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

// gptGenerationRe extracts the generation number from a GPT model id:
// "gpt-5" -> 5, "gpt-5.6-sol" -> 5, "gpt-6-astra" -> 6, "gpt-4o" -> 4.
var gptGenerationRe = regexp.MustCompile(`^gpt-(\d+)`)

// usesMaxCompletionTokens returns true if the model uses max_completion_tokens.
// All GPT-5+ models and o-series reasoning models require this instead of max_tokens.
//
// The generation is PARSED, not prefix-matched against a literal. A hardcoded
// strings.HasPrefix(lower, "gpt-5") silently broke the day gpt-6-astra was added
// (2026-09-05): the request went out with max_tokens and OpenAI rejected every
// call with 400 "Unsupported parameter". Anything gpt-N for N>=5 is covered here,
// so the next generation does not need a code change to be evaluated.
func usesMaxCompletionTokens(model string) bool {
	lower := strings.ToLower(model)
	if m := gptGenerationRe.FindStringSubmatch(lower); m != nil {
		if gen, err := strconv.Atoi(m[1]); err == nil {
			return gen >= 5
		}
	}
	return strings.Contains(lower, "o1") ||
		strings.Contains(lower, "o3")
}
