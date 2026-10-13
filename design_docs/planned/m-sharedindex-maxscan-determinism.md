# M-SHAREDINDEX-MAXSCAN-DETERMINISM: Strict-mode bounded scans must be deterministic

**Status**: Planned
**Target**: v0.54.8
**Priority**: P1 (High)
**Estimated**: 1 day
**Dependencies**: None (fixes a defect in M-DX16, implemented v0.6.0)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Restores the documented Strict-mode contract ("identical results for identical queries on fixed index state") for every parameter combination, not just `maxScan = 0` |
| A2: Replayability | +1 | Stapledon's AI service needs replay-exact `findSimhash` output; the fix makes bounded Strict scans bit-identical across runs, engines, and replays |
| A3: Effect Legibility | 0 | No new effects; the `SharedIndex` effect surface is unchanged |
| A4: Explicit Authority | 0 | No capability changes; `--caps SharedIndex` gating untouched |
| A5: Bounded Verification | +1 | The fix is verifiable with local unit tests plus a run-twice-and-diff CLI check; no external service needed |
| A6: Safe Concurrency | 0 | Read path stays under `RLock`; the sorted key snapshot is computed inside the existing critical section |
| A7: Machines First | +1 | Deterministic output is machine-verifiable; today an agent re-running the same query gets different answers it cannot reason about |
| A8: Minimal Syntax | +1 | No syntax, builtin-signature, or stdlib-signature changes — behavior and doc comments only |
| A9: Cost Visibility | 0 | `maxScan` remains an explicit, caller-supplied bound on scan cost; the added sort cost is proportional to the same N |
| A10: Composability | 0 | No interface changes; all `SharedIndex` implementations and callers compose as before |
| A11: Structured Failure | 0 | No new error paths; no behavior becomes an error |
| A12: System Boundary | 0 | In-process effect package only; no boundary crossings added or removed |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — the change *removes* implicit nondeterminism (Go map iteration order) from a path that claims determinism
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`InMemorySharedIndex.FindSimilarSimHash` and `FindSimilarByEmbedding` iterate a Go map (`for key, entry := range ns`) and stop at `maxScan`. Go randomizes map iteration order, so when `maxScan > 0` the *candidate subset* that gets scored changes from run to run. The `(score DESC, key ASC)` sort applied afterwards cannot fix a different candidate set — it only orders whatever random subset happened to be scanned first.

**Current State:**
- `internal/effects/sharedindex_inmemory.go:91-95` (SimHash path) and `:200-205` (embedding path) both break out of a randomized map loop at `maxScan`.
- The interface contract (`internal/effects/sharedindex.go:17`) states: *"DeterminismStrict guarantees identical results for identical queries on fixed index state."* That guarantee is silently violated whenever `deterministic = true` is combined with `maxScan > 0`.
- Verified live on the current tree (binary v0.54.6, commit 71f4cf6, std v0.54.7): upsert 6 keys (`k1`–`k6`) with distinct simhashes, then `findSimhash("ns", 1, 1, 3, true)` ten times → the top result was `k1` (score 1.0, the true best match) in 7 runs, `k3` in 1, `k4` in 2. With `maxScan = 0` the same query returned `k1` in 10/10 runs.
- Affects **both engines** — the interpreter and the bytecode VM (`--bytecode`, `--strict-bytecode`) share the same Go builtin implementations (`internal/vm/builtins_adapted.go` → `internal/builtins/sharedindex.go` → `internal/effects`), so both inherit the randomized scan.
- Existing tests never catch this because every determinism test uses `maxScan = 0` (`TestInMemorySharedIndex_DeterminismStrict`, `TestInMemorySharedIndex_FindSimilarByEmbedding_DeterministicOrdering`), and `TestInMemorySharedIndex_MaxScan` explicitly only asserts `len(results) <= 10` with the comment *"map iteration order is non-deterministic"* — even though it passes `DeterminismStrict`.

**Impact:**
- **Stapledon's Voyage / AI service (primary)**: wants `sharedindex` with replay-exact output. Any bounded Strict scan is unreplayable today.
- **Any AILANG program** combining `deterministic = true` with `maxScan > 0` for cost bounding on large namespaces — including shipped example `examples/reference/neural_semantic_search.ail:36` (`_sharedindex_find_by_embedding(ns, query_embedding, top_k, 100, true)`), which silently returns different results once a namespace exceeds 100 entries.
- Erodes trust in the Strict/BestEffort contract itself: the flag that exists precisely to guarantee determinism does not.

## Goals

**Primary Goal:** Make `FindSimilarSimHash` and `FindSimilarByEmbedding` honor the DeterminismStrict contract for all `maxScan` values, by scanning keys in ascending key order before applying the bound.

**Success Metrics:**
1. A bounded Strict query (`maxScan > 0`) repeated N times on fixed index state returns byte-identical results (keys, scores, order) in N/N runs.
2. Interpreter and `--strict-bytecode` VM produce identical output for the same program and index state.
3. BestEffort-mode behavior and performance characteristics are unchanged (no sort on that path).
4. The same class of defect (Go map iteration order leaking into observable effect output) is closed across the effects package — including `_sharedmem_keys`, which today returns keys in random order.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Strict + `maxScan > 0` scans the lexicographically-first `maxScan` keys (deterministic prefix scan), rather than erroring or documenting the combination as unsupported | Fixes the contract violation while keeping the useful Strict+bounded combination; changes observable output of every existing Strict+bounded call (from a random subset to the key-order prefix) | human (this doc; alternatives rejected below) | design | low (localized to 2 functions) |
| D2: BestEffort path keeps raw map iteration — no sort, no behavior change | Preserves the documented performance profile of BestEffort; the mode is the explicit opt-out of determinism | agent (follows from mode semantics) | design | low |
| D3: `maxScan` counts entries *examined*, including embedding-less entries skipped by `FindSimilarByEmbedding` | Current behavior; changing it would silently alter BestEffort results too. Documented rather than changed | agent (keep current, document) | design | low |
| D4: Also sort `InMemorySharedCache.Keys()` (fixes `_sharedmem_keys` random ordering — same defect class) | Same-class fix per the systemic-audit principle: map iteration order must not feed observable effect output. Changes observable order of `_sharedmem_keys` from random to sorted | agent (one-line sort, aligned with A1) | design | low |
| D5: Per-query key sort (O(K log K)) instead of maintaining an incrementally-updated sorted key structure | Simplicity; namespace sizes (shared-cognition frames) are small. The maintained structure is a profiling-gated future optimization | agent (perf choice, revisitable) | compile | low now, med if revisited |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — resolved in this document (issue reporter's proposed fix adopted; alternatives evaluated and rejected in Solution Design)
- [x] D4 — resolved in this document (in scope; called out for reviewer attention since it widens the change one file beyond the reported bug)

## Solution Design

### Overview

When `mode == DeterminismStrict`, snapshot the namespace's keys, sort them ascending, and iterate that deterministic order while applying the `maxScan` budget and scoring. When `mode == DeterminismBestEffort`, keep the existing raw map loop. Everything downstream (scoring, `(score DESC, key ASC)` sort, `topK` truncation) is unchanged.

This makes `maxScan` under Strict a **deterministic prefix scan**: the first `maxScan` keys in ascending key order are the candidate set — always the same candidates for the same index state, hence always the same scores, ordering, and `topK` truncation.

### Architecture

**Components:**
1. **`sortedNamespaceKeys` helper** (`internal/effects/sharedindex_inmemory.go`): `func sortedNamespaceKeys(ns map[string]*IndexEntry) []string` — snapshot + `sort.Strings`. Shared by both Find methods.
2. **`FindSimilarSimHash`**: branch on `mode == DeterminismStrict` → iterate `sortedNamespaceKeys(ns)`; else the existing `range ns` loop. The `maxScan` check, scoring, result sort, and `topK` truncation are identical on both paths.
3. **`FindSimilarByEmbedding`**: same branch; the skip-entries-without-embeddings logic and the scan-budget semantics (D3) are unchanged.
4. **`InMemorySharedCache.Keys()`**: append `sort.Strings(keys)` before returning (D4) — makes `_sharedmem_keys` deterministic.

Why key order (and not insertion order or timestamp): timestamps are caller-supplied, may tie, and need not be monotonic; key order is the only canonical total order already used by the codebase (the Strict tie-break is `key ASC`, and `Namespaces()` already sorts). Using key order for the scan prefix makes the whole pipeline consistently keyed on one canonical order.

### Alternatives Considered (rejected)

- **Error on `Strict && maxScan > 0`**: removes a legitimate, already-shipped combination (the neural search example uses it); forces callers to choose between determinism and cost bounding, which is exactly the combination Stapledon needs. Rejected.
- **Document `maxScan` as BestEffort-only**: does not fix the contract violation, only renames it; breaks every existing Strict+bounded caller including `examples/reference/neural_semantic_search.ail`. Rejected — but the *documentation* half is kept: the interface comments and stdlib doc comments will state the prefix-scan semantics explicitly.
- **Maintained sorted key structure** (sorted slice or ordered map updated on Upsert/Delete): avoids the per-query sort but adds invariant-carrying state to every mutation path. Deferred until profiling shows the O(K log K) per-query sort matters (D5).

### Implementation Plan

**Phase 1: Core fix** (~2 hours)
- [ ] Add `sortedNamespaceKeys` helper to `internal/effects/sharedindex_inmemory.go`
- [ ] Branch `FindSimilarSimHash` on Strict mode → sorted-key iteration; BestEffort path untouched
- [ ] Branch `FindSimilarByEmbedding` identically
- [ ] Update the doc comments on both methods and on the `SharedIndex` interface (`internal/effects/sharedindex.go`): *"Strict mode with maxScan > 0 scans the lexicographically first maxScan keys (deterministic prefix scan); use maxScan = 0 for a full deterministic scan"*

**Phase 2: Same-class fix** (~1 hour)
- [ ] Sort the returned slice in `InMemorySharedCache.Keys()` (`internal/effects/sharedmem_inmemory.go`)
- [ ] Update its doc comment and the `_sharedmem_keys` builtin description if it mentions ordering

**Phase 3: Tests, docs, examples** (~3 hours)
- [ ] New unit tests (see Testing Strategy)
- [ ] Update `std/sharedindex.ail` and `std/sem.ail` doc comments for `maxScan` semantics (Strict = key-order prefix scan; BestEffort = unspecified subset)
- [ ] Add a runbook note or small script check: run the repro `.ail` N times, diff outputs (both engines)
- [ ] CHANGELOG entry at release time

### Files to Modify/Create

**New files:** none

**Modified files:**
- `internal/effects/sharedindex_inmemory.go` — helper + two Strict branches, ~40 LOC
- `internal/effects/sharedindex.go` — interface/method doc comments, ~10 LOC
- `internal/effects/sharedmem_inmemory.go` — sort in `Keys()`, ~2 LOC
- `internal/effects/sharedindex_test.go` — new tests, ~80 LOC
- `internal/effects/sharedmem_test.go` — `Keys()` ordering test, ~15 LOC
- `std/sharedindex.ail`, `std/sem.ail` — doc comments only

## Examples

### Example 1: The reported repro becomes stable

**Before** (observed, 10 runs of `findSimhash("ns", 1, 1, 3, true)` over keys `k1`–`k6` with simhashes 1,3,7,15,31,63):
```
[{key: k4, score: 0.953125, ...}]   # run 1
[{key: k1, score: 1.0, ...}]         # run 2
[{key: k3, score: 0.96875, ...}]      # run 3
...                                  # varies every run
```

**After** (deterministic prefix scan: candidates are always k1, k2, k3):
```
[{key: k1, score: 1.0, ...}]          # every run, both engines
```

### Example 2: Recall trade-off is explicit

With Strict + `maxScan = 3`, an entry `zz9` with a perfect match will never be returned (it is outside the lexicographic prefix). This is the deliberate cost of determinism under a bound. Guidance to document: **callers wanting full recall under Strict must pass `maxScan = 0`** — the full scan is already deterministic today and stays so.

## Success Criteria

- [ ] Repro test: 6 keys, `FindSimilarSimHash("ns", 1, 1, 3, DeterminismStrict)` returns `[k1]` on 100/100 consecutive calls (acceptance: new `TestInMemorySharedIndex_StrictMaxScanDeterministic`)
- [ ] Strict + maxScan scans exactly the lexicographically-first `maxScan` keys (acceptance: new test asserting the scanned set via observable results with distinct scores)
- [ ] Embedding path: equivalent determinism test for `FindSimilarByEmbedding` with `maxScan > 0`
- [ ] BestEffort behavior unchanged: existing tests pass without modification
- [ ] `InMemorySharedCache.Keys()` returns sorted keys (acceptance: new test; `_sharedmem_keys` output stable across runs)
- [ ] Interpreter and `--strict-bytecode` produce identical output for the same Strict+bounded program (run-twice-and-diff, both engines)
- [ ] All tests passing (`make test`)
- [ ] Documentation updated (interface comments, stdlib doc comments)

## Testing Strategy

**Unit tests** (`internal/effects/sharedindex_test.go`):
- `TestInMemorySharedIndex_StrictMaxScanDeterministic` — the reported repro: 6 distinct-simhash keys, `maxScan = 3`, Strict; run the query 100 times, assert identical results each run, and assert the top result is `k1` (proves the scanned set is the key-order prefix, since `k1` is only guaranteed to be in every prefix)
- `TestInMemorySharedIndex_StrictMaxScanLexicographicPrefix` — entries with keys covering the whole alphabet range and distinct simhashes; with `maxScan = k` in Strict mode, assert every returned key is among the `k` lexicographically smallest keys
- `TestInMemorySharedIndex_FindSimilarByEmbedding_StrictMaxScanDeterministic` — same shape for the embedding path, including entries *without* embeddings to pin the D3 budget semantics (embedding-less entries consume scan budget)
- BestEffort + maxScan: unchanged behavior (count ≤ maxScan, no panic)
- `TestInMemorySharedCache_KeysSorted` — keys returned in ascending order (`internal/effects/sharedmem_test.go`)

**Integration tests:**
- A `.ail` program performing the 6-key repro, executed N times via `ailang run --caps IO,SharedIndex` and via `ailang run --bytecode --strict-bytecode`; outputs diffed — must be identical within each engine and across engines.

**Manual testing:**
- None required beyond the above.

## Conflict Surface

*(Required: this change touches `internal/effects/`.)*

**1. What semantic positions does this change extend?**
The behavior of the existing `mode`-gated branch inside `FindSimilarSimHash` / `FindSimilarByEmbedding` when `maxScan > 0`. No new parameters, signatures, types, or call positions.

**2. What other valid constructs already live in those positions?**
- `deterministic = false` (BestEffort) with any `maxScan` — unchanged, raw map iteration retained.
- `maxScan = 0` (unbounded) with either mode — unchanged; the full scan is sorted only by the final `(score DESC, key ASC)` pass, exactly as today.
- `topK` truncation — applied after the same result sort; unchanged.
- `FindSimilarByEmbedding`'s skip-without-embeddings rule — unchanged in both modes (D3).

**3. How does the implementation disambiguate?**
The existing `DeterminismMode` enum is the sole switch; the new branch is `if mode == DeterminismStrict`. No inference, no heuristics.

**4. Which existing programs MUST still work post-change? (fixtures)**
- `examples/runnable/stdlib_sharedindex.ail` — uses `maxScan = 0` (full scan); unaffected.
- `examples/reference/neural_semantic_search.ail` — uses `find_by_embedding(..., 100, true)`; keeps working and *gains* stability once namespaces exceed 100 entries.
- `TestInMemorySharedIndex_MaxScan` — asserts only `len(results) <= 10`; still true (scanned prefix is exactly 10).
- `TestInMemorySharedIndex_DeterminismStrict` — `maxScan = 0`; still true.
- `TestInMemorySharedIndex_FindSimilarByEmbedding_DeterministicOrdering` — `maxScan = 0`; still true.
- `std/sem.ail` `find_similar_bounded` — same builtin underneath; gains determinism when `deterministic = true`.

**5. What deliberately changes (intentional incompatibilities)?**
- Strict + `maxScan > 0` results change from "a random maxScan-sized subset, sorted" to "the lexicographically-first maxScan keys, sorted". Any caller that was (incorrectly) relying on Strict+bounded results being stable across runs was already broken; callers relying on specific *lucky* subsets will see different — now stable — output.
- `_sharedmem_keys` output order changes from random to ascending (D4). No program can have depended on the random order except by luck.
- Recall under Strict + bounded scans is now biased toward lexicographically small keys. This is the accepted trade-off of D1; `maxScan = 0` remains the full-recall deterministic option, and this guidance is documented.

## Verification Log

*(Every load-bearing claim above, checked against code or a live run. Toolchain note: no Go compiler is available in this environment, so Go-level claims are verified by reading the source; language-level claims by running the installed `ailang` binary v0.54.6 / commit 71f4cf6 / std v0.54.7 — matching the reported affected versions v0.54.5–v0.54.7.)*

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| 1 | The `maxScan` break sits inside a raw Go map range loop at the two cited sites | Read `internal/effects/sharedindex_inmemory.go:89-98` and `:198-208` — `for key, entry := range ns { if maxScan > 0 && scanned >= maxScan { break } ... }` | Confirmed |
| 2 | Go map iteration order is randomized per run | Go language specification: "The iteration order over maps is not specified and is guaranteed to vary" | Confirmed (language fact) |
| 3 | Strict + `maxScan > 0` returns varying top results on fixed index state | Live repro: 6 keys, `findSimhash("ns", 1, 1, 3, true)` × 10 runs via `ailang run` → `k1` ×7, `k3` ×1, `k4` ×2 | Confirmed (reproduced) |
| 4 | `maxScan = 0` is stable today | Same program with `maxScan = 0` × 5 runs → `k1` 5/5 | Confirmed (control) |
| 5 | The interface claims Strict guarantees identical results | Read `internal/effects/sharedindex.go:17-19`: "DeterminismStrict guarantees identical results for identical queries on fixed index state" | Confirmed (contract violated by #3) |
| 6 | Both engines share the buggy Go implementation | `internal/vm/builtins_adapted.go:6,32` imports `internal/builtins` and resolves the same builtin specs that call `internal/effects`; no separate VM-side index code | Confirmed |
| 7 | `--strict-bytecode` is a real flag of `ailang run` | `ailang run --help`: "-strict-bytecode — With --bytecode: fail instead of falling back to the evaluator" | Confirmed |
| 8 | Existing determinism tests all use `maxScan = 0`; the MaxScan test acknowledges randomness | Read `TestInMemorySharedIndex_DeterminismStrict` (maxScan arg = 0), `TestInMemorySharedIndex_FindSimilarByEmbedding_DeterministicOrdering` (maxScan = 0), `TestInMemorySharedIndex_MaxScan` (comment: "map iteration order is non-deterministic, so we just check count <= 10", called with `DeterminismStrict`) | Confirmed |
| 9 | A shipped example uses Strict + `maxScan > 0` | `examples/reference/neural_semantic_search.ail:36`: `_sharedindex_find_by_embedding(ns, query_embedding, top_k, 100, true)` | Confirmed |
| 10 | Same defect class exists in `InMemorySharedCache.Keys()` (unsorted map iteration feeding observable output) | Read `internal/effects/sharedmem_inmemory.go:140-153` — `Keys()` returns unsorted `for k := range c.store` snapshot; consumed verbatim by `_sharedmem_keys` (`internal/builtins/sharedmem.go:349-368`) | Confirmed |
| 11 | No *other* scan-with-limit-over-map pattern exists in `internal/effects` | `grep -rn "maxScan" internal/ --include=*.go` → only `sharedindex_inmemory.go` (the two sites), `sharedindex.go` (interface/trace), `builtins/sharedindex.go` (arg plumbing), `storage/firestore/messaging_search.go` (Firestore query `.Limit(maxScan)` — server-side ordered query, not map iteration) | Confirmed |
| 12 | `Namespaces()` already sorts its map output (precedent for D1/D4) | Read `sharedindex_inmemory.go:168-177` — collects then `sort.Strings(names)` with comment "Sort for deterministic iteration" | Confirmed |
| 13 | Stapledon context: game simulation sidecar on the bytecode VM wanting replay-exact behavior | `missions/stapledon.toml` — Godot renderer + AILANG sidecar, bytecode VM; issue text: "Stapledon's Voyage wants sharedindex in the AI service with replay-exact output" | Confirmed |
| 14 | Error codes: none proposed (no new diagnostics) | N/A — no new error codes in this design | N/A |
| 15 | No goldens/eval/tests use the find-similar surface, so no regression risk there | `grep -rln "findSimhash\|find_similar\|find_by_embedding" goldens/ eval_projects/ tests/` → empty; only 3 `examples/` files use it (neural_semantic_search: Strict+100; semantic_retrieval: unbounded only; stdlib_sharedindex: maxScan=0) | Confirmed |

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Exact shape of the Strict branch** (early-return helper vs. inline loop over `sortedNamespaceKeys`) — agent may choose, provided BestEffort keeps the raw loop (D2).
- **Whether to extract a shared "scan budget" closure** shared by the two Find methods to avoid duplicating the maxScan/skip logic — agent may choose; two small loops are also acceptable.
- **Test loop counts** (e.g., 100 vs 500 repetitions in determinism tests) — agent may choose, with a bias toward keeping `make test-core` fast.
- **Whether the run-twice-and-diff CLI check lives as a shell one-liner in the PR description or a small script under `tools/`** — agent may choose; prefer existing tooling first (none exists for this today).

## Non-Goals

**Not attempted in this feature:**
- **Maintained sorted key structures / index re-architecture** — per-query sort is sufficient at current namespace sizes; revisit only with profiling evidence (D5, Future Work).
- **Changing `DeterminismMode` semantics, the builtin/stdlib signatures, or the `SharedIndex` interface shape** — behavior fix only.
- **Ordering guarantees for BestEffort mode** — nondeterministic order is the documented contract of that mode.
- **The Firestore-backed search path in `internal/storage/firestore/messaging_search.go`** — uses server-side ordered queries with `.Limit()`, not Go map iteration; no defect found.
- **Recall optimization under bounded Strict scans** (e.g., scanning the `maxScan` *highest-scoring* keys) — would require scoring before bounding, i.e., a full scan; contradictory. `maxScan = 0` is the documented full-recall option.

## Timeline

**Week 1** (1 day total):
- Phase 1: core fix (~2h)
- Phase 2: same-class fix (~1h)
- Phase 3: tests, docs, verification (~3h)

**Total: ~1 day** (estimate ×2 the naive 4 hours, per convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Callers depended on the old (random) Strict+bounded behavior in eval goldens | Low | Verified at doc time: `grep -rln "findSimhash\|find_similar\|find_by_embedding" goldens/ eval_projects/ tests/` → no hits; only `examples/reference/neural_semantic_search.ail` (Strict+maxScan=100), `examples/reference/semantic_retrieval.ail` (imports `find_similar_bounded`, calls only unbounded `find_similar`), and `examples/runnable/stdlib_sharedindex.ail` (maxScan=0) use the surface — none can regress |
| Deterministic prefix scan lowers recall for bounded Strict queries (lexicographic bias) | Med | Documented trade-off (D1); `maxScan = 0` is the full-recall deterministic path; guidance lands in stdlib doc comments |
| Per-query O(K log K) sort cost in Strict mode on very large namespaces | Low | BestEffort path unaffected; maintained sorted structure is a gated future optimization |
| Sorting `Keys()` changes observable `_sharedmem_keys` order | Low | Previous order was random; no program can have relied on it deterministically |

## Related Documents

**Implemented (may inform design):**
- [dx-16-shared-index-deterministic-retrieval.md](../implemented/v0_6_0/dx-16-shared-index-deterministic-retrieval.md) — the original M-DX16 design that introduced SharedIndex, DeterminismMode, and maxScan; this doc fixes a defect in its Strict-mode contract. (Not surfaced by the auto-search's keyword match; added manually.)

**Planned (check for overlap):**
- (none found — SimHash and neural search both returned no matches for "sharedindex maxscan determinism")

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Go language specification, "For statements with range clause": map iteration order is not specified and is guaranteed to vary
- Issue: sharedindex `maxScan > 0` with `deterministic = true` returns random results (Go map iteration), reported against v0.54.5 / v0.54.7-dev 329f4dd
- `missions/stapledon.toml` — Stapledon mission (Godot + AILANG bytecode-VM sidecar), the downstream consumer requiring replay-exact output

## Future Work

- Maintained sorted key structure (or an ordered map) per namespace if profiling shows the per-query sort matters on large namespaces.
- A general lint/review rule for the repo: "no `range` over a Go map may feed a `break`-on-budget or observable output ordering" (this is the second instance of the class; `Namespaces()` fixed it ad hoc in M-DX16, `Keys()` in this doc).

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13
