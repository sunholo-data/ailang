# M-AI-OPENROUTER-REASONING-FIELD: surface reasoning from OR-routed thinking models

**Status**: Implemented (v0.18.9)
**Target**: v0.18.9 (one-day patch follow-up to v0.18.8)
**Priority**: P0 (motoko ThinkingDelta wiring shipped in v0.18.8 silently dropped DeepSeek + every OR-routed reasoning model)
**Dependencies**: ✅ M-AI-STEP-STREAMING-THINKING (v0.18.8)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-10
**Source**: 2026-05-10, Arni reported during motoko_agent acceptance: "Your right - Gemma models works with thinking streams, some minor change must have disabled it for the DeepSeek models". Investigation via direct curl probe of openrouter.ai/api/v1.

## Bug

v0.18.8's `internal/ai/openai/streamstep.go` parsed `delta.reasoning_content` (the OpenAI o1/o3 direct-API spec field). OpenRouter normalizes routed-model reasoning to `delta.reasoning` (a different field name). Confirmed by curl against `deepseek/deepseek-r1` and `anthropic/claude-opus-4.5` via OpenRouter:

```json
{"choices":[{"delta":{
  "content":"",
  "role":"assistant",
  "reasoning":"I need to add",
  "reasoning_details":[{"type":"reasoning.text","text":"I need to add","format":"unknown","index":0}]
}}]}
```

Field is `reasoning`, not `reasoning_content`. v0.18.8 parser silently dropped it. Result: every OR-routed thinking model (deepseek-r1, anthropic-via-OR, qwen-thinking) appeared to NOT emit reasoning — even though our v0.18.8 ThinkingDelta surface was correctly wired all the way through.

Why "Gemma worked but DeepSeek didn't" (Arni's observation): Gemma 3 isn't a thinking model — it doesn't emit any reasoning fields. Motoko's `[think]` panel for Gemma was rendering tag-convention `<thinking>...</thinking>` content embedded in the prompt format (via the v0.18.7 ContentDelta + TUI tag-stripping path). DeepSeek-R1's reasoning was being SENT correctly by OpenRouter but DROPPED at the AILANG parser layer.

Additionally: OpenRouter requires `"include_reasoning":true` in the request body to OPT IN to reasoning chunks. Without it, OR drops reasoning silently at the SOURCE regardless of which field name the underlying provider uses.

## Fix

Two-line additive parser change + one wire-flag addition:

1. `ChatStepStreamDelta` grew a `Reasoning string` field alongside the existing `ReasoningContent string`. Both surface as `ai.StreamThinkingDelta`. In practice only one fires per chunk; dual-field support is provider-shape defensive.

2. `marshalStepBodyWithProvider` refactored into more general `marshalStepBodyWithExtras` taking a list of pre-marshalled `"key":<value>` fragments. OpenRouter streaming path now injects `"include_reasoning":true` alongside the optional `"provider":{...}` routing field.

## Acceptance

- [x] Tests: `TestStreamStep_ParsesOpenRouterReasoningField` (openai), `TestStreamStep_IncludeReasoningOnTheWire` (openrouter).
- [x] All 8 v0.18.8 streaming tests still pass.
- [x] WASM build clean.
- [x] Direct-curl verification with deepseek/deepseek-r1 + anthropic/claude-opus-4.5 via OpenRouter confirmed both emit through `delta.reasoning`.

## Out of scope

- Per-provider reasoning_details metadata surfacing (Anthropic emits `format:"anthropic-claude-v1"`, DeepSeek emits `format:"unknown"`, etc.). Could be useful for provider-specific rendering, but not currently needed by motoko's `[reason]` panel.
- OpenRouter's `reasoning:{max_tokens, exclude}` request-side controls. Currently we just pass `include_reasoning:true` — finer-grained control deferred.
- Direct-API DeepSeek (without OpenRouter routing). DeepSeek's own API uses `delta.reasoning_content` per OpenAI spec, so v0.18.8 already covers it.
