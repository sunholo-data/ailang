package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// M-ANTHROPIC-CACHE-HIT-RATE follow-up: OpenAI caches prompts automatically
// (>=1024 tokens, stable prefix) — there is no request-side knob. The only
// thing we control is whether we RECORD the hit. Before this, the Generate path
// dropped usage.prompt_tokens_details.cached_tokens on the floor, so every
// banked OpenAI eval showed 0 cache reads and a working cache was
// indistinguishable from a broken one.
//
// This test pins the JSON tag and the mapping to the normalized field.
func TestGenerate_RecordsAutomaticCacheHit_ChatCompletions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-1","object":"chat.completion","model":"gpt-5",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":16500,"completion_tokens":5,"total_tokens":16505,
			         "prompt_tokens_details":{"cached_tokens":16000}}
		}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", WithBaseURL(srv.URL))
	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:      "gpt-4o", // legacy models route to Chat Completions
		UserPrompt: "hello",
		MaxTokens:  16,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.CacheReadInputTokens != 16000 {
		t.Errorf("CacheReadInputTokens = %d, want 16000 — automatic cache hits must reach the normalized field or they never get banked",
			resp.CacheReadInputTokens)
	}
}

// The Responses API (gpt-5, o1, o3, codex) reports the same automatic cache
// under a different key — input_tokens_details.cached_tokens. Both paths must
// land on the normalized field or half our OpenAI fleet stays unmeasurable.
func TestGenerate_RecordsAutomaticCacheHit_ResponsesAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_1","object":"response","model":"gpt-5",
			"output":[{"type":"message","role":"assistant",
			           "content":[{"type":"output_text","text":"ok"}]}],
			"usage":{"input_tokens":16500,"output_tokens":5,"total_tokens":16505,
			         "input_tokens_details":{"cached_tokens":16000}}
		}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", WithBaseURL(srv.URL))
	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:      "gpt-5",
		UserPrompt: "hello",
		MaxTokens:  16,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.CacheReadInputTokens != 16000 {
		t.Errorf("CacheReadInputTokens = %d, want 16000 (Responses API input_tokens_details.cached_tokens)",
			resp.CacheReadInputTokens)
	}
}
