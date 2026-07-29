package anthropic

// Anthropic prompt-cache support (M-AI-PROMPT-CACHING, v0.18.4).
//
// Anthropic's Messages API supports up to 4 cache breakpoints per request via
// `cache_control: {type: "ephemeral"}` markers attached to content blocks.
// First turn pays full price + ~25% cache-write surcharge; subsequent turns
// hit the cache and pay ~10% of the cached portion's input cost. Server-side
// TTL is ~5 minutes for "ephemeral".
//
// This file is also imported by internal/ai/openrouter/step.go for the
// `anthropic/...` model-prefix routing path — both backends accept the
// identical cache_control wire shape.

import (
	"encoding/json"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// cacheControlBlock is the Anthropic cache-control marker. JSON wire shape:
//
//	{"type": "ephemeral"}
//
// Anthropic also defines longer-tier markers (1h, 24h) but those are not yet
// generally available; ephemeral is the only stable tier as of v0.18.4.
type cacheControlBlock struct {
	Type string `json:"type"`
}

// systemTextBlock is one entry in the system content array. Used when at
// least one CacheBreakpoint targets position="system".
//
//	{"type":"text", "text":"...", "cache_control":{"type":"ephemeral"}}
type systemTextBlock struct {
	Type         string             `json:"type"`
	Text         string             `json:"text"`
	CacheControl *cacheControlBlock `json:"cache_control,omitempty"`
}

// hasBreakpoint reports whether any breakpoint targets the given position.
func hasBreakpoint(breakpoints []ai.CacheBreakpoint, position string) bool {
	for _, bp := range breakpoints {
		if bp.Position == position {
			return true
		}
	}
	return false
}

// hasSystemBreakpoint returns true iff any breakpoint targets the system
// prompt. Used to choose between string (no cache) and content-array (cached)
// wire shapes for the System field.
func hasSystemBreakpoint(breakpoints []ai.CacheBreakpoint) bool {
	return hasBreakpoint(breakpoints, "system")
}

// systemFieldFromPrompt builds the JSON for the Anthropic `system` field.
// Returns one of:
//
//   - empty json.RawMessage if systemPrompt is empty (caller should omit
//     the field entirely via omitempty semantics)
//   - a bare JSON-quoted string when no system breakpoint is present
//     (bit-for-bit identical to the pre-v0.18.4 wire shape)
//   - a JSON array of one systemTextBlock with cache_control set when a
//     system breakpoint is present
//
// Returning json.RawMessage lets the caller assign to a single field that
// covers both shapes — Anthropic's API accepts either.
func systemFieldFromPrompt(systemPrompt string, breakpoints []ai.CacheBreakpoint) (json.RawMessage, error) {
	if systemPrompt == "" {
		// Empty system prompt — even with a breakpoint, there's nothing to
		// cache. Return nil so the caller can leave the field unset.
		return nil, nil
	}
	if !hasSystemBreakpoint(breakpoints) {
		// Bit-for-bit identical to pre-v0.18.4: a bare JSON string.
		return json.Marshal(systemPrompt)
	}
	// Cache-aware path: emit a content array with cache_control.
	blocks := []systemTextBlock{
		{
			Type:         "text",
			Text:         systemPrompt,
			CacheControl: &cacheControlBlock{Type: "ephemeral"},
		},
	}
	return json.Marshal(blocks)
}

// userContentFromPrompt builds the JSON for a user message's `content` field
// (M-ANTHROPIC-CACHE-HIT-RATE, v0.31.0). Returns one of:
//
//   - a bare JSON-quoted string of cachedPrefix+userPrompt — used whenever
//     there is nothing to cache (no "user_prefix" breakpoint, or an empty
//     prefix). Byte-identical to the pre-v0.31.0 wire shape.
//   - a two-block content array, cache_control on the first block only.
//
// The invariant callers depend on: whichever shape is returned, the text the
// model sees is exactly cachedPrefix+userPrompt. The cached encoding must never
// change what is being asked, only how it is framed — that is what lets prompt
// caching land without invalidating eval baseline comparability (design doc D1).
//
// cache_control goes on block 0 ONLY. A marker on the volatile block would key
// the cache to content that differs every request, so it would write an entry
// per call and never read one — the silent-miss failure this milestone exists
// to remove.
func userContentFromPrompt(cachedPrefix, userPrompt string, breakpoints []ai.CacheBreakpoint) (json.RawMessage, error) {
	if cachedPrefix == "" || !hasBreakpoint(breakpoints, "user_prefix") {
		// Nothing to cache. Concatenate and emit the legacy bare-string shape.
		// Note this still SENDS the prefix — dropping it would corrupt the
		// prompt for callers who set it without declaring a breakpoint.
		return json.Marshal(cachedPrefix + userPrompt)
	}

	blocks := []systemTextBlock{
		{
			Type:         "text",
			Text:         cachedPrefix,
			CacheControl: &cacheControlBlock{Type: "ephemeral"},
		},
	}
	// Anthropic rejects empty text blocks, so only append the remainder when
	// there is one.
	if userPrompt != "" {
		blocks = append(blocks, systemTextBlock{Type: "text", Text: userPrompt})
	}
	return json.Marshal(blocks)
}

// minCacheablePrefixByModel maps a model family substring to Anthropic's
// minimum cacheable prefix in tokens. A prefix shorter than this is SILENTLY
// not cached by the API — no error, just cache_creation_input_tokens: 0.
//
// The minimum is NOT monotonic across generations (Opus 4.6 needs 8x what
// Opus 5 needs), so this cannot be inferred from a version ordering; it has to
// be a table. Longest-match wins, so "claude-opus-4-6" is checked before the
// broader "claude-opus".
var minCacheablePrefixByModel = []struct {
	match     string
	minTokens int
}{
	{"claude-opus-5", 512},
	{"claude-fable-5", 512},
	{"claude-mythos-5", 512},
	{"claude-opus-4-8", 1024},
	{"claude-opus-4-7", 2048},
	{"claude-opus-4-6", 4096},
	{"claude-opus-4-5", 4096},
	{"claude-sonnet-5", 1024},
	{"claude-sonnet-4-6", 1024},
	{"claude-sonnet-4-5", 1024},
	{"claude-haiku-4-5", 4096},
	{"claude-haiku-3-5", 2048},
}

// defaultMinCacheablePrefixTokens is used for models absent from the table
// above. 1024 is the most common tier and the conservative choice: guessing
// too LOW would suppress a warning that should fire, whereas guessing too high
// only risks a spurious warning. This value affects diagnostics only — it never
// changes what is sent — so a default here does not violate the no-silent-
// fallback rule that governs pricing and model config.
const defaultMinCacheablePrefixTokens = 1024

// minCacheablePrefixTokens returns the minimum cacheable prefix for a model.
func minCacheablePrefixTokens(model string) int {
	best := -1
	bestLen := 0
	for _, e := range minCacheablePrefixByModel {
		if strings.Contains(model, e.match) && len(e.match) > bestLen {
			best, bestLen = e.minTokens, len(e.match)
		}
	}
	if best < 0 {
		return defaultMinCacheablePrefixTokens
	}
	return best
}

// approxTokens estimates a token count from raw text at the usual ~4 chars per
// token. This is deliberately an ESTIMATE: it gates a stderr warning, never a
// request. Calling Anthropic's count_tokens endpoint to decide whether to print
// a diagnostic would add a network round-trip per request to save nothing.
func approxTokens(s string) int { return len(s) / 4 }

// cachePrefixTooSmall reports the model's minimum cacheable prefix and whether
// the supplied prefix falls under it.
//
// This exists because the failure is otherwise INVISIBLE. Anthropic accepts a
// too-short cache_control marker and simply declines to cache — which is how
// the eval harness ran with a 70-token system prompt against a 1024-token
// minimum (design doc V8) without anyone noticing the cache never engaged.
func cachePrefixTooSmall(model, prefix string) (minTokens int, tooSmall bool) {
	minTokens = minCacheablePrefixTokens(model)
	return minTokens, approxTokens(prefix) < minTokens
}

// warnIfCachePrefixTooSmall emits a one-shot warning when a request declares a
// cache breakpoint whose target is too short for the model to cache. Purely
// diagnostic — the request proceeds unchanged.
func warnIfCachePrefixTooSmall(req *ai.Request) {
	if len(req.CacheBreakpoints) == 0 {
		return
	}
	check := func(position, text string) {
		if !hasBreakpoint(req.CacheBreakpoints, position) || text == "" {
			return
		}
		if minTokens, tooSmall := cachePrefixTooSmall(req.Model, text); tooSmall {
			ai.WarnOnceCachePrefixTooSmall(req.Model, position, approxTokens(text), minTokens)
		}
	}
	check("system", req.SystemPrompt)
	check("user_prefix", req.CachedPrefix)
}
