# M-AI-CALL-STREAM-HELPER Sprint Plan

**Sprint ID**: `M-AI-CALL-STREAM-HELPER`
**Design doc**: [m-ai-call-stream-helper.md](./m-ai-call-stream-helper.md)
**Target**: v0.15.1 (patch release)
**Created**: 2026-05-05
**Estimated**: ~5 hours (2 milestones)
**Risk level**: Low

---

## Sprint Summary

Ship a synchronous accumulator wrapper `callStream(provider, model, messagesJson) -> Result[string, AIError] ! {AI, Stream, Net}` over v0.15.0's `openaiCompatStream`. Implementation lives entirely in a Go-side builtin `_ai_call_stream` (option C from the design doc) that reuses the existing `_ai_stream_call` op for connection-opening and adds the loop + accumulator + delta extraction in Go.

**Headline outcome**: motoko_agent's v0.15.0 migration drops from ~80–120 LOC of event-loop restructuring to ~10 LOC of 1-line API swaps. Generalises to any future motoko-style consumer.

## Velocity check

Recent velocity (last 7 days, this branch):
- M-AI-PROVIDER-CONFIG (4 milestones, 95 tests, ~3,500 LOC): single full-day session
- M-AI-STREAMING-HELPER (3 milestones, 14 tests, ~700 LOC): half-day session
- v0.15.0 release + post-tag CI fixes: 2 hours

This sprint is much smaller than either of those — pure layering work that reuses 100% of the existing infrastructure. Single-session execution at the lower end of the design-doc's 4–6 hour estimate is realistic.

## Milestones (2)

### M1: Go-side aiCallStream op + `_ai_call_stream` builtin + tests (~3.5 hours, ~270 LOC)

**Files to create:**
- (new section in) `cmd/ailang/configdriven_streaming.go` — `aiCallStream` op (~120 LOC). Registers via `RegisterOp("AI", "callStream", aiCallStream)`. Calls `effects.Call(ctx, "AI", "streamCall", args)` to open the StreamConn, then drives the event loop pulling events off the connection's buffer (using the same machinery `runEventLoop` uses internally), accumulating content deltas via `configdriven.extractPath` against the provider's `streaming.delta_path`. Maps `StreamErrorKind` variants to `AIError` records by parsing the `[code]` prefix from `ProtocolError` messages and matching the named variants (`ConnectionFailed`, `Timeout`, `BudgetExhausted`).
- (new section in) `internal/builtins/ai.go` — `registerAICallStream` (~50 LOC) + `makeAICallStreamType` (`(string, string, string) -> Result[string, AIError] ! {AI, Stream, Net}`) + `aiCallStreamImpl` (delegates via `effects.Call(ctx, "AI", "callStream", args)`).
- (new section in) `cmd/ailang/configdriven_streaming_test.go` — ~150 LOC of integration tests using `httptest.Server` for the SSE mock.

**Acceptance criteria:**
- `_ai_call_stream("test-openai", "gpt-4o", body)` returns `Ok(StringValue{Value: "Hello world"})` after accumulating two delta events on a mock OpenAI server
- Same against a mock Anthropic-shape server
- 401 / 5xx / malformed-JSON / response-path-miss paths return `Err(AIError{...})` with correct `code` (`AuthFailed`, `ConnectionFailed`, `ProtocolError`, etc.) and `retryable` boolean
- `streaming.enabled = false` provider returns `Err(AIError{code: "CapabilityNotSupported"})` (parsed from `[CapabilityNotSupported]` prefix in the underlying ProtocolError message)
- Provider not registered returns `Err(AIError{code: "ProviderNotFound"})`
- Reasoning fields (`reasoning_content`, `thinking`) present in delta events are READ but NOT accumulated — only visible `content` deltas appear in the returned string. Verified by a test asserting the accumulated string equals just the content fields (not concatenated content + reasoning)
- Trace span snapshot test: exactly **one** `AI/streamCall` span emitted per `_ai_call_stream` invocation (the accumulator does NOT emit a separate span)
- The `[DONE]` sentinel terminates the loop cleanly (no spurious deltas after it)
- Anthropic-shape termination (event types like `message_stop`) also terminates cleanly when no `done_sentinel` is configured
- All existing M-AI-PROVIDER-CONFIG + M-AI-STREAMING-HELPER tests still pass (no regression)
- `make ci` passes

### M2: AILANG-side wrapper + docs + motoko-migration update (~1.5 hours, ~120 LOC)

**Files to modify:**
- `std/ai/streaming.ail` — add `callStream(provider, model, messagesJson) -> Result[string, AIError] ! {AI, Stream, Net}` export (~25 LOC including docstring). File grows from 149 LOC to ~175 LOC; the design doc explicitly raises the soft cap to 200 LOC for v0.15.1
- `internal/pipeline/testdata/builtin_types.golden` — regenerate via `UPDATE_GOLDEN=1` to add `_ai_call_stream` line
- `docs/docs/recipes/ai-token-streaming.md` — replace the inline `runStreamCall` user-written-helper section with a "Quick start using callStream" section. Demote the manual event-loop pattern to "Advanced control flow" subsection. Add a "When to use which" decision matrix (`call` vs `callStream` vs `openaiCompatStream` + event loop)
- `docs/docs/guides/custom-ai-providers.md` — one-line cross-reference in the Streaming subsection pointing at `callStream`
- `changelogs/v0.10-current.md` — new "[Unreleased] - targeting v0.15.1" section header (above the existing "[v0.15.0] - 2026-05-04") with this milestone's entry. The migration math from the design doc's Motivating Evidence section is the headline justification
- `design_docs/planned/motoko-agent-v0.15.0-migration.md` — update the "API-shape adaptation pattern" section to use `callStream` directly. Revise the migration estimate downward: 4-5 hours → ~2 hours. Update the "Implementation plan" milestone breakdown to match (M3 from "2-3 hours" → "20 minutes")

**Acceptance criteria:**
- `std/ai/streaming.ail` compiles cleanly (verified by synthetic-import test)
- `callStream` is importable from a fresh AILANG file: `import std/ai/streaming (callStream)` resolves
- Recipe page renders cleanly via `npm run build` in `docs/`
- CHANGELOG entry exists under v0.15.1 referencing the design doc + sprint plan
- motoko-agent-v0.15.0-migration.md reflects the new estimate
- `make ci` passes

## Day-by-day breakdown

Single-session sprint; no day breakdown needed. Hour-by-hour:

| Hour | Task |
|------|------|
| 0:00–0:30 | Read existing infrastructure (configdriven_streaming.go, jsonpath.go, builtins/ai.go); confirm `extractPath` works on the delta-path use case |
| 0:30–2:00 | Write `aiCallStream` op + accumulator loop in cmd/ailang/configdriven_streaming.go |
| 2:00–2:30 | Register `_ai_call_stream` builtin in internal/builtins/ai.go |
| 2:30–3:30 | Integration tests: happy path (OpenAI + Anthropic) + error mapping + reasoning discard + trace span snapshot |
| 3:30–4:00 | Add `callStream` AILANG wrapper to std/ai/streaming.ail; regenerate golden snapshot |
| 4:00–4:30 | Update recipe page + custom-ai-providers + CHANGELOG |
| 4:30–5:00 | Update motoko-agent-v0.15.0-migration.md; revise estimate |

Total: ~5 hours.

## Success metrics

- All M1 + M2 acceptance criteria met
- `make ci` passes
- 14 tests from M-AI-STREAMING-HELPER + 95 from M-AI-PROVIDER-CONFIG still green (no regression)
- Net new code: ~120 LOC implementation + ~25 LOC AILANG + ~150 LOC tests + ~80 LOC docs = **~375 LOC total**
- One new builtin (`_ai_call_stream`); zero new effects; zero changes to underlying primitives

## Dependencies and open questions

**All dependencies shipped** (this is purely a layering milestone):
- M-AI-PROVIDER-CONFIG (v0.15.0) ✅
- M-AI-STREAMING-HELPER (v0.15.0) ✅
- `_ai_stream_call` builtin ✅
- `BuildStreamRequest` + `extractPath` Go helpers ✅

**Open questions (resolvable during execution)**:
1. **Where exactly does the accumulator pull events from?** The design doc says "pull events off the connection's event buffer". Need to confirm whether to use the existing `runEventLoop` machinery (with a Go-side OnEvent closure that closes over a `strings.Builder`) or directly drain the `StreamConnection.events` channel. The former is more reusable and less boilerplate; expect to land on it.
2. **Final-event detection edge case**: when `done_sentinel = "[DONE]"` is configured AND an Anthropic-style `message_stop` event also arrives, which terminates first? Both should work. Test both paths.
3. **`Closed` event with non-200 close code**: should it be `Err(AIError{code: "ConnectionFailed"})` or `Ok(accumulated_so_far)` if some content arrived? Lean toward Err — match the design doc's "structured failure" semantics.

## Risks and mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|-----------|
| Accumulator hangs if SSE connection never sends `[DONE]` and never closes | Low | Inherits the existing `Stream.IdleTimeout` and `MaxDuration` from the underlying StreamConn. Tests use 1-2s timeouts |
| `extractPath` returns nil for an event that has no `delta_path` field (e.g. Anthropic's `message_start`) — accumulator should silently skip, not error | Med | Skip events where extraction fails; only error on response-shape regression for the response_path itself |
| Reasoning extraction inadvertently leaks into the accumulator | Low | Test asserts the returned string is content-only; reasoning is never appended even when `reasoning_path` is configured |
| Trace double-span: `aiCallStream` accidentally emits a span in addition to the `streamCall` span from the underlying op | Med | Snapshot test counts `EffectName=AI` events; must be exactly 1 per call. The `aiCallStream` op explicitly does NOT call `RecordAIEffect` |
| Soft LOC cap on std/ai/streaming.ail | Low | Design doc raises the cap to 200 LOC for v0.15.1; M2 expected to land at ~175 LOC |

## Handoff to sprint-executor

This sprint is small, dependency-free, and acceptance-criteria-rich. Ready for sprint-executor immediately.

```
SPRINT_PLAN_PATH: design_docs/planned/v0_15_1/m-ai-call-stream-helper-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-AI-CALL-STREAM-HELPER.json
```
