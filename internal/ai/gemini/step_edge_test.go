package gemini

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestStep_RejectsRoutingPolicy ensures Step refuses an AIRoutingPolicy
// (only OpenRouter accepts it).
func TestStep_RejectsRoutingPolicy(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"google"},
			AllowFallback: true,
		},
	})
	if err == nil {
		t.Fatal("expected error rejecting routing policy, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeCapabilityNotSupported {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeCapabilityNotSupported)
	}
}

// TestStep_BadAssistantToolArgs: assistant ToolCall with invalid JSON args
// surfaces as a CodeProtocolError before the HTTP call.
func TestStep_BadAssistantToolArgs(t *testing.T) {
	client := NewClient("test-key", WithBaseURL("http://127.0.0.1:1")) // never reached
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "x"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{{
					ID:        "x1",
					Name:      "foo",
					Arguments: `{not valid json`,
				}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for bad tool args, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeProtocolError {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeProtocolError)
	}
}

// TestStep_BadToolParametersSchema: req.Tools entry with malformed JSON
// Schema in Parameters surfaces as a CodeSchemaValidation error.
func TestStep_BadToolParametersSchema(t *testing.T) {
	client := NewClient("test-key", WithBaseURL("http://127.0.0.1:1")) // never reached
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
		Tools: []ai.ToolSchema{{
			Name:       "broken",
			Parameters: `{not valid json`,
		}},
	})
	if err == nil {
		t.Fatal("expected error for bad tool schema, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeSchemaValidation {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeSchemaValidation)
	}
}

// TestStep_UserToolResult_Roundtrip: a Role="user" message with a
// non-empty ToolCallID is treated as a tool result (rare but supported).
func TestStep_UserToolResult_Roundtrip(t *testing.T) {
	captured := &stepRequest{}
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "ack"}}},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, captured, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "do it"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{{
					ID:        "u1",
					Name:      "compute",
					Arguments: `{"x":1}`,
				}},
			},
			// User-role with ToolCallID — should be treated as a tool result.
			{Role: "user", ToolCallID: "u1", Content: "42"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	var fr *stepFunctionResponse
	for _, c := range captured.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil {
				fr = p.FunctionResponse
			}
		}
	}
	if fr == nil {
		t.Fatalf("no functionResponse in captured request: %+v", captured)
	}
	if fr.Name != "compute" {
		t.Errorf("functionResponse.name = %q, want compute", fr.Name)
	}
	if fr.Response["content"] != "42" {
		t.Errorf("functionResponse.response.content = %v, want 42", fr.Response["content"])
	}
}

// TestStep_ToolResult_OrphanFallsBackToText: Role="tool" with a
// ToolCallID that doesn't match any prior assistant ToolCall falls back
// to plain user text (no functionResponse).
func TestStep_ToolResult_OrphanFallsBackToText(t *testing.T) {
	captured := &stepRequest{}
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "ok"}}},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, captured, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "hi"},
			{Role: "tool", ToolCallID: "no-such-id", Content: "orphan body"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	for _, c := range captured.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil {
				t.Errorf("orphan tool result emitted functionResponse: %+v", p.FunctionResponse)
			}
		}
	}
	// Expect the orphan content as plain text on a user message.
	found := false
	for _, c := range captured.Contents {
		for _, p := range c.Parts {
			if p.Text == "orphan body" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("orphan content not surfaced as text; contents=%+v", captured.Contents)
	}
}

// TestStep_AssistantEmptyContent_NoToolCalls: assistant message with no
// content and no tool calls still emits a model content with at least one
// part (Gemini rejects empty parts arrays).
func TestStep_AssistantEmptyContent_NoToolCalls(t *testing.T) {
	captured := &stepRequest{}
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "ok"}}},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, captured, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "x"},
			{Role: "assistant"}, // empty content, no tool calls
			{Role: "user", Content: "again"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	// Find the model entry and ensure parts is non-empty.
	for _, c := range captured.Contents {
		if c.Role == "model" {
			if len(c.Parts) == 0 {
				t.Errorf("model entry has zero parts; Gemini will reject")
			}
		}
	}
}

// TestStep_GenerationConfig_OmittedWhenZero: MaxTokens=0, Temperature=0
// → no GenerationConfig in body.
func TestStep_GenerationConfig_OmittedWhenZero(t *testing.T) {
	captured := &stepRequest{}
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "ok"}}},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, captured, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if captured.GenerationConfig != nil {
		t.Errorf("GenerationConfig non-nil with zero values: %+v", captured.GenerationConfig)
	}
}

// TestStep_GenerationConfig_PassedThrough: MaxTokens + Temperature → set.
func TestStep_GenerationConfig_PassedThrough(t *testing.T) {
	captured := &stepRequest{}
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "ok"}}},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, captured, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:       "gemini-3-flash",
		Messages:    []ai.Message{{Role: "user", Content: "hi"}},
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if captured.GenerationConfig == nil {
		t.Fatal("GenerationConfig is nil; expected non-nil")
	}
	if captured.GenerationConfig.MaxOutputTokens != 2048 {
		t.Errorf("MaxOutputTokens = %d, want 2048", captured.GenerationConfig.MaxOutputTokens)
	}
	if captured.GenerationConfig.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", captured.GenerationConfig.Temperature)
	}
}

// TestStep_ModelFallback_WhenModelVersionMissing: response with no
// modelVersion field falls back to req.Model.
func TestStep_ModelFallback_WhenModelVersionMissing(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "hi"}}},
			FinishReason: "STOP",
		}},
		// No ModelVersion.
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.Model != "gemini-3-flash" {
		t.Errorf("Model fallback = %q, want %q", out.Model, "gemini-3-flash")
	}
}

// TestStep_FinishReason_AllErrorVariants: every Gemini error finishReason
// maps to "error" through mapFinishReason.
func TestStep_FinishReason_AllErrorVariants(t *testing.T) {
	cases := []string{"RECITATION", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII", "UNKNOWN_REASON"}
	for _, reason := range cases {
		t.Run(reason, func(t *testing.T) {
			resp := stepResponseBody{
				Candidates: []stepCandidate{{
					Content:      stepContent{Role: "model", Parts: []stepPart{{Text: ""}}},
					FinishReason: reason,
				}},
			}
			server := newStepServer(t, nil, resp, 0)
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			out, err := client.Step(context.Background(), &ai.Request{
				Model:    "gemini-3-flash",
				Messages: []ai.Message{{Role: "user", Content: "x"}},
			})
			if err != nil {
				t.Fatalf("Step error = %v", err)
			}
			if out.FinishReason != "error" {
				t.Errorf("reason=%q FinishReason = %q, want error", reason, out.FinishReason)
			}
		})
	}
}

// TestStep_FinishReason_EmptyTreatedAsStop: defensive — empty finishReason
// (some streaming-mid responses) maps to "stop".
func TestStep_FinishReason_EmptyTreatedAsStop(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content: stepContent{Role: "model", Parts: []stepPart{{Text: "x"}}},
			// No FinishReason.
		}},
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want stop", out.FinishReason)
	}
}

// TestStep_BadResponseJSON: malformed response body → ProtocolError.
func TestStep_BadResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeProtocolError {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeProtocolError)
	}
}
