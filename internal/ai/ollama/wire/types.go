// Package wire is AILANG's Ollama native-API client: the request and reply
// types for /api/generate, /api/chat, /api/embed and /api/tags, and a small
// HTTP client for them. It replaces github.com/ollama/ollama/api (v0.40.15),
// which carried eight reachable, unfixed govulncheck advisories into the
// binary for the sake of a handful of structs.
//
// The contract is byte parity with that SDK: the same request bodies, headers
// and paths, the same decoding of replies, and the same error texts (callers
// classify errors by their text). The goldens under
// internal/ai/ollama/testdata/wire, internal/builtins/testdata and
// internal/messaging/testdata were captured from the SDK and pin it.
//
// Types carry only the fields some AILANG caller sends or reads. Every field
// left out is omitempty in the SDK, so its absence changes no byte of any
// request AILANG can build; a reply field left out is simply not decoded.
// Adding a field means adding a fixture for it.
package wire

import (
	"encoding/json"
	"strings"
	"time"
)

// ImageData is raw image bytes; encoding/json writes them as base64.
type ImageData []byte

// GenerateRequest is the body of POST /api/generate. Suffix, System and
// Template are not omitempty: the SDK always sent them, empty or not, and so
// does this type. Options is not omitempty either (nil goes out as null).
type GenerateRequest struct {
	Model    string          `json:"model"`
	Prompt   string          `json:"prompt"`
	Suffix   string          `json:"suffix"`
	System   string          `json:"system"`
	Template string          `json:"template"`
	Format   json.RawMessage `json:"format,omitempty"`
	Options  map[string]any  `json:"options"`
}

// GenerateResponse is one streamed /api/generate chunk.
type GenerateResponse struct {
	Model      string     `json:"model"`
	CreatedAt  time.Time  `json:"created_at"`
	Response   string     `json:"response"`
	Thinking   string     `json:"thinking,omitempty"`
	Done       bool       `json:"done"`
	DoneReason string     `json:"done_reason,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Metrics
}

// ChatRequest is the body of POST /api/chat.
type ChatRequest struct {
	Model    string          `json:"model"`
	Messages []Message       `json:"messages"`
	Format   json.RawMessage `json:"format,omitempty"`
	Tools    Tools           `json:"tools,omitempty"`
	Options  map[string]any  `json:"options"`
}

// ChatResponse is one streamed /api/chat chunk.
type ChatResponse struct {
	Model      string    `json:"model"`
	CreatedAt  time.Time `json:"created_at"`
	Message    Message   `json:"message"`
	Done       bool      `json:"done"`
	DoneReason string    `json:"done_reason,omitempty"`
	Metrics
}

// Message is one chat message.
type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	Thinking   string      `json:"thinking,omitempty"`
	Images     []ImageData `json:"images,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolName   string      `json:"tool_name,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// UnmarshalJSON lower-cases the role, as the SDK did.
func (m *Message) UnmarshalJSON(b []byte) error {
	type Alias Message // the SDK's name; it can surface in decode errors
	var a Alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*m = Message(a)
	m.Role = strings.ToLower(m.Role)
	return nil
}

// Metrics are the timing and token counts Ollama reports on the final chunk.
type Metrics struct {
	TotalDuration         time.Duration `json:"total_duration,omitempty"`
	LoadDuration          time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount       int           `json:"prompt_eval_count,omitempty"`
	PromptEvalCachedCount *int          `json:"prompt_eval_cached_count,omitempty"`
	PromptEvalDuration    time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount             int           `json:"eval_count,omitempty"`
	EvalDuration          time.Duration `json:"eval_duration,omitempty"`
}

// EmbedRequest is the body of POST /api/embed. Input is a string, a list of
// strings, or a {"text","image","audio"} map (media base64-encoded).
type EmbedRequest struct {
	Model   string         `json:"model"`
	Input   any            `json:"input"`
	Options map[string]any `json:"options"`
}

// EmbedResponse is the reply to /api/embed.
type EmbedResponse struct {
	Model           string        `json:"model"`
	Embeddings      [][]float32   `json:"embeddings"`
	TotalDuration   time.Duration `json:"total_duration,omitempty"`
	LoadDuration    time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount int           `json:"prompt_eval_count,omitempty"`
}

// ListResponse is the reply to GET /api/tags.
type ListResponse struct {
	Models []ListModelResponse `json:"models"`
}

// ListModelResponse is one locally available model.
type ListModelResponse struct {
	Name         string       `json:"name"`
	Model        string       `json:"model"`
	ModifiedAt   time.Time    `json:"modified_at"`
	Size         int64        `json:"size"`
	Digest       string       `json:"digest"`
	Details      ModelDetails `json:"details,omitempty"`
	Capabilities []string     `json:"capabilities,omitempty"`
}

// ModelDetails describes a model's format and family.
type ModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
	ContextLength     int      `json:"context_length,omitempty"`
	EmbeddingLength   int      `json:"embedding_length,omitempty"`
}
