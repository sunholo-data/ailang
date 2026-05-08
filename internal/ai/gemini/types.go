// Package gemini provides a Google Gemini API client implementing the ai.Provider interface.
// It supports both AI Studio (API key) and Vertex AI (ADC) authentication.
package gemini

import "encoding/json"

// generateRequest represents the request body for generateContent API.
type generateRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	Tools             []toolBlock       `json:"tools,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

// toolBlock is one entry in the top-level "tools" array. Gemini groups
// function declarations together; M-AI-TOOL-LOOP M3 emits a single block
// containing all advertised tools.
type toolBlock struct {
	FunctionDeclarations []functionDeclaration `json:"functionDeclarations"`
}

// functionDeclaration describes one callable tool to the model.
// Parameters is the decoded JSON Schema (object form), since Gemini rejects
// the schema as a string.
type functionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// functionCall is the model-emitted tool invocation that appears as a
// part on a "model" content. Args is decoded JSON (object form).
type functionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// functionResponse is the host-supplied tool result that appears as a
// part on a "user" content (Gemini reuses the user role for tool results).
// Response is a free-form JSON object — by convention M3 wraps the
// stringified tool output as {"content": <string>}.
type functionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// content represents a content block with role and parts.
type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

// part represents a content part (text, inline_data, file_data, functionCall, functionResponse).
type part struct {
	Text             string            `json:"text,omitempty"`
	InlineData       *inlineData       `json:"inlineData,omitempty"`       // For multimodal (images, PDFs, etc.)
	FileData         *fileData         `json:"fileData,omitempty"`         // For file URI references (GCS, Files API)
	FunctionCall     *functionCall     `json:"functionCall,omitempty"`     // Model-emitted tool invocation (Step path)
	FunctionResponse *functionResponse `json:"functionResponse,omitempty"` // Host-supplied tool result (Step path)
}

// inlineData represents inline binary data for multimodal requests.
type inlineData struct {
	MimeType string `json:"mimeType"` // e.g., "application/pdf", "image/png"
	Data     string `json:"data"`     // Base64-encoded content
}

// fileData represents a reference to a file stored externally (GCS, Gemini Files API).
type fileData struct {
	MimeType string `json:"mimeType"` // e.g., "application/pdf", "image/png"
	FileUri  string `json:"fileUri"`  // gs://bucket/file.pdf or Files API URI
}

// generationConfig represents generation parameters.
type generationConfig struct {
	MaxOutputTokens    int              `json:"maxOutputTokens,omitempty"`
	Temperature        float64          `json:"temperature,omitempty"`
	TopP               float64          `json:"topP,omitempty"`
	TopK               int              `json:"topK,omitempty"`
	ResponseMimeType   string           `json:"responseMimeType,omitempty"`   // "application/json" for structured output
	ResponseSchema     *json.RawMessage `json:"responseSchema,omitempty"`     // JSON Schema for structured output
	ResponseModalities []string         `json:"responseModalities,omitempty"` // ["TEXT"], ["IMAGE"], etc.
}

// generateResponse represents the response from generateContent API.
type generateResponse struct {
	Candidates    []candidate   `json:"candidates"`
	UsageMetadata usageMetadata `json:"usageMetadata"`
}

// stepRawResponse is the Step-path response shape — same fields as
// generateResponse plus modelVersion (added by Gemini in v1beta when a
// model alias resolves to a specific revision). Kept separate so the
// legacy Generate path's response shape stays untouched.
type stepRawResponse struct {
	Candidates     []candidate    `json:"candidates"`
	UsageMetadata  usageMetadata  `json:"usageMetadata"`
	ModelVersion   string         `json:"modelVersion,omitempty"`
	PromptFeedback promptFeedback `json:"promptFeedback,omitempty"`
}

// promptFeedback captures safety block information returned by Gemini when
// the prompt is blocked before any candidates are generated.
type promptFeedback struct {
	BlockReason   string `json:"blockReason,omitempty"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

// candidate represents a generation candidate.
type candidate struct {
	Content       content `json:"content"`
	FinishReason  string  `json:"finishReason"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

// usageMetadata represents token usage.
type usageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount,omitempty"`      // Reasoning tokens (Gemini 3+)
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"` // Cache hit tokens (Gemini context-caching)
}

// errorResponse represents an error response from the API.
type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// AuthType indicates the authentication method.
type AuthType string

const (
	// AuthAPIKey uses API key authentication (AI Studio).
	AuthAPIKey AuthType = "apikey"

	// AuthADC uses Application Default Credentials (Vertex AI).
	AuthADC AuthType = "adc"
)
