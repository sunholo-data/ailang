package anthropic

// Live prompt-cache verification for M-ANTHROPIC-CACHE-HIT-RATE.
//
// This is a MANUAL, live, API-key-gated probe. It is NEVER run in default CI:
// it makes two real billed Anthropic calls. It exists because the rest of this
// milestone's tests prove only the WIRE SHAPE — that we emit cache_control in
// the right place. They cannot prove Anthropic actually created and then read a
// cache entry, and that distinction is the whole point of the milestone: the
// failure mode we are fixing is one where the API accepts the marker, returns
// 200, and silently declines to cache (design doc V8).
//
// Run it with:
//
//	ANTHROPIC_API_KEY=sk-ant-... AILANG_LIVE_ANTHROPIC_CACHE=1 \
//	  go test ./internal/ai/anthropic/ -run TestLiveGeneratePromptCache -v -timeout 5m
//
// Optional override:
//
//	AILANG_LIVE_CACHE_MODEL  (default "claude-sonnet-5")
//
// Expected result: call 1 reports cache_creation_input_tokens > 0 (a write),
// call 2 reports cache_read_input_tokens > 0 (a hit). If call 2 reports a
// second write instead of a read, the prefix is being invalidated between calls
// and the cache is not working, no matter how correct the wire shape looks.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestLiveGeneratePromptCache(t *testing.T) {
	if os.Getenv("AILANG_LIVE_ANTHROPIC_CACHE") == "" {
		t.Skip("live probe: set AILANG_LIVE_ANTHROPIC_CACHE=1 (makes 2 real billed calls)")
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("live probe: ANTHROPIC_API_KEY not set")
	}
	model := os.Getenv("AILANG_LIVE_CACHE_MODEL")
	if model == "" {
		model = "claude-sonnet-5"
	}

	// A prefix comfortably over every model tier's minimum (~4k tokens). Must be
	// byte-identical across both calls or the prefix match fails.
	prefix := "You are a reference assistant. Background material follows.\n" +
		strings.Repeat("Fact: the sky is blue and water is wet. ", 1500)

	client := NewClient(apiKey)
	call := func(task string) *ai.Response {
		t.Helper()
		resp, err := client.Generate(context.Background(), &ai.Request{
			Model:            model,
			CachedPrefix:     prefix,
			UserPrompt:       "\n\n## Task\n\n" + task,
			MaxTokens:        16,
			CacheBreakpoints: []ai.CacheBreakpoint{{Position: "user_prefix", TTL: "ephemeral"}},
		})
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		return resp
	}

	// Call 1 writes the cache. The task differs between calls so that ONLY the
	// shared prefix can account for any cache read on call 2.
	first := call("Reply with the single word: one.")
	t.Logf("call 1: input=%d cache_write=%d cache_read=%d",
		first.InputTokens, first.CacheCreationInputTokens, first.CacheReadInputTokens)

	second := call("Reply with the single word: two.")
	t.Logf("call 2: input=%d cache_write=%d cache_read=%d",
		second.InputTokens, second.CacheCreationInputTokens, second.CacheReadInputTokens)

	if first.CacheCreationInputTokens == 0 {
		t.Errorf("call 1 wrote no cache (cache_creation_input_tokens=0) — the marker was accepted but nothing was cached; check the prefix is above this model's minimum")
	}
	if second.CacheReadInputTokens == 0 {
		t.Errorf("call 2 read no cache (cache_read_input_tokens=0) — the prefix is being invalidated between calls; something upstream of the breakpoint differs per request")
	}
	if second.CacheReadInputTokens > 0 && second.CacheReadInputTokens < first.CacheCreationInputTokens/2 {
		t.Errorf("call 2 read only %d of the %d cached tokens — the breakpoint is landing mid-prefix",
			second.CacheReadInputTokens, first.CacheCreationInputTokens)
	}
}
