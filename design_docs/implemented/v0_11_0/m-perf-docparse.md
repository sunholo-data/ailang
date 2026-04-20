# M-PERF-DOCPARSE: Close the Python Performance Gap for Document Parsing

**Status**: Planned
**Target**: v0.11.0
**Priority**: P0 (Critical — AILANG is 50-64x slower than Python on the motivating benchmark)
**Estimated**: 3-5 days
**Dependencies**: M-PERF-GOROUTINE-ID (done)
**Milestone ID**: M-PERF-DOCPARSE
**Created**: 2026-04-09
**Source**: CPU profiling of docparse benchmarks post-goroutine-ID fix

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure performance optimization, no semantic change |
| A2: Replayability | 0 | No trace format changes |
| A3: Effect Legibility | 0 | Effect system unchanged |
| A4: Explicit Authority | 0 | Capability model unchanged |
| A5: Bounded Verification | +1 | Compilation caching reduces verification cost |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Competitive parsing speed enables AI agent document workflows |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Faster execution = lower compute cost per document |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No semantic changes
- [x] A3 (Effects): No effect system changes
- [x] A4 (Authority): No capability changes
- [x] A7 (Machines First): Directly improves machine document processing

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

This is a systemic issue affecting all compute-heavy AILANG programs, but docparse is the
motivating example because it's the public-facing benchmark against Python competitors.
The same fixes (compilation caching, GC reduction) will benefit all AILANG programs.

**Previous perf work:**
- M-PERF3 (v0.8.1) — evaluator quick wins (environment cloning, lookup)
- M-PERF-GOROUTINE-ID (v0.11.0) — eliminated 42% CPU tax, **2.5-3.6x speedup**
- M-BYTECODE-* sprints — wired 80+ builtins to VM for ~0% speedup on docparse

**Lesson learned:** M-BYTECODE-XML-BUILTINS delivered zero measurable improvement because
the bottleneck was never in evaluator dispatch. CPU profiling must guide all perf work.

---

## Problem Statement

### The Gap

AILANG docparse is **50-64x slower than Python (MarkItDown)** on EPUB parsing:

| File | AILANG | MarkItDown (Python) | Gap |
|------|--------|-------------------|-----|
| Alice EPUB (185KB) | 5,028ms | 78ms | **64x slower** |
| Moby Dick EPUB (797KB) | 9,478ms | 190ms | **50x slower** |
| test.epub (tiny) | 4,187ms | 26ms | **161x slower** |

Even in batch mode (compilation cached), AILANG is still dramatically slower:

| File | AILANG (batch) | MarkItDown | Gap |
|------|---------------|------------|-----|
| Alice EPUB | 2,933ms | 78ms | **38x slower** |
| Moby Dick EPUB | 3,727ms | 190ms | **20x slower** |

Note: MarkItDown extracts flat Markdown only — no track changes, comments, headers/footers,
merged-cell detection, or metadata. AILANG extracts far richer structure. The comparison
is on raw throughput, where we should still be competitive.

### Where Time Goes (CPU Profile, Batch Mode, Alice EPUB)

**Total: 2.83s** (batch mode eliminates most compilation overhead)

| Component | % of CPU | Time | What |
|-----------|----------|------|------|
| Type checking | 37% | ~1.0s | `inferCore`, `ApplySubstitution`, `SolveConstraints` |
| GC pressure | 26% | ~0.7s | `scanobject`, `mallocgc`, `gcDrain` |
| Map operations | 12% | ~0.3s | `mapassign`, `maps.Iter.Next`, `maps.table.Delete` |
| Evaluator | 10% | ~0.3s | `evalCore`, `evalCoreApp`, `CallValue` |
| Runtime/syscalls | 15% | ~0.4s | `memclr`, `pthread`, `usleep`, I/O |

### Key Insight

**Type checking runs on every invocation, even in batch mode.** The module cache stores
parsed ASTs but re-runs type inference every time. This is the single biggest win available.

The GC pressure comes from the evaluator's `eval.Value` interface boxing — every intermediate
result is heap-allocated. This is fundamental to the tree-walking evaluator architecture.

### Why This Matters

AILANG positions docparse as a competitive alternative to Python document parsing tools.
Being 50x slower undermines this claim and makes benchmarks embarrassing. We beat Python
on correctness, structural features, and type safety — we just need to not be embarrassingly
slow on raw throughput.

**Target: within 10x of Python** — acceptable for a type-safe, effect-tracked language.
Getting to 2-5x would be excellent. Parity is not expected (Python's libraries are C extensions).

---

## Goals

**Primary Goal:** Reduce AILANG docparse execution time to within 10x of Python competitors.

**Success Metrics:**
- Alice EPUB: ≤800ms in batch mode (currently 2,933ms) — within 10x of MarkItDown's 78ms
- Moby Dick EPUB: ≤2,000ms in batch mode (currently 3,727ms) — within 10x of 190ms
- No regressions in `make test`
- CPU profile shows no single component >30% (balanced workload)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Cache type-checked modules (not just parsed ASTs) | 37% of CPU is type inference that's repeated every run | human | design | high |
| Reduce eval.Value allocations vs full arena allocator | GC is 26% — small wins may suffice vs architectural change | agent | compile | med |
| Optimize xml builtin Go implementations vs keep current | Current Go XML code may have unnecessary allocations | agent | compile | low |
| Skip type checking for unchanged modules in batch mode | Batch mode should skip all compilation for cached modules | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Compilation caching strategy**: Cache type-checked + lowered modules, invalidate on source change
- [ ] **GC reduction approach**: Targeted fixes (sync.Pool, pre-allocation) vs arena allocator

---

## Solution Design

### Overview

Three independent optimization tracks, ordered by expected impact:

1. **Skip redundant type checking** (37% of CPU → target <5%)
2. **Reduce GC pressure** (26% of CPU → target <15%)
3. **Optimize hot-path Go code** (map operations, evaluator) (12% → target <8%)

### Track 1: Compilation Caching (Highest Impact)

**Problem:** Module cache (`[CACHE] std/xml: HIT`) stores parsed ASTs but still runs
type inference, monomorphization, and lowering on every invocation. In batch mode,
this is pure waste — the source hasn't changed.

**Solution:** Extend the module cache to store **fully compiled** module state:
- Type-checked AST with resolved types
- Monomorphized specializations
- Lowered operator forms
- Bytecode image (if compiled)

Cache invalidation: hash of source file content + dependency hashes.

**Files:**
- `internal/pipeline/cache.go` — extend cache to include type-checked state
- `internal/pipeline/module.go` — skip type checking when cache is valid
- `internal/types/` — serialize/deserialize type solutions

**Expected impact:** Eliminates 37% of CPU in batch mode → ~1.7s → ~1.0s for Alice EPUB.

### Track 2: GC Pressure Reduction

**Problem:** The evaluator allocates a new `eval.Value` (interface{}) for every
intermediate result. Go's GC spends 26% of CPU scanning and collecting these.

**Targeted fixes (not a full arena — minimize scope):**

1. **sync.Pool for common value types** — `StringValue`, `IntValue`, `FloatValue`
   are allocated millions of times. Pool and reuse them.

2. **Pre-allocate list backing arrays** — `ListValue.Elements` grows dynamically.
   Pre-size based on expected output.

3. **Reduce map allocations in type checker** — `putTypeTypeMap` and `ApplySubstitution`
   create temporary maps on every call. Use a shared scratch map with reset.

4. **String builder reuse** — XML serialization builds strings with `+` concatenation.
   Use `strings.Builder` pool.

**Files:**
- `internal/eval/values.go` — add sync.Pool for value types
- `internal/types/substitution.go` — reuse scratch maps
- `internal/builtins/xml_impl.go` — optimize Go XML implementations

**Expected impact:** Reduce GC from 26% to ~12% → ~0.8s → ~0.2s saved.

### Track 3: Hot-Path Optimization

**Problem:** Map operations (`mapassign`, `maps.Iter.Next`, `maps.table.Delete`)
are 12% of CPU, likely from environment lookups in the evaluator.

**Targeted fixes:**

1. **Flatten environment lookups** — The evaluator uses chained `map[string]Value`
   environments. For known-depth lookups (local variables), use index-based access.

2. **Batch mode: reuse evaluator state** — Don't recreate the evaluator for each
   batch input if the module is unchanged.

**Files:**
- `internal/eval/eval_evaluator.go` — environment optimization
- `cmd/ailang/main_run.go` — batch mode evaluator reuse

**Expected impact:** Reduce map ops from 12% to ~5% → ~0.2s saved.

### Combined Expected Outcome

| Track | Before (batch) | Saved | After |
|-------|---------------|-------|-------|
| Compilation caching | 1.0s (37%) | ~0.9s | 0.1s |
| GC reduction | 0.7s (26%) | ~0.3s | 0.4s |
| Hot-path optimization | 0.3s (12%) | ~0.15s | 0.15s |
| **Total** | **2.9s** | **~1.35s** | **~1.55s** |

Target: Alice EPUB batch mode **≤800ms** (from 2,933ms).
This puts us at ~10x MarkItDown — acceptable for a typed language.

### Implementation Plan

**Phase 1: Compilation Caching** (~2 days)
- [ ] Extend module cache to store type-checked + lowered state
- [ ] Add source content hashing for cache invalidation
- [ ] Skip type inference when cache hit in batch mode
- [ ] Benchmark: Alice EPUB batch should drop from ~3s to ~2s
- [ ] CPU profile to verify type checking drops below 10%

**Phase 2: GC Reduction** (~1-2 days)
- [ ] Add sync.Pool for StringValue, IntValue, ListValue
- [ ] Pre-allocate list backing arrays where size is known
- [ ] Optimize type checker map allocations (scratch maps)
- [ ] Benchmark: verify GC% drops from 26% to <15%
- [ ] CPU profile before/after

**Phase 3: Hot-Path Optimization** (~1 day)
- [ ] Profile map operations to identify which maps are hottest
- [ ] Optimize environment lookups for local variables
- [ ] Batch mode: reuse evaluator state between inputs
- [ ] Final benchmark sweep: all 3 files, all modes

### Files to Modify/Create

**Modified files:**
- `internal/pipeline/cache.go` — extended caching (~100 LOC)
- `internal/pipeline/module.go` — cache check before type inference (~30 LOC)
- `internal/eval/values.go` — sync.Pool for value types (~50 LOC)
- `internal/types/substitution.go` — scratch map reuse (~30 LOC)
- `internal/eval/eval_evaluator.go` — environment optimization (~50 LOC)
- `cmd/ailang/main_run.go` — batch mode evaluator reuse (~20 LOC)

**Estimated total:** ~280 LOC changes

---

## Benchmarking Protocol

**All performance work MUST follow this protocol (per M-PERF-GOROUTINE-ID lesson):**

### Before Starting Each Phase

```bash
cd /Users/mark/dev/sunholo/ailang-parse

# 1. CPU profile (baseline)
ailang run -cpuprofile /tmp/phase_N_before.prof -batch -caps IO,FS,Env \
  docparse/main.ail -- data/test_files/gutenberg_alice.epub
go tool pprof -top -cum /tmp/phase_N_before.prof | head -20

# 2. Wall-clock benchmarks
time ailang run -batch -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
time ailang run -batch -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_moby_dick.epub
time ailang run -batch -caps IO,FS,Env docparse/main.ail -- data/test_files/stress/docx_10mb.docx

# 3. Competitor comparison
uv run benchmarks/competitors/run_markitdown.py --format epub
```

### After Completing Each Phase

```bash
# 1. CPU profile (verify targeted function eliminated)
ailang run -cpuprofile /tmp/phase_N_after.prof -batch -caps IO,FS,Env \
  docparse/main.ail -- data/test_files/gutenberg_alice.epub
go tool pprof -top -cum /tmp/phase_N_after.prof | head -20

# 2. Same wall-clock benchmarks as before
# 3. Record before/after in changelog and sprint JSON
```

### Sprint Evaluator Integration

The sprint-evaluator will **HARD FAIL** any performance sprint without before/after
profiling data (per scoring_rubric.md category 7).

---

## Success Criteria

- [ ] Alice EPUB ≤ 800ms in batch mode (from 2,933ms)
- [ ] Moby Dick EPUB ≤ 2,000ms in batch mode (from 3,727ms)
- [ ] CPU profile: type checking <10% in batch mode (from 37%)
- [ ] CPU profile: GC <15% (from 26%)
- [ ] All tests passing (`make test`)
- [ ] Benchmark competitor comparison shows ≤10x gap vs MarkItDown
- [ ] Before/after CPU profiles documented (per M-PERF-GOROUTINE-ID lesson)

---

## Testing Strategy

**Unit tests:**
- Module cache hit/miss with type-checked state
- Cache invalidation on source change
- sync.Pool value recycling correctness

**Integration tests:**
- Existing `make test` suite
- Batch mode with multiple files (same module, different inputs)

**Performance tests (MANDATORY):**
- See Benchmarking Protocol above
- All 3 test files, batch mode + normal mode
- Competitor comparison via `run_markitdown.py`

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **sync.Pool vs arena allocator for values** — agent may choose based on profiling results. Start with sync.Pool (simpler), only escalate to arena if insufficient.
- **Which type checker maps to optimize** — agent should profile `putTypeTypeMap` and `ApplySubstitution` to find the hottest paths before optimizing.
- **Cache serialization format** — agent may choose binary vs in-memory (batch mode only needs in-memory).

---

## Non-Goals

**Not attempted in this feature:**
- **Parity with Python/C extensions** — MarkItDown delegates to C-level parsing. We target ≤10x, not 1x.
- **Full bytecode VM for docparse** — 57 effectful builtins still need wiring. Not blocking perf.
- **AOT/native compilation** — Long-term goal, separate design doc.
- **Wiring more builtins to bytecode VM** — Profiling shows this is not the bottleneck.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Caching type-checked state may have subtle correctness issues (stale types) | High | Content-hash invalidation; run full test suite after each cache change |
| sync.Pool may not reduce GC enough (values escape to heap anyway) | Med | Profile after sync.Pool; escalate to arena only if needed |
| Type checker optimization may require deep refactoring | Med | Start with scratch map reuse (localized); defer structural changes |
| MarkItDown comparison is apples-to-oranges (they don't extract structure) | Low | Note in benchmarks: AILANG extracts more features; compare throughput only |

---

## Related Documents

**Implemented:**
- [m-perf-goroutine-id.md](m-perf-goroutine-id.md) — Eliminated 42% CPU tax (prerequisite, done)
- [m-perf3-performance-quick-wins.md](../../implemented/v0_8_1/m-perf3-performance-quick-wins.md) — Prior perf pass
- [m-serve-api-concurrency.md](../../implemented/v0_9_4/m-serve-api-concurrency.md) — Introduced goroutineEvals

**Planned:**
- [m-perf4-bytecode-interpreter.md](../../planned/v1_0_0/m-perf4-bytecode-interpreter.md) — Full bytecode (longer-term)

---

**Document created**: 2026-04-09
**Last updated**: 2026-04-09
