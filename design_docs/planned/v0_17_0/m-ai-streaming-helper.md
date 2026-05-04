# M-AI-STREAMING-HELPER: AI Token Streaming Helper + Cross-Domain Discovery

**Status**: Implemented (M1+M2+M3 landed 2026-05-04, ~5.5 hours, tests green; awaiting v0.15.0 release)
**Target**: v0.15.0 (originally planned v0.17.0; pulled forward because all prerequisites had already shipped in parallel threads)
**Priority**: P1 (motivated by external-consumer evidence — Motoko fork built a parallel streaming subsystem because upstream `std/stream` wasn't discoverable from the AI domain)
**Estimated**: ~6 hours (revised down from 1-2 days; see "Updated estimate" note below)
**Dependencies**: ALL SHIPPED:
- **[M-AI-PROVIDER-CONFIG](../v0_15_0/m-ai-provider-config.md)** ✅ — landed M1-M4 (2026-05-04). `[ai_provider.streaming]` schema in place, registry harvest wired into `setupAIHandler`. 95 tests passing.
- **`_stream_sse_post` builtin** ✅ — landed in v0.15.0 (parallel thread, [internal/effects/stream_sse.go:267](../../../internal/effects/stream_sse.go#L267) and [internal/builtins/stream.go:439](../../../internal/builtins/stream.go#L439)). Specifically for AI API streaming via HTTP POST + SSE — exactly the primitive this milestone needs.
- **[M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md)** ✅ — landed v0.15.0. Bare `!{AI}` desugars to `!{AI[mode=fixed]}`; routing-capable `!{AI[mode=routeable]}` works at type level.

**Folds alongside** [M-EXTERNAL-CONSUMER-DX](./m-external-consumer-dx.md) (still v0.17.0).

### Updated estimate (2026-05-04)

Original estimate: 1-2 days (~8-12 hours). The parallel thread shipped two pieces of infrastructure that this milestone was scoped to also build:

1. `_stream_sse_post` is the load-bearing Go-side primitive (POST-based SSE) that the streaming dispatch needs. **Done.**
2. M-AI-PROVIDER-CONFIG's `[ai_provider.streaming]` schema parsing, `applyAuth`, and request-shape body construction (`buildRequestBody` in [internal/ai/configdriven/shapes.go](../../../internal/ai/configdriven/shapes.go)) are all reusable. **Done.**

Remaining scope: a thin AILANG module ([std/ai/streaming.ail](../../../std/ai/streaming.ail), to be created) plus possibly a small Go bridge that combines the existing pieces — registry lookup → body construction → auth → `_stream_sse_post`. Roughly **~100 LOC AILANG + ~80 LOC Go bridge + tests**. Revised estimate: **~6 hours** for full M1+M2.

### Implementation amendments (2026-05-04, post-execution)

Two deviations from this design doc were made during sprint execution. Both are documented here so the design record reflects what shipped:

1. **One new builtin (`_ai_stream_call`) added** — the design doc's "no new builtin" constraint was incompatible with the AI cap + budget integration requirement. Streaming returns `Result[StreamConn, StreamErrorKind]` (not `string`) and so cannot route through the existing `_ai_call` path. Hard architectural constraint preserved: NO new effect (we reuse `AI`, `Stream`, `Net`).

2. **Cycle-resolution placement** — the `aiStreamCall` op lives in [cmd/ailang/configdriven_streaming.go](../../../cmd/ailang/configdriven_streaming.go) rather than `internal/effects/`. The straightforward placement (effects → configdriven → telemetry → effects) creates an import cycle. cmd/ailang already imports both `internal/effects` and `internal/ai/configdriven` cleanly, so the op registers there at package init via `effects.RegisterOp("AI", "streamCall", ...)`. The builtin (`_ai_stream_call` in `internal/builtins/ai.go`) dispatches via `effects.Call(ctx, "AI", "streamCall", args)` — same pattern as all other AI ops.

3. **v1 simplifications in `std/ai/streaming.ail`** — pre-serialised `messagesJson: string` rather than typed `[Message]` list, and `parseDelta` typed extraction deferred to v1.1 (recipe page documents shape-specific manual extraction patterns for v1). Both are explicitly noted in the module docstring and the recipe page's "v1 limitations" section.

**Total shipped**: ~700 LOC (Go bridge + builtin + AILANG module + 12 tests + recipe + example) plus design-doc / CHANGELOG / master-sequence updates. Tests green across `internal/pkg/`, `internal/ai/`, `internal/ai/configdriven/`, `internal/effects/`, `internal/builtins/`, `cmd/ailang/`.

**Author**: Claude + Mark
**Created**: 2026-05-04
**Implemented**: 2026-05-04
**Author**: Claude + Mark
**Created**: 2026-05-04

---

## Executive Summary

The [arniwesth/ailang motoko fork](https://github.com/arniwesth/ailang/tree/motoko) added 6 new Go files (~1,500 LOC) implementing AI token streaming — a `StreamingProvider` interface, OpenAI SSE client, `AI.callStreamResult` effect op, dedicated builtin, and `std/ai_motoko.ail` surface. Upstream already had **all of this generically**: 5,007 LOC of `Stream` effect infrastructure ([internal/effects/stream*.go](../../../internal/effects/)), a `std/stream` module with `sseConnect`/`onEvent`/`runEventLoop`/`withSSE`, four runnable streaming examples, and an explicit docstring at [internal/effects/stream_sse.go:14-18](../../../internal/effects/stream_sse.go#L14-L18) calling out *"AI APIs (Anthropic, OpenAI, Gemini) for token-by-token response streaming."*

The fork was built because `std/stream` did not surface in agent searches for *"AILANG token streaming,"* *"OpenRouter streaming,"* or *"AI streaming response."* Three independent agent passes missed it (arni's original development, this conversation's research agent, this conversation's planner). That's an **API-shape + discovery problem**, not just a docs problem.

This milestone ships a thin `std/ai/streaming.ail` module (≤150 LOC) that composes the existing `std/stream` primitives, plus the cross-domain discovery fixes that prevent the next analogous miss.

**Scope (in priority order):**

1. **`std/ai/streaming.ail`** — thin helper exposing `openaiCompatStream`, `anthropicStream`, `parseDelta`, with a `TokenDelta` ADT covering `content`/`reasoning_content`/`thinking`.
2. **Cross-links** between `std/ai` and `std/stream` module docstrings.
3. **Recipe page** `docs/docs/recipes/ai-token-streaming.md` with provider-specific recipes.
4. **`ailang prompt` audit + fix** — verify the teaching prompt mentions `std/stream` for streaming use cases.
5. **Cross-domain discovery lint** extending [docs/scripts/check-stdlib-index.sh](../../../docs/scripts/check-stdlib-index.sh) — fail CI when capability-shaped cross-references go missing.

**Explicitly OUT OF SCOPE:**
- New AI streaming effect (rejected — reuse `Stream` effect)
- Provider-plugin / extension API for AI providers (separate v0.18+ design doc, deferred)
- Replacing `std/stream` or `std/ai` (additive only)
- Anything Motoko-fork-specific (covered separately under M-MOTOKO-FORK-INTEGRATION)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; pure composition over deterministic primitives |
| A2: Replayability | 0 | Stream traces unchanged |
| A3: Effect Legibility | +1 | Typed `TokenDelta` surfaces intent vs raw `SSEData` parsing in user code |
| A4: Explicit Authority | +1 | **Updated**: requires `! {AI, Stream, Net}` — AI cap remains the gate for LLM access (corrected from earlier `! {Stream, Net}` sketch which would have bypassed AI cap and broken authority) |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | +2 | Typed helper massively easier for agents to compose vs assembling SSE+auth+delta-parsing primitives — directly addresses the failure mode that produced the Motoko fork |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | **Updated**: AI cap requirement plumbs into per-provider cost declarations from M-AI-PROVIDER-CONFIG; streaming calls now consume AI budget like non-streaming, not just `Stream` budget |
| A10: Composability | +1 | Returns `StreamConn` — caller composes with existing `onEvent`/`runEventLoop` |
| A11: Structured Failure | +1 | `Result[StreamConn, AIError]` vs ad-hoc primitive composition |
| A12: System Boundary | +2 | Cross-domain discovery lint formalizes the `std/X ↔ std/Y` boundary; recipe page is the explicit consumer-facing boundary |

**Net Score: +9** (raised from +7 after A4/A9 updates) → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Helper is deterministic given primitive determinism
- [x] A3 (Effects): No new effect; reuses `AI` + `Stream` + `Net`
- [x] A4 (Authority): AI cap required — preserves authority across streaming surface (corrected from initial `! {Stream, Net}` sketch)
- [x] A7 (Machines First): Whole point is making an AI-shaped API available to agents

### Architectural Correction Note

An earlier sketch of this milestone proposed `! {Stream, Net}` as the effect signature, taking caller-supplied `(baseUrl, apiKey, model)` and bypassing the AI provider registry entirely. That sketch was **rejected** during review because it broke A4 (Explicit Authority — `--caps Net` would have allowed unbounded LLM spend with no AI cap gate) and A9 (Cost Visibility — no per-provider cost declaration consumed). The corrected design routes through the AI provider registry as expanded by [M-AI-PROVIDER-CONFIG](../v0_15_0/m-ai-provider-config.md), preserving all effect-system guarantees. This is the load-bearing architectural decision; do not reintroduce the Net-bypass shape.

---

## Problem Statement

**Three independent agent passes missed `std/stream` when asked about AI token streaming.** The arniwesth/ailang fork built a parallel implementation because their agents — and ours — searched for "AI streaming" and got back AI-namespace results without `std/stream` ranking.

**Current State:**
- `std/stream` (5,007 LOC Go + AILANG module) supports SSE for any HTTP source, including AI APIs (per its own docstring at [internal/effects/stream_sse.go:14-18](../../../internal/effects/stream_sse.go#L14-L18)).
- `std/ai` provides non-streaming `call`/`callJson`/`callJsonSimple` ([internal/effects/ai.go:203-207](../../../internal/effects/ai.go#L203-L207)).
- **No bridge:** nothing in `std/ai` mentions `std/stream`; nothing in `std/ai` namespace surfaces for streaming queries; the v0.16.x [OpenRouter design doc](../../implemented/v0_16_x/m-ai-openrouter-provider.md#L368) explicitly defers streaming.
- Result: external consumer (Motoko) built ~1,500 LOC of redundant Go code, and v0.16.x landed without surfacing the existing capability.

**Impact:**
- External consumers fork the binary instead of using a package.
- AI agents writing AILANG hit the same wall on every fresh attempt — the failure isn't sticky to one consumer.
- AILANG's first-principle (A7: Machines First) is failing in production for one of the most common LLM-app patterns.

---

## Goals

**Primary Goal:** Make AI token streaming a single typed function call in AILANG, and prevent the next analogous discovery failure from happening.

**Success Metrics:**
- Eval baselines: agents writing streaming code reach for `std/ai/streaming` >80% of the time on the next baseline.
- External-consumer feedback: no new fork-style streaming reimplementations after this lands.
- Discovery audit: query *"AILANG OpenAI token streaming"* returns `std/ai/streaming` in the top 3 brain hits.
- Helper module size stays ≤150 LOC.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Reuse `Stream` effect (no new `AI.callStream` op) | Reverses Motoko's architectural choice; locks in "AI streaming = Stream + auth + delta-parsing" | human | design | high |
| One OpenAI-compat function vs per-provider helpers | If we ship `callStreamOpenAI`/`callStreamOpenRouter`/`callStreamTogether`, surface area sprawls | human | design | med |
| Stdlib module vs external package | Stdlib = better discovery; external = easier iteration; conflicting goals | human | design | high |
| `TokenDelta` shape (text + reasoning + done only) | Adding tool_use/finish_reason/usage means every provider quirk leaks into core type | human | design | med |
| Cross-domain lint scope (bidirectional vs informational) | Over-firing breaks CI for legitimate one-way mentions | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Effect signature: `! {AI, Stream, Net}`** — corrected from earlier `! {Stream, Net}` sketch. AI cap gates streaming calls; Stream + Net handle SSE plumbing. The AI handler is a thin pass-through that consumes per-provider budget (declared in M-AI-PROVIDER-CONFIG `cost` block) and emits AI trace spans alongside Stream/Net spans.
- [x] **Routing through M-AI-PROVIDER-CONFIG registry** — streaming dispatches by provider name (`openaiCompatStream("vllm", "llama-3.1-70b", messages)` resolves to whichever package registered the `vllm` provider). No caller-supplied `baseUrl` / `apiKey` — all that lives in the package's `[ai_provider]` config. Caller passes provider name + model + messages only.
- [x] One `openaiCompatStream` function (works for any provider whose config declares `request_shape = "openai_chat"` and `streaming.delta_path`); separate `anthropicStream` for `anthropic_messages` shape.
- [x] **Stdlib module** (not external package) — chosen because the entire point is discovery; stdlib gets surfaced by `ailang stdlib search`, μRAG, and brain ingestion in ways external packages don't.
- [x] `TokenDelta` minimal shape: `{ text, reasoning, done }`. Tool-use / finish-reason / usage belong in caller-defined richer types.
- [ ] Cross-domain lint: confirm seeded bidirectional pairs (`std/stream ↔ std/ai`, `std/stream ↔ std/net`) before turning the gate fatal.

---

## Solution Design

### Overview

A thin AILANG helper that takes a **registered provider name** + model + messages, looks up the provider's config (URL, auth, streaming params) via the [M-AI-PROVIDER-CONFIG](../v0_15_0/m-ai-provider-config.md) registry, and constructs a configured `std/stream.sseConnect` call through the AI effect handler. The handler consumes AI budget per the provider's declared `cost`, emits AI trace spans, and gates on the `AI` capability — same machinery as non-streaming `std/ai.call`. A `parseDelta` helper extracts unified `TokenDelta` records from raw SSE events, using the `streaming.delta_path` / `streaming.reasoning_path` JSONPaths declared in the provider config.

**Net Go changes**: minimal — runtime hook in the AI handler that consumes `streaming` block from registered configs and spawns the SSE connection. No new effect, no new builtin. Most logic lives in AILANG-side `std/ai/streaming.ail`.

### Architecture

**Components:**

1. **`stdlib/std/ai/streaming.ail`** — the helper module. Wraps `std/stream.sseConnect` with provider-specific URL/auth/body construction. Exposes `openaiCompatStream`, `anthropicStream`, `parseDelta`, `TokenDelta`, `AIError`.
2. **Cross-link docstrings** — `std/stream` and `std/ai` each gain a `## See also` block pointing at the other.
3. **`docs/docs/recipes/ai-token-streaming.md`** — provider recipes (OpenAI, OpenRouter, Anthropic), reasoning-model handling, drop-down to `std/stream` primitives.
4. **`cmd/ailang/prompts/<latest>.md`** — patched to reference streaming when describing AI use cases.
5. **`docs/scripts/check-stdlib-index.sh`** extension — verifies bidirectional cross-references for seeded pairs.

### Module Sketch

```ailang
module std/ai/streaming

import std/stream (sseConnect, StreamConn, StreamEvent, SSEData, defaultConfig)
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)

-- Unified delta: covers content (chat), reasoning_content (o1, DeepSeek-R1), thinking (Anthropic)
export type TokenDelta = {
  text: string,         -- normal content delta
  reasoning: string,    -- reasoning_content / thinking delta (empty if not a reasoning model)
  done: bool            -- true on [DONE] sentinel or message_stop
}

export type AIError = {
  code: string,         -- ConnectionFailed | AuthFailed | BudgetExhausted | ...
  message: string,
  retryable: bool
}

-- OpenAI-compatible SSE: dispatches to any registered provider whose config
-- declares request_shape = "openai_chat" and streaming.enabled = true.
-- Provider must be registered via M-AI-PROVIDER-CONFIG (built-in or package-shipped).
-- Returns a StreamConn the caller drives with std/stream.onEvent + runEventLoop.
-- Caller uses parseDelta on each SSEData event to extract TokenDelta.
export func openaiCompatStream(
  provider: string,                -- registered provider name, e.g. "openai", "vllm", "openrouter"
  model: string,                   -- model identifier within that provider
  messages: [Message]
) -> Result[StreamConn, AIError] ! {AI, Stream, Net}

-- Anthropic SSE: dispatches to any registered provider whose config declares
-- request_shape = "anthropic_messages" and streaming.enabled = true.
export func anthropicStream(
  provider: string,                -- registered provider name, e.g. "anthropic"
  model: string,
  messages: [Message]
) -> Result[StreamConn, AIError] ! {AI, Stream, Net}

-- Helper: parse a raw StreamEvent into a TokenDelta using the JSONPaths
-- declared in the provider's [ai_provider.streaming] config (delta_path,
-- reasoning_path, done_sentinel). Returns None for non-data events.
export func parseDelta(provider: string, event: StreamEvent) -> Option[TokenDelta]
```

**Implementation strategy:**
- `openaiCompatStream` and `anthropicStream` look up the provider config in the AI registry (populated by [M-AI-PROVIDER-CONFIG](../v0_15_0/m-ai-provider-config.md)), construct the SSE request via the AI handler (so AI cap, budget, traces all apply), and return a `StreamConn` driven by `std/stream.onEvent` / `runEventLoop`.
- `parseDelta` consumes the `streaming.delta_path` and `streaming.reasoning_path` JSONPaths from the provider config — provider-specific delta shape lives in TOML, not in this AILANG module. ~30 LOC AILANG using `std/json`.
- Capability check at call time: if the registered provider's `capabilities.streaming = false`, return `Err(AIError{ code: "CapabilityNotSupported" })` immediately.

**Hard constraints:**
- NO new effect — reuse `AI` + `Stream` + `Net`.
- NO new builtin — pure AILANG (one runtime hook in the AI handler is the only Go-side change, and it's M2-class plumbing not a builtin).
- NO replacement of `std/stream` APIs — additive only.
- NO caller-supplied URL or API key — all provider config lives in `ailang.toml` `[[ai_provider]]` blocks (M-AI-PROVIDER-CONFIG). Caller passes provider name + model + messages.
- Module size cap: **150 LOC**. If it grows past that, split per-provider or push to external package.

### Implementation Plan

**Phase 1: AI handler hook + helper module + cross-links** (~7 hours)
- [ ] Verify [M-AI-PROVIDER-CONFIG](../v0_15_0/m-ai-provider-config.md) is merged with `[ai_provider.streaming]` block parsed and accessible via the AI handler.
- [ ] Add streaming dispatch path in the AI handler: lookup provider config, construct SSE request from `streaming.endpoint` + auth shape, hand off to `std/stream.sseConnect` machinery, emit AI trace span, charge AI budget per `cost` config.
- [ ] Snapshot test: streaming AI trace span shape == non-streaming AI trace span shape (modulo streaming-specific attributes).
- [ ] Write `stdlib/std/ai/streaming.ail` (≤150 LOC)
- [ ] Add `## See also` cross-link to `stdlib/std/stream/stream.ail`
- [ ] Add `## Token streaming` cross-link to `stdlib/std/ai/<root>.ail`
- [ ] Reproduction tests: mock SSE server (httptest.Server) for OpenAI shape, Anthropic shape, [DONE] sentinel, malformed JSON, missing fields, missing `--caps AI` rejection
- [ ] `examples/runnable/ai_stream_openai.ail` — end-to-end against echo SSE server with `--caps AI,Stream,Net,IO`

**Phase 2: Docs + prompt + lint** (~5 hours)
- [ ] Write `docs/docs/recipes/ai-token-streaming.md` (~200 lines)
- [ ] Audit `cmd/ailang/prompts/<latest>.md`; patch streaming section if absent/weak
- [ ] Extend `docs/scripts/check-stdlib-index.sh` with bidirectional cross-reference check
- [ ] Wire lint into CI; verify a deliberately-broken cross-link fails the gate
- [ ] CHANGELOG.md entry under v0.17.0
- [ ] Cross-reference from [m-external-consumer-dx.md](./m-external-consumer-dx.md) external-consumers guide

### Files to Modify/Create

**New files:**
- `stdlib/std/ai/streaming.ail` — helper module, ~150 LOC
- `docs/docs/recipes/ai-token-streaming.md` — provider recipes, ~200 lines
- `examples/runnable/ai_stream_openai.ail` — end-to-end demo, ~40 LOC
- `internal/builtins/<test>/ai_streaming_test.go` (if AILANG `parseDelta` unit testing needs Go-side fixtures)

**Modified files:**
- `stdlib/std/stream/stream.ail` — `## See also` block, ~5 lines
- `stdlib/std/ai/<root>.ail` — `## Token streaming` block, ~5 lines
- `cmd/ailang/prompts/<latest>.md` — streaming section, ~20 lines
- `docs/scripts/check-stdlib-index.sh` — cross-reference check, ~50 LOC
- `docs/docs/guides/external-consumers.md` (planned in m-external-consumer-dx.md) — reference this milestone
- `CHANGELOG.md` — v0.17.0 entry

---

## Examples

### Example 1: OpenAI streaming chat

**Before** (forced into Motoko-style binary fork or 50+ LOC of hand-rolled `std/stream` plumbing):
```ailang
-- User has to assemble: Authorization header, model+messages JSON body,
-- parse delta JSON, extract content vs reasoning_content vs thinking, handle [DONE]
import std/stream (sseConnect, ...)
-- ... 50+ lines of plumbing ...
```

**After** (URL/auth live in the built-in `openai` provider config, no caller secrets):
```ailang
import std/ai/streaming (openaiCompatStream, parseDelta, TokenDelta)
import std/stream (onEvent, runEventLoop, disconnect)
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)

export func main() -> unit ! {AI, Stream, Net, IO} {
  let result = openaiCompatStream(
    "openai",                                            -- registered provider name
    "gpt-4o",                                            -- model within that provider
    [{ role: "user", content: "Hello!" }]
  );
  match result {
    Ok(conn) => {
      onEvent(conn, \event ->
        match parseDelta("openai", event) {
          Some({ text, reasoning, done }) =>
            if done then false  -- stop
            else { _io_print(text); true },
          None => true  -- non-data event, keep going
        });
      runEventLoop(conn);
      disconnect(conn)
    },
    Err(e) => _io_println(e.message)
  }
}
```

Notice: no `baseUrl`, no `apiKey`, no `getEnv` calls. The `openai` provider's URL and `OPENAI_API_KEY` env reference live in its `[[ai_provider]]` config (built-in or package-shipped). The caller just names a registered provider. AI cap gates the call; AI budget tracks the cost.

### Example 2: Reasoning model (DeepSeek-R1 via OpenRouter)

```ailang
let result = openaiCompatStream(
  "openrouter",                          -- the openrouter provider config handles URL + auth
  "deepseek/deepseek-r1",                -- model identifier within OpenRouter
  messages
);
-- parseDelta("openrouter", event) returns reasoning text in the `reasoning`
-- field per the openrouter provider's `streaming.reasoning_path` JSONPath.
-- UI can show it in a separate "thinking" pane while `text` accumulates the answer.
```

### Example 3: Local vLLM via package-shipped provider

User installed `pkg/sunholo/ai_vllm` (per M-AI-PROVIDER-CONFIG Example 1):

```ailang
let result = openaiCompatStream(
  "vllm",                                -- registered by sunholo/ai_vllm package
  "llama-3.1-70b",
  messages
);
```

Same shape as OpenAI; only the provider name changes. No caller-side knowledge of vLLM URL, auth, or cost.

### Example 3: Drop down to primitives when needed

If a provider isn't OpenAI- or Anthropic-shaped (e.g. Cohere, local llama.cpp with custom JSON), users still use `std/stream` directly — the recipe page documents this fallback.

---

## Success Criteria

- [ ] `make ci` passes
- [ ] `stdlib/std/ai/streaming.ail` compiles, ≤150 LOC
- [ ] `parseDelta` tests cover OpenAI/OpenRouter/Anthropic shapes + edge cases ([DONE] sentinel, missing fields, malformed JSON)
- [ ] Integration tests against mock SSE servers pass
- [ ] `examples/runnable/ai_stream_openai.ail` runs end-to-end
- [ ] `docs/docs/recipes/ai-token-streaming.md` exists; linked from `std/ai`, `std/stream`, `m-external-consumer-dx.md`, CHANGELOG
- [ ] `ailang prompt` mentions `std/ai/streaming` for AI streaming use cases
- [ ] Cross-domain discovery lint runs in CI; deliberately-broken cross-reference test fails as expected
- [ ] CHANGELOG.md entry under v0.17.0 references this design doc
- [ ] Brain ingestion verified: query *"AILANG OpenAI token streaming"* returns `std/ai/streaming` in top 3 results

---

## Testing Strategy

**Unit tests** (AILANG-side):
- `parseDelta` for OpenAI shape (`choices[0].delta.content`)
- `parseDelta` for OpenAI reasoning (`choices[0].delta.reasoning_content`)
- `parseDelta` for Anthropic shape (`content_block_delta.delta.text`)
- `parseDelta` for Anthropic thinking (`content_block_delta.delta.thinking`)
- `parseDelta` for `[DONE]` sentinel (returns `Some({ done: true })`)
- `parseDelta` for malformed JSON (returns `None`)
- `parseDelta` for missing delta fields (returns `Some({ text: "", reasoning: "", done: false })`)

**Integration tests** (Go + AILANG via httptest.Server):
- `openaiCompatStream` connects to mock OpenAI SSE endpoint and accumulates a full message
- `anthropicStream` connects to mock Anthropic SSE endpoint and accumulates a full message
- Auth failure path returns `Err(AIError { code: "AuthFailed" })`
- Connection failure path returns `Err(AIError { code: "ConnectionFailed" })`
- Budget exhaustion path returns `Err(AIError { code: "BudgetExhausted" })`

**Manual testing:**
- Run example against real OpenAI API (manual, gated on env var)
- Run example against real OpenRouter API (manual, gated on env var)
- Run example against real Anthropic API (manual, gated on env var)
- Verify cross-domain lint catches a synthetic broken cross-link

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **`Message` type location** — agent may choose whether `Message` lives in `std/ai`, `std/ai/streaming`, or both. Should match the existing `std/ai` non-streaming surface.
- **`AIError.code` taxonomy** — agent decides initial code set; should align with `error_codes.json` artifact from [m-external-consumer-dx.md](./m-external-consumer-dx.md) Item 3.
- **Lint allowlist format** — agent chooses how to express one-way cross-references in the lint config (TOML / JSON / inline directive in docstrings).
- **Mock SSE server fixtures** — agent picks Go testdata format (JSON files vs inline strings).

---

## Non-Goals

- **New AI streaming effect** — explicitly rejected. The Motoko fork's `AI.callStreamResult` op is structurally redundant with `Stream.sseConnect`.
- **New AI streaming builtin** — pure AILANG only.
- **AI provider plugin/extension API** — separate v0.18+ design doc; do not block streaming on it.
- **Replacement or rewrite of `std/stream`** — additive only.
- **Motoko fork integration** — handled by separate M-MOTOKO-FORK-INTEGRATION proposal.
- **Compiler-version pinning in `ailang.toml`/`ailang.lock`** — surfaced by Motoko consumer's silent-drift problem; deserves its own milestone (proposed M-AILANG-COMPILER-PIN).

---

## Timeline

**Day 1** (~5 hours):
- Phase 1: helper module + cross-links + reproduction tests + example

**Day 2** (~5 hours):
- Phase 2: recipe page + prompt audit + cross-domain lint + CHANGELOG
- CI verification

**Total: ~10 hours across 2 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **M-AI-PROVIDER-CONFIG slips and blocks this milestone** | High | Hard prerequisite; do not start streaming work until config-driven providers are merged. If M-AI-PROVIDER-CONFIG slips out of v0.16, this slips to v0.18 |
| Helper sprawls past 150 LOC as more providers get added | Med | Hard cap in this doc; per-provider helpers move to external packages (`pkg/sunholo/ai_streaming` or `pkg/<provider>/ai_streaming`) |
| `TokenDelta` shape wrong for some provider (e.g. Mistral, future models) | Low | Intentionally minimal (text/reasoning/done); per-provider delta paths live in `streaming.delta_path` / `streaming.reasoning_path` JSONPaths in TOML, not in this AILANG module — adding a provider with a weird delta shape just declares a different JSONPath |
| Cross-domain lint over-fires on legitimate one-way mentions | Med | Allowlist for one-way references; v1 gates only seeded bidirectional pairs; informational warnings for new candidates |
| Mock SSE testing flakes | Low | Use `httptest.Server`, no real network in CI; manual real-API tests gated on env var |
| Reasoning-field naming changes upstream (e.g. OpenAI renames `reasoning_content`) | Low | Per-provider `streaming.reasoning_path` JSONPath in TOML — update the provider config, not AILANG code |
| Streaming trace span structure diverges from non-streaming AI span | Med | Snapshot test pins shape equality. Single AI handler emits both span types — divergence would require deliberate code change |

---

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — Streaming explicitly deferred at lines 368, 378, 440, 496; this milestone closes that gap.
- [design_docs/implemented/v0_10_0/m-ai-image-generation.md](../../implemented/v0_10_0/m-ai-image-generation.md) (0.42 neural match) — Pattern for adding new AI capabilities atop `std/ai`.
- [design_docs/implemented/v0_5_0/M-GAME-E2-ai-effect.md](../../implemented/v0_5_0/M-GAME-E2-ai-effect.md) (0.39 neural match) — Original `AI` effect design.

**Planned (companions / follow-ups):**
- [design_docs/planned/v0_17_0/m-external-consumer-dx.md](./m-external-consumer-dx.md) — Companion milestone; external-consumers guide must cross-reference `std/ai/streaming`.
- M-MOTOKO-FORK-INTEGRATION (to be drafted) — Strategy A: hybrid extraction. Once this milestone lands, arni can drop all six streaming fences from his fork.
- M-AILANG-COMPILER-PIN (to be drafted) — Add `ailang_version` constraint to `ailang.toml` + lock; surfaced by Motoko consumer's silent-drift problem.

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [arniwesth/ailang FORK.md](https://github.com/arniwesth/ailang/blob/motoko/FORK.md) — Motoko fork's documented divergence
- [arniwesth/motoko_agent](https://github.com/arniwesth/motoko_agent) — External consumer driving this milestone
- [internal/effects/stream_sse.go:14-18](../../../internal/effects/stream_sse.go#L14-L18) — Existing SSE infrastructure docstring (already calls out AI use case)
- [internal/effects/stream.go](../../../internal/effects/stream.go), [internal/builtins/stream.go](../../../internal/builtins/stream.go) — Existing `Stream` effect surface
- [examples/runnable/stream_sse.ail](../../../examples/runnable/stream_sse.ail) — Existing SSE example (predates this milestone; recipe page should reference)

---

## Future Work

- **M-AILANG-COMPILER-PIN** (proposed v0.17/v0.18) — `ailang.toml` + `ailang.lock` compiler version constraint; prevents the silent-drift bug Motoko consumer hit.
- **M-AI-PROVIDER-PLUGIN-API** (proposed v0.18+) — Real plugin extension point for AI providers and effect builtins. The genuinely-new pieces of arni's fork (custom OpenAI base-URL routing, OpenRouter prefix routing) belong upstream as flags or as a plugin API; the streaming work should not block this larger design.
- **Streaming for `callJson` / `callJsonSimple`** — Once base streaming lands, evaluate whether structured-output streaming (incremental JSON parsing) is worth its own helper.
- **Tool-use / function-call streaming** — Out of scope here; will need its own design once OpenAI/Anthropic tool-use SSE shapes settle.

---

## Notes for the AI Implementer

Key constraints to keep in mind during implementation:

1. **Do NOT add a new effect.** Reuse `AI` + `Stream` + `Net`. AI cap is required — no Net-bypass shortcut.
2. **Do NOT add a new builtin.** Pure AILANG plus one runtime hook in the existing AI handler.
3. **Do NOT replicate the SSE parser.** Delegate to `std/stream.sseConnect`.
4. **Do NOT take caller-supplied URLs or API keys.** All provider config lives in `[[ai_provider]]` blocks. Caller passes provider name + model + messages.
5. **Do NOT proceed without M-AI-PROVIDER-CONFIG merged.** Strict prerequisite. If you find yourself reaching for `getEnv("OPENAI_API_KEY")` in `std/ai/streaming.ail`, stop — that's the rejected design.
6. **`parseDelta` reads JSONPaths from the provider config, not hardcoded field names.** OpenAI's `reasoning_content` and Anthropic's `thinking` are configured per-provider in `streaming.reasoning_path` — your code applies the JSONPath, the provider config supplies the path. Test against the three shipped provider shapes and edge cases (`[DONE]` sentinel, missing fields, malformed JSON, capability=streaming-disabled).
7. **Keep `TokenDelta` minimal.** Resist the urge to add fields for `tool_use`, `finish_reason`, `usage`, etc — those belong in caller-defined richer delta types.
8. **Snapshot-test trace span equality.** Streaming AI span structure must equal non-streaming AI span structure (same attribute keys, same provider/model/cost recording). This is the load-bearing test for A4/A9 compliance.

If implementation deviates from this doc, **update this doc first.** The whole point of writing it is to prevent the next round of implicit-decision drift.

---

**Document created**: 2026-05-04
**Last updated**: 2026-05-04

DESIGN_DOC_PATH: design_docs/planned/v0_17_0/m-ai-streaming-helper.md
