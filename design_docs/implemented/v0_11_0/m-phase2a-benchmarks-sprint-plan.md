# Sprint Plan: M-PHASE2A-BENCH — Evaluator Performance Benchmarks

## Summary

Benchmark the AILANG evaluator against native Go across 7 workloads to decide whether to build the bytecode VM or ship the embedded evaluator. This is the Phase 2A gate from the M-BYTECODE-VM design doc (Section 9).

**Duration:** 3 days
**Dependencies:** M-CODEGEN-STRATEGIC-REVIEW (complete), M-BYTECODE-VM design doc (complete)
**Risk Level:** Low (read-only measurement, no production code changes)
**Design Doc:** [m-bytecode-vm.md](m-bytecode-vm.md) §9

## Current Status Analysis

### Completed Recently
- M-STD-STRING-PERF: ~650 LOC in 1 day (string performance primitives)
- M-CODEGEN-IR (M1-M5): ~1200 LOC in 3 days (Statement IR architecture)
- M-BYTECODE-VM design doc: reviewed and finalized with semantic contracts

### Velocity
- Recent average: ~300 LOC/day
- Estimated capacity: ~900 LOC for 3-day sprint
- This sprint is benchmark-heavy (test code + AILANG programs), not production code

### Context

The M-BYTECODE-VM design doc defines three decision outcomes based on evaluator performance:
1. **< 5x native Go** on all workloads → ship evaluator, defer bytecode
2. **> 10x native Go** on hot loops (fib, map/filter) → build bytecode VM
3. **Miss across the board** → build bytecode VM with higher priority

### Existing Infrastructure
- `pipeline.RunWithContext()` — full compile-and-evaluate pipeline (entry point for benchmarks)
- `internal/eval/` — TypedEvaluator, SimpleEvaluator, CoreEvaluator
- `internal/builtins/` — existing Go benchmarks for XML/string ops (pattern to follow)
- No `ailang bench` subcommand — benchmarks will be Go `testing.B` tests
- 46 YAML eval benchmarks exist but test AI code gen quality, not runtime performance

## Proposed Milestones

### Milestone 1: AILANG Benchmark Programs + Go Baselines
**Goal:** Write 7 AILANG programs and their native Go equivalents as `testing.B` benchmarks
**Estimated:** 250 LOC (AILANG) + 200 LOC (Go benchmarks) = ~450 LOC
**Duration:** 1 day

**Tasks:**
1. Create `benchmarks/runtime/` directory for AILANG benchmark source files
2. Write 7 AILANG programs:
   - `fib30.ail` — recursive fib(30) (pure function call overhead)
   - `list_map_filter_10k.ail` — map/filter over 10K list (collection throughput)
   - `pattern_match_nested.ail` — nested ADT match, 5 levels deep (dispatch cost)
   - `closure_curried.ail` — curried HOFs + partial application (closure overhead)
   - `cross_boundary.ail` — compiled pure calling IO helper in loop (VM↔eval boundary)
   - `docparse_pipeline.ail` — representative string processing pipeline (real-world mixed)
   - `game_step.ail` — latency-sensitive frame update loop (p95 latency)
3. Create `internal/eval/runtime_bench_test.go` with native Go baselines:
   - `BenchmarkNative_Fib30` — recursive Go fib
   - `BenchmarkNative_ListMapFilter10K` — Go slice map/filter
   - `BenchmarkNative_PatternMatch` — Go switch on tagged union
   - `BenchmarkNative_ClosureCurried` — Go closure calls
   - `BenchmarkNative_CrossBoundary` — Go func call overhead
   - `BenchmarkNative_DocParsePipeline` — Go strings.Builder pipeline
   - `BenchmarkNative_GameStep` — Go struct update loop

**Acceptance Criteria:**
- [ ] All 7 AILANG programs run correctly via `ailang run`
- [ ] All 7 native Go benchmarks run via `go test -bench=.`
- [ ] Native Go benchmarks produce correct results (verified in test)
- [ ] Each benchmark has `-benchmem` support (allocs/op reported)
- [ ] Linting clean

### Milestone 2: Evaluator Benchmarks + Harness
**Goal:** Create a benchmark harness that runs AILANG programs through the evaluator pipeline and measures throughput, allocs/op, and p95 latency
**Estimated:** ~300 LOC
**Duration:** 1 day

**Tasks:**
1. Create `internal/eval/evaluator_bench_test.go` with pipeline integration:
   - Helper: `benchmarkAILANGFile(b *testing.B, filename string)` — loads .ail, runs pipeline.RunWithContext in b.N loop
   - Helper: `benchmarkAILANGLatency(b *testing.B, filename string)` — captures per-iteration times for p95 calculation
   - `BenchmarkEval_Fib30` through `BenchmarkEval_GameStep` — one per workload
2. **Critical: isolate evaluation time from startup** — parse, type-check, and elaborate the program ONCE in `b.StopTimer()`/setup, then `b.ResetTimer()` before the b.N loop that only measures evaluation. The pipeline produces a typed AST; cache that and re-evaluate it each iteration. We are measuring evaluator throughput, NOT compilation speed.
3. Add `b.ReportAllocs()` and `b.ReportMetric()` for custom metrics (p95 where applicable)
4. Create `Makefile` target: `make bench-phase2a` that runs both native and evaluator benchmarks
5. Ensure benchmarks use `-count=5` for statistical reliability

**Acceptance Criteria:**
- [ ] `make bench-phase2a` runs all 14 benchmarks (7 native + 7 evaluator) to completion
- [ ] Benchmark output includes ns/op, B/op, allocs/op for each
- [ ] p95 latency reported for cross-boundary and game-step workloads
- [ ] No test failures from benchmark files
- [ ] Each evaluator benchmark completes within 60s (no infinite loops)

### Milestone 3: Analysis Script + Decision Report
**Goal:** Produce a structured comparison table and decision recommendation
**Estimated:** ~150 LOC
**Duration:** 1 day

**Tasks:**
1. Create `tools/phase2a_report.sh` — parses `go test -bench` JSON output, computes ratios
   - Input: benchmark output from `make bench-phase2a -json`
   - Output: Markdown table with columns: Workload | Native Go | Evaluator | Ratio | Verdict
   - Decision logic: apply the three rules from §9
2. Run benchmarks and generate `design_docs/planned/v0_11_0/phase2a-results.md`
3. Update M-BYTECODE-VM design doc §9 with actual results
4. Add CHANGELOG entry documenting Phase 2A completion

**Acceptance Criteria:**
- [ ] `tools/phase2a_report.sh` generates valid Markdown table from benchmark JSON
- [ ] Decision report contains all 7 workloads with native/evaluator/ratio columns
- [ ] Report includes clear PASS/FAIL per workload against 5x threshold
- [ ] Report concludes with explicit recommendation (ship evaluator OR build bytecode)
- [ ] CHANGELOG entry added under v0.11.0

## Success Metrics
- All 14 benchmarks run and produce results
- Decision report generated with clear recommendation
- All tests passing
- All linting clean
- Results reproducible (someone else can run `make bench-phase2a`)

## Dependencies
- Working AILANG evaluator (already exists)
- AILANG programs must use features available in v0.10.2+ (pattern matching, ADTs, closures, lists)
- No dependency on Statement IR or Go codegen — this benchmarks the *evaluator* path

## Risks
- **Recursive fib(30) may be slow** — AILANG's tree-walking evaluator will be orders of magnitude slower than Go. This is expected and validates the need for bytecode. Mitigation: use fib(25) if fib(30) exceeds 60s.
- **Pipeline startup overhead** — parsing, type checking, and elaboration MUST be excluded from benchmark timing. Mitigation: compile to typed AST once in setup, then only measure re-evaluation in the b.N loop. Use `b.StopTimer()`/`b.ResetTimer()` to isolate pure evaluation cost.
- **List 10K construction** — may need builtin `range(1, 10000)` or generate list. Mitigation: verify list construction works at scale before benchmarking.

## Notes
- This sprint produces a **decision**, not production code. The benchmarks stay as regression tests.
- If evaluator meets targets (< 5x native Go), bytecode VM work is deferred indefinitely.
- If evaluator misses targets, this data directly feeds Phase 2B planning.
- Benchmarks will live in `internal/eval/` alongside the evaluator code they measure.
- AILANG source files go in `benchmarks/runtime/` to separate from the AI eval YAML specs.
