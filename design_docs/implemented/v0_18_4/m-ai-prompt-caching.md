# M-AI-PROMPT-CACHING: Model-independent prompt-cache hints across all AI providers

**Status**: IMPLEMENTED
**Target**: v0.18.4 (or v0.19.0 if the OpenRouter-routing detection forces wider scope)
**Priority**: P1 (Medium — closes a real ~10x cost-multiplier gap on long-context AILANG agent loops, but is NOT the proximate cause of today's 1000x motoko bloat — that one is a motoko-internal config bug, see "Honest framing" below)
**Estimated**: ~6 hours (~210 LOC across `std/ai.ail`, `internal/ai/`, motoko opt-in)
**Dependencies**: None blocking. Composes with M-AI-PROVIDER-CONFIG (v0.15.0, registry already in place) and v0.18.3 hybrid-tool fix (must not break tool_use/tool_result correlation).
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-08

---

## Honest framing — what this fixes vs what it doesn't

Today (2026-05-08) the v0.18.3 final 3-harness comparison showed:

| Harness | Pass rate | Total input tokens | Total cost |
|---------|-----------|--------------------|------------|
| claude-code (haiku-4-5) | 15/15 (100%) | 5,538 | $0.0030 |
| opencode (haiku-4-5) | 15/15 (100%) | 4,213 | $0.0021 |
| motoko (haiku-4-5) | 12/15 (80%) | **5,153,068 (~1000x)** | **$0.7466** |

Two distinct things contribute to that 1000x:

**(A) The dominant cause — a motoko-internal config bug**, found independently by the motoko-explorer agent and filed as `msg_20260508_200802_7a95e4e8` + GitHub issue [arniwesth/motoko_agent#225](https://github.com/arniwesth/motoko_agent/issues/225):
- Shipping `dogfood/config.json` enables `tools.ohmy_pi: true + tools.hybrid: true`
- With `MOTOKO_AGENT_V2=1` (default), `split_by_backend` in `agent_loop_v2` routes `BashExec` to the **Delegated** backend, which is explicitly not-wired
- Every `BashExec` returns `delegated_backend_not_wired:true`
- Hybrid-mode bash extraction takes the same broken path
- Model retries ~3×, then babbles 10-13 more steps with no formal `done`
- Each wasted step costs ~32K input tokens (the eval-suite teaching prompt is ~21K) instead of ~2K
- A/B verified: flipping `ohmy_pi: false` → 15× faster, 6× fewer output tokens, 3 steps to completion

**That bug is NOT in scope for this design doc.** It's fixed in the motoko_agent repo (P0 one-liner: change dogfood default). What this design doc fixes is:

**(B) The structural caching gap — every motoko step pays full input cost.** Even on **healthy** long loops without the BashExec storm, motoko's session JSONLs show `cache_read: null, cache_create: null` for every step. claude-code & opencode go through Anthropic's SDK directly, which auto-applies `cache_control: {type:"ephemeral"}` on system prompts ≥1024 tokens — 90% off cached reads after the first turn. AILANG's `internal/ai/anthropic/step.go` never sets cache_control. So after issue #225 lands, motoko will still pay ~10× more than claude-code on the same workload (not 1000× — but ~10× is a real multiplier on long teaching-prompt-heavy loops).

This design doc closes (B) in a model-independent way — Anthropic, OpenAI, Gemini, OpenRouter all get a uniform AILANG-side knob. (A) is referenced for context but its fix lives in motoko_agent.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Cache hints are advisory; behavior with vs without hints differs only in cost/latency, not in produced bytes (modulo the model-determinism caveat that already applies) |
| A2: Replayability | 0 | Cache state isn't part of the trace; replay is unaffected |
| A3: Effect Legibility | +1 | Cache hints sit on the existing `! {AI}` effect — no new effect, no new authority |
| A4: Explicit Authority | 0 | No new capabilities; AI cap remains the gate |
| A5: Bounded Verification | 0 | Type-checking unaffected (additive optional field with default `[]`) |
| A6: Safe Concurrency | 0 | No concurrency interaction |
| A7: Machines First | +2 | Reduces input-token cost ~10× on long-context loops, which is exactly the cost agents pay; the existing `StepResult.cache_read_input_tokens` field already lets agents observe cache hit/miss per step |
| A8: Minimal Syntax | 0 | One additive field on `Request`; no new syntax |
| A9: Cost Visibility | +2 | Telemetry side already exists (`cache_read_input_tokens` / `cache_creation_input_tokens` on `StepResult`); this milestone makes the request-side knob visible too. Net: agents can see AND control cache behavior |
| A10: Composability | +1 | Same field works across all providers; no per-provider AILANG-side branching |
| A11: Structured Failure | +1 | When a provider can't honor a hint (OpenAI auto-cache, Gemini no-explicit-API), AILANG emits a structured one-shot warning rather than failing or silently dropping |
| A12: System Boundary | +1 | Formalizes the "AILANG-side cache hint → provider-specific wire shape" boundary as a per-provider mapping in Go |

**Net Score: +8** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism — cache hints don't affect produced outputs, only cost
- [x] A3 (Effects): Cache hints flow through the existing `! {AI}` effect; no hidden side effects
- [x] A4 (Authority): No new capabilities granted; AI cap still gates everything
- [x] A7 (Machines First): The whole point — close a ~10× cost gap that human-aimed CLIs (claude-code) don't have

---

## Problem Statement

AILANG agents that loop for many turns over a stable system prompt pay full input-token cost on every step, because `internal/ai/anthropic/step.go` never sets `cache_control` markers on outgoing requests. Confirmed by inspecting motoko session JSONLs (one entry per step):

```json
{"step":0,"in":30707,"out":162,"cache_read":null,"cache_create":null}
{"step":1,"in":30991,"out":85, "cache_read":null,"cache_create":null}
{"step":2,"in":31203,"out":204,"cache_read":null,"cache_create":null}
...
```

Per-turn `~30k` tokens with tiny growth. Total session: 11 steps × 30k = 330k input tokens at full price.

**Current State:**
- Anthropic provider ([internal/ai/anthropic/step.go:41-153](../../../internal/ai/anthropic/step.go#L41)) reads `cache_read_input_tokens` / `cache_creation_input_tokens` from the **response** (telemetry works) but never sets `cache_control` on **request** content blocks
- `internal/ai/provider.go` `Request` struct has no cache-hint field
- `std/ai.ail` `step(model, messages, tools)` signature has no way for callers to opt in to caching
- Other AILANG-driven CLIs that talk to the same Anthropic backend (anything going through `_ai_step`) all pay the same penalty
- claude-code & opencode use the Anthropic SDK directly and get auto-caching for free

**Impact:**
- Real motoko sessions: 330k input tokens × $0.5/1M = $0.16 per session at full price; with caching = ~$0.02 (8× drop)
- Multiplied across the eval suite: a 15-benchmark smoke run is ~$2.40 instead of ~$0.30
- Affects **any** AILANG program using `std/ai.step` in a loop — not just motoko. As more AILANG-native agent loops ship, the gap compounds.
- Telemetry contract is already in place (`StepResult` has the two fields per [std/ai.ail:115-127](../../../std/ai.ail#L115)), so agents can already **observe** cache hit/miss. They just can't **opt in**.

---

## Goals

**Primary Goal:** Add a single model-independent `cache_breakpoints` field to `ai.Request` that, when populated, causes each provider to apply its own caching contract correctly — and when empty, preserves the current zero-cache behavior bit-for-bit.

**Success Metrics:**
- After enabling `cache_breakpoints` in motoko's `agent_loop_v2.ail`, post-release 3-harness smoke comparison shows motoko `input_tokens` within **5×** of claude-code (currently 1000× — but most of the 1000× is issue #225, not this. The realistic goal here is **8-10×** reduction on the cache-related slice)
- All 3 Anthropic backends (Anthropic-direct, AWS Bedrock, GCP Vertex) honor `cache_control` markers without 400 errors. Bedrock is the strict one — see v0.18.1 dotted-tool-name and v0.18.3 hybrid-tool fixes for prior asymmetry pain
- OpenAI and Gemini providers don't break when `cache_breakpoints` is set (NO-OP path correctly chosen)
- `StepResult.cache_read_input_tokens > 0` for at least step 1+ on Anthropic when a cache breakpoint is set on the system prompt
- New AILANG fixture test exercises a 3-turn loop with a stable system prompt and asserts cache_read tokens grow turn-on-turn

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `cache_breakpoints` lives on `Request` (additive optional field) vs new typed `RequestV2` | Backwards-compat for every existing caller of `ai.step`; field default `[]` preserves current behavior. RequestV2 would force a fork | human | design | high |
| Position vocabulary: `"system" \| "last_user" \| "tool_result"` (string-typed) vs typed enum | Strings let stdlib add positions without lockstep Go bumps; typed enum gives compile-time safety. Strings chosen for stdlib evolution latitude — Go side validates and emits warning on unknown | human | design | med |
| OpenRouter routing detection: model-prefix sniffing vs response-field parsing | Determines whether `cache_control` gets through to OpenRouter→Anthropic. Prefix-sniffing is local & cheap but trusts user input; response-field parsing requires a "known route" table after first call. Prefix-sniffing chosen — OpenRouter model strings are documented (`anthropic/claude-3-5-haiku`) | agent | design | med |
| Anthropic placement: only system prompt, or system + last_user + tool_result | Anthropic supports up to 4 cache breakpoints per request. Placing on system gets the 90% win for stable-prompt loops; placing on tool_result helps multi-turn tool dialogues. Phase 1 = system only; Phase 2 = wider placement after telemetry confirms ROI | agent | implementation | low |
| Gemini behavior in Phase 1: NO-OP + warning vs sketch CachedContent | Gemini's Context Caching API is async (separate cache create + reference by ID), which is a different shape than `cache_breakpoints`. Phase 1 ships NO-OP + structured warning; full implementation deferred to Phase 2 (separate design doc) | human | design | med |
| OpenAI behavior: NO-OP silent vs NO-OP + once-per-session info log | Auto-caching is transparent for OpenAI — no AILANG action needed. But silently ignoring user-supplied hints feels wrong (user might think hints are working). Decision: emit one structured info log per session: `cache_hint_ignored_openai_auto_cache` | agent | implementation | low |

### Design Freeze

Before implementation begins:

- [ ] `cache_breakpoints` is added to `Request` (additive, optional, default `[]`). NOT a new RequestV2 type
- [ ] Position string vocabulary frozen at `"system" | "last_user" | "tool_result"` for v0.18.4. Adding more = future design doc
- [ ] OpenRouter routing detection uses model-string prefix only (`anthropic/...` → Anthropic-shape, `openai/...` → OpenAI-shape, `google/...` → Gemini-shape, anything else → no-op). Document this in `docs/docs/guides/custom-ai-providers.md`
- [ ] Phase 1 Anthropic placement = system prompt only. Tool-result and last-user breakpoints land in Phase 2 after telemetry
- [ ] Provider quirk reporting uses one-shot per-session warning via existing telemetry/log channel (NOT a per-step warning — that would flood logs)

---

## Solution Design

### Overview

Add a `cache_breakpoints: [CacheBreakpoint]` field to `std/ai.Request` (and `internal/ai.Request`). Each provider's `step.go` interprets the field according to its own caching contract:

- **Anthropic** (direct + Bedrock + Vertex): emits `cache_control: {type: "ephemeral"}` markers on the matching content block(s) — system, last user, or tool_result
- **OpenRouter**: dispatches to the downstream-provider mapping based on the model-string prefix (`anthropic/...` → Anthropic shape, `openai/...` → OpenAI shape, `google/...` → Gemini shape, other → no-op)
- **OpenAI**: NO-OP. OpenAI auto-caches prompts ≥1024 tokens transparently. Emit a one-shot session info log so the user knows hints were observed but not actioned
- **Gemini**: NO-OP for v0.18.4. Emit a one-shot session warning that explicit Context Caching API integration is deferred (Phase 2)
- **Ollama**: NO-OP silently (local model, no caching API)

The field defaults to `[]`, so every existing `ai.step` caller continues to work bit-for-bit identically. Opt-in is a one-line change at the call site.

### Architecture

**Components:**

1. **`std/ai.CacheBreakpoint` AILANG type** — `{ position: string, ttl: string }`. Added to `std/ai.ail`, exported, documented.

2. **`std/ai.Request` field extension** — adds `cache_breakpoints: [CacheBreakpoint]` (default `[]`) to the existing AILANG-side `step` request shape. Threaded through `_ai_step` builtin → `internal/runtime/ai_handler.go` → `internal/ai/Request`.

3. **`internal/ai/Request` field extension** — adds a `CacheBreakpoints []CacheBreakpoint` field. JSON-tagged for round-trip via the existing handler chain.

4. **Per-provider interpreter** — each provider's `step.go` calls a small helper `applyCacheHints(apiReq, req.CacheBreakpoints)` after building the request body. Helper is provider-specific (Anthropic stamps `cache_control` on content blocks; others log + return).

5. **Provider-quirk warnings** — a small `internal/ai/cache_warnings.go` with a once-per-session `sync.Map[providerName]struct{}` so each session emits one warning per provider, not one per step.

6. **motoko opt-in** — `agent_loop_v2.ail` gets a 3-line change: build a `[CacheBreakpoint{position: "system", ttl: "ephemeral"}]` list and pass it on the `step` call.

### Implementation Plan

**Phase 1: AILANG-side type + Request plumbing** (~1 hour, ~30 LOC)
- [ ] Add `CacheBreakpoint` type to `std/ai.ail` (export)
- [ ] Add `cache_breakpoints: [CacheBreakpoint]` field to `Request` (additive, default `[]`)
- [ ] Update `step` signature to accept the new field (back-compat: existing callers continue to work because default is `[]`)
- [ ] Plumb through `_ai_step` builtin → `internal/runtime/ai_handler.go` → `internal/ai.Request.CacheBreakpoints`

**Phase 2: Anthropic provider** (~2 hours, ~80 LOC)
- [ ] Add `CacheControl *cacheControlBlock` field to `stepContentBlock` (omitempty)
- [ ] In `buildStepRequest` ([anthropic/step.go:195](../../../internal/ai/anthropic/step.go#L195)), after building the request, if `req.CacheBreakpoints` contains `{position: "system", ...}`, restructure system from a string to a `[{type:"text", text:"...", cache_control:{type:"ephemeral"}}]` content array
  - **Anthropic-specific quirk**: system content arrays only allowed in their newer API; verify api-version supports it (the `c.apiVersion` field already plumbs in)
- [ ] Add unit test: build request with `{position:"system", ttl:"ephemeral"}`, assert wire JSON contains the expected cache_control block
- [ ] Add fixture test: 3-turn conversation, assert step 2+ has `CacheReadInputTokens > 0` (uses real API — gated by `INTEGRATION_ANTHROPIC=1` env var, like existing fixtures)
- [ ] Verify v0.18.3 hybrid-tool fix still works (cache_control on system + tool_use_id correlation must not interact)

**Phase 3: OpenRouter routing** (~1 hour, ~30 LOC)
- [ ] In `openrouter/step.go`, parse the model-string prefix (`anthropic/...` → call Anthropic-shape mapper; `openai/...` → OpenAI no-op + warning; `google/...` → Gemini no-op + warning)
- [ ] Reuse the Anthropic mapper from Phase 2 (extract to `internal/ai/anthropic/cache.go` so OpenRouter can import it, or duplicate the small helper inline — agent may choose)
- [ ] Test: `anthropic/claude-3-5-haiku` model with `{position:"system"}` produces a request body containing `cache_control` (matches Anthropic-direct)

**Phase 4: OpenAI / Gemini / Ollama no-op + structured warning** (~1 hour, ~30 LOC)
- [ ] OpenAI: in `openai/step.go`, if `req.CacheBreakpoints` is non-empty, emit one-shot warning `cache_hint_ignored_openai_auto_cache` via `cache_warnings.go`
- [ ] Gemini: same pattern, code `cache_hint_ignored_gemini_no_explicit_api`
- [ ] Ollama: silent no-op (local, no caching)
- [ ] `internal/ai/cache_warnings.go`: `sync.Map[string]struct{}` per process, log once per (provider, session) pair

**Phase 5: motoko opt-in** (~30 min, ~20 LOC)
- [ ] In `motoko_agent/src/core/agent_loop_v2.ail`, modify the `step` call site to pass `cache_breakpoints: [{position:"system", ttl:"ephemeral"}]`
- [ ] Update `motoko_agent/CHANGELOG.md`
- [ ] Smoke test: re-run a single benchmark, inspect JSONL — confirm `cache_read != null` for step 1+

**Phase 6: Validation + docs** (~30 min, ~20 LOC)
- [ ] Re-run 3-harness smoke comparison post-issue-#225 fix — capture before/after motoko input_tokens delta
- [ ] Update `docs/docs/guides/custom-ai-providers.md` with `cache_breakpoints` documentation + per-provider behavior table
- [ ] Update `CHANGELOG.md` v0.18.4 entry
- [ ] Add `examples/runnable/ai_caching.ail` showing a 3-turn loop with caching enabled

### Files to Modify/Create

**New files:**
- `internal/ai/cache_warnings.go` (~30 LOC) — once-per-session warning helper
- `internal/ai/anthropic/cache.go` (~50 LOC) — Anthropic cache_control mapper (importable by OpenRouter)
- `examples/runnable/ai_caching.ail` (~40 LOC) — runnable demo

**Modified files:**
- `std/ai.ail` (+15 LOC) — `CacheBreakpoint` type + `cache_breakpoints` field on Request, update `step` doc
- `internal/ai/provider.go` (+10 LOC) — `CacheBreakpoint` Go type + `CacheBreakpoints` field on Request
- `internal/runtime/ai_handler.go` or wherever `_ai_step` lives (+10 LOC) — pass cache_breakpoints through to provider Request
- `internal/ai/anthropic/step.go` (+40 LOC) — `applyCacheHints` call in buildStepRequest, modified `stepContentBlock` with `CacheControl` field
- `internal/ai/openrouter/step.go` (+30 LOC) — model-prefix routing, dispatch to Anthropic mapper or no-op
- `internal/ai/openai/step.go` (+10 LOC) — no-op + warning
- `internal/ai/gemini/step.go` (+10 LOC) — no-op + warning
- `internal/ai/ollama/step.go` (+5 LOC) — silent no-op (no warning, local model)
- `motoko_agent/src/core/agent_loop_v2.ail` (+5 LOC) — opt-in call (cross-repo change, separate PR)

---

## Conflict Surface

This design touches `internal/ai/` (multiple providers) and `std/ai.ail` (stdlib type extension). Required Conflict Surface analysis:

### Syntactic positions touched

- `Request` struct in `internal/ai/provider.go`: adds an optional field. Wire-format is JSON, additive.
- `std/ai.Request` AILANG record type: adds a field with a default. AILANG records support row-polymorphic access, so existing pattern matches don't break.
- `step` AILANG signature in `std/ai.ail`: changes from `step(model, messages, tools)` to one of:
  - **Option A**: same arity, but `Request` carries the field — current `step(model, messages, tools)` becomes a thin wrapper that calls `stepWithRequest(buildDefaultRequest(...))`
  - **Option B**: `step(model, messages, tools, breakpoints)` — extra positional param breaks all callers
- **Option A chosen** (additive, back-compat). The extra arity is hidden behind the wrapper.

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| `Request` struct field set | `Model`, `SystemPrompt`, `Messages`, `Tools`, `Temperature`, `MaxTokens`, ... | additive — JSON `omitempty` semantics |
| `stepContentBlock` (Anthropic) | `text`, `tool_use`, `tool_result` blocks | adding `cache_control` field on blocks of type `text` (system) and `tool_result` (tool result placement) |
| `step` AILANG signature | 3-arg form used by every existing caller of `_ai_step` | back-compat preserved via wrapper |

### Disambiguation strategy

- **Wire shape**: Anthropic accepts both `system: "string"` and `system: [{type:"text", text:"..."}]`. The new path uses the array form ONLY when `req.CacheBreakpoints` contains a `"system"` entry. Otherwise the string form is preserved. This means the existing wire shape for non-cache-using callers is bit-for-bit unchanged.
- **AILANG-side**: `step(model, messages, tools)` keeps working because the `cache_breakpoints` field defaults to `[]`. No row-polymorphism gymnastics needed.
- **Provider routing**: OpenRouter's model-prefix detection is the only "new ambiguity". Decision: trust the model-string prefix (documented OpenRouter convention). Models without a recognized prefix → no-op + once-per-session warning.

### Programs that MUST still work (regression fixtures)

These are the existing test fixtures that exercise `step` and must continue passing post-change:
1. `internal/ai/anthropic/step_test.go` — all existing tests must pass with `CacheBreakpoints == nil` producing identical wire bytes to today
2. `internal/ai/openai/step_test.go` — same back-compat assertion
3. `internal/ai/openrouter/step_test.go` — same back-compat assertion
4. `examples/runnable/effects/ai_dialogue.ail` (or whatever the canonical AILANG `step`-using example is — verify in `examples/manifest.json`)
5. `motoko_agent/test/conversation_loop.test.ts` — multi-turn loop, must continue to function before any motoko-side opt-in

### What deliberately changes

- Anthropic system-prompt wire shape changes from string → content-array WHEN AND ONLY WHEN `cache_breakpoints` includes `{position:"system"}`. This is opt-in.
- `StepResult.cache_read_input_tokens` and `cache_creation_input_tokens` start showing nonzero values for opted-in callers. Existing telemetry consumers (eval-harness JSONL writer, dashboard cost panels) already handle these fields per [std/ai.ail:115-127](../../../std/ai.ail#L115).

Nothing breaks unless explicitly opted in.

---

## Examples

### Example 1: Default behavior (no cache hints — bit-for-bit unchanged)

**Before and After (identical):**
```ailang
import std/ai (step, Message, ToolSchema, AIError)
import std/result (Result, Ok, Err)

let messages = [
  { role: "system", content: "You are a coding assistant.", tool_calls: [], tool_call_id: "" },
  { role: "user", content: "Write a haiku.", tool_calls: [], tool_call_id: "" }
];

match step("claude-3-5-haiku", messages, []) {
  Err(e) => print("error: " ++ e.message),
  Ok(r) => print(r.message.content)
}
-- Wire bytes UNCHANGED. cache_read still null.
```

### Example 2: Opt in to system-prompt caching

**After (caching enabled — system prompt cached after first turn):**
```ailang
import std/ai (step, Request, CacheBreakpoint, Message)
import std/result (Result, Ok, Err)

let req: Request = {
  model: "claude-3-5-haiku",
  messages: long_conversation,  -- 11 turns, ~21k-token system prompt
  tools: my_tools,
  cache_breakpoints: [
    { position: "system", ttl: "ephemeral" }
  ]
};

match stepWithRequest(req) {
  Ok(r) =>
    print("step in: " ++ show(r.input_tokens) ++ ", cache_read: " ++ show(r.cache_read_input_tokens))
    -- Step 0: in=21000, cache_read=0,    cache_create=21000  (cold)
    -- Step 1: in=21100, cache_read=21000, cache_create=0     (warm)
    -- Step 2: in=21250, cache_read=21100, cache_create=0     (warm)
    -- ...
}
```

Cost per session: cold ~$0.011 + 10 warm × ~$0.001 = ~$0.021 (vs $0.16 without caching, ~8× drop).

### Example 3: motoko opt-in (single-line change in agent_loop_v2.ail)

**Before:**
```ailang
let result = step(model, messages, tools)
```

**After:**
```ailang
let result = stepWithRequest({
  model: model,
  messages: messages,
  tools: tools,
  cache_breakpoints: [{ position: "system", ttl: "ephemeral" }]
})
```

### Example 4: OpenAI with cache hint (NO-OP + one-shot warning)

```ailang
let req: Request = { model: "gpt-4o-mini", messages: msgs, tools: [],
  cache_breakpoints: [{ position: "system", ttl: "ephemeral" }] };
stepWithRequest(req)
-- AILANG logs once per session: "[ai] cache_hint_ignored_openai_auto_cache:
--   OpenAI auto-caches prompts ≥1024 tokens; explicit hints have no effect"
-- Behavior identical to no hints. OpenAI server-side caching still works.
```

---

## Success Criteria

- [ ] `cache_breakpoints: []` (default) produces bit-for-bit identical wire bytes vs pre-change for all 5 providers
- [ ] Anthropic-direct: `{position:"system", ttl:"ephemeral"}` produces a wire body with `system: [{type:"text", text:"...", cache_control:{type:"ephemeral"}}]`
- [ ] AWS Bedrock backend (verified via OpenRouter `anthropic/...` route, since we don't have direct Bedrock client) accepts the same shape without 400 errors
- [ ] 3-turn integration test against real Anthropic API: step 1+ shows `CacheReadInputTokens > 0`
- [ ] OpenAI with `cache_breakpoints != []` logs `cache_hint_ignored_openai_auto_cache` exactly once per process
- [ ] Gemini with `cache_breakpoints != []` logs `cache_hint_ignored_gemini_no_explicit_api` exactly once per process
- [ ] OpenRouter `anthropic/claude-3-5-haiku` route forwards `cache_control` correctly
- [ ] OpenRouter `openai/gpt-4o-mini` route does NOT forward `cache_control` (would be invalid for OpenAI), logs warning
- [ ] All existing tests in `internal/ai/*/step_test.go` pass without modification
- [ ] motoko opt-in change in `agent_loop_v2.ail` produces nonzero `cache_read` in JSONL on the next eval-suite smoke run
- [ ] Documentation updated (`docs/docs/guides/custom-ai-providers.md`)
- [ ] CHANGELOG entry under v0.18.4
- [ ] `examples/runnable/ai_caching.ail` runs and demonstrates cache hits

---

## Testing Strategy

**Unit tests:**
- `anthropic/cache_test.go`: `applyCacheHints` produces correct wire shape for system / last_user / tool_result positions
- `openrouter/cache_routing_test.go`: model-prefix sniffing dispatches to correct mapper
- `cache_warnings_test.go`: once-per-session semantics (concurrent calls produce 1 warning)

**Integration tests** (gated by `INTEGRATION_ANTHROPIC=1`):
- `anthropic/cache_integration_test.go`: 3-turn loop with stable system prompt; assert `CacheReadInputTokens > 0` on step 1 and 2
- `openrouter/cache_integration_test.go`: same shape via OpenRouter `anthropic/...` model

**Regression-surface tests** (per Conflict Surface enumeration):
- Snapshot test: `Request{...no cache_breakpoints}` → wire bytes match golden file (locked by hash)
- All `step_test.go` files in all providers run unchanged

**Manual testing:**
- Re-run 3-harness smoke comparison after motoko opt-in lands; verify motoko input-token total drops measurably
- Verify warning text shows up in eval-suite session logs once, not per-step

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Where `applyCacheHints` lives in the Anthropic package (separate `cache.go` vs inline in `step.go`) — agent may choose
- Whether the OpenRouter Anthropic-mapper is imported from `internal/ai/anthropic` or duplicated inline (~30 LOC) — agent may choose, optimizing for testability
- Exact warning text format — agent may choose, but must be greppable (`cache_hint_ignored_<provider>_<reason>`)
- Whether `examples/runnable/ai_caching.ail` uses an Anthropic or Ollama model for demonstration — agent may choose; if Anthropic, gate behind `ANTHROPIC_API_KEY` check at runtime
- Whether `stepWithRequest` is the actual wrapper name or if `step` is overloaded — agent may choose. Anything back-compat is acceptable

---

## Non-Goals

**Not attempted in this feature:**
- Gemini Context Caching API integration — Phase 2 / separate design doc. Async cache-create-then-reference is structurally different
- Per-message `cache_control` placement granularity beyond the 3 documented positions — wider vocabulary deferred until telemetry shows it matters
- Detection of Bedrock-vs-Anthropic-direct routing — both honor `cache_control` identically per Anthropic docs; no per-backend code path needed
- Auto-detection of "should we cache?" based on prompt size — explicit opt-in only. Auto-heuristics are a future enhancement
- Caching for non-`step` AI calls (`call`, `callJson`, `callJsonSimple`, `callImage`) — these are single-shot, no loop, marginal value
- Cross-session cache reuse — Anthropic ephemeral TTL is 5 minutes; users wanting longer-lived caches need the upcoming Anthropic 1h+ tiers (out of scope until Anthropic stabilizes the API)

---

## Timeline

**Day 1** (~3 hours):
- Phase 1: AILANG type + Request plumbing (1h)
- Phase 2: Anthropic provider (2h)

**Day 2** (~3 hours):
- Phase 3: OpenRouter routing (1h)
- Phase 4: OpenAI/Gemini/Ollama no-op + warnings (1h)
- Phase 5: motoko opt-in (30 min)
- Phase 6: Validation + docs (30 min)

**Total: ~6 hours across 2 calendar days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Anthropic system content-array shape rejected by older `apiVersion` | High | Verify `c.apiVersion` is recent enough at request build time; fall back to string form + warning if old |
| Bedrock strict validation rejects `cache_control` on tool_result blocks (precedent: v0.18.3 hybrid-tool correlation) | Medium | Phase 1 only places cache_control on `system`; defer tool_result placement to Phase 2 with explicit Bedrock testing |
| OpenRouter changes its model-string prefix convention | Low | One greppable place to update; once-per-session warning surfaces unknown prefixes loudly |
| Anthropic counts cache_create as billable on first cold turn (which it does, at 1.25× normal price) | Low | Documented; one cold turn vs N warm turns is still net win for any N≥2 |
| Warnings flood logs in long sessions | Low | `sync.Map[providerName]struct{}` ensures one warning per (process, provider) — explicitly tested |
| motoko opt-in interacts with v0.18.3 hybrid-bash tool_use_id correlation | Medium | Phase 2 unit test specifically combines system cache_control + hybrid tool_use synthesis |

---

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_18_3/m-motoko-hybrid-tool-correlation.md](../../implemented/v0_18_3/m-motoko-hybrid-tool-correlation.md) — Precedent for Anthropic content-block manipulation. Same package (`internal/ai/anthropic/`) modified for Bedrock correlation. Cache_control placement on tool_result must not interact with synthesized tool_use blocks
- [design_docs/implemented/v0_18_1/m-motoko-eval-harness-hardening.md](../../implemented/v0_18_1/m-motoko-eval-harness-hardening.md) — Established the Bedrock-vs-Anthropic-direct asymmetry pattern (dotted-tool-name issue). Same pattern applies here: Bedrock validates strictly, Anthropic-direct is permissive
- [design_docs/implemented/v0_15_0/m-ai-provider-config.md](../../planned/v0_15_0/m-ai-provider-config.md) — `[[ai_provider]]` registry. The cache-hint contract should be expressible in the registry schema for future config-driven providers (Phase 2 work, not blocking)
- [design_docs/implemented/v0_17_0/m-ai-streaming-helper.md](../../implemented/v0_17_0/m-ai-streaming-helper.md) — `std/ai/streaming` shape; if streaming gains caching support later, follow the same `cache_breakpoints` field pattern

**Auto-search results (lower relevance, retained for review):**
- [design_docs/implemented/v0_6_0/m-doc-sem-lazy-embeddings.md](../../implemented/v0_6_0/m-doc-sem-lazy-embeddings.md) (0.46) — different "caching" (doc embeddings), unrelated
- [design_docs/implemented/v0_3_22/M-EVAL-HTTP-FIX-sprint-plan.md](../../implemented/v0_3_22/M-EVAL-HTTP-FIX-sprint-plan.md) (0.43) — eval HTTP plumbing, unrelated
- [design_docs/planned/v0_15_2/m-eval-followups.md](../../planned/v0_15_2/m-eval-followups.md) (0.42) — possibly worth checking if it has overlapping eval-cost concerns
- [design_docs/planned/v0_13_0/m-dx-agent-eval-gaps.md](../../planned/v0_13_0/m-dx-agent-eval-gaps.md) (0.41) — agent-eval friction; possible alignment

**External / cross-repo:**
- [arniwesth/motoko_agent#225](https://github.com/arniwesth/motoko_agent/issues/225) — the proximate-cause BashExec storm bug (NOT addressed here, see Honest Framing)
- Anthropic prompt caching docs: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
- OpenAI prompt caching: https://platform.openai.com/docs/guides/prompt-caching (auto, no API)
- Gemini Context Caching: https://ai.google.dev/gemini-api/docs/caching (async, deferred to Phase 2)

---

## References

- [Design Axioms](/docs/references/axioms) — A7 (Machines First) and A9 (Cost Visibility) are the load-bearing axioms here
- [Philosophical Foundations](/docs/references/philosophical-foundations)
- AILANG `internal/ai/provider.go:23-160` — current `Request` and `Response` structs (cache fields exist on Response, missing on Request)
- AILANG `std/ai.ail:115-127` — `StepResult` already has `cache_read_input_tokens` / `cache_creation_input_tokens` (telemetry contract pre-existing)
- AILANG `internal/ai/anthropic/step.go:111-119` — response-side cache token plumbing (the gap is on the request side)

---

## Future Work

- **Phase 2 — Wider Anthropic placement**: tool_result and last_user breakpoints (after telemetry confirms ROI for system-only)
- **Phase 2 — Gemini Context Caching API**: async cache-create + cached-content reference. Will need a new effect-row-friendly handle type (`CachedContent`) and a separate `step` overload that takes one. Separate design doc
- **Phase 3 — Auto-cache heuristic**: AILANG could detect "system prompt > N tokens AND > M turns predicted" and apply caching automatically. Would need usage telemetry to tune thresholds
- **Phase 3 — Provider-config-driven cache schema**: extend `[[ai_provider]]` blocks in `ailang.toml` to declare `cache_shape: "anthropic_ephemeral" | "openai_auto" | "gemini_cachedcontent" | "none"` so config-driven providers honor cache hints uniformly
- **Cross-repo motoko v3 design**: longer-term motoko could expose a per-task cache profile (e.g. "this benchmark is single-shot, don't cache" vs "this benchmark is multi-turn agent, cache aggressively"). Out of scope for v0.18.4 — motoko opt-in is uniform/system-only

---

**Document created**: 2026-05-08
**Last updated**: 2026-05-08
