# Sprint Plan: M-AI-TOOL-LOOP

## Summary

Add multi-turn AI tool dispatch to AILANG: `std/ai.step` (one model turn), `std/ai.runTools` (loop driver), `AIError` typed error (lifted from Motoko fork), and `callResult`/`callJsonResult` Result-returning variants of the existing single-shot calls. Extend the AI provider interface with `Step`, plumb tool calls and typed errors through the trace, implement against Claude + Gemini + OpenRouter. Unblocks pure-AILANG agentic workflows, the planned `docparse legal review` workflow in ailang-parse v0.18.0, and harmonizes with the `motoko_agent` harness (currently rolling its own tool dispatch in `tool_contract.ail`/`tool_runtime.ail`).

**Sprint ID:** M-AI-TOOL-LOOP
**Target Version:** v0.17.0
**Design Doc:** [design_docs/planned/v0_17_0/m-ai-tool-loop.md](m-ai-tool-loop.md)
**Duration:** 8 working days (~48-58 hours, ~3 calendar weeks if part-time)
**Dependencies:** None blocking (M-UNIFIED-AI-PROVIDERS shipped v0.5.10; M-AI-OPENROUTER v0.16.0 lands cleanly under us but is not required)
**Downstream consumers:**
- ailang-parse v0.18.0 Part 3 (`docparse legal review` CLI)
- `sunholo-data/motoko_agent` (swap custom tool dispatch onto `runTools`)
- arniwesth/ailang motoko branch (retire `std/ai_motoko` parallel namespace; its `_motoko` files become re-export shims)

**Risk Level:** Medium (provider format normalization + Motoko coordination)

**Awaiting design-freeze decisions (see design doc § High-Impact Decisions):**
- ⏳ Extend existing `Request`/`Response` with `Messages`/`Tools`/`ToolCalls` (recommended) vs new `StepRequest` type
- ⏳ Tool schema as opaque JSON Schema string (recommended) vs typed AILANG ADT
- ⏳ Loop in user-space via `runTools` (recommended) vs baked into the builtin
- ⏳ Provider parity: Claude + Gemini + OpenRouter in this sprint; OpenAI/Ollama return typed error if tools requested
- ⏳ Adopt Motoko `AIError` shape verbatim (recommended) — needs arniwesth sign-off before M0 closes
- ⏳ Streaming via separate `stepStream` follow-up in v0.17.x (recommended) vs in-step chunk callback

This plan assumes the recommended decisions are ratified. If any flips, milestone estimates shift by 0.5-1 day each. The two new Motoko-related decisions are the most consequential — they should be confirmed before M0 begins.

## Current Status Analysis

### Completed Recently (last 14 days, informs velocity)

- ✅ **M-AGENT-MCP** sprint (7/8 milestones, in-flight): server-side filtering, per-module stdlib snapshots, MCP HTTP transport hardening
- ✅ **MCP server-side filtering** for `docs_search`, `example_for_concept`, `stdlib_search`
- ✅ **Per-stdlib-module JSON** for `stdlib_module` MCP tool
- ✅ **OpenRouter provider** sprint plan in flight (M-AI-OPENROUTER v0.16.0)
- ✅ **Verify-examples skip list** for intentionally-failing examples

### Velocity

- **Recent average**: ~150-250 LOC/day for milestone-style work; multi-day milestones land in 1-3 calendar days
- **Estimated capacity for this sprint**: ~1,800-2,200 LOC (implementation + tests) over 6 working days
- **Confidence**: Medium — provider format normalization (Anthropic vs Gemini vs OpenAI tool-use shapes) is fiddly; the eval harness already does some of this work in Go and we can lift normalization patterns from there

### Remaining from Design Doc

- ⏳ **M0: Motoko coordination + AIError adoption** (~50 LOC impl, 0.5d)
- ⏳ **M1: Provider interface + Request/Response extension + AIError + error mapping** (~400 LOC impl + 200 LOC tests = 600 LOC, 1d)
- ⏳ **M2: Anthropic adapter `Step` + AIError mapping** (~350 LOC impl + 250 LOC tests = 600 LOC, 1d)
- ⏳ **M3: Gemini adapter `Step` + AIError mapping** (~350 LOC impl + 250 LOC tests = 600 LOC, 1d)
- ⏳ **M4: OpenRouter + OpenAI/Ollama parity + AIError mapping** (~200 LOC impl + 150 LOC tests = 350 LOC, 0.5d)
- ⏳ **M5: `callResult`/`callJsonResult` + `_ai_step` builtin + AILANG types** (~300 LOC impl + 150 LOC tests = 450 LOC, 1.5d)
- ⏳ **M6: `std/ai.step` and `runTools` AILANG impl** (~150 LOC impl + 200 LOC tests = 350 LOC, 1d)
- ⏳ **M7: Trace schema (incl. AIError) + replay + motoko_agent compat swap + docs + release** (~250 LOC + docs + CHANGELOG, 1.5d)

**Total estimate:** ~2,950 LOC across 8 milestones, 8 working days.

## Proposed Milestones

### M0: Motoko coordination + AIError adoption

**Goal:** Lock the typed-error contract and tool-dispatch callback signature against actual Motoko-side usage before any code lands. Confirm the proposed `AIError` shape matches `std/ai_motoko.AIError` byte-for-byte. Confirm `dispatch: (ToolCall) -> string` is sufficient for `motoko_agent`'s tool-runtime needs. If either needs widening, do it here — not after M1 ships.

**Estimated:** ~50 LOC (mostly notes + a stub re-export in `std/ai_motoko.ail`); 0.5 day (~3-4 hours)

**Tasks:**
- Read [arniwesth/ailang motoko branch `std/ai_motoko.ail`](https://github.com/arniwesth/ailang/blob/motoko/std/ai_motoko.ail) — diff `AIError` shape against the design doc; flag any divergence
- Read `sunholo-data/motoko_agent/src/core/tool_contract.ail` and `tool_runtime.ail` — extract the `(ToolCall) -> string` callback shape they actually use; confirm it covers their needs (per-call timeout? structured tool errors? conversation context?)
- If `dispatch` signature needs widening, propose the change in this doc and bump the M6 estimate
- Open a coordination issue / message to arniwesth confirming AIError verbatim adoption + streaming follow-up plan
- Update `std/ai_motoko.ail` (in our repo's compatibility shim, not the fork) with a one-line stub re-export plan documented for M5 to wire up

**Files to read (no edits this milestone):**
- `arniwesth/ailang/std/ai_motoko.ail` (motoko branch)
- `arniwesth/ailang/internal/effects/ai_motoko.go` (motoko branch) — confirms how Motoko fills the AIError fields
- `sunholo-data/motoko_agent/src/core/tool_contract.ail`
- `sunholo-data/motoko_agent/src/core/tool_runtime.ail`

**Acceptance Criteria:**
- [ ] AIError shape in design doc confirmed verbatim-compatible with `std/ai_motoko.AIError` (or design doc updated to match)
- [ ] `dispatch: (ToolCall) -> string` confirmed sufficient for motoko_agent's tool runtime (or signature widened in design doc)
- [ ] arniwesth has signed off on the AIError adoption + streaming-as-follow-up plan in writing

**Risks:**
- arniwesth disagrees on AIError shape — Mitigation: this milestone is exactly the place to surface that; better to find out before M1
- motoko_agent's existing dispatch contract needs richer inputs/outputs than `(ToolCall) -> string` — Mitigation: widen the signature here; M6 absorbs the extra implementation cost (~+0.5d)

---

### M1: Provider interface + Request/Response extension + AIError type

**Goal:** Add `Messages`, `Tools`, `ToolCalls`, `FinishReason` fields to `internal/ai.Request`/`Response`. Add `Step(ctx, *Request) (*Response, error)` method to the `Provider` interface. Add `AIError` Go struct + the provider-error → AIError mapping table in `internal/ai/errors.go`. All existing providers stub `Step` with `AIError{code: "internal", message: "not yet implemented", retryable: false}`; existing `Generate` keeps working unchanged.

**Estimated:** ~400 LOC implementation + ~200 LOC tests = 600 LOC
**Duration:** 1 day (~6-8 hours)

**Tasks:**
- Morning: Define `AIError` struct in `internal/ai/errors.go` matching the AILANG-side record byte-for-byte (`Provider`, `StatusCode`, `Retryable`, `Code`, `Message`)
- Morning: Define error-code constants (`CodeAuthFailed`, `CodeRateLimit`, `CodeTimeout`, `CodeContextLength`, `CodeToolsNotSupported`, `CodeSchemaValidation`, `CodeTransport`, `CodeModelNotFound`, `CodeInternal`)
- Morning: Implement `func ClassifyHTTPError(provider string, statusCode int, body []byte) *AIError` per the mapping table in the design doc (single source of truth — every adapter calls this rather than rolling its own)
- Morning: Define `Message`, `ToolSchema`, `ToolCall` structs in `internal/ai/provider.go`
- Morning: Add `Messages`, `Tools` to `Request`; `ToolCalls`, `FinishReason` to `Response`
- Afternoon: Add `Step` to `Provider` interface; stub in `openai/`, `anthropic/`, `gemini/`, `ollama/`, `openrouter/` returning `&AIError{Code: CodeInternal, Message: "step not yet implemented", Retryable: false}`
- Afternoon: Unit tests for `ClassifyHTTPError` covering every code in the mapping table
- Afternoon: Unit tests for the new struct round-trips and the interface contract

**Files to create:**
- `internal/ai/errors.go` (~150 LOC) — `AIError` struct, code constants, `ClassifyHTTPError`, conversion to/from Go `error`
- `internal/ai/errors_test.go` (~150 LOC) — mapping-table tests
- `internal/ai/provider_step_test.go` (~100 LOC) — interface contract tests

**Files to modify:**
- `internal/ai/provider.go` (~120 LOC delta) — extend Request/Response, add Step
- `internal/ai/openai/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/anthropic/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/gemini/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/ollama/handler.go` (~20 LOC delta) — stub Step
- `internal/ai/openrouter/handler.go` (~20 LOC delta) — stub Step

**Acceptance Criteria:**
- [ ] `go build ./...` clean — all providers compile with new interface
- [ ] `go test ./internal/ai/...` passes (existing tests unaffected)
- [ ] `errors_test.go` covers every `AIError.Code` value in the design doc's mapping table
- [ ] `provider_step_test.go` verifies Step exists on all providers and returns a stub `AIError` (not panic, not nil)
- [ ] `AIError` Go struct field names + JSON tags match the AILANG record field order (so the round-trip in M5 is mechanical)

**Risks:**
- Adding a method to an interface is a breaking change for any external Provider impls. Mitigation: search-grep for external impls first; document in CHANGELOG as "Provider interface extension"
- Mapping table edge cases (e.g. Anthropic's `overloaded_error` 529 — is it retryable?). Mitigation: per-provider override hook in `ClassifyHTTPError`; document each override in code comment with provider doc link

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

**Estimated:** ~150 LOC implementation + ~100 LOC tests = 250 LOC
**Duration:** 0.5 day (~3-4 hours)

**Tasks:**
- Morning: Implement `Step` in `openrouter/` and `openai/` against OpenAI Chat Completions tool-use format (very similar — both use the `tools`/`tool_calls` schema)
- Morning: Implement Ollama `Step`: if `len(req.Tools) == 0`, call existing `Generate`; otherwise return `ErrToolsNotSupported`
- Afternoon: Tests covering passthrough behavior on OpenRouter, OpenAI live-shape parsing, Ollama not-supported path

**Files to create:**
- `internal/ai/openai/step.go` (~80 LOC)
- `internal/ai/openrouter/step.go` (~30 LOC) — thin wrapper over openai's Step using OpenRouter's HTTP base
- `internal/ai/openai/step_test.go` (~100 LOC)

**Files to modify:**
- `internal/ai/ollama/handler.go` (~30 LOC delta) — Step routing

**Acceptance Criteria:**
- [ ] `go test ./internal/ai/openai/... ./internal/ai/openrouter/... ./internal/ai/ollama/...` passes
- [ ] OpenRouter route to a tool-supporting model (e.g. `anthropic/claude-sonnet-4.5`) executes a tool call end-to-end
- [ ] Ollama with tools returns the typed error rather than silently dropping tools

---

### M5: `callResult`/`callJsonResult` + `_ai_step` builtin + AILANG types

**Goal:** Wire the Go `Provider.Step` to AILANG via a new `_ai_step` builtin. Add `_ai_call_result` and `_ai_call_json_result` builtins (Result-returning variants of the existing single-shot calls). Define the AILANG-side records (`AIError`, `ToolSchema`, `ToolCall`, `Message`, `StepResult`) with conversion to/from the Go structs.

**Estimated:** ~300 LOC implementation + ~150 LOC tests = 450 LOC
**Duration:** 1.5 days (~9-12 hours)

**Tasks:**
- Day 1 morning: Define AILANG record types in `std/ai.ail` — `AIError`, `ToolSchema`, `ToolCall`, `Message`, `StepResult` (definitions only; functions in M6)
- Day 1 morning: Add `_ai_call_result(input) -> Result[string, AIError]` builtin — wraps the existing `_ai_call` path but catches the Go error and converts to `AIError` via `ClassifyHTTPError` (or similar for non-HTTP errors)
- Day 1 morning: Add `_ai_call_json_result(input, schema) -> Result[string, AIError]` builtin — same pattern over `_ai_call_json`
- Day 1 afternoon: Add `_ai_step(model, messages, tools) -> Result[StepResult, AIError]` builtin in `internal/builtins/ai_step.go`
- Day 1 afternoon: Implement record→Go-struct converters for `Message`, `ToolSchema` and Go-struct→record converters for `StepResult`, `ToolCall`, `AIError`
- Day 2 morning: Hook all three new builtins into the AI effect handler — same dispatch pattern as `_ai_call_json`
- Day 2 morning: Builtin tests covering type signatures, capability requirement (`AI`), record round-trip, AIError population on simulated failures (HTTP fixture for each `code` value)
- Day 2 afternoon: Snapshot regen: `make snapshot` to update stdlib JSON snapshot for MCP

**Files to create:**
- `internal/builtins/ai_step.go` (~200 LOC) — `_ai_step`, `_ai_call_result`, `_ai_call_json_result` builtins + record↔struct converters

**Files to modify:**
- `internal/builtins/ai.go` (~80 LOC delta) — register the three new builtins
- `std/ai.ail` (~80 LOC delta) — record type definitions for `AIError`, `ToolSchema`, `ToolCall`, `Message`, `StepResult`
- `internal/builtins/ai_test.go` (~150 LOC delta) — builtin tests including AIError population
- `internal/stdlib/snapshots/std_ai.json` (regenerated)

**Acceptance Criteria:**
- [ ] `_ai_step` callable from AILANG; round-trips a single `Message`/`ToolCall`/`StepResult`
- [ ] `_ai_call_result` and `_ai_call_json_result` callable from AILANG; happy path returns `Ok(string)`, simulated provider errors return `Err(AIError{...})` with the correct `code`/`retryable`/`statusCode`
- [ ] All three builtin signatures include `! {AI}` effect; calling without capability fails at runtime with the standard capability error
- [ ] `AIError` AILANG record fields match `internal/ai.AIError` Go struct fields exactly (regression-tested via a snapshot)
- [ ] `make snapshot` regenerates stdlib JSON cleanly; MCP `stdlib_module std/ai` returns all new types

**Risks:**
- Record-to-struct conversion for nested types (`Message.tool_calls: list[ToolCall]`) is tedious. Mitigation: lift the pattern from existing `callJson` schema-string handling; add a converter helper if it shows up >2 places
- AIError JSON snapshot test couples Go and AILANG sides — both must update together. Mitigation: a single `make snapshot` regenerates both (already true for stdlib)

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

**Goal:** Per-step trace events capture messages-in, tools-advertised, tool_calls-emitted, dispatch-results, tokens, cost, AND any AIError. Replay reconstructs a `runTools` conversation. Run a real motoko_agent compat swap to validate the API against actual external usage. Update CHANGELOG and design-docs index. Move design doc to `implemented/v0_17_x/`.

**Estimated:** ~250 LOC + docs + CHANGELOG
**Duration:** 1.5 days (~9-12 hours)

**Tasks:**
- Day 1 morning: Extend trace event schema in `internal/trace/events.go` with `ai.step.request`, `ai.step.response`, and `ai.error` event types (the last carrying full `AIError` fields for telemetry/dashboard consumption)
- Day 1 morning: Wire trace emission from `_ai_step` and `_ai_call_result`/`_ai_call_json_result` builtins: capture redacted message snapshot (truncate large content), tool list, response, tokens, cost, AIError on failure
- Day 1 afternoon: Replay path: `ailang trace replay <trace_id>` for a `runTools` conversation reconstructs and re-runs the loop deterministically (assuming dispatch is pure); on a failed conversation, replay surfaces the recorded AIError verbatim
- Day 2 morning: **motoko_agent compat swap** — clone `sunholo-data/motoko_agent`, identify one tool-loop site in `src/core/`, replace its custom `tool_runtime` dispatch with `std/ai.runTools`, run their existing tests; document the diff as the canonical migration example
- Day 2 morning: Coordinate with arniwesth to land the `std/ai_motoko.ail` re-export shim in their fork (separate PR, but coordinated timing)
- Day 2 afternoon: Update [docs/docs/guides/ai-effects.md](docs/docs/guides/ai-effects.md) (or create) with `callResult`/`callJsonResult`, `step`/`runTools`, AIError sections; include the worked example AND the motoko_agent migration example
- Day 2 afternoon: CHANGELOG entry under v0.17.0 referencing this design doc, the sprint plan, the M-EXTERNAL-CONSUMER-DX companion, and the Motoko coordination
- Day 2 afternoon: Move `design_docs/planned/v0_17_0/m-ai-tool-loop.md` and `m-ai-tool-loop-sprint-plan.md` to `design_docs/implemented/v0_17_x/` once release is cut

**Files to modify:**
- `internal/trace/events.go` (~120 LOC delta) — incl. AIError fields
- `internal/builtins/ai_step.go` (~60 LOC delta) — emit trace events for all three new builtins
- `cmd/ailang/trace.go` (~40 LOC delta) — replay path for tool-loop traces
- `docs/docs/guides/ai-effects.md` (~120 LOC delta or new file)
- `CHANGELOG.md` (~25 LOC delta)
- Move + Status header update on the two design docs

**Files to create (in motoko_agent, separate PR):**
- `examples/ailang-tool-loop-migration.md` (~80 LOC) — diff + commentary of the swap

**Acceptance Criteria:**
- [ ] `ailang trace list --hours 1` shows `ai.step.request`/`ai.step.response`/`ai.error` events from a `runTools` invocation
- [ ] `ailang trace replay <id>` reproduces the conversation given the same dispatch callback (deterministic), including any recorded AIError
- [ ] Trace size stays within existing `AILANG_TRACE_MAX_SPANS` budget on a 10-step loop (no truncation rollup needed in normal use)
- [ ] motoko_agent's existing tests pass after swapping one tool-loop site to `std/ai.runTools`; the diff is committed as the migration example
- [ ] arniwesth has reviewed and approved the `std/ai_motoko` shim PR against their fork (or explicitly deferred it to v0.17.1)
- [ ] CHANGELOG.md entry references the design doc, sprint plan, M-EXTERNAL-CONSUMER-DX, and Motoko coordination
- [ ] Both docs landed in `design_docs/implemented/v0_17_x/`
- [ ] `make ci` passes (build, test, lint, verify-examples, file-size check)

---

## Cross-cutting acceptance criteria (sprint-level)

- [ ] `make ci` passes at end of M7
- [ ] All four providers (Claude, Gemini, OpenRouter, OpenAI) execute a tool call end-to-end against fixtures with correctly-classified `AIError` on simulated failures
- [ ] Worked example `examples/ai_tool_loop.ail` runs on a live provider (smoke test, manual) and demonstrates the typed-error retry branch
- [ ] Trace replay deterministic for `runTools` conversations
- [ ] CHANGELOG entry references both this sprint plan and the design doc
- [ ] Both docs moved to `design_docs/implemented/v0_17_x/`
- [ ] **motoko_agent compatibility validated:** at least one tool-loop site swapped to `std/ai.runTools` with passing tests; diff committed as the migration example
- [ ] **Motoko fork harmonized:** `std/ai_motoko.callResult` / `callJsonResult` become re-export shims (in arniwesth's fork, coordinated PR); streaming variants follow in v0.17.x
- [ ] Downstream ailang-parse v0.18.0 Part 3 (`docparse legal review`) is unblocked: spot-check by importing `std/ai` and round-tripping a fake tool call from an ailang-parse test

## Total: 8 working days, ~2,950 LOC

| Milestone | Duration | LOC |
|-----------|----------|-----|
| M0: Motoko coordination + AIError adoption | 0.5d | 50 |
| M1: Provider interface + AIError + error mapping | 1d | 600 |
| M2: Anthropic Step + AIError mapping | 1d | 600 |
| M3: Gemini Step + AIError mapping | 1d | 600 |
| M4: OpenRouter/OpenAI/Ollama parity + AIError | 0.5d | 350 |
| M5: `callResult`/`callJsonResult` + `_ai_step` builtin | 1.5d | 450 |
| M6: AILANG `step`/`runTools` + `callResult`/`callJsonResult` wrappers | 1d | 350 |
| M7: Trace + replay + motoko_agent swap + docs + release | 1.5d | 250 + docs |
| **Total** | **8d** | **~2,950** |

Parallelization opportunities:
- **M2 and M3 are independent** — could run in parallel via Task sub-agents to shave ~1 day off if needed
- **M0 motoko_agent code-read can begin before sprint kickoff** — no AILANG-side changes, just reading two files in another repo and confirming a callback signature; treat as pre-sprint reading rather than a milestone if you prefer
