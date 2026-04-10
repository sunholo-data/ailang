# M-PERF6: Runtime Performance Hotspots

**Status**: Planned
**Target**: v0.10.x
**Priority**: P1 (Medium — improves DocParse and all I/O-heavy workloads)
**Estimated**: 2-3 days
**Dependencies**: M-INCREMENTAL-TYPECHECK (implemented), M-PERF-DOCPARSE (implemented)

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
**Last updated**: 2026-04-10
