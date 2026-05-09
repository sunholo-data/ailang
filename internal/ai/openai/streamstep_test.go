package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestStreamStep_ParsesContentAndUsage verifies that StreamStep's per-chunk
// callback fires for each non-empty content delta and once for the final
// usage chunk, AND that the accumulated *ai.Response matches what Step
// would have produced for the same logical content.
func TestStreamStep_ParsesContentAndUsage(t *testing.T) {
	// Realistic OpenAI Chat Completions streaming sequence:
	//   - 3 content deltas
	//   - finish_reason chunk (delta empty, finish_reason="stop")
	//   - usage chunk (when stream_options.include_usage:true)
	//   - [DONE] sentinel
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":", "},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"world!"},"finish_reason":null}]}`,
		``,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		``,
		`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4o","choices":[],"usage":{"prompt_tokens":42,"completion_tokens":7,"total_tokens":49,"prompt_tokens_details":{"cached_tokens":10}}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	client := NewClient("test-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	var contentChunks []string
	var usageChunks []ai.StreamUsage
	onChunk := func(chunk ai.StreamChunk) {
		switch c := chunk.(type) {
		case ai.StreamContentDelta:
			contentChunks = append(contentChunks, c.Text)
		case ai.StreamUsage:
			usageChunks = append(usageChunks, c)
		}
	}

	resp, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gpt-4o",
		Messages: []ai.Message{{Role: "user", Content: "say hello"}},
	}, onChunk)
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	wantText := "Hello, world!"
	if resp.Text != wantText {
		t.Errorf("resp.Text = %q, want %q", resp.Text, wantText)
	}
	if got := strings.Join(contentChunks, ""); got != wantText {
		t.Errorf("concatenated chunks = %q, want %q", got, wantText)
	}
	if len(contentChunks) != 3 {
		t.Errorf("ContentDelta count = %d, want 3 (chunks: %v)", len(contentChunks), contentChunks)
	}

	if len(usageChunks) != 1 {
		t.Fatalf("Usage chunk count = %d, want 1", len(usageChunks))
	}
	gotUsage := usageChunks[0]
	if gotUsage.InputTokens != 42 || gotUsage.OutputTokens != 7 {
		t.Errorf("Usage tokens = %d/%d, want 42/7", gotUsage.InputTokens, gotUsage.OutputTokens)
	}
	if gotUsage.CacheReadInputTokens != 10 {
		t.Errorf("Usage.CacheReadInputTokens = %d, want 10", gotUsage.CacheReadInputTokens)
	}
	// OpenAI doesn't surface cache-creation as a separate count.
	if gotUsage.CacheCreationInputTokens != 0 {
		t.Errorf("Usage.CacheCreationInputTokens = %d, want 0 (OpenAI doesn't surface this)", gotUsage.CacheCreationInputTokens)
	}

	if resp.InputTokens != 42 || resp.OutputTokens != 7 {
		t.Errorf("resp tokens = %d/%d, want 42/7", resp.InputTokens, resp.OutputTokens)
	}
	if resp.CacheReadInputTokens != 10 {
		t.Errorf("resp.CacheReadInputTokens = %d, want 10", resp.CacheReadInputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("resp.FinishReason = %q, want stop", resp.FinishReason)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("resp.Model = %q, want gpt-4o", resp.Model)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("resp.ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
}

// TestStreamStep_ParsesToolCallFragments verifies that fragmented tool_calls
// across multiple chunks reassemble into single Response.ToolCalls entries
// with assembled JSON arguments.
func TestStreamStep_ParsesToolCallFragments(t *testing.T) {
	// Tool-call fragments: id+name on first fragment, arguments split
	// across 3 fragments (mirrors realistic OpenAI streaming behaviour).
	sseBody := strings.Join([]string{
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_xyz","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}`,
		``,
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}}]}`,
		``,
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"London\"}"}}]}}]}`,
		``,
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		``,
		`data: {"id":"chatcmpl-2","object":"chat.completion.chunk","model":"gpt-4o","choices":[],"usage":{"prompt_tokens":50,"completion_tokens":12,"total_tokens":62}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	client := NewClient("test-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	resp, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gpt-4o",
		Messages: []ai.Message{{Role: "user", Content: "weather?"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_xyz" {
		t.Errorf("ToolCall.ID = %q, want call_xyz", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want get_weather", tc.Name)
	}
	if tc.Arguments != `{"city":"London"}` {
		t.Errorf("ToolCall.Arguments = %q, want %q", tc.Arguments, `{"city":"London"}`)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("resp.FinishReason = %q, want tool_calls", resp.FinishReason)
	}
}

// TestStreamStep_StreamFlagAndOptionsSet verifies that StreamStep sends
// "stream":true AND "stream_options":{"include_usage":true} — the latter
// is what makes OpenAI emit a final usage chunk at all.
func TestStreamStep_StreamFlagAndOptionsSet(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = buf
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gpt-4o",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}
	body := string(capturedBody)
	if !strings.Contains(body, `"stream":true`) {
		t.Errorf("request body missing stream:true: %s", body)
	}
	if !strings.Contains(body, `"include_usage":true`) {
		t.Errorf("request body missing include_usage:true: %s", body)
	}
}

// TestStreamStep_HTTPError verifies that non-2xx responses surface as a
// typed *ai.AIError with the inner OpenAI error message hoisted.
func TestStreamStep_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":{"message":"Incorrect API key","type":"invalid_request_error","code":"invalid_api_key"}}`)
	}))
	defer srv.Close()

	client := NewClient("bad-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gpt-4o",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	}, func(_ ai.StreamChunk) {})
	if err == nil {
		t.Fatal("StreamStep returned nil error for 401 response")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("error type = %T, want *ai.AIError", err)
	}
	if !strings.Contains(aiErr.Message, "Incorrect API key") {
		t.Errorf("AIError.Message = %q, want it to contain 'Incorrect API key'", aiErr.Message)
	}
}
