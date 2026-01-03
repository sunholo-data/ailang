// Package ollama provides an Ollama API client implementing the ai.Provider interface.
// Ollama enables local model inference for cost-free, offline AI capabilities.
package ollama

import (
	"context"
	"fmt"
	"os"
	"strings"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo/ailang/internal/ai"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var ollamaTracer = otel.Tracer("ai.ollama")

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
// It uses Ollama's Chat API for instruction following.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Start OTEL span
	ctx, span := ollamaTracer.Start(ctx, "ollama.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "ollama"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.endpoint", c.endpoint),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	var response strings.Builder

	// Build messages
	messages := []ollamaapi.Message{}

	if req.SystemPrompt != "" {
		messages = append(messages, ollamaapi.Message{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	messages = append(messages, ollamaapi.Message{
		Role:    "user",
		Content: req.UserPrompt,
	})

	// Build options
	options := map[string]interface{}{
		"seed":    int64(42), // Deterministic by default
		"num_ctx": 8192,      // Reasonable context window
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

	// Use Chat API for instruction following
	err := c.client.Chat(ctx, &ollamaapi.ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Options:  options,
	}, func(resp ollamaapi.ChatResponse) error {
		response.WriteString(resp.Message.Content)
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

	// Note: Ollama doesn't report tokens the same way, so we leave them at 0
	span.SetAttributes(
		attribute.String("ai.response_model", req.Model),
		attribute.String("ai.response_preview", telemetry.Truncate(response.String(), 100)),
	)

	return &ai.Response{
		Text:         response.String(),
		Model:        req.Model,
		InputTokens:  0, // Ollama doesn't report tokens the same way
		OutputTokens: 0,
		TotalTokens:  0,
	}, nil
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "ollama"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}
