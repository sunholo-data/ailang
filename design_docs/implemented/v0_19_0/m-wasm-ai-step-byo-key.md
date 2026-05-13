# M-WASM-AI-STEP-BYO-KEY: WASM `ai.step` via direct BYO-key provider fetch

**Status**: Implemented (v0.19.0, 2026-05-13)
**Target**: v0.19.0
**Priority**: P1
**Estimated**: ~5 days (~40 hours)
**Actual**: 1 session, 2 sprint-evaluator rounds (scored 78 → 91/100). Browser smoke-test against real openrouter/auto + ai.call BYO-key regression check still recommended before user-facing announcement of the demo.
**Dependencies**: existing WASM-JS bridge in `cmd/wasm/effects.go` (already shipped); existing AILANG-demo BYO-key pattern (`ailangSetAIHandler` → fetch with localStorage key)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-13
**Source**: 2026-05-13 conversation. The `ai.call` BYO-key pattern is already proven across multiple AILANG demos. This doc extends that proven pattern to the multi-turn `ai.step` (and `stepWithCache` / `stepWithStream`) family. Sister design: M-WASM-AI-STEP-VIA-MESSAGES (message-bus path with higher latency but centralized control). The two are complementary — apps choose at boot time which transport to wire up.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Network calls are wall-clock variable; same as today's `ai.call` |
| A2: Replayability | 0 | Mirror `ai.call`'s replay story (no change) |
| A3: Effect Legibility | +1 | `! {AI}` annotation already exists; new impl backs it explicitly |
| A4: Explicit Authority | +1 | API key from localStorage is explicit; user-set; never ambient |
| A5: Bounded Verification | 0 | Zero type-system change |
| A6: Safe Concurrency | 0 | Single-goroutine WASM model |
| A7: Machines First | +1 | Same agent code runs server- or browser-side without rewrite |
| A8: Minimal Syntax | +1 | Zero new syntax |
| A9: Cost Visibility | +1 | Returned StepResult.{input,output,cache_*}_tokens carry through unchanged; user pays directly |
| A10: Composability | +1 | Stacks with existing `ailangSetAIHandler` pattern (which it generalises) |
| A11: Structured Failure | +1 | Returns `Result[StepResult, AIError]` — same shape as server-side |
| A12: System Boundary | +1 | One explicit network boundary per `ai.step` call; user-controlled provider URL + key |

**Net Score: +7** → **Decision: ✅ Move forward**

### Hard Violation Check
- [x] A1, A3, A4, A7 — none violated

---

## Problem Statement

Today's `cmd/wasm/effects.go::WasmAIHandler.Step` returns a hard error:
```go
return nil, fmt.Errorf("ai.step not supported in WASM environment")
```
Same for `StepWithCache` and `StepWithStream`. Yet `ai.call` (single-shot text→text) works fine in WASM today via `ailangSetAIHandler(fn)` — a JS callback that does `fetch` to whichever provider with the user's localStorage key.

**Gap**: any AILANG agent that uses `ai.step` (i.e., the multi-turn tool-loop) — *which is every meaningful agent including motoko* — can't run in browser at all. The infrastructure to bridge AILANG↔JS for this exists (the `WasmAIHandler` interface, the JS callback registration pattern, the value converters). Only the wire-up is missing.

**Why this is the cheap path**:
- `WasmAIHandler` interface is already defined and instantiated
- The JS-callback bridging pattern is proven by `ai.call`
- AILANG `Message` / `ToolCall` / `ToolSchema` records auto-convert to JS objects via existing `ailangValueToJS` (lines 146-210 of effects.go)
- `*ai.Response` round-trips back via `jsToAILANGValue`
- Several existing AILANG demos already follow the BYO-key model — same deploy story

**Current state:**
- AILANG WASM build has `ai.call` + BYO-key pattern shipped, working in production demos.
- `ai.step` / `stepWithCache` / `stepWithStream` all stub out in WASM.
- Anthropic, OpenRouter, Gemini AI Studio all support direct browser fetch (CORS-friendly with the right header for Anthropic).
- OpenAI does NOT support direct browser CORS — would need a tiny proxy or fall back to OpenRouter route.

**Impact**: 
- Cannot demo any tool-loop agent in browser today.
- Forces every "try AILANG agent in your browser" landing page to either use the limited `ai.call` model (no tools) or run a backend.
- Blocks the headline motoko-in-browser story even with the M-WASM-CLOUD-MESSAGES infrastructure (which gives you a thin-client UI, not in-browser orchestration).

---

## Goals

**Primary Goal:** AILANG-WASM `ai.step` works in browser via a JS-callback handler that does direct provider fetch with user's localStorage API key — same proven model as `ai.call`, extended to multi-turn tool dispatch.

**Success Metrics:**
- Browser AILANG can call `ai.step("openrouter/auto", msgs, tools)` and get a real `Result[StepResult, AIError]` back — verified via real Anthropic + OpenRouter calls.
- WASM binary delta < 50KB (no new dependencies).
- `ai.stepWithCache` works with the same handler interface (cache breakpoints serialize cleanly to JS).
- `ai.stepWithStream` callback (which carries effects) fires per-chunk with proper `StreamChunk` ADTs — leverages existing closure-as-JS-function bridge (line 167 of effects.go).
- Demo `examples/wasm-step-byo-key/` shows a real multi-turn agent loop running entirely in browser, no backend, with browser-safe tools (e.g., `dom.querySelector`, fetch-from-URL).
- Existing `ailangSetAIHandler` (for `ai.call`) keeps working — additive.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| New JS hook name(s): `ailangSetAIStepHandler` vs extend `ailangSetAIHandler` | Public API stability for every existing demo's onboarding shim | human | design | high |
| Whether `stepWithStream` callback runs on the JS event loop or AILANG goroutine | Affects how `js.FuncOf` interacts with the SSE-reader; could starve the browser if wrong | agent | compile | med |
| Tool-call result envelope — pass JSON string or already-decoded record | `ai.step` already accepts/returns JSON-encoded ToolCall.arguments; consistency vs ergonomics | agent | compile | low |
| Streaming Promise vs ReadableStream API on the JS side | Affects how providers' SSE implementations get reused (Anthropic vs OpenAI streaming protocols) | agent | runtime | med |
| Behaviour when handler not registered: error (current `Call` model) vs silent no-op | Aligns with `Call` today (errors); deviation would break consistency | human | design | low |

### Design Freeze
- [ ] **Hook naming** — proposed: `ailangSetAIStepHandler(fn)`, `ailangSetAIStepWithCacheHandler(fn)`, `ailangSetAIStepWithStreamHandler(fn)`. Three hooks for three handlers; mirrors the Go-side `WasmAIHandler` interface 1:1.
- [ ] **Error path** — handler-missing returns `Err({code: "no_handler", message: "...", retryable: false})` not a Go panic. Locked.

---

## Solution Design

### Overview

Replace the three error-returning stubs in `cmd/wasm/effects.go::WasmAIHandler` with implementations that delegate to JS-registered callbacks. Each callback receives JS-converted `[]Message`, `[]ToolSchema`, etc., and returns a `*ai.Response` (sync or via Promise).

### Architecture

```
Browser:
  AILANG code:     match step("openrouter/auto", msgs, tools) { ... }
                              │
                              ▼ via _ai_step builtin
                   internal/effects/ai_step.go
                              │
                              ▼ via ctx.AI.Step(...)
                   WasmAIHandler.Step(model, msgs, tools)
                              │  
                              ▼ converts msgs/tools → JS
                   stepCallback.Invoke(jsModel, jsMsgs, jsTools)
                              │
                              ▼ awaitJSResult (already exists for Promises)
                   JS handler:
                     async (model, msgs, tools) => {
                       const key = localStorage.getItem('ANTHROPIC_API_KEY');
                       const resp = await fetch('https://api.anthropic.com/v1/messages', {
                         method: 'POST',
                         headers: { 'x-api-key': key, ... },
                         body: JSON.stringify({ model, messages: msgs, tools, max_tokens: 4096 })
                       });
                       return await resp.json();   // shape matches ai.Response
                     }
                              │
                              ▼ jsToAILANGValue → AILANG Result
                   AILANG code:                      → Ok(StepResult { ... })
```

### Components

1. **`WasmAIHandler.Step` impl** (~50 LOC in `cmd/wasm/effects.go`):
   - Replaces the error stub
   - Converts `[]ai.Message` → JS array of `{role, content, tool_calls, tool_call_id}` objects via existing `ailangValueToJS`
   - Converts `[]ai.ToolSchema` → JS array of `{name, description, parameters}` objects
   - Invokes `h.stepCallback` (newly stored on the handler)
   - Awaits JS result via `awaitJSResult` (already exists)
   - Converts JS response → `*ai.Response` (parsed StepResult shape)

2. **`WasmAIHandler.StepWithCache` impl** (~30 LOC additional):
   - Same as Step, plus a 4th arg: `[]ai.CacheBreakpoint`
   - Converts cache_breakpoints to JS objects with `{position, ttl}` fields
   - Same callback contract; JS handler decides what to do with cache hints (Anthropic supports them via `cache_control` body field)

3. **`WasmAIHandler.StepWithStream` impl** (~80 LOC):
   - 5 args including `func(ai.StreamChunk)` callback
   - The callback is wrapped as a JS function via `js.FuncOf` (existing pattern at line 167 of effects.go)
   - JS handler receives the wrapped callback and fires it on each parsed SSE chunk
   - `StreamChunk` ADT (ContentDelta / Usage / ThinkingDelta) serializes cleanly via the `_ctor`/`_fields` convention already in `ailangValueToJS`

4. **JS hook setters** (~30 LOC):
   - `ailangSetAIStepHandler(fn)` — stores fn on `WasmAIHandler.stepCallback`
   - `ailangSetAIStepWithCacheHandler(fn)`
   - `ailangSetAIStepWithStreamHandler(fn)`
   - All three follow the same shape as the existing `ailangSetAIHandler`

5. **Demo** (`examples/wasm-step-byo-key/`):
   - `index.html` — loads WASM, prompts for OPENROUTER_API_KEY, wires the three handlers
   - `agent.ail` — a minimal multi-turn loop calling `ai.step("openrouter/auto", ...)` with a browser-safe tool (e.g., a `WebFetch` that does `fetch()`)
   - Demonstrates the loop running entirely in browser, no backend

### Implementation Plan

**Phase 1 — Step + StepWithCache** (~3 days)
- [ ] Day 1: Replace `WasmAIHandler.Step` stub. JS conversion helpers for `[]Message` and `[]ToolSchema` (small wrappers around `ailangValueToJS`). Unit test against a mock JS callback.
- [ ] Day 2: `StepWithCache` (same shape + cache_breakpoints arg). Verify cache hints serialize cleanly.
- [ ] Day 3: JS hook setters. End-to-end test: real Anthropic call from a manual harness (`ailang run --target wasm` if such a thing exists, or a tiny test page).

**Phase 2 — StepWithStream** (~2 days)
- [ ] Day 4: `StepWithStream` impl. The tricky bit is the per-chunk callback — verify `js.FuncOf`-wrapped AILANG closures fire correctly when called from inside a JS streaming reader's `for await` loop.
- [ ] Day 5: SSE parsing in the JS shim (provided as docs example; not bundled into AILANG itself). Demo of streaming response rendering token-by-token to DOM.

### Files to Modify/Create

**New files:**
- `examples/wasm-step-byo-key/index.html` (~100 LOC) — demo page + JS shim
- `examples/wasm-step-byo-key/agent.ail` (~80 LOC) — minimal multi-turn loop
- `examples/wasm-step-byo-key/README.md` (~50 LOC) — deploy guide
- `docs/docs/guides/wasm-ai-step-byo-key.md` (~120 LOC) — full guide

**Modified files:**
- `cmd/wasm/effects.go` (+150 LOC): replace 3 stubs + add 3 hook setters
- `cmd/wasm/main.go` (+15 LOC): register the 3 new global JS functions
- `changelogs/v0.10-current.md` (~50 LOC entry under [v0.19.0])

---

## Examples

### Example 1: Browser-side agent loop with browser-safe tool

```html
<!-- examples/wasm-step-byo-key/index.html -->
<script>
  let key = localStorage.getItem("OPENROUTER_API_KEY")
         ?? prompt("Your OpenRouter API key:");
  localStorage.setItem("OPENROUTER_API_KEY", key);

  ailangSetAIStepHandler(async (model, messages, tools) => {
    const resp = await fetch("https://openrouter.ai/api/v1/chat/completions", {
      method: "POST",
      headers: {
        "Authorization": "Bearer " + key,
        "Content-Type": "application/json",
        "HTTP-Referer": location.origin,
      },
      body: JSON.stringify({ model, messages, tools, max_tokens: 4096 }),
    });
    const data = await resp.json();
    // Map provider response → ai.Response shape
    return {
      message: data.choices[0].message,
      tool_calls: data.choices[0].message.tool_calls ?? [],
      finish_reason: data.choices[0].finish_reason,
      input_tokens: data.usage.prompt_tokens,
      output_tokens: data.usage.completion_tokens,
      cache_read_input_tokens: 0,
      cache_creation_input_tokens: 0,
    };
  });
</script>
```

```ailang
-- agent.ail
import std/ai (step, Message, ToolSchema)
import std/io (println)
import std/dom (innerHTML)

export func main() -> () ! {AI, DOM} = {
  let tools: [ToolSchema] = [
    { name: "WebFetch", description: "Fetch a URL", parameters: "..." }
  ];
  let msgs: [Message] = [
    { role: "user", content: "Fetch https://example.com and summarize",
      tool_calls: [], tool_call_id: "" }
  ];
  match step("openrouter/auto", msgs, tools) {
    Ok(result) => innerHTML("#out", result.message.content),
    Err(e)     => innerHTML("#out", "error: " ++ e.message)
  }
}
```

---

## Success Criteria

- [ ] `ai.step` works end-to-end against `openrouter/auto` from a browser demo with localStorage API key
- [ ] `ai.stepWithCache` works against Anthropic direct (with `cache_control` hints) — verifies cache_breakpoints serialize cleanly
- [ ] `ai.stepWithStream` per-chunk callback fires correctly, ContentDelta + Usage variants both observed
- [ ] WASM binary delta < 50KB (verified via `ls -la bin/ailang.wasm` before/after)
- [ ] Existing `ai.call` BYO-key demos still work (additive, no regression)
- [ ] Demo `examples/wasm-step-byo-key/` deploys to docs site and works on real Anthropic / OpenRouter
- [ ] CORS limitations documented (Anthropic needs `anthropic-dangerous-direct-browser-access: true`; OpenAI not supported direct; OpenRouter works)
- [ ] All Go-side tests pass: `make test`
- [ ] `make lint` clean

---

## Testing Strategy

**Unit tests (Go side):**
- Register a mock JS callback via `ailangSetAIStepHandler`; assert AILANG `[]Message` round-trips through the JS conversion correctly.
- Mock JS callback returns `{message: {...}, tool_calls: [...]}`; assert AILANG `Result[StepResult, AIError]` matches.
- Mock callback throws — assert AILANG sees `Err(AIError)` not Go panic.

**Integration tests:**
- Manual: drive the demo page against a real provider, observe streaming tokens render to DOM.
- Negative test: invalid API key → provider 401 → AILANG sees `Err({code: "auth", ...})`.

---

## Deferred Decisions

- **CORS proxy fallback** — agent may include a tiny optional proxy server (~50 LOC Go) that wraps OpenAI for users who specifically want it. Recommended deferred to a separate sprint.
- **Per-provider response normalization** — agent may either ship a single canonical handler that adapts each provider's response to AILANG's `Response` shape, or document the shape and let each demo write its own adapter. Lean: document only, demos adapt.
- **Streaming protocol details** — Anthropic SSE vs OpenAI SSE differ in framing; the JS shim handles this. Agent may include only Anthropic + OpenRouter examples in the demo.

---

## Non-Goals

- **Server-side proxy for non-CORS providers** — out of scope; recommend OpenRouter or BYO proxy.
- **API key encryption in localStorage** — same risk as every existing BYO-key demo; not changing that posture here.
- **Tool dispatch on server** — this is browser-side only. Hybrid (some tools server, some browser) is M-WASM-AI-STEP-VIA-MESSAGES territory.
- **Centralized cost tracking** — user pays direct; cost flows directly to user's provider account. No AILANG-side billing.
- **Conversation persistence** — browser is responsible for saving conversation state (localStorage / IndexedDB / wherever).

---

## Conflict Surface

**Touches no parser/typechecker code.** Pure runtime extension.

- `cmd/wasm/effects.go` — replaces 3 stubs + adds 3 JS hooks. WASM-only build (`//go:build js && wasm`). Risk: bloats binary. Mitigation: <50KB target.
- `cmd/wasm/main.go` — registers new JS globals. Trivially additive.
- No changes to internal/effects, internal/ai, std/ai, or anything compile-time.

**Programs that MUST still work** (regression fixtures):
1. Existing `ai.call` BYO-key WASM demos — unchanged (`ailangSetAIHandler` still the same).
2. CLI `ai.step` — unchanged (different code path: server-side `ai_step.go`).
3. Server-side AILANG agents (motoko, eval-runner) — unchanged.
4. WASM binary loads in <2s on a slow connection (~7MB current; 50KB delta is noise).

---

## Timeline

**Days 1-3:** Step + StepWithCache implementation + tests
**Days 4-5:** StepWithStream + demo
**Total: ~5 days (~40 hours)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OpenAI CORS rejection blocks key providers | Med | Document OpenRouter as the primary "all-providers-via-one-CORS-friendly-route" path; OpenAI direct via proxy as opt-in |
| Streaming SSE differs across providers | Low | Demo handles Anthropic + OpenRouter; OpenAI deferred |
| User's API key leaks via XSS | Med | Same risk as every existing BYO-key demo; documented; not new |
| `js.FuncOf` callback for streaming starves browser event loop | Med | Already proven in `ai.call` model; same pattern |
| Provider response shapes drift, breaking the conversion | Low | Conversion logic lives in JS shim (user-controllable); not baked into AILANG |

---

## Related Documents

**Sister doc (parallel implementation, complementary)**:
- [M-WASM-AI-STEP-VIA-MESSAGES](./m-wasm-ai-step-via-messages.md) — message-bus-mediated `ai.step`. Higher latency but centralized cost + no CORS issues. Apps choose at boot which transport.

**Companion v0.19.0 work**:
- [M-WASM-CLOUD-MESSAGES](./m-wasm-cloud-messages.md) — browser AILANG over the cloud message bus (different concern: thin-client UI, doesn't need ai.step at all)
- [M-COORDINATOR-INBOX-WILDCARDS](./m-coordinator-inbox-wildcards.md) — companion v0.19.0 cloud-arch sprint

**Builds on:**
- Existing AILANG-WASM `ai.call` BYO-key demos (in production today)
- M-AI-PROMPT-CACHING (v0.18.4) — cache_breakpoints surface
- M-AI-STEP-STREAMING (v0.18.7) — StepWithStream introduction

---

## Future Work

- **M-WASM-AI-STEP-PROVIDER-ADAPTERS** — package per-provider response-normalization adapters as reusable JS shims so demos don't reinvent the wheel
- **M-WASM-OPENAI-PROXY** — tiny optional Cloud Run proxy for OpenAI (since OpenAI rejects direct browser CORS)
- **M-WASM-AI-STEP-CHOICE** — runtime selector that picks BYO-key vs message-bus path per call (e.g., based on cost-tracking config)

---

**Document created**: 2026-05-13
**Last updated**: 2026-05-13
