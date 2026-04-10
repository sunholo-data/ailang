# Sprint Plan: M-PERF-DOCPARSE

## Summary
Close the Python performance gap for AILANG docparse by eliminating redundant type checking in batch mode, reducing GC pressure via value pooling, and optimizing hot-path map operations. Target: Alice EPUB ≤800ms batch (from 2,933ms).

**Duration:** 3 days (estimated 6-8 hours active work)
**Dependencies:** M-PERF-GOROUTINE-ID (done)
**Risk Level:** Medium (cache correctness requires careful testing)
**Design Doc:** [m-perf-docparse.md](m-perf-docparse.md)

## Current Status Analysis

### Completed Recently
- ✅ M-PERF-GOROUTINE-ID: ~200 LOC in 1 day (2.5-3.6x speedup)
- ✅ Sprint evaluator perf checks: ~90 LOC

### Velocity
- Recent average: ~400 LOC/day (implementation + tests)
- Estimated capacity: ~280 LOC implementation + ~150 LOC tests = ~430 LOC total
- Conservative buffer: 3 days for ~430 LOC

### Current Benchmarks (post-goroutine-ID fix)
| File | Normal | Batch | MarkItDown | Gap |
|------|--------|-------|------------|-----|
| Alice EPUB (185KB) | 5,028ms | 2,933ms | 78ms | 38x |
| Moby Dick EPUB (797KB) | 9,478ms | 3,727ms | 190ms | 20x |

### CPU Profile (Batch Mode, Alice EPUB)
| Component | % CPU | Target |
|-----------|-------|--------|
| Type checking | 37% | <5% (cache) |
| GC pressure | 26% | <15% (pools) |
| Map operations | 12% | <8% (optimize) |

## Proposed Milestones

### Milestone 1: M1_COMPILATION_CACHE — Skip Redundant Type Checking
**Goal:** Cache fully compiled module state (type-checked + lowered), skip recompilation on cache hit in batch mode
**Estimated:** 130 LOC implementation + 60 LOC tests = 190 LOC
**Duration:** 1 day

**Context:**
- `pipeline_module.go:195-221` — current cache lookup (only stores interface digest, not type-checked state)
- `cache_store.go:20-38` — CacheEntry stores IfaceDigest + IfaceJSON but not compiled module
- `cache_key.go:22-38` — content hash + dependency hash invalidation (reusable)
- Batch mode in `main_run.go:383` compiles once but still runs full type inference

**Tasks:**
1. Extend `CacheEntry` in `cache_store.go` to store compiled module state (elaborated AST, type solutions, lowered forms)
2. In `pipeline_module.go`, after cache hit, skip type inference + monomorphization + lowering — return cached compiled state
3. Add cache invalidation test: modify source → cache miss → recompile
4. CPU profile before/after: type checking should drop from 37% to <5%
5. Benchmark: Alice EPUB batch should drop from ~3s to ~2s

**Acceptance Criteria:**
- [ ] Batch mode skips type checking for unchanged modules (verified via log output or profile)
- [ ] CPU profile shows type checking <10% in batch mode
- [ ] Alice EPUB batch ≤2,000ms
- [ ] Cache invalidation works correctly (source change → recompile)
- [ ] `make test` passes — no regressions
- [ ] Before/after CPU profiles captured

**Risks:**
- Stale cached types causing subtle correctness bugs — Mitigation: content-hash invalidation + full test suite
- Cached state may be large in memory — Mitigation: batch mode only, not persisted to disk

### Milestone 2: M2_GC_REDUCTION — Reduce Allocations and GC Pressure
**Goal:** Pool frequently-allocated value types and reduce temporary map allocations in type checker
**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** 1 day

**Context:**
- `eval/value.go` — 13 concrete Value types, all pointer receivers (heap-allocated)
- `types/pool.go:1-65` — already has 3 sync.Pool instances for type maps (M-PERF6)
- `types/unification_substitution.go:8` — ApplySubstitution uses pooled visited maps
- GC is 26% of CPU — `scanobject`, `mallocgc`, `gcDrain`

**Tasks:**
1. Add sync.Pool for `StringValue` and `IntValue` in `eval/value.go` (highest-frequency allocations)
2. Add `Release()` method pattern — evaluator returns values to pool after use
3. Pre-allocate `ListValue.Elements` slices where output size is predictable (XML builtins)
4. Optimize string building in XML builtins — replace `+` concatenation with `strings.Builder`
5. CPU profile before/after: GC should drop from 26% to <15%

**Acceptance Criteria:**
- [ ] sync.Pool wired for StringValue and IntValue
- [ ] CPU profile shows GC <18% (from 26%)
- [ ] No use-after-free bugs (pooled values not shared across goroutines)
- [ ] `make test` passes
- [ ] Before/after CPU profiles captured

**Risks:**
- Pooled values escaping to concurrent contexts — Mitigation: only pool in single-threaded batch path
- Minimal GC improvement if values escape to heap anyway — Mitigation: profile first, escalate if needed

### Milestone 3: M3_HOTPATH_AND_BENCHMARK — Optimize Map Ops + Final Verification
**Goal:** Optimize environment lookups, reuse evaluator state in batch mode, run final benchmark sweep
**Estimated:** 70 LOC implementation + 40 LOC tests = 110 LOC
**Duration:** 1 day

**Context:**
- `eval/env.go:10-93` — chained `map[string]Value` with parent pointer, `sync.RWMutex`
- `eval/eval_evaluator.go:126` — `Fork()` creates fresh evaluator per request
- `main_run.go` — batch mode creates fresh runtime per input
- Map ops are 12% of CPU

**Tasks:**
1. Profile to identify which map operations are hottest (environment lookups vs type checker maps)
2. In batch mode, reuse evaluator state between inputs (skip `Fork()` overhead for sequential batch)
3. Optimize `Environment.Get()` for local variable lookups (avoid parent chain walk for depth-0)
4. Final benchmark sweep: all 3 files, batch + normal mode
5. Competitor comparison via `run_markitdown.py`
6. Update CHANGELOG.md with results
7. Update design doc status

**Acceptance Criteria:**
- [ ] Alice EPUB batch ≤800ms (primary target)
- [ ] Moby Dick EPUB batch ≤2,000ms
- [ ] CPU profile: no single component >30%
- [ ] All 3 benchmark files tested, results documented
- [ ] Competitor comparison shows ≤10x gap vs MarkItDown
- [ ] CHANGELOG.md updated
- [ ] Design doc moved to implemented/ or status updated
- [ ] `make test` passes
- [ ] Before/after CPU profiles for all 3 phases documented

**Risks:**
- Combined optimizations may not reach ≤800ms target — Mitigation: each phase is independently valuable; reassess target if needed

## Success Metrics
- Alice EPUB batch: ≤800ms (from 2,933ms) — **3.7x improvement needed**
- Moby Dick EPUB batch: ≤2,000ms (from 3,727ms) — **1.9x improvement needed**
- CPU profile balanced: no component >30%
- All tests passing
- Before/after profiles for each phase (sprint evaluator requirement)

## Dependencies
- M-PERF-GOROUTINE-ID ✅ (done — eliminated 42% CPU tax)
- ailang-parse benchmark files at `/Users/mark/dev/sunholo/ailang-parse/data/test_files/`
- `-cpuprofile` flag available (added in M-PERF-GOROUTINE-ID)

## Benchmarking Protocol
Per design doc — before AND after each milestone:
```bash
cd /Users/mark/dev/sunholo/ailang-parse
ailang run -cpuprofile /tmp/mN_before.prof -batch -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
go tool pprof -top -cum /tmp/mN_before.prof | head -20
time ailang run -batch -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
time ailang run -batch -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_moby_dick.epub
```

## Notes
- This sprint focuses on batch mode performance (the primary docparse use case)
- Normal (single-file) mode will also benefit from M2 and M3 but not from M1
- sync.Pool approach chosen over arena allocator (simpler, lower risk, can escalate later)
- The 800ms target is ambitious — if M1 alone gets us to ~1.5s that's still a major win
