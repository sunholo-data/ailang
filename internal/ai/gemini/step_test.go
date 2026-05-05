package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// stepRequest mirrors the Gemini generateContent request body shape used
// for function calling. Defined locally for assertions on captured bodies.
type stepRequest struct {
	Contents          []stepContent          `json:"contents"`
	SystemInstruction *stepContent           `json:"systemInstruction,omitempty"`
	Tools             []stepToolDeclarations `json:"tools,omitempty"`
	GenerationConfig  *generationConfig      `json:"generationConfig,omitempty"`
}

type stepContent struct {
	Role  string     `json:"role,omitempty"`
	Parts []stepPart `json:"parts"`
}

type stepPart struct {
	Text             string                `json:"text,omitempty"`
	FunctionCall     *stepFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *stepFunctionResponse `json:"functionResponse,omitempty"`
}

type stepFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type stepFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type stepToolDeclarations struct {
	FunctionDeclarations []stepFunctionDecl `json:"functionDeclarations"`
}

type stepFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// stepResponseBody mirrors the Gemini response shape including functionCall
// parts. Defined locally so we can build fixtures without polluting types.go.
type stepResponseBody struct {
	Candidates    []stepCandidate `json:"candidates"`
	UsageMetadata usageMetadata   `json:"usageMetadata,omitempty"`
	ModelVersion  string          `json:"modelVersion,omitempty"`
}

type stepCandidate struct {
	Content      stepContent `json:"content"`
	FinishReason string      `json:"finishReason"`
	Index        int         `json:"index"`
}

// newStepServer creates a test server. The handler receives the captured
// request body for inspection and writes the supplied response body.
func newStepServer(t *testing.T, capture *stepRequest, response interface{}, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if err := json.Unmarshal(body, capture); err != nil {
				t.Errorf("failed to decode request body: %v\nbody=%s", err, string(body))
			}
		}
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if response != nil {
			switch v := response.(type) {
			case string:
				_, _ = w.Write([]byte(v))
			case []byte:
				_, _ = w.Write(v)
			default:
				_ = json.NewEncoder(w).Encode(v)
			}
		}
	}))
}

// TestStep_TextOnly_HappyPath: single text part, finishReason STOP, no tools.
func TestStep_TextOnly_HappyPath(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content: stepContent{
				Role:  "model",
				Parts: []stepPart{{Text: "Hello there"}},
			},
			FinishReason: "STOP",
		}},
		UsageMetadata: usageMetadata{
			PromptTokenCount: 5, CandidatesTokenCount: 3, TotalTokenCount: 8,
		},
		ModelVersion: "gemini-3-flash-preview",
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "Say hi"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.Text != "Hello there" {
		t.Errorf("Text = %q, want %q", out.Text, "Hello there")
	}
	if out.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "stop")
	}
	if len(out.ToolCalls) != 0 {
		t.Errorf("ToolCalls len = %d, want 0", len(out.ToolCalls))
	}
	if out.InputTokens != 5 || out.OutputTokens != 3 || out.TotalTokens != 8 {
		t.Errorf("token counts wrong: in=%d out=%d total=%d", out.InputTokens, out.OutputTokens, out.TotalTokens)
	}
	if out.Model != "gemini-3-flash-preview" {
		t.Errorf("Model = %q, want %q", out.Model, "gemini-3-flash-preview")
	}
}

// TestStep_SingleFunctionCall: STOP + one functionCall part → tool_calls override.
func TestStep_SingleFunctionCall(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content: stepContent{
				Role: "model",
				Parts: []stepPart{
					{FunctionCall: &stepFunctionCall{
						Name: "read_doc",
						Args: map[string]interface{}{"name": "nda.docx"},
					}},
				},
			},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "Use read_doc on nda.docx"},
		},
		Tools: []ai.ToolSchema{{
			Name:        "read_doc",
			Description: "Read a doc",
			Parameters:  `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`,
		}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(out.ToolCalls))
	}
	tc := out.ToolCalls[0]
	if tc.ID != "0_0" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "0_0")
	}
	if tc.Name != "read_doc" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "read_doc")
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
		t.Fatalf("ToolCall.Arguments invalid JSON: %v (raw=%s)", err, tc.Arguments)
	}
	if args["name"] != "nda.docx" {
		t.Errorf("Arguments name = %v, want nda.docx", args["name"])
	}
}

// TestStep_MultipleFunctionCalls: text + 2 calls; IDs increment.
func TestStep_MultipleFunctionCalls(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content: stepContent{
				Role: "model",
				Parts: []stepPart{
					{Text: "Reading two docs."},
					{FunctionCall: &stepFunctionCall{Name: "read_doc", Args: map[string]interface{}{"name": "a.docx"}}},
					{FunctionCall: &stepFunctionCall{Name: "read_doc", Args: map[string]interface{}{"name": "b.docx"}}},
				},
			},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "read both"}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.Text != "Reading two docs." {
		t.Errorf("Text = %q, want %q", out.Text, "Reading two docs.")
	}
	if out.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want %q", out.FinishReason, "tool_calls")
	}
	if len(out.ToolCalls) != 2 {
		t.Fatalf("ToolCalls len = %d, want 2", len(out.ToolCalls))
	}
	if out.ToolCalls[0].ID != "0_0" {
		t.Errorf("ToolCalls[0].ID = %q, want 0_0", out.ToolCalls[0].ID)
	}
	if out.ToolCalls[1].ID != "0_1" {
		t.Errorf("ToolCalls[1].ID = %q, want 0_1", out.ToolCalls[1].ID)
	}
}

// TestStep_FunctionResponse_Roundtrip: prior tool result message becomes
// a user role message with a functionResponse part containing the prior
// tool name (looked up from prior assistant ToolCalls).
func TestStep_FunctionResponse_Roundtrip(t *testing.T) {
	captured := &stepRequest{}
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content: stepContent{
				Role:  "model",
				Parts: []stepPart{{Text: "OK done."}},
			},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, captured, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "read it"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{{
					ID:        "abc123",
					Name:      "read_doc",
					Arguments: `{"name":"nda.docx"}`,
				}},
			},
			{
				Role:       "tool",
				ToolCallID: "abc123",
				Content:    "the document body",
			},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}

	// Locate the functionResponse part in the captured request.
	var fr *stepFunctionResponse
	for _, c := range captured.Contents {
		for _, p := range c.Parts {
			if p.FunctionResponse != nil {
				fr = p.FunctionResponse
			}
		}
	}
	if fr == nil {
		t.Fatalf("no functionResponse part in captured request: %+v", captured)
	}
	if fr.Name != "read_doc" {
		t.Errorf("functionResponse.name = %q, want read_doc", fr.Name)
	}
	if fr.Response["content"] != "the document body" {
		t.Errorf("functionResponse.response.content = %v, want %q", fr.Response["content"], "the document body")
	}
}

// TestStep_TurnIndex_IDsIncrement: 2 prior assistant turns ⇒ next call IDs
// start at "2_0".
func TestStep_TurnIndex_IDsIncrement(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content: stepContent{
				Role: "model",
				Parts: []stepPart{
					{FunctionCall: &stepFunctionCall{Name: "noop", Args: map[string]interface{}{}}},
				},
			},
			FinishReason: "STOP",
		}},
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "one"},
			{Role: "user", Content: "second"},
			{Role: "assistant", Content: "two"},
			{Role: "user", Content: "third — please call noop"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(out.ToolCalls))
	}
	if out.ToolCalls[0].ID != "2_0" {
		t.Errorf("ToolCall.ID = %q, want 2_0", out.ToolCalls[0].ID)
	}
}

// TestStep_FinishReason_Length: MAX_TOKENS, no calls.
func TestStep_FinishReason_Length(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: "truncated..."}}},
			FinishReason: "MAX_TOKENS",
		}},
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "ramble"}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.FinishReason != "length" {
		t.Errorf("FinishReason = %q, want length", out.FinishReason)
	}
}

// TestStep_FinishReason_Safety: SAFETY → "error".
func TestStep_FinishReason_Safety(t *testing.T) {
	resp := stepResponseBody{
		Candidates: []stepCandidate{{
			Content:      stepContent{Role: "model", Parts: []stepPart{{Text: ""}}},
			FinishReason: "SAFETY",
		}},
	}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	out, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "bad"}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if out.FinishReason != "error" {
		t.Errorf("FinishReason = %q, want error", out.FinishReason)
	}
}

// TestStep_AuthFailed_401
func TestStep_AuthFailed_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"code":401,"message":"Invalid key","status":"UNAUTHENTICATED"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeAuthFailed {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeAuthFailed)
	}
}

// TestStep_RateLimit_429
func TestStep_RateLimit_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeRateLimit {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeRateLimit)
	}
	if !aiErr.Retryable {
		t.Error("Retryable = false, want true")
	}
}

// TestStep_5xx_Internal
func TestStep_5xx_Internal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`Service Unavailable`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeInternal)
	}
	if !aiErr.Retryable {
		t.Error("Retryable = false, want true")
	}
}

// TestStep_ContextLength_400
func TestStep_ContextLength_400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"context length exceeded","status":"INVALID_ARGUMENT"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeContextLength {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeContextLength)
	}
}

// TestStep_TransportError_Cancel: pre-canceled context.
func TestStep_TransportError_Cancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not be reached because ctx is already canceled.
		w.WriteHeader(200)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(ctx, &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeTimeout {
		t.Errorf("Code = %q, want %q", aiErr.Code, ai.CodeTimeout)
	}
}

// TestStep_RequestBody_SystemPromptTopLevel: SystemPrompt → top-level
// system_instruction; no system role in contents.
func TestStep_RequestBody_SystemPromptTopLevel(t *testing.T) {
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
		Model:        "gemini-3-flash",
		SystemPrompt: "You are a helpful assistant.",
		Messages: []ai.Message{
			{Role: "system", Content: "ignored — system goes top-level"},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if captured.SystemInstruction == nil {
		t.Fatal("captured SystemInstruction is nil; expected top-level")
	}
	if len(captured.SystemInstruction.Parts) == 0 ||
		captured.SystemInstruction.Parts[0].Text != "You are a helpful assistant." {
		t.Errorf("SystemInstruction parts = %+v", captured.SystemInstruction.Parts)
	}
	for _, c := range captured.Contents {
		if c.Role == "system" {
			t.Errorf("contents contained system role; should be skipped: %+v", c)
		}
	}
}

// TestStep_RequestBody_ToolArgsAsObject: prior assistant ToolCall with
// JSON-string Arguments is decoded into an args OBJECT, not a string.
func TestStep_RequestBody_ToolArgsAsObject(t *testing.T) {
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
			{Role: "user", Content: "call foo"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{{
					ID:        "x1",
					Name:      "foo",
					Arguments: `{"name":"foo","count":3}`,
				}},
			},
			{Role: "tool", ToolCallID: "x1", Content: "done"},
		},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}

	// Find the assistant turn's functionCall part and check args is an object.
	var fc *stepFunctionCall
	for _, c := range captured.Contents {
		if c.Role != "model" {
			continue
		}
		for _, p := range c.Parts {
			if p.FunctionCall != nil {
				fc = p.FunctionCall
			}
		}
	}
	if fc == nil {
		t.Fatalf("no functionCall in captured.Contents (model role); contents=%+v", captured.Contents)
	}
	if fc.Args["name"] != "foo" {
		t.Errorf("args.name = %v, want foo", fc.Args["name"])
	}
	// JSON numbers decode to float64 in interface{}.
	if v, ok := fc.Args["count"].(float64); !ok || v != 3 {
		t.Errorf("args.count = %v (%T), want float64(3)", fc.Args["count"], fc.Args["count"])
	}
}

// TestStep_EmptyCandidates_Error: server returns empty candidates list.
func TestStep_EmptyCandidates_Error(t *testing.T) {
	resp := stepResponseBody{Candidates: []stepCandidate{}}
	server := newStepServer(t, nil, resp, 0)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Step(context.Background(), &ai.Request{
		Model:    "gemini-3-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error for empty candidates, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("err type = %T, want *ai.AIError", err)
	}
	if aiErr.Code != ai.CodeProtocolError && aiErr.Code != ai.CodeInternal {
		t.Errorf("Code = %q, want ProtocolError or Internal", aiErr.Code)
	}
}

// TestStep_RequestBody_ToolDeclarations: req.Tools become tools[0].functionDeclarations.
func TestStep_RequestBody_ToolDeclarations(t *testing.T) {
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
		Messages: []ai.Message{{Role: "user", Content: "use tool"}},
		Tools: []ai.ToolSchema{{
			Name:        "lookup",
			Description: "Look up a key",
			Parameters:  `{"type":"object","properties":{"key":{"type":"string"}},"required":["key"]}`,
		}},
	})
	if err != nil {
		t.Fatalf("Step error = %v", err)
	}
	if len(captured.Tools) != 1 {
		t.Fatalf("captured.Tools len = %d, want 1", len(captured.Tools))
	}
	if len(captured.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("FunctionDeclarations len = %d, want 1", len(captured.Tools[0].FunctionDeclarations))
	}
	d := captured.Tools[0].FunctionDeclarations[0]
	if d.Name != "lookup" {
		t.Errorf("decl Name = %q, want lookup", d.Name)
	}
	if d.Description != "Look up a key" {
		t.Errorf("decl Description = %q, want %q", d.Description, "Look up a key")
	}
	if d.Parameters["type"] != "object" {
		t.Errorf("decl Parameters.type = %v, want object", d.Parameters["type"])
	}
}
