package openrouter

// OpenRouter prompt-cache hint dispatch (M-AI-PROMPT-CACHING, v0.18.4).
//
// OpenRouter speaks OpenAI Chat Completions on the wire but routes the
// underlying call to whatever provider the model-string prefix names
// (anthropic/, openai/, google/, mistralai/, etc.). For prompt caching:
//
//   - "anthropic/..." → OpenRouter forwards Anthropic-style cache_control
//     markers on the system message. We stamp them after the OpenAI-shape
//     body is built, mutating the system message's Content from a bare
//     JSON string to a content array with cache_control set.
//
//   - "openai/..." → OpenAI auto-caches transparently; cache_control would
//     be invalid on an OpenAI request. NO-OP + once-per-session warning.
//
//   - "google/..." → Gemini's caching uses an async CachedContent API with
//     no synchronous request-side hint. NO-OP + once-per-session warning.
//
//   - Anything else (mistralai/, meta-llama/, unknown prefixes) → silent
//     NO-OP. We don't claim to know whether any given upstream supports
//     hints, so we don't emit a warning that might be wrong.

import (
	"encoding/json"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

// providerKeyForRoute returns the canonical provider name for the model
// prefix. Used by warnings to distinguish OpenRouter→OpenAI from direct
// OpenAI calls in stderr (so eval-harness debug output is unambiguous).
func providerKeyForRoute(model string) string {
	switch {
	case strings.HasPrefix(model, "anthropic/"):
		return "openrouter_routed_to_anthropic"
	case strings.HasPrefix(model, "openai/"):
		return "openrouter_routed_to_openai"
	case strings.HasPrefix(model, "google/"):
		return "openrouter_routed_to_gemini"
	default:
		return "openrouter_routed_to_unknown"
	}
}

// applyCacheHintsForRoute mutates apiReq in place to apply cache_breakpoints
// per the routed-to-provider's contract. Empty breakpoints = no-op (apiReq
// unchanged). Called after openai.BuildChatStepRequest so we can mutate
// the OpenAI-shape message array directly.
//
// For Anthropic-routed calls, this stamps cache_control:{type:"ephemeral"}
// on the system message's content array (per OpenRouter's pass-through
// convention for Anthropic models).
func applyCacheHintsForRoute(apiReq *openai.ChatStepRequest, model string, breakpoints []ai.CacheBreakpoint) error {
	if len(breakpoints) == 0 {
		return nil
	}
	switch {
	case strings.HasPrefix(model, "anthropic/"):
		return stampAnthropicCacheControl(apiReq, breakpoints)
	case strings.HasPrefix(model, "openai/"):
		ai.WarnOnceCacheHintIgnored(providerKeyForRoute(model), "auto_cache")
		return nil
	case strings.HasPrefix(model, "google/"):
		ai.WarnOnceCacheHintIgnored(providerKeyForRoute(model), "no_explicit_api")
		return nil
	default:
		// Unknown route — silent. We don't know whether the underlying
		// provider supports hints, so we neither apply nor warn.
		return nil
	}
}

// anthropicSystemTextBlock is the wire shape for one entry in the system
// message's content array when routing to Anthropic via OpenRouter. Mirrors
// internal/ai/anthropic/cache.go but lives in OpenRouter to avoid a circular
// import (openrouter already imports openai; pulling in anthropic too would
// be cleaner if the helpers were in internal/ai itself).
type anthropicSystemTextBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"`
}

// hasSystemBreakpoint mirrors the helper in internal/ai/anthropic/cache.go.
// Duplicated here to avoid the circular import problem; both are tiny.
func hasSystemBreakpoint(breakpoints []ai.CacheBreakpoint) bool {
	for _, bp := range breakpoints {
		if bp.Position == "system" {
			return true
		}
	}
	return false
}

// stampAnthropicCacheControl finds the system-role message in apiReq.Messages
// and replaces its Content (currently a JSON-quoted string) with a content
// array containing one text block with cache_control:{type:"ephemeral"}.
//
// If no system message is present, nothing happens (nothing to cache).
// If breakpoints don't include a "system" entry, also no-op (Phase 1
// supports system-only placement; tool_result and last_user are deferred
// per the design doc Non-Goals).
func stampAnthropicCacheControl(apiReq *openai.ChatStepRequest, breakpoints []ai.CacheBreakpoint) error {
	if !hasSystemBreakpoint(breakpoints) {
		return nil
	}
	for i, msg := range apiReq.Messages {
		if msg.Role != "system" {
			continue
		}
		// Decode the existing Content (a JSON-quoted string) back to plain text.
		var systemText string
		if err := json.Unmarshal(msg.Content, &systemText); err != nil {
			// Already a content array? Don't double-wrap; bail without error.
			return nil
		}
		if systemText == "" {
			// Nothing to cache.
			return nil
		}
		blocks := []anthropicSystemTextBlock{
			{
				Type:         "text",
				Text:         systemText,
				CacheControl: &anthropicCacheControl{Type: "ephemeral"},
			},
		}
		raw, err := json.Marshal(blocks)
		if err != nil {
			return err
		}
		apiReq.Messages[i].Content = raw
		return nil
	}
	// No system message — nothing to cache.
	return nil
}
