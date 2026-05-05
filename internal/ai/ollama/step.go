package ollama

import (
	"context"
	"encoding/json"
	"strings"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Ollama's local /api/chat endpoint does NOT
// uniformly support tool calling across the model catalogue (some models
// support it via Modelfile templating, most don't), so v1 of this adapter
// rejects tool requests with a typed error and otherwise routes the
// conversation through Ollama's chat endpoint.
//
// Behaviour:
//   - len(req.Tools) > 0   → AIError{Code: ToolsNotSupported, Retryable: false}
//   - req.Messages empty   → delegate to Generate (legacy single-shot path)
//   - req.Messages present → multi-turn chat (no tool dispatch)
//
// Multi-turn responses always have FinishReason="stop" and ToolCalls=nil.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if len(req.Tools) > 0 {
		return nil, ai.NewAIError(ai.CodeToolsNotSupported,
			"ollama: tool calling not supported (set req.Tools=nil to fall through to chat)", false)
	}

	// Single-shot path: no Messages → fall back to existing Generate.
	if len(req.Messages) == 0 {
		return c.Generate(ctx, req)
	}

	// Multi-turn chat: translate req.Messages → ollamaapi.Message[].
	messages := make([]ollamaapi.Message, 0, len(req.Messages)+1)

	// If req.SystemPrompt is set AND no system-role message in req.Messages,
	// prepend the system prompt. If req.Messages already has a system role,
	// don't double up — req.Messages wins.
	hasSystemMsg := false
	for _, m := range req.Messages {
		if m.Role == "system" {
			hasSystemMsg = true
			break
		}
	}
	if req.SystemPrompt != "" && !hasSystemMsg {
		messages = append(messages, ollamaapi.Message{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, m := range req.Messages {
		role := m.Role
		// Ollama recognizes user / assistant / system / tool. Map "tool"
		// (which carries a ToolCallID) to a user message with a tool-result
		// prefix, since Ollama doesn't have native tool-result semantics.
		if role == "tool" {
			messages = append(messages, ollamaapi.Message{
				Role:    "user",
				Content: "[tool result] " + m.Content,
			})
			continue
		}
		messages = append(messages, ollamaapi.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	options := map[string]interface{}{
		"seed":    int64(42),
		"num_ctx": 8192,
	}
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	} else {
		options["num_predict"] = 4096
	}
	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}

	chatReq := &ollamaapi.ChatRequest{
		Model:    req.Model,
		Messages: messages,
		Options:  options,
	}
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			chatReq.Format = json.RawMessage(req.ResponseSchema)
		} else {
			chatReq.Format = json.RawMessage(`"json"`)
		}
	}

	var response strings.Builder
	err := c.client.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
		response.WriteString(resp.Message.Content)
		return nil
	})
	if err != nil {
		return nil, ai.ClassifyError(err)
	}

	return &ai.Response{
		Text:         response.String(),
		Model:        req.Model,
		FinishReason: "stop",
		// Ollama doesn't report tokens uniformly across versions; leave at 0.
		InputTokens:  0,
		OutputTokens: 0,
		TotalTokens:  0,
	}, nil
}
