# M-COG-MEMORY: Semantic Memory — `!: SharedMem` + `!: SemanticSearch`

**Status**: Planned
**Target**: v0.22.0
**Priority**: P0
**Estimated**: ~25h across 1 week
**Parent**: [M-WASM-REFLECTIVE-RUNTIME (Cognitive OS umbrella)](../v0_21_0/m-wasm-reflective-runtime.md)
**Dependencies**:
- **[M-COG-RUNTIME](../v0_21_0/m-cog-runtime.md)** — must ship first (provides effect plumbing + event log to extend)
- Existing AILANG embedding/semantic-search infrastructure (used in `ailang docs search --neural`)

---

## Scope

This is **Child 2 of 3** of the Cognitive OS arc. It adds the **filesystem analog**: durable cognitive state via two new effects.

**What this doc covers:**
- `!: SharedMem` effect — key/value cognitive memory with `remember` / `recall` / `link`
- `!: SemanticSearch` effect — `query` / `index` over embedding store
- IndexedDB provider (browser) + pluggable provider interface
- New cognitive events: `MemoryWrite`, `MemoryRead`, `SemanticQuery`, `MemoryLink`
- Memory replay: event log can reconstruct memory state at any point

**What this doc explicitly defers:**
- Firestore / cloud providers (→ [M-COG-MESH](./m-cog-mesh.md), needs WebSocket transport)
- Distributed memory consistency (→ M-COG-MESH)
- CRDT merge for concurrent writes (future work)

**Shippable deliverable:** at end of v0.22.0, an AILANG agent can persist cognitive state across browser restarts. Kill the tab mid-task; resume in a new tab from the event log; `recall(TaskState)` returns the last checkpoint.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | Memory reads/writes deterministically ordered via Lamport clock |
| A2: Replayability | +2 | Memory state fully reconstructible from event log |
| A3: Effect Legibility | +2 | Two new effects — memory ops are physical, not implicit globals |
| A4: Explicit Authority | +2 | `SharedMem` budget in manifest; agents without import can't touch memory |
| A5: Bounded Verification | +1 | Content-hashed values enable divergence detection |
| A6: Safe Concurrency | +1 | Reads/writes serialized through scheduler; no concurrent-write CRDT (yet) |
| A7: Machines First | +2 | Filesystem-of-cognition primitive, not human-facing storage |
| A8: Minimal Syntax | +1 | 2 new effect labels; no new constructs |
| A9: Cost Visibility | +1 | Memory + query budgets in manifest |
| A10: Composability | +2 | Pluggable provider interface; same agent code over IndexedDB / file / Firestore |
| A11: Structured Failure | +1 | `MemoryFull` / `QueryFailed` typed failures |
| A12: System Boundary | +2 | Provider boundary explicit; capability gated |

**Net: +19** → ✅ Proceed

### Hard Violation Check
- [x] A1, A3, A4, A7 all ≥ 0

---

## Problem Statement

After M-COG-RUNTIME ships, agents have ephemeral runtime state but no durable cognitive state:
- Kill the browser tab → lose all reasoning
- Can't checkpoint long tasks
- Can't share learned context between agents
- Can't query past traces semantically

This is the filesystem-analog gap in the Cognitive OS framing.

---

## Goals

**Primary:** Add the durable cognitive state layer — `!: SharedMem` and `!: SemanticSearch` — so cognition survives runtime instances.

**Success Metrics:**
- Program with `!: SharedMem` can `remember` and `recall` across tab restarts
- Program with `!: SemanticSearch` can index AILANG values and query by embedding similarity
- Memory state reconstructible from event log at any point in time
- Provider interface lets the same agent code run over IndexedDB or file backend
- Budget overrun on `SharedMem.writes` or `SemanticSearch.queries` → typed failure + event
- Memory ops appear in event log with Lamport clock, deterministically replayable

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Semantic indexing strategy (eager / lazy / agent-controlled) | Cost vs. recall trade-off | human | design | high |
| Memory key namespace (flat / hierarchical / agent-scoped) | Cross-agent sharing semantics | human | design | high |
| Embedding model surface (provider-supplied / agent-supplied) | Determinism across providers | human | design | high |
| `recall` failure semantics (`Option` vs. typed `MemoryError`) | Ergonomic call sites | human | compile | med |
| Memory event compaction policy (none v1 / snapshot every N events) | Replay performance | human | compile | med |
| Pluggable provider interface (Go trait shape) | Future provider additions | agent | compile | low |
| IndexedDB store schema | Forward-migration | agent | compile | low |

### Design Freeze

- [ ] Semantic indexing = **agent-controlled** (explicit `index` calls; no implicit indexing on `remember`)
- [ ] Memory key namespace = **agent-scoped by default**, opt-in shared via explicit `SharedScope` type
- [ ] Embedding model = **provider-supplied, declared in manifest** (agent code is model-agnostic)
- [ ] `recall` returns **`Option[Frame]`**; missing keys are not errors
- [ ] Compaction = **none in v0.22.0** (snapshotting → future)

---

## Conflict Surface

**Path note:** follows the same execution-model conventions as [M-COG-RUNTIME](../v0_21_0/m-cog-runtime.md) — no new codegen subtree; effects live in [internal/effects/](../../../internal/effects/) with `_js.go` build-tagged variants, cognition primitives in [internal/cognition/](../../../internal/cognition/), WASM glue in [cmd/wasm/effects.go](../../../cmd/wasm/effects.go).

Touches [internal/effects/](../../../internal/effects/), `internal/cognition/event_log.go` (extends with memory events), `cmd/wasm/effects.go` (new JS callbacks), and the existing embedding code path in [internal/messaging/embedder*.go](../../../internal/messaging/).

### Positions extended
1. **Effect rows** — adding `SharedMem`, `SemanticSearch` labels (alongside `DOM`/`Msg`/`Trace` from M-COG-RUNTIME)
2. **WASM imports** — new namespaces `host.mem.*`, `host.semsearch.*`
3. **Cognitive event log** — new event kinds: `MemoryWrite`, `MemoryRead`, `MemoryLink`, `SemanticQuery`
4. **Capability manifest** — new budget keys: `SharedMem.writes`, `SharedMem.reads`, `SemanticSearch.queries`
5. **Existing semantic-search infra** — runtime API wraps it; CLI behavior unchanged

### Existing constructs in these positions
- Effect rows: continues additive pattern from M-COG-RUNTIME
- WASM imports: namespaced; no collision
- Event log: M-COG-RUNTIME established the schema; new event kinds are additive
- Manifest: new budget keys follow existing `Net.requests` / `DOM.patches` pattern
- Semantic-search infra: `ailang docs search --neural` continues to work byte-identically

### Programs that MUST still work
1. All M-COG-RUNTIME-era programs (pure / `!: IO` / `!: DOM` / `!: Msg` / `!: Trace`)
2. `ailang docs search --neural` CLI byte-identical
3. Existing embedding pipeline tests pass
4. All M-COG-RUNTIME demo examples replay identically (memory events absent = no-op)

### Deliberately changed
- Programs using `!: SharedMem` / `!: SemanticSearch` require `host.mem.*` / `host.semsearch.*` imports — link failure otherwise
- Event log gains new event types; M-COG-RUNTIME replayers skip unknown (forward-compat)

---

## Solution Design

### `!: SharedMem` Effect

```ailang
remember : (key: MemKey, frame: Frame) -> Unit ! SharedMem
recall   : (key: MemKey) -> Option[Frame] ! SharedMem
link     : (a: MemKey, b: MemKey, kind: LinkKind) -> Unit ! SharedMem
```

Backed by pluggable provider (IndexedDB browser default; file backend for native). Provider interface:

```go
type MemoryProvider interface {
    Write(key MemKey, value []byte) error
    Read(key MemKey) ([]byte, bool, error)
    Link(a, b MemKey, kind LinkKind) error
    Name() string
}
```

### `!: SemanticSearch` Effect

```ailang
index : (key: MemKey, embedding: Embedding) -> Unit ! SemanticSearch
query : (q: Query, k: Int) -> List[(MemKey, Similarity)] ! SemanticSearch
```

Embedding produced by provider-declared model (manifest declares model ID + dims). Agent code is model-agnostic.

### Memory Replay

Memory state at any clock-tick is reconstructible by replaying `MemoryWrite` + `MemoryLink` events up to that point. New event types:

```ailang
type CognitiveEvent = ...
  | MemoryWrite   { node: NodeId, key: MemKey, valueHash: Hash, clock: Lamport }
  | MemoryRead    { node: NodeId, key: MemKey, valueHash: Hash, clock: Lamport }
  | MemoryLink    { node: NodeId, a: MemKey, b: MemKey, kind: LinkKind, clock: Lamport }
  | SemanticQuery { node: NodeId, query: Query, resultIds: List[MemKey], clock: Lamport }
```

Replay: events fed in clock order; provider rebuilds memory state. `recall` calls during replay return the value-hash recorded in the original log (not a re-fetch).

---

## Implementation Plan

### Phase 1: Effect Plumbing (~6h)

- [ ] Add `SharedMem`, `SemanticSearch` to effect row vocabulary
- [ ] Define `MemKey`, `Frame`, `LinkKind`, `Embedding`, `Query` types
- [ ] Wire `host.mem.*`, `host.semsearch.*` import declarations
- [ ] Extend capability manifest with new budgets
- [ ] Negative tests: missing import → link failure

### Phase 2: Memory Provider + IndexedDB (~10h)

- [ ] `MemoryProvider` trait (`internal/effects/memory_provider.go`)
- [ ] IndexedDB browser provider (browser-side: `memory_indexeddb.js`, location TBD per M-COG-RUNTIME Phase 3)
- [ ] File provider for native host (`internal/effects/memory_file.go`)
- [ ] `stdlib/std/sharedmem.ail` — effect bindings
- [ ] Memory event types extending event log (`internal/cognition/event_log.go`)
- [ ] Persistence-across-restart test (kill + resume)

### Phase 3: Semantic Search Integration (~6h)

- [ ] `SemanticSearchProvider` trait
- [ ] Wire to existing embedding pipeline (reuse, don't fork)
- [ ] `stdlib/std/semsearch.ail` — effect bindings
- [ ] Browser provider: `host.semsearch.*` over existing embedding store
- [ ] Determinism contract: provider must return ordered results

### Phase 4: Replay + Tests (~3h)

- [ ] Memory state reconstruction from event log
- [ ] Replay test: same events → same memory state across runs
- [ ] Demo example: `examples/wasm_reflective/persistent_task.ail`
- [ ] Docs + CHANGELOG

### Files to Modify/Create

**New files (Go-side):**
- `internal/effects/sharedmem.go` + `sharedmem_js.go` — SharedMem effect handler + JS bridge (~300 LOC total)
- `internal/effects/semsearch.go` + `semsearch_js.go` — SemanticSearch effect handler + JS bridge (~300 LOC total)
- `internal/effects/memory_provider.go` — provider trait + dispatch (~250 LOC)
- `internal/effects/memory_file.go` — file backend for native host (~200 LOC)

**New files (browser-side, alongside M-COG-RUNTIME host code):**
- `memory_indexeddb.js` — IndexedDB SharedMem provider (~300 LOC)
- `semsearch_browser.js` — browser semantic search provider (~250 LOC)

**New files (stdlib + examples):**
- `stdlib/std/sharedmem.ail` — SharedMem bindings (~150 LOC)
- `stdlib/std/semsearch.ail` — SemanticSearch bindings (~150 LOC)
- `examples/wasm_reflective/persistent_task.ail` — demo (~120 LOC)

**Modified files:**
- `internal/effects/ops.go` + `capability.go` — register `SharedMem`, `SemanticSearch` ops + budgets (~80 LOC)
- `internal/types/` — add `SharedMem`, `SemanticSearch` effect labels (~30 LOC)
- `internal/cognition/manifest.go` — new budget keys (~50 LOC)
- `internal/cognition/event_log.go` — extend with memory events (~100 LOC)
- `cmd/wasm/effects.go` — wire `host.mem.*` + `host.semsearch.*` JS callbacks (~150 LOC)
- Browser host shim — wire memory + semsearch handlers (~150 LOC, extends M-COG-RUNTIME `host.js`)
- Browser replay engine — memory event replay (~100 LOC, extends M-COG-RUNTIME `replay.js`)
- `docs/docs/guides/wasm-runtime.md` — semantic memory section
- CHANGELOG

---

## Examples

### Example 1: Resumable Task Across Browser Restart

```ailang
module demo/persistent_task

import std/sharedmem (remember, recall)
import std/cognition (send, NodeId, self)

type TaskState = Started | InProgress(Int) | Complete

export func resumableTask() -> Unit ! {SharedMem, Msg} {
  let state = recall(MemKey("task_state")) match
    | Some(s) -> deserialize(s)
    | None    -> Started

  let next = step(state)
  remember(MemKey("task_state"), serialize(next))
  if not isComplete(next) then
    send(self(), Continue)
}
```

Kill the tab → IndexedDB persists event log → new tab replays → `recall` returns last state → task resumes.

### Example 2: Semantic Recall Across Sessions

```ailang
module demo/semantic_recall

import std/sharedmem (remember)
import std/semsearch (index, query)

export func learnFromTrace(t: Trace) -> Unit ! {SharedMem, SemanticSearch} {
  let key = MemKey(hash(t))
  remember(key, serialize(t));
  index(key, embed(summary(t)))
}

export func recallSimilar(q: Query) -> List[Trace] ! {SharedMem, SemanticSearch} {
  let hits = query(q, k = 5)
  hits |> map(fn (k, _sim) => recall(k))
       |> filterMap(identity)
}
```

---

## Success Criteria

- [ ] `SharedMem`, `SemanticSearch` effects typecheck through row inference
- [ ] `!: SharedMem` programs persist state across tab restarts (IndexedDB)
- [ ] `!: SemanticSearch` programs index + query embeddings deterministically
- [ ] Memory state reconstructible from event log
- [ ] Pluggable provider interface: same agent over IndexedDB or file backend
- [ ] Budget enforcement: overrun → typed failure + event
- [ ] Existing `ailang docs search --neural` CLI byte-identical
- [ ] All M-COG-RUNTIME demos still replay identically
- [ ] CHANGELOG + docs updated
- [ ] Demo example added

---

## Testing Strategy

**Unit:**
- Effect row inference (positive + negative)
- Provider trait conformance tests
- Memory event log serialization round-trip
- Embedding determinism (same input → same vector)

**Integration:**
- WASM module with `!: SharedMem` runs against IndexedDB provider
- Persistence-across-restart: write → kill → restart → read
- Replay reconstructs memory state byte-identically
- Behavior-equivalence: `ailang docs search --neural` CLI before/after

**Manual:**
- Demo example in browser: persistent task survives tab kill

---

## Deferred Decisions

- **Memory event compaction strategy** — agent choice; v0.22.0 ships uncompacted
- **IndexedDB store schema** — agent choice; must support forward migration
- **`MemKey` byte encoding** — agent choice; deterministic ordering is constraint
- **Link kinds enum membership** — agent may add common kinds (`Causes`, `Refines`, `Contradicts`)
- **Embedding-store on-disk format** — provider-internal; opaque to agent code

---

## Non-Goals

- Distributed memory consistency / CRDT merge (future)
- Firestore or cloud providers (→ M-COG-MESH, needs WebSocket)
- Memory compaction / snapshotting (future)
- Cross-origin memory sharing (security model unresolved)
- Encrypted memory at rest (future)

---

## Timeline

**Days 1–2 (~6h):** Phase 1 — effect plumbing
**Days 3–4 (~10h):** Phase 2 — memory provider + IndexedDB
**Day 5 (~6h):** Phase 3 — semantic search integration
**Day 6 (~3h):** Phase 4 — replay + tests + demo

**Total: ~25h across 1 week**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| IndexedDB quota exceeded | Med | Configurable retention policy; export-to-file fallback |
| Embedding model nondeterminism breaks replay | High | Manifest declares model + version; provider conformance test |
| Provider trait surface drifts after first impl | Med | Lock trait shape after IndexedDB ships; deprecate via wrappers |
| Memory event log inflates beyond M-COG-RUNTIME budget | Med | Per-effect budget separate from total event log size |
| `recall` semantics surprise agent authors | Low | `Option`-returning; document explicit "missing key is not an error" |

---

## Related Documents

- **Parent**: [M-WASM-REFLECTIVE-RUNTIME (umbrella)](../v0_21_0/m-wasm-reflective-runtime.md)
- **Prev**: [M-COG-RUNTIME](../v0_21_0/m-cog-runtime.md) — provides effect + event log infrastructure
- **Next**: [M-COG-MESH](./m-cog-mesh.md) — uses durable memory for cross-agent state

**Related infra:**
- AILANG semantic-search / embedding pipeline (reused, not forked)
- [m-doc-sem-lazy-embeddings](../../implemented/v0_6_0/m-doc-sem-lazy-embeddings.md) — embedding-store precedent

## References

- [Design Axioms](/docs/references/axioms)

---

**Document created**: 2026-05-18
**Last updated**: 2026-05-18

---

**DESIGN_DOC_PATH**: `design_docs/planned/v0_22_0/m-cog-memory.md`
