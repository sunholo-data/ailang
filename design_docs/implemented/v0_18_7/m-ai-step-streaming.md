# M-AI-STEP-STREAMING: Typed-StepResult streaming via per-chunk callback

**Status**: Implemented (v0.18.7)
**Target**: v0.18.7 (patch on top of v0.18.6, ships within ~3 days of design freeze)
**Priority**: P1 (blocks motoko TUI streaming UX which Arni explicitly requested in [arniwesth/motoko_agent#10](https://github.com/arniwesth/motoko_agent/discussions/10); not blocking AILANG itself)
**Estimated**: ~1 day Phase 1 (~250-350 LOC + tests). Phase 2 (ToolCallDelta + ThinkingDelta) is a separate sprint
**Dependencies**: ✅ M-AI-PROMPT-CACHING (v0.18.4) — same shape (additive opt-in callback function); reuses the AIHandler interface evolution pattern. ✅ M-AI-STREAMING-HELPER (v0.15.0) — provides the `_ai_stream_call` builtin + `std/ai/streaming` SSE machinery this design unifies with the typed `step()` path.
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-09

---

## Source event

Today (2026-05-09), Arni Westh opened [arniwesth/motoko_agent#10 Discussion](https://github.com/arniwesth/motoko_agent/discussions/10) — "Bring back token-by-token stream rendering in TUI". Direct quote of the gap:

> Motoko's TUI previously rendered LLM output token-by-token, using the AILANG fork's `std/ai_motoko.callStreamResult`. After migrating to upstream AILANG (PR #3, v0.15.2 → v0.18.x), this capability was lost. The current agent loop calls `std/ai.stepWithCache()`, which drives the full SSE event loop internally in the Go runtime and returns a complete `StepResult` only after the model finishes responding. From the caller's perspective, the call blocks for 5–30 seconds with no visibility into partial output.

Motoko's TUI streaming infrastructure (event protocol, incremental markdown renderer, seq tracking) is fully built. The only missing piece is per-token visibility into `step()` / `stepWithCache()`.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Streaming is observational; doesn't change produced bytes vs non-streaming `step()` for the same input |
| A2: Replayability | 0 | Per-chunk callbacks aren't part of the trace contract; final StepResult IS, and matches non-streaming |
| A3: Effect Legibility | +1 | Callback's effects flow through row polymorphism — `(StreamChunk) -> () ! ρ` extends `! {AI}` to `! {AI \| ρ}` automatically |
| A4: Explicit Authority | 0 | No new caps; AI cap remains the gate. Callback runs under caller's existing effect ceiling |
| A5: Bounded Verification | 0 | Type-checking unaffected (additive function + ADT) |
| A6: Safe Concurrency | 0 | Callback runs synchronously inside the SSE read loop — single-threaded, predictable |
| A7: Machines First | +2 | Closes a real UX regression for AI agents (motoko) consuming AILANG's typed AI surface. Per-chunk visibility = better latency observability for any future AILANG agent |
| A8: Minimal Syntax | 0 | One additive function + small ADT; no new syntax |
| A9: Cost Visibility | +1 | `Usage` chunk surfaces token counts during stream rather than only at end — finer-grained cost-budget gating becomes possible |
| A10: Composability | +1 | Same `Message` / `ToolSchema` / `CacheBreakpoint` types as `step()` / `stepWithCache()`; callers can wrap the streaming variant in higher-order combinators |
| A11: Structured Failure | +1 | Returns same `Result[StepResult, AIError]` as `step()` — failure modes typed the same way |
| A12: System Boundary | +1 | Codifies "streaming chunk → typed accumulator" as the AILANG↔Go boundary; pre-existing `_ai_call_stream` already does this for string-only output, this generalizes to typed StepResult |

**Net Score: +7** → **Decision: ✅ Strong proceed** (A7 +2 + A12 +1 are the load-bearing axioms; the design closes the real UX regression Arni surfaced)

### Hard Violation Check

- [x] A1 (Determinism): Streaming is observational; same output as non-streaming for same input
- [x] A3 (Effects): Callback effects flow through row polymorphism — explicit, not hidden
- [x] A4 (Authority): No new capabilities granted; AI cap still gates everything
- [x] A7 (Machines First): The whole point — restores typed-streaming UX for AI agents

---

## Problem Statement

**Current state (v0.18.6).** Three streaming-adjacent surfaces in stdlib, but the missing diagonal is exactly what motoko needs:

| Layer | Returns typed result? | Streams chunks? |
|---|---|---|
| `std/ai.callStream*` (v0.15.1) | ❌ string-only output | ✅ SSE-based |
| `std/ai/streaming.openaiCompatStream` / `anthropicStream` (v0.15.0) | ❌ raw SSE chunks (caller parses JSON) | ✅ |
| **`std/ai.step` / `stepWithCache` (v0.17/v0.18.4)** | **✅ full `StepResult`** | **❌ blocking, no chunks** |
| `std/stream.ssePost` etc. (v0.18.4) | ❌ raw SSE | ✅ |

Motoko needs the **typed StepResult ✅ + per-chunk callback ✅** quadrant — currently empty.

**Three holes in the existing path** (despite shipped streaming infra):

1. **No typed StepResult from streaming calls.** `anthropicStream` returns `Result[StreamConn, StreamErrorKind]`. Caller parses raw `SSEData(eventType, data)` JSON manually and accumulates `tool_calls` / `input_tokens` / `cache_read_input_tokens` themselves. The whole point of `step()` was to give callers structured output instead of JSON parsing.

2. **Built-in providers don't expose this path.** Comment in [std/ai/streaming.ail:15-17](../../../std/ai/streaming.ail) says: *"Built-in providers (openai, anthropic, gemini, ollama, openrouter) are NOT routable through this in v1 — they have their own streaming code paths in future milestones."* Motoko's `openrouter/anthropic/claude-haiku-4-5` model can't use `anthropicStream` directly without first being wrapped as a `[[ai_provider]]` block in `ailang.toml` — friction Arni shouldn't need to hit.

3. **`parseDelta` typed extraction is deferred.** [streaming.ail:65-72](../../../std/ai/streaming.ail) defines `TokenDelta` but the comment explicitly says: *"parseDelta is deferred to v1.1. For v1, callers extract token text from raw SSEData JSON manually."*

**Impact:**
- Motoko's TUI hides 5-30s latency behind a "thinking..." spinner with no incremental text. Every motoko user feels this on every turn
- Any future AILANG agent wanting typed-result streaming hits the same wall
- Re-implementing Anthropic/OpenAI SSE parsers in userland AILANG to recover this would be fragile (~150-200 LOC per provider) and would diverge whenever upstream `step()` internals change

---

## Goals

**Primary Goal:** Add `std/ai.stepWithStream` — same return shape as `step()`/`stepWithCache()`, plus a per-chunk callback that fires during SSE processing — so motoko (and any future agent) can render LLM output incrementally without giving up typed StepResult, tool dispatch, cost tracking, or cache metrics.

**Success Metrics:**
- `stepWithStream` produces a `StepResult` byte-identical to what `stepWithCache` would return for the same input
- Per-chunk callback fires ≥2× during a typical 200-token Anthropic response (verified via integration test gated behind `AILANG_INTEGRATION_TESTS=1`)
- Existing `step()` / `stepWithCache()` callers unchanged (regression: 22 v0.18.4 unit tests + all `internal/ai/*` step tests pass without modification)
- Motoko's `agent_loop_v2.ail` opt-in is a 5-10 line change (single call site replacement, mirrors v0.18.4 cache opt-in)
- `ailang lock + ailang check` clean for both AILANG and downstream motoko_agent

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `StreamChunk` ADT shape (Phase 1: `ContentDelta + Usage` only vs richer set) | The ADT becomes a public stdlib type — adding variants later is back-compat (just a new constructor) but renaming/removing existing ones isn't | human | design | high |
| Phase 1 NO-OP fallback for Gemini/Ollama vs full streaming impl | Gemini's streamGenerateContent + Ollama's stream:true have different shapes from Anthropic/OpenAI. Doing them in Phase 1 doubles the sprint. Phase 1 ships fallback (call regular Step, fire one final ContentDelta + Usage) | human | design | low |
| Callback effect row: `! {IO}` baseline vs caller-supplied row | Hard-coding `! {IO}` would prevent callbacks that need other effects (e.g. emit_event uses `! {IO}` but a future caller may need `! {Net}`). Row-polymorphic — caller's effect ceiling extends transitively | agent | implementation | low |
| AIHandler interface evolution: extend with `StepWithStream` method (5th interface method) vs separate StreamingHandler interface | Mirror v0.18.4 `StepWithCache` precedent — same interface, same evolution pattern. Stub handlers (test fakes, WasmAIHandler) can default-delegate to Step | agent | implementation | low |
| Tool-call streaming: omit from Phase 1 (Phase 2 follow-up sprint) vs include | Anthropic emits tool_use input_json_delta deltas; OpenAI streams tool_calls[].function.arguments fragments. Both require buffering until block close — non-trivial per provider. Motoko's IMMEDIATE need is content streaming for visible UX, not tool-call visibility. Phase 2 separate sprint | human | design | med |
| Backpressure / pause callback: omit | Synchronous callback inside SSE read loop = simple model. Caller can throw to abort. No pause-resume protocol in v1 | agent | design | low |

### Design Freeze

Before implementation begins:

- [ ] `StreamChunk` v1 ADT = `ContentDelta(string) | Usage({input_tokens, output_tokens, cache_read_input_tokens, cache_creation_input_tokens})` only
- [ ] Phase 1 providers with full streaming: Anthropic (direct + via OpenRouter prefix-routing), OpenAI direct
- [ ] Phase 1 NO-OP fallback providers: Gemini, Ollama, configdriven (via `[[ai_provider]]`)
- [ ] AIHandler gains 7th method `StepWithStream(model, messages, tools, breakpoints, onChunk)` — same evolution shape as v0.18.4 (which added `StepWithCache` as 4th)
- [ ] Callback effect row is row-polymorphic (`(StreamChunk) -> () ! ρ`); not hardcoded to `! {IO}`
- [ ] Tool-call streaming variants (`ToolCallDelta`, `ThinkingDelta`) deferred to a Phase 2 separate sprint; Phase 1 still buffers tool_use blocks internally and surfaces them in the final `StepResult.ToolCalls` (just no per-chunk visibility into them)

---

## Solution Design

### Overview

Add `stepWithStream` as a 5-arg AILANG function that:
1. Takes the same first 4 arguments as `stepWithCache` (`model, messages, tools, cache_breakpoints`)
2. Plus a 5th `on_chunk: (StreamChunk) -> () ! ρ` callback
3. Returns `Result[StepResult, AIError]` — bit-for-bit same shape as `stepWithCache`'s return
4. Each provider's streaming Go code (already exists for Anthropic/OpenAI/OpenRouter via `_ai_stream_call`) gets wrapped in a typed-StepResult accumulator that ALSO invokes the user callback per content chunk

### Architecture

**Components:**

1. **AILANG-side surface** (`std/ai.ail`):
   - New type `StreamChunk` (sum of `ContentDelta(string)` and `Usage({...})`)
   - New function `stepWithStream(model, messages, tools, cache_breakpoints, on_chunk)`
   - Both exported

2. **Builtin** (`internal/builtins/ai_step.go`):
   - New `_ai_step_with_stream` (5-arg) — model, messages list, tools list, cache_breakpoints list, on_chunk callback (closure value)

3. **Effect op** (`internal/effects/ai_step.go`):
   - New `aiStepWithStream` — decodes args (existing `decodeMessages`/`decodeToolSchemas`/`decodeCacheBreakpoints` reused), wraps the closure as a Go callback, calls `ctx.AI.StepWithStream(...)`, encodes result back as `Result[StepResult, AIError]`

4. **AIHandler interface** (`internal/effects/ai.go`):
   - 7th method: `StepWithStream(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error)`
   - All implementations updated (per v0.18.4 precedent: 5 implementations including stubs and WasmAIHandler)

5. **Per-provider streaming + typed accumulator** (full impl: `internal/ai/anthropic/`, `internal/ai/openai/`, `internal/ai/openrouter/`):
   - Anthropic: open `messages?stream=true` SSE; loop over `event: content_block_delta` (fire `ContentDelta`), `event: message_delta` (accumulate stop_reason + final usage → fire `Usage`), `event: message_stop` (terminate). Tool blocks buffered internally for `StepResult.ToolCalls`
   - OpenAI / OpenRouter: open `chat/completions?stream=true`; loop over `data: {delta:{content:"..."}}` (fire `ContentDelta`), final `data: {usage:{...}}` (fire `Usage`)
   - NO-OP fallback providers (Gemini, Ollama, configdriven): call existing `Step`/`StepWithCache`, fire one synthetic `ContentDelta(text)` + one `Usage` from the final response. Caller still gets a single callback firing — UX degraded but contract preserved

6. **`ai.StreamChunk` Go type + `ai.Response.ToStreamUsage()` helper**:
   - In `internal/ai/provider.go` — `StreamChunk` interface with `streamChunkContent` and `streamChunkUsage` concrete types

### Implementation Plan

**Phase 1: AILANG-side surface + AIHandler evolution** (~2 hours, ~80 LOC)
- [ ] Add `StreamChunk` Go type to `internal/ai/provider.go`
- [ ] Add `StreamChunk` AILANG type to `std/ai.ail` + `stepWithStream` function (just declares the surface; impl follows)
- [ ] Add 7th AIHandler interface method `StepWithStream(...)` in `internal/effects/ai.go`
- [ ] Implement on `AIContext` (delegate), `StubAIHandler` (call `Step` + fire one ContentDelta), `fakeStepHandler` (test stub: capture callback for assertion), `ai.Handler` (real entry; delegates to provider), `WasmAIHandler` (return "not supported in WASM" matching `Step`/`StepWithCache` precedent)
- [ ] Add `_ai_step_with_stream` builtin
- [ ] Add `aiStepWithStream` effect op + closure wrapper (Go callback wraps the AILANG closure)

**Phase 2: Anthropic full streaming impl** (~2 hours, ~120 LOC)
- [ ] In `internal/ai/anthropic/`: new `streamStep.go` (or extend existing `step.go`) with `StreamStep(ctx, req, onChunk)` method
- [ ] Set `stream: true` on the Messages API request body
- [ ] Parse SSE stream: handle `message_start` (init Response), `content_block_start` (note tool_use blocks for buffering), `content_block_delta` (fire `ContentDelta` for text deltas; buffer for tool_use input_json_delta), `content_block_stop` (close tool_use block, finalize ToolCall), `message_delta` (accumulate usage + stop_reason), `message_stop` (return assembled `*ai.Response`)
- [ ] Same `cache_control` plumbing as v0.18.4 (reuse existing `applyCacheHints` helper)
- [ ] Same Bedrock-strict tool_use_id correlation handling as v0.18.3 (reuse existing path)
- [ ] Wire into `Handler.StepWithStream` (pass-through)

**Phase 3: OpenAI / OpenRouter streaming impl** (~1.5 hours, ~80 LOC)
- [ ] In `internal/ai/openai/`: same shape — set `stream: true`, parse `data:` SSE lines, fire `ContentDelta` per `delta.content`, fire `Usage` from final `data` event with `usage`
- [ ] In `internal/ai/openrouter/`: dispatch to OpenAI-shape streaming OR reuse `applyCacheHintsForRoute` model-prefix logic from v0.18.4 to route `anthropic/...` models through Anthropic-shape streaming (similar prefix-routing pattern)
- [ ] Reuse existing OpenAI `BuildChatStepRequest` / `ParseChatStepResponse` helpers where applicable (they're already exported)

**Phase 4: NO-OP fallback providers + integration test + docs** (~1.5 hours, ~70 LOC)
- [ ] Gemini, Ollama: add `StepWithStream` that calls `Step`/`StepWithCache` then fires one synthetic `ContentDelta(text)` + `Usage`. Document this as a v1 limitation; full streaming is a Phase 2 sprint
- [ ] Configdriven: same fallback for v1 (note: `[[ai_provider]]` already has streaming via `openaiCompatStream` — could be unified later)
- [ ] Unit test (mocks): assemble content from on_chunk callbacks, assert final string equals `StepResult.Text`
- [ ] Integration test (gated behind `AILANG_INTEGRATION_TESTS=1`): real Anthropic call, assert callback fires ≥2 times for a non-trivial response, assert assembled content == final `StepResult.Text`
- [ ] CHANGELOG entry + update [`std/ai/streaming.ail`'s "v1 limitations"](../../../std/ai/streaming.ail) section to mention `stepWithStream` as the typed alternative to raw `openaiCompatStream`/`anthropicStream`

### Files to Modify/Create

**New files:**
- `internal/ai/anthropic/streamStep.go` (~80 LOC) — Anthropic streaming impl
- `internal/ai/openai/streamStep.go` (~50 LOC) — OpenAI streaming impl
- `internal/ai/openrouter/streamStep.go` (~30 LOC) — model-prefix streaming dispatch
- `internal/effects/ai_step_with_stream_test.go` (~80 LOC) — round-trip + content-assembly tests

**Modified files:**
- `std/ai.ail` (+25 LOC) — `StreamChunk` type, `stepWithStream` function
- `internal/ai/provider.go` (+20 LOC) — Go-side `StreamChunk` interface + 2 concrete types
- `internal/ai/handler.go` (+15 LOC) — `Handler.StepWithStream` delegating to provider
- `internal/effects/ai.go` (+30 LOC) — interface method + AIContext + StubAIHandler delegations
- `internal/effects/ai_step.go` (+80 LOC) — `aiStepWithStream` effect op + closure-wrapping helper
- `internal/effects/ai_step_test.go` (+10 LOC) — fakeStepHandler.StepWithStream stub
- `internal/builtins/ai_step.go` (+60 LOC) — `_ai_step_with_stream` builtin registration
- `internal/ai/anthropic/handler.go` or step.go (+10 LOC) — wire StreamStep into the existing Handler structure
- `internal/ai/openai/step.go` / `internal/ai/openrouter/step.go` (+5 LOC each) — wire StreamStep
- `internal/ai/gemini/step.go` (+10 LOC) — NO-OP fallback delegating to Step
- `internal/ai/ollama/step.go` (+10 LOC) — NO-OP fallback
- `internal/ai/configdriven/...` (+10 LOC) — NO-OP fallback
- `cmd/wasm/effects.go` (+10 LOC) — WasmAIHandler.StepWithStream returning "not supported in WASM" (per v0.18.4 precedent)

---

## Examples

### Example 1: Default behavior — `stepWithCache` unchanged

```ailang
import std/ai (stepWithCache, Message, ToolSchema)

let r = stepWithCache(model, messages, tools, breakpoints);
-- r.message.content has the assembled text
-- r.tool_calls has any tool invocations
-- r.cache_read_input_tokens has cache telemetry
```

This call is **bit-for-bit unchanged** post-this-milestone. Existing motoko `stepWithCache` callers continue to work without edits.

### Example 2: New `stepWithStream` — typed result + per-chunk callback

```ailang
import std/ai (stepWithStream, StreamChunk, Message, ToolSchema, CacheBreakpoint)
import std/io (println)

let on_chunk = func(chunk: StreamChunk) -> () ! {IO} {
  match chunk {
    ContentDelta(text) => println(text),    -- print as it arrives
    Usage(u) => println("tokens: in=${show(u.input_tokens)} out=${show(u.output_tokens)}")
  }
};

let breakpoints = [{ position: "system", ttl: "ephemeral" }];
match stepWithStream(model, messages, tools, breakpoints, on_chunk) {
  Ok(r) => {
    -- r is the SAME StepResult shape as stepWithCache returns
    -- Tool dispatch, cost tracking, cache metrics all populated
    println("");
    println("done — finish_reason: ${r.finish_reason}")
  },
  Err(e) => println("error: ${e.message}")
}
```

### Example 3: motoko opt-in (cross-repo, ships in motoko PR after this lands)

In `motoko_agent/src/core/test/stub_step.ail` (the same file we patched in v0.18.4 for cache opt-in):

```diff
-LiveAI => {
-  { result: stepWithCache(model, msgs, tools_with_extensions(rt), system_prompt_cache_breakpoint()), next_provider: LiveAI }
-},
+LiveAI => {
+  let on_chunk = make_stream_event_emitter(session_id, stream_id, next_seq);
+  { result: stepWithStream(model, msgs, tools_with_extensions(rt), system_prompt_cache_breakpoint(), on_chunk), next_provider: LiveAI }
+},
```

`make_stream_event_emitter` builds a closure that calls motoko's existing `emit_event` to fire `thinking_delta` events with `seq` numbers — the TUI's incremental markdown renderer is already wired for these events (no TUI changes needed).

---

## Conflict Surface

This design touches `internal/ai/` (multiple providers), `internal/effects/`, `internal/builtins/`, and `std/ai.ail` — same area as M-AI-PROMPT-CACHING (v0.18.4). Required Conflict Surface analysis:

### Syntactic positions touched

- AILANG `std/ai`: adds export `StreamChunk` type + export `stepWithStream` function. Additive, no removals/renames.
- `internal/ai.Request`: no struct changes (cache_breakpoints field already there from v0.18.4)
- `internal/ai.Response`: no struct changes (token-count fields already there)
- New `internal/ai.StreamChunk` Go type — interface + 2 concrete types. Doesn't conflict with anything
- `effects.AIHandler` interface: 7th method added. Same evolution pattern as v0.18.4 (which added `StepWithCache` as the 4th)

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| `effects.AIHandler` interface | `Call`, `CallJson`, `CallImage`, `CallImageBase64`, `Step`, `StepWithCache` (6 methods today) | adding `StepWithStream` as 7th |
| `std/ai` exports | type + function exports (`AIError`, `Message`, `ToolCall`, `ToolSchema`, `StepResult`, `CacheBreakpoint`, `step`, `stepWithCache`, `call*`, etc.) | adding `StreamChunk` type + `stepWithStream` function |
| `internal/ai/<provider>/step.go` | `Step` (non-streaming) + sometimes `StreamStep` for callStream-only path | adding new typed-StepResult-streaming variant; reuses existing SSE Go code where present |

### Disambiguation strategy

- All additions are NEW exported names (`StreamChunk`, `stepWithStream`, `StepWithStream` method) — no collision risk with existing code
- Existing `step()` and `stepWithCache()` keep their exact wire shape, return type, AND Go-side dispatch. Empty `cache_breakpoints` continues to work bit-for-bit identically; no callers of those need to change anything
- AIHandler interface gains a 7th method — every implementation (5 existing) needs `StepWithStream` added. Stubs and Wasm return "not supported" or default-delegate to `Step`. Same churn as v0.18.4 — proven pattern

### Programs that MUST still work (regression fixtures)

These exist today and must continue passing post-change:
1. `internal/ai/anthropic/step_test.go` — all unchanged
2. `internal/ai/openai/step_test.go` — all unchanged
3. `internal/ai/openrouter/step_test.go` — all unchanged
4. `internal/effects/ai_step_test.go` (v0.17.0 + v0.18.4 tests) — all unchanged
5. `internal/effects/ai_step_with_cache_test.go` (v0.18.4 22 unit tests) — all unchanged
6. `internal/ai/anthropic/cache_test.go` (v0.18.4) — all unchanged
7. `examples/runnable/ai_caching.ail` (v0.18.4 demo) — must still type-check + run cleanly
8. motoko_agent `src/core/test/stub_step.ail` — must still compile pre-opt-in (motoko's stepWithCache call remains valid)

### What deliberately changes

- `effects.AIHandler` interface widens by one method. This is INTENTIONAL — same shape as v0.18.4. Anyone embedding AILANG with a custom `AIHandler` implementation needs to add a `StepWithStream` method (which can default-delegate to `Step` for "I don't care about streaming" handlers). Documented as a single-line stub addition, with `WasmAIHandler` as the canonical example
- `std/ai/streaming.ail`'s "v1 limitations" comment block updated to point at `stepWithStream` as the typed-StepResult alternative to raw `openaiCompatStream`/`anthropicStream`. No code change in streaming.ail; just doc

---

## Success Criteria

- [ ] `stepWithStream` produces a `StepResult` whose fields match what `stepWithCache` would have returned for the same input (verified via test fixture comparing both calls)
- [ ] Per-chunk callback fires ≥2 times for a non-trivial Anthropic response (integration test with `AILANG_INTEGRATION_TESTS=1`)
- [ ] Assembling content from `on_chunk` callbacks produces a string equal to `StepResult.Text` (unit test with mock provider)
- [ ] All existing `step()` and `stepWithCache()` tests pass without modification
- [ ] All existing `internal/ai/<provider>/*_test.go` tests pass without modification
- [ ] WASM build succeeds (preempts the v0.18.4 trap — `WasmAIHandler.StepWithStream` returns "not supported in WASM" matching its `Step`/`StepWithCache` siblings)
- [ ] `examples/runnable/ai_streaming.ail` (new) — runnable demo of `stepWithStream` with a stub handler
- [ ] `std/ai/streaming.ail` doc comment updated to point at `stepWithStream`
- [ ] CHANGELOG entry under v0.18.7
- [ ] Cross-repo motoko opt-in path documented in design doc (the actual motoko commit ships separately on PR #7 successor)

---

## Testing Strategy

**Unit tests:**
- `aiStepWithStream` round-trip: AILANG closure value → Go callback → fires per chunk → encoded result back to AILANG `Result[StepResult, AIError]`
- `decodeStreamChunk` (no decoder needed — chunks flow Go→AILANG only); but encoder for `StreamChunk` Ailang values
- `StreamChunk` Go type variants serialize correctly (test with mock callback that captures all chunks)
- Empty `cache_breakpoints` + non-streaming-aware handler path: stub provider returns full text → exactly one synthetic `ContentDelta` + one `Usage` callback fires

**Integration tests** (gated by `AILANG_INTEGRATION_TESTS=1`):
- Real Anthropic streaming call: assert per-chunk callbacks fire ≥2 times; assembled content == final `StepResult.Text`; tool_calls properly buffered and present in final `StepResult.ToolCalls`
- Real OpenAI streaming call: same shape

**Regression-surface tests:**
- One snapshot test per existing v0.18.4 test in `internal/ai/anthropic/cache_test.go` — verify cache_control wire bytes still match for the (non-streaming) `Step` call
- Snapshot of `_ai_step_with_cache` builtin signature (golden file already exists from v0.18.4)
- Snapshot of new `_ai_step_with_stream` builtin signature (new golden file)

**Manual testing:**
- `examples/runnable/ai_streaming.ail` — print each chunk; verify the print stream is incremental (not one bulk dump at end)
- Motoko opt-in (after cross-repo commit lands): verify TUI shows token-by-token rendering in a real eval-suite run

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Closure-wrapping helper name in `internal/effects/ai_step.go` — agent may choose
- Exact Go callback signature (`func(*ai.StreamChunk) error` vs `func(*ai.StreamChunk)`) — agent may choose; should match what's ergonomic in the per-provider streaming paths
- Whether to ship `examples/runnable/ai_streaming.ail` against `--ai-stub` (works without API key) or against a real provider with conditional skip — agent may choose
- Where the OpenRouter prefix-routing for streaming lives (extracted helper vs inline in `streamStep.go`) — agent may choose
- StubAIHandler's StepWithStream behavior: fire one ContentDelta with the full default response, OR don't fire at all (since stubs don't account tokens) — agent may choose; recommend the former for compatibility

---

## Non-Goals

**Not attempted in this milestone:**

- **`ToolCallDelta` variant** — Anthropic streams tool_use input_json_delta deltas; OpenAI streams tool_calls[].function.arguments fragments. Both require buffering the JSON until block-close, then emitting a complete `ToolCall`. Phase 2 — separate sprint M-AI-STEP-STREAMING-TOOLS
- **`ThinkingDelta` variant** — Anthropic-extended-thinking-specific. Rare in current motoko usage. Phase 2 if/when needed
- **Full streaming for Gemini/Ollama** — they have streaming APIs but different shapes (Gemini's `streamGenerateContent`, Ollama's NDJSON `stream:true`). Phase 1 ships fallback (call regular Step + fire one synthetic ContentDelta + Usage). Full impl is a Phase 2 sprint per provider
- **Backpressure / pause-resume callback semantics** — callback runs synchronously inside the SSE read loop; caller can `panic` to abort. No flow control protocol in v1
- **Unification of `openaiCompatStream`/`anthropicStream` (raw SSE) WITH `stepWithStream` (typed)** — they remain two surfaces. The raw path stays useful for callers who want SSE without typed parsing (e.g. proxying to a non-AILANG client). Document the choice in the streaming.ail v1 limitations section
- **`StreamChunk.Done` sentinel** — the `Result[StepResult, AIError]` return value already signals completion; a redundant Done variant would just be noise

---

## Timeline

**Day 1** (~5-6 hours):
- Phase 1: AILANG-side surface + AIHandler evolution (2h)
- Phase 2: Anthropic full streaming impl (2h)
- Phase 3: OpenAI / OpenRouter streaming impl (1.5h)

**Day 2** (~2 hours, mostly tests + docs):
- Phase 4: NO-OP fallbacks + integration test + CHANGELOG + doc updates

**Total: ~7-8 hours across 2 calendar days**

After the AILANG sprint lands as v0.18.7, motoko's cross-repo opt-in is a 5-line change in `motoko_agent/src/core/test/stub_step.ail` (mirrors v0.18.4 cache opt-in pattern). Arni can also opt in himself without coordination — the surface is stable.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| AIHandler 7-method interface breaks downstream embeddings (anyone implementing custom AIHandler needs to add StepWithStream) | Medium | Same shape as v0.18.4 (which added StepWithCache as 4th method). Stub-delegation pattern documented; WasmAIHandler is the canonical "I don't support this" example. v0.18.4 didn't break any known consumer; this won't either |
| WASM build fails (v0.18.4 trap) | Medium | **Preemptive `GOOS=js GOARCH=wasm go build ./cmd/wasm` runs BEFORE tagging the release** — codified in v0.18.5+ release-manager pre-tag check. WasmAIHandler gets an explicit StepWithStream stub returning "not supported in WASM" |
| Anthropic SSE parsing has edge cases (incomplete deltas, stream interruption) | Medium | Reuse existing `internal/ai/anthropic` SSE-parsing code from `_ai_stream_call` path which has been stable since v0.15.0. Only the typed-accumulator wrapper is new |
| Tool-call streaming gets requested as Phase 2 but never lands → motoko stuck on no-tool-call-deltas indefinitely | Low | Document in this design that `ToolCallDelta` is Phase 2 of a separately-trackable sprint M-AI-STEP-STREAMING-TOOLS. motoko already gets full tool_calls in the final `StepResult.ToolCalls` — only per-chunk visibility into tool_call args is missing, not the tool calls themselves |
| Callback misbehavior (slow/blocking callback stalls SSE read) | Low | Documented contract: "callback runs synchronously inside the SSE read loop; should not block on I/O. For long callback work, queue and process async." No runtime enforcement — same as v0.18.4 cache opt-in's effect-row contract |

---

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_18_4/m-ai-prompt-caching.md](../../implemented/v0_18_4/m-ai-prompt-caching.md) — **direct precedent**. Same shape (additive opt-in callback function), same AIHandler interface evolution pattern (added `StepWithCache` as 4th method, this adds `StepWithStream` as 7th). The High-Impact Decisions structure in this doc deliberately mirrors v0.18.4's
- [design_docs/planned/v0_17_0/m-ai-streaming-helper.md](../v0_17_0/m-ai-streaming-helper.md) (status: implemented in v0.15.0) — `_ai_stream_call` builtin + `std/ai/streaming.{openaiCompatStream,anthropicStream}`. Provides the per-provider SSE Go code this design wraps with a typed accumulator
- [design_docs/implemented/v0_11_0/m-streaming-zip-xml.md](../../implemented/v0_11_0/m-streaming-zip-xml.md) — earlier streaming infra (general `std/stream` primitives `sseConnect`, `ssePost`, `withSSE`, `onEvent`, `runEventLoop`)
- [design_docs/implemented/v0_18_3/m-motoko-hybrid-tool-correlation.md](../../implemented/v0_18_3/m-motoko-hybrid-tool-correlation.md) — Bedrock tool_use_id correlation. The streaming Anthropic impl must preserve this correctness for tool_use blocks streamed in fragments

**Auto-search results (lower relevance, retained for review):**
- [design_docs/planned/v1_0_0/m-agent-orchestration.md](../v1_0_0/m-agent-orchestration.md) (0.33) — different concern (multi-agent), not directly related

**External / cross-repo:**
- [arniwesth/motoko_agent#10 Discussion](https://github.com/arniwesth/motoko_agent/discussions/10) — source event
- AILANG `std/ai/streaming.ail` — explicit v1 limitations that this milestone partially closes
- `internal/builtins/ai.go:294-450` — existing `_ai_stream_call` + `_ai_call_stream` builtins (the prior art this generalizes to typed StepResult)

---

## References

- [Design Axioms](/docs/references/axioms) — A7 (Machines First) and A12 (System Boundary) are the load-bearing axioms here
- AILANG `std/ai.ail` — current `step` / `stepWithCache` surface
- AILANG `internal/effects/ai.go` — current AIHandler interface (becomes 7-method post-this)
- AILANG `internal/ai/anthropic/step.go` + `internal/ai/openai/step.go` — current non-streaming step paths to wrap with streaming variant

---

## Future Work

- **Phase 2 — `ToolCallDelta` variant** (M-AI-STEP-STREAMING-TOOLS): per-fragment visibility into tool_use input_json_delta and tool_calls argument deltas. Requires per-provider JSON-fragment buffering. Separate sprint
- **Phase 2 — `ThinkingDelta` variant**: Anthropic extended-thinking deltas. Separate sprint when Anthropic's thinking-mode usage in motoko grows
- **Phase 2 — Full streaming for Gemini/Ollama**: their SSE/NDJSON shapes differ from Anthropic/OpenAI. Per-provider sprints
- **Unify `openaiCompatStream`/`anthropicStream` with `stepWithStream`**: deprecate the raw path in favor of the typed one. After at least 1-2 minor versions of stepWithStream usage to verify no edge cases the raw path covered are missed
- **Cancellation token**: pass a context/cancel handle into stepWithStream so callers can abort an in-flight stream cleanly. Currently caller can `panic` from the callback to break the loop, but a typed cancel is nicer

---

**Document created**: 2026-05-09
**Author**: Claude Opus 4.7 + Mark, surfaced by arniwesth via motoko_agent#10
