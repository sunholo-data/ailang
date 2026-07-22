package anthropic

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
// but sets stream:true and parses the resulting Anthropic SSE event stream
// into a typed StepResult — identical Response shape to Step on success.
//
// Per-chunk callback semantics:
//   - ai.StreamContentDelta is fired once per content_block_delta event of
//     type=text_delta. The callback is the user's incremental-render hook.
//   - ai.StreamUsage is fired exactly once at message_delta time, carrying
//     the final input_tokens / output_tokens / cache_*_input_tokens.
//   - tool_use input deltas (input_json_delta) are accumulated into a
//     per-block JSON buffer; the assembled block is appended to the final
//     Response.ToolCalls when content_block_stop fires for that block. Tool
//     deltas are NOT streamed individually in Phase 1 — that's deferred to
//     Phase 2 (ToolCallDelta variant).
//
// Implements ai.StreamingProvider so ai.Handler.StepWithStream type-asserts
// successfully and dispatches natively rather than NO-OP-falling-back.
func (c *Client) StreamStep(ctx context.Context, req *ai.Request, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	ctx, span := telemetry.StartSpan(ctx, anthropicTracer, "anthropic.streamStep",
		trace.WithAttributes(
			attribute.String("ai.provider", "anthropic"),
			attribute.String("ai.model", req.Model),
			attribute.Int("ai.tools_count", len(req.Tools)),
			attribute.Int("ai.messages_count", len(req.Messages)),
			attribute.Bool("ai.streaming", true),
		),
	)
	defer span.End()

	// M-AI-REASONING-EFFORT: resolve reasoning controls BEFORE building/marshaling.
	reasoning, rErr := ai.ResolveReasoning(req, "anthropic", req.Model)
	if rErr != nil {
		recordSpanError(span, asAIError(rErr))
		return nil, rErr
	}

	apiReq, aiErr := buildStepRequest(req, reasoning)
	if aiErr != nil {
		recordSpanError(span, aiErr)
		return nil, aiErr
	}
	apiReq.Stream = true

	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		e := ai.NewAIError(ai.CodeInternal, fmt.Sprintf("anthropic: failed to marshal stream request: %v", err), false)
		recordSpanError(span, e)
		return nil, e
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		e := ai.ClassifyError(err)
		recordSpanError(span, e)
		return nil, e
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", c.apiVersion)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		e := ai.ClassifyError(err)
		recordSpanError(span, e)
		return nil, e
	}
	defer func() { _ = httpResp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", httpResp.StatusCode))

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		body, _ := io.ReadAll(httpResp.Body)
		bodyStr := string(body)
		var errEnv errorResponse
		if json.Unmarshal(body, &errEnv) == nil && errEnv.Error.Message != "" {
			bodyStr = errEnv.Error.Message
		}
		e := ai.ClassifyHTTPError("anthropic", httpResp.StatusCode, bodyStr)
		recordSpanError(span, e)
		return nil, e
	}

	out, parseErr := parseAnthropicSSEStream(httpResp.Body, onChunk)
	if parseErr != nil {
		recordSpanError(span, parseErr)
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

// streamEvent is the line-prefix shape of a single Anthropic SSE event.
// Anthropic emits one event in two consecutive lines:
//
//	event: <eventType>
//	data: {<json>}
//	(blank line)
//
// We reassemble these by tracking the most recent "event:" line until we
// hit a "data:" line, then dispatch on (eventType, dataJSON).
//
//nolint:unused // SSE event shape retained alongside the streaming parser
type sseEvent struct {
	eventType string
	data      string
}

// streamMessageStart is the JSON payload of an Anthropic "message_start"
// event. We extract the initial usage block (which carries input_tokens
// + cache_*_input_tokens) here so the final Response has cache metrics
// even though message_delta only refreshes output_tokens.
type streamMessageStart struct {
	Message struct {
		ID    string         `json:"id"`
		Model string         `json:"model"`
		Role  string         `json:"role"`
		Usage anthropicUsage `json:"usage"`
	} `json:"message"`
}

// streamContentBlockStart is the JSON payload of an Anthropic
// "content_block_start" event. For tool_use blocks we capture the ID + name
// here so subsequent input_json_delta events can be associated by index.
type streamContentBlockStart struct {
	Index        int `json:"index"`
	ContentBlock struct {
		Type  string          `json:"type"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
		Text  string          `json:"text,omitempty"`
	} `json:"content_block"`
}

// streamContentBlockDelta is the JSON payload of an Anthropic
// "content_block_delta" event. The discriminator is delta.type:
//   - "text_delta"        → delta.text accumulates assistant content
//   - "input_json_delta"  → delta.partial_json accumulates tool_use args
//   - "thinking_delta"    → delta.thinking accumulates extended-thinking
//     content (v0.18.8, claude-opus-4.5+ with
//     thinking enabled). Surfaces as
//     ai.StreamThinkingDelta to the user callback.
type streamContentBlockDelta struct {
	Index int `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
		Thinking    string `json:"thinking,omitempty"`
	} `json:"delta"`
}

// streamMessageDelta is the JSON payload of an Anthropic "message_delta"
// event. It carries the FINAL stop_reason + the FINAL output_tokens
// (Anthropic streams output token count only at end-of-message, unlike
// OpenAI which streams it per chunk).
type streamMessageDelta struct {
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage anthropicUsage `json:"usage"`
}

// blockState tracks the per-content-block accumulator while parsing the
// SSE stream. text blocks accumulate Text; tool_use blocks accumulate
// inputJSON until content_block_stop emits the assembled ToolCall.
type blockState struct {
	blockType string
	id        string
	name      string
	text      strings.Builder
	inputJSON strings.Builder
}

// parseAnthropicSSEStream consumes the SSE event stream from an Anthropic
// streaming Messages API response and produces a typed *ai.Response. The
// onChunk callback fires per text delta and once at message_delta with the
// final usage block.
//
// Errors are wrapped as *ai.AIError so callers don't need to translate.
// Bad JSON inside events maps to CodeProtocolError; the stream is aborted.
func parseAnthropicSSEStream(body io.Reader, onChunk func(ai.StreamChunk)) (*ai.Response, *ai.AIError) {
	scanner := bufio.NewScanner(body)
	// Anthropic events can carry large payloads (tool_use input objects,
	// long text deltas). Default 64KB scanner buffer is too small —
	// matches std/ai/streaming.openaiCompatStream's choice for the same
	// reason. 1MB keeps memory bounded but accommodates realistic deltas.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	out := &ai.Response{}
	blocks := make(map[int]*blockState) // index → accumulator
	var pendingEvent string

	emitToolCall := func(b *blockState) {
		if b == nil || b.blockType != "tool_use" {
			return
		}
		args := b.inputJSON.String()
		if args == "" {
			args = "{}"
		}
		out.ToolCalls = append(out.ToolCalls, ai.ToolCall{
			ID:        b.id,
			Name:      b.name,
			Arguments: args,
		})
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			pendingEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if pendingEvent == "" || data == "" {
				continue
			}
			if err := dispatchAnthropicSSEEvent(pendingEvent, data, out, blocks, onChunk); err != nil {
				return nil, err
			}
			pendingEvent = ""
		case line == "":
			pendingEvent = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, ai.ClassifyError(err)
	}

	// Finalize any blocks that didn't see a content_block_stop (defensive —
	// Anthropic always emits it, but we don't want to drop tool_use data
	// if a stream is truncated mid-flight).
	for _, b := range blocks {
		emitToolCall(b)
	}

	out.Text = strings.Join(collectBlockTexts(blocks), "")

	return out, nil
}

// dispatchAnthropicSSEEvent routes one parsed (eventType, dataJSON) pair to
// the matching accumulator + fires onChunk if appropriate. Extracted from
// parseAnthropicSSEStream so the parser stays small.
func dispatchAnthropicSSEEvent(
	eventType, data string,
	out *ai.Response,
	blocks map[int]*blockState,
	onChunk func(ai.StreamChunk),
) *ai.AIError {
	switch eventType {
	case "message_start":
		var ev streamMessageStart
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("anthropic: failed to parse message_start: %v", err), false)
		}
		out.Model = ev.Message.Model
		out.InputTokens = ev.Message.Usage.InputTokens
		out.CacheReadInputTokens = ev.Message.Usage.CacheReadInputTokens
		out.CacheCreationInputTokens = ev.Message.Usage.CacheCreationInputTokens

	case "content_block_start":
		var ev streamContentBlockStart
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("anthropic: failed to parse content_block_start: %v", err), false)
		}
		b := &blockState{
			blockType: ev.ContentBlock.Type,
			id:        ev.ContentBlock.ID,
			name:      ev.ContentBlock.Name,
		}
		// content_block_start may carry partial seed text (rare, but
		// some Anthropic responses prefill the block).
		if ev.ContentBlock.Text != "" {
			b.text.WriteString(ev.ContentBlock.Text)
			if onChunk != nil {
				onChunk(ai.StreamContentDelta{Text: ev.ContentBlock.Text})
			}
		}
		blocks[ev.Index] = b

	case "content_block_delta":
		var ev streamContentBlockDelta
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("anthropic: failed to parse content_block_delta: %v", err), false)
		}
		b, ok := blocks[ev.Index]
		if !ok {
			// Defensive: unknown block index — initialize a generic text
			// block rather than dropping the data.
			b = &blockState{blockType: "text"}
			blocks[ev.Index] = b
		}
		switch ev.Delta.Type {
		case "text_delta":
			b.text.WriteString(ev.Delta.Text)
			if onChunk != nil && ev.Delta.Text != "" {
				onChunk(ai.StreamContentDelta{Text: ev.Delta.Text})
			}
		case "input_json_delta":
			b.inputJSON.WriteString(ev.Delta.PartialJSON)
		case "thinking_delta":
			// Extended-thinking chunk (v0.18.8). Reasoning is NOT
			// accumulated into the block's text builder — that gets
			// concatenated into Response.Text, which by API contract
			// excludes reasoning. Reasoning flows ONLY through the
			// callback so consumers can choose to render or drop.
			if onChunk != nil && ev.Delta.Thinking != "" {
				onChunk(ai.StreamThinkingDelta{Text: ev.Delta.Thinking})
			}
		}

	case "content_block_stop":
		var ev struct {
			Index int `json:"index"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("anthropic: failed to parse content_block_stop: %v", err), false)
		}
		b, ok := blocks[ev.Index]
		if !ok {
			return nil
		}
		if b.blockType == "tool_use" {
			args := b.inputJSON.String()
			if args == "" {
				args = "{}"
			}
			out.ToolCalls = append(out.ToolCalls, ai.ToolCall{
				ID:        b.id,
				Name:      b.name,
				Arguments: args,
			})
		}
		// Mark block as consumed by deleting from the live map so the
		// finalize pass in parseAnthropicSSEStream doesn't double-emit.
		delete(blocks, ev.Index)
		// Re-insert as a text-only carcass so collectBlockTexts can still
		// pick up the accumulated text in stream order.
		if b.blockType == "text" {
			blocks[ev.Index] = b
		}

	case "message_delta":
		var ev streamMessageDelta
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("anthropic: failed to parse message_delta: %v", err), false)
		}
		out.OutputTokens = ev.Usage.OutputTokens
		out.TotalTokens = out.InputTokens + out.OutputTokens
		out.FinishReason = mapStopReason(ev.Delta.StopReason)
		if onChunk != nil {
			onChunk(ai.StreamUsage{
				InputTokens:              out.InputTokens,
				OutputTokens:             out.OutputTokens,
				CacheReadInputTokens:     out.CacheReadInputTokens,
				CacheCreationInputTokens: out.CacheCreationInputTokens,
			})
		}

	case "message_stop":
		// No payload to parse beyond what message_delta already carried.

	case "ping", "":
		// Anthropic emits periodic ping events to keep the connection alive.
		// Nothing to do; just don't surface them as a protocol error.
	}

	return nil
}

// collectBlockTexts returns the assembled text from all text-typed blocks
// in stream-emitted order. Blocks are stored in a map by index, so we sort
// by key — Anthropic always emits indices monotonically, but we don't
// rely on map iteration order for the final assembled string.
func collectBlockTexts(blocks map[int]*blockState) []string {
	if len(blocks) == 0 {
		return nil
	}
	indices := make([]int, 0, len(blocks))
	for idx := range blocks {
		indices = append(indices, idx)
	}
	// Insertion sort (n is tiny — typically 1-3 blocks per response).
	for i := 1; i < len(indices); i++ {
		for j := i; j > 0 && indices[j-1] > indices[j]; j-- {
			indices[j-1], indices[j] = indices[j], indices[j-1]
		}
	}
	parts := make([]string, 0, len(indices))
	for _, idx := range indices {
		b := blocks[idx]
		if b.blockType == "text" {
			parts = append(parts, b.text.String())
		}
	}
	return parts
}
