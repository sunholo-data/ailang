# M-PERF6: Runtime Performance Hotspots

**Status**: Planned
**Target**: v0.11.3 — v0.12.x
**Priority**: P1 (Medium — improves DocParse and all I/O-heavy workloads)
**Estimated**: 2-3 days (Phases 1-3) + 1-2 days (Phase 4, closure hotspots)
**Dependencies**: M-INCREMENTAL-TYPECHECK (implemented), M-PERF-DOCPARSE (implemented)
**Additional Reports**:
- ailang-parse msg `e234c455` (2026-04-10) — XLSX 10-50x slower than competitors; CPU profile identifies closure env cloning and FallbackResolver as top hotspots on tight-loop `map` over 140K cells. **See new Phase 4 below.**

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure performance optimizations, no semantic changes |
| A2: Replayability | 0 | Traces unchanged |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Faster execution = more practical for AI agents |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Faster = more predictable costs |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

After M-PERF-GOROUTINE-ID, M-PERF-DOCPARSE, and M-INCREMENTAL-TYPECHECK, DocParse performance
improved dramatically (Alice EPUB: 12.1s → 2.97s, Moby Dick: 35.1s → 7.27s). Compilation is
now cached and skipped on warm runs.

**But the remaining runtime is dominated by Go runtime overhead, not AILANG logic.**

CPU profiling of Alice EPUB (warm cache, 2.73s total, 1.62s sampled):

| Category | % of CPU | Time | Root Cause |
|----------|----------|------|------------|
| Memory management (`madvise` + `memclr`) | 44% | ~1.0s | Heap allocation pressure from Value objects |
| Cache loading (`LoadArtifacts`) | 15% | ~0.4s | JSON unmarshal of CoreTypeInfo per module |
| GC pressure (`gcDrain`, `scanobject`) | 14% | ~0.4s | Consequence of allocation pressure |
| String conversion (`.String()`) | 11% | ~0.3s | `println` output + trace recording serialize values |
| Struct copies (`duffcopy`) | 9% | ~0.2s | Large Value structs passed by value |
| Evaluator core loop | 28% | ~0.5s | Inherent evaluation cost (addressed by bytecode VM) |

**Key insight:** 58% of CPU (memory + GC) is Go runtime overhead from excessive allocations.
The evaluator itself is only 28% of runtime. The biggest wins come from reducing allocations,
not optimizing the evaluator.

**Current State:**
- Alice EPUB warm: 2.97s (target was <2s)
- Moby Dick EPUB warm: 7.27s (target was <3s)
- 10MB DOCX: 2.18s

**Impact:**
- All AILANG programs with significant output or complex data structures
- DocParse is the primary benchmark but gains apply broadly

## Goals

**Primary Goal:** Reduce DocParse warm-cache runtime by 40-50% through allocation reduction and serialization optimization.

**Success Metrics:**
- Alice EPUB warm cache < 2.0s (from 2.97s)
- Moby Dick EPUB warm cache < 5.0s (from 7.27s)
- No regressions in `make test` or `make verify-examples`
- CPU profile shows memory management < 25% (from 44%)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| CoreTypeInfo serialization format (gob vs msgpack vs custom) | Affects all cached module loads, 15% of runtime | agent | design | low |
| Value struct layout changes (pointer vs value) | Affects all evaluator code that creates/passes Values | human | design | high |
| Output buffering strategy (buffer+flush vs lazy) | Affects println semantics and streaming behavior | agent | compile | low |
| GOGC tuning vs sync.Pool for hot types | Different tradeoffs: tuning is global, Pool is targeted | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] CoreTypeInfo serialization: switch to gob (same as Core AST, proven fast)
- [ ] Value struct changes: defer to Phase 3 investigation, not required for Phase 1-2 wins

## Solution Design

### Overview

Three phases, ordered by impact/effort ratio (highest first):

1. **Phase 1: CoreTypeInfo → gob** (15% of runtime, ~1 hour) — Switch serialization format
2. **Phase 2: Output buffering** (11% of runtime, ~2 hours) — Buffer println, flush at end
3. **Phase 3: Allocation reduction** (44% of runtime, ~1 day) — Reduce heap pressure

### Phase 1: CoreTypeInfo Serialization → gob

**Problem:** `LoadArtifacts` spends 15% of CPU on `encoding/json.Unmarshal` → `UnmarshalType` for
every cached module. JSON is slow for complex nested type trees.

**Fix:** Use `encoding/gob` (already used for Core AST in `core/gob.go`). Gob is 3-5x faster
than JSON for Go struct serialization and produces smaller output.

**Changes:**
- `internal/pipeline/cache_store.go`: Replace `coretypeinfo.json` with `coretypeinfo.gob`
- `internal/types/json.go`: Add gob registration for all Type implementations (already have JSON)
- Cache format version bump to invalidate old caches

**Risk:** Low. Gob is already proven for Core AST. Cache invalidation handles migration.

### Phase 2: Output Buffering for println

**Problem:** Every `println` call invokes `.String()` on each argument, then writes to stdout.
DocParse prints hundreds of text blocks. Each `.String()` recursively serializes nested values
(Lists, Records, Tagged values), creating temporary strings that are immediately GC'd.

**Fix:** Buffer stdout writes, flush at program exit or on explicit flush.

**Changes:**
- `internal/eval/eval_simple.go`: Use `bufio.Writer` wrapping `os.Stdout`
- `internal/eval/eval_typed_helpers.go`: Same buffered writer
- Add flush at program exit in `main.go`

**Note:** `.String()` cost is inherent to output — we can't avoid it for printed values.
But buffering reduces syscall overhead and may reduce GC pressure from many small writes.

### Phase 3: Allocation Reduction (Investigation)

**Problem:** 44% of CPU in `madvise`/`memclr` + 14% in GC = 58% total memory management.
This is the dominant cost but requires careful investigation.

**Investigation targets:**
1. **`growslice` (15% cum):** Which slices are growing? Likely argument lists or environment chains.
   Fix: pre-allocate with capacity hints.
2. **`mallocgc` (15% cum):** Which types are allocated most? Use `go tool pprof -alloc_objects`.
   Fix: `sync.Pool` for hot types (Environment, argument slices).
3. **`duffcopy` (9%):** Which structs are copied? Likely Value interface boxes.
   Fix: Pass `*Value` instead of `Value` where safe.
4. **GOGC tuning:** Currently GOGC=500 for CLI. May need per-workload tuning or
   `debug.SetMemoryLimit()` instead.

**This phase requires allocation profiling before committing to a fix.** Run:
```bash
ailang run -memprofile /tmp/docparse_alice.mem -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
go tool pprof -top -alloc_objects /tmp/docparse_alice.mem
```

### Files to Modify/Create

**Phase 1:**
- `internal/pipeline/cache_store.go` — Switch CoreTypeInfo to gob (~30 LOC change)
- `internal/types/gob.go` — New file, gob registration for Type implementations (~50 LOC)

**Phase 2:**
- `internal/eval/eval_simple.go` — Buffered stdout (~10 LOC)
- `internal/eval/eval_typed_helpers.go` — Buffered stdout (~10 LOC)
- `cmd/ailang/main.go` — Flush buffer at exit (~5 LOC)

**Phase 3:**
- TBD based on allocation profiling results

## Success Criteria

- [ ] CoreTypeInfo loads via gob instead of JSON
- [ ] Cache format auto-migrates (old JSON caches invalidated)
- [ ] println output is buffered
- [ ] Alice EPUB warm cache < 2.0s
- [ ] Moby Dick EPUB warm cache < 5.0s
- [ ] Allocation profile shows reduced alloc_objects vs baseline
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] CHANGELOG updated with benchmark results

## Testing Strategy

**Unit tests:**
- Gob round-trip tests for all Type implementations (extend existing JSON tests)
- Buffered output test: verify println output matches unbuffered

**Integration tests:**
- DocParse cold/warm cycle produces identical output
- `AILANG_NO_CACHE=1` still works

**Benchmarks:**
- CPU profile before/after each phase
- Allocation profile before/after Phase 3
- Wall clock: Alice EPUB, Moby Dick EPUB, 10MB DOCX

### Phase 4: Closure / Environment Hotspots on Tight-Loop `map`

**Added 2026-04-14 in response to msg `e234c455` (ailang-parse XLSX perf).**

**Problem:** `map(\cell. parseXlsxCell(cell, ss), cells)` over 140K elements dominates
XLSX parsing at 5.27s (vs Kreuzberg 82ms, Pandoc 1.68s — 10-50x slower). CPU profile
on `poi_many_merges.xlsx` (829KB, 50K rows, 140K cells):

| Function | Flat | Cum | Category |
|----------|------|-----|----------|
| `runtime.madvise` | — | 51.2% | GC thrashing |
| `runtime.memclrNoHeapPointers` | — | 17.1% | Zeroing new allocations |
| `runtime.mallocgc` | — | 21.7% | Allocation pressure |
| `FallbackResolver.ResolveValue` | 11.5% | 14.8% | Per-cell name resolution |
| `Environment.Clone` | 0.2% | 13.9% | Env clone per closure invocation |
| `listMapImpl` | — | 13.1% | New cons cell per element |

**Root cause:** The idiomatic `map(\cell. f(cell), cells)` pattern allocates:
- 140K closure captures + **140K environment clones**
- 140K new list nodes (cons cells)
- ~1.4M total allocations → GC dominates at **51% of runtime**

This is not an XLSX-specific problem — it affects **all tight-loop code** including
Phases 1-3 DocParse benchmarks. Phase 4 is the highest-ROI remaining perf win after
Phases 1-3 land.

**Fix, in priority order (P0 → P3):**

#### Phase 4a (P0): Pure-closure `Environment.Clone` elision

**Hypothesis:** When `map`/`filter`/`fold`/`foldr` invokes a closure that is
(a) pure (no effects declared) and (b) captures only immutable bindings, cloning the
environment per call is wasted work. The captured bindings cannot be shadowed or
mutated inside the body.

**Strategy:**
- At closure construction time, compute a `pureCapture` flag: effects-empty row type AND
  only-immutable-capture (the Core elaborator already knows both)
- In `evalCoreApp`'s closure path, skip `Environment.Clone` when `pureCapture == true`
  AND no shadowing binding is introduced in the body
- Fall back to cloning if the body does introduce a shadow (rare for one-liner lambdas)

**Expected win:** ~14% cumulative CPU saved on tight-loop `map`. Likely larger in
practice because reduced allocation pressure compounds with GC relief.

**Files:**
- `internal/eval/eval_core.go` — closure construction + application paths (~60 LOC)
- `internal/core/core.go` — add `PureCapture bool` to `Lam` / `Closure` (~10 LOC)
- `internal/elaborate/*` — set `PureCapture` at elaboration time (~30 LOC)
- Tests: effect-trace must match unoptimized version exactly

**Risk:** Medium. Environment-sharing bugs in closures are historically nasty. Mitigate
with: (i) the `PureCapture` flag is opt-in; (ii) comprehensive property-based tests
comparing cloned vs shared semantics on random pure programs.

#### Phase 4b (P1): FallbackResolver fast path for local bindings

**Hypothesis:** `FallbackResolver.ResolveValue` at 11.5% flat on hot loops suggests
name resolution is going through a generic slow path even when the binding is known
to be local at compile time. The elaborator already resolves most names; anything
reaching `FallbackResolver` is either a builtin or a cross-module reference.

**Strategy:**
- Audit what reaches `FallbackResolver.ResolveValue` during `map(\x. f(x), xs)` — which
  names are not pre-resolved?
- Likely candidates: `f` (user-defined function from another module), builtin names
- Add a monomorphized fast path: cache the resolved `Value` on first lookup, reuse
  on subsequent invocations when the closure is called repeatedly

**Files:**
- `internal/eval/resolver.go` (or equivalent) — per-closure resolution cache (~50 LOC)

**Risk:** Low. Cache is closure-scoped; invalidation is automatic (GC'd with closure).

#### Phase 4c (P2): In-place list map for single-reference inputs

**Hypothesis:** `listMapImpl` allocates a new cons cell per element. When the input
list is single-referenced (not aliased), cons cells can be mutated in place.

**Strategy:**
- Reference-count or ownership-track `ListValue`; if refcount == 1 on entry to
  `listMapImpl`, reuse head pointers
- Fall back to allocation when refcount > 1

**Note:** This is significantly more invasive than 4a/4b. Defer pending data from
4a/4b — they may already close the gap to target perf.

**Files:** `internal/eval/list.go`, `internal/builtins/list.go` (~100 LOC)

**Risk:** High. Refcounting/ownership tracking in Go is unusual; easy to introduce
aliasing bugs. Consider deferring unless 4a+4b are insufficient.

#### Phase 4d (P3): `parseFoldChildren` builtin

**Hypothesis:** Current XLSX code does `parseElements(xml, "row", 5000)` → materializes
list of 5000 XmlNodes → `map` over the list. The intermediate list is pure waste.

**Strategy:** Add `parseFoldChildren(xml, parentTag, childTag, init, f)` that folds
directly over child elements of a parent without materializing the list.

**Overlap with M-PARSEFOLD-EARLY-TERMINATION:** This is a natural extension of the
sentinel-fold pattern. Consider implementing as part of that sprint if timing aligns.

**Files:** `internal/builtins/xml.go` (~50 LOC), examples, tests

**Risk:** Low. Straightforward extension of existing XML fold builtins.

#### Phase 4 Success Criteria

- [ ] `poi_many_merges.xlsx` (829KB, 50K rows): ≤1.5s (from 5.27s, ~3.5x improvement)
- [ ] CPU profile on same workload: `madvise` < 20% (from 51%), `Environment.Clone` cum < 3% (from 13.9%)
- [ ] No regression on DocParse benchmarks (Alice EPUB, Moby Dick, 10MB DOCX)
- [ ] `make test`, `make verify-examples` pass
- [ ] Property test: pure-closure elision produces identical traces to non-elided version on 1000 random inputs

#### Phase 4a Results (measured 2026-04-14, commit 5798cd5d)

**The original 5.27s baseline for `poi_many_merges.xlsx` in the table above was stale / machine-specific — it does NOT match current reality.** Fresh A/B measurements on the same file on this machine (macOS arm64, AILANG_NO_TRACE=1):

| Benchmark | File | Pre-M1 | Post-M1 | Delta |
|-----------|------|--------|---------|-------|
| poi_many_merges.xlsx | 829KB, 50K rows | **425.45s** | **408.78s** | **-3.9%** |
| gutenberg_alice.epub | 185KB | — | 1.98s | ✅ meets ≤2s target |
| gutenberg_moby_dick.epub | 797KB | — | 2.75s | ✅ meets ≤3s target |

**What this means:**
- M1 (universal `Clone` → `NewChildEnvironment` swap) delivered a real but modest ~4% win on XLSX, not the ~14% projected from the old 5.27s profile
- The 1.5s hard target is **not achievable** from evaluator-level optimizations alone (we are ~270× off)
- DocParse EPUB benchmarks meet their targets — no regression
- The dominant XLSX cost is almost certainly in the **xlsx_parser itself** (parsing 140K cells via `map(parseXlsxCell, cells)`), not in `Environment.Clone`

**Simplified M1 implementation:** The universal `Clone` → `NewChildEnvironment` swap turned out to be strictly safe in AILANG semantics — params write only to the new child scope, lookups traverse the parent chain, and AILANG never mutates existing bindings in a shared parent env. The design-doc `PureCapture` flag was dropped as unnecessary (saved ~150 LOC).

**Next steps:**
- Fresh CPU profile on current post-M1 state to find the real hotspot at 408s
- Re-evaluate whether M2 (FallbackResolver cache) is worth ~11% of 408s, or pivot to xlsx_parser-level optimization (batched cell parsing, string-table memoization, `parseFoldChildren` from Phase 4d)

#### Phase 4a Post-M1 CPU Profile (2026-04-14, `AILANG_NO_TRACE=1`)

Ran `ailang run -cpuprofile ... docparse/main.ail poi_many_merges.xlsx` with M1 installed. Total: 417.31s real, 389.40s sampled. **The hotspot distribution changed dramatically:**

| Function | Flat | Cum | Notes |
|----------|------|-----|-------|
| `FallbackResolver.ResolveValue` | **64.52%** | 67.87% | ← new dominant hotspot, 5.6× larger than projected |
| `evalCoreApp` / `evalCore` / `evalCoreLet` / `evalCoreMatch` | ~12% each cum | — | expected evaluator dispatch |
| `runtime.gcBgMarkWorker` | — | 9.84% | was dominant; now moderate |
| `listConcatImpl` | — | 6.64% | list ops |
| `runtime.mallocgc` | 0.12% | 6.38% | was 21.7% |
| `runtime.madvise` | **3.72%** | 3.72% | **was 51.2%** — M1 effect |
| `Environment.Clone` | — | — | **gone from top 30** — M1 eliminated it |

**M1 effect on allocation pressure:** The original profile showed `madvise`+`mallocgc`+`memclrNoHeapPointers` totalling **90%** of runtime as cumulative GC cost. Post-M1: `madvise` is 3.72%, `mallocgc` is 6.38% cum. **M1 crushed GC pressure from ~90% → ~10%** — the headline win was hidden by the larger FallbackResolver hotspot.

**FallbackResolver root cause (identified):** `eval_operations.go:160` (and two siblings at 81, 641) wraps `e.resolver` in a new `FallbackResolver` on every cross-module function application. Even with pop-on-return restoring `oldResolver`, each ResolveValue traverses a chain whose depth = cross-module call-stack depth. On `map(\cell. parseXlsxCell(cell, ss), cells)` over 140K cells, every builtin lookup and module reference walks this chain. The `f.Primary.ResolveValue(ref)` call in `eval_evaluator.go:25` is the hot line — interface-method dispatch + error-interface boxing at scale.

**Revised M2 strategy:** The original plan ("per-closure resolution cache") is sound but the simpler fix may dominate:
1. **Avoid chain growth:** in `eval_operations.go:160`, if `e.resolver` is already a `FallbackResolver` whose `Secondary == fn.Resolver`, skip the re-wrap entirely.
2. **Concrete type dispatch:** replace the `GlobalResolver` interface field in the hot path with a concrete struct and direct call to `moduleGlobalResolver.ResolveValue` (which itself is only 330ms cum — the terminal resolver is cheap; the chain is expensive).
3. **Per-closure cache:** still valuable for builtins resolved hundreds of thousands of times.

Any one of these likely saves >50s on this workload. All three together could plausibly bring `poi_many_merges.xlsx` from 408s to the 250–300s range — still far from 1.5s (which requires xlsx_parser-level work), but a significant step.

#### Phase 4a M2a Results (2026-04-14)

Implemented option #1 (chain-growth guard) — a 5-line change adding `resolverCovers(chain, target)` helper that walks the existing chain once and skips the re-wrap if `target` is already reachable. Applied at all 3 function-application sites.

| Metric | Pre-M1 | Post-M1 | Post-M2a | M2a delta | Total |
|--------|--------|---------|----------|-----------|-------|
| poi_many_merges.xlsx | 425.45s | 408.78s | **64.46s** | **-344s (6.3×)** | **6.6×** |
| FallbackResolver.ResolveValue flat | — | 64.52% | <1% (out of top 15) | — | — |
| Alice EPUB | — | 1.98s | 1.98s | 0 | meets target |
| Moby Dick EPUB | — | 2.75s | 2.75s | 0 | meets target |

The post-M2a hotspot distribution is dominated by GC (`gcDrain` 41% cum, `gcBgMarkWorker` 30% cum) and `listConcatImpl` (13.46% cum). At 64s total, these absolute costs are ~26s and ~9s respectively — much smaller than what M2a removed.

**Original M2 design (per-closure resolution cache) is now deferred** — the chain-growth guard solved 99% of the FallbackResolver problem at 5% of the LOC cost. A cache would save further microseconds per lookup but the headroom is small.

**Next obvious targets if more XLSX speedup is wanted:**
- `listConcatImpl` 13% cum (~9s) — likely related to building the cell list before mapping
- GC pressure 41% cum — would benefit from value pooling for primitive types or reduced allocation in `evalCore` dispatch
- xlsx_parser-level: batched cell parsing, sharedStrings lookup memoization (unrelated to evaluator)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Phase 3 specific fixes — agent decides based on allocation profiling data
- Whether to use `sync.Pool` vs pre-allocation vs pointer passing — agent chooses based on profile
- Buffer size for println output — agent may choose (default: `bufio.NewWriter` 4KB default is fine)

## Non-Goals

**Not attempted in this feature:**
- Bytecode VM — separate sprint (M-PERF4), would eliminate evaluator overhead entirely
- Parallel module loading — complex, unclear benefit for sequential evaluation
- Custom allocator — too invasive, Go's runtime allocator is good enough with reduced pressure
- Trace recording optimization — traces are opt-in and rarely enabled in production

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gob format changes break cache | Low | Cache version bump forces rebuild, no data loss |
| Buffered output changes println timing | Low | Flush at exit; buffering is transparent |
| Phase 3 yields marginal gains | Med | Profile first, only implement changes with >5% impact |
| Value struct changes break evaluator | High | Phase 3 is investigation-first; no changes without profiling evidence |

## Related Documents

**Implemented (prior art):**
- M-PERF-GOROUTINE-ID — Eliminated `runtime.Stack()` bottleneck (42% → 0%), 2.5-3.6x speedup
- M-PERF-DOCPARSE — Deferred CoreTI substitution + GC tuning, 1.6x speedup
- M-INCREMENTAL-TYPECHECK — Compilation skip via cached artifacts, 34% on Moby Dick

**Planned (related):**
- [M-PERF4: Bytecode Interpreter](design_docs/planned/v1_0_0/m-perf4-bytecode-interpreter.md) — Would replace evaluator entirely

## Profiling Data (Baseline)

```
CPU Profile: Alice EPUB warm cache (2.73s duration, 1.62s sampled)

Top by flat time:
  0.39s  24%  runtime.madvise
  0.33s  20%  runtime.memclrNoHeapPointers
  0.15s   9%  runtime.duffcopy
  0.09s   6%  runtime.scanobject
  0.06s   4%  runtime.(*mspan).heapBitsSmallForAddr
  0.06s   4%  runtime.usleep

Top by cumulative time:
  0.65s  40%  runtime.systemstack
  0.46s  28%  eval.(*CoreEvaluator).evalCore
  0.43s  27%  eval.(*CoreEvaluator).evalCoreApp
  0.41s  25%  runtime.(*mheap).allocSpan
  0.33s  20%  eval.(*CoreEvaluator).evalCoreMatch
  0.24s  15%  pipeline.(*CacheStore).LoadArtifacts
  0.23s  14%  types.UnmarshalCoreTypeInfo
  0.18s  11%  eval.(*ListValue).String
  0.14s   9%  eval.(*TaggedValue).String
  0.14s   9%  builtins.listMapImpl
```

## Future Work

- Bytecode VM (M-PERF4) would eliminate evaluator overhead (28% of current runtime)
- Memory-mapped cache files could eliminate LoadArtifacts deserialization entirely
- Parallel module artifact loading (currently sequential)

---

**Document created**: 2026-04-10
**Last updated**: 2026-04-14 (added Phase 4: closure/env hotspots from XLSX perf msg e234c455)
