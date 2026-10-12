# CoreTypeInfo cache blob: subtree-shared encoding (motoko `session.ail` 188 MB → under 1 MB)

**Status**: Implemented (v0.54.7, #1788)
**Target**: v0.54.7
**Priority**: P0 (every motoko start pays a 70–130 s recompile; Cloud Run jobs always do)
**Estimated**: 1.5 days
**Dependencies**: None. Supersedes the CoreTI-dedup half of [m-compile-cache-large-module-blobs](../v0_52_2/m-compile-cache-large-module-blobs.md) (its configurable-ceiling half stays separate).
**Issue**: #1778 (triage: #1779; the triage's suspect commit and fix are corrected below)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Encoder walks node IDs in sorted order and map keys sorted, so the blob is byte-deterministic for a fixed CoreTypeInfo (today's gob of a Go map is not) |
| A2: Replayability | 0 | No execution-semantics change; a cache hit decodes to a value `reflect.DeepEqual` to what today's gob round trip yields |
| A3: Effect Legibility | 0 | No effect surface touched; effect rows round-trip with all metadata (budgets, params, provenance) |
| A4: Explicit Authority | 0 | Stamp authorization (payload digests, module ID, cache key) unchanged |
| A5: Bounded Verification | +1 | Ceilings stay; the blob they bound shrinks by >100× instead of the ceiling being raised again (third time: #1328 raised 16/32→64 MiB) |
| A6: Safe Concurrency | 0 | Cache write path and locking unchanged |
| A7: Machines First | +1 | Agents running motoko (rig and Cloud Run) stop paying ~2 min of model-idle startup per run |
| A8: Minimal Syntax | 0 | No language change |
| A9: Cost Visibility | +1 | Guard test reports encoded bytes and node counts, so the next type-expansion change shows up as a number in CI |
| A10: Composability | 0 | Cache-layer only |
| A11: Structured Failure | 0 | Encode failures keep today's behavior (CACHE_WRITE_FAILED + fresh compile); malformed blobs fail decode with ARTIFACT_INVALID |
| A12: System Boundary | 0 | No boundary change |
| **Net** | **+4** | |

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (removes some)
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

motoko's `src/core/session.ail` (6,339 lines, unchanged since 2026-09-21) stopped compile-caching between 2026-10-08 and v0.54.5. Every motoko session log starts with `CACHE_WRITE_FAILED … coretypeinfo.gob: ARTIFACT_TOO_LARGE`, and every run recompiles.

**Measured (2026-10-12, motoko tree `mk-20261002`; each build has `maxArtifactBlobBytes` raised to 1 GiB so the blob is written; `git bisect run` between 5ab23e89a and accdfc4bf~1):**

| build | coretypeinfo.gob | fresh `check` (6 parallel runs) |
|---|---|---|
| 5ab23e89a (10-08) … 4787024d9 | 29.3–29.4 MB | 34 s |
| **0295c49b4 (#1706, M-ALIAS-BODY-CLOSURE)** — first bad | **188.2 MB** | — |
| accdfc4bf~1, v0.54.5 | 188.2 MB | 83 s |

The triage doc (#1779) named accdfc4bf (#616); its parent is already 188 MB, so it is not the cause.

**Mechanism.** `closeInterfaceTypes` (`internal/pipeline/alias_body_closure.go`) publishes every nullary alias as its fully expanded structural body, recursively, where it used to publish the bare `TCon`. Expansion is memoized, so in memory these are shared pointers. `encoding/gob` does not preserve pointer sharing: CoreTypeInfo (`map[uint64]Type`, `internal/types/typeinfo.go:46`) writes the whole expansion out again under every node that mentions the alias.

**Blob anatomy (decoded with the production codec):**

| | good (5ab23e89a) | bad (v0.54.5) |
|---|---|---|
| CoreTypeInfo entries | 17,314 | 17,314 |
| total type nodes | 701,018 | 5,865,264 (8.4×) |
| largest entry | 1,424 nodes | 17,724 nodes |
| `ScriptedTool` record expansions | 3,388 | 96,664 |
| distinct whole types | 4,187 | 4,622 |
| **distinct subtrees (hash-consed)** | **1,456** | **2,398** |

Two conclusions follow, both measured:
1. **Whole-type dedup does not work.** This is what #1779 and m-compile-cache-large-module-blobs propose. A table of distinct whole types plus a node→index map encodes to 187.0 MB, down from 188.2 MB. The size is *inside* each type (about 40 KB per distinct type), not in repeats across nodes.
2. **Subtree sharing works.** 5.87M nodes are 2.4k distinct subtrees. The pre-regression blob was already more than 99% redundant too.

**Impact:** every motoko run on the rig and in the cloud image, plus any large AILANG program that imports record aliases. Even the pre-regression 29 MB was paid on every cache read.

## Goals

**Primary Goal:** Encode CoreTypeInfo as a hash-consed DAG, so each distinct type subtree is written once and the blob size tracks distinct structure rather than expanded size.

**Success Metrics:**
- motoko `session.ail` CoreTypeInfo blob: 188 MB to under 2 MB, and the module caches (no CACHE_WRITE_FAILED)
- Warm `ailang check src/core/session.ail` (cache hit) in a few seconds rather than 70–130 s
- The decode of every encoded CoreTypeInfo is `reflect.DeepEqual` to today's gob round trip of the same value (equivalence corpus: the std modules plus examples)
- A guard test fails CI if a fixture with nested imported record aliases encodes to more than a fixed budget

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix the encoding, not the ceiling | Raising to 256 MiB "works" but reads and decodes 188 MB per start and grows with every alias | agent (measured above) | design | low |
| Hash-cons subtrees (DAG), not whole types | Whole-type dedup measured at 0.6% savings | agent (measured) | design | low |
| Decoded types share pointers | A consumer that mutates a CoreTypeInfo type in place would corrupt siblings | agent; verified below (V5) | design | med |
| Equivalence oracle = today's gob round trip, not the in-memory value | gob already normalizes (empty map → nil, empty slice → nil); matching gob is what keeps cache-hit behavior identical to today | agent | design | low |
| Unknown `types.Type` implementors fall back to an embedded gob value | A missed variant must degrade to today's encoding, never to silent loss | agent | design | low |
| `cacheKeyVersion` v6 → v7 | The old blobs must never be decoded by the new codec | agent | compile | low (coordinate with #1771, which also bumps; the second to merge takes the next number) |

### Design Freeze

No human-ratified items: every decision above is internal to the cache format and backed by a measurement or a code read. The quorum review is skipped because none of the four triggers fires: there are no freeze items, the change is localized to `internal/pipeline` cache codec files, there's no cost/banking/schema surface, and all premises are in-repo and re-measurable.

## Solution Design

### Overview

Replace `encodeCoreTI`/`decodeCoreTI` in `productionCacheArtifactCodec` (`internal/pipeline/cache_artifacts.go:118,130`) with a DAG codec.

```go
type coreTIBlob struct {
    Format uint8      // 1
    Nodes  []coreTINode // post-order: every child index < parent index
    IDs    []uint64   // sorted Core node IDs
    Roots  []int32    // Roots[i] = node ref for IDs[i]
}

type coreTINode struct {
    Tag        uint8   // which types.Type variant
    Name       string  // TCon/TVar/TVar2/RowVar name; TRecord TypeName
    Kind       types.Kind
    A, B       int32   // fixed children (refs: index+1, 0 = nil)
    Kids       []int32 // list children (TTuple elems, TApp args, TFunc2 params, record/row fields)
    Keys       []string // sorted field/label names, parallel to Kids where the variant is a map
    Flags      uint8   // TFunc2 Implicit/ConcreteEffectContract
    Label      types.Label
    Budgets, MinBudgets map[string]*int
    Params     map[string]map[string]string
    Provenance map[string]ast.Span
    Raw        types.Type // fallback for an implementor not in the switch
}
```

**Encode:** walk each root. A pointer-identity memo (`map[types.Type]int32`) means a shared pointer is visited once. Each node's children are encoded first, then the node is keyed by its tag, scalars, metadata and child refs, and looked up in a hash-cons table, so structurally equal subtrees from *different* pointers also collapse. The result is gob-encoded once.

**Decode:** one pass over `Nodes` in index order, building each value from already-built children. Maps are created only when non-empty and slices only when non-empty, matching gob's normalization.

**Nil fidelity:** a nil interface child is ref 0. A nil `*Row`/`*RowVar` in a concrete-typed field (`TFunc2.EffectRow`, `TRecord2.Row`, `Row.Tail`) is ref 0. A *typed nil inside a `Type` interface* is unencodable by gob today ("cannot encode nil pointer"), and the DAG encoder returns an error in the same case, so the cache-write outcome is unchanged.

### Architecture

Per-variant field mapping (the 15 `Substitute` implementors in `internal/types`):

| Variant | Fields used |
|---|---|
| TCon, TVar | Name |
| TVar2, RowVar | Name, Kind |
| TList, TArray | A = Element |
| TMap | A = Key, B = Value |
| TTuple | Kids = Elements |
| TRecord | Keys/Kids = Fields, A = Row, Name = TypeName |
| TRecordOpen | Keys/Kids = Fields, A = Row |
| TRecord2 | A = Row (*Row) |
| TApp | A = Constructor, Kids = Args |
| TFunc2 | Kids = Params, A = Return, B = EffectRow (*Row), Flags |
| TLabelled | A = Inner, Label |
| Row | Kind, Keys/Kids = Labels, A = Tail (*RowVar), Budgets, MinBudgets, Params, Provenance |
| other | Raw (gob interface value, today's behavior) |

Two tests keep the mapping complete:
- **Source enumeration** (copied from `TestAliasBodyClosure_AllTypeVariants`): every `Substitute` receiver in `internal/types` has an encode case.
- **Field enumeration via reflection**: each variant struct's field count is pinned, so adding a field to `Row` without encoding it fails the test instead of silently dropping data.

### Implementation Plan

**M1: DAG codec (`internal/pipeline/cache_coreti_dag.go`, ~250 LOC)**
- `encodeCoreTIDAG(types.CoreTypeInfo) ([]byte, error)`, `decodeCoreTIDAG([]byte) (types.CoreTypeInfo, error)`
- Unit tests: per-variant round trip; nil and typed-nil handling; shared-subtree collapse (a 10k-entry CoreTI of one nested record encodes to under 64 KB); determinism (two encodes are byte-identical); malformed input (child ref ≥ own index, unknown tag, truncated) returns an error and never panics; source and field enumeration.

**M2: Wire in plus equivalence and guard**
- `productionCacheArtifactCodec` uses the DAG codec; `cacheKeyVersion` v7; update the four `"v6"` test seeds.
- Equivalence corpus test: compile the std modules and a set of examples, then for each module assert `reflect.DeepEqual(decodeDAG(encodeDAG(cti)), gobRoundTrip(cti))`.
- Guard test (#1778 ask 3): a two-module fixture where module A has a 6-deep chain of record aliases and module B uses them across ~200 functions. Compile it, read the written `coretypeinfo.gob`, assert it is under a fixed budget, and assert at least 10× smaller than the plain-gob encoding of the same CoreTI.

**M3: Real-world verification plus follow-up**
- motoko `session.ail`: blob size and cold/warm `check` time, recorded in the implementation report.
- The fresh-compile slowdown (34 s → 83 s at the same commits) is **not** fixed here. File a follow-up issue with a CPU profile of a fresh `check`.

### Files to Modify/Create

- `internal/pipeline/cache_coreti_dag.go` (new, ~250 LOC): codec
- `internal/pipeline/cache_coreti_dag_test.go` (new, ~300 LOC): unit, enumeration, equivalence, guard
- `internal/pipeline/cache_artifacts.go` (~6 LOC): codec wiring; comment on lines 26–32 updated
- `internal/pipeline/cache_key.go` (1 LOC): v7
- `internal/pipeline/cache_invalidation_test.go`, `internal/pipeline/alias_body_closure_runtime_test.go`, `cmd/ailang/serve_api_mcp_surface_test.go`: v6 → v7 seeds
- `changelogs/unreleased/2026-10-12-coreti-shared-encoding.md` (new)

## Conflict Surface

Not a parser/typechecker change, but the codec sits on every cache hit, so:
1. **Positions extended:** the `coretypeinfo.gob` payload format only. `core.gob`, `iface.json` and `constructors.json` are unchanged.
2. **Other readers of the blob:** only `readModuleArtifacts` → `decodeCoreTI` (`cache_artifacts.go:338`). No external tool reads it (V4).
3. **Old blobs:** never decoded by the new codec, because `cacheKeyVersion` v7 invalidates every v6 entry (the existing migration test covers the manifest path).
4. **Must still work:** `TestCompileCacheClear_Artifacts`, `TestCacheArtifacts_*`, `TestAliasBodyClosure_*` (cache-hit runtime), `cmd/ailang` cache tests, and `make test-core`.
5. **Deliberate change:** decoded CoreTypeInfo types share pointers where today every entry is a private tree (V5).

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | 0295c49b4 is the first commit producing the 188 MB blob | `git bisect run` with a size probe (6 steps, 29.3 MB vs 188.2 MB) | Confirmed |
| V2 | Whole-type dedup saves under 1% | Decode the 188 MB blob, dedup by per-value gob bytes, re-encode | 188.2 → 187.0 MB. Confirmed |
| V3 | Subtree hash-consing collapses to about 2.4k nodes | Hash-cons walk over the decoded blob | 2,398 (bad) / 1,456 (good). Lower bound: the key ignores budgets and provenance |
| V4 | The CoreTI blob is read only via `decodeCoreTI` | `grep -rn artifactCoreTIName internal/` | Only `cache_artifacts.go:189` (write) and `:338` (read) |
| V5 | No pipeline consumer mutates a CoreTypeInfo type in place | grep for field writes (`.Fields[..]=`, `.Labels[..]=`, `.TypeName=`, `.Element=`, …) in internal/{pipeline,eval,vm,codegen,link,iface} | 4 sites, all on clones/copies: `validate_effects_rows.go:24,32,103`, `alias_body_closure.go:37`, `iface/json.go:163` |
| V6 | 15 concrete `types.Type` implementors, all gob-registered | `grep "^func (.*) Substitute(" internal/types/*.go`; `internal/types/gob.go` | 15/15 |
| V5b | Row metadata maps (`Budgets`, `MinBudgets`, `Params`, `Provenance`) are shared between decoded rows; nothing writes into them in place | grep for map writes and `delete(…)` on those fields: `validate_effects_rows.go:32,38,91,103`, `typechecker_app_effects.go:117,127`, `row_unification.go:193-203`, `typechecker_latent_mask.go:82` | All write into clones (`cloneEffectRow`, `cloneApplicationRow`, a copied `TFunc2`) or into the fresh maps `UnionEffectRows` builds; its nil-input passthrough is cloned by `unionRequiredEffectRows`. Confirmed |
| V7 | gob refuses a typed nil inside an interface, so today's blobs never contain one | `TestCoreTIDAG_TypedNilFailsLikeGob` | Confirmed: both codecs refuse |
| V9 | gob keeps an empty non-nil map distinct from nil but collapses empty slices | `TestCoreTIDAG_EveryVariantMatchesGob` zero cases (first run failed on `Fields: {}` until flagged) | Confirmed: encoded as `ctiFlagEmptyMap` |
| V10 | `TLabelled` labels serialize through `MarshalLabel` (its own `GobEncode`), not through plain gob | `internal/types/label_gob.go` | Confirmed: the DAG codec stores the same `MarshalLabel` bytes |
| V8 | `"v6"` is pinned in 4 test sites | `grep -rn '"v6"' internal/ cmd/` | Confirmed (listed in Files) |

## Examples

### Before
```
$ ailang check src/core/session.ail     # every run
CACHE_WRITE_FAILED module=src/core/session stage=encoding … ARTIFACT_TOO_LARGE
real 71–126 s
```

### After
```
$ ailang check src/core/session.ail     # first run writes the cache (coretypeinfo.gob < 2 MB)
$ ailang check src/core/session.ail     # cache hit
real a few seconds
```

## Success Criteria

- [ ] DAG codec round-trips every variant; equivalence with gob holds on the std plus examples corpus
- [ ] Source- and field-enumeration tests fail on an uncovered variant or field
- [ ] Malformed blobs return errors, never panic
- [ ] Guard test: fixture blob under budget and ≥10× smaller than plain gob
- [ ] motoko `session.ail` caches; blob size and warm time recorded
- [ ] `cacheKeyVersion` v7, seeds updated
- [ ] `make test-core`, `make lint`, `make check-file-sizes`, `make check-boundaries` pass; changelog fragment added
- [ ] Follow-up issue filed for the fresh-compile slowdown, with profile

## Testing Strategy

Unit tests (M1), corpus equivalence and the fixture guard (M2), a real-world motoko measurement (M3). The equivalence oracle is today's gob round trip, so any behavior difference on a cache hit is a test failure, not a judgment call.

## Deferred Decisions

- Hash-cons key construction (string builder vs. fnv hash with collision check): agent may choose. It must be exact; a hash-only key without an equality check is not allowed.
- Exact guard budget: agent sets it from the measured fixture size with about 4× headroom.

## Non-Goals

- **Fixing the 34 s → 83 s fresh-compile slowdown.** That's a separate cause, probably type copying in substitution and unification. It gets a follow-up issue with a profile.
- **Reverting or narrowing #1706's alias closure.** Closure fixes a real soundness bug (#1614), and the encoding is what should absorb its size.
- **Configurable ceilings** (the other half of m-compile-cache-large-module-blobs).
- **DAG-encoding `core.gob`** (1.7 MB here, not a problem).

## Timeline

- M1: 0.75 day
- M2: 0.5 day
- M3: 0.25 day

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| A field silently dropped on encode | Reflection field-count pin per variant, plus gob-equivalence corpus |
| Pointer sharing exposes an in-place mutation | V5 audit; cache-hit runtime tests (`TestAliasBodyClosure_*` runtime, `cmd/ailang` cache tests) run on decoded values |
| Hostile or corrupt blob causes huge allocation | Blob is still read under `maxArtifactBlobBytes`; decoder bounds-checks every ref against the current index and errors on unknown tags |
| `cacheKeyVersion` collision with #1771 | The second to merge takes the next version; the conflict is a one-line rebase |

## Related Documents

- [m-compile-cache-large-module-blobs](../v0_52_2/m-compile-cache-large-module-blobs.md): planned v0.52.2. Its CoreTI whole-type dedup is superseded by V2's measurement; its configurable ceilings are unaffected
- [inline-test-per-case-recompile-1328](../ailang-core-triage/inline-test-per-case-recompile-1328.md): the 16/32 → 64 MiB raise
- [m-alias-body-closure](../v0_52_4/m-alias-body-closure.md): the change whose expansion this absorbs
- [coretypeinfo-gob-inflation-breaks-session-cache](../ailang-core-triage/coretypeinfo-gob-inflation-breaks-session-cache.md): the triage (#1779), corrected here

## References

- [Design Axioms](/docs/references/axioms)
- Issue #1778 and its verification comment

## Future Work

- Profile and fix the fresh-compile regression (follow-up issue from M3)
- Consider the same DAG encoding for `iface.json` export schemes if interface sizes grow

---

**Document created**: 2026-10-12
**Last updated**: 2026-10-12
