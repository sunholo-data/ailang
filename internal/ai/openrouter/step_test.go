package openrouter

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

// stepReqBody mirrors the wire shape Step sends to OpenRouter so tests can
// inspect what was actually serialized. OpenRouter speaks the OpenAI Chat
// Completions wire format with one extension: a top-level `provider` field
// that translates the AIRoutingPolicy.
type stepReqBody struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Messages    []stepReqMessage `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
	Tools       []stepReqToolDef `json:"tools,omitempty"`
	Provider    *providerField   `json:"provider,omitempty"`
}

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
// canned response. Used by tests that need to inspect outbound wire shape.
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
		"id":"chatcmpl-x","object":"chat.completion","model":"anthropic/claude-sonnet-4.5",
		"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
		"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
	}`))
}

func writeCannedChatResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

// TestStep_TextOnly_HappyPath — single text response, finish_reason "stop".
func TestStep_TextOnly_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-abc","object":"chat.completion","model":"anthropic/claude-sonnet-4.5",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Hello there!"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":12,"completion_tokens":7,"total_tokens":19}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.Text != "Hello there!" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello there!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", resp.FinishReason)
	}
	if resp.InputTokens != 12 || resp.OutputTokens != 7 || resp.TotalTokens != 19 {
		t.Errorf("tokens in/out/total = %d/%d/%d, want 12/7/19",
			resp.InputTokens, resp.OutputTokens, resp.TotalTokens)
	}
	if resp.Model != "anthropic/claude-sonnet-4.5" {
		t.Errorf("Model = %q, want anthropic/claude-sonnet-4.5", resp.Model)
	}
}

// TestStep_SingleToolCall_PassthroughFormat — verify the OpenAI Chat
// Completions tool-use format passes through unchanged via OpenRouter.
func TestStep_SingleToolCall_PassthroughFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"chatcmpl-tool","object":"chat.completion","model":"anthropic/claude-sonnet-4.5",
			"choices":[{"index":0,"message":{"role":"assistant","content":null,
				"tool_calls":[{"id":"call_or","type":"function","function":{"name":"read_doc","arguments":"{\"name\":\"nda.docx\"}"}}]
			},"finish_reason":"tool_calls"}],
			"usage":{"prompt_tokens":50,"completion_tokens":25,"total_tokens":75}
		}`)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "Read nda.docx"}},
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
		t.Errorf("FinishReason = %q, want tool_calls", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_or" || tc.Name != "read_doc" {
		t.Errorf("tool call = %+v", tc)
	}
	if tc.Arguments != `{"name":"nda.docx"}` {
		t.Errorf("Arguments = %q, want verbatim JSON string", tc.Arguments)
	}
}

// TestStep_RoutingPolicy_AppliesToBody — verify the body has the expected
// `provider` field when Routing is set, with the same translation as Generate.
func TestStep_RoutingPolicy_AppliesToBody(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"anthropic", "openai"},
			AllowFallback: true,
			Require:       []ai.AICapability{ai.CapStructuredOutputs},
			Prefer:        ai.PreferCheapest,
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if sent.Provider == nil {
		t.Fatal("provider field missing in body, want non-nil")
	}
	if len(sent.Provider.Order) != 2 || sent.Provider.Order[0] != "anthropic" || sent.Provider.Order[1] != "openai" {
		t.Errorf("Order = %v, want [anthropic, openai]", sent.Provider.Order)
	}
	if sent.Provider.AllowFallbacks == nil || !*sent.Provider.AllowFallbacks {
		t.Errorf("AllowFallbacks = %v, want true", sent.Provider.AllowFallbacks)
	}
	if len(sent.Provider.RequireParameters) != 1 || sent.Provider.RequireParameters[0] != "structured_outputs" {
		t.Errorf("RequireParameters = %v, want [structured_outputs]", sent.Provider.RequireParameters)
	}
	if sent.Provider.Sort != "price" {
		t.Errorf("Sort = %q, want price", sent.Provider.Sort)
	}
}

// TestStep_RoutingPolicy_PlusTools_Composes — set BOTH Routing and Tools.
// Both must end up in the body correctly.
func TestStep_RoutingPolicy_PlusTools_Composes(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "Use the tool"}},
		Tools: []ai.ToolSchema{
			{
				Name:        "read_doc",
				Description: "Read",
				Parameters:  `{"type":"object","properties":{"n":{"type":"string"}}}`,
			},
		},
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"anthropic"},
			AllowFallback: false,
		},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if len(sent.Tools) != 1 {
		t.Fatalf("tools len = %d, want 1 (Tools must be present alongside Routing)", len(sent.Tools))
	}
	if sent.Tools[0].Function.Name != "read_doc" {
		t.Errorf("tools[0].name = %q", sent.Tools[0].Function.Name)
	}
	if sent.Provider == nil {
		t.Fatal("provider field missing; routing must be present alongside tools")
	}
	if len(sent.Provider.Order) != 1 || sent.Provider.Order[0] != "anthropic" {
		t.Errorf("Order = %v, want [anthropic]", sent.Provider.Order)
	}
	if sent.Provider.AllowFallbacks == nil || *sent.Provider.AllowFallbacks {
		t.Errorf("AllowFallbacks = %v, want explicit false", sent.Provider.AllowFallbacks)
	}
}

// TestStep_AuthFailed_401 — provider is "openrouter" in error.
func TestStep_AuthFailed_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"message":"User not found","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
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

// TestStep_RateLimit_429 — same shape as OpenAI, verifies provider="openrouter"
// is used in the classification path (the resulting AIError doesn't carry
// provider, but the inner-message hoisting must still work).
func TestStep_RateLimit_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
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
	if !strings.Contains(strings.ToLower(aiErr.Message), "rate limit") {
		t.Errorf("Message = %q, want it to contain 'rate limit' (hoisted)", aiErr.Message)
	}
}

// TestStep_HTTPReferer_XTitle_Headers — verify HTTP-Referer and X-Title are
// set on the request when the env vars are present.
func TestStep_HTTPReferer_XTitle_Headers(t *testing.T) {
	t.Setenv("OPENROUTER_HTTP_REFERER", "https://ailang.example")
	t.Setenv("OPENROUTER_X_TITLE", "ailang-test")

	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if got := cap.headers.Get("HTTP-Referer"); got != "https://ailang.example" {
		t.Errorf("HTTP-Referer = %q, want https://ailang.example", got)
	}
	if got := cap.headers.Get("X-Title"); got != "ailang-test" {
		t.Errorf("X-Title = %q, want ailang-test", got)
	}
	if got := cap.headers.Get("Authorization"); got != "Bearer k" {
		t.Errorf("Authorization = %q, want Bearer k", got)
	}
}

// --- additional coverage tests ---

// TestStep_NoProviderField_WhenNoRouting confirms that without Routing the
// outbound body has no `provider` field (back-compat fast path).
func TestStep_NoProviderField_WhenNoRouting(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	var sent stepReqBody
	if err := json.Unmarshal(cap.captured, &sent); err != nil {
		t.Fatalf("captured body not JSON: %v", err)
	}
	if sent.Provider != nil {
		t.Errorf("Provider field present when no routing: %+v", sent.Provider)
	}
}

// TestStep_5xx_Internal — 5xx maps to CodeInternal (retryable).
func TestStep_5xx_Internal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`Service Unavailable`))
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
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

// TestStep_TransportError — pre-cancelled context surfaces as CodeTimeout.
func TestStep_TransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(ctx, &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
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

// TestStep_MalformedJSONResponse covers the parse-error path.
func TestStep_MalformedJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeProtocolError {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeProtocolError)
	}
}

// TestStep_RequestedModel_SetEvenWhenServerEchosDifferent confirms that
// resp.RequestedModel mirrors req.Model so callers can detect routing
// behavior end-to-end.
func TestStep_RequestedModel_SetEvenWhenServerEchosDifferent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"x","object":"chat.completion","model":"anthropic/claude-sonnet-4.5-actual",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.RequestedModel != "anthropic/claude-sonnet-4.5" {
		t.Errorf("RequestedModel = %q, want %q", resp.RequestedModel, "anthropic/claude-sonnet-4.5")
	}
	if resp.Model != "anthropic/claude-sonnet-4.5-actual" {
		t.Errorf("Model = %q, want server-reported value", resp.Model)
	}
}

// TestStep_InvalidToolParameters_ShortCircuits ensures translation errors
// surface from the openai helper without an HTTP call.
func TestStep_InvalidToolParameters_ShortCircuits(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Tools: []ai.ToolSchema{
			{Name: "bad", Parameters: "not-json"},
		},
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
}
