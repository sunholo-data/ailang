# Sprint Plan: M-COG-RUNTIME

> **Status: SHIPPED** — Go-side substrate complete, commit range `4131d612..b530e482` on `dev`.
> Evaluation: PASS at 91/100 (round 1) — see [`.ailang/state/evaluations/eval_M-COG-RUNTIME_round_1.json`](../../../.ailang/state/evaluations/eval_M-COG-RUNTIME_round_1.json).
> Browser-side substrate continues in [M-COG-RUNTIME-BROWSER](../../planned/v0_21_0/m-cog-runtime-browser.md).

## Summary

Ship the foundation for the Cognitive OS: three new effects (`DOM`, `Msg`, `Trace`), a transport-independent message fabric (LocalWorker + BroadcastChannel), a cognitive event log with Lamport clocks, a deterministic scheduler, and a browser DOM patch runtime with byte-identical replay. Extends the existing evaluator-in-WASM path (`cmd/wasm/`) — no new codegen backend.

**Duration:** 14–17 working days (~3 weeks elapsed)
**Dependencies:**
- Existing WASM evaluator ([cmd/wasm/main.go](../../../cmd/wasm/main.go), [cmd/wasm/effects.go](../../../cmd/wasm/effects.go))
- Existing effect system + JS build tag pattern (`internal/effects/*_js.go`)
- **Existing AI step pattern** ([internal/effects/ai_step.go](../../../internal/effects/ai_step.go), [internal/effects/ai.go](../../../internal/effects/ai.go)) — DOM and Msg handlers follow this richer interface shape, not the simpler `Call` pattern
- Existing messaging subsystem ([internal/messaging/](../../../internal/messaging/)) — runtime API added alongside CLI
- Existing trace infrastructure ([internal/trace/](../../../internal/trace/))

**Risk Level:** Medium — three significant subsystems (effects, fabric, browser runtime); browser-side determinism is the hardest part

**Revision note:** estimates were revised after discovering [internal/effects/ai_step.go](../../../internal/effects/ai_step.go) as the closer analog. DOM and Msg handlers now use the step-pattern interface (`ApplyPatch` / `ApplyBatch` / `Subscribe` for DOM; `Send` / `Recv` / `Subscribe` for Msg) with `Result[..., Err]`-returning variants. M1 grew from ~900 LOC / 5 days to ~1,200 LOC / 6 days; sprint total from ~2,800 LOC / 13–16 days to ~3,100 LOC / 14–17 days. See ["Handler Interfaces — Step Pattern Alignment" in the design doc](./m-cog-runtime.md#handler-interfaces--step-pattern-alignment) for rationale.

**Design Doc:** [m-cog-runtime.md](./m-cog-runtime.md)
**Parent (umbrella):** [m-wasm-reflective-runtime.md](./m-wasm-reflective-runtime.md)
**Target:** v0.21.x

---

## Design Freeze (locked before sprint starts)

These cross-cutting decisions are locked per the design doc's recommendations:

- [x] **DOM authority** = scoped regions only (one DOM root per agent)
- [x] **Layout policy** = canonical layer; ban animations + time-dependent rendering in v1
- [x] **Scheduler** = single-threaded deterministic event loop + Lamport clocks
- [x] **Transports** = LocalWorker + BroadcastChannel only (others deferred to M-COG-MESH)
- [x] **Event log** = IndexedDB append-only, JSONL exportable
- [x] **Effect labels** = `DOM`, `Msg`, `Trace` (locked across all Cognitive OS docs)

---

## Current Status Analysis

### Completed Recently (velocity baseline)
- **M-EXT-AUTHOR-DX** (4 milestones, v0.20.1): ~670 LOC compressed into ~1–2 days (peak)
- **M-WASM-TRACE** (3 milestones, v0.11.1): ~325 LOC in 3 days (~110 LOC/day — comparable WASM sprint)
- **M-WASM-DICTIONARY-DISPATCH** (v0.9.2): shipped — analog precedent for adding new WASM-host behavior
- **M-WASM-CLOSURE-ENV / STREAM-BRIDGE** (v0.8.1): shipped — WASM subsystem precedents

### Velocity
- Comparable WASM-subsystem velocity: ~150–250 LOC/day (implementation + tests)
- This sprint estimates ~3,100 LOC across 3 milestones over ~14–17 days
- Average target: ~200 LOC/day (slightly conservative because of cross-language Go/JS work)

### Remaining from Design Doc
- M1 (Effect Plumbing, step-pattern handlers): ~1,200 LOC (Go + stdlib)
- M2 (Message Fabric + Event Log): ~1,100 LOC (Go + browser-side)
- M3 (Patch Runtime + Scheduler + Replay): ~800 LOC (Go scheduler + browser-side)

---

## Proposed Milestones

### M1: EFFECT_PLUMBING (step-pattern handlers)
**Goal:** Add `DOM`, `Msg`, `Trace` effects to the type system; implement **step-pattern handlers** (`ApplyPatch` / `ApplyBatch` / `Subscribe` for DOM; `Send` / `Recv` / `Subscribe` for Msg) following the [AIHandler interface](../../../internal/effects/ai.go) shape; wire them through `cmd/wasm/effects.go` to JS callbacks; emit a capability manifest sidecar.
**Estimated:** ~800 implementation + ~400 tests = ~1,200 LOC
**Duration:** 6 days (revised from 5)

**Tasks (day-by-day):**

- **Day 1** ✅ **DONE** (commit `59339cda`): Effect-row vocabulary
  - Located effect-label registration sites in `internal/types/`
  - Added `DOM`, `Msg` labels (Trace already shipped v0.11.1 via M-WASM-TRACE)
  - Tests: `TestIsKnownEffect_CognitiveOS`, `TestElaborateEffectRow_CognitiveEffects`, extended `TestSubsumeEffectRows_NoHierarchy`
  - Net: +58 LOC; all `internal/types/` tests green; lint clean

- **Day 2**: DOM step-pattern handler (~280 LOC)
  - `internal/effects/dom.go` — `DOMHandler` interface (`ApplyPatch` / `ApplyBatch` / `Subscribe`), `DOMPatch` ADT (`AddPanel` / `UpdateNode` / `RemoveNode` / `AddTimeline`), `DOMEvent` ADT (clicks/input/viewport), `StubDOMHandler` for tests (mirrors `StubAIHandler` shape)
  - `internal/effects/dom_js.go` — JS-build-tag handler skeleton (full bridge in Day 5–6)
  - `internal/effects/dom_native.go` — non-JS fallback returning `ErrNoDOMHandler`
  - Op registration via `init()`: bare ops (`applyPatch`, `applyBatch`, `subscribe`) + Result-returning variants (`applyPatchResult`, `applyBatchResult`)
  - `stdlib/std/dom.ail` — AILANG bindings (`addPanel`, `updateNode`, `subscribe`)
  - Tests: op registration, `DOMPatch` ADT round-trip, `StubDOMHandler` behaviour, Result-variant shape

- **Day 3**: Msg step-pattern handler (~240 LOC)
  - `internal/effects/msg.go` — `MsgHandler` interface (`Send` / `Recv` / `Subscribe`), `Mailbox` / `Message` / `SendResult` types, `StubMsgHandler`
  - `internal/effects/msg_js.go` + `msg_native.go` — build-tag pair; native routes to existing [internal/messaging/store.go](../../../internal/messaging/store.go)
  - Result-returning op variants (`sendResult`, `recvResult`, `subscribeResult`)
  - **Critical:** behaviour-equivalence snapshot — capture `ailang messages list --compact` output pre-change; diff post-change must be empty
  - `stdlib/std/cognition.ail` — AILANG messaging API skeleton
  - Tests: native handler round-trips through `internal/messaging/`; `StubMsgHandler` simulates peer round-trips

- **Day 4**: Trace cognitive-event extension + capability manifest (~150 LOC)
  - `internal/effects/trace_cognition.go` — extends existing `Trace` effect with cognitive-event emit ops (event-log persistence is M2; here we record + buffer in-memory)
  - `internal/effects/ops.go` + `capability.go` — register new caps + budgets (`DOM.patches`, `Msg.sends`, `Trace.events`)
  - `internal/cognition/manifest.go` — JSON manifest generator (effects + budgets + transports list)
  - Tests: manifest schema round-trip, budget enforcement at registration boundary

- **Day 5–6**: `cmd/wasm/effects.go` WASM wire-up + negative tests + M1 checkpoint (~530 LOC across both days)
  - Wire JS callbacks for `host.dom.*` (3 methods), `host.msg.*` (3 methods), `host.trace.*` (cognitive emit) using existing `setAIHandler` + `awaitJSResult` + `jsRejectionToString` patterns
  - `cmd/wasm/main.go` — register pluggable DOM/Msg handlers in `WasmREPL` init
  - Subscribe-callback cleanup tests: cancel function frees JS side cleanly (no leaked references)
  - Negative tests: program using `!: DOM` without registered handler → typed AILANG `Result.Err(NoHandler)` (analog of existing `ErrNoAIHandler`)
  - **M1 checkpoint:** typecheck passes + behaviour-equivalence test green + lint clean + all DOM/Msg ops registered with both bare and Result-returning variants

**Acceptance Criteria:**
- [x] `DOM`, `Msg`, `Trace` typecheck through row inference (positive + negative) — Day 1 done
- [ ] `DOMHandler` step-pattern interface + `StubDOMHandler` ship with Result-returning op variants
- [ ] `MsgHandler` step-pattern interface + `StubMsgHandler` ship with Result-returning op variants
- [ ] `.wasm` build emits accompanying `manifest.json` with effects + budgets + transports
- [ ] `cmd/wasm/effects.go` exposes `host.dom.*`, `host.msg.*`, `host.trace.*` JS callbacks
- [ ] `ailang messages send/list/read/ack` byte-identical before/after (regression-tested via snapshot diff)
- [ ] Subscribe callbacks cancel cleanly (no leaked JS references)
- [ ] All existing WASM examples still compile + run identically
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Step-pattern interface surface is bigger than original simple-Call estimate — Mitigation: scope already revised (M1 ~900 → ~1,200 LOC; sprint +1 day buffer)
- Subscribe-callback lifecycle complexity (Go side holding JS refs) — Mitigation: study `StepWithStream.onChunk` lifecycle in [ai_step.go:279](../../../internal/effects/ai_step.go#L279); model cancel function the same way
- Behavior-equivalence test for CLI messaging needs careful capture-baseline — Mitigation: snapshot `ailang messages list --compact` pre-change, diff post-change

---

### M2: MESSAGE_FABRIC_AND_EVENT_LOG
**Goal:** Implement the transport trait (LocalWorker + BroadcastChannel), the cognitive event log with Lamport clocks, and IndexedDB persistence. Ship JSONL export/import.
**Estimated:** ~750 implementation + ~350 tests = ~1,100 LOC
**Duration:** 5–6 days

**Tasks (day-by-day):**

- **Day 6**: Transport trait + LocalWorker
  - `internal/cognition/transport.go` — `Transport` interface (`Send`, `Recv`, `Name`) + LocalWorker in-tab impl
  - Wire `internal/effects/msg.go` → `Transport` for runtime dispatch
  - Tests: in-tab send/recv round-trips

- **Day 7**: BroadcastChannel transport
  - `internal/cognition/transport_broadcast.go` — `syscall/js`-bridged BroadcastChannel impl (build-tagged `js && wasm`)
  - Cross-tab test (manual + scripted via headless browser)
  - Pluggable provider check: same agent code runs over both transports

- **Day 8**: Cognitive event log types + Lamport clock
  - `internal/cognition/event_log.go` — `CognitiveEvent` ADT (`MessageSent`, `MessageReceived`, `TraceCaptured`, `PatchApplied`, `CapabilityExceeded`), serialization
  - `internal/cognition/clock.go` — Lamport clock with monotonicity invariants
  - Tests: clock invariants, event serialization round-trip

- **Day 9**: Browser host shim (Phase A — message + trace handlers)
  - Decide browser-side JS location: **default `docs/static/wasm/cognitive-runtime/`** (alongside existing WASM assets) — agent confirms in Phase 3 if needed
  - `host.js` — exposes `host.msg.*` and `host.trace.*` to WASM via existing js-callback pattern
  - `event_log_indexeddb.js` — IndexedDB append-only store
  - JSONL export/import functions

- **Day 10**: Integration test + behavior-equivalence
  - End-to-end: WASM program with `!: {Msg, Trace}` runs in browser, sends message, emits cognitive events, persists to IndexedDB
  - Replay test: load JSONL log, re-emit events in clock order, verify byte-identical state
  - Re-run CLI messaging behavior-equivalence (regression check)

- **Day 11 (buffer)**: Address browser-side determinism quirks, clean up
  - Likely: IndexedDB transaction sequencing, JS clock-skew handling
  - **Checkpoint:** 2 tabs exchange messages via BroadcastChannel; event log replays identically across 3 runs

**Acceptance Criteria:**
- [ ] Transport trait + LocalWorker + BroadcastChannel impls land
- [ ] Two WASM workers in same tab exchange messages via LocalWorker
- [ ] Two browser tabs exchange messages via BroadcastChannel
- [ ] Cognitive event log persists to IndexedDB; JSONL export round-trips
- [ ] Lamport clock invariants hold (monotonic, tiebreaks via sender ID)
- [ ] 3 runs of identical scenario → byte-identical event log
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- `syscall/js` BroadcastChannel surface may need polyfill in older browsers — Mitigation: ship feature-detection + graceful degradation (LocalWorker only)
- IndexedDB transaction ordering is async — Mitigation: serialize writes through a single promise chain in `event_log_indexeddb.js`
- JSONL schema drift between this milestone and M-COG-MEMORY — Mitigation: schema is forward-compat (unknown event kinds skipped)

---

### M3: PATCH_RUNTIME_AND_SCHEDULER
**Goal:** Browser DOM patch runtime with canonical (deterministic) node IDs, single-threaded deterministic scheduler, replay engine, budget enforcement. Ship the demo example.
**Estimated:** ~550 implementation + ~250 tests = ~800 LOC
**Duration:** 5 days

**Tasks (day-by-day):**

- **Day 12**: Deterministic scheduler (Go + JS sides)
  - `internal/cognition/scheduler.go` — single-threaded event-loop driver; mailbox ordering via `(lamportClock, senderID)` tiebreak; patch ordering via `(clock, region, contentHash)`
  - `scheduler.js` — JS-side event loop (microtask-based, no `setTimeout`)
  - Tests: scheduler produces identical event order across 3 runs

- **Day 13**: Canonical DOM layer + patch runtime
  - `canonical_dom.js` — content-hashed node IDs (banishes `Date.now()`, `Math.random()`, browser-supplied IDs)
  - `host.js` Phase B — `host.dom.patch(patchBytes)` decodes IR, applies to scoped region only
  - Capability budget enforcement: trap on overrun via existing `internal/effects/budget.go` pattern → emit `CapabilityExceeded` event

- **Day 14**: Replay engine
  - `replay.js` — load JSONL event log, feed scheduler in clock order, reapply patches, assert DOM equality at each step
  - End-to-end replay test: same event log produces identical DOM in 3 separate browser sessions

- **Day 15**: Demo example + docs
  - `examples/wasm_reflective/single_agent_replay.ail` — minimal program that uses `!: {DOM, Msg, Trace}`, sends a message, mutates DOM, replays cleanly
  - Update `docs/docs/guides/wasm-runtime.md` with Cognitive OS section
  - Update `docs/docs/guides/agent-messaging.md` with runtime API + event log

- **Day 16 (buffer / polish)**: Address determinism regressions, ship CHANGELOG
  - Likely: canonical-DOM corner cases (event listener identity, text-node normalization)
  - CHANGELOG entry under v0.21.x
  - **Final checkpoint:** demo runs + replays + all M1/M2 tests still green

**Acceptance Criteria:**
- [ ] Single-threaded deterministic scheduler ships
- [ ] Canonical DOM layer: content-hashed node IDs, no random, no `Date.now()`
- [ ] Browser host applies DOM patches deterministically (same event log → same DOM)
- [ ] Replay engine reconstructs DOM byte-identically from event log
- [ ] Capability budget overrun → typed AILANG failure + `CapabilityExceeded` event in log
- [ ] Demo example (`single_agent_replay.ail`) passes + replays
- [ ] CHANGELOG + docs updated
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Browser DOM has hidden nondeterminism (layout pass timing, font loading) — Mitigation: canonical layout layer bans animations; layout is content-hashed not timestamp-based
- Replay test flakiness — Mitigation: assert DOM-string equality at each patch application, not just final state
- Patch IR size for complex UIs — Mitigation: diff-based patches + content-hashed node reuse (already in scope)

---

## Success Metrics

- Test coverage: maintain or improve current package coverage; new files ≥80%
- **Examples working and verified:** `examples/wasm_reflective/single_agent_replay.ail` runs + replays in browser
- Behavior equivalence: `ailang messages` CLI byte-identical pre/post sprint
- Documentation:
  - `docs/docs/guides/wasm-runtime.md` — Cognitive OS section
  - `docs/docs/guides/agent-messaging.md` — runtime API + event log
  - CHANGELOG entry under v0.21.x
- All tests passing: ✅
- All linting passing: ✅

---

## Dependencies

- **Locked:** all design freeze items (see top of this doc)
- **External:** none — entirely additive within the existing AILANG codebase
- **Forward-locking for siblings:**
  - Effect-label vocabulary (`DOM`/`Msg`/`Trace`) → reused by M-COG-MEMORY (adds `SharedMem`/`SemanticSearch`)
  - Event-log schema → extended by M-COG-MEMORY (memory events) + M-COG-MESH (distributed events)
  - Transport trait shape → extended by M-COG-MESH (WebSocket / FirestoreRelay impls)
  - Lamport clock → upgraded to vector clocks in M-COG-MESH

---

## Open Questions

- **Browser-side JS location.** Default: `docs/static/wasm/cognitive-runtime/` (sibling to existing WASM assets). Agent may revisit Day 9 if a separate `web/runtime/` tree makes more sense for testing. *Deferred decision per design doc.*
- **Patch IR wire format.** Protobuf vs. MessagePack vs. AILANG-native JSON. *Deferred decision; trace round-trip is the acceptance test.*
- **JSONL extension.** `.ailog` vs. `.aitrace` vs. `.jsonl`. *Deferred; agent chooses Day 9.*

---

## Notes

- This is **Child 1 of 3** of the Cognitive OS arc ([umbrella](./m-wasm-reflective-runtime.md)). On completion: ship as v0.21.x release, then start M-COG-MEMORY sprint planning.
- **WASM execution model is the existing evaluator-in-WASM path** — no bytecode VM coupling, no new codegen. New effects follow the established `internal/effects/*_js.go` build-tag pattern.
- The bytecode VM ([internal/bytecode/compiler/](../../../internal/bytecode/compiler/)) is orthogonal; when it eventually ships it integrates via the same `internal/effects/` layer.
- Day 11 and Day 16 are explicit buffer days — assume browser-side determinism work will consume them.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_21_0/m-cog-runtime-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-COG-RUNTIME.json`
