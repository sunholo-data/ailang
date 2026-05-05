# M-AI-CALL-STREAM-HELPER: Synchronous accumulator wrapper for AI token streaming

**Status**: Implemented (v0.15.1)
**Target**: v0.15.1 (patch release)
**Implemented**: 2026-05-05 (commit 7c1bae03)
**Evaluation**: passed round 1, score 97/100 — see `.ailang/state/evaluations/eval_M-AI-CALL-STREAM-HELPER_round_1.json`
**Priority**: P2 (low — `openaiCompatStream` + manual event loop already works; this is ergonomic sugar with measurable downstream impact)
**Estimated**: 4-6 hours
**Dependencies**:
- [M-AI-STREAMING-HELPER](../v0_17_0/m-ai-streaming-helper.md) ✅ shipped in v0.15.0 — provides `_ai_stream_call`, `openaiCompatStream`, `anthropicStream`, the streaming dispatch chain, and the `[ai_provider.streaming]` schema with `delta_path`/`reasoning_path`/`done_sentinel`
- [M-AI-PROVIDER-CONFIG](../v0_15_0/m-ai-provider-config.md) ✅ shipped in v0.15.0
**Author**: Claude + Mark
**Created**: 2026-05-05

---

## Executive Summary

v0.15.0 shipped `std/ai/streaming.openaiCompatStream` which returns `Result[StreamConn, StreamErrorKind]` — callers drive the SSE event loop manually with `onEvent` + `runEventLoop` + `disconnect` and accumulate token deltas themselves. That's correct architecturally (the event-loop pattern is the right primitive) but it's **~20 LOC of boilerplate per call site** for the most common shape: *"give me the final string from a streamed AI response."*

The arniwesth/ailang motoko fork's `callStreamResult(input, step, stream_id, model)` provided exactly that shape: synchronous accumulator returning a single string. The [motoko_agent v0.15.0 migration plan](../motoko-agent-v0.15.0-migration.md) requires rewriting 4 `.ail` call sites (~80-120 LOC delta) to lose this convenience — and every future motoko-style consumer would face the same boilerplate tax.

This milestone ships a thin Go-side builtin `_ai_call_stream` and an AILANG wrapper `std/ai/streaming.callStream(provider, model, messagesJson) -> Result[string, AIError]` that drives the event loop and accumulator inside the Go runtime. Underlying primitives unchanged — this is purely a convenience layer on the existing `_ai_stream_call` infrastructure.

**Headline outcome**: motoko_agent's call-site migration drops from ~80-120 LOC delta to ~10 LOC delta — a 1-line API swap (`callStreamResult(...)` → `callStream(...)`) instead of a structural rewrite.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new nondeterminism; same underlying transport as `openaiCompatStream` |
| A2: Replayability | 0 | Trace span shape unchanged — emits the same `AI/streamCall` event from `_ai_stream_call`; new builtin is a thin wrapper |
| A3: Effect Legibility | +1 | Caller's effect row stays explicit `! {AI, Stream, Net}`; the accumulator pattern is a *named* operation rather than a hand-rolled loop, which improves readability when grepping for "where do we stream?" |
| A4: Explicit Authority | +1 | Reuses AI cap gating from the underlying `_ai_stream_call`; no new authority, no bypass surface |
| A5: Bounded Verification | 0 | No new type-system surface |
| A6: Safe Concurrency | 0 | Single-threaded accumulation; same as direct event loop |
| A7: Machines First | +2 | **Headline axiom**: the v0.15.0 boilerplate-per-call-site (~20 LOC) is a measurable AI-agent friction point. Eval baselines for AILANG-as-target-language will show the accumulator helper as a 1-line drop-in vs a structural pattern that agents must compose correctly. Confirmed by motoko fork's existence — three independent agent passes (arni's, this conversation's research, this conversation's planner) wrote new Go to avoid this exact composition |
| A8: Minimal Syntax | 0 | No new syntax — just a new exported function |
| A9: Cost Visibility | 0 | Cost tracking flows through the underlying `_ai_stream_call` (which routes through the AI handler with full budget/cap accounting) |
| A10: Composability | +1 | Builds on existing primitives without replacing them; `openaiCompatStream` + `onEvent` + `runEventLoop` remain available for callers needing fine-grained control |
| A11: Structured Failure | +1 | Returns `Result[string, AIError]` with structured fields (`code`, `message`, `retryable`) — a richer error than `Result[string, StreamErrorKind]` because we can map streaming-specific errors (auth, capability, parse) into AI-specific categories |
| A12: System Boundary | 0 | No new boundary; same SSE transport, same provider config |

**Net Score: +6** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Wraps deterministic primitives; no new nondeterminism
- [x] A3 (Effects): Effect row preserved — `! {AI, Stream, Net}` from caller's perspective
- [x] A4 (Authority): AI cap still required; cannot be invoked without `--caps AI`
- [x] A7 (Machines First): The whole point — make a high-frequency pattern a 1-line call

---

## Motivating Evidence

### The motoko fork's `callStreamResult` was popular for a reason

arniwesth/ailang@motoko's `std/ai_motoko.callStreamResult(input, step, stream_id, model) -> AIStreamResult` has 4 call sites in motoko_agent (`src/core/rpc.ail`, `src/core/ext/compose/{compose,claimcheck,author_loop}.ail`). Each call is ~3 lines. The migration to v0.15.0's `openaiCompatStream` + manual event loop turns each call into ~20 lines. **Net delta of ~70-80 LOC** of mechanical boilerplate that adds zero functionality — just bookkeeping.

### Three agent passes wrote duplicate streaming infrastructure

The retrospective analysis in [motoko-integration-sequence.md](../motoko-integration-sequence.md) Phase 0 captured that arni's agent, this conversation's research agent, and this conversation's first sprint planner *all independently* missed `std/stream.sseConnect` when asked about AI streaming. The accumulator-style API in the fork's `callStreamResult` was the **discoverable shape** they each landed on. v0.15.0 ships the right primitive (`openaiCompatStream`) but not the discoverable shape — this milestone closes that gap.

### Migration math

motoko_agent v0.15.0 migration estimated effort, with vs without `callStream`:

| Effort | Without `callStream` (v0.15.0 only) | With `callStream` (v0.15.1) |
|--------|---------------------------------------|------------------------------|
| Install script | 30 min | 30 min |
| Provider config (`[[ai_provider]]` blocks) | 30 min | 30 min |
| 4 `.ail` call-site rewrites | **2-3 hours** (event-loop restructure each) | **20 min** (1-line swap each) |
| TS codegen update | 30 min | 10 min |
| Smoke test + iteration | 1 hour | 30 min |
| **Total** | **4-5 hours** | **2 hours** |

So `callStream` cuts the motoko migration from one full afternoon to one short morning. Generalises to any future motoko-style consumer.

---

## Design

### AILANG-side surface (added to `std/ai/streaming`)

```ailang
-- callStream: synchronous accumulator wrapper around openaiCompatStream.
--
-- Runs the SSE event loop internally and returns the accumulated content
-- as a single string, OR a structured AIError if the stream failed.
-- Equivalent in shape to std/ai.call but for the streaming path:
--
--   let r = call("Hello!")                    -- ! {AI}
--   match callStream("openai", "gpt-4o", body) -- ! {AI, Stream, Net}
--
-- Uses the registered provider's [ai_provider.streaming] config (delta_path,
-- reasoning_path, done_sentinel) to drive content extraction. Caller does
-- NOT manage the event loop — for fine-grained control (e.g. cancellation
-- mid-stream, render-as-you-go), use openaiCompatStream + onEvent directly.
--
-- Args:
--   provider     - registered provider name (e.g. "openai", "vllm")
--   model        - model identifier (e.g. "gpt-4o-mini", "llama-3.1-70b")
--   messagesJson - pre-serialised JSON array of {role, content} records
--
-- Returns:
--   Ok(text)  - accumulated content stream concatenated
--   Err(e)    - structured AIError with code/message/retryable
export func callStream(
  provider: string,
  model: string,
  messagesJson: string
) -> Result[string, AIError] ! {AI, Stream, Net} {
  _ai_call_stream(provider, model, messagesJson)
}
```

**No `Anthropic` variant** — the v0.15.0 `anthropicStream` and `openaiCompatStream` already share the same dispatch path; the *provider config* picks the SSE wire shape, not the function name. `callStream` is therefore the universal entry point: it works against any registered provider whose `[ai_provider.streaming]` block declares `enabled = true` and a `delta_path`.

**Reasoning is captured but discarded by default in v0.15.1.** A separate `callStreamWithReasoning(...) -> Result[{text: string, reasoning: string}, AIError]` is reserved for v0.15.2 if demand emerges (motoko_agent's TUI doesn't render reasoning separately today, so v0.15.1's lossy default is fine).

### Go-side implementation: `_ai_call_stream` builtin

Lives in `cmd/ailang/configdriven_streaming.go` (same file as `aiStreamCall`, sharing the cycle-resolution placement). Reuses the existing `_ai_stream_call` op via `effects.Call(ctx, "AI", "streamCall", ...)` to open the connection — the new builtin only adds the loop/accumulator/extraction.

```go
// aiCallStream implements AI.callStream(provider, model, messages_json) ->
// Result[string, AIError]. Synchronous accumulator wrapper around the
// streamCall op. Drives the event loop in Go so AILANG-side callers get
// a single-string result.
func aiCallStream(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Validate args, look up provider in registry (mirrors aiStreamCall preflight)
    // 2. Call effects.Call(ctx, "AI", "streamCall", args) to open StreamConn
    // 3. On Ok(conn):
    //    a. Pull events off the connection's event buffer (existing infra)
    //    b. For each SSEData event, parse JSON and extract content via the
    //       provider's streaming.delta_path (using existing JSONPath helper
    //       in internal/ai/configdriven/jsonpath.go)
    //    c. Stop on streaming.done_sentinel (default "[DONE]") or Closed/StreamError
    //    d. Disconnect cleanly
    //    e. Return Ok(accumulated_string) wrapped as AILANG Ok value
    // 4. On Err(streamErr): map StreamErrorKind to AIError record:
    //    - ConnectionFailed   -> AIError{code: "ConnectionFailed", retryable: false, ...}
    //    - Timeout            -> AIError{code: "Timeout", retryable: true, ...}
    //    - BudgetExhausted    -> AIError{code: "BudgetExhausted", retryable: false, ...}
    //    - ProtocolError      -> AIError{code: parseProtocolErrorTag(msg), ...}
    //                            (extracts the "[ProviderNotFound]" / "[CapabilityNotSupported]"
    //                            prefix from the message — see commit 493f4f40)
}
```

### Where the accumulator state lives

**This is the load-bearing decision** vs the AILANG-side alternatives I considered:

| Option | Mutability mechanism | Pros | Cons | Decision |
|--------|----------------------|------|------|----------|
| **(a) AILANG `ref`/`:=` syntax** | User-level mutable refs | Pure AILANG, no new builtin | AILANG doesn't expose `ref` syntax in stdlib today (Go runtime has `RefCell` but no surface syntax); adding it just for this is overkill | ❌ Rejected |
| **(b) `runEventLoopFold(conn, init, foldFn)` runtime primitive** | Threaded through fold | Functional, composable | Larger surface change to `internal/effects/stream.go`; `onEvent` already exists with mutating handler-internal state via Go closures, the AILANG side would need new wiring to expose fold | ❌ Rejected for v0.15.1; reconsider v0.16+ |
| **(c) Go-side accumulator builtin** | Go function-local `strings.Builder` | Smallest surface change, matches existing `_ai_call_*` family | Adds one more builtin to the AI catalog | ✅ **Chosen** |

**Why (c) wins**: AILANG already has builtins like `_ai_call`, `_ai_call_json`, `_ai_call_image` that hide bookkeeping inside Go. `_ai_call_stream` is the natural fifth entry in that catalog. Mutability stays inside Go where it's safe; AILANG's surface stays purely-functional. Same pattern that lets `_ai_call` work without exposing HTTP retry loops to AILANG callers.

### Trace span behaviour

`_ai_call_stream` does NOT emit its own trace span. The underlying `_ai_stream_call` already records `AI/streamCall` via `RecordAIEffect`; wrapping it in another span would double-count. The accumulator is a transformation on the same logical AI call.

If granular per-delta tracing is needed in the future (e.g. for replay debugging), it should land as a separate `AILANG_TRACE=deep` mode rather than in this milestone.

### Files to modify

**New code** (~120 LOC + ~150 LOC tests):

- `internal/builtins/ai.go` — register `_ai_call_stream` builtin (~30 LOC, mirrors existing `registerAICall`)
- `cmd/ailang/configdriven_streaming.go` — `aiCallStream` op + register via `RegisterOp("AI", "callStream", aiCallStream)` (~80 LOC)
- `cmd/ailang/configdriven_streaming_test.go` — extend with happy-path + error-mapping + reasoning-discard tests (~150 LOC)
- `internal/pipeline/testdata/builtin_types.golden` — regenerate (1 new line for `_ai_call_stream`)

**AILANG-side** (~25 LOC):

- `std/ai/streaming.ail` — add `callStream(provider, model, messagesJson) -> Result[string, AIError]` export at the bottom; cross-reference in module docstring's "Quick start" section
- File grows from 149 LOC to ~175 LOC. Soft cap of 150 was an explicit design constraint in the v0.15.0 milestone — this raises it to 200 LOC for v0.15.1. Justified because the helper is a load-bearing v1.1 promotion, not new surface area.

**Documentation**:

- `docs/docs/recipes/ai-token-streaming.md` — replace the inline `runStreamCall` AILANG snippet (which the recipe page currently shows callers writing for themselves) with a note pointing at `callStream`. Demote inline event-loop pattern to "advanced control flow" subsection
- `docs/docs/guides/custom-ai-providers.md` — add a one-line cross-reference in the Streaming sub-section
- `changelogs/v0.10-current.md` — new "[Unreleased] - targeting v0.15.1" entry; the migration math from this doc's Motivating Evidence section is the headline justification
- `design_docs/planned/motoko-agent-v0.15.0-migration.md` — update the API-shape adaptation section to use `callStream` directly; revise the migration estimate downward (from 4-5 hours to ~2 hours)

---

## Implementation Plan

Single-session sprint, ~4-6 hours total. No staged milestones — each component is independently testable but small enough to ship together.

| Hour | Task |
|------|------|
| 0:00-0:30 | Read existing infrastructure: `_ai_stream_call` + `BuildStreamRequest` + JSONPath helper. Confirm `extractPath` works for the delta-path use case |
| 0:30-1:30 | Write `aiCallStream` op in `cmd/ailang/configdriven_streaming.go`. Reuse the `streamCall` op for connection-opening; add the loop + accumulator + extraction logic |
| 1:30-2:00 | Register `_ai_call_stream` builtin in `internal/builtins/ai.go` |
| 2:00-3:30 | Write integration tests: happy path (OpenAI shape, Anthropic shape), error mapping (ProtocolError → AIError code prefix), reasoning discard, malformed delta JSON |
| 3:30-4:00 | Add `callStream` AILANG wrapper to `std/ai/streaming.ail`; raise file's LOC cap; update cross-link docs |
| 4:00-4:30 | Regenerate golden snapshot; type-check the AILANG-side wrapper against the synthetic-import test |
| 4:30-5:00 | Update recipe page + custom-ai-providers guide + CHANGELOG entry |
| 5:00-6:00 | Update motoko-agent-v0.15.0-migration.md to use `callStream`; revise effort estimate downward |

---

## Acceptance Criteria

- [ ] `make ci` passes
- [ ] `_ai_call_stream(provider, model, messagesJson)` returns `Result[string, AIError]` against a mock OpenAI SSE server (happy path)
- [ ] Same against a mock Anthropic-shape server (`anthropic_messages` request_shape)
- [ ] `streaming.enabled = false` provider returns `Err(AIError{ code: "CapabilityNotSupported" })`
- [ ] Provider not registered returns `Err(AIError{ code: "ProviderNotFound" })`
- [ ] AI cap missing fails with same error semantics as `_ai_call` (no special-case streaming behaviour)
- [ ] Trace inspection: exactly one `AI/streamCall` span emitted per `callStream` invocation (the accumulator does NOT emit its own span)
- [ ] Reasoning fields (`reasoning_content`, `thinking`) are correctly discarded — only the visible `content` accumulates
- [ ] AILANG-side `callStream` wrapper compiles cleanly via synthetic-import test
- [ ] motoko_agent migration plan updated to reference `callStream`; estimate revised downward
- [ ] CHANGELOG.md v0.15.1 entry exists with migration-math motivation

---

## Deferred Decisions

- **`callStreamWithReasoning`** — emit a `{text, reasoning}` record so reasoning models surface their thinking. Reserved for v0.15.2 if motoko_agent or another consumer needs it. v0.15.1 ships the lossy default
- **Tool-call deltas** — OpenAI streams function-call invocations as a separate delta type. The v0.15.0 `[ai_provider.streaming]` schema doesn't have a `tool_call_path` yet. Tool-call streaming is part of the schema-v2 follow-up (M-AI-PROVIDER-CONFIG Future Work)
- **Configurable timeout** — `callStream` inherits the underlying provider's timeout; if callers need stream-specific timeouts, that's an extension to the schema or the helper signature
- **Per-delta callback for live rendering** — for "render as you go" UX (TUI streaming output), callers should drop down to `openaiCompatStream` + `onEvent`. `callStream` is intentionally for "give me the final string"
- **AILANG-side `ref`/`runEventLoopFold` exposure** — option (b) above is appealing for general-purpose AILANG functional programming but blocked on parser/type-system work. Tracked as a future ergonomics improvement, not in scope here

---

## Non-Goals

- **No streaming-only effect** — reuses `! {AI, Stream, Net}`, same as `openaiCompatStream`
- **No replacement of `openaiCompatStream`** — the event-loop API stays available; `callStream` is layered on top
- **No new schema** — uses existing `[ai_provider.streaming]` config from M-AI-PROVIDER-CONFIG
- **No new error variants** — `AIError` record shape from v0.15.0 is sufficient
- **No multi-provider fanout** — caller passes one provider name; `callStream` doesn't try multiple providers (that's OpenRouter's job)

---

## Risks and Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|-----------|
| Accumulator silently drops events that aren't `SSEData` (e.g. `Closed`) | Med | Test asserts loop terminates correctly on `Closed` and `StreamError` events; explicit fall-through cases |
| `delta_path` JSONPath miss returns empty string instead of clear error | Low | When 0 chunks accumulate but the loop terminated normally, return `Ok("")` (not Err) — matches OpenAI's behaviour for empty completions |
| Schema-version drift if v2 schema changes streaming sub-block shape | Low | `callStream` consumes the existing v1 fields only (`delta_path`, `done_sentinel`); v2 additions stay opt-in |
| Performance regression on long streams (string concatenation in loop) | Low | Use `strings.Builder` in Go; AILANG-side never sees the intermediate state |
| Confusion between `callStream`, `openaiCompatStream`, and `call` | Med | Crystal-clear docstring + a "When to use which" decision matrix in the recipe page |

---

## Related Documents

- [m-ai-streaming-helper.md](../v0_17_0/m-ai-streaming-helper.md) — the v0.15.0 design doc; the `openaiCompatStream` API this milestone wraps
- [m-ai-provider-config.md](../v0_15_0/m-ai-provider-config.md) — the registry + `[ai_provider.streaming]` schema this consumes
- [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md) — the downstream consumer whose migration math motivates this; will be updated post-implementation to use `callStream` directly
- [motoko-integration-sequence.md](../motoko-integration-sequence.md) — Phase 3 of the master plan; this milestone makes Phase 3's Phase-3-PR-B half cheaper to land
- [docs/docs/recipes/ai-token-streaming.md](../../../docs/docs/recipes/ai-token-streaming.md) — recipe page that gets simplified post-implementation

---

## Notes for the AI Implementer

1. **Reuse, don't reimplement** — `aiCallStream` should call `effects.Call(ctx, "AI", "streamCall", args)` to get the StreamConn; the loop logic builds on top. Don't duplicate provider lookup, body construction, or auth.
2. **JSONPath extraction is the existing helper** — `internal/ai/configdriven/jsonpath.go::extractPath` already handles the path syntax; just call it once per `SSEData` event with the provider's `streaming.delta_path`.
3. **`done_sentinel` is OpenAI-specific** — Anthropic uses event-type-based termination (`message_stop`). The accumulator should terminate on EITHER condition. Read both from the spec.
4. **Reasoning fields exist in the schema but stay unused in v0.15.1** — `streaming.reasoning_path` is read but the extracted reasoning is silently discarded. The v1.1 helper sketch in m-ai-streaming-helper.md was always lossy on reasoning; this milestone preserves that.
5. **Don't add a new effect** — the constraint from M-AI-STREAMING-HELPER carries forward.
6. **Don't bypass the AI handler** — `_ai_call_stream` MUST go through `effects.Call("AI", "streamCall", ...)`, not directly to `effects.StreamSSEPost`. This preserves trace, budget, and cap integration.

If implementation deviates from this doc, update this doc first.

---

**Document created**: 2026-05-05
**Last updated**: 2026-05-05
