# Sprint Plan: M-COG-RUNTIME-BROWSER

## Summary

Ship the browser-side substrate that lights up the WASM bridges from M-COG-RUNTIME — JS host, canonical DOM layer, IndexedDB persistence sink, replay engine, Subscribe wiring with FnCaller drain pattern, Trace cognitive-event extension, and a Playwright test harness. Deploy a public demo at [`ailang.sunholo.com/demos/cognitive-os-runtime/`](https://ailang.sunholo.com/demos/cognitive-os-runtime/) mirroring the [`wasm-step-byo-key`](https://ailang.sunholo.com/demos/wasm-step-byo-key/) precedent.

**Duration:** 10–12 working days (~2 weeks elapsed)
**Dependencies:**
- M-COG-RUNTIME Go-side substrate (✅ shipped: 13 sprint commits + 3 cleanup commits, ~6,765 LOC on `dev`, PASS @ 91/100)
- WasmDOMHandler + WasmMsgHandler JS bridges ([`cmd/wasm/effects_cognition.go`](../../../cmd/wasm/effects_cognition.go))
- `internal/cognition/` scheduler, replay, transport, event log, clock, manifest primitives
- Existing WASM REPL assets ([`docs/static/wasm/`](../../../docs/static/wasm/))
- BYO-key playbook precedent ([`docs/static/demos/wasm-step-byo-key/`](../../../docs/static/demos/wasm-step-byo-key/))

**Risk Level:** Medium — JS/Go boundary work is intrinsically trickier than pure-Go work (Subscribe FnCaller drain, canonical DOM determinism across browsers, IndexedDB transaction sequencing). Mitigation: 5 well-bounded phases, each independently testable; Playwright provides browser-matrix coverage.

**Design Doc:** [m-cog-runtime-browser.md](./m-cog-runtime-browser.md)
**Parent (umbrella):** [m-wasm-reflective-runtime.md](./m-wasm-reflective-runtime.md)
**Predecessor (shipped):** [m-cog-runtime.md](../../implemented/v0_21_0/m-cog-runtime.md)
**Target:** v0.21.x continuation

---

## Design Freeze (locked in design doc, do not revisit)

- [x] **Browser JS location** = `docs/static/wasm/cognitive-runtime/` (Docusaurus auto-pickup, same pattern as existing `docs/static/wasm/`)
- [x] **FnCaller-from-goroutine** = queue + drain pattern via `_cog_drain()` builtin (preserves single-threaded determinism)
- [x] **IndexedDB schema** = single object store `cognitive_events`, monotonic primary key, DB-level version field for forward migration
- [x] **Canonical DOM IDs** = content-hash of `(region, ctor, fields, parent-content-hash)` — deterministic across runs
- [x] **JS test harness** = Playwright for browser-matrix coverage (Chromium / Firefox / WebKit)
- [x] **Demo URL structure** = `ailang.sunholo.com/demos/cognitive-os-runtime/` mirroring `wasm-step-byo-key`

---

## Current Status Analysis

### Completed Recently (M-COG-RUNTIME velocity baseline)
- **M1** (5 days): 7 commits, ~3,300 LOC implementation+tests (effects, handlers, builtins, WASM bridges)
- **M2** (5–6 days): 5 commits, ~2,515 LOC (clock, event log, transport, native handler, BroadcastChannel)
- **M3 Go-side** (1 day): 1 commit, ~950 LOC (scheduler, replay, demo, CHANGELOG)
- **Cumulative**: 13 sprint commits across 11 working days, ~6,765 LOC, ~140 tests, PASS @ 91/100

### Velocity
- Recent M-COG-RUNTIME pace: ~600 LOC/day (compressed; high test density)
- This sprint targets ~50h / ~2,500 LOC over 10 days = ~250 LOC/day
- Lower target reflects (a) more JS than Go (less familiar test patterns), (b) browser-side determinism work, (c) Playwright bring-up overhead

### Remaining from Design Doc
- Phase 1 (~15h, ~600 LOC): browser host.js + canonical_dom.js + DOM bridge
- Phase 2 (~10h, ~250 LOC): Msg bridge + BroadcastChannel JS wire-up
- Phase 3 (~15h, ~700 LOC): IndexedDB sink + JS replay engine
- Phase 4 (~10h, ~500 LOC): Subscribe ops + FnCaller drain pattern (Go-side + browser-side)
- Phase 5 (~10h, ~400 LOC): Trace cognitive-event extension + Playwright harness + public demo
- **Total: ~50h, ~2,450 LOC**

---

## Proposed Milestones

### M1: BROWSER_HOST_AND_CANONICAL_DOM
**Goal:** Browser-side JS host that receives `host.dom.patch` callbacks from WASM and applies them to a scoped DOM region via canonical, deterministic node IDs. End-to-end DOM mutation works for a single agent.
**Estimated:** ~400 implementation + ~200 tests = ~600 LOC (Phase 1)
**Duration:** 3 days

**Tasks (day-by-day):**

- **Day 1**: Browser host skeleton (~200 LOC)
  - `docs/static/wasm/cognitive-runtime/host.js` — registers callbacks for the 4 M-COG-RUNTIME JS globals (`ailangSetDOMApplyPatchHandler`, `ailangSetDOMApplyBatchHandler`, `ailangSetMsgSendHandler`, `ailangSetMsgRecvHandler`)
  - Per-region scoped containers: `RegionID` → `<div data-cog-region="${region}">`
  - Decodes the `{ctor, fields}` patch shape (from `domPatchToJS` Go-side)
  - Returns `{node_id, budget_remaining}` to WASM
  - Smoke-test in `index.html` with a hand-crafted patch invocation

- **Day 2**: Canonical DOM layer (~300 LOC)
  - `docs/static/wasm/cognitive-runtime/canonical_dom.js`
  - Content-hash node IDs: `hash(region, ctor, JSON.stringify(fields), parentNodeID)` — pure SHA-256 or BLAKE3, no random/Date.now
  - Patch application: `AddPanel` / `UpdateNode` / `RemoveNode` / `AddTimeline` each have a deterministic DOM-element template
  - Layout determinism: ban font fallbacks (use `font-display: block` or system stack only), inline all styles in `<style data-cog-runtime>`
  - Unit tests via the Playwright harness (stood up in M5) but with synthetic test pages

- **Day 3**: DOM bridge integration + smoke test (~100 LOC)
  - Wire `host.js` → `canonical_dom.js` for the full pipeline
  - Smoke test: hand-load a `.wasm` REPL → call `ailang/applyPatch("agent_a", AddPanel("T", "C"))` from JS → verify the rendered DOM matches the canonical-node-ID prediction
  - Document the wire format in `docs/static/wasm/cognitive-runtime/README.md` (briefly — full docs land in M5)
  - **M1 Checkpoint:** browser tab can load WASM, the AILANG-side `applyPatch` call mutates a scoped DOM region, the rendered HTML matches expected canonical IDs

**Acceptance Criteria:**
- [ ] `host.js` exposes the 4 callback registrations and they're wired to WASM via `ailangSet*Handler` globals
- [ ] `canonical_dom.js` produces deterministic node IDs (same `(region, ctor, fields, parent-hash)` → same ID across reloads)
- [ ] No `Date.now()` / `Math.random()` / browser-supplied auto-IDs in the layout layer
- [ ] Hand-crafted JS test: `applyPatch("r", AddPanel("T", "C"))` mutates the scoped region; same call twice = same DOM
- [ ] All existing M-COG-RUNTIME Go-side tests still pass
- [ ] Linting clean

**Risks:**
- Browser engine layout variance (font defaults, viewport-dependent sizing) — Mitigation: inline styles + system font stack; Playwright matrix test in M5 catches drift
- Content-hash collision risk for nested AddPanel patches — Mitigation: include parent-content-hash + sibling-index in the hash input

---

### M2: MSG_BRIDGE_AND_BROADCASTCHANNEL
**Goal:** Browser-side JS that wires `host.msg.send` and `host.msg.recv` callbacks to the BroadcastChannel Web API. Two tabs exchange messages with sender/receiver-Lamport-clock preservation.
**Estimated:** ~180 implementation + ~70 tests = ~250 LOC (Phase 2)
**Duration:** 1 day

**Tasks (day-by-day):**

- **Day 4**: Msg bridge + BroadcastChannel wire-up (~250 LOC)
  - Extension of `host.js` (existing file from M1): registers `ailangSetMsgSendHandler` callback that posts to `new BroadcastChannel(mailbox)`
  - Registers `ailangSetMsgRecvHandler` callback that pulls from a per-mailbox arrival queue populated by `BroadcastChannel.onmessage`
  - `Recv` returns a `Promise` for async cases (Go-side `awaitJSResult` handles both sync + async)
  - Per-tab agent identity stored in `localStorage` (or auto-generated nanoid if absent) — used as `From` in outgoing envelopes
  - Two-tab smoke test in `index.html`: tab A calls `sendMsg("inbox_b", "ping")` → tab B's recvMsg loop receives `{from: "tab-a-id", payload: "ping", clock: N}`
  - **M2 Checkpoint:** cross-tab message exchange works; sender Lamport clock preserved on receiver side

**Acceptance Criteria:**
- [ ] `host.js` registers `ailangSetMsgSendHandler` + `ailangSetMsgRecvHandler` with BroadcastChannel-backed implementations
- [ ] Two browser tabs (same origin) exchange messages; sender's Lamport clock is preserved in the receiver's recvMsg return value
- [ ] Sender stamp identifies the originating tab uniquely
- [ ] `Recv` supports async (Promise return) for transport types that need to await arrival
- [ ] All M-COG-RUNTIME `TestLocalWorker_*` + `TestNativeMsgHandler_*` tests still pass (no Go-side changes here)
- [ ] Linting clean

**Risks:**
- BroadcastChannel API differences across Safari/Firefox/Chrome — Mitigation: feature-detect + skip the demo with a clear "BroadcastChannel unavailable" message on unsupported engines (M5 Playwright matrix verifies)
- Sender-identity collision when two tabs share a localStorage origin but open independently — Mitigation: nanoid (32 bits min) + `Date.now()` salt at first-load

---

### M3: INDEXEDDB_SINK_AND_REPLAY
**Goal:** Persist the cognitive event log to IndexedDB across page reloads. Replay engine reconstructs the previous session's DOM byte-identically.
**Estimated:** ~500 implementation + ~200 tests = ~700 LOC (Phase 3)
**Duration:** 3 days

**Tasks (day-by-day):**

- **Day 5**: IndexedDB sink (~300 LOC)
  - `docs/static/wasm/cognitive-runtime/event_log_indexeddb.js`
  - Implements the Go-side `Sink` interface from JS: a JS function `emit(event)` writes each event to IndexedDB transactionally
  - Single object store `cognitive_events`, primary key = monotonic counter
  - Secondary index on `(clock, sender)` for canonical-order queries
  - DB-level schema version field with explicit upgrade callback for forward migration
  - Wire into `host.js`: every `host.trace.emit` and side-channel from DOM/Msg writes to the sink

- **Day 6**: JS replay engine (~250 LOC)
  - `docs/static/wasm/cognitive-runtime/replay.js`
  - Loads JSONL from IndexedDB (canonical-order query via secondary index) OR from a file picker
  - Feeds events through the JS-side scheduler (M4) in canonical order
  - Re-applies DOM patches via `canonical_dom.js`
  - Asserts DOM equality via content-hash comparison at the end of replay

- **Day 7**: JS scheduler + replay integration (~150 LOC)
  - `docs/static/wasm/cognitive-runtime/scheduler.js`
  - Microtask-based single-threaded event loop (no `setTimeout`, no `setInterval` — preserves determinism)
  - Mirror the Go-side `Scheduler.Dispatch` ordering: events sorted by `(LamportClock, Sender)` before dispatch
  - Replay test: refresh page → IndexedDB replay engine reconstructs the prior session's DOM byte-identically
  - **M3 Checkpoint:** kill tab mid-session, refresh, observe DOM reconstructed from event log

**Acceptance Criteria:**
- [ ] `event_log_indexeddb.js` writes every cognitive event (MessageSent / MessageReceived / PatchApplied / CapabilityExceeded / TraceCaptured) transactionally to IndexedDB
- [ ] Canonical-order query via `(clock, sender)` secondary index returns events in identical order to the Go-side replay
- [ ] DB-level schema version supports forward migration without data loss
- [ ] `replay.js` reconstructs the prior session's DOM byte-identically after page reload
- [ ] Content-hash equality check verifies replay correctness
- [ ] JS scheduler dispatches in deterministic canonical order
- [ ] Linting clean

**Risks:**
- IndexedDB transaction sequencing under concurrent writers — Mitigation: serialize writes through a single Promise chain in `event_log_indexeddb.js`
- Browser-engine-specific IndexedDB quirks (Safari WAL behavior, Firefox quota policies) — Mitigation: Playwright matrix test in M5 + feature-detect quota with clear error messaging
- IndexedDB schema migration ergonomics for v2+ — Mitigation: explicit version + upgrade callback now; document migration playbook in README

---

### M4: SUBSCRIBE_OPS_AND_FNCALLER_DRAIN
**Goal:** Wire DOM/Msg Subscribe ops through to AILANG closures via a queue + drain pattern (`_cog_drain()` builtin). M-COG-RUNTIME's `not_implemented` stubs become real implementations.
**Estimated:** ~370 implementation + ~130 tests = ~500 LOC (Phase 4, mixed Go + browser-side)
**Duration:** 2 days

**Tasks (day-by-day):**

- **Day 8**: Go-side Subscribe ops + FnCaller drain (~300 LOC)
  - `internal/effects/dom.go` + `msg.go`: add `subscribe` ops that take an AILANG callback (`eval.Value`) and register it in a per-EffContext pending-callback queue
  - `internal/effects/cog_drain.go` (NEW, ~200 LOC) — `_cog_drain(timeout_ms)` builtin: AILANG-callable op that drains the queue, invoking each callback via `ctx.FnCaller`. Returns count of dispatched events.
  - `internal/builtins/cog_drain.go` (NEW, ~80 LOC) — `RegisterEffectBuiltin` spec for `_cog_drain`
  - Update `cmd/wasm/effects_cognition.go`: `WasmDOMHandler.Subscribe` + `WasmMsgHandler.Subscribe` switch from `not_implemented` to real impls that register a JS callback for the named region/mailbox + enqueue arrivals into the per-EffContext queue + return a cancel function
  - Unit tests for the queue mechanics: enqueue under callback, drain dispatches via FnCaller, cancel removes registration
  - Update `std/dom.ail` and `std/cognition.ail` with the new `subscribe` / `drain` AILANG wrappers

- **Day 9**: Browser-side Subscribe wiring (~200 LOC)
  - Extend `host.js`: each Subscribe-call registers a long-lived JS callback that fires on DOM events (click/input) for the named region OR on BroadcastChannel arrivals for the named mailbox
  - Subscribe cancel: tracks every `js.FuncOf` and calls `.Release()` on cancel to prevent leaks
  - Smoke test: AILANG program with `subscribe(region, ["click"], onEvent)` + `drain(5000)` — clicks in the demo HTML trigger AILANG closure invocations
  - **M4 Checkpoint:** Subscribe ops work end-to-end (browser click → AILANG closure invocation via FnCaller drain pattern)

**Acceptance Criteria:**
- [ ] `_cog_drain(timeout_ms)` AILANG-callable builtin lands; AILANG callers control event-loop progression
- [ ] `WasmDOMHandler.Subscribe` and `WasmMsgHandler.Subscribe` return real cancel functions (not `not_implemented` errors)
- [ ] Browser DOM click events flow back into AILANG closures via the drain pattern
- [ ] BroadcastChannel arrivals fire AILANG callbacks registered via `subscribeMsg(mailbox, onMsg)`
- [ ] Cancel function releases the `js.FuncOf` reference (no leaks across 100 subscribe/cancel cycles)
- [ ] All M-COG-RUNTIME tests still pass; new tests cover the drain mechanics
- [ ] Linting clean

**Risks:**
- FnCaller-from-goroutine deadlocks — Mitigation: queue + drain pattern serializes through the evaluator's own goroutine (no direct Invoke from arbitrary goroutines); test re-entrant `subscribe` from inside a drain
- `js.FuncOf` leak risk if cancel paths miss `.Release()` — Mitigation: track every JS callback in a per-handler registry; test cancel-100x under heap-snapshot assertion
- Drain-timeout semantics surprise callers — Mitigation: document explicitly; drain(0) = process pending then return immediately; drain(N) = block up to N ms or until pending is empty, whichever first

---

### M5: TRACE_EXTENSION_PLAYWRIGHT_AND_DEMO
**Goal:** Cognitive-Trace extension emits TraceCapturedEvent into the event log. Playwright headless harness verifies replay-determinism + cross-tab messaging in CI. Public demo deployed.
**Estimated:** ~300 implementation + ~100 tests = ~400 LOC (Phase 5)
**Duration:** 2 days

**Tasks (day-by-day):**

- **Day 10**: Trace extension + Playwright harness (~200 LOC)
  - `internal/effects/trace_cognition.go` (NEW, ~150 LOC) — extends the existing `Trace` effect with an op that emits `TraceCapturedEvent` records into the cognitive event log. Bridges `trace.Collector` spans (function_enter / function_exit) to cognitive events.
  - Behavior-equivalence regression: `--emit-trace jsonl` output pre/post extension must be byte-identical (this is the existing trace.Collector path, untouched)
  - Playwright project setup: `tests/playwright/cognitive-os.spec.ts`
  - Two-tab integration test: open `ailang.sunholo.com/demos/cognitive-os-runtime/` in both tabs, run an AILANG program with `!: {DOM, Msg}`, verify tab-A `sendMsg` arrives in tab-B's recvMsg loop with sender clock preserved
  - DOM-replay determinism test: run program → capture event log → refresh → assert reconstructed DOM matches via content-hash equality
  - Browser matrix: Chromium, Firefox, WebKit

- **Day 11**: Public demo deployment + docs + CHANGELOG (~200 LOC)
  - `examples/cognitive_os/README.md` (NEW, ~80 LOC) — same shape as `examples/wasm_step_byo_key/README.md`: explains scoped DOM, BroadcastChannel, IndexedDB replay-on-reload; `ailang check` command; `make wasm-serve` flow
  - `examples/cognitive_os/index.html` (NEW, ~150 LOC) — JS host page with the same shape as `examples/wasm_step_byo_key/index.html`: registers all 4+ `ailangSet*Handler`s, loads `single_agent_replay.ail` module, exposes "Run" / "Reset" / "Refresh & Replay" buttons + a DOM region for the agent to write to
  - `docs/static/demos/cognitive-os-runtime/` (mirror of `examples/cognitive_os/`) — Docusaurus pick up; published at `ailang.sunholo.com/demos/cognitive-os-runtime/`
  - `examples/cognitive_os/single_agent_replay.ail` (EXPAND existing ~50 LOC → ~100 LOC) — add a subscribe-and-drain loop demonstrating click events flowing back into AILANG
  - Update `docs/docs/guides/wasm-integration.md` Cognitive OS section: replace deferred ⏳ markers with shipped ✅ ones; add live-demo link
  - CHANGELOG entry under v0.21.x for M-COG-RUNTIME-BROWSER
  - **M5 Checkpoint:** Playwright passes in CI; public demo live at `ailang.sunholo.com/demos/cognitive-os-runtime/`

**Acceptance Criteria:**
- [ ] `TraceCapturedEvent` emitted by `Trace` effect ops; appears in the cognitive event log
- [ ] `--emit-trace jsonl` output byte-identical pre/post extension (behavior-equivalence regression test)
- [ ] Playwright test passes in CI across Chromium/Firefox/WebKit
- [ ] Two-tab cross-browser BroadcastChannel demo verified
- [ ] DOM replay byte-equality verified across browser refreshes
- [ ] Public demo deployed at `ailang.sunholo.com/demos/cognitive-os-runtime/`
- [ ] `examples/cognitive_os/` mirrors `examples/wasm_step_byo_key/` shape (README + .ail + index.html)
- [ ] `docs/docs/guides/wasm-integration.md` Cognitive OS section updated with live-demo link + status-table ✅ ↔ ⏳ flips
- [ ] CHANGELOG entry under v0.21.x
- [ ] Linting clean

**Risks:**
- Playwright bring-up time underestimated — Mitigation: Day 11 has explicit buffer; if Playwright matrix is flaky, stage to Chromium-only for v1 with Firefox/WebKit promoted to "best-effort" until issues stabilize
- Demo deployment friction (Docusaurus build, Vercel/Pages config) — Mitigation: follow the `wasm-step-byo-key` deployment exactly; same `docs/static/demos/` path is auto-picked-up
- Trace behavior-equivalence regression — Mitigation: snapshot `--emit-trace jsonl` output pre-change, diff post-change is empty; this is the existing path, just add an additional emit hook

---

## Success Metrics

- **Test coverage:** Playwright passes in CI across Chromium/Firefox/WebKit; ~30 new tests across `internal/effects/cog_drain*`, `internal/effects/trace_cognition*`, Playwright suite
- **Public demo deployed:** `ailang.sunholo.com/demos/cognitive-os-runtime/` accessible with working DOM mutation + cross-tab Msg + replay-on-reload
- **DOM replay byte-equality:** content-hash equality test passes after page refresh
- **Cross-tab Msg with Lamport clock preservation:** verified via Playwright
- **No behavior regressions:**
  - All 140 M-COG-RUNTIME Go-side tests pass
  - `--emit-trace jsonl` output byte-identical pre/post Trace extension
  - `ailang messages` CLI byte-identical (anchor inherited from M-COG-RUNTIME)
  - `cmd/wasm/effects.go` stays under 800 LOC (split done in 342ff65b)
- **Documentation:**
  - `docs/docs/guides/wasm-integration.md` Cognitive OS section updated
  - `examples/cognitive_os/README.md` published
  - CHANGELOG entry under v0.21.x
- **All tests passing:** ✅
- **All linting passing:** ✅

---

## Dependencies

- **Locked:** all design freeze items (see top of this doc)
- **Predecessor:** M-COG-RUNTIME Go-side substrate (✅ shipped, PASS @ 91/100)
- **External:**
  - Playwright (devDependency) — npm install on first CI run
  - Browser engines: Chromium / Firefox / WebKit (Playwright manages binaries)
  - No new Go dependencies
- **Forward-locking for next sprints:**
  - `_cog_drain()` builtin shape locks the contract for M-COG-MEMORY's `SharedMem.subscribe(key, onWrite)` and M-COG-MESH's distributed-transport callbacks
  - IndexedDB schema (single `cognitive_events` store) locks the contract for M-COG-MEMORY's persistent SharedMem entries
  - Canonical DOM hashing locks the wire shape for M-COG-MESH cross-device replay

---

## Open Questions / Deferred Decisions

- **Playwright vs Puppeteer for CI** — deferred to executor. Playwright preferred (browser matrix); fallback to Puppeteer only if Playwright bring-up is unexpectedly painful.
- **JS bundling tool** (esbuild / rollup / vite / none) — deferred to executor. The existing `docs/static/wasm/` doesn't bundle (raw `.js` files served direct from static); follow the same pattern unless a clear reason emerges.
- **IndexedDB transaction batching strategy** — deferred to executor. Single-promise serialization is the default; batch only if M5 perf measurement shows it's needed.
- **Cross-origin agent communication** — explicitly out of scope; M-COG-MESH territory.
- **Service worker / offline-first** — explicitly out of scope.
- **`drain(timeout_ms)` semantics** — design doc says blocking; executor confirms whether `drain(0)` means "non-blocking drain" or "drain pending only" during Day 8 implementation.

---

## Non-Goals (explicitly out of scope for this sprint)

- **WebSocket / FirestoreRelay / WebRTC transports** → M-COG-MESH (v0.22.x → v0.23)
- **`!: SharedMem` / `!: SemanticSearch`** → M-COG-MEMORY (v0.22.0)
- **Multi-agent collaborative demo (4-agent topology)** → M-COG-MESH
- **Vector clocks** → M-COG-MESH
- **Custom React-style component DSL** → out of scope
- **Service worker / offline-first** → follow-up
- **Cross-origin agent communication** → M-COG-MESH (needs capability tokens)
- **Combined AI + Cognitive OS demo** (BYO-key chat + scoped DOM render) → post-M-COG-RUNTIME-BROWSER, ideally as a v0.22.x example

---

## Notes

- **Velocity expectation:** M-COG-RUNTIME shipped at ~600 LOC/day (compressed). This sprint targets ~250 LOC/day reflecting (a) JS/browser unfamiliarity, (b) Playwright bring-up, (c) browser-determinism work. Day 11 has explicit buffer.
- **Dev-time browser inspection:** if `chrome-devtools` MCP server is available in the executor session, prefer it for interactive DOM/IndexedDB inspection during M1–M3 (faster than scripted JS probes). Playwright remains the locked CI test harness — chrome-devtools is dev-time only. Fallback if unavailable: `make wasm-serve` + manual browser inspection + scripted JS probes via `eval()`.
- **Behavior-equivalence anchor inherited:** `internal/messaging/` zero diff continues (no Go-side changes touch it). The runtime API remains a sibling to the `ailang messages` CLI.
- **File-size discipline inherited:** `cmd/wasm/effects.go` already split into `effects.go` + `effects_cognition.go` (commit `342ff65b`). New Go files in M4 add ~280 LOC across 2 new files — keeps each under 800.
- **Cross-cutting decisions all locked at design-doc time** — sprint-executor doesn't need human input on any item flagged in Design Freeze.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_21_0/m-cog-runtime-browser-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-COG-RUNTIME-BROWSER.json`
