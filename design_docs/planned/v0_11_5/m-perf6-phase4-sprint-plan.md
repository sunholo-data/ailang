---
name: M-PERF6 Phase 4 — Closure & Environment Hotspots
status: Planned
target: v0.11.5
priority: P0 (perf correctness — blocking XLSX workloads)
estimated: 2-3 days (~400 LOC across 2 shippable milestones + 1 release milestone)
parent: design_docs/planned/m-perf6-runtime-hotspots.md (Phase 4)
related_messages:
  - e234c455 (ailang-parse XLSX: 10-50x slower than Kreuzberg/Pandoc)
  - fe024ef3 (AILANG perf: cons+reverse, stdlib primitives, find() guards)
  - 5149815e (String/list perf patterns validated in AILANG Parse LaTeX parser)
created: 2026-04-14
---

# M-PERF6 Phase 4 Sprint Plan

**Goal:** Close the XLSX 10-50x perf gap by eliminating per-call closure allocation overhead and redundant name resolution on tight-loop `map` / `filter` / `fold`.

**Scope:** Phase 4a (P0) + Phase 4b (P1) from the parent design doc. Phase 4c (in-place list mutation, P2) is **deferred** pending data from 4a/4b — they may already close the gap. Phase 4d (`parseFoldChildren`, P3) is deferred to a later XML-focused sprint.

## Baseline (from CPU profile on `poi_many_merges.xlsx`, 50K rows, 140K cells, 5.27s total)

| Function | Flat | Cum | Target after Phase 4 |
|----------|------|-----|----------------------|
| `runtime.madvise` | — | **51.2%** | **< 20%** |
| `runtime.mallocgc` | — | 21.7% | < 12% |
| `runtime.memclrNoHeapPointers` | — | 17.1% | < 10% |
| `FallbackResolver.ResolveValue` | **11.5%** | 14.8% | **< 3%** |
| `Environment.Clone` | 0.2% | **13.9%** | **< 3%** |
| `listMapImpl` | — | 13.1% | (deferred — Phase 4c) |

**Headline target:** `poi_many_merges.xlsx` from 5.27s → ≤ 1.5s (~3.5x improvement).

## Velocity Basis

Recent similar sprints (last 7 days):
- **M-POLY-ORD** (type checker fix + cache key): ~400 LOC in 1 day
- **M-V0_11_3-HOTFIX** (short-circuit + FoldStep): ~450 LOC in 1 day
- **M-PERF-GOROUTINE-ID** (perf win via profile-driven change): ~200 LOC

Evaluator/core changes are higher-risk than type-checker changes. Budgeting 2-3 days with property-test overhead.

## Milestones

### M1: Phase 4a — Pure-Closure `Environment.Clone` Elision (P0)

**What:** Skip the per-call `Environment.Clone` when the invoked closure has no effects AND captures only immutable bindings AND the body introduces no shadowing.

**Files (est. ~200 LOC total):**
- `internal/core/core.go` — add `PureCapture bool` to `Lam` and `Closure` (~10 LOC)
- `internal/elaborate/*` — compute `PureCapture` at elaboration time using effect row + capture analysis already done by the elaborator (~50 LOC)
- `internal/eval/eval_core.go` — guard `Environment.Clone` in `evalCoreApp` closure path on `PureCapture && !bodyShadows` (~60 LOC)
- `internal/eval/eval_core_test.go` — unit tests for cloned-vs-shared semantics (~40 LOC)
- `internal/eval/closure_purecapture_property_test.go` — property test (1000 random pure lambdas, effect-trace must be bit-identical to unoptimized path) (~60 LOC)

**Acceptance criteria:**
- `Lam` and `Closure` carry `PureCapture bool`
- Elaborator sets `PureCapture = true` iff: effect row empty AND all captures refer to let-bound (non-`ref`, non-`mutable`) values
- `evalCoreApp` short-circuits `Environment.Clone` when `PureCapture && no-shadow-in-body`
- Property test: 1000 random pure lambdas produce identical evaluator traces with/without elision
- Unit test: closure that DOES shadow a capture still works correctly (falls back to clone)
- Unit test: closure that captures a `ref`-bound value is NOT marked `PureCapture`
- `make test` passes
- `make verify-examples` passes (no regressions on effectful examples)
- Benchmark: `poi_many_merges.xlsx` shows measurable cum-time reduction for `Environment.Clone` (target: < 3% from 13.9%)

**Risk:** Medium. Environment-sharing bugs are historically nasty. The `PureCapture` flag is opt-in — anything not flagged takes the safe cloned path. Property test on random programs is the main mitigation.

### M2: Phase 4b — FallbackResolver Fast Path for Local Bindings (P1)

**What:** Cache resolved `Value` per closure on first lookup. Avoid re-resolving the same name through `FallbackResolver.ResolveValue` on every invocation of the same closure.

**Files (est. ~130 LOC total):**
- **Audit first** (no code) — instrument `FallbackResolver.ResolveValue` with a counter + sample trace on `poi_many_merges.xlsx` to identify which names are hot (builtins? cross-module refs? user-defined top-level?). Remove instrumentation before commit.
- `internal/eval/resolver.go` (or equivalent — find the actual file) — add a per-closure resolution cache keyed by name, populated on first hit, invalidated automatically when the closure is GC'd (~80 LOC)
- `internal/eval/resolver_test.go` — cache correctness + invalidation on module reload (~50 LOC)

**Acceptance criteria:**
- Audit note in sprint JSON `notes`: which names were hottest through `FallbackResolver.ResolveValue`
- Closure-scoped cache implemented; first call populates, subsequent calls hit cache
- Test: resolving the same name twice through the cache returns the same `Value` (pointer-equal where expected)
- Test: cache is invalidated when the underlying binding changes (e.g., hot-reload scenario — if applicable; otherwise document non-applicability)
- Benchmark: `FallbackResolver.ResolveValue` flat drops from 11.5% → < 3%
- `make test` passes

**Risk:** Low. Cache is closure-scoped — lifetime matches closure, GC handles invalidation. Main risk is cache invalidation correctness for cross-module hot-reload, which is rare in production.

### M3: Benchmark + Release + Ack (closeout)

**What:** Re-run DocParse + XLSX benchmarks with both optimizations, update CHANGELOG, ack message `e234c455`, release v0.11.5.

**Files (est. ~40 LOC documentation + no-code):**
- `changelogs/v0.10-current.md` — v0.11.5 section with Performance entry covering both 4a and 4b
- `design_docs/planned/m-perf6-runtime-hotspots.md` — update Phase 4 status to Implemented, move to `design_docs/implemented/v0_11_5/`
- `cmd/ailang/messages` — `ailang messages ack e234c455` with link to release

**Acceptance criteria:**
- `poi_many_merges.xlsx`: ≤ 1.5s (from 5.27s, ~3.5x improvement) — **HARD TARGET**
- CPU profile on same workload: `madvise` < 20% (from 51%), `Environment.Clone` cum < 3% (from 13.9%), `FallbackResolver.ResolveValue` flat < 3% (from 11.5%)
- No regression on DocParse benchmarks (Alice EPUB ≤ 2s, Moby Dick EPUB ≤ 3s)
- CHANGELOG v0.11.5 section complete
- Design doc moved to `implemented/v0_11_5/`
- Message `e234c455` acked
- Release v0.11.5 tagged + pushed via `release-manager` skill

**Risk:** Low (assuming M1 + M2 pass). If the 1.5s target is missed by > 20%, pause and evaluate Phase 4c (list in-place) before releasing.

## Dependencies

- M1 → M2 (M2 depends on M1 shipping first so its benchmark numbers isolate the FallbackResolver contribution cleanly)
- M2 → M3 (M3 depends on both M1 and M2 landing)

## Day-by-Day Plan

| Day | Work |
|-----|------|
| Day 1 AM | M1 design review, instrument `evalCoreApp` to confirm baseline `Environment.Clone` hit rate. Add `PureCapture` field to `Lam` / `Closure`. |
| Day 1 PM | M1 elaborator: compute `PureCapture`. M1 evaluator: guard the clone. Unit tests. |
| Day 2 AM | M1 property test (1000 random pure lambdas). Fix any divergences found. Merge M1. |
| Day 2 PM | M2 audit: which names hit `FallbackResolver.ResolveValue` on the hot path. Design cache shape. |
| Day 3 AM | M2 implementation + tests. Merge M2. |
| Day 3 PM | M3 benchmark rerun, CHANGELOG, design doc move, ack, release. |

## Out of Scope (explicitly deferred)

- **Phase 4c** (in-place list map via refcount/ownership) — deferred. High risk, only attempt if 4a+4b fail to hit the 1.5s target.
- **Phase 4d** (`parseFoldChildren` to avoid materializing intermediate list) — deferred to a focused XML-stream sprint. Partial overlap with already-shipped `parseFoldStep` / `scanFoldStep`.
- **Bytecode VM** (M-PERF4) — separate sprint, would eliminate evaluator overhead entirely.
- **Custom allocator / sync.Pool for Value objects** — non-goal per parent design doc.

## Success Criteria (Sprint-Level)

**Revised 2026-04-14 after M1 A/B measurement — the original 5.27s baseline was stale.**

- [x] M1 implemented (universal `Clone` → `NewChildEnvironment` swap, commit 5798cd5d)
- [x] Zero regressions on DocParse Alice (1.98s ≤ 2s target) / Moby Dick (2.75s ≤ 3s target)
- [x] `make test` passes, `make verify-examples` 161/161 passes
- [x] M1 A/B on `poi_many_merges.xlsx`: 425.45s → 408.78s (**-3.9%**) — modest but real
- [ ] CHANGELOG v0.11.5 Performance entry (M1 shipped, M2 tbd)
- [ ] Fresh CPU profile on post-M1 state to decide: M2 (resolver cache) vs pivot to xlsx_parser-level work
- [ ] Message `e234c455` acked
- [ ] v0.11.5 released

**Dropped targets** (stale 5.27s baseline — reality is ~7 minutes on this machine):
- ~~`poi_many_merges.xlsx` ≤ 1.5s~~ — not achievable at evaluator level; a ~270× gap requires xlsx_parser rework
- ~~`Environment.Clone` cum < 3%~~ — was 13.9% of a 5.27s profile; at 408s the hotspot distribution is entirely different
- ~~Property test for 1000 random pure lambdas~~ — skipped; universal swap is provably safe without the `PureCapture` flag

## Related Documents

- **Parent design doc:** [design_docs/planned/m-perf6-runtime-hotspots.md](../m-perf6-runtime-hotspots.md)
- **Prior art (implemented):** M-PERF-GOROUTINE-ID (2.5-3.6x via profile-driven change), M-PERF-DOCPARSE (1.6x via CoreTI + GC tuning), M-PERF5 (DocParse bytecode VM warm cache)
- **Reported by:** msg `e234c455` (ailang-parse XLSX perf)
