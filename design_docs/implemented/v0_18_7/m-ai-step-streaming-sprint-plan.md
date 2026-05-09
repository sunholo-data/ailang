# Sprint Plan: M-AI-STEP-STREAMING (v0.18.7)

**Sprint ID**: M-AI-STEP-STREAMING
**Target Version**: v0.18.7
**Design Doc**: [m-ai-step-streaming.md](m-ai-step-streaming.md)
**Estimated**: ~7-8 hours, ~330 LOC across implementation + tests + docs
**Risk Level**: Medium (touches `internal/ai/` provider streaming + AIHandler interface; back-compat is the main risk surface — but pattern proven by M-AI-PROMPT-CACHING v0.18.4)
**Created**: 2026-05-09

---

## Sprint Goal

Ship `std/ai.stepWithStream` — same return shape as `step()`/`stepWithCache()`, plus a per-chunk callback firing during SSE processing — so motoko (and any future AILANG agent) can render LLM output incrementally without giving up typed StepResult, tool dispatch, cost tracking, or cache metrics. Closes [arniwesth/motoko_agent#10](https://github.com/arniwesth/motoko_agent/discussions/10).

**Out of scope** (per design doc Non-Goals):
- `ToolCallDelta` + `ThinkingDelta` chunk variants (Phase 2 separate sprint)
- Full streaming for Gemini/Ollama (Phase 1 ships NO-OP fallback; full Phase 2)
- Backpressure / pause-resume callback semantics
- motoko cross-repo opt-in (separate PR after this lands)

---

## Velocity Context

Recent v0.18.x sprints (today): typical 2-4h sprints land 80-300 LOC each. M-AI-PROMPT-CACHING (v0.18.4) was the closest precedent — same shape (additive opt-in callback), 4 milestones, ~430 LOC, ~40min wall-clock. This sprint is bigger (~330 LOC for the streaming + accumulator wrappers per provider) but follows the exact proven shape.

---

## Milestone Breakdown

4 milestones, sequential dependencies, ~7-8h total.

### M1 — AILANG surface + AIHandler interface evolution

**Estimated**: 2 hours, ~80 LOC

**Description**: Add the `StreamChunk` Go type and AILANG type. Add `stepWithStream` function to `std/ai.ail`. Add `_ai_step_with_stream` builtin and `aiStepWithStream` effect op (with closure-wrapping helper that bridges the AILANG closure to a Go callback). Widen `effects.AIHandler` interface from 6 methods to 7 — all 5 existing implementations need `StepWithStream` (AIContext delegates; StubAIHandler fires one synthetic ContentDelta+Usage; fakeStepHandler test stub captures callback for assertion; `ai.Handler` real entry delegates to provider; `WasmAIHandler` returns "not supported in WASM" matching its Step/StepWithCache siblings).

**Files**:
- `internal/ai/provider.go` (+25 LOC) — `StreamChunk` interface + `streamChunkContent`/`streamChunkUsage` concrete types
- `std/ai.ail` (+20 LOC) — `StreamChunk` type, `stepWithStream` function (declaration only; impl flows through builtin)
- `internal/effects/ai.go` (+30 LOC) — interface 7th method + AIContext delegate + StubAIHandler synthetic-callback impl
- `internal/effects/ai_step.go` (+60 LOC) — `aiStepWithStream` effect op + closure-wrapping helper
- `internal/effects/ai_step_test.go` (+10 LOC) — `fakeStepHandler.StepWithStream` test stub
- `internal/builtins/ai_step.go` (+50 LOC) — `_ai_step_with_stream` builtin registration + arg decoding
- `internal/ai/handler.go` (+15 LOC) — `Handler.StepWithStream` delegating to provider
- `cmd/wasm/effects.go` (+10 LOC) — `WasmAIHandler.StepWithStream` "not supported"

**Acceptance criteria**:
- AILANG-side: `import std/ai (stepWithStream, StreamChunk)` resolves
- `_ai_step_with_stream` builtin signature appears in golden file (regenerate)
- `effects.AIHandler` interface compiles with all 5 implementations including new method
- `go build ./...` clean
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` clean (WASM trap preemption per design doc Risk #2)
- AILANG closure → Go callback round-trip works (unit test calls `aiStepWithStream` with `fakeStepHandler` that fires 2 chunks; AILANG closure captures both)
- Existing `internal/effects/ai_step_test.go` + `ai_step_with_cache_test.go` (22+ v0.18.4 tests) all pass without modification

**Dependencies**: none

---

### M2 — Anthropic streaming + typed-StepResult accumulator

**Estimated**: 2 hours, ~120 LOC

**Description**: Implement `Step.StreamStep(ctx, req, onChunk)` for Anthropic. Set `stream: true` on Messages API request. Parse SSE event stream (`message_start` → init, `content_block_start`/`content_block_delta`/`content_block_stop` → text deltas + tool_use buffering, `message_delta` → stop_reason + final usage, `message_stop` → return assembled `*ai.Response`). Fire `ContentDelta` per text delta. Fire `Usage` from `message_delta`. Buffer tool_use blocks for final `StepResult.ToolCalls`. Reuse v0.18.4's `applyCacheHints` for cache_control plumbing. Reuse v0.18.3's hybrid-tool tool_use_id correlation (no new code).

**Files**:
- `internal/ai/anthropic/streamStep.go` (~110 LOC, new) — Anthropic streaming impl
- `internal/ai/anthropic/handler.go` or `step.go` (+10 LOC) — wire `StreamStep` into Handler
- `internal/ai/anthropic/streamStep_test.go` (~50 LOC, new) — unit tests with mock SSE source

**Acceptance criteria**:
- Anthropic `StreamStep` produces `*ai.Response` byte-equal to what `Step` would return for the same input (unit test with golden response fixture)
- `ContentDelta` fires for each `content_block_delta` text event (unit test asserts ≥2 fires for a 2-block response)
- `Usage` fires once after `message_delta` with `cache_read_input_tokens` populated (unit test)
- Tool_use blocks buffered → appear in final `StepResult.ToolCalls` correctly (unit test with multi-tool fixture)
- v0.18.4 cache_control wire shape preserved (existing snapshot test from `cache_test.go` still passes for non-streaming path)
- v0.18.3 hybrid-tool synthesis still works against Bedrock (no regression — verified by existing `step_test.go`)
- Existing `internal/ai/anthropic/*_test.go` tests pass without modification

**Dependencies**: M1 (need `AIHandler.StepWithStream` interface signature)

---

### M3 — OpenAI + OpenRouter streaming

**Estimated**: 1.5 hours, ~80 LOC

**Description**: OpenAI streaming via `chat/completions?stream=true`. Parse `data:` SSE lines, fire `ContentDelta` per `delta.content`, fire `Usage` from final `data:` event with `usage`. OpenRouter dispatches by model prefix (anthropic/* → Anthropic-shape streaming via M2's impl; openai/google/other → OpenAI-shape). Reuse OpenAI `BuildChatStepRequest` / `ParseChatStepResponse` helpers where possible. Tool_calls argument deltas are buffered internally (final StepResult.ToolCalls populated) — per-fragment ToolCallDelta is Phase 2.

**Files**:
- `internal/ai/openai/streamStep.go` (~50 LOC, new) — OpenAI streaming impl
- `internal/ai/openai/step.go` (+10 LOC) — wire StreamStep
- `internal/ai/openrouter/streamStep.go` (~30 LOC, new) — model-prefix routing
- `internal/ai/openrouter/step.go` (+5 LOC) — wire StreamStep
- `internal/ai/openai/streamStep_test.go` (~30 LOC, new) — unit tests

**Acceptance criteria**:
- OpenAI `StreamStep` produces `*ai.Response` byte-equal to `Step` for same input
- `ContentDelta` fires per OpenAI SSE `delta.content`; `Usage` from final `usage` block
- OpenRouter routes `anthropic/claude-3-5-haiku` (or similar) through Anthropic-shape streaming (uses M2's impl)
- OpenRouter routes `openai/gpt-4o-mini` through OpenAI-shape streaming
- Existing `internal/ai/openai/*_test.go` and `internal/ai/openrouter/*_test.go` pass without modification

**Dependencies**: M2 (OpenRouter `anthropic/*` routing reuses M2's Anthropic streaming impl)

---

### M4 — NO-OP fallbacks + integration test + CHANGELOG + tutorial

**Estimated**: 1.5 hours, ~70 LOC

**Description**: Add `StepWithStream` to Gemini/Ollama/configdriven providers as NO-OP fallback (call existing `Step`/`StepWithCache`, fire one synthetic `ContentDelta(text)` + one `Usage`). Add unit test asserting content assembly via callback equals `StepResult.Text`. Add integration test (gated by `AILANG_INTEGRATION_TESTS=1`) hitting real Anthropic. Add `examples/runnable/ai_streaming.ail` demo (uses `--ai-stub` so it works without API key). Update CHANGELOG entry. Update `std/ai/streaming.ail` "v1 limitations" comment to mention `stepWithStream`.

**Files**:
- `internal/ai/gemini/step.go` (+10 LOC) — NO-OP `StepWithStream` fallback
- `internal/ai/ollama/step.go` (+10 LOC) — NO-OP fallback
- `internal/ai/configdriven/...` (+10 LOC) — NO-OP fallback
- `internal/ai/anthropic/streamStep_integration_test.go` (~30 LOC, new) — gated `AILANG_INTEGRATION_TESTS=1`
- `examples/runnable/ai_streaming.ail` (~30 LOC, new) — runnable demo
- `examples/manifest.json` (+8 LOC) — register example
- `std/ai/streaming.ail` (+5 LOC doc-only) — point at `stepWithStream` in v1 limitations
- `changelogs/v0.10-current.md` (+30 LOC) — `## [Unreleased]` entry

**Acceptance criteria**:
- Gemini/Ollama/configdriven NO-OP fallback produces a working callback flow (single ContentDelta + Usage; verified by unit test)
- Unit test: assemble content via callback closure, assert equals `StepResult.Text`
- Integration test passes with `AILANG_INTEGRATION_TESTS=1` against real Anthropic; per-chunk callback fires ≥2 times for non-trivial response; assembled content equals `StepResult.Text`
- `examples/runnable/ai_streaming.ail` runs cleanly with `--ai-stub`
- `make verify-examples` passes
- `make test` clean
- `make lint` clean
- `make brain-index-syntax-reset` produces clean indexed counts (sanity)

**Dependencies**: M1, M2, M3

---

## Day-by-Day Plan

**Day 1** (~5-6 hours, today):
- Hour 1-2: M1 (interface + builtin + closure wrapping)
- Hour 3-4: M2 (Anthropic streaming + typed accumulator)
- Hour 5: M3 (OpenAI + OpenRouter)

**Day 2** (~2 hours, tomorrow morning):
- Hour 6-7: M4 (NO-OP fallbacks + integration test + CHANGELOG + example)
- Release v0.18.7 patch
- Bump motoko-bisect-gap1's `install-prerequisites.sh` pin to v0.18.7

**Total: ~7-8 hours across 2 calendar days**

---

## Success Metrics

- All 4 milestones pass acceptance criteria
- `make test` and `make lint` clean
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` clean (preempts v0.18.4 trap)
- `make verify-examples` passes (new ai_streaming.ail runs)
- New test files: unit tests in `internal/effects/ai_step_with_stream_test.go`, per-provider `streamStep_test.go` files, integration test
- All existing `internal/ai/*/step_test.go` and `internal/effects/ai_step*.go` tests pass without modification (the back-compat assertion)
- Documentation: `std/ai/streaming.ail` "v1 limitations" comment updated; `examples/runnable/ai_streaming.ail` registered in manifest
- CHANGELOG entry under v0.18.7

**Post-sprint validation gate** (NOT a sprint milestone — separate cross-repo work):
- motoko-side opt-in lands in `motoko_agent/src/core/test/stub_step.ail` (5-10 line change, mirrors v0.18.4 cache opt-in pattern)
- TUI shows token-by-token rendering in a real motoko eval-suite run

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| AIHandler 7-method interface breaks downstream embeddings | Medium | Mirror v0.18.4 (which added StepWithCache as 4th) — proven pattern. WasmAIHandler is canonical "I don't support this" example; all stubs default-delegate to Step + fire one synthetic ContentDelta+Usage |
| WASM build fails (v0.18.4 trap) | Medium | Pre-tag check now standard. Specifically run `GOOS=js GOARCH=wasm go build ./cmd/wasm` after M1 completes — catches missing `WasmAIHandler.StepWithStream` immediately |
| Anthropic SSE parsing has edge cases (incomplete deltas, stream interruption mid-tool_use) | Medium | Reuse existing v0.15.0 SSE-parsing code; only the typed-accumulator wrapper is new. Unit tests with mock SSE source cover edge cases |
| Closure-bridging Go↔AILANG has subtleties | Medium | `_ai_call_stream` already does this for string accumulation since v0.15.0; we generalize the pattern. Use the same effect-op closure-handle convention |
| Sprint underestimated due to per-provider quirks (Anthropic vs OpenAI vs OpenRouter SSE shape differences) | Low | Each provider has its existing streaming code path (since v0.15.0); we wrap, not write from scratch. If M3 balloons, the simplest cut is: drop OpenRouter prefix-routing for streaming (just call OpenAI-shape always), defer to v0.18.8 |

---

## References

- Design doc: [m-ai-step-streaming.md](m-ai-step-streaming.md)
- Source event: [arniwesth/motoko_agent#10 Discussion](https://github.com/arniwesth/motoko_agent/discussions/10)
- Direct shape precedent: M-AI-PROMPT-CACHING (v0.18.4, [design_docs/implemented/v0_18_4/m-ai-prompt-caching.md](../../implemented/v0_18_4/m-ai-prompt-caching.md))
- Existing infra to wrap: M-AI-STREAMING-HELPER (v0.15.0, `_ai_stream_call` in `internal/builtins/ai.go:294`)

---

**Document created**: 2026-05-09
