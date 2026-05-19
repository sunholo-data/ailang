# M-COG-RUNTIME: Cognition Runtime — Effects, Fabric, Event Log, Scheduler, DOM Replay

**Status**: Planned
**Target**: v0.21.x
**Priority**: P0
**Estimated**: ~95h across 3 weeks
**Parent**: [M-WASM-REFLECTIVE-RUNTIME (Cognitive OS umbrella)](./m-wasm-reflective-runtime.md)
**Dependencies**:
- Existing WASM evaluator/runtime ([m-wasm-dictionary-dispatch](../../implemented/v0_9_2/m-wasm-dictionary-dispatch.md), [m-wasm-closure-env](../../implemented/v0_8_1/m-wasm-closure-env.md))
- Effect system (`!: IO, FS, Net, ...`)
- Existing AILANG messaging CLI (`ailang messages send/list/read/ack`) — runtime API added alongside

---

## Scope

This is **Child 1 of 3** of the Cognitive OS arc. It delivers the foundation: the effect plumbing, the message fabric, the cognitive event log, the deterministic scheduler, and the DOM replay engine.

**What this doc covers:**
- Phase 1 (~30h): Effect + IR plumbing — `DOM`, `Msg`, `Trace` effect labels + WASM imports + capability manifest
- Phase 2 (~35h): Message Fabric + Cognitive Event Log + LocalWorker + BroadcastChannel transports
- Phase 3 (~30h): Browser DOM patch runtime + deterministic scheduler + replay engine

**What this doc explicitly defers:**
- `!: SharedMem` and `!: SemanticSearch` (→ [M-COG-MEMORY](../v0_22_0/m-cog-memory.md))
- WebSocket / Firestore / A2A / MCP / WebRTC transports (→ [M-COG-MESH](../v0_22_0/m-cog-mesh.md))
- Multi-agent collaborative demo (→ [M-COG-MESH](../v0_22_0/m-cog-mesh.md))
- Vector clocks (Lamport only here)

**Shippable deliverable:** at end of v0.21.x, an AILANG program can compile to WASM, run in a browser tab, mutate the DOM deterministically, send messages to another worker (in-tab or cross-tab), and replay byte-identically from the cognitive event log.

---

## WASM Execution Model (Important — Not a New Backend)

**This doc extends the existing WASM evaluator path; it does NOT introduce a new codegen backend.**

AILANG's WASM target ([cmd/wasm/main.go](../../../cmd/wasm/main.go)) compiles the **AILANG tree-walking evaluator + REPL** into a WASM binary, exposing it to JavaScript via `syscall/js`. Programs don't compile *to* WASM bytecode — the **interpreter** runs in WASM and evaluates source. Effect handlers wire to JS callbacks via [cmd/wasm/effects.go](../../../cmd/wasm/effects.go) and per-effect `*_js.go` files in [internal/effects/](../../../internal/effects/) (e.g. `process_context_js.go`, `stream_async_process_js.go`).

**Why this matters:**
- Adding `!: DOM` / `!: Msg` follows the **same pattern** as existing v0.19.0 work ([m-wasm-cloud-messages](../v0_19_0/m-wasm-cloud-messages.md), [m-wasm-ai-step-via-messages](../v0_19_0/m-wasm-ai-step-via-messages.md)): one new file in `internal/effects/`, one JS-callback wire-up in `cmd/wasm/effects.go`, one browser-side handler.
- The bytecode VM work in [internal/bytecode/compiler/](../../../internal/bytecode/compiler/) + [internal/vm/](../../../internal/vm/) is a **separate execution backend** at EvalOnly-parity stage — orthogonal to this doc. When bytecode VM eventually ships, it integrates via the same `internal/effects/` layer; this design does not couple to its completion.
- The `Msg` effect wraps the existing [internal/messaging/](../../../internal/messaging/) subsystem as a runtime API; the `ailang messages` CLI continues to work byte-identically against the same store.

**What this means concretely:** no new IR. No new codegen pass. We add effect handlers, wire JS callbacks, write browser-side host code, and emit a capability manifest sidecar. That's it.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Deterministic scheduler + Lamport clocks + canonical DOM patches |
| A2: Replayability | +2 | Cognitive event log enables byte-identical replay |
| A3: Effect Legibility | +2 | New `DOM`, `Msg`, `Trace` effects — cognition is physical |
| A4: Explicit Authority | +2 | Capability manifest enforces budgets; no import = impossible effect |
| A5: Bounded Verification | +1 | Patches verifiable; event log enables divergence detection |
| A6: Safe Concurrency | +2 | Single-threaded event loop + deterministic mailbox ordering |
| A7: Machines First | +2 | Runtime optimized for agent cognition, not human UX |
| A8: Minimal Syntax | +1 | 3 new effect labels; no new constructs |
| A9: Cost Visibility | +1 | Capability budgets per node, in manifest + event log |
| A10: Composability | +2 | Transport-independent agent semantics |
| A11: Structured Failure | +1 | WASM traps → typed AILANG failures; `CapabilityExceeded` events |
| A12: System Boundary | +2 | Strong isolation; transport abstraction makes boundaries explicit |

**Net: +20** → ✅ Proceed

### Hard Violation Check
- [x] A1, A3, A4, A7 all ≥ 0

---

## Problem Statement

Without this foundation, the rest of the Cognitive OS can't exist:
- No effect labels → no `!: DOM` for [M-COG-MEMORY](../v0_22_0/m-cog-memory.md) to extend
- No event log → no replayability for [M-COG-MESH](../v0_22_0/m-cog-mesh.md)
- No scheduler → multi-agent execution nondeterministic
- No transport trait → can't add new transports incrementally

Current state: AILANG WASM runs single-agent, no DOM access, no cross-worker messaging, no event log, no scheduler abstraction.

---

## Goals

**Primary:** Deliver the WASM runtime foundation + message fabric + event log that the rest of the Cognitive OS builds on.

**Success Metrics:**
- `ailang compile --target wasm-reflective` emits `.wasm` + `manifest.json` (with effects + transports list)
- Program with `!: DOM` runs in browser, applies canonical patches
- Two WASM workers in same tab exchange messages via LocalWorker transport
- Two browser tabs exchange messages via BroadcastChannel transport
- Same scenario produces byte-identical event log across 3 runs
- Replay of event log reconstructs identical DOM
- Capability budget overrun produces typed AILANG failure + `CapabilityExceeded` event
- All existing WASM examples still compile + run identically
- All existing `ailang messages` CLI commands byte-identical behavior

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| DOM authority granularity (scoped regions per agent) | Multi-agent UI sharing safety | human | design | high |
| Deterministic layout policy (canonical layer, no time-dependent rendering) | Replay fidelity | human | design | high |
| Scheduler model (single-threaded event loop + Lamport) | Replay correctness | human | design | high |
| Transport set in MVP (LocalWorker + BroadcastChannel) | Bounds MVP scope | human | design | high |
| Event log storage (IndexedDB append-only + JSONL export) | Durability + tooling | human | design | high |
| Effect-label names (`DOM`, `Msg`, `Trace`) | Forward-compat with M-COG-MEMORY + M-COG-MESH | human | design | med |
| `!: DOM` granularity (atomic patches vs. transactional batches) | Failure semantics | human | compile | med |
| Capability manifest format (JSON file vs. WASM custom section) | Tooling alignment | agent | compile | med |
| Patch IR encoding (Protobuf / MessagePack / AILANG-native) | Trace size | agent | compile | low |

### Design Freeze

- [ ] DOM authority = **scoped regions only** (one DOM root per agent)
- [ ] Layout policy = **canonical layer**, ban animations + time-dependent rendering in v1
- [ ] Scheduler = **single-threaded deterministic event loop + Lamport clocks**
- [ ] Transports = **LocalWorker + BroadcastChannel only** (others in M-COG-MESH)
- [ ] Event log = **IndexedDB append-only, JSONL exportable**
- [ ] Effect labels = **`DOM`, `Msg`, `Trace`** (locked across all Cognitive OS docs)

---

## Conflict Surface

Touches [internal/effects/](../../../internal/effects/), [cmd/wasm/](../../../cmd/wasm/), `internal/types/` (effect row vocabulary), new [internal/cognition/](../../../internal/cognition/) subtree, and [internal/messaging/](../../../internal/messaging/). Required analysis:

### Positions extended
1. **Effect rows** — adding `DOM`, `Msg`, `Trace` labels
2. **WASM imports** — new namespaces `host.dom.*`, `host.msg.*`, `host.trace.*`
3. **Trace events** — new kinds: `DOMPatch`, `MessageSent`, `MessageReceived`, `CapabilityExceeded`
4. **Capability manifest** — new effect labels + transport list field
5. **Messaging API** — runtime API alongside existing CLI; both wire to same event log

### Existing constructs in these positions
- Effect rows: `!: IO`, `!: {IO, FS}` — additive, no syntactic conflict
- WASM imports: existing `host.io.*`, `host.fs.*` — namespacing prevents collision
- Trace events: existing `EffectStart`, `EffectEnd`, `Trap` — additive
- Messaging: existing `ailang messages send/list/read/ack` continues to work; runtime API is sibling

### Programs that MUST still work
1. Pure programs (zero host imports)
2. `!: IO` programs (current `host.io.*` imports)
3. Polymorphic effect handlers (`forall r. ... ! r`)
4. All `examples/*_wasm.ail` files
5. All `ailang messages` CLI commands (byte-identical)
6. All existing benchmarks (≤5% regression budget on non-cognition workloads)

### Deliberately changed
- Programs using `!: DOM` / `!: Msg` fail at link time if host doesn't provide imports — intentional capability gating
- Trace files gain new event types; old replayers skip unknown (forward-compat)
- Messaging gains runtime API (effect-typed `send`/`recv`) alongside CLI

---

## Solution Design

### Effect Plumbing (Phase 1)

Add `DOM`, `Msg`, `Trace` to effect row vocabulary in `internal/types/effects.go`. WASM emitter wires corresponding host imports. Capability manifest generator emits JSON with effect list + budget map + transport list.

```ailang
fn applyHeatmap(t: Trace) !: DOM = AddPanel({ title: "Heatmap", content: renderHeatmap(t) })
fn relayTrace(t: Trace) !: Msg = send(NodeId("verifier"), TraceReport { trace: t })
```

```json
{
  "module": "agent",
  "effects": ["DOM", "Msg"],
  "budgets": { "DOM.patches": 20, "Msg.sends": 100 },
  "transports": ["LocalWorker", "BroadcastChannel"]
}
```

### Message Fabric + Event Log (Phase 2)

**Transport trait** (Go-side):
```go
type Transport interface {
    Send(to NodeID, payload []byte) error
    Recv() (msg Message, err error)
    Name() string
}
```

Implementations: `LocalWorkerTransport`, `BroadcastChannelTransport`. Agent semantics unchanged across transports.

**Cognitive Event Log** (append-only, IndexedDB-backed, JSONL exportable):

```ailang
type CognitiveEvent =
  | MessageSent       { from: NodeId, to: Mailbox, msgId: MsgId, payloadHash: Hash, clock: Lamport }
  | MessageReceived   { node: NodeId, msgId: MsgId, clock: Lamport }
  | TraceCaptured     { node: NodeId, span: TraceSpan }
  | PatchApplied      { node: NodeId, region: DOMRegion, patch: DOMPatch, clock: Lamport }
  | CapabilityExceeded { node: NodeId, effect: EffectLabel, budget: Budget }
```

Memory + semantic events deferred to M-COG-MEMORY; distributed events deferred to M-COG-MESH.

### Deterministic Scheduler (Phase 3)

Single-threaded event loop in browser host. Mailbox ordering: `(lamportClock, senderID)` tiebreak. Patch ordering: `(clock, region, contentHash)`. Lamport-clock invariant maintained on every event emit.

### DOM Replay Engine (Phase 3)

- Canonical layout layer: content-hashed node IDs, no random, no `Date.now()`
- Patch IR: structured `DOMPatch` ADT (AddPanel / UpdateNode / RemoveNode / AddTimeline)
- Replay: feed event log to scheduler in original clock order, apply patches, assert DOM equality

---

## Implementation Plan

### Phase 1: Effect Plumbing (~30h)

- [ ] Add `DOM`, `Msg`, `Trace` to effect row vocabulary (find existing `IO`/`FS`/`Net` label site in `internal/types/`)
- [ ] Define `DOMPatch` types in `internal/effects/dom.go`; JS bridge in `internal/effects/dom_js.go`
- [ ] `internal/effects/msg.go` + `msg_js.go` — runtime `send`/`recv` wrapping `internal/messaging/`
- [ ] `internal/effects/trace_cognition.go` — `!: Trace` for cognitive event emission
- [ ] `internal/effects/ops.go` + `capability.go` — register new ops + caps + budgets
- [ ] `cmd/wasm/effects.go` — wire JS callbacks for `host.dom.*`, `host.msg.*`, `host.trace.*` (follow `awaitJSResult` pattern)
- [ ] `internal/cognition/manifest.go` — capability manifest generator (effects + budgets + transports)
- [ ] Typechecker tests: each new effect flows through row inference
- [ ] Negative tests: missing JS handler → runtime failure (analog of "missing import")
- [ ] `stdlib/std/dom.ail` — DOM effect bindings
- [ ] `stdlib/std/cognition.ail` — messaging + tracing API skeleton

### Phase 2: Message Fabric + Event Log (~35h)

- [ ] Transport trait (`internal/cognition/transport.go`) + LocalWorker impl
- [ ] BroadcastChannel transport (`internal/cognition/transport_broadcast.go`, build-tagged `js && wasm`)
- [ ] Runtime messaging API in `internal/effects/msg.go` wired to existing `internal/messaging/store.go`
- [ ] Append-only cognitive event log (`internal/cognition/event_log.go`)
- [ ] Lamport logical clock (`internal/cognition/clock.go`)
- [ ] Browser host shim — exposes `host.msg.*`, `host.trace.*` to WASM (location TBD: `docs/static/wasm/host.js` or new `web/runtime/`)
- [ ] IndexedDB event log store (browser-side)
- [ ] JSONL export/import for event logs
- [ ] Behavior-equivalence tests: `ailang messages` CLI byte-identical post-change

### Phase 3: Patch Runtime + Scheduler + Replay (~30h)

- [ ] Deterministic event-loop scheduler (`internal/cognition/scheduler.go`)
- [ ] JS-side scheduler (`scheduler.js`)
- [ ] Canonical DOM layer (`canonical_dom.js`)
- [ ] Browser patch runtime (`host.js` — `host.dom.patch` handler)
- [ ] Replay engine (`replay.js`)
- [ ] Capability budget enforcement: trap + `CapabilityExceeded` event (via existing `internal/effects/budget.go` pattern)
- [ ] End-to-end test: program with `!: {DOM, Msg}` runs, mutates DOM, sends message, replays identically
- [ ] Demo example: `examples/wasm_reflective/single_agent_replay.ail`

### Files to Modify/Create

**Important:** paths reflect the existing AILANG layout — WASM lives in [cmd/wasm/](../../../cmd/wasm/), effects in [internal/effects/](../../../internal/effects/) (with `_js.go` build-tagged variants for browser builds), messaging in [internal/messaging/](../../../internal/messaging/). No new `codegen/` subtree is introduced.

**New files (Go-side):**
- `internal/effects/dom.go` + `internal/effects/dom_js.go` — DOM effect: shared types + JS-build-tag handler that bridges to host (~400 LOC total)
- `internal/effects/msg.go` + `internal/effects/msg_js.go` — `!: Msg` runtime API wrapping [internal/messaging/](../../../internal/messaging/) (~250 LOC total)
- `internal/effects/trace_cognition.go` — `!: Trace` effect for cognitive event emission (~150 LOC; extends existing trace plumbing)
- `internal/cognition/event_log.go` — append-only cognitive event log types + persistence (~400 LOC)
- `internal/cognition/clock.go` — Lamport clock (~150 LOC)
- `internal/cognition/scheduler.go` — deterministic event-loop scheduler (~350 LOC)
- `internal/cognition/transport.go` — transport trait + LocalWorker impl (~250 LOC)
- `internal/cognition/transport_broadcast.go` — BroadcastChannel transport via `syscall/js` (build-tagged) (~200 LOC)
- `internal/cognition/manifest.go` — capability manifest generator (effects + budgets + transports) (~250 LOC)

**New files (browser-side, under [docs/static/wasm/](../../../docs/static/wasm/) or a new `web/runtime/` sibling — agent-choice in Phase 3):**
- `host.js` — browser host harness exposing `host.dom.*`, `host.msg.*`, `host.trace.*` to WASM (~600 LOC)
- `replay.js` — replay engine (~350 LOC)
- `canonical_dom.js` — deterministic node ID layer (~250 LOC)
- `scheduler.js` — JS-side event loop (~300 LOC)
- `event_log_indexeddb.js` — IndexedDB store (~250 LOC)

**New files (stdlib + examples):**
- `stdlib/std/dom.ail` — DOM effect bindings (~200 LOC)
- `stdlib/std/cognition.ail` — `Msg` + `Trace` API (~200 LOC)
- `examples/wasm_reflective/single_agent_replay.ail` — demo (~150 LOC)

**Modified files:**
- `internal/effects/ops.go` — register `DOM`, `Msg`, `Trace` ops + budgets (~80 LOC)
- `internal/effects/capability.go` — add new capability labels (~30 LOC)
- `internal/types/` (effect row vocabulary — find the existing `IO`/`FS`/`Net` label site and extend) (~50 LOC)
- `cmd/wasm/effects.go` — wire JS callbacks for `host.dom.*`, `host.msg.*`, `host.trace.*` following existing `awaitJSResult` / `jsRejectionToString` patterns (~200 LOC)
- `cmd/wasm/main.go` — registry wiring for new effect handlers in `WasmREPL` init (~50 LOC)
- `cmd/wasm/trace.go` — extend with cognitive event emission (~100 LOC)
- `internal/messaging/messages.go` (and adjacent) — expose runtime-callable API alongside CLI; both wire to same store (~200 LOC, additive)
- `docs/docs/guides/wasm-runtime.md` — Cognitive OS section
- `docs/docs/guides/agent-messaging.md` — runtime API + event log

---

## Examples

### Example 1: Single-Agent Replayable DOM

**Before:** AILANG WASM can compute but can't render or replay.

**After:**

```ailang
module demo/heatmap

import std/dom (AddPanel)
import std/cognition (Trace)

export func renderHeatmap(t: Trace) -> Unit ! {DOM, Trace} {
  let panel = AddPanel({ title: "Failure Heatmap", content: render(t) })
  emit(TraceCaptured { span: "renderHeatmap" })
}
```

Run twice → byte-identical event log. Export log → import in another tab → identical DOM.

### Example 2: Cross-Tab Messaging

```ailang
module demo/cross_tab

import std/cognition (send, recv, NodeId)

-- Tab A
export func sender() -> Unit ! Msg {
  send(NodeId("tab-b"), Hello)
}

-- Tab B
export func receiver() -> Unit ! Msg {
  let msg = recv()
  -- ...
}
```

Manifest declares `"transports": ["BroadcastChannel"]`. Tabs share via `BroadcastChannel`. Event log records sends + receives with Lamport clocks; receiver order is deterministic.

---

## Success Criteria

- [ ] `DOM`, `Msg`, `Trace` effects typecheck through row inference
- [ ] `.wasm` + `manifest.json` (with transports) emitted from `ailang compile --target wasm-reflective`
- [ ] Browser host applies DOM patches deterministically (same event log → same DOM)
- [ ] Two WASM workers in same tab exchange messages via LocalWorker
- [ ] Two browser tabs exchange messages via BroadcastChannel
- [ ] 3 runs of same scenario → byte-identical event log
- [ ] Replay of exported event log → byte-identical DOM
- [ ] Capability budget overrun → typed AILANG failure + `CapabilityExceeded` event
- [ ] All existing WASM examples compile + run identically
- [ ] All existing `ailang messages` CLI commands byte-identical
- [ ] Effect-label vocabulary documented for M-COG-MEMORY / M-COG-MESH to extend
- [ ] CHANGELOG + docs updated
- [ ] Demo example added

---

## Testing Strategy

**Unit:**
- Each new effect's row inference (positive + negative)
- Patch IR round-trip
- Capability manifest matches effect rows + transports
- Lamport clock invariants
- Canonical DOM node ID stability

**Integration:**
- WASM module with `!: {DOM, Msg, Trace}` links + runs against host harness
- Event log → replay → byte-identical state
- Cross-tab BroadcastChannel transport
- Budget enforcement traps + emits `CapabilityExceeded`
- Behavior-equivalence: `ailang messages` CLI before/after change

**Manual:**
- Demo example in browser: render heatmap, replay log

---

## Deferred Decisions

- **Patch IR wire format** (Protobuf / MessagePack / AILANG-native) — agent choice; trace round-trip = acceptance
- **Lamport clock encoding** (varint / fixed64) — agent choice
- **Panel layout primitives** (flexbox-canonical / grid-canonical) — agent choice; deterministic output is constraint
- **Event log file extension** (`.ailog` / `.aitrace` / `.jsonl`) — agent choice
- **JS host shim packaging** (npm vs. embedded in `ailang serve`) — agent choice in Phase 3
- **IndexedDB schema versioning** — agent choice; must support forward migration

---

## Non-Goals

- `!: SharedMem` + `!: SemanticSearch` (→ M-COG-MEMORY)
- WebSocket / Firestore / A2A / MCP / WebRTC transports (→ M-COG-MESH)
- Vector clocks (Lamport only here; vector clocks in M-COG-MESH)
- Multi-agent collaborative demo (→ M-COG-MESH)
- Server-side native WASM host (browser only in v0.21.x)

---

## Timeline

**Week 1 (~30h):** Phase 1 — effect plumbing + IR + manifest
**Week 2 (~35h):** Phase 2 — fabric + event log + LocalWorker + BroadcastChannel
**Week 3 (~30h):** Phase 3 — DOM runtime + scheduler + replay + demo

**Total: ~95h across 3 weeks**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Browser DOM nondeterminism breaks replay | High | Canonical layout layer; ban animations/time-dependent rendering |
| Effect labels chosen now constrain later docs | High | Cross-cutting decisions locked in umbrella doc |
| Existing `ailang messages` CLI users see regressions | High | Behavior-equivalence tests; runtime API is sibling |
| Event log grows unbounded in long-running tabs | Med | Truncation events + size cap; full retention policy in M-COG-MEMORY |
| Patch IR explodes for complex UIs | Med | Diff-based patches + content-hashed node reuse |

---

## Related Documents

- **Parent**: [M-WASM-REFLECTIVE-RUNTIME (umbrella)](./m-wasm-reflective-runtime.md)
- **Next**: [M-COG-MEMORY](../v0_22_0/m-cog-memory.md) — adds `!: SharedMem` + `!: SemanticSearch`
- **After**: [M-COG-MESH](../v0_22_0/m-cog-mesh.md) — adds collaborative demo + distributed transports

**Implemented (foundation):**
- [m-wasm-dictionary-dispatch](../../implemented/v0_9_2/m-wasm-dictionary-dispatch.md)
- [m-wasm-closure-env](../../implemented/v0_8_1/m-wasm-closure-env.md)
- [m-wasm-stream-bridge](../../implemented/v0_8_1/m-wasm-stream-bridge.md)

## References

- [Design Axioms](/docs/references/axioms)
- [AILANG Agent Messaging Guide](../../../docs/docs/guides/agent-messaging.md)

---

**Document created**: 2026-05-18
**Last updated**: 2026-05-18

---

**DESIGN_DOC_PATH**: `design_docs/planned/v0_21_0/m-cog-runtime.md`
