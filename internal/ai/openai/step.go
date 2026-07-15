package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// aiHTTPLogPath resolves where to append the raw AI HTTP wire log, from
// AILANG_AI_HTTP_LOG, else the $HOME/.ailang/state/ai-http-log sentinel file
// (the sentinel path propagates to subprocesses where custom env vars do NOT —
// e.g. motoko's bun→ailang chain — which is why the env alone wasn't enough to
// see what motoko actually sends). Empty string = logging off.
func aiHTTPLogPath() string {
	if p := strings.TrimSpace(os.Getenv("AILANG_AI_HTTP_LOG")); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".ailang", "state", "ai-http-log"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// logAIWire appends the EXACT bytes sent and received for one chat-completions
// HTTP call. This is the authoritative "what did we actually put on the wire"
// record — captured at the http.Client boundary, after request serialization and
// before response parsing, so it reflects reality even when our struct-level dump
// or a network tap can't reach the process.
func logAIWire(url string, reqBody, respBody []byte) {
	path := aiHTTPLogPath()
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	for _, rec := range []map[string]any{
		{"kind": "http_request", "url": url, "body": string(reqBody)},
		{"kind": "http_response", "url": url, "body": string(respBody)},
	} {
		if b, mErr := json.Marshal(rec); mErr == nil {
			_, _ = f.Write(append(b, '\n'))
		}
	}
}

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). It translates req.Messages + req.Tools into
// OpenAI's Chat Completions tool-use shape and parses tool_calls out of the
// response into resp.ToolCalls.
//
// Wire shape contract:
//   - SystemPrompt is prepended as a {"role":"system"} message ONLY when
//     req.Messages does not already contain a system-role entry. If
//     req.Messages has a system message, req.Messages wins and SystemPrompt
//     is dropped.
//   - User messages with no ToolCallID emit {"role":"user","content":<text>}.
//   - User messages with a ToolCallID, and Role="tool" messages, emit
//     {"role":"tool","tool_call_id":<id>,"content":<text>}.
//   - Assistant messages with no ToolCalls emit {"role":"assistant","content":<text>}.
//   - Assistant messages with non-empty ToolCalls emit
//     {"role":"assistant","content":null,"tool_calls":[...]} when Content is
//     empty, or {"role":"assistant","content":<text>,"tool_calls":[...]} when
//     Content is non-empty. tool_calls[].function.arguments is passed through
//     verbatim — OpenAI emits it as a JSON STRING and round-trips it the same.
//
// Errors are returned as *ai.AIError exclusively. Non-2xx responses are
// classified via ai.ClassifyHTTPError; transport / context errors via
// ai.ClassifyError. The inner error.message field of an OpenAI error
// envelope, when present, is hoisted into the AIError.Message for clarity.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if req.Routing != nil && (req.Routing.HasRouting() || req.Routing.PriceCapSet()) {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported,
			"openai: AIRoutingPolicy not supported; use openrouter instead", false)
	}
	// M-AI-PROMPT-CACHING (v0.18.4): OpenAI auto-caches prompts >=1024 tokens
	// transparently. Explicit cache hints have no effect; warn once per session
	// so callers know hints were observed but ignored. The call proceeds normally.
	if len(req.CacheBreakpoints) > 0 {
		ai.WarnOnceCacheHintIgnored("openai", "auto_cache")
	}

	ctx, span := telemetry.StartSpan(ctx, openaiTracer, "openai.step",
		trace.WithAttributes(
			attribute.String("ai.provider", "openai"),
			attribute.String("ai.model", req.Model),
			attribute.Int("ai.tools_count", len(req.Tools)),
			attribute.Int("ai.messages_count", len(req.Messages)),
		),
	)
	defer span.End()

	apiReq, aiErr := BuildChatStepRequest(req)
	if aiErr != nil {
		recordStepError(span, aiErr)
		return nil, aiErr
	}

	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		e := ai.NewAIError(ai.CodeInternal, fmt.Sprintf("openai: failed to marshal request: %v", err), false)
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
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}
	defer func() { _ = resp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e := ai.ClassifyError(err)
		recordStepError(span, e)
		return nil, e
	}

	// Authoritative wire capture: exact request + response bytes (off unless the
	// AILANG_AI_HTTP_LOG env / ai-http-log sentinel is set).
	logAIWire(c.baseURL+"/chat/completions", jsonBody, body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := classifyChatHTTPError("openai", resp.StatusCode, body)
		recordStepError(span, e)
		return nil, e
	}

	out, parseErr := ParseChatStepResponse(body, req.Model)
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

// ============================================================================
// Wire types — exported so OpenRouter can reuse the same shape. The OpenAI
// Chat Completions tool-use schema is the canonical home for this format.
// ============================================================================

// ChatStepRequest is the on-the-wire request body for a Chat Completions
// tool-use call. Exported so OpenRouter can extend it with its provider-routing
// field by composition (see openrouter package).
//
// MaxTokens vs MaxCompletionTokens: OpenAI's GPT-5+ and o-series reasoning
// models (o1, o3) reject max_tokens with HTTP 400 "Unsupported parameter:
// 'max_tokens' is not supported with this model. Use 'max_completion_tokens'
// instead." BuildChatStepRequest routes the value to whichever field the
// model accepts (see usesMaxCompletionTokens). Only one of the two will be
// set on a given request — the omitempty tags keep the wire payload clean.
type ChatStepRequest struct {
	Model               string            `json:"model"`
	Messages            []ChatStepMessage `json:"messages"`
	Tools               []ChatStepToolDef `json:"tools,omitempty"`
	MaxTokens           int               `json:"max_tokens,omitempty"`
	MaxCompletionTokens int               `json:"max_completion_tokens,omitempty"`
	Temperature         float64           `json:"temperature,omitempty"`
	// Stream + StreamOptions are set by StreamStep (M-AI-STEP-STREAMING
	// v0.18.7). omitempty keeps the non-streaming Step request bit-for-bit
	// identical to pre-v0.18.7 wire bytes.
	Stream        bool              `json:"stream,omitempty"`
	StreamOptions *ChatStreamOption `json:"stream_options,omitempty"`
}

// ChatStreamOption controls per-stream behaviour. The only field we set is
// IncludeUsage which tells OpenAI to emit a final chunk carrying the usage
// block (otherwise streamed responses contain NO token counts at all).
type ChatStreamOption struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatStepMessage is one entry in the messages array. Content is a
// json.RawMessage so callers can emit a JSON string OR JSON null (for
// assistant messages with tool_calls and empty text).
type ChatStepMessage struct {
	Role       string             `json:"role"`
	Content    json.RawMessage    `json:"content"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	ToolCalls  []ChatStepToolCall `json:"tool_calls,omitempty"`
}

// ChatStepToolCall is one tool invocation in an assistant message. Note that
// Function.Arguments is a JSON STRING (not an object) — OpenAI requires this.
type ChatStepToolCall struct {
	ID       string                   `json:"id"`
	Type     string                   `json:"type"`
	Function ChatStepToolCallFunction `json:"function"`
}

// ChatStepToolCallFunction carries the tool name and the arguments STRING.
// json.RawMessage is used because the wire value MUST be a JSON string
// literal containing JSON — re-encoding as Go string would double-escape.
type ChatStepToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ChatStepToolDef is one entry in the tools array advertised to the model.
// Parameters is a decoded JSON Schema OBJECT (unlike Arguments which is a string).
type ChatStepToolDef struct {
	Type     string                  `json:"type"`
	Function ChatStepToolDefFunction `json:"function"`
}

// ChatStepToolDefFunction carries the tool's name, description, and the
// JSON-Schema parameters as a decoded object on the wire.
type ChatStepToolDefFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatStepResponse is the parsed response body for a Chat Completions call.
type ChatStepResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Model   string           `json:"model"`
	Choices []ChatStepChoice `json:"choices"`
	Usage   ChatStepUsage    `json:"usage"`
}

// ChatStepChoice is one completion choice. Message.Content is RawMessage so
// we can accept JSON null or a string transparently.
type ChatStepChoice struct {
	Index        int                 `json:"index"`
	Message      ChatStepRespMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

// ChatStepRespMessage is the assistant message inside a choice. Reasoning models
// over OpenAI-compat return their thinking in `reasoning` (Ollama) or
// `reasoning_content` (DashScope/DeepSeek) — both captured here so it is not
// silently dropped (Qwen-Agent#789).
type ChatStepRespMessage struct {
	Role             string             `json:"role"`
	Content          json.RawMessage    `json:"content"`
	ToolCalls        []ChatStepToolCall `json:"tool_calls,omitempty"`
	Reasoning        string             `json:"reasoning,omitempty"`
	ReasoningContent string             `json:"reasoning_content,omitempty"`
}

// ChatStepUsage is the token-usage block.
type ChatStepUsage struct {
	PromptTokens        int                       `json:"prompt_tokens"`
	CompletionTokens    int                       `json:"completion_tokens"`
	TotalTokens         int                       `json:"total_tokens"`
	PromptTokensDetails *ChatStepPromptTokDetails `json:"prompt_tokens_details,omitempty"`
}

// ChatStepPromptTokDetails carries OpenAI's prompt-cache breakdown
// (cached_tokens). Anthropic-style cache_creation_input_tokens has no
// equivalent on OpenAI — the cache write isn't surfaced separately.
type ChatStepPromptTokDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// ChatStepErrorEnvelope is the OpenAI/OpenRouter error response shape.
type ChatStepErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// ============================================================================
// Translation helpers — exported so OpenRouter can call them directly.
// ============================================================================

// BuildChatStepRequest converts an ai.Request into the on-the-wire
// ChatStepRequest body. Returns *ai.AIError on translation failure (e.g.
// malformed JSON in a tool's Parameters schema).
//
// The translation rules are documented on Client.Step.
func BuildChatStepRequest(req *ai.Request) (*ChatStepRequest, *ai.AIError) {
	out := &ChatStepRequest{
		Model: req.Model,
	}
	if req.MaxTokens > 0 {
		// GPT-5+ and o-series require max_completion_tokens; legacy models
		// require max_tokens. usesMaxCompletionTokens routes by model id.
		if usesMaxCompletionTokens(req.Model) {
			out.MaxCompletionTokens = req.MaxTokens
		} else {
			out.MaxTokens = req.MaxTokens
		}
	}
	if req.Temperature > 0 {
		out.Temperature = req.Temperature
	}

	// SystemPrompt → system message, but ONLY if req.Messages doesn't already
	// have a system-role entry. req.Messages wins.
	hasSystemInMessages := false
	for _, m := range req.Messages {
		if m.Role == "system" {
			hasSystemInMessages = true
			break
		}
	}
	if req.SystemPrompt != "" && !hasSystemInMessages {
		raw, err := json.Marshal(req.SystemPrompt)
		if err != nil {
			return nil, ai.NewAIError(ai.CodeInternal,
				fmt.Sprintf("openai: failed to marshal SystemPrompt: %v", err), false)
		}
		out.Messages = append(out.Messages, ChatStepMessage{
			Role:    "system",
			Content: raw,
		})
	}

	for i, m := range req.Messages {
		switch m.Role {
		case "system":
			raw, err := json.Marshal(m.Content)
			if err != nil {
				return nil, ai.NewAIError(ai.CodeInternal,
					fmt.Sprintf("openai: failed to marshal system content for messages[%d]: %v", i, err), false)
			}
			out.Messages = append(out.Messages, ChatStepMessage{
				Role:    "system",
				Content: raw,
			})

		case "user":
			if m.ToolCallID != "" {
				// Edge case: user-with-ToolCallID is treated as a tool result
				// (same shape as Role="tool").
				raw, err := json.Marshal(m.Content)
				if err != nil {
					return nil, ai.NewAIError(ai.CodeInternal,
						fmt.Sprintf("openai: failed to marshal user-as-tool content for messages[%d]: %v", i, err), false)
				}
				out.Messages = append(out.Messages, ChatStepMessage{
					Role:       "tool",
					ToolCallID: m.ToolCallID,
					Content:    raw,
				})
				continue
			}
			raw, err := json.Marshal(m.Content)
			if err != nil {
				return nil, ai.NewAIError(ai.CodeInternal,
					fmt.Sprintf("openai: failed to marshal user content for messages[%d]: %v", i, err), false)
			}
			out.Messages = append(out.Messages, ChatStepMessage{
				Role:    "user",
				Content: raw,
			})

		case "tool":
			raw, err := json.Marshal(m.Content)
			if err != nil {
				return nil, ai.NewAIError(ai.CodeInternal,
					fmt.Sprintf("openai: failed to marshal tool content for messages[%d]: %v", i, err), false)
			}
			out.Messages = append(out.Messages, ChatStepMessage{
				Role:       "tool",
				ToolCallID: m.ToolCallID,
				Content:    raw,
			})

		case "assistant":
			msg := ChatStepMessage{Role: "assistant"}
			if len(m.ToolCalls) == 0 {
				raw, err := json.Marshal(m.Content)
				if err != nil {
					return nil, ai.NewAIError(ai.CodeInternal,
						fmt.Sprintf("openai: failed to marshal assistant content for messages[%d]: %v", i, err), false)
				}
				msg.Content = raw
			} else {
				// content is null when no text accompanies tool_calls,
				// otherwise the text string.
				if m.Content == "" {
					msg.Content = json.RawMessage("null")
				} else {
					raw, err := json.Marshal(m.Content)
					if err != nil {
						return nil, ai.NewAIError(ai.CodeInternal,
							fmt.Sprintf("openai: failed to marshal assistant content for messages[%d]: %v", i, err), false)
					}
					msg.Content = raw
				}
				for j, tc := range m.ToolCalls {
					// Arguments is a JSON STRING containing JSON; re-encode as
					// a JSON string literal on the wire. Use json.Marshal of a
					// Go string to handle quoting/escaping correctly.
					argsValue := tc.Arguments
					if argsValue == "" {
						argsValue = "{}"
					}
					argsRaw, err := json.Marshal(argsValue)
					if err != nil {
						return nil, ai.NewAIError(ai.CodeInternal,
							fmt.Sprintf("openai: failed to encode tool_call[%d].Arguments for messages[%d]: %v", j, i, err), false)
					}
					msg.ToolCalls = append(msg.ToolCalls, ChatStepToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: ChatStepToolCallFunction{
							Name:      tc.Name,
							Arguments: argsRaw,
						},
					})
				}
			}
			out.Messages = append(out.Messages, msg)

		default:
			return nil, ai.NewAIError(ai.CodeInternal,
				fmt.Sprintf("openai: unknown message role %q at messages[%d]", m.Role, i), false)
		}
	}

	for i, ts := range req.Tools {
		var schema json.RawMessage
		if ts.Parameters == "" {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		} else {
			var probe any
			if err := json.Unmarshal([]byte(ts.Parameters), &probe); err != nil {
				return nil, ai.NewAIError(ai.CodeInternal,
					fmt.Sprintf("openai: tools[%d].Parameters is not valid JSON: %v", i, err), false)
			}
			schema = json.RawMessage(ts.Parameters)
		}
		out.Tools = append(out.Tools, ChatStepToolDef{
			Type: "function",
			Function: ChatStepToolDefFunction{
				Name:        ts.Name,
				Description: ts.Description,
				Parameters:  schema,
			},
		})
	}

	return out, nil
}

// ParseChatStepResponse decodes a successful Chat Completions response body
// into an ai.Response. requestedModel is used as a fallback when the response
// model field is empty (rare but defensive).
func ParseChatStepResponse(body []byte, requestedModel string) (*ai.Response, *ai.AIError) {
	var raw ChatStepResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, ai.NewAIError(ai.CodeProtocolError,
			fmt.Sprintf("openai: failed to parse response: %v", err), false)
	}
	if len(raw.Choices) == 0 {
		return nil, ai.NewAIError(ai.CodeProtocolError,
			"openai: response contained no choices", false)
	}

	choice := raw.Choices[0]

	// Decode message.content — may be JSON null (assistant w/ tool calls) or a string.
	text := ""
	if len(choice.Message.Content) > 0 && string(choice.Message.Content) != "null" {
		if err := json.Unmarshal(choice.Message.Content, &text); err != nil {
			// content might be a non-string (e.g. content blocks array — not
			// expected but defensive). Fall back to raw bytes as text.
			text = string(choice.Message.Content)
		}
	}

	// Reasoning models (Qwen3/DeepSeek over OpenAI-compat) return their thinking in
	// `reasoning` (Ollama) or `reasoning_content` (DashScope/DeepSeek). Capture it —
	// dropping it silently loses content (Qwen-Agent#789).
	reasoning := choice.Message.ReasoningContent
	if reasoning == "" {
		reasoning = choice.Message.Reasoning
	}

	var toolCalls []ai.ToolCall
	for _, tc := range choice.Message.ToolCalls {
		// Pass the arguments string through verbatim. Function.Arguments is a
		// json.RawMessage of a JSON string literal; decode the string.
		var argsStr string
		if len(tc.Function.Arguments) > 0 {
			if err := json.Unmarshal(tc.Function.Arguments, &argsStr); err != nil {
				// Defensive: treat as raw bytes if not a JSON string literal.
				argsStr = string(tc.Function.Arguments)
			}
		}
		toolCalls = append(toolCalls, ai.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: argsStr,
		})
	}

	// Recovery: when the model emitted NO native tool_calls but wrote a Hermes-style
	// <tool_call>{...}</tool_call> block inside its text or reasoning (which Ollama's
	// /v1 sometimes fails to lift into the tool_calls field for Qwen3 thinking
	// models), parse it out so the agent loop sees the action instead of finalizing
	// the turn as a no-op (the "0 tool calls / disengagement" failure mode).
	finish := MapChatFinishReason(choice.FinishReason)
	if len(toolCalls) == 0 {
		if recovered := extractHermesToolCalls(text + "\n" + reasoning); len(recovered) > 0 {
			toolCalls = recovered
			finish = "tool_calls"
		}
	}

	model := raw.Model
	if model == "" {
		model = requestedModel
	}

	cacheRead := 0
	if raw.Usage.PromptTokensDetails != nil {
		cacheRead = raw.Usage.PromptTokensDetails.CachedTokens
	}
	return &ai.Response{
		Text:                 text,
		Reasoning:            reasoning,
		InputTokens:          raw.Usage.PromptTokens,
		OutputTokens:         raw.Usage.CompletionTokens,
		TotalTokens:          raw.Usage.TotalTokens,
		CacheReadInputTokens: cacheRead,
		Model:                model,
		ToolCalls:            toolCalls,
		FinishReason:         finish,
	}, nil
}

// hermesToolCallRe matches Hermes-style tool calls, e.g.
//
//	<tool_call>{"name":"WriteFile","arguments":{"path":"x.ail","content":"..."}}</tool_call>
//
// Non-greedy on the inner group so it stops at the FIRST </tool_call> (the inner
// JSON may itself contain braces, so we rely on the closing tag, not brace
// balancing).
var hermesToolCallRe = regexp.MustCompile(`(?s)<tool_call>\s*(.*?)\s*</tool_call>`)

// extractHermesToolCalls recovers tool calls a model emitted as Hermes-style
// <tool_call>{...}</tool_call> blocks inside its text/reasoning, which Ollama's
// /v1 sometimes fails to lift into the native tool_calls field (Qwen3 thinking
// models). Returns nil when there are none or all blocks fail to parse.
func extractHermesToolCalls(s string) []ai.ToolCall {
	if !strings.Contains(s, "<tool_call>") {
		return nil
	}
	var out []ai.ToolCall
	for i, m := range hermesToolCallRe.FindAllStringSubmatch(s, -1) {
		var parsed struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(m[1])), &parsed); err != nil || parsed.Name == "" {
			continue
		}
		args := strings.TrimSpace(string(parsed.Arguments))
		if args == "" || args == "null" {
			args = "{}"
		}
		out = append(out, ai.ToolCall{
			ID:        fmt.Sprintf("hermes_%d", i),
			Name:      parsed.Name,
			Arguments: args,
		})
	}
	return out
}

// MapChatFinishReason converts the OpenAI Chat Completions finish_reason
// vocabulary into the normalized ai.Response.FinishReason values.
//
// Mapping:
//
//	"stop"           → "stop"
//	"tool_calls"     → "tool_calls"
//	"function_call"  → "tool_calls" (legacy single-function alias)
//	"length"         → "length"
//	"content_filter" → "error"
//	anything else    → "error"
func MapChatFinishReason(r string) string {
	switch r {
	case "stop":
		return "stop"
	case "tool_calls":
		return "tool_calls"
	case "function_call":
		return "tool_calls"
	case "length":
		return "length"
	case "content_filter":
		return "error"
	}
	return "error"
}

// classifyChatHTTPError wraps ai.ClassifyHTTPError with OpenAI-style envelope
// parsing so AIError.Message carries the human-readable error.message field
// rather than the raw JSON envelope. Exported via classifyChatHTTPErrorFor for
// the openrouter adapter to share.
func classifyChatHTTPError(provider string, statusCode int, body []byte) *ai.AIError {
	return ClassifyChatHTTPErrorFor(provider, statusCode, body)
}

// ClassifyChatHTTPErrorFor classifies an OpenAI-format error envelope. Both
// OpenAI and OpenRouter speak this shape — provider distinguishes which name
// is reported in the AIError.Message when the body is empty.
func ClassifyChatHTTPErrorFor(provider string, statusCode int, body []byte) *ai.AIError {
	msg := string(body)
	var env ChatStepErrorEnvelope
	if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
		msg = env.Error.Message
	}
	// Trim noise but preserve case for downstream substring matching.
	msg = strings.TrimSpace(msg)
	return ai.ClassifyHTTPError(provider, statusCode, msg)
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
