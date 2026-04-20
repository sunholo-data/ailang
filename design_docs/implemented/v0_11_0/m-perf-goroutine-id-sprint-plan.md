# Sprint Plan: M-PERF-GOROUTINE-ID

## Summary
Eliminate the `runtime.Stack()` bottleneck that taxes every AILANG builtin call by ~42% of CPU time. Replace with cheap goroutine ID extraction (`goid` package) and add a fast-path that skips the lookup entirely for single-goroutine programs.

**Duration:** 1 day (3-4 hours active work)
**Dependencies:** None
**Risk Level:** Low — pure performance optimization, no semantic changes
**Design Doc:** `design_docs/planned/v0_11_0/m-perf-goroutine-id.md`

## Current Status Analysis

### Completed Recently
- ✅ M-BYTECODE-XML-BUILTINS: ~1200 LOC in 1 session (17 XML builtins)
- ✅ M-BYTECODE-PURE-EFFECTS: ~950 LOC in 1 session (3 JSON builtins)
- ✅ M-BYTECODE-REGALLOC-FIX: ~280 LOC in 1 session (register allocation bugs)

### Velocity
- Recent average: ~1000 LOC/day (high pace, recent sprints)
- This sprint is small: ~80 LOC changes + ~60 LOC tests = ~140 LOC total
- Well within single-session capacity

### Problem Discovery
- CPU profile of docparse Alice EPUB revealed:
  - **42% of CPU** in `runtime.Stack()` / `builtinGoroutineID()`
  - Called on **every** builtin invocation via `getEffContext()`
  - Regression introduced in v0.9.4 (M-SERVE-API-CONCURRENCY)
  - Never caught because no CPU profiling in sprint evaluation

### Baseline Benchmarks (before fix)

| Benchmark | Evaluator | Bytecode | Target |
|-----------|-----------|----------|--------|
| Alice EPUB (185KB) | 12.1s | 12.0s | ≤8s |
| Moby Dick EPUB (797KB) | 35.1s | 33.8s | proportional |
| 10MB DOCX | 4.2s | 4.2s | proportional |

## Proposed Milestones

### Milestone 1: M1_GOID_FASTPATH — Replace runtime.Stack with goid + fast-path
**Goal:** Eliminate the `runtime.Stack()` bottleneck and add single-goroutine fast-path
**Estimated:** ~60 LOC implementation + ~40 LOC tests = ~100 LOC
**Duration:** ~2 hours

**Tasks:**
1. Add `github.com/petermattis/goid` dependency (`go get`)
2. In `internal/runtime/builtins.go`:
   - Replace `builtinGoroutineID()` body with `return goid.Get()`
   - Add `goroutineEvalCount atomic.Int64` field to `BuiltinRegistry`
   - Increment in `SetGoroutineEvaluator`, decrement in `ClearGoroutineEvaluator`
   - Add fast-path in `getEffContext()`: if `goroutineEvalCount == 0`, skip sync.Map lookup
3. Write unit tests:
   - Fast-path returns shared evaluator context when no forks registered
   - After `SetGoroutineEvaluator`, returns forked evaluator context
   - After `ClearGoroutineEvaluator`, fast-path reactivates
4. Run `make test` — all existing tests must pass

**Acceptance Criteria:**
- [ ] `builtinGoroutineID()` no longer calls `runtime.Stack()`
- [ ] Fast-path skips sync.Map when `goroutineEvalCount == 0`
- [ ] New unit tests pass for fast-path behavior
- [ ] `make test` passes (no regressions)
- [ ] `make lint` clean

**Risks:**
- `goid` package may not build on all CI platforms → Mitigation: package supports linux/darwin/windows on amd64/arm64, matches our CI matrix

### Milestone 2: M2_CLEANUP — Remove duplicate goroutineID implementations
**Goal:** Consolidate 4 independent `runtime.Stack()` goroutine ID implementations
**Estimated:** ~20 LOC changed (mostly deletions)
**Duration:** ~30 min

**Tasks:**
1. `internal/runtime/entrypoint.go`: Replace `goroutineID()` with `goid.Get()` (debug logging only)
2. `internal/apiserver/routes.go`: Remove unused `goroutineID()` function
3. `internal/embed/embed.go`: Replace `embedGoroutineID()` with `goid.Get()`
4. Run `make test` and `make lint`

**Acceptance Criteria:**
- [ ] No `runtime.Stack()` calls remain for goroutine ID extraction
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- `apiserver/routes.go` goroutineID may have callers we missed → Mitigation: compiler will catch; grep first

### Milestone 3: M3_BENCHMARK_VERIFY — Profile and benchmark to confirm fix
**Goal:** Verify the bottleneck is eliminated and measure actual speedup
**Estimated:** ~20 LOC (changelog entry)
**Duration:** ~1 hour

**Tasks:**
1. `make quick-install` to update binary
2. Run CPU profile: `ailang run -cpuprofile /tmp/alice_after.prof -bytecode -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub`
3. Verify: `go tool pprof -top -cum /tmp/alice_after.prof | head -20` — `runtime.Stack` must be absent from top-20
4. Run all 3 benchmarks (Alice EPUB, Moby Dick EPUB, 10MB DOCX) with both evaluator and bytecode
5. Record results in changelog
6. Update design doc with final benchmarks

**Acceptance Criteria:**
- [ ] `runtime.Stack` / `traceback` absent from top-20 CPU profile entries
- [ ] Docparse Alice EPUB ≤ 8s (evaluator) — down from 12s
- [ ] Moby Dick EPUB shows proportional improvement
- [ ] Changelog updated with before/after benchmarks
- [ ] Design doc updated with results

**Benchmark commands (from ailang-parse directory):**
```bash
cd /Users/mark/dev/sunholo/ailang-parse

# Evaluator
time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_moby_dick.epub
time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/stress/docx_10mb.docx

# Bytecode
time ailang run -bytecode -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
time ailang run -bytecode -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_moby_dick.epub
time ailang run -bytecode -caps IO,FS,Env docparse/main.ail -- data/test_files/stress/docx_10mb.docx

# CPU profile
ailang run -cpuprofile /tmp/after.prof -bytecode -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
go tool pprof -top -cum /tmp/after.prof | head -20
```

**Risks:**
- Speedup may be less than 42% if GC or evaluator overhead increases proportionally → Mitigation: profile will show where time shifted

## Success Metrics
- `runtime.Stack` eliminated from CPU hot path
- Alice EPUB: ≤8s (from 12s) in evaluator mode
- All tests passing (`make test`)
- Linting clean (`make lint`)
- Changelog updated

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `go.mod` / `go.sum` | Add `goid` dependency | ~2 |
| `internal/runtime/builtins.go` | Replace goroutineID, add fast-path, add atomic counter | ~40 |
| `internal/runtime/builtins_test.go` | New: fast-path unit tests | ~40 |
| `internal/runtime/entrypoint.go` | Replace `goroutineID()` with `goid.Get()` | ~-10 |
| `internal/apiserver/routes.go` | Remove dead `goroutineID()` | ~-10 |
| `internal/embed/embed.go` | Replace `embedGoroutineID()` | ~-5 |
| `changelogs/v0.11-current.md` | Benchmark results | ~15 |

**Total: ~140 LOC (including deletions and tests)**
