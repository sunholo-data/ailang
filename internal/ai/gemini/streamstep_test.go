package gemini

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
// callback fires for each non-empty text part and once at end-of-stream
// with the FINAL (not running) usage block.
//
// Gemini emits running usage on every chunk and we deliberately surface
// only the last one — this test pins that behaviour.
func TestStreamStep_ParsesContentAndUsage(t *testing.T) {
	// Realistic Gemini streamGenerateContent SSE sequence:
	//   - 3 content chunks with running usage totals
	//   - final chunk with finishReason=STOP and authoritative usage
	sseBody := strings.Join([]string{
		`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}]}}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":42,"candidatesTokenCount":1,"totalTokenCount":43}}`,
		``,
		`data: {"candidates":[{"content":{"role":"model","parts":[{"text":", "}]}}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":42,"candidatesTokenCount":3,"totalTokenCount":45}}`,
		``,
		`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"world!"}]}}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":42,"candidatesTokenCount":7,"totalTokenCount":49}}`,
		``,
		`data: {"candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[]}}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":42,"candidatesTokenCount":7,"totalTokenCount":49,"cachedContentTokenCount":10}}`,
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
	resp, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gemini-2.5-flash",
		Messages: []ai.Message{{Role: "user", Content: "say hello"}},
	}, func(chunk ai.StreamChunk) {
		switch c := chunk.(type) {
		case ai.StreamContentDelta:
			contentChunks = append(contentChunks, c.Text)
		case ai.StreamUsage:
			usageChunks = append(usageChunks, c)
		}
	})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	wantText := "Hello, world!"
	if resp.Text != wantText {
		t.Errorf("resp.Text = %q, want %q", resp.Text, wantText)
	}
	if len(contentChunks) != 3 {
		t.Errorf("ContentDelta count = %d, want 3 (chunks: %v)", len(contentChunks), contentChunks)
	}

	if len(usageChunks) != 1 {
		t.Fatalf("Usage chunk count = %d, want 1 (Gemini fires once at end-of-stream, NOT per-chunk)", len(usageChunks))
	}
	gotUsage := usageChunks[0]
	if gotUsage.InputTokens != 42 || gotUsage.OutputTokens != 7 {
		t.Errorf("Usage tokens = %d/%d, want 42/7 (final values, not running totals)", gotUsage.InputTokens, gotUsage.OutputTokens)
	}
	if gotUsage.CacheReadInputTokens != 10 {
		t.Errorf("Usage.CacheReadInputTokens = %d, want 10 (CachedContent token count)", gotUsage.CacheReadInputTokens)
	}
	if gotUsage.CacheCreationInputTokens != 0 {
		t.Errorf("Usage.CacheCreationInputTokens = %d, want 0 (Gemini doesn't surface this)", gotUsage.CacheCreationInputTokens)
	}

	if resp.InputTokens != 42 || resp.OutputTokens != 7 {
		t.Errorf("resp tokens = %d/%d, want 42/7", resp.InputTokens, resp.OutputTokens)
	}
	if resp.CacheReadInputTokens != 10 {
		t.Errorf("resp.CacheReadInputTokens = %d, want 10", resp.CacheReadInputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("resp.FinishReason = %q, want stop (mapped from STOP)", resp.FinishReason)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("resp.Model = %q, want gemini-2.5-flash", resp.Model)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("resp.ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
}

// TestStreamStep_ParsesFunctionCalls verifies that functionCall parts in
// the streamed candidates produce ai.ToolCall entries with deterministic
// turn-index based IDs (matching the non-streaming Step contract).
func TestStreamStep_ParsesFunctionCalls(t *testing.T) {
	// Gemini delivers the entire functionCall in ONE chunk (unlike OpenAI's
	// fragment-streaming) — args are a complete object, not partial JSON.
	sseBody := strings.Join([]string{
		`data: {"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"get_weather","args":{"city":"London"}}}]}}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":12,"totalTokenCount":62}}`,
		``,
		`data: {"candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[]}}],"modelVersion":"gemini-2.5-flash","usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":12,"totalTokenCount":62}}`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	client := NewClient("test-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	resp, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gemini-2.5-flash",
		Messages: []ai.Message{{Role: "user", Content: "weather?"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	// Turn index is 0 (no prior assistant messages); call index is 0.
	if tc.ID != "0_0" {
		t.Errorf("ToolCall.ID = %q, want 0_0 (turn_call deterministic)", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall.Name = %q, want get_weather", tc.Name)
	}
	if tc.Arguments != `{"city":"London"}` {
		t.Errorf("ToolCall.Arguments = %q, want %q", tc.Arguments, `{"city":"London"}`)
	}
	// finishReason=STOP gets overridden to tool_calls when ToolCalls is non-empty.
	if resp.FinishReason != "tool_calls" {
		t.Errorf("resp.FinishReason = %q, want tool_calls", resp.FinishReason)
	}
}

// TestStreamStep_StreamURLUsesAltSse verifies that StreamStep targets the
// streamGenerateContent endpoint (not generateContent) AND requests SSE
// framing via alt=sse — without alt=sse, Gemini returns a JSON array which
// our parser doesn't handle.
func TestStreamStep_StreamURLUsesAltSse(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, `data: {"candidates":[{"finishReason":"STOP","content":{"parts":[]}}],"usageMetadata":{}}`+"\n\n")
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "gemini-2.5-flash",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}
	if !strings.Contains(capturedPath, ":streamGenerateContent") {
		t.Errorf("URL missing :streamGenerateContent: %s", capturedPath)
	}
	if !strings.Contains(capturedPath, "alt=sse") {
		t.Errorf("URL missing alt=sse: %s", capturedPath)
	}
}

// TestStreamStep_HTTPError verifies that non-200 responses surface as a
// typed *ai.AIError with the inner Gemini error message hoisted.
func TestStreamStep_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":{"code":400,"message":"Invalid model: foo","status":"INVALID_ARGUMENT"}}`)
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "foo",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	}, func(_ ai.StreamChunk) {})
	if err == nil {
		t.Fatal("StreamStep returned nil error for 400 response")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("error type = %T, want *ai.AIError", err)
	}
	if !strings.Contains(aiErr.Message, "Invalid model: foo") {
		t.Errorf("AIError.Message = %q, want it to contain 'Invalid model: foo'", aiErr.Message)
	}
}
