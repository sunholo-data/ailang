# M-COG-RUNTIME-BROWSER: Cognitive OS Browser Substrate

**Status**: Planned (follow-up to M-COG-RUNTIME Go-side substrate)
**Target**: v0.21.x (continuation)
**Priority**: P0
**Estimated**: ~50–70h across 2 weeks
**Parent**: [M-WASM-REFLECTIVE-RUNTIME (Cognitive OS umbrella)](./m-wasm-reflective-runtime.md)
**Predecessor**: [M-COG-RUNTIME (Go-side substrate, shipped)](./m-cog-runtime.md)
**Dependencies**:
- M-COG-RUNTIME M1–M3 Go-side substrate (✅ shipped: 13 commits, ~6,765 LOC on `dev`)
- WasmDOMHandler + WasmMsgHandler JS bridges ([`cmd/wasm/effects.go`](../../../cmd/wasm/effects.go))
- Scheduler, Replay, EventLog primitives ([`internal/cognition/`](../../../internal/cognition/))
- Existing WASM REPL assets ([`docs/static/wasm/`](../../../docs/static/wasm/))

---

## Scope

M-COG-RUNTIME shipped the **Go-side substrate** for the Cognitive OS (effects, handlers, builtins, stdlib, WASM bridges, Lamport clock, cognitive event log, transport trait, deterministic scheduler, JSONL replay). End-to-end functional on the native side. This follow-up ships the **browser-side JavaScript host** that lights up the WASM bridges and makes the runtime actually reflective in a real browser.

**What this doc covers:**
- Browser-side `host.js` — receives `host.dom.patch` / `host.msg.send` / `host.msg.recv` callbacks from WASM, applies them to the real DOM / BroadcastChannel
- Canonical DOM layer (`canonical_dom.js`) — content-hashed node IDs, banishment of `Date.now()` / `Math.random()` / browser-supplied IDs, deterministic layout
- JS-side replay engine (`replay.js`) — loads a JSONL event log, feeds the host scheduler, asserts DOM equality across runs
- IndexedDB persistence sink (`event_log_indexeddb.js`) — implements the Go-side `Sink` interface from JS so the cognitive event log survives tab restarts
- JS-side scheduler (`scheduler.js`) — microtask-based event loop that mirrors the Go-side Scheduler's `(Clock, Sender)` ordering for browser-only event sources (DOM events, BroadcastChannel arrivals)
- Subscribe op wiring — `_dom_subscribe` and `_msg_subscribe` AILANG-callable builtins that bridge to `FnCaller` for re-entering AILANG closures from event-handler goroutines
- Cognitive Trace extension — extends the existing `Trace` effect to emit `TraceCapturedEvent` records into the cognitive event log

**What this doc explicitly does NOT cover:**
- `!: SharedMem` / `!: SemanticSearch` (→ [M-COG-MEMORY](../v0_22_0/m-cog-memory.md))
- WebSocket / FirestoreRelay / cross-device transports (→ [M-COG-MESH](../v0_22_0/m-cog-mesh.md))
- The collaborative 4-agent demo (→ M-COG-MESH)
- Vector clocks for distributed ordering (→ M-COG-MESH)

**Shippable deliverable:** at completion, an AILANG program with `!: {DOM, Msg}` runs in a browser tab, mutates a real DOM region deterministically, sends messages to another tab via BroadcastChannel, persists its cognitive event log to IndexedDB across page reloads, and replays byte-identically when the log is re-loaded.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Canonical DOM IDs + deterministic JS scheduler + Lamport ordering preserved |
| A2: Replayability | +2 | Cognitive event log + replay engine completes the round-trip |
| A3: Effect Legibility | +1 | No new effect labels (uses DOM/Msg/Trace from M-COG-RUNTIME) |
| A4: Explicit Authority | +2 | Browser host enforces scoped-region DOM authority; capability manifest gates JS bridges |
| A5: Bounded Verification | +1 | Replay byte-equivalence test verifies the DOM-as-replayable claim |
| A6: Safe Concurrency | +2 | JS scheduler microtask-only (no setTimeout); preserves single-threaded dispatch |
| A7: Machines First | +2 | Designed for agent cognition, not human UX; canonical layout banishes time-deps |
| A8: Minimal Syntax | 0 | No new syntax; pure runtime addition |
| A9: Cost Visibility | +1 | Subscribe budgets recorded in event log on overrun |
| A10: Composability | +2 | Browser host plugs into the existing Transport trait; M-COG-MESH transports drop in |
| A11: Structured Failure | +1 | DOM/Msg failures route through existing Result-variant ops |
| A12: System Boundary | +2 | Browser host is the most explicit boundary in the whole runtime |

**Net: +18** → ✅ Proceed

### Hard Violation Check
- [x] A1, A3, A4, A7 all ≥ 0

---

## Problem Statement

M-COG-RUNTIME shipped the Go-side substrate but the browser-side JS that the WASM bridges call back into doesn't exist yet. Without it:

- `WasmDOMHandler.ApplyPatch` returns `no_handler` because no JS callback is registered
- The deterministic event log can't persist across tab restarts (no IndexedDB sink wired)
- The "deterministic distributed replay" property is only proven in-process (Go round-trip test) — not yet across a real browser refresh
- Subscribe ops are stubs that return `not_implemented` — they need the JS event loop + `FnCaller` bridge to actually deliver DOM events / message arrivals to AILANG closures

This is the **last mile** that takes the Cognitive OS from "shippable Go primitives" to "agent reflectivity in a real browser."

---

## Goals

**Primary:** Deliver the browser-side substrate so the full Cognitive OS pipeline runs end-to-end in a browser tab with deterministic replay across page reloads.

**Success Metrics:**
- AILANG program with `!: DOM` in a browser tab mutates a scoped region; same program + same event log produces byte-identical DOM
- AILANG program with `!: Msg` sends to a sibling tab via BroadcastChannel; the receiver's event log records the arrival with sender's Lamport clock
- IndexedDB persists the cognitive event log across page reloads; second-tab replay reconstructs the first tab's session
- Subscribe op (`_dom_subscribe`, `_msg_subscribe`) delivers events to an AILANG closure via FnCaller without blocking the scheduler
- `examples/cognitive_os/single_agent_replay.ail` runs in browser AND native (with appropriate host configuration)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Browser-side JS location (`docs/static/wasm/cognitive-runtime/` vs new `web/runtime/`) | Test harness + Docusaurus integration | human | design | high |
| FnCaller-from-goroutine safety model (queue + drain vs direct invoke) | Deadlock + determinism risk | human | design | high |
| IndexedDB schema versioning + migration | Forward-compat across sprint boundary | human | design | high |
| Canonical DOM ID strategy (content-hash vs path-from-region-root) | Replay determinism + diff size | human | design | high |
| JS test harness choice (Playwright / Puppeteer / WPT / none for v1) | Test reproducibility | human | design | med |
| BroadcastChannel-tab-pair test harness shape | Cross-tab CI coverage | human | design | med |
| Subscribe cancel-token format (opaque int handle vs js.Func unref) | Memory leak prevention | agent | compile | low |

### Design Freeze (recommendations)

- [ ] Browser JS location = **`docs/static/wasm/cognitive-runtime/`** (sibling to existing WASM assets; Docusaurus picks it up automatically)
- [ ] FnCaller-from-goroutine = **queue + drain on AILANG-callable `_cog_drain()` op** (caller controls when the evaluator goroutine processes pending callbacks — preserves single-threaded determinism)
- [ ] IndexedDB schema = **single object store `cognitive_events`, primary key = monotonic counter**, schema version field at the database level for future migrations
- [ ] Canonical DOM IDs = **content-hash of (region, ctor, fields, parent-content-hash)** — deterministic across runs, no ambient identity
- [ ] JS test harness = **Playwright for v1** (mature, headless, reproducible across CI)

---

## Conflict Surface

Touches the browser-side WASM REPL assets ([`docs/static/wasm/`](../../../docs/static/wasm/)), [`cmd/wasm/effects.go`](../../../cmd/wasm/effects.go) (Subscribe wiring), and [`internal/builtins/`](../../../internal/builtins/) (`_cog_drain` op + Subscribe builtins).

### Positions extended

1. **`cmd/wasm/effects.go`** — `WasmDOMHandler.Subscribe` and `WasmMsgHandler.Subscribe` switch from `not_implemented` to a real implementation using JS callbacks + a cancel registry
2. **AILANG-callable builtins** — `_dom_subscribe`, `_msg_subscribe`, `_cog_drain` join the existing 8 Cognitive OS builtins
3. **Browser DOM** — agents mutate real DOM elements (currently they don't — the WASM bridge errors out)
4. **IndexedDB** — new origin-scoped object store `cognitive_events`
5. **Trace effect** — extended to emit `TraceCapturedEvent` records into the cognitive event log (currently the M2 `Trace` plumbing just records to in-memory trace.Collector)

### Existing constructs in these positions

- WasmDOMHandler / WasmMsgHandler are already wired with stubs; this just replaces the stub returns
- Builtin registry already has 8 Cognitive OS builtins; 3 more is additive
- Browser DOM has no AILANG-controlled regions today; new regions don't conflict with anything
- IndexedDB has no AILANG-owned stores today
- Trace effect already exists; extension is additive (existing trace.Collector behavior unchanged)

### Programs that MUST still work
1. All M-COG-RUNTIME programs (Go-side substrate tests)
2. The full `internal/cognition/` test suite (140+ tests)
3. All existing WASM examples in [`examples/wasm_step_byo_key/`](../../../examples/wasm_step_byo_key/) etc.
4. The native AILANG demo [`examples/cognitive_os/single_agent_replay.ail`](../../../examples/cognitive_os/single_agent_replay.ail) — must continue type-checking and running with `NativeMsgHandler`
5. The existing trace.Collector emission to `--emit-trace jsonl`

### Deliberately changed

- `WasmDOMHandler.Subscribe` and `WasmMsgHandler.Subscribe` start returning real cancel functions instead of typed `not_implemented` errors
- `Trace` effect ops gain an additional side-effect (event log append) — the M2 `TraceCapturedEvent` type was defined in advance for this

---

## Solution Design

### Phase 1: Browser Host Skeleton + DOM Bridge (~15h)

`docs/static/wasm/cognitive-runtime/host.js`:
- Registers callbacks for the 4 M1 JS globals (`ailangSetDOMApplyPatchHandler` etc.)
- Implements per-region scoped containers in the DOM: each `RegionID` maps to a `<div data-cog-region="${region}">` element
- Receives the `{ctor, fields}` patch shape from `domPatchToJS` and applies via the canonical DOM layer
- Returns `{node_id, budget_remaining}` to WASM

`canonical_dom.js`:
- Content-hash node IDs: `nodeID = hash(region, ctor, JSON.stringify(fields), parentNodeID)`
- No `Date.now()`, `Math.random()`, or browser-supplied IDs — deterministic across runs
- Removes default browser styling that could affect layout determinism (font fallbacks, viewport-dependent sizing)

### Phase 2: Msg Bridge + BroadcastChannel Wire-up (~10h)

`host.js` extension:
- Registers `ailangSetMsgSendHandler` callback that posts to BroadcastChannel
- Registers `ailangSetMsgRecvHandler` callback that drains a per-mailbox queue populated by BroadcastChannel `onmessage`
- The Go-side `BroadcastChannelTransport` from M2 is the analog; this just wires JS to it via the bridge

### Phase 3: IndexedDB Sink + Replay Engine (~15h)

`event_log_indexeddb.js`:
- Implements the Go-side `Sink` interface from JS: a JS function that mirrors `Sink.Emit(event)` writes each event to IndexedDB transactionally
- Single object store `cognitive_events`, primary key = monotonic counter, secondary index on `(clock, sender)` for canonical-order queries

`replay.js`:
- Loads JSONL from IndexedDB or a file picker
- Feeds events through the JS-side scheduler in canonical order
- Re-applies DOM patches via `canonical_dom.js`
- Asserts DOM equality with a content-hash comparison at the end

### Phase 4: Subscribe Wiring + FnCaller Bridge (~10h)

`internal/effects/dom.go` + `msg.go`:
- Add `subscribe` ops that take an AILANG callback (`eval.Value`) + register it in a per-EffContext pending-callback queue
- New `_cog_drain(timeout_ms)` builtin: AILANG-callable op that drains the queue, invoking each callback via `ctx.FnCaller`

`cmd/wasm/effects.go`:
- `WasmDOMHandler.Subscribe` + `WasmMsgHandler.Subscribe` switch from `not_implemented` to:
  - Register a JS callback for the named region/mailbox
  - The JS callback enqueues arrivals into the per-EffContext pending queue
  - Return a cancel function that unregisters both sides

This is the **canonical pattern** for AILANG closures invoked from async event sources — solves the goroutine-safety problem by serializing through the evaluator's own goroutine.

### Phase 5: Trace Extension + JS Test Harness + Public Demo (~10h)

`internal/effects/trace_cognition.go`:
- Extends the existing `Trace` effect with an op that emits `TraceCapturedEvent` into the cognitive event log
- Bridges existing trace.Collector spans (function_enter / function_exit) to cognitive events for unified replay

Playwright harness:
- Browser-driven test: `npx playwright test` opens 2 tabs, runs an AILANG program that uses `!: {DOM, Msg}`, captures the event log
- Asserts: same event log → same final DOM after refresh + replay
- Asserts: tab-A's `sendMsg` arrives in tab-B's recvMsg loop

**Public demo (mirrors `wasm-step-byo-key` structure):**

Model on the shipped [`wasm-step-byo-key`](https://ailang.sunholo.com/demos/wasm-step-byo-key/) demo (v0.19.0, M-WASM-AI-STEP-BYO-KEY M4):

| File | M-WASM-AI-STEP-BYO-KEY (precedent) | M-COG-RUNTIME-BROWSER (this doc) |
|------|------------------------------------|----------------------------------|
| Live URL | `ailang.sunholo.com/demos/wasm-step-byo-key/` | `ailang.sunholo.com/demos/cognitive-os-runtime/` |
| Examples dir | `examples/wasm_step_byo_key/` | `examples/cognitive_os/` (`single_agent_replay.ail` already shipped) |
| Static-site mirror | `docs/static/demos/wasm-step-byo-key/` | `docs/static/demos/cognitive-os-runtime/` |
| README | Sets BYO-key context, type-check command, `make wasm-serve` flow | Same shape: explains scoped DOM, BroadcastChannel, replay-on-reload |
| AILANG module | `chat.ail` — `ask` / `askCached` / `askStreaming` exports | `single_agent_replay.ail` already exists; expand with subscribe loop + replay export |
| `index.html` | Registers `ailangSetAIStep*Handler`s, wires Anthropic/OpenRouter | Registers `ailangSetDOMApplyPatchHandler` + `ailangSetMsgSendHandler` + `ailangSetMsgRecvHandler`, wires BroadcastChannel + IndexedDB |

The two demos sit side-by-side on the demos page and **layer** rather than compete: BYO-key shows "LLM in browser, no server"; this demo shows "agent UI + messaging in browser, no server". A future combined demo can use both effects in one session (`!: {AI, DOM, Msg}`).

The JS-host playbook is identical between them — registration patterns, singleton wiring, awaitJSResult helpers all reused from `cmd/wasm/effects.go` and `effects_cognition.go`. Zero pattern duplication, only effect-specific JS payloads.

---

## Implementation Plan

| Phase | Days | LOC est | Deliverable |
|-------|------|---------|-------------|
| Phase 1 | 1–3 | ~600 | `host.js` + `canonical_dom.js` + DOM bridge |
| Phase 2 | 4 | ~250 | Msg bridge + BroadcastChannel JS wire-up |
| Phase 3 | 5–7 | ~700 | IndexedDB sink + JS replay engine |
| Phase 4 | 8–9 | ~500 | Subscribe ops + FnCaller drain pattern |
| Phase 5 | 10 | ~400 | Trace extension + Playwright harness |

**Total: ~2,500 LOC across 2 weeks (~10 working days)**

### Files to Modify/Create

**New files (browser-side, all under `docs/static/wasm/cognitive-runtime/`):**
- `host.js` — main WASM↔JS bridge (~500 LOC)
- `canonical_dom.js` — deterministic node IDs + layout (~300 LOC)
- `replay.js` — JSONL replay engine (~250 LOC)
- `event_log_indexeddb.js` — IndexedDB Sink impl (~200 LOC)
- `scheduler.js` — JS-side microtask scheduler (~150 LOC)
- `index.html` — demo page + Playwright test entrypoint (~100 LOC)

**New files (Go-side):**
- `internal/effects/trace_cognition.go` — Trace effect extension (~150 LOC)
- `internal/effects/cog_drain.go` — `_cog_drain` op + per-EffContext callback queue (~200 LOC)
- `internal/builtins/cog_drain.go` — builtin registration (~80 LOC)

**Modified files:**
- `cmd/wasm/effects.go` — replace Subscribe stubs with real impls (~200 LOC)
- `internal/effects/dom.go` — add `subscribe` op via FnCaller bridge (~80 LOC)
- `internal/effects/msg.go` — add `subscribe` op via FnCaller bridge (~80 LOC)
- `internal/cognition/event_log.go` — JS-side Sink interface (~50 LOC if any Go-side adjustments)
- `std/dom.ail` + `std/cognition.ail` — expose subscribe and drain
- `examples/cognitive_os/` — add browser-side demo HTML

**New files (CI):**
- `.github/workflows/cognitive-os-browser.yml` — Playwright in CI (~50 LOC)

---

## Examples

### Example 1: Single-tab DOM mutation with replay

`agent.ail` (compiled to WASM):

```ailang
import std/dom (DOMPatch, AddPanel, UpdateNode, applyPatch)
import std/cognition (sendMsg)

export func mainLoop() -> Unit ! {DOM, Msg, Trace} = {
  let _ = applyPatch("agent_a", AddPanel("Status", "Running..."));
  let _ = sendMsg("logger", "agent_a started");
  ()
}
```

Browser page (`index.html`):

```html
<div data-cog-region="agent_a"></div>
<script>
  // Load WASM REPL, register handlers, load agent module, invoke mainLoop()
  await ailangSetDOMApplyPatchHandler((region, patch) => {
    return CognitiveOS.applyPatch(region, patch);
  });
  // ... etc for the other handlers
  await ailangCall("agent/mainLoop");
</script>
```

Refresh the page → replay engine reads IndexedDB log → reconstructs same DOM byte-identically.

### Example 2: Subscribe to clicks

```ailang
import std/dom (subscribe, DOMEvent, Click)
import std/cognition (drain)

export func clickWatcher() -> Unit ! {DOM} = {
  let cancel = subscribe("agent_a", ["click"], fn (e: DOMEvent) -> Unit ! {DOM} => {
    match e {
      Click(node) => applyPatch("agent_a", UpdateNode(node, "clicked!")),
      _ => ()
    }
  });
  -- Run the event loop for 5 seconds
  drain(5000);
  cancel()
}
```

---

## Success Criteria

- [ ] AILANG program with `!: DOM` mutates a scoped region in a real browser tab
- [ ] Same program + same event log produces byte-identical DOM (verified via content-hash equality)
- [ ] BroadcastChannel send/recv works across 2 browser tabs in the same origin
- [ ] IndexedDB persists the cognitive event log across page reloads
- [ ] Replay engine reconstructs the previous session's DOM after refresh
- [ ] `_dom_subscribe` + `_cog_drain` deliver DOM events to AILANG closures via FnCaller
- [ ] `_msg_subscribe` + `_cog_drain` deliver message arrivals to AILANG closures
- [ ] Trace effect emits `TraceCapturedEvent` records into the cognitive event log
- [ ] Playwright headless test runs the demo + replay end-to-end in CI
- [ ] All M-COG-RUNTIME (Go-side) tests continue to pass
- [ ] Documentation: [`docs/docs/guides/wasm-integration.md`](../../../docs/docs/guides/wasm-integration.md) Cognitive OS section expanded with the live-demo link (the M3-shipped section already exists)
- [ ] **Public demo deployed** at `ailang.sunholo.com/demos/cognitive-os-runtime/` mirroring the `wasm-step-byo-key` shape (sibling files in `examples/cognitive_os/` and `docs/static/demos/cognitive-os-runtime/`)
- [ ] CHANGELOG entry under v0.21.x

---

## Testing Strategy

**Go-side:**
- Subscribe op unit tests via StubDOMHandler / StubMsgHandler
- `_cog_drain` queue invariants (FIFO, FnCaller invocation, cancel cleanup)
- Trace extension: TraceCapturedEvent appears in event log on span-end

**JS-side (Playwright):**
- DOM round-trip: program runs → event log captured → replay → DOM matches
- Cross-tab: tab A sends → tab B receives with sender's Lamport clock preserved
- Persistence: refresh + replay reconstructs prior state
- Canonical DOM: same event log produces byte-identical innerHTML across browser variants (Chromium / Firefox / WebKit)

**Manual:**
- Click around the demo page, verify subscribe callbacks fire and the cognitive event log records each click

---

## Deferred Decisions

- **Playwright vs Puppeteer for CI** — agent choice; Playwright preferred for browser-matrix coverage but Puppeteer is lighter
- **IndexedDB transaction batching strategy** — agent choice; reasonable defaults sufficient for v1
- **JS bundling tool (esbuild / rollup / vite)** — agent choice; whatever the existing WASM REPL uses if any
- **Service worker caching for offline replay** — explicitly deferred to follow-up
- **Cross-origin BroadcastChannel security model** — explicitly deferred to M-COG-MESH (cross-origin needs capability tokens)

---

## Non-Goals

- **WebSocket / FirestoreRelay / WebRTC transports** — those land in M-COG-MESH
- **`!: SharedMem` / `!: SemanticSearch`** — M-COG-MEMORY scope
- **Multi-agent collaborative demo (4-agent topology)** — M-COG-MESH scope
- **Vector clocks** — M-COG-MESH scope
- **Custom React-style components in the canonical DOM layer** — out of scope; the layer is intentionally minimal
- **Service worker / offline-first** — follow-up
- **Cross-origin agent communication** — M-COG-MESH

---

## Timeline

**Week 1** (~25h):
- Days 1–3: Phase 1 (host.js + canonical_dom.js + DOM bridge)
- Day 4: Phase 2 (Msg bridge + BroadcastChannel JS wire-up)

**Week 2** (~25h):
- Days 5–7: Phase 3 (IndexedDB sink + JS replay engine)
- Days 8–9: Phase 4 (Subscribe ops + FnCaller drain)
- Day 10: Phase 5 (Trace extension + Playwright harness) + checkpoint

**Total: ~50h across 2 weeks**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Canonical DOM determinism across browser engines | High | Test matrix Chromium / Firefox / WebKit via Playwright; ban time/random deps |
| IndexedDB schema migration painful later | Med | Version field at DB level + explicit upgrade callback; test forward-migration |
| FnCaller-from-goroutine deadlocks | High | Queue + drain pattern (no direct invoke from event goroutines); test re-entrant cases |
| Subscribe cancel leaks js.Func refs | High | Pair every `js.FuncOf` with a tracked cancel that calls `.Release()`; test cancel-100x |
| BroadcastChannel quirks across browsers | Med | Playwright tests each engine independently; Safari needs explicit testing |
| Playwright CI cost | Med | Run only on tagged commits initially; promote to every-push after stabilization |
| Existing trace.Collector behavior regression | High | Behavior-equivalence test on `--emit-trace jsonl` output pre/post Trace extension |

---

## Strategic Positioning

After M-COG-RUNTIME (Go-side) + M-COG-RUNTIME-BROWSER (this doc), the Cognitive OS substrate is **complete and shippable in a browser**:

- ✅ Effect labels (DOM, Msg, Trace)
- ✅ Step-pattern handlers with stubs + Result variants
- ✅ Capability manifest
- ✅ Lamport clock + cognitive event log
- ✅ Transport trait (LocalWorker + BroadcastChannel)
- ✅ Native bridge (NativeMsgHandler)
- ✅ Deterministic scheduler + JSONL replay
- ⏳ **Browser host + canonical DOM + IndexedDB persistence + Subscribe wiring (this doc)**

After this ships, M-COG-MEMORY adds durable cognitive state (`!: SharedMem`, `!: SemanticSearch`), and M-COG-MESH adds distributed transports (WebSocket / FirestoreRelay) + the 4-agent demo + vector clocks. The strategic claim of the umbrella ("deterministic substrate for distributed self-improving cognition") becomes fully demonstrable.

---

## Related Documents

- **Parent**: [M-WASM-REFLECTIVE-RUNTIME (umbrella)](./m-wasm-reflective-runtime.md)
- **Predecessor**: [M-COG-RUNTIME (Go-side, shipped)](./m-cog-runtime.md)
- **Next**: [M-COG-MEMORY](../v0_22_0/m-cog-memory.md) — durable cognitive state
- **After**: [M-COG-MESH](../v0_22_0/m-cog-mesh.md) — distributed transports + 4-agent demo

**WASM precedent:**
- [m-wasm-ai-step-byo-key](../v0_19_0/m-wasm-ai-step-byo-key.md) — browser-side BYO-key pattern (the JS-host playbook this doc inherits)
- [m-wasm-trace](../../implemented/v0_11_1/m-wasm-trace.md) — JS-side trace handler

## References

- [Design Axioms](/docs/references/axioms)
- [Playwright Docs](https://playwright.dev/) — CI test harness
- [IndexedDB API](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API) — browser persistence
- [BroadcastChannel API](https://developer.mozilla.org/en-US/docs/Web/API/Broadcast_Channel_API) — cross-tab transport

---

**Document created**: 2026-05-19
**Last updated**: 2026-05-19

---

**DESIGN_DOC_PATH**: `design_docs/planned/v0_21_0/m-cog-runtime-browser.md`
