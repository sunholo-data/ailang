# M-COG-MESH: Distributed Cognitive Mesh — Collaborative Demo + Distributed Transports

**Status**: Planned
**Target**: v0.22.x → v0.23
**Priority**: P0
**Estimated**: ~45h+ across 2 weeks (Phase 1: demo, Phase 2: distributed transports)
**Parent**: [M-WASM-REFLECTIVE-RUNTIME (Cognitive OS umbrella)](../v0_21_0/m-wasm-reflective-runtime.md)
**Dependencies**:
- **[M-COG-RUNTIME](../v0_21_0/m-cog-runtime.md)** — provides effects + fabric + event log + scheduler
- **[M-COG-MEMORY](./m-cog-memory.md)** — provides `!: SharedMem` for cross-agent state

---

## Scope

This is **Child 3 of 3** of the Cognitive OS arc. It delivers the **distributed cognition** layer: multiple agents collaborating across nodes and devices.

**What this doc covers:**
- **Phase 1 (~25h, v0.22.x): Collaborative 4-agent demo** using transports from M-COG-RUNTIME (LocalWorker + BroadcastChannel). Validates that the umbrella vision actually composes.
- **Phase 2 (~20h+, v0.23.0): Distributed transports** — WebSocket, FirestoreRelay; A2A / MCP / WebRTC as opt-ins.
- Vector clocks for true distributed ordering (upgrade from Lamport)
- New cognitive events: `VerificationPassed`, `EvalFailed`, distributed-mesh events
- Topology visualizer (live graph of agents + message flow + capability usage)

**What this doc explicitly defers:**
- CRDT-based concurrent-write resolution (future work)
- Capability marketplaces (far future)
- Formal verification of UI mutations / SMT (far future)
- Native (non-browser) host runtime (separate doc)

**Shippable deliverable:** at end of v0.23.0, four agents (planner / executor / ui-agent / verifier) collaborate across two devices via WebSocket + FirestoreRelay to detect a failure, synthesize a harness patch, verify it, and distribute it deterministically. The full event log replays byte-identically on a third machine.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Vector clocks + canonical mailbox ordering preserve determinism across nodes |
| A2: Replayability | +2 | Distributed event log replays byte-identically on third machine |
| A3: Effect Legibility | +2 | All cross-node ops require `!: Msg` (no hidden network) |
| A4: Explicit Authority | +2 | Transport-independent authority; each transport gated separately in manifest |
| A5: Bounded Verification | +2 | Verifier agent validates patches before broadcast; divergence detected via log |
| A6: Safe Concurrency | +2 | Per-node deterministic scheduler + vector clocks for cross-node order |
| A7: Machines First | +2 | Mesh designed for agent cognition, not human collaboration UX |
| A8: Minimal Syntax | 0 | No new effect labels; reuses M-COG-RUNTIME vocabulary |
| A9: Cost Visibility | +1 | Per-transport budgets; aggregated in event log |
| A10: Composability | +2 | Same agent code over any transport — strongest composability gain |
| A11: Structured Failure | +1 | Network failures → typed `TransportError`; capability events preserved |
| A12: System Boundary | +2 | Every cross-node hop is an explicit transport boundary |

**Net: +20** → ✅ Proceed

### Hard Violation Check
- [x] A1, A3, A4, A7 all ≥ 0

---

## Problem Statement

After M-COG-RUNTIME + M-COG-MEMORY ship, single-tab and cross-tab cognition works. But:
- Cognition still bounded by one origin (same browser, same machine)
- No way to demonstrate collaborative self-improvement (the headline strategic claim)
- No way to validate that the transport abstraction actually generalizes
- Lamport clocks insufficient for true distributed ordering

This doc closes the loop on the umbrella vision.

---

## Goals

**Primary:** Prove the Cognitive OS framing by demonstrating four agents collaborating across two devices, with byte-identical replay on a third machine.

**Success Metrics:**
- 4-agent topology (planner / executor / ui-agent / verifier) runs end-to-end via LocalWorker
- Same topology runs end-to-end via WebSocket across two devices, **zero source-code changes**
- Distributed event log replays byte-identically on a third machine
- Failure → trace → patch synthesis → verifier approval → broadcast loop completes
- Vector clocks correctly order events across nodes
- Topology visualizer shows live agent graph + message flow + capability usage
- All M-COG-RUNTIME + M-COG-MEMORY demos still pass

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Transport priority order (WebSocket-first vs. WebRTC-first) | What ships in v0.23 vs. later | human | design | high |
| Vector clock granularity (per-node / per-effect / hybrid) | Replay cost | human | design | high |
| Cross-origin auth model (capability tokens / origin allow-list / explicit pairing) | Security posture | human | design | high |
| FirestoreRelay schema (cognition-log shape) | Cloud-side compatibility | human | design | high |
| Patch synthesis approach for demo (rule-based vs. LLM-call via `!: AI`) | Demo realism | human | design | med |
| Topology-visualizer location (in-browser panel / separate page / CLI tool) | UX entry point | human | design | med |
| Vector clock encoding | Wire format | agent | compile | low |
| Demo agent prompt templates | Demo polish | agent | compile | low |

### Design Freeze

- [ ] Transport priority = **WebSocket first** (cloud relay path), **FirestoreRelay second** (durable async), **WebRTC / A2A / MCP deferred**
- [ ] Vector clocks = **per-node** (one entry per participating node)
- [ ] Cross-origin auth = **explicit pairing via shared capability token** (no ambient origin trust)
- [ ] FirestoreRelay schema = **append-only collection mirroring JSONL event log**
- [ ] Demo patch synthesis = **rule-based for Phase 1** (deterministic); LLM-via-`!: AI` as v0.23.x stretch
- [ ] Topology visualizer = **in-browser panel** (uses `!: DOM` so it dogfoods the runtime)

---

## Conflict Surface

**Path note:** follows the same execution-model conventions as [M-COG-RUNTIME](../v0_21_0/m-cog-runtime.md) — transports live in [internal/cognition/](../../../internal/cognition/) (the trait was locked there); no new codegen subtree.

Touches `internal/cognition/transport*.go` (new transport impls), `internal/cognition/clock.go` (Lamport → vector), `internal/cognition/manifest.go` (extended transport list), and the browser host.

### Positions extended
1. **Transport trait** — adding WebSocket, FirestoreRelay implementations (trait shape was locked in M-COG-RUNTIME)
2. **Logical clocks** — vector clocks alongside Lamport (Lamport remains for single-node)
3. **Capability manifest** — transports list gains new entries
4. **Cognitive event log** — new event kinds: `VerificationPassed`, `EvalFailed`, `TransportError`, `NodeJoined`, `NodeLeft`
5. **Cross-origin authority** — capability token in manifest

### Existing constructs in these positions
- Transport trait: locked in M-COG-RUNTIME; this doc adds implementations, doesn't reshape
- Clocks: Lamport from M-COG-RUNTIME continues working for single-node scenarios
- Manifest: transports list already exists; just adds entries
- Event log: new event kinds additive

### Programs that MUST still work
1. All M-COG-RUNTIME programs (single-agent, LocalWorker, BroadcastChannel)
2. All M-COG-MEMORY programs (persistence across restart)
3. Lamport-clock event logs from earlier versions replay correctly
4. Programs using only LocalWorker transport: zero behavioral change

### Deliberately changed
- Programs declaring `WebSocket` in manifest require network — link succeeds without it but `send` fails at runtime
- Vector clocks add bytes to every cross-node event; single-node logs unaffected
- Cross-origin agents must present capability token; ambient trust removed

---

## Solution Design

### Phase 1: Collaborative 4-Agent Demo (v0.22.x)

Uses **only** transports from M-COG-RUNTIME (LocalWorker + BroadcastChannel). Proves the architecture composes before adding distributed complexity.

**Topology:**
```
                ┌────────────┐
                │  planner   │  decides what to evaluate
                └─────┬──────┘
                      │ send(executor, Task)
                      ▼
                ┌────────────┐
                │  executor  │  runs eval, captures trace
                └─────┬──────┘
       ┌──────────────┼──────────────┐
       │ trace        │              │ remember(state)
       ▼              ▼              ▼
┌────────────┐  ┌────────────┐  ┌──────────────┐
│ ui-agent   │  │  verifier  │  │ SharedMem    │
│            │  │            │  │ (state log)  │
└─────┬──────┘  └─────┬──────┘  └──────────────┘
      │ synthesize     │ validate
      └──────┬─────────┘
             ▼ broadcast(ApprovedPatch)
       (all nodes apply)
```

**Events captured in log:** `MessageSent`, `MessageReceived`, `PatchApplied`, `MemoryWrite`, `VerificationPassed`, `EvalFailed`. Demo passes if log replays byte-identically.

### Phase 2: Distributed Transports (v0.23.0)

Adds:
- **WebSocket transport** — connects to `ailang serve` relay
- **FirestoreRelay transport** — durable cloud relay; can resume after disconnect
- **Vector clocks** — `Map[NodeId, Int]` per event for true partial ordering

```go
// Trait already shape-locked in M-COG-RUNTIME; new impls plug in:
type WebSocketTransport struct { ... }
type FirestoreRelayTransport struct { ... }
```

**Cross-origin auth model:**
```json
{
  "capability_token": "agent_x:hmac:...",
  "trusted_peers": ["agent_y:pubkey:...", "agent_z:pubkey:..."]
}
```
No ambient origin trust. Token rotation handled out-of-band.

### Topology Visualizer

In-browser panel built using `!: DOM` from M-COG-RUNTIME (dogfoods the runtime). Live graph of:
- Active nodes (boxes)
- Message flow (animated edges per event)
- Capability usage (per-node bars vs. budget)
- Event-log scrubber for replay

---

## Implementation Plan

### Phase 1: Collaborative 4-Agent Demo (~25h, v0.22.x)

- [ ] 4-agent demo modules: `planner.ail`, `executor.ail`, `ui_agent.ail`, `verifier.ail`
- [ ] Rule-based patch synthesizer in `ui_agent.ail` (deterministic for replay)
- [ ] `VerificationPassed`, `EvalFailed` cognitive event kinds
- [ ] Demo orchestrator harness (browser-side: `topology_demo.js`, location follows M-COG-RUNTIME)
- [ ] Topology visualizer (`stdlib/std/topology.ail` + browser-side `topology_panel.js`)
- [ ] End-to-end test: scenario produces byte-identical event log across 3 runs
- [ ] Demo documentation in `docs/docs/guides/cognitive-os.md`

### Phase 2: Distributed Transports (~20h+, v0.23.0)

- [ ] Vector clock upgrade (`internal/cognition/clock.go`)
- [ ] WebSocket transport (`internal/cognition/transport_websocket.go`)
- [ ] FirestoreRelay transport (`internal/cognition/transport_firestore.go`)
- [ ] `ailang serve` relay endpoint
- [ ] Capability token auth model
- [ ] Cross-device replay test: 4-agent demo across 2 devices, replay on 3rd
- [ ] `TransportError`, `NodeJoined`, `NodeLeft` cognitive event kinds
- [ ] Documentation: deployment topologies

### Files to Modify/Create

**New files (Phase 1):**
- `examples/cognitive_os/planner.ail` — planner agent (~100 LOC)
- `examples/cognitive_os/executor.ail` — executor agent (~150 LOC)
- `examples/cognitive_os/ui_agent.ail` — UI agent + rule-based synthesizer (~180 LOC)
- `examples/cognitive_os/verifier.ail` — verifier agent (~120 LOC)
- `stdlib/std/topology.ail` — topology API (~150 LOC)
- Browser-side `topology_panel.js` — visualizer (~400 LOC, location alongside M-COG-RUNTIME host code)
- Browser-side `topology_demo.js` — orchestrator (~250 LOC)

**New files (Phase 2, Go-side):**
- `internal/cognition/transport_websocket.go` — WebSocket transport (~300 LOC)
- `internal/cognition/transport_firestore.go` — FirestoreRelay transport (~350 LOC)
- `internal/cognition/transport_auth.go` — capability token auth (~200 LOC)
- `cmd/ailang/serve_relay.go` — relay endpoint for `ailang serve` (~250 LOC)

**New files (Phase 2, browser-side):**
- `transport_websocket.js` — browser-side WebSocket (~200 LOC)
- `transport_firestore.js` — browser-side FirestoreRelay (~250 LOC)

**Modified files:**
- `internal/cognition/clock.go` — add vector clocks (~150 LOC)
- `internal/cognition/event_log.go` — new event kinds (~100 LOC)
- `internal/cognition/manifest.go` — capability token field (~50 LOC)
- `docs/docs/guides/cognitive-os.md` — new guide
- `docs/docs/guides/wasm-runtime.md` — distributed section
- CHANGELOG

---

## Examples

### Example 1: 4-Agent Collaborative Self-Improvement (Phase 1)

```ailang
module demo/executor

import std/cognition (send, recv, NodeId)
import std/sharedmem (remember)

export func executor() -> Unit ! {Eval, Msg, SharedMem, Trace} {
  let result = runBenchmarks()
  remember(MemKey("last_eval"), serialize(result));
  if hasParseFailures(result.trace) then
    send(NodeId("ui-agent"), TraceReport { trace: result.trace })
}
```

```ailang
module demo/verifier

import std/cognition (recv, broadcast)

export func verifier() -> Unit ! Msg {
  let proposal = recv()
  if validatePatchShape(proposal.patch) &&
     withinCapability(proposal.patch) then {
    emit(VerificationPassed { claim: hash(proposal.patch) });
    broadcast(ApprovedPatch { patch: proposal.patch })
  }
}
```

### Example 2: Same Agent, Distributed Transport (Phase 2)

```ailang
-- executor.ail unchanged from Phase 1
```

```json
// manifest for Phase 2
{
  "module": "executor",
  "effects": ["Eval", "Msg", "SharedMem", "Trace"],
  "budgets": { "Msg.sends": 100, "SharedMem.writes": 50 },
  "transports": ["WebSocket", "FirestoreRelay"],
  "capability_token": "executor:hmac:...",
  "trusted_peers": ["ui-agent:pubkey:...", "verifier:pubkey:..."]
}
```

**Same source code. Different deployment.** Agent on laptop. UI agent on phone. Verifier in cloud. Event log replays anywhere.

---

## Success Criteria

### Phase 1 (v0.22.x)
- [ ] 4-agent demo runs end-to-end via LocalWorker + BroadcastChannel
- [ ] Demo produces byte-identical event log across 3 runs
- [ ] Rule-based patch synthesizer is deterministic
- [ ] Topology visualizer shows live graph + message flow + capability usage
- [ ] All prior demos (M-COG-RUNTIME + M-COG-MEMORY) still pass

### Phase 2 (v0.23.0)
- [ ] WebSocket transport runs 4-agent demo across 2 devices, zero source changes
- [ ] FirestoreRelay transport provides durable async cognition
- [ ] Vector clocks correctly order events across nodes
- [ ] Cross-device event log replays byte-identically on 3rd machine
- [ ] Capability token auth prevents unauthorized peers
- [ ] Documentation: deployment topologies guide

---

## Testing Strategy

**Unit:**
- Vector clock invariants (causality preserved, partial order correct)
- Transport trait conformance (each impl behaves identically at agent boundary)
- Capability token verification
- Event log: vector-clock events serialize round-trip

**Integration:**
- 4-agent demo: byte-identical event log across runs
- Cross-device demo: 2 devices + 3rd-machine replay
- Network-partition handling: disconnect → reconnect → resume from FirestoreRelay
- Negative: untrusted peer rejection

**Manual:**
- Topology visualizer in browser: scrub event log, see replay
- Cross-device demo: laptop + phone setup; visualizer shows distributed flow

---

## Deferred Decisions

- **Vector clock encoding** — agent choice; varint preferred
- **WebSocket reconnection backoff** — agent choice; exponential default
- **FirestoreRelay collection naming** — agent choice; must be deterministic from manifest
- **Demo agent prompt copy** — agent choice; documentation only
- **Topology panel visual design** — agent choice; deterministic node layout is constraint
- **Patch synthesizer rule set for demo** — agent choice; must be deterministic

---

## Non-Goals

- CRDT-based concurrent-write resolution (future)
- WebRTC / A2A / MCP transports (future, after WebSocket + FirestoreRelay prove out)
- Capability marketplaces (far future)
- Formal verification / SMT (far future)
- Native non-browser host (separate doc)
- LLM-driven patch synthesis (stretch for v0.23.x, not MVP)
- End-to-end encryption (future)

---

## Timeline

**Phase 1 — v0.22.x (Week 1, ~25h):** 4-agent demo + topology visualizer
**Phase 2 — v0.23.0 (Week 2+, ~20h+):** Vector clocks + WebSocket + FirestoreRelay + cross-device replay

**Total: ~45h+ across 2 weeks (Phase 2 may stretch)**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Vector clocks expensive in browser | Med | Profile early; fall back to Lamport for single-node scenarios |
| WebSocket reconnection breaks replay | High | FirestoreRelay as durable backstop; event log replay tolerates gaps |
| Capability token compromise | High | Rotation supported; short-lived tokens; manifest declares pubkeys |
| Cross-origin demo too fragile to ship | Med | Phase 1 single-origin proves architecture; Phase 2 strictly additive |
| Demo patch synthesizer non-deterministic | High | Rule-based for Phase 1; LLM only as stretch with seed pinning |
| FirestoreRelay vendor lock-in | Med | Trait-based; alternate cloud providers pluggable |
| Topology visualizer drains capability budget | Low | Visualizer runs in separate node with own budget |

---

## Related Documents

- **Parent**: [M-WASM-REFLECTIVE-RUNTIME (umbrella)](../v0_21_0/m-wasm-reflective-runtime.md)
- **Prev**: [M-COG-RUNTIME](../v0_21_0/m-cog-runtime.md) — provides effects + fabric + event log
- **Prev**: [M-COG-MEMORY](./m-cog-memory.md) — provides durable cross-agent state

## References

- [Design Axioms](/docs/references/axioms)
- [AILANG Agent Messaging Guide](../../../docs/docs/guides/agent-messaging.md)
- [Lamport, "Time, Clocks, and the Ordering of Events"](https://lamport.azurewebsites.net/pubs/time-clocks.pdf) — vector clock foundation
- [Firestore Realtime Updates](https://firebase.google.com/docs/firestore/query-data/listen) — durable async relay

---

**Document created**: 2026-05-18
**Last updated**: 2026-05-18

---

**DESIGN_DOC_PATH**: `design_docs/planned/v0_22_0/m-cog-mesh.md`
