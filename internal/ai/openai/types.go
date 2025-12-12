// Package openai provides an OpenAI API client implementing the ai.Provider interface.
// It supports both Chat Completions API and Responses API (for codex models).
package openai

// chatRequest represents the request body for Chat Completions API.
type chatRequest struct {
	Model               string        `json:"model"`
	Messages            []chatMessage `json:"messages"`
	MaxTokens           int           `json:"max_tokens,omitempty"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"` // GPT-5.1+ use this
	Temperature         float64       `json:"temperature,omitempty"`
	Seed                *int64        `json:"seed,omitempty"`
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
