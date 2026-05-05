package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// stepReqBody mirrors the wire shape Step sends to Anthropic so tests can
// inspect what was actually serialized. It is intentionally permissive:
// content can be a plain string OR a content-block array, so we decode it
// as json.RawMessage and decode further per case.
type stepReqBody struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	System      string           `json:"system,omitempty"`
	Messages    []stepReqMessage `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
	Tools       []stepReqToolDef `json:"tools,omitempty"`
}

type stepReqMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type stepReqToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// stepReqContentBlock is what each entry of a content-array message decodes
// to (used by tests that assert on tool_use / tool_result shapes).
type stepReqContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

// captureHandler records the last request body it sees and serves a
// canned response. Useful for the tests that need to assert on the
// outbound wire shape.
type captureHandler struct {
	captured []byte
	respond  func(w http.ResponseWriter, r *http.Request)
}

func (h *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	h.captured = body
	if h.respond != nil {
		h.respond(w, r)
		return
	}
	// Default: empty success
	resp := messagesResponse{
		ID:    "msg_test",
		Type:  "message",
		Role:  "assistant",
		Model: "claude-sonnet-4-5",
		Content: []contentBlock{
			{Type: "text", Text: "ok"},
		},
		StopReason: "end_turn",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func TestStep_TextOnly_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			ID:    "msg_abc",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-5-20251001",
			Content: []contentBlock{
				{Type: "text", Text: "Hello there!"},
			},
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  12,
				OutputTokens: 7,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.Text != "Hello there!" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello there!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
	if resp.InputTokens != 12 || resp.OutputTokens != 7 || resp.TotalTokens != 19 {
		t.Errorf("tokens in/out/total = %d/%d/%d, want 12/7/19",
			resp.InputTokens, resp.OutputTokens, resp.TotalTokens)
	}
	if resp.Model != "claude-sonnet-4-5-20251001" {
		t.Errorf("Model = %q, want claude-sonnet-4-5-20251001", resp.Model)
	}
}

func TestStep_SingleToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			ID:    "msg_tool",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-5",
			Content: []contentBlock{
				{
					Type:  "tool_use",
					ID:    "toolu_01abc",
					Name:  "read_doc",
					Input: json.RawMessage(`{"name":"nda.docx"}`),
				},
			},
			StopReason: "tool_use",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "Read nda.docx"},
		},
		Tools: []ai.ToolSchema{
			{
				Name:        "read_doc",
				Description: "Read a doc by name",
				Parameters:  `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_01abc" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", tc.ID, "toolu_01abc")
	}
	if tc.Name != "read_doc" {
		t.Errorf("ToolCalls[0].Name = %q, want %q", tc.Name, "read_doc")
	}
	// Arguments must be valid JSON of the input object.
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
		t.Fatalf("ToolCalls[0].Arguments not valid JSON: %v (%q)", err, tc.Arguments)
	}
	if args["name"] != "nda.docx" {
		t.Errorf("Arguments name = %v, want nda.docx", args["name"])
	}
}

func TestStep_MultipleToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			ID:    "msg_multi",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-5",
			Content: []contentBlock{
				{Type: "text", Text: "I'll fetch both."},
				{
					Type:  "tool_use",
					ID:    "toolu_1",
					Name:  "read_doc",
					Input: json.RawMessage(`{"name":"a.txt"}`),
				},
				{
					Type:  "tool_use",
					ID:    "toolu_2",
					Name:  "read_doc",
					Input: json.RawMessage(`{"name":"b.txt"}`),
				},
			},
			StopReason: "tool_use",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "Read a and b"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.Text != "I'll fetch both." {
		t.Errorf("Text = %q, want %q", resp.Text, "I'll fetch both.")
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("ToolCalls len = %d, want 2", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "toolu_1" || resp.ToolCalls[1].ID != "toolu_2" {
		t.Errorf("Tool IDs = %q,%q want toolu_1,toolu_2",
			resp.ToolCalls[0].ID, resp.ToolCalls[1].ID)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "tool_calls")
	}
}

func TestStep_ToolResultFeedback(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "Read nda.docx"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "toolu_xyz",
						Name:      "read_doc",
						Arguments: `{"name":"nda.docx"}`,
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: "toolu_xyz",
				Content:    "Lorem ipsum doc body.",
			},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.Text != "ok" {
		t.Errorf("Text = %q, want %q", resp.Text, "ok")
	}

	// Inspect the captured outbound body.
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Messages) != 3 {
		t.Fatalf("sent messages len = %d, want 3", len(sent.Messages))
	}
	// Final message must be a user role with a content array containing
	// a tool_result block referencing toolu_xyz.
	last := sent.Messages[2]
	if last.Role != "user" {
		t.Errorf("last message role = %q, want user", last.Role)
	}
	var blocks []stepReqContentBlock
	if err := json.Unmarshal(last.Content, &blocks); err != nil {
		t.Fatalf("last content not an array: %v (%s)", err, string(last.Content))
	}
	if len(blocks) != 1 {
		t.Fatalf("tool_result blocks len = %d, want 1", len(blocks))
	}
	if blocks[0].Type != "tool_result" {
		t.Errorf("block type = %q, want tool_result", blocks[0].Type)
	}
	if blocks[0].ToolUseID != "toolu_xyz" {
		t.Errorf("tool_use_id = %q, want toolu_xyz", blocks[0].ToolUseID)
	}
	// content can be a string or array; we accept either.
	var asStr string
	if err := json.Unmarshal(blocks[0].Content, &asStr); err == nil {
		if asStr != "Lorem ipsum doc body." {
			t.Errorf("tool_result content = %q, want %q", asStr, "Lorem ipsum doc body.")
		}
	}
}

func TestStep_FinishReason_Length(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			ID:    "msg_long",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-5",
			Content: []contentBlock{
				{Type: "text", Text: "truncated..."},
			},
			StopReason: "max_tokens",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "long question"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.FinishReason != "length" {
		t.Errorf("FinishReason = %q, want length", resp.FinishReason)
	}
}

func TestStep_AuthFailed_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeAuthFailed {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeAuthFailed)
	}
	if aiErr.Retryable {
		t.Errorf("Retryable = true, want false")
	}
}

func TestStep_RateLimit_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeRateLimit {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeRateLimit)
	}
	if !aiErr.Retryable {
		t.Errorf("Retryable = false, want true")
	}
}

func TestStep_5xx_Internal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`Service Unavailable`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
	if !aiErr.Retryable {
		t.Errorf("Retryable = false, want true")
	}
}

func TestStep_ContextLength_400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"context length exceeded for model"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeContextLength {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeContextLength)
	}
	if aiErr.Retryable {
		t.Errorf("Retryable = true, want false")
	}
}

func TestStep_TransportError_Timeout(t *testing.T) {
	// Use a server we never call: cancel ctx before the request.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never be hit.
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(ctx, &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeTimeout {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeTimeout)
	}
}

func TestStep_RequestBody_SystemPromptTopLevel(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:        "claude-sonnet-4-5",
		SystemPrompt: "You are concise.",
		Messages: []ai.Message{
			// Include a system-role message which MUST be skipped from
			// the messages array (Anthropic forbids it there).
			{Role: "system", Content: "ignored-extra"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}

	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if sent.System != "You are concise." {
		t.Errorf("system = %q, want %q", sent.System, "You are concise.")
	}
	for _, m := range sent.Messages {
		if m.Role == "system" {
			t.Errorf("messages contains a system role; must be top-level only: %+v", sent.Messages)
		}
	}
}

func TestStep_RequestBody_ToolInputAsObject(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "do it"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{
						ID:        "toolu_obj",
						Name:      "foo",
						Arguments: `{"name":"foo"}`,
					},
				},
			},
			{Role: "tool", ToolCallID: "toolu_obj", Content: "done"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}

	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	// The assistant message should be index 1.
	if len(sent.Messages) != 3 {
		t.Fatalf("sent messages len = %d, want 3", len(sent.Messages))
	}
	asst := sent.Messages[1]
	if asst.Role != "assistant" {
		t.Fatalf("messages[1] role = %q, want assistant", asst.Role)
	}
	var blocks []stepReqContentBlock
	if err := json.Unmarshal(asst.Content, &blocks); err != nil {
		t.Fatalf("assistant content not an array: %v (%s)", err, string(asst.Content))
	}
	// Find the tool_use block.
	var found *stepReqContentBlock
	for i := range blocks {
		if blocks[i].Type == "tool_use" {
			found = &blocks[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("no tool_use block in assistant content: %+v", blocks)
	}
	// Critical: input must be a JSON object, not a JSON string.
	// json.RawMessage preserves the raw bytes, so we can inspect.
	if len(found.Input) == 0 {
		t.Fatalf("tool_use input is empty")
	}
	if found.Input[0] != '{' {
		t.Errorf("tool_use input not an object: %s", string(found.Input))
	}
	var inputAsString string
	if err := json.Unmarshal(found.Input, &inputAsString); err == nil {
		t.Errorf("tool_use input decoded as string %q, want object", inputAsString)
	}
	var inputAsMap map[string]any
	if err := json.Unmarshal(found.Input, &inputAsMap); err != nil {
		t.Fatalf("tool_use input not a JSON object: %v", err)
	}
	if inputAsMap["name"] != "foo" {
		t.Errorf("tool_use input.name = %v, want foo", inputAsMap["name"])
	}
}

// --- additional coverage tests ---

func TestStep_DefaultMaxTokensAndTemperature(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:       "claude-sonnet-4-5",
		Messages:    []ai.Message{{Role: "user", Content: "hi"}},
		Temperature: 0.7,
		// MaxTokens unset → must default to 4096
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if sent.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096 (default)", sent.MaxTokens)
	}
	if sent.Temperature == nil {
		t.Errorf("Temperature missing in body, want 0.7 emitted")
	} else if *sent.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", *sent.Temperature)
	}
}

func TestStep_ToolsSerialization(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		Tools: []ai.ToolSchema{
			{
				Name:        "read_doc",
				Description: "Read a doc",
				Parameters:  `{"type":"object","properties":{"name":{"type":"string"}}}`,
			},
			{
				Name: "no_schema",
				// Empty Parameters → adapter must default to {"type":"object"}
			},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(sent.Tools))
	}
	if sent.Tools[0].Name != "read_doc" || sent.Tools[0].Description != "Read a doc" {
		t.Errorf("tools[0] = %+v", sent.Tools[0])
	}
	// input_schema must be a JSON object on the wire.
	if len(sent.Tools[0].InputSchema) == 0 || sent.Tools[0].InputSchema[0] != '{' {
		t.Errorf("tools[0].input_schema not an object: %s", string(sent.Tools[0].InputSchema))
	}
	// Empty Parameters → default {"type":"object"}.
	var defSchema map[string]any
	if err := json.Unmarshal(sent.Tools[1].InputSchema, &defSchema); err != nil {
		t.Fatalf("tools[1].input_schema not JSON: %v", err)
	}
	if defSchema["type"] != "object" {
		t.Errorf("tools[1].input_schema = %v, want type=object default", defSchema)
	}
}

func TestStep_InvalidToolArguments(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{ID: "id1", Name: "x", Arguments: "not-json"},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("Step() expected error for malformed arguments, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
}

func TestStep_InvalidToolParameters(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		Tools: []ai.ToolSchema{
			{Name: "bad", Parameters: "not-json"},
		},
	})
	if err == nil {
		t.Fatal("Step() expected error for malformed tool parameters, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
}

func TestStep_UnknownRole(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "moderator", Content: "no such role"}},
	})
	if err == nil {
		t.Fatal("Step() expected error for unknown role, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
}

func TestStep_EmptyToolCallArguments(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "go"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{ID: "tid", Name: "noargs", Arguments: ""},
				},
			},
			{Role: "tool", ToolCallID: "tid", Content: "done"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	asst := sent.Messages[1]
	var blocks []stepReqContentBlock
	if err := json.Unmarshal(asst.Content, &blocks); err != nil {
		t.Fatalf("assistant content not array: %v", err)
	}
	for _, b := range blocks {
		if b.Type == "tool_use" {
			if string(b.Input) != "{}" {
				t.Errorf("empty Arguments → input = %s, want {}", string(b.Input))
			}
		}
	}
}

func TestStep_StopReason_Stop_Sequence_And_Unknown(t *testing.T) {
	tests := []struct {
		stopReason string
		wantFinish string
	}{
		{"stop_sequence", "stop"},
		{"weird_unrecognized_value", "error"},
	}
	for _, tt := range tests {
		t.Run(tt.stopReason, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := messagesResponse{
					Content: []contentBlock{
						{Type: "text", Text: "x"},
					},
					StopReason: tt.stopReason,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			resp, err := client.Step(context.Background(), &ai.Request{
				Model:    "claude-sonnet-4-5",
				Messages: []ai.Message{{Role: "user", Content: "x"}},
			})
			if err != nil {
				t.Fatalf("Step() error = %v", err)
			}
			if resp.FinishReason != tt.wantFinish {
				t.Errorf("FinishReason = %q, want %q", resp.FinishReason, tt.wantFinish)
			}
		})
	}
}

func TestStep_UserWithToolCallID_AsToolResult(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			// User role with non-empty ToolCallID — must be emitted as a
			// tool_result content array (same shape as Role="tool").
			{Role: "user", ToolCallID: "toolu_user_route", Content: "result-payload"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(sent.Messages))
	}
	var blocks []stepReqContentBlock
	if err := json.Unmarshal(sent.Messages[0].Content, &blocks); err != nil {
		t.Fatalf("user-with-ToolCallID content not array: %v (%s)",
			err, string(sent.Messages[0].Content))
	}
	if len(blocks) != 1 || blocks[0].Type != "tool_result" {
		t.Fatalf("blocks = %+v, want a single tool_result block", blocks)
	}
	if blocks[0].ToolUseID != "toolu_user_route" {
		t.Errorf("tool_use_id = %q, want toolu_user_route", blocks[0].ToolUseID)
	}
}

func TestStep_AssistantPlainTextNoTools(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "claude-sonnet-4-5",
		Messages: []ai.Message{
			{Role: "user", Content: "hi"},
			// Assistant with no ToolCalls — must emit content as plain string.
			{Role: "assistant", Content: "I am thinking..."},
			{Role: "user", Content: "go on"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(sent.Messages))
	}
	asst := sent.Messages[1]
	if asst.Role != "assistant" {
		t.Fatalf("messages[1] role = %q, want assistant", asst.Role)
	}
	// Content must be a JSON string (starts with `"`), not an array.
	if len(asst.Content) == 0 || asst.Content[0] != '"' {
		t.Errorf("plain assistant content not a JSON string: %s", string(asst.Content))
	}
	var asString string
	if err := json.Unmarshal(asst.Content, &asString); err != nil {
		t.Errorf("assistant content not a string: %v", err)
	} else if asString != "I am thinking..." {
		t.Errorf("assistant content = %q, want %q", asString, "I am thinking...")
	}
}

func TestStep_MultipleTextBlocksJoined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: "part one"},
				{Type: "text", Text: "part two"},
			},
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.Text != "part one\npart two" {
		t.Errorf("Text = %q, want %q (newline-joined)", resp.Text, "part one\npart two")
	}
}

func TestStep_MalformedJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error for malformed JSON, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeProtocolError {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeProtocolError)
	}
}
