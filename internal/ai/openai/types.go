// Package openai provides an OpenAI API client implementing the ai.Provider interface.
// It supports both Chat Completions API and Responses API (for codex models).
package openai

import "encoding/json"

// chatRequest represents the request body for Chat Completions API.
type chatRequest struct {
	Model               string              `json:"model"`
	Messages            []chatMessage       `json:"messages"`
	MaxTokens           int                 `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                 `json:"max_completion_tokens,omitempty"` // GPT-5.1+ use this
	Temperature         float64             `json:"temperature,omitempty"`
	Seed                *int64              `json:"seed,omitempty"`
	ResponseFormat      *chatResponseFormat `json:"response_format,omitempty"` // Structured output
}

// chatResponseFormat configures structured output for Chat Completions API.
type chatResponseFormat struct {
	Type       string          `json:"type"`                  // "json_schema" or "json_object"
	JSONSchema *chatJSONSchema `json:"json_schema,omitempty"` // Schema definition
}

// chatJSONSchema defines the JSON schema for structured output.
type chatJSONSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

// chatMessage represents a message in the Chat Completions API.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse represents the response from Chat Completions API.
type chatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   chatUsage    `json:"usage"`
}

// chatChoice represents a completion choice.
type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// chatUsage represents token usage in the response.
type chatUsage struct {
	PromptTokens            int `json:"prompt_tokens"`
	CompletionTokens        int `json:"completion_tokens"`
	TotalTokens             int `json:"total_tokens"`
	CompletionTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details,omitempty"`
}

// errorResponse represents an error response from the API.
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// APIType indicates which OpenAI API to use.
type APIType string

const (
	// APIChatCompletions uses the /v1/chat/completions endpoint.
	APIChatCompletions APIType = "chat"

	// APIResponses uses the /v1/responses endpoint (for codex models).
	APIResponses APIType = "responses"
)

// ============================================================================
// Responses API Types (/v1/responses)
// ============================================================================

// responsesRequest represents the request body for Responses API.
type responsesRequest struct {
	Model     string              `json:"model"`
	Input     []responsesInput    `json:"input"`
	Reasoning *responsesReasoning `json:"reasoning,omitempty"`
	MaxTokens int                 `json:"max_output_tokens,omitempty"`
	Text      *responsesText      `json:"text,omitempty"` // Structured output config
}

// responsesText configures structured output for Responses API.
type responsesText struct {
	Format responsesTextFormat `json:"format"`
}

// responsesTextFormat defines the format constraint for Responses API.
type responsesTextFormat struct {
	Type   string          `json:"type"`             // "json_schema" or "json_object"
	Name   string          `json:"name,omitempty"`   // Schema name
	Schema json.RawMessage `json:"schema,omitempty"` // JSON Schema
	Strict bool            `json:"strict,omitempty"` // Enforce strict schema
}

// responsesInput represents an input item in the Responses API.
// Uses "developer" role instead of "system" for instructions.
type responsesInput struct {
	Role    string `json:"role"` // "developer", "user", or "assistant"
	Content string `json:"content"`
}

// responsesReasoning controls reasoning effort for the model.
type responsesReasoning struct {
	Effort string `json:"effort"` // "none", "low", "medium", "high"
}

// responsesResponse represents the response from Responses API.
type responsesResponse struct {
	ID     string                `json:"id"`
	Object string                `json:"object"`
	Model  string                `json:"model"`
	Output []responsesOutputItem `json:"output"`
	Usage  responsesUsage        `json:"usage"`
}

// responsesOutputItem represents a polymorphic output item.
// Type can be: "message", "reasoning", "function_call"
type responsesOutputItem struct {
	Type    string             `json:"type"`
	Status  string             `json:"status,omitempty"`
	Role    string             `json:"role,omitempty"`    // for message type
	Content []responsesContent `json:"content,omitempty"` // for message type
	Summary []responsesSummary `json:"summary,omitempty"` // for reasoning type
	// Function call fields
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	CallID    string `json:"call_id,omitempty"`
}

// responsesContent represents content within an output item.
type responsesContent struct {
	Type string `json:"type"` // "output_text", "refusal", etc.
	Text string `json:"text"`
}

// responsesSummary represents a reasoning summary.
type responsesSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// responsesUsage represents token usage in Responses API.
type responsesUsage struct {
	InputTokens   int `json:"input_tokens"`
	OutputTokens  int `json:"output_tokens"`
	TotalTokens   int `json:"total_tokens"`
	OutputDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details,omitempty"`
}

// ensureStrictSchemaCompliance makes a JSON Schema compatible with OpenAI's strict
// mode, which requires "additionalProperties": false and "required" listing all
// properties on every object-type schema.
func ensureStrictSchemaCompliance(schema json.RawMessage) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &obj); err != nil {
		return schema
	}
	// Only process object types
	if typeRaw, ok := obj["type"]; ok {
		var t string
		if json.Unmarshal(typeRaw, &t) != nil || t != "object" {
			return schema
		}
	}
	// Inject additionalProperties: false if missing
	if _, ok := obj["additionalProperties"]; !ok {
		obj["additionalProperties"] = json.RawMessage("false")
	}
	// Inject required with all property keys if missing
	if _, ok := obj["required"]; !ok {
		if propsRaw, ok := obj["properties"]; ok {
			var props map[string]json.RawMessage
			if json.Unmarshal(propsRaw, &props) == nil && len(props) > 0 {
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				reqJSON, _ := json.Marshal(keys)
				obj["required"] = json.RawMessage(reqJSON)
			}
		}
	}
	result, err := json.Marshal(obj)
	if err != nil {
		return schema
	}
	return result
}
