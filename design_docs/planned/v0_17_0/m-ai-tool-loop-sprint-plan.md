# Sprint Plan: M-AI-TOOL-LOOP

## Summary

Add multi-turn AI tool dispatch to AILANG: `std/ai.step` (one model turn), `std/ai.runTools` (loop driver), and `callResult`/`callJsonResult` Result-returning variants of the existing single-shot calls. **Reuses** the existing `AIError` type from `std/ai/streaming` (shipped v0.15.0). Extends the AI provider interface with `Step`, plumbs tool calls and typed errors through the trace, implements against Claude + Gemini + OpenRouter + OpenAI. Unblocks pure-AILANG agentic workflows, the planned `docparse legal review` workflow in ailang-parse v0.18.0, and lets `motoko_agent` retire its `tool_contract.ail`/`tool_runtime.ail` user-space tool dispatch (~200 LOC of agent-loop boilerplate).

**Sprint ID:** M-AI-TOOL-LOOP
**Target Version:** v0.17.0
**Design Doc:** [design_docs/planned/v0_17_0/m-ai-tool-loop.md](m-ai-tool-loop.md)
**Duration:** 7 working days (~42-52 hours, ~2 calendar weeks if part-time)
**Refreshed:** 2026-05-05 against v0.15.1 codebase — pulled out AIError adoption work (already shipped v0.15.0) and streaming-related milestones (already shipped v0.15.0/v0.15.1)
**Dependencies:** All shipped:
- ✅ M-UNIFIED-AI-PROVIDERS (v0.5.10)
- ✅ M-AI-OPENROUTER (v0.16.x — wired into `ailang run` 2026-05-04)
- ✅ M-AI-PROVIDER-CONFIG (v0.15.0)
- ✅ M-AI-EFFECT-MODES (v0.15.0)
- ✅ M-AI-STREAMING-HELPER (v0.15.0)
- ✅ M-AI-CALL-STREAM-HELPER (v0.15.1) — shipped `AIError` type

**Companion in v0.17.0:**
- [m-external-consumer-dx.md](m-external-consumer-dx.md) — error_codes.json artifact + diagnostics motoko_agent will consume
- [m-bench-motoko-executor.md](m-bench-motoko-executor.md) — adds motoko_agent as a benchmark executor (downstream consumer of this sprint via cleaner agent-loop API)

**Downstream consumers (after v0.17.0 ships):**
- ailang-parse v0.18.0 Part 3 (`docparse legal review` CLI)
- `arniwesth/motoko_agent` (swap custom tool dispatch onto `runTools` — coordinate with [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md))

**Risk Level:** Low-Medium — provider format normalization is the main risk; AIError + streaming coordination is no longer a risk (shipped).

**Open decisions (see design doc § High-Impact Decisions):**
- ⏳ `dispatch: (ToolCall) -> string` callback signature — confirm sufficient for motoko_agent or widen. Resolves in M0.

All other decisions (extend Request/Response, opaque JSON Schema, user-space loop, AIError shape, streaming-as-deferred, provider parity) are recommended-and-defaulted based on the v0.15.x decisions that already shipped. Open to flipping any of them in review, but defaults track the rest of the codebase.

## Current Status Analysis

### Completed Recently (informs velocity + reduces this sprint's scope)

- ✅ **M-AI-CALL-STREAM-HELPER** (v0.15.1, 2026-05-05) — `callStream` synchronous accumulator returning `Result[string, AIError]`. ~555 LOC, ~5 hours wall-clock against 4-6h estimate. **Establishes the AIError type this sprint reuses.**
- ✅ **M-AI-STREAMING-HELPER** (v0.15.0) — lower-level `openaiCompatStream`/`anthropicStream` primitives over registered `[[ai_provider]]` config. ~700 LOC, ~5.5h wall-clock against 6h estimate.
- ✅ **M-AI-PROVIDER-CONFIG** (v0.15.0) — declarative `[[ai_provider]]` TOML blocks; 95 tests passing.
- ✅ **M-AI-EFFECT-MODES** (v0.15.0) — `!{AI[mode=fixed]}` / `!{AI[mode=routeable]}` via M-EFFECT-REFINEMENT.
- ✅ **M-AI-OPENROUTER** (v0.16.x, 2026-05-04) — wired into `ailang run` (commit `67254452`).
- ✅ **M-EVAL-COST-AND-SPEED-BUDGETS** (v0.15.1) — `Task.Budget *CostBudget` plumbed through executor interface.

### Velocity

- **Recent average**: ~150-250 LOC/day for milestone-style work; the recent v0.15.x AI-related milestones consistently landed in or under estimate (5-5.5h actual on 4-6h estimates).
- **Estimated capacity for this sprint**: ~1,500-2,000 LOC (implementation + tests) over 7 working days
- **Confidence**: Medium-high — provider format normalization (Anthropic vs Gemini vs OpenAI tool-use shapes) remains the main risk; the eval harness already does some of this work in Go and we can lift normalization patterns from there. AIError shape risk eliminated (already shipped).

### Remaining from Design Doc

- ⏳ **M0: Confirm dispatch callback signature** against motoko_agent's existing tool runtime (~50 LOC notes, 0.25d)
- ⏳ **M1: Provider interface + Request/Response extension + extend wrapErrAsAIError** (~300 LOC impl + 150 LOC tests = 450 LOC, 0.75d)
- ⏳ **M2: Anthropic adapter `Step`** (~300 LOC impl + 200 LOC tests = 500 LOC, 1d)
- ⏳ **M3: Gemini adapter `Step`** (~300 LOC impl + 200 LOC tests = 500 LOC, 1d)
- ⏳ **M4: OpenAI/OpenRouter `Step` + Ollama tools_not_supported** (~200 LOC impl + 150 LOC tests = 350 LOC, 0.5d)
- ⏳ **M5: `callResult`/`callJsonResult` + `_ai_step` builtin + AILANG types** (~250 LOC impl + 150 LOC tests = 400 LOC, 1d)
- ⏳ **M6: `std/ai.step` and `runTools` AILANG impl + worked example** (~200 LOC impl + 200 LOC tests = 400 LOC, 1d)
- ⏳ **M7: Trace schema + replay + motoko_agent compat swap + docs + release** (~200 LOC + docs + CHANGELOG, 1.5d)

**Total estimate:** ~2,200 LOC across 8 milestones, 7 working days. (Down from 2,950 LOC / 8d in the pre-refresh estimate — savings come from reusing the existing AIError type and dropping the streaming work.)

## Proposed Milestones

### M0: Confirm dispatch callback signature

**Goal:** Lock the `dispatch: (ToolCall) -> string` signature against actual `motoko_agent` usage before any code lands. AIError shape work is no longer in scope (already shipped v0.15.0; this sprint reuses the existing type unchanged).

**Estimated:** ~50 LOC notes; 0.25 day (~2 hours)

**Tasks:**
- Read `arniwesth/motoko_agent/src/core/tool_contract.ail` and `tool_runtime.ail` — extract the callback shape their tool runtime actually uses; confirm `(ToolCall) -> string` covers their needs (per-call timeout? structured tool errors? conversation context?)
- If `dispatch` signature needs widening (e.g. `(ToolCall) -> Result[string, ToolError]` or `(ToolCall, ConversationCtx) -> string`), update the design doc and bump M6 estimate by 0.5d
- Open a coordination message in [motoko-integration-sequence.md](../motoko-integration-sequence.md) status board so arniwesth knows the tool-loop work is starting and what the proposed callback shape is

**Files to read (no edits this milestone):**
- `arniwesth/motoko_agent/src/core/tool_contract.ail`
- `arniwesth/motoko_agent/src/core/tool_runtime.ail`

**Acceptance Criteria:**
- [ ] `dispatch: (ToolCall) -> string` confirmed sufficient for motoko_agent's tool runtime (or signature widened in design doc)
- [ ] motoko-integration-sequence.md status board updated with M-AI-TOOL-LOOP entry

**Risks:**
- motoko_agent's existing dispatch contract needs richer inputs/outputs than `(ToolCall) -> string` — Mitigation: widen the signature here; M6 absorbs the extra implementation cost (~+0.5d)

---

### M1: Provider interface + Request/Response extension

**Goal:** Add `Messages`, `Tools`, `ToolCalls`, `FinishReason` fields to `internal/ai.Request`/`Response`. Add `Step(ctx, *Request) (*Response, error)` method to the `Provider` interface. Extend the existing `wrapErrAsAIError` (currently in `internal/effects/stream.go`) — or factor a shared `internal/ai/errors.go` if call sites multiply — to emit the new codes (`rate_limit`, `context_length`, `schema_validation`, `tools_not_supported`). All existing providers stub `Step` with `AIError{code: "internal", message: "not yet implemented", retryable: false}`; existing `Generate` keeps working unchanged.

**Estimated:** ~300 LOC implementation + ~150 LOC tests = 450 LOC
**Duration:** 0.75 day (~5-6 hours)

**Tasks:**
- Read existing `wrapErrAsAIError` in `internal/effects/stream.go` to confirm shape; decide whether to extend in place or factor to `internal/ai/errors.go` (factor recommended once non-streaming AI calls also use it)
- Define `Message`, `ToolSchema`, `ToolCall` structs in `internal/ai/provider.go`
- Add `Messages`, `Tools` to `Request`; `ToolCalls`, `FinishReason` to `Response`
- Add `Step` to `Provider` interface; stub in `openai/`, `anthropic/`, `gemini/`, `ollama/`, `openrouter/` returning `AIError{Code: "internal", Message: "step not yet implemented", Retryable: false}`
- Add new code constants (`rate_limit`, `context_length`, `schema_validation`, `tools_not_supported`) alongside the existing ones; extend `wrapErrAsAIError` to recognize them
- Unit tests for the extended classifier covering every new code
- Unit tests for the new struct round-trips and the interface contract

**Files to create:**
- `internal/ai/errors.go` (~80 LOC) — IF factoring out of `internal/effects/stream.go`; otherwise extend in place
- `internal/ai/errors_test.go` (~100 LOC) — new-code mapping tests
- `internal/ai/provider_step_test.go` (~100 LOC) — interface contract tests

**Files to modify:**
- `internal/ai/provider.go` (~120 LOC delta) — extend Request/Response, add Step
- `internal/effects/stream.go` (~30 LOC delta) — IF extending in place; otherwise import from new errors.go
- `internal/ai/openai/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/anthropic/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/gemini/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/ollama/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/openrouter/handler.go` (~20 LOC delta) — stub Step

**Acceptance Criteria:**
- [ ] `go build ./...` clean — all providers compile with new interface
- [ ] `go test ./internal/ai/...` and `go test ./internal/effects/...` pass (existing tests unaffected)
- [ ] `errors_test.go` covers every new `AIError.code` value (`rate_limit`, `context_length`, `schema_validation`, `tools_not_supported`)
- [ ] `provider_step_test.go` verifies Step exists on all providers and returns a stub `AIError` (not panic, not nil)
- [ ] AIError shape unchanged from v0.15.0: `{code, message, retryable}` only — no new fields

**Risks:**
- Adding a method to an interface is a breaking change for any external Provider impls. Mitigation: search-grep for external impls first; document in CHANGELOG as "Provider interface extension"
- Mapping table edge cases (e.g. Anthropic's `overloaded_error` 529 — is it retryable?). Mitigation: per-provider override hook; document each override in code comment with provider doc link

---

### M2: Anthropic adapter `Step`

**Goal:** Real `Step` implementation against Anthropic's tool-use format. Translates `Request.Messages` + `Request.Tools` to Anthropic's `messages` + `tools` shape; parses `tool_use` content blocks into `Response.ToolCalls`.

**Estimated:** ~300 LOC implementation + ~200 LOC tests = 500 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Morning: Read Anthropic Messages API tool-use docs; sketch translation table for Message roles and content-block types
- Morning: Implement `(c *Client) Step(ctx, *Request)` in `internal/ai/anthropic/step.go`
- Morning: Translate `Request.Tools` → Anthropic `tools` array; translate `Request.Messages` → Anthropic `messages` array (assistant `tool_use` content blocks, user `tool_result` content blocks)
- Afternoon: Parse Anthropic response: `content[]` may contain text blocks AND `tool_use` blocks; collapse into `Response.Text` + `Response.ToolCalls`
- Afternoon: Map Anthropic `stop_reason` → `Response.FinishReason` (`end_turn` → "stop", `tool_use` → "tool_calls", `max_tokens` → "length")
- Afternoon: HTTP-fixture tests: text-only step, single tool call, multi-tool call (sequential), tool-result feedback, malformed response

**Files to create:**
- `internal/ai/anthropic/step.go` (~250 LOC)
- `internal/ai/anthropic/step_test.go` (~200 LOC)
- `internal/ai/anthropic/testdata/step_*.json` (~6 fixture files)

**Files to modify:**
- `internal/ai/anthropic/handler.go` (~20 LOC delta) — wire Step

**Acceptance Criteria:**
- [ ] `go test ./internal/ai/anthropic/...` passes with ≥80% coverage on `step.go`
- [ ] Manual smoke test: `ailang --provider anthropic` running a 2-turn conversation with one `read_doc` tool call works against the live API
- [ ] All four `FinishReason` values produced by the adapter under controlled fixtures

**Risks:**
- Anthropic content-block model is heterogeneous (text, tool_use, image, etc.) — easy to drop blocks. Mitigation: explicit switch with default-error
- Tool-call IDs are Anthropic-assigned strings; must round-trip exactly to `tool_result` blocks. Mitigation: pass through verbatim, never regenerate

---

### M3: Gemini adapter `Step`

**Goal:** Same as M2 but for Gemini's `functionCall` / `functionResponse` format. Note: Gemini uses snake_case for `function_call` in REST and camelCase in some SDK docs; normalize at the adapter.

**Estimated:** ~300 LOC implementation + ~200 LOC tests = 500 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Morning: Read Gemini function-calling REST docs; sketch translation table for Content roles ("user", "model", "function") and Part types ("text", "functionCall", "functionResponse")
- Morning: Implement `(c *Client) Step(ctx, *Request)` in `internal/ai/gemini/step.go`
- Morning: Translate `Request.Tools` → Gemini `tools[].functionDeclarations[]`; translate `Request.Messages` → Gemini `contents[]` with appropriate `parts[]`
- Afternoon: Parse Gemini response: `candidates[0].content.parts[]` may contain text + `functionCall` parts; collapse into `Response.Text` + `Response.ToolCalls`
- Afternoon: Map Gemini `finishReason` → `Response.FinishReason`
- Afternoon: HTTP-fixture tests mirroring M2

**Files to create:**
- `internal/ai/gemini/step.go` (~250 LOC)
- `internal/ai/gemini/step_test.go` (~200 LOC)
- `internal/ai/gemini/testdata/step_*.json` (~6 fixture files)

**Files to modify:**
- `internal/ai/gemini/handler.go` (~20 LOC delta) — wire Step

**Acceptance Criteria:**
- [ ] `go test ./internal/ai/gemini/...` passes with ≥80% coverage on `step.go`
- [ ] Manual smoke test: `ailang --provider gemini` running a 2-turn conversation with one tool call works against the live API
- [ ] Tool-call IDs are stable across the round-trip (Gemini doesn't assign IDs natively — adapter generates a deterministic ID from the call index per turn)

**Risks:**
- Gemini's lack of native tool-call IDs forces adapter-side ID generation. Mitigation: deterministic `${turn_index}_${call_index}` so replay is stable
- Function-result feedback uses a `function` role that didn't exist in older Gemini API versions. Mitigation: pin to v1beta or later; document version requirement

---

### M4: OpenRouter passthrough + OpenAI/Ollama parity

**Goal:** OpenRouter routes to whatever the underlying model supports; the adapter just passes tools/messages through unchanged (it speaks OpenAI Chat Completions). OpenAI gets a real `Step` (since the format is what OpenRouter passes through anyway). Ollama returns `ErrToolsNotSupported` when tools are present, else falls back to `Generate`.

**Estimated:** ~200 LOC implementation + ~150 LOC tests = 350 LOC
**Duration:** 0.5 day (~3-4 hours)

**Tasks:**
- Morning: Implement `Step` in `openai/` against OpenAI Chat Completions tool-use format
- Morning: Implement `Step` in `openrouter/` as a thin passthrough over `openai/`'s implementation using OpenRouter's HTTP base — OpenRouter already speaks OpenAI Chat Completions
- Morning: Implement Ollama `Step`: if `len(req.Tools) == 0`, call existing `Generate`; otherwise return `AIError{code: "tools_not_supported", retryable: false}`
- Afternoon: Tests covering OpenAI live-shape parsing, OpenRouter passthrough behaviour (verify `Routing` field still works alongside tools), Ollama not-supported path

**Files to create:**
- `internal/ai/openai/step.go` (~100 LOC)
- `internal/ai/openrouter/step.go` (~40 LOC) — thin wrapper over openai's Step using OpenRouter's HTTP base
- `internal/ai/openai/step_test.go` (~120 LOC)
- `internal/ai/openrouter/step_test.go` (~50 LOC)

**Files to modify:**
- `internal/ai/ollama/handler.go` (~40 LOC delta) — Step routing

**Acceptance Criteria:**
- [ ] `go test ./internal/ai/openai/... ./internal/ai/openrouter/... ./internal/ai/ollama/...` passes
- [ ] OpenRouter route to a tool-supporting model (e.g. `anthropic/claude-sonnet-4.5`) executes a tool call end-to-end
- [ ] OpenRouter `Step` correctly composes with the existing `Routing` field added by M-AI-OPENROUTER (a routed-model tool call works)
- [ ] Ollama with tools returns the typed error rather than silently dropping tools

---

### M5: `callResult`/`callJsonResult` + `_ai_step` builtin + new AILANG types

**Goal:** Wire the Go `Provider.Step` to AILANG via a new `_ai_step` builtin. Add `_ai_call_result` and `_ai_call_json_result` builtins (Result-returning variants of the existing single-shot calls). Define new AILANG records (`ToolSchema`, `ToolCall`, `Message`, `StepResult`) in `std/ai.ail` and re-import `AIError` from `std/ai/streaming.ail` (no new error type).

**Estimated:** ~250 LOC implementation + ~150 LOC tests = 400 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Morning: Re-export `AIError` from `std/ai.ail` (`import std/ai/streaming (AIError)`) — single source of truth
- Morning: Define new AILANG record types in `std/ai.ail` — `ToolSchema`, `ToolCall`, `Message`, `StepResult` (definitions only; functions in M6)
- Morning: Add `_ai_call_result(input) -> Result[string, AIError]` builtin — wraps existing `_ai_call` path, catches Go error, converts to `AIError` via the extended `wrapErrAsAIError`
- Morning: Add `_ai_call_json_result(input, schema) -> Result[string, AIError]` builtin — same pattern over `_ai_call_json`
- Afternoon: Add `_ai_step(model, messages, tools) -> Result[StepResult, AIError]` builtin in `internal/builtins/ai_step.go`
- Afternoon: Implement record→Go-struct converters for `Message`, `ToolSchema` and Go-struct→record converters for `StepResult`, `ToolCall` (AIError converter already exists from v0.15.1)
- Afternoon: Hook all three new builtins into the AI effect handler — same dispatch pattern as `_ai_call_json`
- Afternoon: Builtin tests covering type signatures, capability requirement (`AI`), record round-trip, AIError population on simulated failures
- Afternoon: Snapshot regen: `make snapshot` to update stdlib JSON snapshot for MCP

**Files to create:**
- `internal/builtins/ai_step.go` (~180 LOC) — `_ai_step`, `_ai_call_result`, `_ai_call_json_result` builtins + record↔struct converters

**Files to modify:**
- `internal/builtins/ai.go` (~80 LOC delta) — register the three new builtins
- `std/ai.ail` (~50 LOC delta) — re-export `AIError`, add `ToolSchema`/`ToolCall`/`Message`/`StepResult` definitions
- `internal/builtins/ai_test.go` (~150 LOC delta) — builtin tests including AIError population
- `internal/pipeline/testdata/builtin_types.golden` (regenerated)
- `internal/stdlib/snapshots/std_ai.json` (regenerated)

**Acceptance Criteria:**
- [ ] `_ai_step` callable from AILANG; round-trips a single `Message`/`ToolCall`/`StepResult`
- [ ] `_ai_call_result` and `_ai_call_json_result` callable from AILANG; happy path returns `Ok(string)`, simulated provider errors return `Err(AIError{...})` with the correct `code`/`retryable`
- [ ] All three builtin signatures include `! {AI}` effect; calling without capability fails at runtime with the standard capability error
- [ ] `AIError` reused unchanged from v0.15.0 — `std/ai.AIError` is literally the same type as `std/ai/streaming.AIError`
- [ ] `make snapshot` regenerates stdlib JSON cleanly; MCP `stdlib_module std/ai` returns all new types

**Risks:**
- Record-to-struct conversion for nested types (`Message.tool_calls: list[ToolCall]`) is tedious. Mitigation: lift the pattern from existing `callJson` schema-string handling; add a converter helper if it shows up >2 places
- Re-exporting AIError across two stdlib modules — confirm AILANG's import-resolver handles this cleanly without producing a "duplicate type" warning. Mitigation: prior art exists in other stdlib modules; if resolver complains, move AIError canonical home to `std/ai.ail` and have `std/ai/streaming.ail` re-import (cleaner anyway)

---

### M6: `std/ai.step` and `runTools` AILANG impl

**Goal:** Author the user-facing functions `step` and `runTools` in AILANG on top of the `_ai_step` builtin. Ship the worked example from the design doc as `examples/ai_tool_loop.ail`.

**Estimated:** ~150 LOC implementation + ~150 LOC tests = 300 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Morning: Implement `step` (thin wrapper) and `runTools` (loop driver) in `std/ai.ail`
- Morning: Loop driver pseudo-code:
  ```
  let rec loop(msgs, budget) =
    if budget <= 0
    then Err({message: "step budget exhausted", provider: "", statusCode: 0,
              retryable: false, code: "internal"})
    else match step(model, msgs, tools) {
      Ok(res) =>
        if res.finish_reason == "tool_calls"
          then loop(append(msgs, [res.message] ++ map(tc -> tool_msg(tc, dispatch(tc)), res.tool_calls)), budget - 1)
          else Ok(append(msgs, [res.message])),
      Err(e) => Err(e)   -- typed AIError propagates verbatim
    }
  ```
- Morning: Author `callResult`/`callJsonResult` AILANG wrappers over the M5 builtins
- Afternoon: Author `examples/ai_tool_loop.ail` from the design-doc worked example, demonstrating typed-error retry decision
- Afternoon: AILANG-side tests for the loop driver: terminates on no tool_calls, dispatches sequentially, propagates AIError, respects step budget, retryable-vs-fatal demonstration
- Afternoon: Update `prompts/v0.17.0.md` (or wherever the latest teaching prompt lives) with the AI tool-loop section AND the AIError typed-error section

**Files to create:**
- `examples/ai_tool_loop.ail` (~70 LOC) — includes typed-error retry-decision branch
- `tests/std/ai_tool_loop_test.ail` (~200 LOC)

**Files to modify:**
- `std/ai.ail` (~150 LOC delta) — `callResult`, `callJsonResult`, `step`, `runTools` definitions
- `prompts/<latest>.md` (~50 LOC delta) — teaching section for tool loops + AIError

**Acceptance Criteria:**
- [ ] `ailang run --entry main --caps AI,IO --ai gemini-3-flash-preview examples/ai_tool_loop.ail` completes the worked example end-to-end
- [ ] `make verify-examples` passes (new example added to expected outputs)
- [ ] `runTools` returns `Err(AIError{code: "internal", message: "step budget exhausted", ...})` when the loop exceeds the budget
- [ ] `runTools` propagates AIError from `step` verbatim (provider/statusCode/code preserved)
- [ ] Worked example demonstrates the `if e.retryable` branch path
- [ ] `runTools` propagates dispatch errors via the same typed-error path rather than panicking

**Risks:**
- The loop driver uses `let rec` — verify that AILANG's tail-call handling is sufficient for the typical 5-10 step loop (per agent-message-budget conventions). If not, fall back to a bounded `for`-style accumulator (currently expressed as a folded `range`)

---

### M7: Trace schema + replay + motoko_agent compat swap + docs + release

**Goal:** Per-step trace events capture messages-in, tools-advertised, tool_calls-emitted, dispatch-results, tokens, cost, AND any AIError — mirroring the existing `streamCall` span shape so dashboards/telemetry consumers see uniform events across streaming and tool-loop AI calls. Replay reconstructs a `runTools` conversation. Run a real motoko_agent compat swap to validate the API against actual external usage. Update CHANGELOG and design-docs index. Move design doc to `implemented/v0_17_x/`.

**Estimated:** ~200 LOC + docs + CHANGELOG
**Duration:** 1.5 days (~9-12 hours)

**Tasks:**
- Day 1 morning: Extend trace event schema in `internal/trace/events.go` with `ai.step.request` and `ai.step.response` event types (mirroring the existing `AI/streamCall` span shape from M-AI-CALL-STREAM-HELPER); AIError fields piggyback on the response event when present
- Day 1 morning: Wire trace emission from `_ai_step` and `_ai_call_result`/`_ai_call_json_result` builtins: capture redacted message snapshot (truncate large content), tool list, response, tokens, cost, AIError on failure. Reuse the snapshot-test pattern from v0.15.1's `configdriven_callstream_test.go` (no double-span)
- Day 1 afternoon: Replay path: `ailang trace replay <trace_id>` for a `runTools` conversation reconstructs and re-runs the loop deterministically (assuming dispatch is pure); on a failed conversation, replay surfaces the recorded AIError verbatim
- Day 2 morning: **motoko_agent compat swap** — clone `arniwesth/motoko_agent`, identify one tool-loop site in `src/core/tool_runtime.ail`, replace with `std/ai.runTools`, run their existing tests; document the diff as the canonical migration example. Coordinate timing with [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md) — likely lands in the same PR or a follow-up
- Day 2 afternoon: Update [docs/docs/guides/ai-effects.md](docs/docs/guides/ai-effects.md) (or create) with `callResult`/`callJsonResult`, `step`/`runTools` sections; include the worked example AND the motoko_agent migration example. Note that AIError section already lives in [docs/docs/recipes/ai-token-streaming.md](docs/docs/recipes/ai-token-streaming.md) from v0.15.1 — cross-link, don't duplicate
- Day 2 afternoon: CHANGELOG entry under v0.17.0 referencing this design doc, the sprint plan, the M-EXTERNAL-CONSUMER-DX companion, and the motoko-integration-sequence master plan
- Day 2 afternoon: Update [motoko-integration-sequence.md](../motoko-integration-sequence.md) status board to ✅ for M-AI-TOOL-LOOP
- Day 2 afternoon: Move `design_docs/planned/v0_17_0/m-ai-tool-loop.md` and `m-ai-tool-loop-sprint-plan.md` to `design_docs/implemented/v0_17_x/` once release is cut

**Files to modify:**
- `internal/trace/events.go` (~80 LOC delta)
- `internal/builtins/ai_step.go` (~60 LOC delta) — emit trace events for all three new builtins
- `cmd/ailang/trace.go` (~40 LOC delta) — replay path for tool-loop traces
- `docs/docs/guides/ai-effects.md` (~100 LOC delta or new file)
- `CHANGELOG.md` (~20 LOC delta)
- `design_docs/planned/motoko-integration-sequence.md` (~5 LOC delta) — status board update
- Move + Status header update on the two design docs

**Files to create (in motoko_agent, coordinated PR):**
- `examples/ailang-tool-loop-migration.md` (~80 LOC) — diff + commentary of the swap

**Acceptance Criteria:**
- [ ] `ailang trace list --hours 1` shows `ai.step.request`/`ai.step.response` events from a `runTools` invocation
- [ ] `ailang trace replay <id>` reproduces the conversation given the same dispatch callback (deterministic), including any recorded AIError
- [ ] Trace size stays within existing `AILANG_TRACE_MAX_SPANS` budget on a 10-step loop (no truncation rollup needed in normal use)
- [ ] No-double-span snapshot test passes (mirroring v0.15.1's `callStream` test)
- [ ] motoko_agent's existing tests pass after swapping one tool-loop site to `std/ai.runTools`; the diff is committed as the migration example
- [ ] motoko-integration-sequence.md status board updated to ✅ for M-AI-TOOL-LOOP
- [ ] CHANGELOG.md entry references the design doc, sprint plan, M-EXTERNAL-CONSUMER-DX, and motoko-integration-sequence
- [ ] Both docs landed in `design_docs/implemented/v0_17_x/`
- [ ] `make ci` passes (build, test, lint, verify-examples, file-size check)

---

## Cross-cutting acceptance criteria (sprint-level)

- [ ] `make ci` passes at end of M7
- [ ] All four tool-capable providers (Claude, Gemini, OpenAI, OpenRouter) execute a tool call end-to-end against fixtures with correctly-classified `AIError` on simulated failures
- [ ] Ollama returns `AIError{code: "tools_not_supported", retryable: false}` when tools are present
- [ ] Worked example `examples/ai_tool_loop.ail` runs on a live provider (smoke test, manual) and demonstrates the typed-error retry branch
- [ ] Trace replay deterministic for `runTools` conversations
- [ ] AIError reused unchanged from v0.15.0 — no schema migration, no new error type
- [ ] CHANGELOG entry references both this sprint plan and the design doc
- [ ] Both docs moved to `design_docs/implemented/v0_17_x/`
- [ ] **motoko_agent compatibility validated:** at least one tool-loop site swapped to `std/ai.runTools` with passing tests; diff committed as the migration example
- [ ] **motoko-integration-sequence.md status board updated** to ✅ for M-AI-TOOL-LOOP
- [ ] Downstream ailang-parse v0.18.0 Part 3 (`docparse legal review`) is unblocked: spot-check by importing `std/ai` and round-tripping a fake tool call from an ailang-parse test

## Total: 7 working days, ~2,200 LOC

| Milestone | Duration | LOC |
|-----------|----------|-----|
| M0: Confirm dispatch callback signature | 0.25d | 50 |
| M1: Provider interface + Request/Response + extend wrapErrAsAIError | 0.75d | 450 |
| M2: Anthropic Step | 1d | 500 |
| M3: Gemini Step | 1d | 500 |
| M4: OpenAI/OpenRouter Step + Ollama tools_not_supported | 0.5d | 350 |
| M5: `callResult`/`callJsonResult` + `_ai_step` builtin + new types | 1d | 400 |
| M6: AILANG `step`/`runTools` + worked example | 1d | 400 |
| M7: Trace + replay + motoko_agent swap + docs + release | 1.5d | 200 + docs |
| **Total** | **7d** | **~2,200** |

Pre-refresh estimate was 8d / 2,950 LOC — the refresh saves 1d / 750 LOC by reusing the existing v0.15.0 `AIError` type and the existing v0.15.1 `wrapErrAsAIError` classifier instead of building them fresh.

Parallelization opportunities:
- **M2 and M3 are independent** — could run in parallel via Task sub-agents to shave ~1 day off if needed
- **M0 motoko_agent code-read** is a pre-sprint reading task; can complete before formal sprint kickoff
- **M7 motoko_agent compat swap** can run in parallel with the docs/CHANGELOG work on Day 2
