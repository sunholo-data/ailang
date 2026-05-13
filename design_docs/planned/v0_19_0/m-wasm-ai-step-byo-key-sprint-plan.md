# Sprint Plan: M-WASM-AI-STEP-BYO-KEY (v0.19.0)

**Design doc**: [m-wasm-ai-step-byo-key.md](./m-wasm-ai-step-byo-key.md)
**Target**: v0.19.0
**Estimated**: ~5 days (~40 hours), but at recent velocity (M-EXT-PORTABILITY-GATE shipped ~1500 LOC across 2 sessions yesterday) realistically **2 sessions / ~10 hours of focused work** to ship code+tests+demo.
**Risk level**: **Low** — pure runtime extension, no parser/typechecker/codegen touched, all infrastructure pre-verified
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-13

---

## Discovery (pre-planning)

Verified what's in place vs needs creating:

| Component | State |
|---|---|
| `cmd/wasm/effects.go::WasmAIHandler` interface + struct | EXISTS (line 95). Has `callback js.Value` field today; needs `stepCallback`, `stepWithCacheCallback`, `stepWithStreamCallback` added |
| `WasmAIHandler.Step` / `StepWithCache` / `StepWithStream` | STUBS that return errors (lines 127-143). Replace with JS-callback-backed impls. |
| `awaitJSResult` for sync + Promise returns | EXISTS (lines 17-59). Reuse as-is. |
| `ailangValueToJS` / `jsToAILANGValue` value converters | EXIST (lines 146-274). Handle records, lists, ADTs, closures, Uint8Array. AILANG `Message` / `ToolCall` / `ToolSchema` records auto-convert. |
| `ailangSetAIHandler` JS hook setter (for `ai.call`) | EXISTS (line 312). Three new sister setters mirror this pattern. |
| `cmd/wasm/main.go` global function registration | Need to add 3 new globals: `ailangSetAIStepHandler`, `ailangSetAIStepWithCacheHandler`, `ailangSetAIStepWithStreamHandler`. |
| `internal/effects/ai_step.go` | Server-side `ai.step` impl works (verified yesterday with mcp 0.2.3 cascade test against `openrouter/auto` returning "ok"). WASM bridge calls into the same `WasmAIHandler.Step` via the `EffContext.AI` interface. |
| Existing test patterns | `internal/repl/wasm_effects_test.go` exists for the WASM effect flow. Will mirror its structure for new tests. |

**Velocity calibration**: yesterday's M-EXT-PORTABILITY-GATE + cascade work shipped:
- ~1500 LOC across 2 sessions of focused work
- Multiple cross-repo coordinated commits (ailang-packages 11 republishes + motoko_agent pin bumps + dev/test/prod cloud config updates)
- ~13 commits with full tests + dry-run validations + e2e proof

For **this** sprint (~400 LOC code + ~150 LOC tests + ~370 LOC demo/docs = ~920 LOC total), realistic estimate is **1 session (~5 hours)** for code-and-tests, **+1 session (~3 hours)** for demo + docs + e2e validation = **2 sessions, ~8-10 hours**.

---

## Milestones

### M1 — Step + JS hook (~150 LOC, ~2 hours)

**Goal**: Browser AILANG `ai.step("openrouter/auto", msgs, tools)` returns a real `Result[StepResult, AIError]` via a JS-callback handler that does direct provider fetch.

**Tasks**:

1. **Extend `WasmAIHandler` struct** (`cmd/wasm/effects.go`):
   - Add `stepCallback js.Value` field
   - Existing `callback` field stays (for `ai.call`)

2. **Replace `WasmAIHandler.Step` stub** (lines 127-131):
   - If `!h.stepCallback.Truthy()` → return `*ai.Response = nil, error("no JS step handler — call ailangSetAIStepHandler(fn) first")`
   - Convert `model` to `js.ValueOf(model)`
   - Convert `[]ai.Message` to JS array using helper `messagesToJS(msgs)` — wraps existing `ailangValueToJS` for each message record
   - Convert `[]ai.ToolSchema` to JS array using helper `toolsToJS(tools)`
   - Invoke `h.stepCallback.Invoke(jsModel, jsMsgs, jsTools)`
   - Await via `awaitJSResult` (existing helper)
   - Convert resolved JS value → `*ai.Response` via helper `jsToResponse(v)` — extracts `message`, `tool_calls`, `finish_reason`, `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens` fields

3. **Add `setAIStepHandler` global function** (`cmd/wasm/effects.go` + register in `cmd/wasm/main.go`):
   - Mirrors `setAIHandler` (line 312) shape
   - Stores callback on `replInstance.repl`'s WasmAIHandler.stepCallback
   - Returns `{success: true}` on success

4. **Tests** (`internal/repl/wasm_step_test.go` — NEW; non-WASM build, exercises the `WasmAIHandler.Step` logic via Go):
   - Skip on non-`js && wasm` build OR test the JS-conversion helpers in isolation (the actual `js.Value` calls require a WASM env). Practical approach: refactor `messagesToJS`/`toolsToJS`/`jsToResponse` into pure Go functions that take/return interface{} so they're unit-testable on the host without `syscall/js`. Then a minimal WASM-side smoke is enough.

**Acceptance**:
- [ ] WASM build succeeds: `GOOS=js GOARCH=wasm go build ./cmd/wasm/`
- [ ] WASM binary delta < 50KB (measure via `ls -la cmd/wasm/main.wasm` before/after)
- [ ] Helper functions `messagesToJS`, `toolsToJS`, `jsToResponse` have unit tests covering: empty inputs, single-message, multi-message with tool_calls, error envelope shape
- [ ] Calling Step without registering handler returns proper Go error (not panic)
- [ ] `make test` passes including new tests
- [ ] `make lint` clean

**Risk**: Low. Pattern mirrors existing `Call` impl exactly.

---

### M2 — StepWithCache + JS hook (~100 LOC, ~1.5 hours)

**Goal**: `ai.stepWithCache(model, msgs, tools, cache_breakpoints)` works in browser. Cache hints serialize cleanly to JS so the JS shim can pass them as Anthropic `cache_control` body fields.

**Depends on**: M1 (reuses `messagesToJS`, `toolsToJS`, `jsToResponse` helpers)

**Tasks**:

1. **Replace `WasmAIHandler.StepWithCache` stub** (lines 133-137):
   - Same shape as Step + 4th arg: `cacheBreakpoints []ai.CacheBreakpoint`
   - New helper `cacheBreakpointsToJS([]ai.CacheBreakpoint) interface{}` — converts to JS array of `{position, ttl}` objects
   - Invoke `h.stepWithCacheCallback.Invoke(jsModel, jsMsgs, jsTools, jsCacheBreakpoints)`
   - Same await + response-convert pipeline as M1

2. **Add `setAIStepWithCacheHandler` global function** — same shape as M1's setter

3. **Tests** (extend `wasm_step_test.go`):
   - `cacheBreakpointsToJS` round-trip
   - StepWithCache stub-missing returns proper error
   - Empty cache_breakpoints array case (most common — degrades to plain Step semantics)

**Acceptance**:
- [ ] CacheBreakpoint records round-trip cleanly (each has `position` + `ttl` fields)
- [ ] Empty cache_breakpoints serializes to JS empty array, not null
- [ ] `make test` + `make lint` still clean

**Risk**: Low. Additive on top of M1.

---

### M3 — StepWithStream + JS hook (~150 LOC, ~2.5 hours)

**Goal**: `ai.stepWithStream(model, msgs, tools, cache_breakpoints, on_chunk)` works in browser. The 5th arg `on_chunk: (StreamChunk) -> () ! e` is an AILANG closure — already wrapped as a callable JS function via existing `js.FuncOf` bridge (line 167 of effects.go).

**Depends on**: M1 + M2

**Tasks**:

1. **Replace `WasmAIHandler.StepWithStream` stub** (lines 139-143):
   - 5 args including `onChunk func(ai.StreamChunk)` Go callback
   - Convert the Go callback to a JS-callable wrapper: `js.FuncOf(...)` that maps JS args → AILANG `StreamChunk` ADT → Go callback invocation
   - Pass that wrapped callback as the 5th arg to `h.stepWithStreamCallback.Invoke(...)`
   - JS shim decides what to do with the callback (typically: parse SSE chunks, fire callback per delta)
   - Same await + response-convert pipeline as M1/M2 for the eventual final response

2. **Add `setAIStepWithStreamHandler` global function** — same shape

3. **`StreamChunk` ADT bridge verification**:
   - `StreamChunk = ContentDelta(string) | Usage({...}) | ThinkingDelta(string)`
   - All three variants should serialize via existing `_ctor`/`_fields` convention (TaggedValue → JS object)
   - Verify with a test that constructs each variant and round-trips through `ailangValueToJS`

4. **Tests** (extend `wasm_step_test.go`):
   - Closure-as-JS-function wrapping (mock JS callback that fires onChunk N times — count invocations)
   - StreamChunk variant serialization (ContentDelta, Usage, ThinkingDelta — all three round-trip)

**Acceptance**:
- [ ] AILANG closure passed as `on_chunk` is callable from JS (verify by mocking handler that fires it 3 times — observe 3 invocations on Go side)
- [ ] All three StreamChunk variants round-trip through the JS bridge correctly
- [ ] `make test` + `make lint` still clean

**Risk**: Med — `js.FuncOf` interaction with the SSE-reader pattern is the trickiest part. Mitigation: existing `ai.call` already uses the same pattern for non-streaming; this is just a tighter loop.

---

### M4 — Demo + docs + WASM-build verification (~370 LOC, ~3 hours)

**Goal**: A real "AILANG agent loop in browser, no backend" demo — provable end-to-end against `openrouter/auto` with a localStorage API key.

**Depends on**: M1 + M2 + M3

**Tasks**:

1. **`examples/wasm-step-byo-key/index.html`** (~100 LOC):
   - Loads `ailang.wasm` via `WebAssembly.instantiateStreaming`
   - Prompts for OPENROUTER_API_KEY (stores in localStorage)
   - Wires `ailangSetAIStepHandler(async (model, msgs, tools) => fetch(...))`
   - Wires a tiny `ailangSetEffectHandler("DOM", { setText: ... })` for output (no full std/dom yet — that's M-WASM-CLOUD-MESSAGES territory)
   - Provides a text-input + submit button + output div

2. **`examples/wasm-step-byo-key/agent.ail`** (~80 LOC):
   - Minimal multi-turn loop calling `ai.step("openrouter/auto", msgs, [])`
   - Renders response content via the DOM effect handler
   - Demonstrates the AGENT LOOP runs in browser (not just a one-shot call)

3. **`examples/wasm-step-byo-key/README.md`** (~60 LOC):
   - How to deploy: drop into a static-host directory with the `.wasm` blob
   - CORS expectations: openrouter/auto works direct; Anthropic needs the dangerous-direct-browser-access header; OpenAI rejects entirely
   - Cost expectations: each step ~$0.001-0.005 depending on model

4. **`docs/docs/guides/wasm-ai-step-byo-key.md`** (~130 LOC):
   - Full guide walking through deploying your own
   - The three handler hooks documented
   - JS shim examples for openrouter, anthropic
   - Troubleshooting (CORS, auth, response shape)

5. **WASM build delta verification**:
   - Build `cmd/wasm/main.wasm` BEFORE the change → record size
   - Build AFTER → assert delta < 50KB
   - Document in CHANGELOG entry

6. **CHANGELOG entry** (~50 LOC under `[v0.19.0]`):
   - Three new JS hooks
   - Demo location
   - CORS notes
   - Cross-link to design doc + sister sprints

**Acceptance**:
- [ ] Demo loads in a browser, prompts for API key, accepts user prompt
- [ ] Demo successfully calls openrouter/auto and renders the response
- [ ] WASM binary delta < 50KB measured
- [ ] CHANGELOG entry under [v0.19.0]
- [ ] Docs page `docs/docs/guides/wasm-ai-step-byo-key.md` ships
- [ ] All sprint tests + lint clean

**Risk**: Low for the demo+docs themselves. The validation step (live openrouter call from a real browser page) is medium-risk — depends on local browser/CORS/network being clean.

---

## Day-by-day breakdown

| Session | Milestones | Hours | Deliverable |
|---|---|---|---|
| 1 (am) | M1: Step + JS hook | 2h | New helpers + replaced stub + setter + unit tests |
| 1 (am) | M2: StepWithCache + JS hook | 1.5h | Cache-aware variant + tests |
| 1 (pm) | M3: StepWithStream + JS hook | 2.5h | Streaming variant + closure-bridge verification |
| 2 (am) | M4 part 1: Demo HTML + AILANG | 1.5h | Working demo against openrouter/auto |
| 2 (am) | M4 part 2: Docs + CHANGELOG | 1h | docs/wasm-ai-step-byo-key.md + CHANGELOG entry |
| 2 (pm) | M4 part 3: WASM build verify + e2e | 0.5h | Binary delta < 50KB, e2e screenshot |

**Total: ~9 hours = 2 focused sessions.**

---

## Repo coordination

| Repo | Branch | What lands |
|---|---|---|
| `sunholo-data/ailang` | `dev` | M1 + M2 + M3 + M4 (all in one repo, no cross-repo coordination needed) |

No companion package or external repo work. Self-contained sprint.

---

## Success metrics

- **Browser-side ai.step works**: `examples/wasm-step-byo-key/` demo runs against real openrouter/auto, returns a real StepResult, no backend.
- **Three handlers parity with three Go-side methods**: Step / StepWithCache / StepWithStream all wired to JS.
- **Existing demos unchanged**: `ailangSetAIHandler` (for `ai.call`) keeps working exactly as today (additive change).
- **Binary size discipline**: <50KB delta enforced.
- **Tests**: ≥6 new unit tests covering helpers + handler-missing error paths.
- **Docs**: New guide page + CHANGELOG entry.

---

## Risks

| Risk | Mitigation |
|---|---|
| `messagesToJS` / `jsToResponse` helpers fragile if AILANG `Message` shape changes | Helpers wrap existing `ailangValueToJS` / `jsToAILANGValue` — they ride along with shape changes automatically |
| Streaming closure bridge starves browser event loop | `ai.call` already uses the same pattern in production; same mechanic for streaming |
| OpenAI direct fetch rejected by CORS | Demo uses `openrouter/auto` only; document OpenAI as proxy-required in README |
| WASM binary delta exceeds 50KB | Strict measurement step in M4 catches this; if exceeded, look for unused conversion-helper bloat |
| `js.FuncOf`-wrapped Go callback vs AILANG closure (already wrapped) creates a double-wrap | Check existing line 167 pattern carefully — it already handles AILANG closures cleanly; new code just passes the Go callback the same way |
| Demo HTML page CORS issues against openrouter | Browsers will block if served from `file://`; demo README documents serving via `python -m http.server` |

---

## Files modified

| File | Change | LOC |
|---|---|---|
| `cmd/wasm/effects.go` | extend WasmAIHandler struct + replace 3 stubs + 3 setters + 3 helpers | +250 |
| `cmd/wasm/main.go` | register 3 new global JS functions | +15 |
| `internal/repl/wasm_step_test.go` | NEW unit tests for helpers + missing-handler paths | +150 |
| `examples/wasm-step-byo-key/index.html` | NEW demo HTML + JS shim | +100 |
| `examples/wasm-step-byo-key/agent.ail` | NEW agent loop AILANG | +80 |
| `examples/wasm-step-byo-key/README.md` | NEW deploy guide | +60 |
| `docs/docs/guides/wasm-ai-step-byo-key.md` | NEW full guide | +130 |
| `changelogs/v0.10-current.md` | [v0.19.0] section entry | +50 |
| **Total** | | **~835 LOC** |

(Estimate revised up from ~250 to ~835 to account for tests + demo + docs. Code-only is still ~265 LOC; the bulk is demo + docs.)

---

## Notes for the executor

- **No design doc gaps known** — design doc is comprehensive; design freeze items are confirmed
- **No cross-repo coordination** — single-repo sprint
- **No new dependencies** — pure Go stdlib + existing AILANG infra
- **Pre-verified infrastructure** — every WASM bridge component used here is already shipped and proven
- **Hand-off pattern**: after each milestone, run `make test` + `make lint` + WASM build (`GOOS=js GOARCH=wasm go build ./cmd/wasm/`) before the next
- **CHANGELOG location**: `changelogs/v0.10-current.md` under a new `## [v0.19.0]` section (or `[Unreleased]` if v0.19.0 hasn't been opened yet — check first)
