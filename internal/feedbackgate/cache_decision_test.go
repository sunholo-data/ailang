package feedbackgate

import "testing"

// M-ANTHROPIC-CACHE-HIT-RATE M2: records WHY the classifier declares no prompt-
// cache breakpoint, so the decision is re-checked against evidence rather than
// re-litigated from memory.
//
// Anthropic silently declines to cache a prefix below the model's minimum — no
// error, cache_creation_input_tokens: 0. Declaring a breakpoint here would look
// like a win and do nothing.
func TestClassifierPromptIsBelowCacheMinimum(t *testing.T) {
	// claude-haiku-4-5 (the default classifier model) requires 4096 tokens.
	const haiku45MinCacheableTokens = 4096

	approxTokens := len(DefaultPrompt()) / 4
	if approxTokens >= haiku45MinCacheableTokens {
		t.Fatalf("classifier prompt is now ~%d tokens, at or above the %d-token minimum for claude-haiku-4-5 —"+
			" caching it is now worthwhile, so revisit the deliberate no-breakpoint decision in classifier.go",
			approxTokens, haiku45MinCacheableTokens)
	}
	t.Logf("classifier prompt ~%d tokens vs %d-token minimum: correctly not cached", approxTokens, haiku45MinCacheableTokens)
}
