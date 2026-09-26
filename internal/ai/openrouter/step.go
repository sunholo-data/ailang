package openrouter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
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

	// M-AI-REASONING-EFFORT: resolve reasoning controls for OpenRouter BEFORE
	// building/marshaling. OpenRouter routes reasoning through its own
	// reasoning{} block (see reasoningExtra), NOT OpenAI's native
	// reasoning_effort field, so we pass ReasoningNone to the shared builder.
	reasoning, rErr := ai.ResolveReasoning(req, "openrouter", req.Model)
	if rErr != nil {
		ai.RecordSpanError(span, rErr)
		return nil, rErr
	}

	// Build the OpenAI-format Chat Completions body via the shared helper.
	chatReq, aiErr := openai.BuildChatStepRequest(req, ai.ReasoningDecision{})
	if aiErr != nil {
		ai.RecordSpanError(span, aiErr)
		return nil, aiErr
	}

	// M-AI-PROMPT-CACHING (v0.18.4): apply cache_breakpoints per the routed-
	// to-provider's contract. anthropic/* gets cache_control stamped on the
	// system message; openai/* and google/* warn-and-no-op; unknown prefixes
	// silent no-op. Empty breakpoints = no-op (bit-for-bit identical wire bytes).
	if cacheErr := applyCacheHintsForRoute(chatReq, req.Model, req.CacheBreakpoints); cacheErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to apply cache hints: %v", cacheErr), false)
		ai.RecordSpanError(span, e)
		return nil, e
	}

	// Wrap the shared body in an OpenRouter-extended envelope that adds the
	// optional `provider` field. We marshal the wrapped struct so the wire
	// JSON is exactly: { ...chat completions fields, provider?: {...} }.
	provider, rerr := translatePolicy(req.Routing)
	if rerr != nil {
		e := ai.NewAIError(ai.CodeSchemaValidation,
			fmt.Sprintf("openrouter: invalid routing policy: %v", rerr), false)
		ai.RecordSpanError(span, e)
		return nil, e
	}
	var extras [][]byte
	if provider != nil {
		provBytes, mErr := json.Marshal(provider)
		if mErr != nil {
			e := ai.NewAIError(ai.CodeInternal,
				fmt.Sprintf("openrouter: failed to marshal provider field: %v", mErr), false)
			ai.RecordSpanError(span, e)
			return nil, e
		}
		extras = append(extras, append([]byte(`"provider":`), provBytes...))
	}
	reasoningFrags, rfErr := reasoningExtras(reasoning)
	if rfErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to marshal reasoning field: %v", rfErr), false)
		ai.RecordSpanError(span, e)
		return nil, e
	}
	extras = append(extras, reasoningFrags...)

	// Broadcast correlation (M-OPENROUTER-BROADCAST-INGEST M3). Contributes no
	// fragments when the caller set no correlation, so the spliced body stays
	// byte-identical to before.
	corrFrags, cErr := correlationExtras(req.Correlation)
	if cErr != nil {
		e := ai.NewAIError(ai.CodeSchemaValidation,
			fmt.Sprintf("openrouter: invalid correlation: %v", cErr), false)
		ai.RecordSpanError(span, e)
		return nil, e
	}
	extras = append(extras, corrFrags...)

	body, marshalErr := marshalStepBodyWithExtras(chatReq, extras)
	if marshalErr != nil {
		e := ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("openrouter: failed to marshal request: %v", marshalErr), false)
		ai.RecordSpanError(span, e)
		return nil, e
	}

	res, err := ai.DoJSON(ctx, ai.JSONCall{
		Provider: "openrouter",
		Client:   c.httpClient,
		URL:      c.baseURL + "/chat/completions",
		Headers:  c.requestHeaders(req.Attribution),
		Body:     body,
	}, nil)
	if res != nil {
		span.SetAttributes(attribute.Int("http.status_code", res.StatusCode))
	}
	if err != nil {
		e := ai.ClassifyError(err)
		ai.RecordSpanError(span, e)
		return nil, e
	}

	out, parseErr := openai.ParseChatStepResponse(res.Body, req.Model)
	if parseErr != nil {
		ai.RecordSpanError(span, parseErr)
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

// reasoningExtras returns the OpenRouter-specific reasoning wire fragment(s)
// for a resolved reasoning decision, to be spliced into the top-level request
// body via marshalStepBodyWithExtras. Returns nil for ReasoningNone (no
// reasoning field emitted — byte-identical to pre-v0.31.0). Marshals the same
// reasoningField shape the Generate path uses.
func reasoningExtras(d ai.ReasoningDecision) ([][]byte, error) {
	var rf *reasoningField
	switch d.Kind {
	case ai.ReasoningEffortKind:
		rf = &reasoningField{Effort: d.Effort}
	case ai.ReasoningMaxTokensKind:
		rf = &reasoningField{MaxTokens: d.MaxTokensReasoning}
	default:
		return nil, nil
	}
	b, err := json.Marshal(rf)
	if err != nil {
		return nil, err
	}
	return [][]byte{append([]byte(`"reasoning":`), b...)}, nil
}
