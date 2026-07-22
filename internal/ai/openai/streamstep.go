package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Compile-time guarantee: *Client satisfies ai.StreamingProvider so the
// type-assertion in ai.Handler.StepWithStream succeeds.
var _ ai.StreamingProvider = (*Client)(nil)

// StreamStep is the streaming variant of Step, introduced by
// M-AI-STEP-STREAMING (v0.18.7). It builds the same request body as Step
// but sets stream:true + stream_options.include_usage:true and parses the
// resulting OpenAI Chat Completions SSE event stream into a typed
// StepResult — identical Response shape to Step on success.
//
// Per-chunk callback semantics:
//   - ai.StreamContentDelta is fired once per non-empty content delta
//     (choices[0].delta.content). The callback is the user's incremental-
//     render hook.
//   - ai.StreamUsage is fired once at the final chunk (which contains the
//     usage block per stream_options.include_usage:true).
//   - tool_calls deltas are accumulated into a per-index buffer; the
//     assembled tool_calls become Response.ToolCalls. Tool deltas are NOT
//     streamed individually in Phase 1 — that's deferred to Phase 2
//     (ToolCallDelta variant).
//
// Implements ai.StreamingProvider so ai.Handler.StepWithStream type-asserts
// successfully and dispatches natively rather than NO-OP-falling-back.
func (c *Client) StreamStep(ctx context.Context, req *ai.Request, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	if req.Routing != nil && (req.Routing.HasRouting() || req.Routing.PriceCapSet()) {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported,
			"openai: AIRoutingPolicy not supported; use openrouter instead", false)
	}
	if len(req.CacheBreakpoints) > 0 {
		ai.WarnOnceCacheHintIgnored("openai", "auto_cache")
	}

	ctx, span := telemetry.StartSpan(ctx, openaiTracer, "openai.streamStep",
		trace.WithAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", req.Model),
			attribute.Int("ai.tools_count", len(req.Tools)),
			attribute.Int("ai.messages_count", len(req.Messages)),
			attribute.Bool("ai.streaming", true),
		),
	)
	defer span.End()

	// M-AI-REASONING-EFFORT: resolve reasoning controls BEFORE building/marshaling.
	reasoning, rErr := ai.ResolveReasoning(req, "openai", req.Model)
	if rErr != nil {
		recordStepError(span, asAIError(rErr))
		return nil, rErr
	}

	apiReq, aiErr := BuildChatStepRequest(req, reasoning)
	if aiErr != nil {
		recordStepError(span, aiErr)
		return nil, aiErr
	}
	apiReq.Stream = true
	apiReq.StreamOptions = &ChatStreamOption{IncludeUsage: true}

	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		e := ai.NewAIError(ai.CodeInternal, fmt.Sprintf("openai: failed to marshal stream request: %v", err), false)
		recordStepError(span, e)
		return nil, e
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}
	defer func() { _ = httpResp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", httpResp.StatusCode))

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		e := ClassifyChatHTTPErrorFor("openai", httpResp.StatusCode, body)
		recordStepError(span, e)
		return nil, e
	}

	out, parseErr := ParseChatStepSSEStream(httpResp.Body, req.Model, onChunk)
	if parseErr != nil {
		recordStepError(span, parseErr)
		return nil, parseErr
	}

	span.SetAttributes(
		attribute.Int("ai.tokens_in", out.InputTokens),
		attribute.Int("ai.tokens_out", out.OutputTokens),
		attribute.Int("ai.tool_calls", len(out.ToolCalls)),
		attribute.String("ai.finish_reason", out.FinishReason),
	)

	return out, nil
}

// ChatStepStreamChunk is one SSE chunk from a Chat Completions streaming
// response. Same envelope as ChatStepResponse but with `delta` instead of
// `message` inside each choice, and an OPTIONAL usage block that only
// appears in the final chunk (when stream_options.include_usage:true).
type ChatStepStreamChunk struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Model   string                 `json:"model"`
	Choices []ChatStepStreamChoice `json:"choices"`
	Usage   *ChatStepUsage         `json:"usage,omitempty"`
}

// ChatStepStreamChoice is one delta-bearing choice in a streaming chunk.
// Note that finish_reason is null until the second-to-last chunk.
type ChatStepStreamChoice struct {
	Index        int                 `json:"index"`
	Delta        ChatStepStreamDelta `json:"delta"`
	FinishReason string              `json:"finish_reason,omitempty"`
}

// ChatStepStreamDelta is the partial assistant message in a streaming
// chunk. Role is set on the FIRST chunk; Content accumulates per chunk;
// ToolCalls fragments accumulate per chunk per tool-call index.
//
// Reasoning fields (v0.18.8 + v0.18.9): per-chunk model-reasoning text.
// Surfaces as ai.StreamThinkingDelta to the user callback. Does NOT
// accumulate into Response.Text — the final assistant content comes
// only from delta.content. Two fields because providers don't agree:
//
//   - ReasoningContent (`reasoning_content`): OpenAI o1/o3 direct API.
//     Per the OpenAI Chat Completions reasoning-model spec (v0.18.8).
//   - Reasoning (`reasoning`): OpenRouter's normalized field. EVERY
//     OpenRouter-routed reasoning model surfaces here, regardless of
//     the underlying provider's native field name (DeepSeek-R1 via
//     Novita, Anthropic Claude via Bedrock, Qwen-thinking, etc.).
//     Verified by direct-curl probe of openrouter.ai/api/v1 against
//     deepseek/deepseek-r1 and anthropic/claude-opus-4.5 — both
//     emit through `delta.reasoning` (v0.18.9).
//
// At parse time we fire one ThinkingDelta per non-empty value across
// either field. In practice only one fires per chunk; the dual field
// support is just provider-shape defensive.
type ChatStepStreamDelta struct {
	Role             string                       `json:"role,omitempty"`
	Content          string                       `json:"content,omitempty"`
	ReasoningContent string                       `json:"reasoning_content,omitempty"`
	Reasoning        string                       `json:"reasoning,omitempty"`
	ToolCalls        []ChatStepStreamToolCallFrag `json:"tool_calls,omitempty"`
}

// ChatStepStreamToolCallFrag is one fragment of a tool_call in a streaming
// chunk. ID + Function.Name appear ONCE on the first fragment for that
// tool-call index; Function.Arguments accumulates per chunk as a JSON-string
// fragment.
type ChatStepStreamToolCallFrag struct {
	Index    int                                `json:"index"`
	ID       string                             `json:"id,omitempty"`
	Type     string                             `json:"type,omitempty"`
	Function ChatStepStreamToolCallFragFunction `json:"function,omitempty"`
}

// ChatStepStreamToolCallFragFunction carries the per-fragment name + args.
type ChatStepStreamToolCallFragFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// toolCallAccumulator accumulates per-chunk fragments of one tool_call into
// a single ai.ToolCall. ID + Name come from the first fragment; Arguments
// is the byte-concatenation of all per-chunk argument fragments.
type toolCallAccumulator struct {
	id        string
	name      string
	arguments strings.Builder
}

// ParseChatStepSSEStream consumes the SSE event stream from an OpenAI Chat
// Completions streaming response and produces a typed *ai.Response. The
// onChunk callback fires per content delta and once at end-of-stream with
// the final usage block.
//
// SSE wire format (no event-type lines, just data: <json>):
//
//	data: {"id":"...","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"}}]}
//	data: {"id":"...","object":"chat.completion.chunk","choices":[{"delta":{"content":"!"}}]}
//	data: {"id":"...","object":"chat.completion.chunk","choices":[{"delta":{},"finish_reason":"stop"}]}
//	data: {"id":"...","object":"chat.completion.chunk","usage":{...}}
//	data: [DONE]
//
// Errors are wrapped as *ai.AIError so callers don't need to translate.
func ParseChatStepSSEStream(body io.Reader, requestedModel string, onChunk func(ai.StreamChunk)) (*ai.Response, *ai.AIError) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	out := &ai.Response{Model: requestedModel}
	var contentParts []string
	toolCalls := make(map[int]*toolCallAccumulator)
	finishReason := ""

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk ChatStepStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("openai: failed to parse stream chunk: %v", err), false)
		}

		if chunk.Model != "" {
			out.Model = chunk.Model
		}

		// Final-chunk usage block (when stream_options.include_usage:true).
		if chunk.Usage != nil {
			out.InputTokens = chunk.Usage.PromptTokens
			out.OutputTokens = chunk.Usage.CompletionTokens
			out.TotalTokens = chunk.Usage.TotalTokens
			if chunk.Usage.PromptTokensDetails != nil {
				out.CacheReadInputTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			if onChunk != nil {
				onChunk(ai.StreamUsage{
					InputTokens:          out.InputTokens,
					OutputTokens:         out.OutputTokens,
					CacheReadInputTokens: out.CacheReadInputTokens,
					// OpenAI doesn't surface cache-creation as a separate count.
					CacheCreationInputTokens: 0,
				})
			}
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				contentParts = append(contentParts, choice.Delta.Content)
				if onChunk != nil {
					onChunk(ai.StreamContentDelta{Text: choice.Delta.Content})
				}
			}
			// Reasoning content fragment.
			//   v0.18.8: delta.reasoning_content (OpenAI o1/o3 direct).
			//   v0.18.9: delta.reasoning (OpenRouter-normalized — every
			//            OR-routed reasoning model emits via this field
			//            regardless of underlying provider, including
			//            DeepSeek-R1, Anthropic-via-OR, Qwen-thinking).
			// Both fire as ai.StreamThinkingDelta. NOT accumulated into
			// Response.Text — the final answer comes via delta.content.
			if choice.Delta.ReasoningContent != "" && onChunk != nil {
				onChunk(ai.StreamThinkingDelta{Text: choice.Delta.ReasoningContent})
			}
			if choice.Delta.Reasoning != "" && onChunk != nil {
				onChunk(ai.StreamThinkingDelta{Text: choice.Delta.Reasoning})
			}
			for _, tcFrag := range choice.Delta.ToolCalls {
				acc, ok := toolCalls[tcFrag.Index]
				if !ok {
					acc = &toolCallAccumulator{}
					toolCalls[tcFrag.Index] = acc
				}
				if tcFrag.ID != "" {
					acc.id = tcFrag.ID
				}
				if tcFrag.Function.Name != "" {
					acc.name = tcFrag.Function.Name
				}
				if tcFrag.Function.Arguments != "" {
					acc.arguments.WriteString(tcFrag.Function.Arguments)
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, ai.ClassifyError(err)
	}

	out.Text = strings.Join(contentParts, "")
	out.FinishReason = MapChatFinishReason(finishReason)

	if len(toolCalls) > 0 {
		// Sort tool-call accumulators by index so the final order matches
		// the order OpenAI emitted them on the wire. n is tiny (typically
		// 1-5 tool calls per response).
		indices := make([]int, 0, len(toolCalls))
		for idx := range toolCalls {
			indices = append(indices, idx)
		}
		for i := 1; i < len(indices); i++ {
			for j := i; j > 0 && indices[j-1] > indices[j]; j-- {
				indices[j-1], indices[j] = indices[j], indices[j-1]
			}
		}
		out.ToolCalls = make([]ai.ToolCall, 0, len(indices))
		for _, idx := range indices {
			acc := toolCalls[idx]
			args := acc.arguments.String()
			if args == "" {
				args = "{}"
			}
			out.ToolCalls = append(out.ToolCalls, ai.ToolCall{
				ID:        acc.id,
				Name:      acc.name,
				Arguments: args,
			})
		}
	}

	return out, nil
}
