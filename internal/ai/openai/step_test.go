package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// stepReqBody mirrors the wire shape Step sends to OpenAI Chat Completions
// so tests can inspect what was actually serialized.
type stepReqBody struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Messages    []stepReqMessage `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
	Tools       []stepReqToolDef `json:"tools,omitempty"`
}

// stepReqMessage uses json.RawMessage for content so tests can verify the
// JSON null vs string distinction (assistant with tool_calls must emit null).
type stepReqMessage struct {
	Role       string            `json:"role"`
	Content    json.RawMessage   `json:"content"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	ToolCalls  []stepReqToolCall `json:"tool_calls,omitempty"`
}

type stepReqToolCall struct {
	ID       string                  `json:"id"`
	Type     string                  `json:"type"`
	Function stepReqToolCallFunction `json:"function"`
}

type stepReqToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type stepReqToolDef struct {
	Type     string                 `json:"type"`
	Function stepReqToolDefFunction `json:"function"`
}

type stepReqToolDefFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// captureHandler records the last request body it sees and serves a
// canned response.
type captureHandler struct {
	captured []byte
	headers  http.Header
	respond  func(w http.ResponseWriter, r *http.Request)
}

func (h *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	h.captured = body
	h.headers = r.Header.Clone()
	if h.respond != nil {
		h.respond(w, r)
		return
	}
	// Default: empty success
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{
		"id":"chatcmpl-x","object":"chat.completion","model":"gpt-5",
		"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
	}`))
}

// canned successful chat completion response.
func writeCannedChatResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func TestStep_TextOnly_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-abc","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Hello there!"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":12,"completion_tokens":7,"total_tokens":19}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
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
	if resp.Model != "gpt-5" {
		t.Errorf("Model = %q, want gpt-5", resp.Model)
	}
}

func TestStep_SingleToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-tool","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":null,
				"tool_calls":[{"id":"call_abc","type":"function","function":{"name":"read_doc","arguments":"{\"name\":\"nda.docx\"}"}}]
			},"finish_reason":"tool_calls"}],
			"usage":{"prompt_tokens":50,"completion_tokens":25,"total_tokens":75}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
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
	if tc.ID != "call_abc" {
		t.Errorf("ToolCalls[0].ID = %q, want call_abc", tc.ID)
	}
	if tc.Name != "read_doc" {
		t.Errorf("ToolCalls[0].Name = %q, want read_doc", tc.Name)
	}
	// Arguments must be passed through verbatim as a JSON STRING (containing JSON).
	if tc.Arguments != `{"name":"nda.docx"}` {
		t.Errorf("ToolCalls[0].Arguments = %q, want verbatim JSON string %q",
			tc.Arguments, `{"name":"nda.docx"}`)
	}
}

func TestStep_MultipleToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-multi","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":null,
				"tool_calls":[
					{"id":"call_1","type":"function","function":{"name":"read_doc","arguments":"{\"name\":\"a.txt\"}"}},
					{"id":"call_2","type":"function","function":{"name":"read_doc","arguments":"{\"name\":\"b.txt\"}"}}
				]
			},"finish_reason":"tool_calls"}],
			"usage":{"prompt_tokens":50,"completion_tokens":25,"total_tokens":75}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "Read a and b"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if len(resp.ToolCalls) != 2 {
		t.Fatalf("ToolCalls len = %d, want 2", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_1" || resp.ToolCalls[1].ID != "call_2" {
		t.Errorf("ToolCalls IDs = %q,%q want call_1,call_2",
			resp.ToolCalls[0].ID, resp.ToolCalls[1].ID)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls", resp.FinishReason)
	}
}

func TestStep_ArgumentsAreStringNotObject(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
		Messages: []ai.Message{
			{Role: "user", Content: "go"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{ID: "call_abc", Name: "read_doc", Arguments: `{"name":"nda.docx"}`},
				},
			},
			{Role: "tool", ToolCallID: "call_abc", Content: "done"},
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
	if len(asst.ToolCalls) != 1 {
		t.Fatalf("assistant tool_calls len = %d, want 1", len(asst.ToolCalls))
	}
	args := asst.ToolCalls[0].Function.Arguments
	// Critical: arguments must be a JSON STRING (starts with `"`), not an object.
	if len(args) == 0 || args[0] != '"' {
		t.Errorf("arguments not a JSON string: %s", string(args))
	}
	var s string
	if err := json.Unmarshal(args, &s); err != nil {
		t.Fatalf("arguments could not decode as string: %v (%s)", err, string(args))
	}
	if s != `{"name":"nda.docx"}` {
		t.Errorf("arguments string content = %q, want verbatim %q",
			s, `{"name":"nda.docx"}`)
	}
}

func TestStep_ToolResultRoundtrip(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
		Messages: []ai.Message{
			{Role: "user", Content: "Read it"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{
					{ID: "call_xyz", Name: "read_doc", Arguments: `{"name":"x"}`},
				},
			},
			{Role: "tool", ToolCallID: "call_xyz", Content: "doc body"},
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
	tool := sent.Messages[2]
	if tool.Role != "tool" {
		t.Errorf("messages[2] role = %q, want tool", tool.Role)
	}
	if tool.ToolCallID != "call_xyz" {
		t.Errorf("messages[2] tool_call_id = %q, want call_xyz", tool.ToolCallID)
	}
	var contentStr string
	if err := json.Unmarshal(tool.Content, &contentStr); err != nil {
		t.Fatalf("tool content not a string: %v", err)
	}
	if contentStr != "doc body" {
		t.Errorf("tool content = %q, want doc body", contentStr)
	}
}

func TestStep_SystemPromptInMessages(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:        "gpt-5",
		SystemPrompt: "You are concise.",
		Messages: []ai.Message{
			{Role: "user", Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2 (system+user)", len(sent.Messages))
	}
	if sent.Messages[0].Role != "system" {
		t.Errorf("messages[0] role = %q, want system", sent.Messages[0].Role)
	}
	var contentStr string
	if err := json.Unmarshal(sent.Messages[0].Content, &contentStr); err != nil {
		t.Fatalf("system content not a string: %v", err)
	}
	if contentStr != "You are concise." {
		t.Errorf("system content = %q, want %q", contentStr, "You are concise.")
	}
}

func TestStep_SystemPromptNotDuplicated(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:        "gpt-5",
		SystemPrompt: "ignored",
		Messages: []ai.Message{
			{Role: "system", Content: "messages-system-wins"},
			{Role: "user", Content: "Hi"},
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	systemCount := 0
	for _, m := range sent.Messages {
		if m.Role == "system" {
			systemCount++
		}
	}
	if systemCount != 1 {
		t.Fatalf("system messages count = %d, want 1 (req.Messages wins, SystemPrompt skipped)", systemCount)
	}
	// Verify the surviving system message is the one from req.Messages.
	for _, m := range sent.Messages {
		if m.Role == "system" {
			var s string
			if err := json.Unmarshal(m.Content, &s); err != nil {
				t.Fatalf("system content not a string: %v", err)
			}
			if s != "messages-system-wins" {
				t.Errorf("system content = %q, want messages-system-wins (req.SystemPrompt should be skipped)", s)
			}
		}
	}
}

func TestStep_FinishReason_Length(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-len","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":"truncated..."},"finish_reason":"length"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.FinishReason != "length" {
		t.Errorf("FinishReason = %q, want length", resp.FinishReason)
	}
}

func TestStep_FinishReason_ContentFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-cf","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":""},"finish_reason":"content_filter"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.FinishReason != "error" {
		t.Errorf("FinishReason = %q, want error", resp.FinishReason)
	}
}

func TestStep_FinishReason_LegacyFunctionCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-fc","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":null,
				"tool_calls":[{"id":"call_legacy","type":"function","function":{"name":"f","arguments":"{}"}}]},
				"finish_reason":"function_call"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls (legacy function_call alias)", resp.FinishReason)
	}
}

func TestStep_AuthFailed_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
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
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
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
	// Bonus: hoisted inner message
	if !strings.Contains(strings.ToLower(aiErr.Message), "rate limit") {
		t.Errorf("Message = %q, want it to contain 'rate limit' (hoisted inner error.message)", aiErr.Message)
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
		Model:    "gpt-5",
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
		_, _ = w.Write([]byte(`{"error":{"message":"This model's maximum context length is 8192 tokens","type":"invalid_request_error","code":"context_length_exceeded"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
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

func TestStep_TransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never be hit.
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(ctx, &ai.Request{
		Model:    "gpt-5",
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

func TestStep_EmptyChoices_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-x","object":"chat.completion","model":"gpt-5",
			"choices":[],
			"usage":{"prompt_tokens":1,"completion_tokens":0,"total_tokens":1}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error for empty choices, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	// Either ProtocolError or Internal is sensible — both signal a server-side
	// shape violation.
	if aiErr.Code != ai.CodeProtocolError && aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want CodeProtocolError or CodeInternal", aiErr.Code)
	}
}

func TestStep_AssistantContentNullPlusToolCalls(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
		Messages: []ai.Message{
			{Role: "user", Content: "go"},
			{
				Role: "assistant",
				// Content empty, ToolCalls non-empty → must serialise content as JSON null
				ToolCalls: []ai.ToolCall{
					{ID: "call_n", Name: "f", Arguments: `{}`},
				},
			},
			{Role: "tool", ToolCallID: "call_n", Content: "done"},
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
	if asst.Role != "assistant" {
		t.Fatalf("messages[1] role = %q, want assistant", asst.Role)
	}
	// Content must be the literal JSON `null`.
	contentBytes := strings.TrimSpace(string(asst.Content))
	if contentBytes != "null" {
		t.Errorf("assistant content = %q, want literal JSON null", contentBytes)
	}
	if len(asst.ToolCalls) != 1 {
		t.Fatalf("assistant tool_calls len = %d, want 1", len(asst.ToolCalls))
	}
}
