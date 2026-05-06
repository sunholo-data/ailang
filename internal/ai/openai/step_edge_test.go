package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestStep_RoutingPolicyRejected verifies that a routing policy with
// HasRouting()==true is rejected with CodeCapabilityNotSupported.
func TestStep_RoutingPolicyRejected(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()
	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Routing:  &ai.AIRoutingPolicy{Order: []string{"openai"}, AllowFallback: true},
	})
	if err == nil {
		t.Fatal("Step() expected error for routing policy, got nil")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeCapabilityNotSupported {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeCapabilityNotSupported)
	}
}

// TestStep_AssistantWithTextAndToolCalls covers the path where an assistant
// message has BOTH content and tool_calls — content should be the string,
// not null.
func TestStep_AssistantWithTextAndToolCalls(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
		Messages: []ai.Message{
			{Role: "user", Content: "go"},
			{
				Role:    "assistant",
				Content: "Let me check the doc.",
				ToolCalls: []ai.ToolCall{
					{ID: "call_t", Name: "read_doc", Arguments: `{"name":"x"}`},
				},
			},
			{Role: "tool", ToolCallID: "call_t", Content: "done"},
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
	var s string
	if err := json.Unmarshal(asst.Content, &s); err != nil {
		t.Fatalf("assistant content not a string: %v (%s)", err, string(asst.Content))
	}
	if s != "Let me check the doc." {
		t.Errorf("assistant content = %q, want the text string (not null)", s)
	}
}

// TestStep_UserWithToolCallID_AsTool covers the user-with-ToolCallID edge
// case (treated as a tool result).
func TestStep_UserWithToolCallID_AsTool(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
		Messages: []ai.Message{
			{Role: "user", ToolCallID: "call_route", Content: "the result"},
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
	if sent.Messages[0].Role != "tool" {
		t.Errorf("role = %q, want tool", sent.Messages[0].Role)
	}
	if sent.Messages[0].ToolCallID != "call_route" {
		t.Errorf("tool_call_id = %q, want call_route", sent.Messages[0].ToolCallID)
	}
}

// TestStep_ToolsSerialization confirms tools array shape and the empty-
// Parameters default.
func TestStep_ToolsSerialization(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Tools: []ai.ToolSchema{
			{
				Name:        "read_doc",
				Description: "Read a doc",
				Parameters:  `{"type":"object","properties":{"name":{"type":"string"}}}`,
			},
			{
				Name: "no_schema",
				// Empty Parameters → adapter must default to a permissive schema.
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
	if sent.Tools[0].Type != "function" {
		t.Errorf("tools[0].type = %q, want function", sent.Tools[0].Type)
	}
	if sent.Tools[0].Function.Name != "read_doc" {
		t.Errorf("tools[0].name = %q, want read_doc", sent.Tools[0].Function.Name)
	}
	// parameters must be a JSON object on the wire.
	if len(sent.Tools[0].Function.Parameters) == 0 || sent.Tools[0].Function.Parameters[0] != '{' {
		t.Errorf("tools[0].parameters not an object: %s", string(sent.Tools[0].Function.Parameters))
	}
	// Empty Parameters → default object schema.
	var defSchema map[string]any
	if err := json.Unmarshal(sent.Tools[1].Function.Parameters, &defSchema); err != nil {
		t.Fatalf("tools[1].parameters not JSON: %v", err)
	}
	if defSchema["type"] != "object" {
		t.Errorf("tools[1].parameters = %v, want type=object default", defSchema)
	}
}

// TestStep_InvalidToolParameters checks that malformed JSON in a tool's
// Parameters surfaces as an *ai.AIError without an HTTP call.
func TestStep_InvalidToolParameters(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
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

// TestStep_UnknownRole verifies an unknown message role is rejected.
func TestStep_UnknownRole(t *testing.T) {
	server := httptest.NewServer(&captureHandler{})
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "moderator", Content: "x"}},
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

// TestStep_EmptyToolCallArgumentsDefaultsToObject covers the empty-Arguments
// path: empty string → "{}" on the wire.
func TestStep_EmptyToolCallArgumentsDefaultsToObject(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gpt-5",
		Messages: []ai.Message{
			{Role: "user", Content: "x"},
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
	if len(asst.ToolCalls) != 1 {
		t.Fatalf("tool_calls len = %d, want 1", len(asst.ToolCalls))
	}
	var s string
	if err := json.Unmarshal(asst.ToolCalls[0].Function.Arguments, &s); err != nil {
		t.Fatalf("arguments not a JSON string: %v", err)
	}
	if s != "{}" {
		t.Errorf("empty Arguments → wire string = %q, want %q", s, "{}")
	}
}

// TestStep_MaxTokensAndTemperatureForwarded verifies non-zero MaxTokens and
// Temperature serialise into the request body. GPT-5+ and o-series models
// reject "max_tokens" with HTTP 400 ("Unsupported parameter; use
// 'max_completion_tokens' instead"), so BuildChatStepRequest routes the
// value to MaxCompletionTokens for those models. This test pins both
// branches: a legacy model (gpt-4o) → max_tokens, a reasoning model
// (gpt-5) → max_completion_tokens. Regression test for an issue surfaced
// by the M9 motoko_agent provider matrix run on 2026-05-06.
func TestStep_MaxTokensAndTemperatureForwarded(t *testing.T) {
	cases := []struct {
		name          string
		model         string
		wantLegacy    int // expected max_tokens
		wantReasoning int // expected max_completion_tokens
	}{
		{name: "legacy gpt-4o uses max_tokens", model: "gpt-4o", wantLegacy: 2048, wantReasoning: 0},
		{name: "reasoning gpt-5 uses max_completion_tokens", model: "gpt-5", wantLegacy: 0, wantReasoning: 2048},
		{name: "reasoning gpt-5-mini uses max_completion_tokens", model: "gpt-5-mini", wantLegacy: 0, wantReasoning: 2048},
		{name: "reasoning o1-preview uses max_completion_tokens", model: "o1-preview", wantLegacy: 0, wantReasoning: 2048},
		{name: "reasoning o3-mini uses max_completion_tokens", model: "o3-mini", wantLegacy: 0, wantReasoning: 2048},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cap := &captureHandler{}
			server := httptest.NewServer(cap)
			defer server.Close()

			client := NewClient("k", WithBaseURL(server.URL))
			_, err := client.Step(context.Background(), &ai.Request{
				Model:       tc.model,
				Messages:    []ai.Message{{Role: "user", Content: "x"}},
				MaxTokens:   2048,
				Temperature: 0.42,
			})
			if err != nil {
				t.Fatalf("Step() error = %v", err)
			}
			var sent stepReqBody
			if err := json.Unmarshal(cap.captured, &sent); err != nil {
				t.Fatalf("captured body not JSON: %v", err)
			}
			if sent.MaxTokens != tc.wantLegacy {
				t.Errorf("max_tokens = %d, want %d", sent.MaxTokens, tc.wantLegacy)
			}
			if sent.MaxCompletionTokens != tc.wantReasoning {
				t.Errorf("max_completion_tokens = %d, want %d", sent.MaxCompletionTokens, tc.wantReasoning)
			}
			if sent.Temperature == nil || *sent.Temperature != 0.42 {
				t.Errorf("temperature = %v, want 0.42", sent.Temperature)
			}
		})
	}
}

// TestStep_AuthHeaderSet verifies the Bearer auth header lands on the request.
func TestStep_AuthHeaderSet(t *testing.T) {
	cap := &captureHandler{}
	server := httptest.NewServer(cap)
	defer server.Close()

	client := NewClient("sk-secret", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if got := cap.headers.Get("Authorization"); got != "Bearer sk-secret" {
		t.Errorf("Authorization = %q, want Bearer sk-secret", got)
	}
	if got := cap.headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
}

// TestStep_MalformedJSONResponse covers the parse-error path on a 200 body
// that isn't JSON.
func TestStep_MalformedJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
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

// TestStep_FinishReason_Unknown maps an unrecognised finish_reason to "error".
func TestStep_FinishReason_Unknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeCannedChatResponse(w, `{
			"id":"x","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":"x"},"finish_reason":"weird_thing"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`)
	}))
	defer server.Close()

	client := NewClient("k", WithBaseURL(server.URL))
	resp, err := client.Step(context.Background(), &ai.Request{
		Model:    "gpt-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if resp.FinishReason != "error" {
		t.Errorf("FinishReason = %q, want error (unknown maps to error)", resp.FinishReason)
	}
}
