package anthropic

import (
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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). It translates req.Messages + req.Tools into
// Anthropic's Messages API tool-use shape and parses tool_use content blocks
// out of the response into resp.ToolCalls.
//
// Wire shape contract:
//   - SystemPrompt is sent as the top-level "system" field; system-role
//     entries in req.Messages are dropped.
//   - User messages with no ToolCallID emit a plain string content.
//   - User messages with a ToolCallID, and Role="tool" messages, emit a
//     content array containing a single tool_result block.
//   - Assistant messages with no ToolCalls emit a plain string content.
//   - Assistant messages with ToolCalls emit a content array combining an
//     optional text block (when Content is non-empty) and one tool_use block
//     per ToolCall. Tool inputs are decoded from the JSON-string Arguments
//     and re-marshaled as a JSON object on the wire (Anthropic requires
//     "input" to be a JSON object, not a string).
//
// Errors are returned as *ai.AIError exclusively. Non-2xx responses are
// classified via ai.ClassifyHTTPError; transport / context errors via
// ai.ClassifyError. The inner error.message field of an Anthropic error
// envelope, when present, is hoisted into the AIError.Message for clarity.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	ctx, span := telemetry.StartSpan(ctx, anthropicTracer, "anthropic.step",
		trace.WithAttributes(
			attribute.String("ai.provider", "anthropic"),
			attribute.String("ai.model", req.Model),
			attribute.Int("ai.tools_count", len(req.Tools)),
			attribute.Int("ai.messages_count", len(req.Messages)),
		),
	)
	defer span.End()

	apiReq, aiErr := buildStepRequest(req)
	if aiErr != nil {
		recordSpanError(span, aiErr)
		return nil, aiErr
	}

	jsonBody, err := json.Marshal(apiReq)
	if err != nil {
		e := ai.NewAIError(ai.CodeInternal, fmt.Sprintf("anthropic: failed to marshal request: %v", err), false)
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
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", c.apiVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		e := ai.ClassifyError(err)
		recordSpanError(span, e)
		return nil, e
	}
	defer func() { _ = resp.Body.Close() }()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e := ai.ClassifyError(err)
		recordSpanError(span, e)
		return nil, e
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyStr := string(body)
		// Try to hoist the inner error.message for a cleaner AIError.Message.
		var errEnv errorResponse
		if json.Unmarshal(body, &errEnv) == nil && errEnv.Error.Message != "" {
			bodyStr = errEnv.Error.Message
		}
		e := ai.ClassifyHTTPError("anthropic", resp.StatusCode, bodyStr)
		recordSpanError(span, e)
		return nil, e
	}

	var result messagesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		e := ai.NewAIError(ai.CodeProtocolError, fmt.Sprintf("anthropic: failed to parse response: %v", err), false)
		recordSpanError(span, e)
		return nil, e
	}

	out := &ai.Response{
		InputTokens:              result.Usage.InputTokens,
		OutputTokens:             result.Usage.OutputTokens,
		TotalTokens:              result.Usage.InputTokens + result.Usage.OutputTokens,
		CacheReadInputTokens:     result.Usage.CacheReadInputTokens,
		CacheCreationInputTokens: result.Usage.CacheCreationInputTokens,
		Model:                    result.Model,
		FinishReason:             mapStopReason(result.StopReason),
	}

	var textParts []string
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			// Re-marshal block.Input as the canonical JSON object string
			// for ToolCall.Arguments. block.Input is already a json.RawMessage
			// of the object Anthropic emitted.
			var args string
			if len(block.Input) > 0 {
				args = string(block.Input)
			} else {
				args = "{}"
			}
			out.ToolCalls = append(out.ToolCalls, ai.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		}
	}
	out.Text = strings.Join(textParts, "\n")

	span.SetAttributes(
		attribute.Int("ai.tokens_in", out.InputTokens),
		attribute.Int("ai.tokens_out", out.OutputTokens),
		attribute.Int("ai.tool_calls", len(out.ToolCalls)),
		attribute.String("ai.finish_reason", out.FinishReason),
	)

	return out, nil
}

// stepMessage / stepContentBlock / stepToolDef define the on-the-wire shape
// for the Step request body. They are distinct from the Generate-only types
// in client.go because Generate's messageContent.Content is a plain string,
// while Step needs json.RawMessage to support both string and content-array.
type stepMessagesRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      json.RawMessage `json:"system,omitempty"`
	Messages    []stepMessage   `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	Tools       []stepToolDef   `json:"tools,omitempty"`
	// Stream is set true by StreamStep (M-AI-STEP-STREAMING v0.18.7) to
	// switch the wire format to SSE. omitempty keeps the non-streaming
	// Step request bit-for-bit identical to pre-v0.18.7 wire bytes.
	Stream bool `json:"stream,omitempty"`
}

type stepMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type stepToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// stepContentBlock is the wire shape for content blocks in a content array.
// Tool inputs and tool results are JSON values on the wire — modeled as
// json.RawMessage to preserve fidelity.
type stepContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

// buildStepRequest translates an ai.Request into the Anthropic Messages API
// request body. Returns *ai.AIError on translation failure (e.g. malformed
// JSON in a ToolCall.Arguments).
func buildStepRequest(req *ai.Request) (*stepMessagesRequest, *ai.AIError) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	apiReq := &stepMessagesRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
	}
	systemField, err := systemFieldFromPrompt(req.SystemPrompt, req.CacheBreakpoints)
	if err != nil {
		return nil, ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("anthropic: failed to marshal system field: %v", err), false)
	}
	apiReq.System = systemField
	if req.Temperature > 0 {
		apiReq.Temperature = req.Temperature
	}

	for i, m := range req.Messages {
		switch m.Role {
		case "system":
			// Anthropic separates system from messages — drop here.
			continue

		case "user":
			if m.ToolCallID != "" {
				// User-with-ToolCallID is a tool result message.
				block := stepContentBlock{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   json.RawMessage(strconvJSONString(m.Content)),
				}
				blocks := []stepContentBlock{block}
				raw, err := json.Marshal(blocks)
				if err != nil {
					return nil, ai.NewAIError(ai.CodeInternal,
						fmt.Sprintf("anthropic: failed to marshal tool_result content for messages[%d]: %v", i, err), false)
				}
				apiReq.Messages = append(apiReq.Messages, stepMessage{Role: "user", Content: raw})
			} else {
				raw, err := json.Marshal(m.Content)
				if err != nil {
					return nil, ai.NewAIError(ai.CodeInternal,
						fmt.Sprintf("anthropic: failed to marshal user content for messages[%d]: %v", i, err), false)
				}
				apiReq.Messages = append(apiReq.Messages, stepMessage{Role: "user", Content: raw})
			}

		case "tool":
			// Tool-result feedback — Anthropic models this as a user message
			// with a content array containing a tool_result block.
			block := stepContentBlock{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   json.RawMessage(strconvJSONString(m.Content)),
			}
			blocks := []stepContentBlock{block}
			raw, err := json.Marshal(blocks)
			if err != nil {
				return nil, ai.NewAIError(ai.CodeInternal,
					fmt.Sprintf("anthropic: failed to marshal tool content for messages[%d]: %v", i, err), false)
			}
			apiReq.Messages = append(apiReq.Messages, stepMessage{Role: "user", Content: raw})

		case "assistant":
			if len(m.ToolCalls) == 0 {
				raw, err := json.Marshal(m.Content)
				if err != nil {
					return nil, ai.NewAIError(ai.CodeInternal,
						fmt.Sprintf("anthropic: failed to marshal assistant content for messages[%d]: %v", i, err), false)
				}
				apiReq.Messages = append(apiReq.Messages, stepMessage{Role: "assistant", Content: raw})
			} else {
				var blocks []stepContentBlock
				if m.Content != "" {
					blocks = append(blocks, stepContentBlock{Type: "text", Text: m.Content})
				}
				for j, tc := range m.ToolCalls {
					// Decode arguments string (JSON object) so we can
					// re-marshal as a JSON object on the wire — Anthropic
					// requires "input" to be an object, not a string.
					var input json.RawMessage
					if tc.Arguments == "" {
						input = json.RawMessage("{}")
					} else {
						var probe any
						if err := json.Unmarshal([]byte(tc.Arguments), &probe); err != nil {
							return nil, ai.NewAIError(ai.CodeInternal,
								fmt.Sprintf("anthropic: tool_call[%d].Arguments is not valid JSON for messages[%d]: %v", j, i, err), false)
						}
						input = json.RawMessage(tc.Arguments)
					}
					blocks = append(blocks, stepContentBlock{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Name,
						Input: input,
					})
				}
				raw, err := json.Marshal(blocks)
				if err != nil {
					return nil, ai.NewAIError(ai.CodeInternal,
						fmt.Sprintf("anthropic: failed to marshal assistant content for messages[%d]: %v", i, err), false)
				}
				apiReq.Messages = append(apiReq.Messages, stepMessage{Role: "assistant", Content: raw})
			}

		default:
			return nil, ai.NewAIError(ai.CodeInternal,
				fmt.Sprintf("anthropic: unknown message role %q at messages[%d]", m.Role, i), false)
		}
	}

	for i, ts := range req.Tools {
		var schema json.RawMessage
		if ts.Parameters == "" {
			schema = json.RawMessage(`{"type":"object"}`)
		} else {
			var probe any
			if err := json.Unmarshal([]byte(ts.Parameters), &probe); err != nil {
				return nil, ai.NewAIError(ai.CodeInternal,
					fmt.Sprintf("anthropic: tools[%d].Parameters is not valid JSON: %v", i, err), false)
			}
			schema = json.RawMessage(ts.Parameters)
		}
		apiReq.Tools = append(apiReq.Tools, stepToolDef{
			Name:        ts.Name,
			Description: ts.Description,
			InputSchema: schema,
		})
	}

	return apiReq, nil
}

// strconvJSONString quotes s as a JSON string literal. Uses json.Marshal to
// guarantee correct escaping (newlines, quotes, control chars, unicode).
func strconvJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// json.Marshal of a string only fails on invalid UTF-8 in pathological
		// cases. Fall back to a safe empty string — caller has already
		// committed to the wire shape.
		return `""`
	}
	return string(b)
}

// mapStopReason converts Anthropic's stop_reason into the normalized
// ai.Response.FinishReason vocabulary. Empty input maps to "" (back-compat).
func mapStopReason(reason string) string {
	switch reason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	case "":
		return ""
	default:
		return "error"
	}
}

// recordSpanError annotates the span with the AIError's code/message and
// marks it as a failure. Centralized so all error paths produce the same
// telemetry shape.
func recordSpanError(span trace.Span, e *ai.AIError) {
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
