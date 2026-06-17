package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Ollama's /api/chat endpoint now supports native
// tool/function calling for models that advertise it (e.g. Qwen, Llama 3.1+),
// so this adapter translates req.Tools into Ollama's tool schema, threads tool
// calls + tool results through the conversation, and parses tool_calls back out
// of the response. This is what lets AILANG-native agents (e.g. motoko_agent)
// run a real tool loop against a local Ollama model at $0.
//
// Behaviour:
//   - no Messages AND no Tools → delegate to Generate (legacy single-shot path)
//   - otherwise               → multi-turn chat, tools advertised when present
//   - response carries ToolCalls + FinishReason="tool_calls" when the model
//     decides to call a tool; otherwise FinishReason="stop".
//
// Note: a model whose Modelfile template lacks tool support will simply never
// emit tool_calls (it answers in prose). Ollama returns an error only for a
// malformed request, which we surface via ClassifyError.
// bareModel strips the AILANG provider-routing prefix ("ollama:" or "ollama/")
// from a model string so the Ollama API receives the raw model tag — e.g.
// "ollama:qwen3.5:35b-a3b-mxfp8" → "qwen3.5:35b-a3b-mxfp8". GuessProvider uses
// the prefix to ROUTE to this provider, but Ollama's /api/chat rejects the
// prefixed form as "invalid model name". Only the first segment is stripped, so
// the model's own ":tag" (e.g. ":35b-a3b-mxfp8") is preserved.
func bareModel(m string) string {
	if i := strings.IndexAny(m, ":/"); i > 0 && strings.EqualFold(m[:i], "ollama") {
		return m[i+1:]
	}
	return m
}

func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Single-shot path: no Messages and no Tools → fall back to Generate.
	if len(req.Messages) == 0 && len(req.Tools) == 0 {
		return c.Generate(ctx, req)
	}

	// Tool-calling path: route through Ollama's OpenAI-compatible /v1 endpoint
	// (M-OLLAMA-V1-TOOLCALLING). Small local models (e.g. qwen3.x) reliably emit
	// `tool_calls` over /v1/chat/completions — Ollama's compat layer normalizes
	// the model's tool-call output — but frequently emit ZERO native tool calls
	// over /api/chat, which silently degrades AILANG-native agents (motoko) to
	// non-agentic 0-shot. pi/opencode (the 96%/79% reference harnesses) both drive
	// Ollama via /v1 for exactly this reason. Reuse AILANG's OpenAI provider
	// pointed at the Ollama host's /v1 (dummy key; Ollama ignores auth). Set
	// AILANG_OLLAMA_NATIVE_TOOLS=1 to force the legacy native /api/chat tool path.
	if len(req.Tools) > 0 && os.Getenv("AILANG_OLLAMA_NATIVE_TOOLS") != "1" {
		v1 := openai.NewClient("ollama", openai.WithBaseURL(strings.TrimRight(c.endpoint, "/")+"/v1"))
		r2 := *req
		r2.Model = bareModel(req.Model)
		return v1.Step(ctx, &r2)
	}

	// Translate req.Messages → ollamaapi.Message[], threading tool calls/results.
	messages := make([]ollamaapi.Message, 0, len(req.Messages)+1)

	// Prepend req.SystemPrompt unless req.Messages already carries a system role.
	hasSystemMsg := false
	for _, m := range req.Messages {
		if m.Role == "system" {
			hasSystemMsg = true
			break
		}
	}
	if req.SystemPrompt != "" && !hasSystemMsg {
		messages = append(messages, ollamaapi.Message{Role: "system", Content: req.SystemPrompt})
	}

	for _, m := range req.Messages {
		om := ollamaapi.Message{Role: m.Role, Content: m.Content}
		// Tool-result message: Ollama has a native "tool" role with tool_call_id.
		if m.Role == "tool" {
			om.ToolCallID = m.ToolCallID
		}
		// Assistant message that issued tool calls: re-attach them so the model
		// sees its own prior calls in the running conversation.
		if len(m.ToolCalls) > 0 {
			om.ToolCalls = make([]ollamaapi.ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				oc := ollamaapi.ToolCall{Function: ollamaapi.ToolCallFunction{Name: tc.Name}}
				if tc.Arguments != "" {
					// Best-effort: Ollama wants structured args; a parse failure
					// just yields empty args rather than dropping the whole call.
					_ = json.Unmarshal([]byte(tc.Arguments), &oc.Function.Arguments)
				}
				om.ToolCalls = append(om.ToolCalls, oc)
			}
		}
		messages = append(messages, om)
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
		Model:    bareModel(req.Model),
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

	// Advertise tools: translate each ai.ToolSchema → ollamaapi.Tool. The
	// ToolSchema.Parameters is a JSON-Schema string; unmarshal it into Ollama's
	// structured parameter type.
	if len(req.Tools) > 0 {
		tools := make(ollamaapi.Tools, 0, len(req.Tools))
		for _, t := range req.Tools {
			fn := ollamaapi.ToolFunction{Name: t.Name, Description: t.Description}
			if t.Parameters != "" {
				if err := json.Unmarshal([]byte(t.Parameters), &fn.Parameters); err != nil {
					return nil, ai.NewAIError(ai.CodeToolsNotSupported,
						fmt.Sprintf("ollama: invalid JSON-Schema parameters for tool %q: %v", t.Name, err), false)
				}
			}
			tools = append(tools, ollamaapi.Tool{Type: "function", Function: fn})
		}
		chatReq.Tools = tools
	}

	var response strings.Builder
	var toolCalls []ai.ToolCall
	err := c.client.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
		response.WriteString(resp.Message.Content)
		// Tool calls arrive in the message (possibly across streamed chunks).
		for _, tc := range resp.Message.ToolCalls {
			toolCalls = append(toolCalls, ai.ToolCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments.String(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, ai.ClassifyError(err)
	}

	finish := "stop"
	if len(toolCalls) > 0 {
		finish = "tool_calls"
	}
	return &ai.Response{
		Text:         response.String(),
		Model:        req.Model,
		FinishReason: finish,
		ToolCalls:    toolCalls,
		// Ollama doesn't report tokens uniformly across versions; leave at 0.
		InputTokens:  0,
		OutputTokens: 0,
		TotalTokens:  0,
	}, nil
}
