# M-AI-STREAMING-HELPER Sprint Plan

**Sprint ID**: `M-AI-STREAMING-HELPER`
**Design doc**: [m-ai-streaming-helper.md](./m-ai-streaming-helper.md)
**Target**: v0.15.0 (pulled forward from v0.17.0 — see [motoko-integration-sequence.md](../motoko-integration-sequence.md) Phase 1)
**Created**: 2026-05-04
**Estimated**: ~6 hours (3 milestones)
**Risk level**: Low

## Sprint summary

Ship the AILANG-side helper that closes the AI streaming discovery + ergonomics gap. **All Go-side primitives already shipped in parallel threads** (M-AI-PROVIDER-CONFIG M1-M4 just landed with 95 tests; `_stream_sse_post` builtin already exists; `[ai_provider.streaming]` schema already parsed). The remaining work is:

1. A small Go bridge that ties the existing pieces together (registry lookup → body construction → auth → SSE-POST call)
2. A thin AILANG module at `std/ai/streaming.ail` that exposes the bridge with typed `TokenDelta` parsing
3. Recipe page + runnable example + cross-link docstrings

This is the load-bearing milestone for unblocking PR-A on the motoko fork (drops their 6 streaming Go files entirely).

## Velocity check

Recent velocity for this branch (last 7 days, sprint-executor cadence):
- M-AI-PROVIDER-CONFIG: M1+M2+M3+M4 in one session, ~1500 LOC implementation + 1200 LOC tests + docs + 95 tests passing → comfortably exceeds the ~6-hour estimate for this small follow-on.
- `_stream_sse_post` already in place removes the largest unknown.
- All schema work is reused — no new TOML fields.

**Recommended pace**: ship all 3 milestones in a single session. Risk is low because each milestone is independently testable and the Go primitives have an existing test suite to compose against.

## Architectural deviation from design doc

The design doc says "NO new builtin — pure AILANG plus one runtime hook in the existing AI handler." That phrasing was written when we expected streaming to dispatch through the `string -> string` `_ai_call` path. Streaming actually returns a `StreamConn`, not a string — so it can't go through the existing AI handler signature.

**Sprint deviation**: introduce a single new builtin `_ai_stream_call(provider_name: string, model: string, messages_json: string) -> Result[StreamConn, AIError] ! {AI, Stream, Net}` that does registry lookup + body construction + auth + `StreamSSEPost`. This is the load-bearing change vs the design doc.

**Why this is OK**: The design's hard constraint was "NO new effect" — that's preserved (we reuse `AI`, `Stream`, `Net`). The "no new builtin" was a simplicity note that turns out to be incompatible with the AI-cap-and-budget integration requirement. Per the design doc's own footer ("If implementation deviates from this doc, update this doc first"), the design doc must be amended to reflect this decision before sprint sign-off.

## Milestones

### M1: Go bridge + `_ai_stream_call` builtin + tests (~3.5 hrs, ~250 LOC + 200 LOC tests)

**Files to create:**
- `internal/ai/configdriven/streaming.go` — `StreamCall` Go entry point (registry lookup → reuses `buildRequestBody` from `shapes.go` → reuses `applyAuth` from `auth.go` → calls `effects.StreamSSEPost` infrastructure)
- `internal/ai/configdriven/streaming_test.go` — httptest.Server tests for OpenAI shape + Anthropic shape, snapshot test asserting streaming AI span shape == non-streaming AI span shape

**Files to modify:**
- `internal/builtins/ai.go` (or similar) — register `_ai_stream_call(provider, model, messages_json) -> Result[StreamConn, AIError] ! {AI, Stream, Net}` builtin
- `internal/effects/ai.go` — register the `AI.streamCall` effect op that delegates to `configdriven.StreamCall`

**Acceptance criteria:**
- `_ai_stream_call("openai", "gpt-4o", "[{\"role\":\"user\",\"content\":\"hi\"}]")` returns a `StreamConn` against a mock SSE server
- AI cap missing → fails fast with clear error
- Provider not registered → returns `Err(AIError{ code: "ProviderNotFound" })`
- Unknown shape → returns `Err(AIError{ code: "UnsupportedShape" })`
- Snapshot test confirms span attributes match non-streaming `_ai_call` shape (same `ai.provider`, `ai.model`, `ai.tokens_in`, etc., plus streaming-specific `ai.stream_id`)
- httptest.Server tests cover OpenAI and Anthropic SSE wire formats with realistic delta JSON

### M2: `std/ai/streaming.ail` AILANG module + cross-links (~1.5 hrs, ~120 LOC + 50 LOC docstring updates)

**Files to create:**
- `std/ai/streaming.ail` (≤150 LOC) — exports:
  - `type TokenDelta = { text: string, reasoning: string, done: bool }`
  - `type AIError = { code: string, message: string, retryable: bool }`
  - `func openaiCompatStream(provider: string, model: string, messages: [Message]) -> Result[StreamConn, AIError] ! {AI, Stream, Net}` — calls `_ai_stream_call` with serialised messages
  - `func anthropicStream(provider: string, model: string, messages: [Message]) -> Result[StreamConn, AIError] ! {AI, Stream, Net}` — same builtin, different shape selected by registered provider config
  - `func parseDelta(provider: string, event: StreamEvent) -> Option[TokenDelta]` — looks up the provider's `streaming.delta_path` and `streaming.reasoning_path` JSONPaths from registered config and applies them to the SSE event
- (No new test files — covered by M1 Go-side tests + the M3 example exercising the AILANG surface)

**Files to modify:**
- `std/ai.ail` — add `## Token streaming` cross-link section pointing at `std/ai/streaming` and the recipe page
- (Find or create) `stdlib/std/stream/stream.ail` or equivalent — add `## See also` cross-link pointing at `std/ai/streaming`

**Acceptance criteria:**
- `std/ai/streaming.ail` ≤150 LOC
- Cross-link docstrings present in both `std/ai` and `std/stream`
- Module compiles and exports surface match the design doc
- Caller does NOT supply baseUrl or apiKey — only `provider`, `model`, `messages`

### M3: Recipe page + runnable example + design-doc reconciliation (~1 hr, ~250 LOC docs + 50 LOC AILANG example + 50 LOC CHANGELOG)

**Files to create:**
- `examples/runnable/ai_stream_openai.ail` — end-to-end demo against `openai` provider, accumulates streaming output, shows reasoning-text handling. Capabilities: `--caps AI,Stream,Net,IO`.
- `docs/docs/recipes/ai-token-streaming.md` (~200 lines) — three provider recipes (OpenAI, OpenRouter, Anthropic), reasoning-model handling, dispatch decision tree, drop-down to `std/stream` primitives for non-AI streaming.

**Files to modify:**
- `design_docs/planned/v0_17_0/m-ai-streaming-helper.md` — amend to reflect the `_ai_stream_call` builtin (architectural deviation note above), update Status to "Implemented", note v0.15.0 ship.
- `changelogs/v0.10-current.md` — add bullets under the `[Unreleased] - targeting v0.15.0` heading (alongside the M-AI-PROVIDER-CONFIG entry I just added).
- `design_docs/planned/motoko-integration-sequence.md` — flip M-AI-STREAMING-HELPER row to ✅ in the status board, update Phase 2 / Phase 3 status accordingly.
- After release: move `m-ai-streaming-helper.md` and this sprint plan to `design_docs/implemented/v0_16_x/`.

**Acceptance criteria:**
- Recipe page renders cleanly in Docusaurus
- Example runs end-to-end against a mock SSE server (gated test in CI; manual run against real OpenAI documented but not in CI)
- CHANGELOG entry references the new module + recipe page + example
- Design doc updated and ready to move to `implemented/`
- Master sequence doc reflects new state

## Day-by-day breakdown

This is a single-session sprint (~6 hours). No multi-day breakdown needed.

| Hour | Task |
|------|------|
| 0:00-0:30 | Read existing infrastructure: `internal/ai/configdriven/`, `internal/effects/stream_sse.go`, current `_ai_call` builtin registration, `AIProviderStreaming` struct |
| 0:30-2:00 | M1: write `streaming.go`, register `_ai_stream_call` builtin, register `AI.streamCall` effect op |
| 2:00-3:30 | M1: httptest.Server tests, snapshot trace-shape test |
| 3:30-4:30 | M2: write `std/ai/streaming.ail`, cross-link docstrings |
| 4:30-5:00 | M3: write runnable example, verify it compiles + runs against mock |
| 5:00-5:45 | M3: recipe page (`docs/docs/recipes/ai-token-streaming.md`) |
| 5:45-6:00 | M3: amend design doc, write CHANGELOG bullets, update master sequence doc |

## Success metrics

- All M1+M2+M3 acceptance criteria met
- `make ci` passes
- Existing 95 M-AI-PROVIDER-CONFIG tests still pass (no regression)
- Total new LOC: implementation ~250 + AILANG ~120 + tests ~200 + docs ~250 + example ~50 = **~870 LOC**
- Single new builtin (`_ai_stream_call`); no new effect; no new module-level abstractions
- Streaming AI span snapshot identical (modulo streaming-specific keys) to non-streaming span — verified by test

## Dependencies and open questions

**All dependencies shipped:**
- M-AI-PROVIDER-CONFIG ✅ (this session)
- `_stream_sse_post` builtin ✅ (parallel thread, v0.15.0)
- M-AI-EFFECT-MODES ✅ (v0.15.0)
- M-AI-OPENROUTER ✅ (v0.16.x, wired into `ailang run`)

**Open question to resolve before/during execution:**
1. The `Message` type — does it already exist in `std/ai`? If not, should it be defined in `std/ai/streaming.ail` or hoisted to `std/ai.ail` so non-streaming calls can adopt it later?
2. Should `parseDelta` accept the StreamEvent variant directly, or should it take the raw JSON string from a generic SSE event? (Affects the AILANG signature; minor.)
3. Where does the `_ai_stream_call` builtin's effect op register live — `internal/effects/ai.go` (existing AI ops) or a new `internal/effects/ai_streaming.go`? (Sprint-executor decides; both are fine.)

**Risks and mitigations:**

| Risk | Likelihood | Mitigation |
|------|-----------|-----------|
| Existing `_stream_sse_post` infrastructure has subtle limitations the design didn't anticipate | Low | Sprint-executor reads `internal/effects/stream_sse.go` end-to-end before writing M1 |
| Snapshot test for span equality is hard to make stable | Med | Use only deterministic attribute keys; mock the trace exporter |
| `std/ai/streaming.ail` exceeds 150 LOC | Low | Hard cap; if exceeded, push parseDelta logic into the Go bridge (TokenDelta becomes a new builtin return type) |
| Providers register streaming = false in their config; the M1 builtin needs to refuse cleanly | Low | Add `CapabilityNotSupported` error path in M1 |

## Handoff to sprint-executor

This sprint is small enough to execute in a single session. The acceptance criteria are concrete and the milestones are independently testable. Ready for sprint-executor.

```
SPRINT_PLAN_PATH: design_docs/planned/v0_17_0/m-ai-streaming-helper-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-AI-STREAMING-HELPER.json
```
