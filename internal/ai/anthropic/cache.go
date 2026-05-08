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

// hasSystemBreakpoint returns true iff any breakpoint targets the system
// prompt. Used to choose between string (no cache) and content-array (cached)
// wire shapes for the System field.
func hasSystemBreakpoint(breakpoints []ai.CacheBreakpoint) bool {
	for _, bp := range breakpoints {
		if bp.Position == "system" {
			return true
		}
	}
	return false
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
