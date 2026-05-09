# M-AI-STEP-STREAMING-THINKING: API-level reasoning as separate stream variant

**Status**: Implemented (v0.18.8)
**Target**: v0.18.8 (patch on top of v0.18.7, ~3 hours sprint)
**Priority**: P1 (motoko TUI thinking-pane requirement, prompted by user feedback during v0.18.7 acceptance testing)
**Dependencies**: ✅ M-AI-STEP-STREAMING (v0.18.7) — establishes the `stepWithStream` API + `StreamChunk` ADT shape this extends.
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-09
**Source**: User question during v0.18.7 motoko_agent integration testing — "is it possible to have access to the thinking traces in a good way?" — when wiring v0.18.7 stepWithStream into motoko_agent's TUI, glm-5's `<thinking>...</thinking>` content was leaking into the visible pane. That led to the realization that motoko also needs API-level reasoning (Anthropic extended-thinking, OpenAI o1/o3 reasoning_content, Gemini thoughts) which v0.18.7 dropped.

---

## Problem statement

v0.18.7's `stepWithStream` exposed two `StreamChunk` variants — `ContentDelta` and `Usage`. Reasoning content from API-level thinking models was DROPPED by all three native SSE parsers:

- **Anthropic**: extended thinking enabled on claude-opus-4.5+ produces `content_block_delta` events with `delta.type:"thinking_delta"` carrying a `thinking` field. The v0.18.7 dispatcher only handled `text_delta` and `input_json_delta`; `thinking_delta` fell through the switch and was silently dropped.
- **OpenAI**: o1/o3 reasoning models emit `delta.reasoning_content` alongside `delta.content` on chat completion chunks. The v0.18.7 `ChatStepStreamDelta` struct only declared `content` (and `tool_calls`); `reasoning_content` was dropped at the JSON unmarshal step.
- **Gemini**: gemini-2.5+ thinking models emit content parts with a `thought:true` flag indicating the part carries reasoning. The v0.18.7 `part` struct didn't declare `Thought`; the parser couldn't distinguish reasoning text from answer text and concatenated everything into `Response.Text`.

The convention-level case (models that embed `<thinking>...</thinking>` text tags inline in their content output — glm-5, deepseek-r1, qwen-thinking) is **explicitly out of scope for this design**. Tag-convention reasoning arrives via the existing `ContentDelta` channel; consumers split with a tag-aware helper. This design covers only API-level reasoning where the provider tells us a chunk IS reasoning via a separate field.

## Goals

1. Surface API-level reasoning as a third `StreamChunk` variant, distinct from content.
2. Reasoning text MUST NOT leak into `StepResult.message.content` (which by API contract is assistant-visible content only). The existing `Response.ReasonTokens` field is the cost-telemetry channel; the new `ThinkingDelta` callback is the rendering channel.
3. Backward compatible: existing v0.18.7 callers that match only `ContentDelta`/`Usage` continue to work.
4. Per-provider symmetry: same callback shape across all three first-class providers (Anthropic / OpenAI / Gemini) so motoko's TUI doesn't need per-provider rendering paths.

## Solution

### ADT extension (additive)

```ailang
type StreamChunk =
    ContentDelta(string)     -- existing (v0.18.7)
  | ThinkingDelta(string)    -- NEW (v0.18.8)
  | Usage({...})             -- existing (v0.18.7)
```

### Per-provider mapping

| Provider | API field | Wire shape |
|---|---|---|
| Anthropic | `content_block_delta.delta.thinking` (when `delta.type:"thinking_delta"`) | New `Thinking string` field on `streamContentBlockDelta`; `thinking_delta` case in dispatcher fires `ai.StreamThinkingDelta` |
| OpenAI | `delta.reasoning_content` | New `ReasoningContent string` field on `ChatStepStreamDelta`; per-chunk loop fires `ai.StreamThinkingDelta` when non-empty |
| Gemini | `parts[].thought == true` (text in `parts[].text`) | New `Thought bool` field on `part`; parser branches on `thought` flag to fire `ai.StreamThinkingDelta` and exclude text from `Response.Text` |
| OpenRouter | Inherits whichever upstream provider's shape the routed-to model uses | Pass-through (uses `openai.ParseChatStepSSEStream`, which now handles `reasoning_content`) |
| Ollama, config-driven | NO-OP fallback path unchanged | Reasoning never surfaces — these providers don't expose API-level reasoning |

### Reasoning excluded from `Response.Text`

This is the load-bearing invariant: **reasoning flows ONLY through the `ThinkingDelta` callback, never into `Response.Text`**. Consumers that want to log/render reasoning capture it via the callback; consumers that don't care can drop it (or just not match on `ThinkingDelta`).

For Anthropic, this is enforced by NOT writing `delta.thinking` into the block's text builder (which `Response.Text` is built from). For OpenAI, `delta.reasoning_content` is never appended to `contentParts`. For Gemini, parts with `thought:true` short-circuit before the `textParts = append(...)` line.

`Response.ReasonTokens` (existing, populated by Anthropic/Gemini) is the cost-telemetry counterpart — it tracks the count without exposing the text.

## Conflict surface

This sprint touches:

- `internal/ai/provider.go` — adds `StreamThinkingDelta` struct + marker method on `StreamChunk`. Existing variants unchanged. Callers that type-switch on `StreamChunk` and have a `default` arm get unhandled-default for thinking; callers without a default arm silently ignore (Go behavior).
- `internal/ai/{anthropic,openai,gemini}/streamstep.go` — additive struct fields + new dispatch arms. Bit-for-bit identical wire bytes for non-thinking models.
- `internal/effects/ai_step.go` — `encodeStreamChunk` switch grows a new case. Default `return nil` arm catches anything unencodable (existing behavior).
- `std/ai.ail` — `StreamChunk` ADT grows a new constructor.

**Pattern-match exhaustiveness**: AILANG check `match` arms at type-check time. Adding a new ADT constructor IS a breaking change for any consumer module that pattern-matches `StreamChunk` without a wildcard arm. **Acceptable risk** because:
1. v0.18.7 was released today — minimal user adoption yet.
2. The fix is one line (add `ThinkingDelta(_) => ()` arm).
3. The variant is the entire point of v0.18.8.

External consumers identified: `examples/runnable/ai_streaming.ail` (updated in this sprint), `motoko_agent/src/core/agent_loop_v2.ail` (will be updated in the v0.18.8 motoko follow-up commit).

## Acceptance criteria

- [x] New `ai.StreamThinkingDelta` struct compile-time satisfies the `ai.StreamChunk` interface (marker method present).
- [x] Each of Anthropic / OpenAI / Gemini emits `ai.StreamThinkingDelta` for the appropriate API field.
- [x] Reasoning text does NOT appear in `Response.Text` for any provider (asserted by tests).
- [x] Encoder produces `TaggedValue{CtorName:"ThinkingDelta", Fields:[StringValue]}` matching the AILANG ADT shape.
- [x] WASM build clean.
- [x] All v0.18.7 unit tests still pass (no regression).
- [x] One new unit test per provider covering thinking-mode SSE shape.
- [x] One round-trip integration test verifying the encoder + AILANG callback bridge.
- [x] CHANGELOG + design-doc archive complete.
- [x] Example file updated with `ThinkingDelta` arm.

## Future work (deferred)

- **ToolCallDelta variant** (Phase 3): partial tool_use blocks streamed as they assemble. Requires per-provider JSON-fragment buffering. Scope: separate sprint.
- **Tag-convention splitter** (motoko-side helper): split `<thinking>...</thinking>` from `ContentDelta` content for models that don't emit API-level reasoning (glm-5, deepseek-r1). Lives in motoko_agent, not AILANG (AILANG can't know which provider's content uses which tag convention).
