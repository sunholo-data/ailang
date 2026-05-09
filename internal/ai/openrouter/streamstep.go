package openrouter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Compile-time guarantee: *Client satisfies ai.StreamingProvider so the
// type-assertion in ai.Handler.StepWithStream succeeds.
var _ ai.StreamingProvider = (*Client)(nil)

// StreamStep is the streaming variant of Step, introduced by
// M-AI-STEP-STREAMING (v0.18.7). OpenRouter speaks OpenAI Chat Completions
// SSE natively (regardless of the routed-to provider's native protocol —
// anthropic/* models still emit OpenAI-shape chunks via OpenRouter's
// translation layer), so we reuse openai.ParseChatStepSSEStream.
//
// Per-chunk callback semantics match openai.StreamStep — see that doc.
//
// Routing composition: when req.Routing is non-zero, translatePolicy emits
// a providerField that rides alongside Tools / Messages on the wire (same
// as Step). Routing + streaming + tool-loop work end-to-end against any
// model OpenRouter supports.
//
// Cache breakpoint dispatch follows the same prefix-routing rules as Step
// (anthropic/* gets cache_control stamped; openai/* warns-and-no-ops; etc.).
func (c *Client) StreamStep(ctx context.Context, req *ai.Request, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	ctx, span := telemetry.StartSpan(ctx, openrouterTracer, "openrouter.streamStep",
		trace.WithAttributes(
			attribute.String("ai.provider", "openrouter"),
			attribute.String("ai.model", req.Model),
			attribute.Int("ai.tools_count", len(req.Tools)),
			attribute.Int("ai.messages_count", len(req.Messages)),
			attribute.Bool("ai.has_routing", req.Routing != nil && req.Routing.HasRouting()),
			attribute.Bool("ai.streaming", true),
		),
	)
	defer span.End()

	chatReq, aiErr := openai.BuildChatStepRequest(req)
	if aiErr != nil {
		recordStepError(span, aiErr)
		return nil, aiErr
	}
	chatReq.Stream = true
	chatReq.StreamOptions = &openai.ChatStreamOption{IncludeUsage: true}

	// Apply cache hints per the routed-to-provider's contract — same path
	// as Step. anthropic/* gets cache_control on system; openai/* warns;
	// other prefixes silent no-op. Empty breakpoints = no-op.
	if cacheErr := applyCacheHintsForRoute(chatReq, req.Model, req.CacheBreakpoints); cacheErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to apply cache hints: %v", cacheErr), false)
		recordStepError(span, e)
		return nil, e
	}

	body, marshalErr := marshalStepBodyWithProvider(chatReq, translatePolicy(req.Routing))
	if marshalErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to marshal stream request: %v", marshalErr), false)
		recordStepError(span, e)
		return nil, e
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	setAttributionHeaders(httpReq)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}
	defer func() { _ = httpResp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", httpResp.StatusCode))

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(httpResp.Body)
		e := openai.ClassifyChatHTTPErrorFor("openrouter", httpResp.StatusCode, respBody)
		recordStepError(span, e)
		return nil, e
	}

	out, parseErr := openai.ParseChatStepSSEStream(httpResp.Body, req.Model, onChunk)
	if parseErr != nil {
		recordStepError(span, parseErr)
		return nil, parseErr
	}

	// OpenRouter-specific Response enrichment: surface RequestedModel for
	// routing diff visibility (matches Step).
	out.RequestedModel = req.Model

	span.SetAttributes(
		attribute.Int("ai.tokens_in", out.InputTokens),
		attribute.Int("ai.tokens_out", out.OutputTokens),
		attribute.Int("ai.tool_calls", len(out.ToolCalls)),
		attribute.String("ai.finish_reason", out.FinishReason),
	)

	return out, nil
}
