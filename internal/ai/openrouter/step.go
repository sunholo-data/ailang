package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). OpenRouter speaks OpenAI Chat Completions
// natively, so we reuse the shared translation helpers from
// internal/ai/openai (BuildChatStepRequest, ParseChatStepResponse,
// MapChatFinishReason, ClassifyChatHTTPErrorFor) and compose them with
// OpenRouter's two extensions:
//   - top-level `provider` field (translated from req.Routing via translatePolicy)
//   - HTTP-Referer and X-Title attribution headers (lifted from env vars,
//     same as the existing Generate path)
//
// Errors are returned as *ai.AIError exclusively. Non-2xx responses are
// classified via ai.ClassifyHTTPError; transport / context errors via
// ai.ClassifyError. The inner error.message field of an OpenAI-format error
// envelope, when present, is hoisted into the AIError.Message for clarity.
//
// Routing composition: when req.Routing is non-zero, translatePolicy emits a
// providerField that rides alongside Tools / Messages on the wire. A routed
// model like "anthropic/claude-sonnet-4.5" with a tool catalog works
// end-to-end against the OpenRouter URL.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	ctx, span := telemetry.StartSpan(ctx, openrouterTracer, "openrouter.step",
		trace.WithAttributes(
			attribute.String("ai.provider", "openrouter"),
			attribute.String("ai.model", req.Model),
			attribute.Int("ai.tools_count", len(req.Tools)),
			attribute.Int("ai.messages_count", len(req.Messages)),
			attribute.Bool("ai.has_routing", req.Routing != nil && req.Routing.HasRouting()),
		),
	)
	defer span.End()

	// Build the OpenAI-format Chat Completions body via the shared helper.
	chatReq, aiErr := openai.BuildChatStepRequest(req)
	if aiErr != nil {
		recordStepError(span, aiErr)
		return nil, aiErr
	}

	// M-AI-PROMPT-CACHING (v0.18.4): apply cache_breakpoints per the routed-
	// to-provider's contract. anthropic/* gets cache_control stamped on the
	// system message; openai/* and google/* warn-and-no-op; unknown prefixes
	// silent no-op. Empty breakpoints = no-op (bit-for-bit identical wire bytes).
	if cacheErr := applyCacheHintsForRoute(chatReq, req.Model, req.CacheBreakpoints); cacheErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to apply cache hints: %v", cacheErr), false)
		recordStepError(span, e)
		return nil, e
	}

	// Wrap the shared body in an OpenRouter-extended envelope that adds the
	// optional `provider` field. We marshal the wrapped struct so the wire
	// JSON is exactly: { ...chat completions fields, provider?: {...} }.
	body, marshalErr := marshalStepBodyWithProvider(chatReq, translatePolicy(req.Routing))
	if marshalErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to marshal request: %v", marshalErr), false)
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
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	setAttributionHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}
	defer func() { _ = resp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := openai.ClassifyChatHTTPErrorFor("openrouter", resp.StatusCode, respBody)
		recordStepError(span, e)
		return nil, e
	}

	out, parseErr := openai.ParseChatStepResponse(respBody, req.Model)
	if parseErr != nil {
		recordStepError(span, parseErr)
		return nil, parseErr
	}

	// OpenRouter-specific Response enrichment: surface RequestedModel so
	// callers can see the diff when routing is engaged. (Generate sets this;
	// keeping Step consistent.)
	out.RequestedModel = req.Model

	span.SetAttributes(
		attribute.Int("ai.tokens_in", out.InputTokens),
		attribute.Int("ai.tokens_out", out.OutputTokens),
		attribute.Int("ai.tool_calls", len(out.ToolCalls)),
		attribute.String("ai.finish_reason", out.FinishReason),
	)

	return out, nil
}

// marshalStepBodyWithProvider serialises the OpenAI ChatStepRequest with
// optional OpenRouter-specific extensions appended at the top level.
//
// extraFields is a list of pre-marshalled `"key":<value>` fragments that
// get spliced into the JSON body before the closing `}`. Used to inject
// OpenRouter-specific fields like "provider":{...} (routing policy) and
// "include_reasoning":true (v0.18.9 — opt-in for reasoning chunks on
// OpenRouter-routed thinking models like deepseek-r1, anthropic-via-OR,
// qwen-thinking; without this flag OpenRouter drops reasoning silently).
//
// Splice approach (vs. decode-into-map round-trip): both cheaper and
// keeps the OpenAI helper completely decoupled from OpenRouter-specific
// extensions. Safe because json.Marshal on a Go struct is guaranteed to
// emit a single object — no leading/trailing whitespace, terminator is
// the final byte.
func marshalStepBodyWithExtras(chatReq *openai.ChatStepRequest, extraFields [][]byte) ([]byte, error) {
	chatBytes, err := json.Marshal(chatReq)
	if err != nil {
		return nil, err
	}
	if len(extraFields) == 0 {
		return chatBytes, nil
	}
	totalExtra := 0
	for _, f := range extraFields {
		totalExtra += len(f) + 1 // +1 for leading comma
	}
	out := make([]byte, 0, len(chatBytes)+totalExtra)
	out = append(out, chatBytes[:len(chatBytes)-1]...)
	for _, f := range extraFields {
		out = append(out, ',')
		out = append(out, f...)
	}
	out = append(out, '}')
	return out, nil
}

// marshalStepBodyWithProvider preserves the original signature for the
// non-streaming Step path (only injects a provider field). Wraps the
// new extras-list helper for backward compatibility.
func marshalStepBodyWithProvider(chatReq *openai.ChatStepRequest, provider *providerField) ([]byte, error) {
	if provider == nil {
		return marshalStepBodyWithExtras(chatReq, nil)
	}
	provBytes, err := json.Marshal(provider)
	if err != nil {
		return nil, err
	}
	field := append([]byte(`"provider":`), provBytes...)
	return marshalStepBodyWithExtras(chatReq, [][]byte{field})
}

// recordStepError annotates the span with the AIError's code/message and
// marks it as a failure.
func recordStepError(span trace.Span, e *ai.AIError) {
	if e == nil {
		return
	}
	span.SetAttributes(
		attribute.String("error.code", e.Code),
		attribute.String("error.message", telemetry.Truncate(e.Message, 200)),
		attribute.Bool("error.retryable", e.Retryable),
	)
	span.SetStatus(codes.Error, e.Code)
}
