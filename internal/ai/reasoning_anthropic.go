package ai

import (
	"sort"
	"strings"
)

// Anthropic thinking-control generations (M-AI-REASONING-EFFORT follow-up).
//
// Anthropic REMOVED the fixed-budget extended-thinking control
// (thinking: {type:"enabled", budget_tokens:N}) starting with Claude Opus 4.7.
// Sending it to a 4.7-or-later model is a hard HTTP 400, not a degraded run.
// The replacement is adaptive thinking plus a qualitative effort:
//
//	{"thinking": {"type": "adaptive"}, "output_config": {"effort": "high"}}
//
// budget_tokens remains valid only on Opus 4.6 / Sonnet 4.6 (deprecated there)
// and older models. This file is the ONE place that records which generation a
// given Anthropic model belongs to; the anthropic client reads it to pick the
// wire shape, and ResolveReasoning reads it to reject an explicit token budget
// on a model that cannot express one.
//
// Fail-loud, consistent with reasoning_capability.go: an unlisted model has NO
// assumed generation. Callers must reject rather than guess a shape — guessing
// wrong is exactly the 400 this table exists to prevent.

// AnthropicThinkingStyle records how one Anthropic model generation expresses
// thinking control on the wire.
type AnthropicThinkingStyle struct {
	// Adaptive is true for Opus 4.7 and later, where thinking is controlled by
	// {type:"adaptive"} + output_config.effort and budget_tokens is rejected.
	// False for 4.6-and-older, which still honor {type:"enabled",budget_tokens}.
	Adaptive bool

	// CanDisable reports whether the model accepts an explicit request for no
	// thinking at all.
	//
	// Budget-style models express disablement by omitting the block, which is
	// always available. Adaptive-style models need an explicit
	// {type:"disabled"} — accepted on Opus 4.7/4.8, Opus 5 and Sonnet 5, but
	// REJECTED with a 400 on Fable 5 / Mythos 5, where thinking is always on.
	//
	// Opus 5 additionally rejects {type:"disabled"} when output_config.effort is
	// "xhigh" or "max". That combination is unreachable here: this resolver's
	// effort vocabulary tops out at "high", and disablement is only ever emitted
	// for effort "off", which sends no effort field at all.
	CanDisable bool
}

// anthropicThinkingStyles is the generation table, keyed by the model id with
// any -YYYYMMDD snapshot suffix stripped.
//
// Adding a model here is a claim about its API surface, not its quality — check
// the vendor migration notes for the generation boundary before extending it.
// A model registered in reasoningCapabilities MUST appear here; that invariant
// is pinned by TestAnthropicThinkingStyle_CoversCapabilityTable.
var anthropicThinkingStyles = map[string]AnthropicThinkingStyle{
	// --- Adaptive generation: budget_tokens REMOVED (400) -----------------
	"claude-opus-5":   {Adaptive: true, CanDisable: true},
	"claude-opus-4-8": {Adaptive: true, CanDisable: true},
	"claude-opus-4-7": {Adaptive: true, CanDisable: true},
	"claude-sonnet-5": {Adaptive: true, CanDisable: true},
	// Fable 5 / Mythos 5: thinking is ALWAYS on. An explicit
	// {type:"disabled"} is a 400 at any effort, so "off" is not expressible.
	// Fable 5.1 (2026-09-01) keeps that surface unchanged — same generation,
	// same always-on adaptive thinking, default effort "high".
	"claude-fable-5-1":      {Adaptive: true, CanDisable: false},
	"claude-fable-5":        {Adaptive: true, CanDisable: false},
	"claude-mythos-5":       {Adaptive: true, CanDisable: false},
	"claude-mythos-preview": {Adaptive: true, CanDisable: false},

	// --- Budget generation: thinking.budget_tokens still honored ----------
	// (Deprecated on the 4.6 pair, functional. Disablement is by omission.)
	"claude-opus-4-6":   {Adaptive: false, CanDisable: true},
	"claude-sonnet-4-6": {Adaptive: false, CanDisable: true},
	"claude-opus-4-5":   {Adaptive: false, CanDisable: true},
	"claude-sonnet-4-5": {Adaptive: false, CanDisable: true},
	"claude-haiku-4-5":  {Adaptive: false, CanDisable: true},
	"claude-opus-4-1":   {Adaptive: false, CanDisable: true},
	"claude-opus-4-0":   {Adaptive: false, CanDisable: true},
	"claude-sonnet-4-0": {Adaptive: false, CanDisable: true},
	// Claude 3.x models are deliberately absent: they have no thinking control
	// at all, so a reasoning request on one must fail rather than pick a shape.
}

// AnthropicThinkingStyleFor returns the thinking-control generation for an
// Anthropic model id, tolerating a -YYYYMMDD snapshot suffix
// (e.g. "claude-sonnet-4-5-20250929" resolves as "claude-sonnet-4-5").
//
// ok is false for any model not in the table. Callers MUST fail loudly on
// !ok rather than defaulting to a shape.
func AnthropicThinkingStyleFor(model string) (AnthropicThinkingStyle, bool) {
	s, ok := anthropicThinkingStyles[stripSnapshotSuffix(model)]
	return s, ok
}

// anthropicThinkingStyleModels returns the registered model ids, sorted. Test
// and diagnostic helper — keeps error messages and assertions deterministic.
func anthropicThinkingStyleModels() []string {
	out := make([]string, 0, len(anthropicThinkingStyles))
	for m := range anthropicThinkingStyles {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// stripSnapshotSuffix removes a trailing "-YYYYMMDD" dated-snapshot suffix.
// Anthropic ships some models under both a bare alias and a dated id; both name
// the same API surface, so both must resolve to the same generation.
func stripSnapshotSuffix(model string) string {
	i := strings.LastIndex(model, "-")
	if i < 0 || len(model)-i-1 != 8 {
		return model
	}
	for _, r := range model[i+1:] {
		if r < '0' || r > '9' {
			return model
		}
	}
	return model[:i]
}

// AnthropicEffortLevel maps this resolver's canonical qualitative effort to the
// Anthropic output_config.effort value.
//
// Anthropic's vocabulary is low/medium/high/xhigh/max; the three values this
// resolver can produce map 1:1. "off" has no effort value — it is expressed as
// disabled thinking — so ok is false for it.
func AnthropicEffortLevel(effort string) (string, bool) {
	switch effort {
	case ReasoningEffortLow, ReasoningEffortMedium, ReasoningEffortHigh:
		return effort, true
	}
	return "", false
}
