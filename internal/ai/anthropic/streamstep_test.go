package anthropic

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
// callback fires for each text_delta event and once at message_delta with
// the final usage block, AND that the accumulated *ai.Response matches what
// Step would have produced for the same logical content.
func TestStreamStep_ParsesContentAndUsage(t *testing.T) {
	// Realistic Anthropic SSE event sequence — three text deltas + final
	// message_delta with usage. Mirrors the wire format from
	// https://docs.anthropic.com/en/api/messages-streaming.
	sseBody := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_01","model":"claude-sonnet-4-5","role":"assistant","content":[],"stop_reason":null,"usage":{"input_tokens":42,"output_tokens":1,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":", "}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"world!"}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_tokens":42,"output_tokens":7,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
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
		Model:        "claude-sonnet-4-5",
		Messages:     []ai.Message{{Role: "user", Content: "say hello"}},
		SystemPrompt: "be terse",
		MaxTokens:    100,
	}, onChunk)
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	// Final assembled text equals the concatenation of all ContentDelta chunks.
	wantText := "Hello, world!"
	if resp.Text != wantText {
		t.Errorf("resp.Text = %q, want %q", resp.Text, wantText)
	}
	if got := strings.Join(contentChunks, ""); got != wantText {
		t.Errorf("concatenated chunks = %q, want %q", got, wantText)
	}

	// 3 text deltas should have fired 3 ContentDelta callbacks.
	if len(contentChunks) != 3 {
		t.Errorf("ContentDelta count = %d, want 3 (chunks: %v)", len(contentChunks), contentChunks)
	}

	// Usage callback should fire exactly once at message_delta.
	if len(usageChunks) != 1 {
		t.Fatalf("Usage chunk count = %d, want 1", len(usageChunks))
	}
	gotUsage := usageChunks[0]
	if gotUsage.InputTokens != 42 {
		t.Errorf("Usage.InputTokens = %d, want 42", gotUsage.InputTokens)
	}
	if gotUsage.OutputTokens != 7 {
		t.Errorf("Usage.OutputTokens = %d, want 7", gotUsage.OutputTokens)
	}
	if gotUsage.CacheReadInputTokens != 10 {
		t.Errorf("Usage.CacheReadInputTokens = %d, want 10", gotUsage.CacheReadInputTokens)
	}
	if gotUsage.CacheCreationInputTokens != 5 {
		t.Errorf("Usage.CacheCreationInputTokens = %d, want 5", gotUsage.CacheCreationInputTokens)
	}

	// Final Response must carry the same token counts as the Usage chunk.
	if resp.InputTokens != 42 || resp.OutputTokens != 7 {
		t.Errorf("resp tokens = %d/%d, want 42/7", resp.InputTokens, resp.OutputTokens)
	}
	if resp.CacheReadInputTokens != 10 || resp.CacheCreationInputTokens != 5 {
		t.Errorf("resp cache tokens = %d/%d, want 10/5", resp.CacheReadInputTokens, resp.CacheCreationInputTokens)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("resp.FinishReason = %q, want stop (mapped from end_turn)", resp.FinishReason)
	}
	if resp.Model != "claude-sonnet-4-5" {
		t.Errorf("resp.Model = %q, want claude-sonnet-4-5", resp.Model)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("resp.ToolCalls len = %d, want 0", len(resp.ToolCalls))
	}
}

// TestStreamStep_ParsesToolUseBlock verifies that a streamed tool_use block
// (input_json_delta fragments accumulated until content_block_stop) becomes
// a single Response.ToolCalls entry with the assembled JSON.
func TestStreamStep_ParsesToolUseBlock(t *testing.T) {
	sseBody := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_02","model":"claude-sonnet-4-5","role":"assistant","content":[],"stop_reason":null,"usage":{"input_tokens":50,"output_tokens":1}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_xyz","name":"get_weather","input":{}}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"city\":"}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"London\"}"}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"input_tokens":50,"output_tokens":12}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, sseBody)
	}))
	defer srv.Close()

	client := NewClient("test-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	resp, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "what's the weather"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "toolu_xyz" {
		t.Errorf("ToolCall.ID = %q, want toolu_xyz", tc.ID)
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

// TestStreamStep_HTTPError verifies that non-2xx responses surface as a
// typed *ai.AIError with the inner Anthropic error message hoisted.
func TestStreamStep_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"bad api key"}}`)
	}))
	defer srv.Close()

	client := NewClient("bad-key", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	}, func(_ ai.StreamChunk) {})
	if err == nil {
		t.Fatal("StreamStep returned nil error for 401 response")
	}
	aiErr, ok := err.(*ai.AIError)
	if !ok {
		t.Fatalf("error type = %T, want *ai.AIError", err)
	}
	if !strings.Contains(aiErr.Message, "bad api key") {
		t.Errorf("AIError.Message = %q, want it to contain 'bad api key'", aiErr.Message)
	}
}

// TestStreamStep_StreamFlagSet verifies that StreamStep sends "stream":true
// in the request body — non-streaming Step does NOT, and the omitempty on
// the Stream field is what guarantees that.
func TestStreamStep_StreamFlagSet(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = buf
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"x\",\"model\":\"m\",\"role\":\"a\",\"usage\":{}}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{}}\n\n")
	}))
	defer srv.Close()

	client := NewClient("k", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := client.StreamStep(context.Background(), &ai.Request{
		Model:    "claude-sonnet-4-5",
		Messages: []ai.Message{{Role: "user", Content: "x"}},
	}, func(_ ai.StreamChunk) {})
	if err != nil {
		t.Fatalf("StreamStep returned error: %v", err)
	}
	if !strings.Contains(string(capturedBody), `"stream":true`) {
		t.Errorf("request body missing stream:true: %s", capturedBody)
	}
}
