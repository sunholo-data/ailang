package configdriven

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
)

// buildRequestBody serialises a request to JSON using the shape declared in
// the provider's [[ai_provider]] config. Returns the JSON bytes ready to be
// POSTed.
//
// Shapes:
//   - openai_chat        — OpenAI Chat Completions wire format (also used by
//     OpenRouter, Together, Groq, Anyscale, Fireworks,
//     DeepInfra, Perplexity, vLLM, llama.cpp openai-compat)
//   - anthropic_messages — Anthropic Messages API wire format
//   - simple_completion  — single-prompt-string format (Ollama-style)
//   - custom             — request_template Go template (deferred until v2 if needed)
func buildRequestBody(shape string, req *ai.Request) ([]byte, error) {
	switch shape {
	case "openai_chat":
		return buildOpenAIChat(req)
	case "anthropic_messages":
		return buildAnthropicMessages(req)
	case "simple_completion":
		return buildSimpleCompletion(req)
	case "custom":
		return nil, fmt.Errorf("request_shape=\"custom\" is reserved for schema v2; not implemented in v1")
	default:
		return nil, fmt.Errorf("unknown request_shape %q (valid: openai_chat, anthropic_messages, simple_completion)", shape)
	}
}

// openaiMessage matches OpenAI's chat message shape: {role, content}.
type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiChatRequest matches the OpenAI Chat Completions request shape.
type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	// ResponseFormat is added when req.ResponseFormat == "json"
	ResponseFormat *responseFormatField `json:"response_format,omitempty"`
}

type responseFormatField struct {
	Type   string          `json:"type"` // "json_object" or "json_schema"
	Schema json.RawMessage `json:"json_schema,omitempty"`
}

func buildOpenAIChat(req *ai.Request) ([]byte, error) {
	messages := make([]openaiMessage, 0, 2)
	if req.SystemPrompt != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, openaiMessage{Role: "user", Content: req.UserPrompt})

	body := openaiChatRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			body.ResponseFormat = &responseFormatField{
				Type:   "json_schema",
				Schema: json.RawMessage(req.ResponseSchema),
			}
		} else {
			body.ResponseFormat = &responseFormatField{Type: "json_object"}
		}
	}
	return json.Marshal(body)
}

// anthropicContentBlock matches Anthropic's content-block shape.
type anthropicContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicMessagesRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"` // Anthropic requires this
}

func buildAnthropicMessages(req *ai.Request) ([]byte, error) {
	maxTok := req.MaxTokens
	if maxTok == 0 {
		maxTok = 4096 // Anthropic-required default
	}
	body := anthropicMessagesRequest{
		Model:     req.Model,
		System:    req.SystemPrompt,
		MaxTokens: maxTok,
		Messages: []anthropicMessage{{
			Role:    "user",
			Content: []anthropicContentBlock{{Type: "text", Text: req.UserPrompt}},
		}},
	}
	return json.Marshal(body)
}

// simpleCompletionRequest matches the Ollama-style single-prompt shape.
type simpleCompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	System      string  `json:"system,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

func buildSimpleCompletion(req *ai.Request) ([]byte, error) {
	body := simpleCompletionRequest{
		Model:       req.Model,
		Prompt:      req.UserPrompt,
		System:      req.SystemPrompt,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	return json.Marshal(body)
}
