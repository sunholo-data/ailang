// Package ollama provides an Ollama API client implementing the ai.Provider interface.
// Ollama enables local model inference for cost-free, offline AI capabilities.
package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var ollamaTracer = telemetry.Tracer("ai.ollama")

const (
	defaultEndpoint = "http://localhost:11434"
)

// Client implements ai.Provider for local Ollama models.
type Client struct {
	client   *ollamaapi.Client
	endpoint string
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithEndpoint sets a custom Ollama endpoint.
func WithEndpoint(endpoint string) ClientOption {
	return func(c *Client) {
		c.endpoint = endpoint
	}
}

// NewClient creates a new Ollama client.
// It reads OLLAMA_HOST from environment, defaulting to http://localhost:11434.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		endpoint: defaultEndpoint,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Check environment variable
	if envEndpoint := os.Getenv("OLLAMA_HOST"); envEndpoint != "" {
		c.endpoint = envEndpoint
	}

	// Set OLLAMA_HOST for the client library
	os.Setenv("OLLAMA_HOST", c.endpoint)

	// Create Ollama client
	client, err := ollamaapi.ClientFromEnvironment()
	if err != nil {
		return nil, ai.NewProviderError("ollama", 0, "failed to create Ollama client", err)
	}

	c.client = client
	return c, nil
}

// CheckConnection verifies Ollama is running and accessible.
func (c *Client) CheckConnection(ctx context.Context) error {
	_, err := c.client.List(ctx)
	if err != nil {
		return ai.NewProviderError("ollama", 0, fmt.Sprintf(
			"Ollama not running at %s: %v\n\n"+
				"To start Ollama:\n"+
				"  1. Install: https://ollama.ai/download\n"+
				"  2. Start:   ollama serve\n"+
				"  3. Pull:    ollama pull codellama:7b",
			c.endpoint, err), nil)
	}
	return nil
}

// Generate implements ai.Provider.
// It uses Ollama's Generate API (/api/generate) rather than Chat (/api/chat):
// for a single-turn tool-less request they are semantically equivalent (the
// model template is applied either way, with System as the system override),
// but measured on the rig (ollama 0.33.1, gemma4:e4b, 2026-08-28) the same
// byte-identical schema-enforced request takes ~0.6s via /api/generate and
// 15-26s via /api/chat, with server-side runner time ~1-2s in both cases —
// the gap is in ollama's chat-endpoint scheduling. fb_2dbfd79dbe2d1d3c.
// The tool paths (Step, streaming Step) still use /api/chat because they need
// multi-turn messages and tool definitions, which Generate cannot express.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Bound the call so a stalled native /api/chat stream can't hang forever
	// (M-OLLAMA-NATIVE-TIMEOUT; mirrors Step). No-op when called via Step, which
	// already applied the deadline.
	ctx, cancel := ollamaCallContext(ctx)
	defer cancel()

	if req.Routing != nil && req.Routing.HasRouting() {
		return nil, ai.NewProviderError("ollama", 0,
			"this provider does not support AIRoutingPolicy; use openrouter instead",
			ai.ErrRoutingNotSupported)
	}
	if ai.RequestsImage(req) {
		return nil, ai.NewProviderError("ollama", 0, "image generation not supported by provider \"ollama\" (model: "+req.Model+")", nil)
	}
	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, ollamaTracer, "ollama.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "ollama"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.endpoint", c.endpoint),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	var response strings.Builder

	// Build options
	options := map[string]interface{}{
		"seed": int64(42), // Deterministic by default
	}
	// num_ctx is omitted by default so ollama sizes context from the model —
	// see resolveOllamaNumCtx (step.go) for why the old hardcoded 8192 sat below
	// the prompts we actually send.
	if n, ok := resolveOllamaNumCtx(); ok {
		options["num_ctx"] = n
	}

	// Set max output tokens if specified
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	} else {
		options["num_predict"] = 4096 // Default max output
	}

	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}

	// Build generate request (strip any "ollama:"/"ollama/" routing prefix —
	// Ollama rejects the prefixed form as an invalid model name).
	genReq := &ollamaapi.GenerateRequest{
		Model:   bareModel(req.Model),
		Prompt:  req.FullUserPrompt(),
		System:  req.SystemPrompt,
		Options: options,
	}

	// Add JSON format if structured output requested
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			genReq.Format = json.RawMessage(req.ResponseSchema)
		} else {
			genReq.Format = json.RawMessage(`"json"`)
		}
	}

	// Use Generate API for instruction following
	var tally tokenTally
	err := c.client.Generate(ctx, genReq, func(resp ollamaapi.GenerateResponse) error {
		response.WriteString(resp.Response)
		tally.observe(resp.Metrics)
		return nil
	})

	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, ai.NewProviderError("ollama", 0, err.Error(), err)
	}
	// Surface a swallowed ctx deadline/cancel (stalled native stream) explicitly —
	// see the matching guard in Step (M-OLLAMA-NATIVE-TIMEOUT).
	if ctxErr := ctx.Err(); ctxErr != nil {
		span.RecordError(ctxErr)
		span.SetStatus(codes.Error, ctxErr.Error())
		return nil, ai.NewProviderError("ollama", 0, ctxErr.Error(), ctxErr)
	}

	span.SetAttributes(
		attribute.String("ai.response_model", req.Model),
		attribute.String("ai.response_preview", telemetry.Truncate(response.String(), 100)),
		attribute.Int("ai.tokens_in", tally.in),
		attribute.Int("ai.tokens_out", tally.out),
	)

	out := &ai.Response{
		Text:  response.String(),
		Model: req.Model,
	}
	tally.apply(out)
	return out, nil
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "ollama"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}
