package openrouter

import (
	"context"
	"net/http"
	"os"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var openrouterTracer = telemetry.Tracer("ai.openrouter")

const (
	defaultBaseURL     = "https://openrouter.ai/api/v1"
	defaultHTTPReferer = "https://ailang.sunholo.com"
	defaultXTitle      = "AILANG"
	defaultCategories  = "cli-agent,cloud-agent"
)

// setAttributionHeaders stamps OpenRouter app-attribution headers on r.
//
// HTTP-Referer is mandatory for app pages and rankings.
// X-OpenRouter-Title is the new canonical header (v0.16.0+); X-Title is kept
// for backwards compatibility with older OpenRouter middleware.
// X-OpenRouter-Categories places the app in marketplace buckets.
//
// Precedence (highest to lowest):
//  1. Per-request overrides from req.Attribution (if set)
//  2. Environment variables OPENROUTER_HTTP_REFERER, OPENROUTER_X_TITLE,
//     OPENROUTER_CATEGORIES
//  3. Built-in defaults
func setAttributionHeaders(r *http.Request, attr *ai.Attribution) {
	referer := defaultHTTPReferer
	title := defaultXTitle
	categories := defaultCategories

	// Layer 2: env vars override defaults
	if v := os.Getenv("OPENROUTER_HTTP_REFERER"); v != "" {
		referer = v
	}
	if v := os.Getenv("OPENROUTER_X_TITLE"); v != "" {
		title = v
	}
	if v := os.Getenv("OPENROUTER_CATEGORIES"); v != "" {
		categories = v
	}

	// Layer 1: per-request overrides take highest precedence
	if attr != nil {
		if attr.HTTPReferer != "" {
			referer = attr.HTTPReferer
		}
		if attr.Title != "" {
			title = attr.Title
		}
		if attr.Categories != "" {
			categories = attr.Categories
		}
	}

	r.Header.Set("HTTP-Referer", referer)
	r.Header.Set("X-OpenRouter-Title", title) // Canonical (v0.16.0+)
	r.Header.Set("X-Title", title)            // Backwards compat
	r.Header.Set("X-OpenRouter-Categories", categories)
}

// Client implements ai.Provider for OpenRouter's unified Chat Completions API.
//
// Unlike the OpenAI client there is no API-type detection — OpenRouter is
// Chat-Completions only. Image generation is not supported; callers should
// use a Gemini image model instead.
type Client struct {
	apiKey     string
	baseURL    string
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

// NewClient creates a new OpenRouter client.
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

// Generate implements ai.Provider. It always routes to Chat Completions.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if ai.RequestsImage(req) {
		return nil, ai.NewProviderError(
			"openrouter", 0,
			"image generation not supported by openrouter (use a Gemini image model)",
			nil,
		)
	}

	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, openrouterTracer, "openrouter.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "openrouter"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.api_type", "chat"),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	resp, err := c.generateChat(ctx, req)
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
		attribute.Int("ai.tokens_cached", resp.CachedTokens),
		attribute.String("ai.cost_usd", resp.CostUSD),
		attribute.String("ai.response_preview", telemetry.Truncate(resp.Text, 100)),
	)

	return resp, nil
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "openrouter"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}
