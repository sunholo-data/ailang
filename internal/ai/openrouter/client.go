package openrouter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/strutil"
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
	for k, vs := range attributionHeaders(attr) {
		for _, v := range vs {
			r.Header.Set(k, v)
		}
	}
}

// attributionHeaders builds the OpenRouter attribution headers (see
// setAttributionHeaders) in http.Header form for ai.DoJSON callers.
func attributionHeaders(attr *ai.Attribution) http.Header {
	referer := defaultHTTPReferer
	title := defaultXTitle
	categories := defaultCategories

	envAttr := config.OpenRouterAttributionConfig() // Layer 2: env vars override defaults
	if v := envAttr.HTTPReferer; v != "" {
		referer = v
	}
	if v := envAttr.XTitle; v != "" {
		title = v
	}
	if v := envAttr.Categories; v != "" {
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

	headers := make(http.Header)
	headers.Set("HTTP-Referer", referer)
	headers.Set("X-OpenRouter-Title", title)
	headers.Set("X-Title", title)
	headers.Set("X-OpenRouter-Categories", categories)
	headers.Set("X-Session-Id", sessionID())
	return headers
}

// processSessionID is the x-session-id used when OPENROUTER_SESSION_ID is
// unset: one per process, which is one session for an agent run.
var processSessionID = sync.OnceValue(func() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ailang-%d", os.Getpid())
	}
	return "ailang-" + hex.EncodeToString(b)
})

// sessionID is the OpenRouter session-affinity id. Without one OpenRouter
// load-balances every call of a conversation across upstream hosts. Measured
// 2026-10-12 on GLM-5.3-Flash, same task, same time: pi (which sends
// x-session-id) stayed on one host for all 81 calls; motoko via std/ai was
// spread over three, and on one of them the model's tool calls stopped
// matching its own reasoning ({"op":"tree","module":"x"} for 25+ steps).
func sessionID() string {
	if v := config.OpenRouterSessionID(); v != "" {
		return v
	}
	return processSessionID()
}

// FrozenAttribution resolves environment/default precedence before a deferred
// handler is installed. Returning complete fields prevents late environment
// changes from changing attribution on the first or later requests.
func FrozenAttribution(attr *ai.Attribution) *ai.Attribution {
	h := attributionHeaders(attr)
	return &ai.Attribution{HTTPReferer: h.Get("HTTP-Referer"), Title: h.Get("X-OpenRouter-Title"), Categories: h.Get("X-OpenRouter-Categories")}
}

// requestHeaders is the bearer auth plus attribution every OpenRouter call carries.
func (c *Client) requestHeaders(attr *ai.Attribution) http.Header {
	h := attributionHeaders(attr)
	h.Set("Authorization", "Bearer "+c.apiKey)
	return h
}

// Client implements ai.Provider for OpenRouter's unified Chat Completions API.
//
// Unlike the OpenAI client there is no API-type detection — OpenRouter is
// Chat-Completions only. Image generation goes through the same endpoint with
// "modalities": ["image","text"] (Generate path only; Step/StreamStep reject
// image output loudly).
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
	// Image requests (ResponseModalities ["IMAGE"]) take the same chat path:
	// generateChat adds modalities/image_config and harvests message.images
	// (M-OPENROUTER-IMAGE-OUTPUT, #1500). A non-image model surfaces the
	// upstream error as a typed ProviderError — no client-side allowlist.

	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, openrouterTracer, "openrouter.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "openrouter"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.api_type", "chat"),
			attribute.String("ai.prompt_preview", strutil.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	// M-AI-REASONING-EFFORT: resolve reasoning controls BEFORE marshaling/dispatch.
	reasoning, rErr := resolveReasoning(req)
	if rErr != nil {
		span.RecordError(rErr)
		span.SetStatus(codes.Error, rErr.Error())
		return nil, rErr
	}

	resp, err := c.generateChat(ctx, req, reasoning)
	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", strutil.Truncate(err.Error(), 200)),
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
		attribute.String("ai.response_preview", strutil.Truncate(resp.Text, 100)),
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
