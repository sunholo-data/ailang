# M-WASM-REFLECTIVE-RUNTIME: Cognitive OS (Umbrella)

**Status**: Planned (parent doc — execution lives in child sprints)
**Target**: v0.21.0 → v0.23.0 (multi-version arc)
**Priority**: P0 (Strategic Runtime Direction)
**Estimated**: ~165h total across 3 child sprints
**Dependencies**:
- Existing WASM evaluator/runtime ([m-wasm-dictionary-dispatch](../../implemented/v0_9_2/m-wasm-dictionary-dispatch.md), [m-wasm-closure-env](../../implemented/v0_8_1/m-wasm-closure-env.md), [m-wasm-stream-bridge](../../implemented/v0_8_1/m-wasm-stream-bridge.md))
- Effect system (`!: IO, FS, Net, ...`)
- AILANG messaging system (promoted to first-class semantic substrate)
- Reflection roadmap ([m-reflect-structural-reflection](../v0_13_0/m-reflect-structural-reflection.md))

---

## Why This Is an Umbrella

The original "reflective WASM runtime" design grew into a **Cognitive OS** spanning ~145–165h. Single sprint plans bog down past ~50h, and high-impact decisions stay locked too long. This doc captures the **strategic vision**; execution lives in three child design docs, each independently shippable:

| Child | Scope | Target | Estimate | Status |
|-------|-------|--------|----------|--------|
| **[M-COG-RUNTIME](../../implemented/v0_21_0/m-cog-runtime.md)** | Effects + Message Fabric + Event Log + Deterministic Scheduler (Go-side substrate) | v0.21.x | ~105h | **✅ Shipped** (13 commits, ~6,765 LOC, PASS @ 91/100) |
| **[M-COG-RUNTIME-BROWSER](./m-cog-runtime-browser.md)** | Browser host JS + canonical DOM + IndexedDB persistence + Subscribe wiring + Trace ext | v0.21.x | ~50h | **✅ Shipped** (~3,400 LOC, user-verified end-to-end, demo at ailang.sunholo.com/demos/cognitive-os-runtime/) |
| **[M-COG-MEMORY](../v0_22_0/m-cog-memory.md)** | `!: SharedMem` + `!: SemanticSearch` + IndexedDB | v0.22.0 | ~25h | Planned |
| **[M-COG-MESH](../v0_22_0/m-cog-mesh.md)** | Collaborative 4-agent demo + distributed transports | v0.22.x → v0.23 | ~45h+ | Planned |

Each child has its own axiom scoring, conflict-surface analysis, high-impact decisions, files-to-modify, success criteria, and sprint plan.

---

## Strategic Vision (the part that doesn't fit in any single sprint)

AILANG becomes a **Cognitive OS** — a deterministic substrate for distributed self-improving cognition.

| OS Primitive | AILANG Equivalent |
|--------------|-------------------|
| Process | Agent node (WASM instance) |
| IPC | AILANG messages (typed, replayable) |
| Filesystem | Semantic memory (`!: SharedMem`, `!: SemanticSearch`) |
| Scheduler | Deterministic single-threaded event loop |
| Permissions | Effect system + capability manifest |
| Kernel boundary | WASM sandbox |
| Logs | Structured traces + cognitive event log |
| Package manager | Cognitive components (WASM Component Model) |

### Key insight: runtime is transient, cognition is durable

| Runtime State | Cognitive State |
|---------------|-----------------|
| WASM memory | Semantic memory |
| Local worker | Distributed mesh |
| Active execution | Durable traces |
| Process state | Cognition history (event log) |

**Workers die. Tabs close. Browsers suspend. Phones disconnect.** Continuity lives in the **message fabric**, not the runtime instance.

### Architectural Stack

```
AILANG Source
    ↓
Canonical IR
    ↓
WASM Runtime Nodes (per-agent isolation)     ←── M-COG-RUNTIME builds this
    ↓
AILANG Message Fabric (transport-independent) ←── M-COG-RUNTIME builds this
    ↓
Semantic Memory                               ←── M-COG-MEMORY builds this
    ↓
Distributed Cognitive Mesh                    ←── M-COG-MESH builds this
```

---

## Umbrella Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Replayable runtime + canonical patches + deterministic scheduler + logical clocks |
| A2: Replayability | +2 | Full execution + UI replay + cognitive event log = distributed replay |
| A3: Effect Legibility | +2 | New `DOM`, `SharedMem`, `SemanticSearch`, `Msg` effects make cognition physical |
| A4: Explicit Authority | +2 | WASM host controls grants; transport abstraction makes authority transport-independent |
| A5: Bounded Verification | +2 | Structured patches + event log enable divergence detection |
| A6: Safe Concurrency | +2 | Isolated WASM cells; deterministic mailbox ordering |
| A7: Machines First | +2 | Cognitive OS framing — primitives for agents, not humans |
| A8: Minimal Syntax | +1 | 4 new effect labels; no new constructs |
| A9: Cost Visibility | +1 | Capability budgets observable per-node, aggregated in event log |
| A10: Composability | +2 | Transport-independent agents; component-model alignment |
| A11: Structured Failure | +1 | WASM traps → typed AILANG failures; event log preserves context |
| A12: System Boundary | +2 | Strong isolation; transport abstraction makes every boundary explicit |

**Net Score: +21**

### Hard Violation Check
- [x] A1, A3, A4, A7 all ≥ 0

Per-child scoring lives in each child doc.

---

## Sequencing & Critical Path

```
M-COG-RUNTIME (v0.21.x, ~95h)
    │
    │   delivers: effect plumbing, message fabric, event log, scheduler, DOM replay
    │   gates: M-COG-MEMORY needs the effect-plumbing + event-log foundation
    ▼
M-COG-MEMORY (v0.22.0, ~25h)
    │
    │   delivers: !: SharedMem + !: SemanticSearch + IndexedDB providers
    │   gates: M-COG-MESH demo needs durable memory for cross-agent state
    ▼
M-COG-MESH (v0.22.x → v0.23, ~45h+)

    delivers: collaborative 4-agent demo, distributed transports
```

**Each phase has independent shippable value:**
- After M-COG-RUNTIME: single-agent WASM with replayable DOM mutations + cross-tab messaging
- After M-COG-MEMORY: durable cognition that survives runtime death
- After M-COG-MESH: multi-agent collaborative self-improvement across machines

---

## Cross-Cutting High-Impact Decisions

Decisions that affect multiple children — must be aligned across docs:

| Decision | Resolved In | Why Cross-Cutting |
|----------|-------------|-------------------|
| Effect-label vocabulary (`DOM`, `SharedMem`, `SemanticSearch`, `Msg`) | M-COG-RUNTIME | All three docs use these names |
| Capability manifest schema (incl. transport list) | M-COG-RUNTIME | M-COG-MESH extends with more transports |
| Cognitive event log schema | M-COG-RUNTIME | M-COG-MEMORY adds memory events; M-COG-MESH adds distributed events |
| Logical-clock scheme (Lamport → vector) | M-COG-RUNTIME (Lamport), M-COG-MESH (vector) | Must be forward-compatible |
| Transport trait surface | M-COG-RUNTIME (LocalWorker + BroadcastChannel), M-COG-MESH (WebSocket, etc.) | Trait designed once, implementations added |

---

## Demo Vision (post-mesh)

> **An agent improves its own harness across two devices.**

1. Agent on laptop runs evals, fails on parse errors
2. Failure message + trace flow across WebSocket transport to laptop's verifier
3. UI agent (in browser tab on phone) synthesizes a `errorHeatmap` panel patch
4. Verifier on laptop validates patch shape + capability
5. Patch broadcast over Firestore relay to all connected nodes
6. Both devices apply patch deterministically; event log records every step
7. Re-run on laptop; phone visualizes the heatmap in real-time

This demonstrates: reflection + constrained self-improvement + deterministic cognitive tooling + safe runtime evolution + distributed cognition.

---

## Non-Goals (umbrella-level)

- **Not a general browser framework** — not a React replacement
- **Not arbitrary self-modification** — agents may not execute raw JS or escape capability boundaries
- **Not formal verification / SMT** — capability gating is the verification, not theorem proving

---

## Risks (umbrella-level)

| Risk | Mitigation |
|------|------------|
| Children diverge on effect/manifest schemas | This umbrella doc owns cross-cutting decisions; children reference it |
| Scope creep within children | Each child has independent shippable value; can defer mesh entirely if needed |
| Velocity assumptions wrong | Children are independently estimable; re-baseline after M-COG-RUNTIME completes |

---

## Future Extensions (beyond v0.23)

- CRDT-based collaborative agent runtimes
- Formal verification of UI mutations
- Distributed WASM cognitive clusters
- Capability marketplaces
- Browser-native autonomous IDEs
- Cognitive event log as Merkle DAG (content-addressed cognition)

---

## Related Documents

**Children (execution):**
- [M-COG-RUNTIME](./m-cog-runtime.md) — effects + fabric + event log + scheduler + DOM replay
- [M-COG-MEMORY](../v0_22_0/m-cog-memory.md) — SharedMem + SemanticSearch
- [M-COG-MESH](../v0_22_0/m-cog-mesh.md) — collaborative demo + distributed transports

**Implemented (foundation):**
- [m-wasm-dictionary-dispatch](../../implemented/v0_9_2/m-wasm-dictionary-dispatch.md)
- [m-wasm-closure-env](../../implemented/v0_8_1/m-wasm-closure-env.md)
- [m-wasm-stream-bridge](../../implemented/v0_8_1/m-wasm-stream-bridge.md)

**Planned (related):**
- [m-perf4-bytecode-interpreter](../v1_0_0/m-perf4-bytecode-interpreter.md) — complementary execution path
- [m-reflect-structural-reflection](../v0_13_0/m-reflect-structural-reflection.md) — dependency

## References

- [Design Axioms](/docs/references/axioms)
- [Philosophical Foundations](/docs/references/philosophical-foundations)
- [WASM Component Model](https://component-model.bytecodealliance.org/)
- [AILANG Agent Messaging Guide](../../../docs/docs/guides/agent-messaging.md)

---

**Document created**: 2026-05-18
**Last updated**: 2026-05-18 (v3 — refactored from monolithic doc into umbrella + 3 children)

---

**DESIGN_DOC_PATH**: `design_docs/planned/v0_21_0/m-wasm-reflective-runtime.md`
