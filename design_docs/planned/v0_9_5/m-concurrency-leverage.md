# M-CONCURRENCY-LEVERAGE: Leverage Fork() Across All Execution Modes

**Status**: Planned
**Target**: v0.9.5
**Priority**: P1 (High — multiplies the value of M-SERVE-API-CONCURRENCY across the platform)
**Estimated**: 3-4 days
**Dependencies**: M-SERVE-API-CONCURRENCY (d604390e, bbf99d3f, 926e430a — complete)
**Milestone ID**: M-CONCURRENCY-LEVERAGE
**Created**: 2026-03-19
**Source**: Follow-up to serve-api concurrency work — Fork() + thread-safe Environment unlocks parallelism in batch mode, eval harness, and coordinator

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Parallel execution preserves determinism — each Fork is isolated, results are order-independent |
| A2: Replayability | 0 | Same inputs → same outputs regardless of parallelism level |
| A3: Effect Legibility | 0 | Effects remain explicit per-function |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | +2 | Direct improvement — uses the concurrency infrastructure we just built |
| A7: Machines First | +1 | Faster eval cycles = faster AI agent iteration |
| A9: Cost Visibility | +1 | `--parallel N` makes concurrency explicit and configurable |
| A10: Composability | +1 | Fork() composes with all evaluation modes uniformly |

**Net Score: +6** → **Decision: Move forward**

---

## Problem Statement

We built per-request `Fork()` and thread-safe `Environment` for serve-api concurrency. But three other execution modes still run sequentially even though they could safely parallelize:

| Mode | Current | Potential Speedup | Impact |
|------|---------|-------------------|--------|
| **Batch mode** (`ailang run --batch`) | Sequential loop | N inputs on M cores → ~Mx faster | DocParse eval: 46 benchmarks × 3-4s each = 2.5min → ~40s |
| **Eval harness** | Sequential benchmarks | `MaxConcurrent: 10` field exists but unused | Full eval suite: 46 benchmarks → 5x faster |
| **Coordinator** | `Limit: 1` task at a time | `MaxWorktrees: 3` config exists but unused | Multi-agent sprints: 3 parallel tasks |
| **Go embed API** | Already concurrent-safe | N/A | Needs documentation |

### Concrete Impact: DocParse Eval

DocParse runs 46 benchmarks via `ailang eval-run`. Each benchmark:
1. Compiles the AILANG program (~200ms)
2. Runs the evaluation (~2-4s per document)
3. Compares against golden output

Total: ~2.5 minutes sequential. With 8-core parallel: ~40 seconds.

### Concrete Impact: Batch Mode

```bash
# DocParse processes 100 documents:
ailang run --batch docparse/main.ail doc1.json doc2.json ... doc100.json
# Current: 100 × 50ms = 5s (sequential)
# With --parallel 8: ~700ms
```

---

## Goals

**Primary Goal:** Leverage Fork() to parallelize batch mode, eval harness, and coordinator task execution, reducing wall-clock time by 3-8x for multi-item workloads.

**Success Metrics:**
- `ailang run --batch --parallel 8` processes 8 inputs concurrently
- Eval harness uses `MaxConcurrent` config for parallel benchmark execution
- Coordinator processes up to `MaxWorktrees` tasks concurrently
- Go embed API has documented concurrency examples
- No regressions — sequential mode unchanged when `--parallel 1` (default)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Default parallelism level | Too high = resource exhaustion; too low = no benefit | human | design | low |
| Error handling: fail-fast vs collect-all | Batch: should one failure stop all? | human | design | med |
| Output ordering: preserve input order vs completion order | Affects downstream tooling that parses output | human | design | med |

### Design Freeze

- [x] Default parallelism: `--parallel 1` (sequential, opt-in parallelism)
- [ ] Error handling: fail-fast for batch (stop on first error), collect-all for eval
- [ ] Output ordering: preserve input order for batch (deterministic), any order for eval (speed)

---

## Solution Design

### Phase 1: Batch Mode Parallelism (~1 day)

Add `--parallel N` flag to `ailang run --batch`:

```bash
# Sequential (default, backward compatible)
ailang run --batch prog.ail input1 input2 input3

# Parallel with 4 workers
ailang run --batch --parallel 4 prog.ail input1 input2 input3

# Parallel with CPU count
ailang run --batch --parallel 0 prog.ail input1 input2 input3  # 0 = runtime.NumCPU()
```

**Implementation:**

```go
// cmd/ailang/main_run.go — executeBatchMode()

func executeBatchParallel(ctx context.Context, result *pipeline.Result, inputs []string, parallel int) error {
    if parallel <= 0 {
        parallel = runtime.NumCPU()
    }

    sem := make(chan struct{}, parallel)
    var mu sync.Mutex
    var firstErr error
    var wg sync.WaitGroup

    // Results indexed by input order for deterministic output
    results := make([]batchResult, len(inputs))

    for i, input := range inputs {
        sem <- struct{}{} // acquire semaphore
        wg.Add(1)
        go func(idx int, inp string) {
            defer wg.Done()
            defer func() { <-sem }() // release semaphore

            // Each goroutine gets its own runtime + evaluator (via Fork)
            res, err := executeBatchItem(ctx, result, inp, ...)
            results[idx] = batchResult{output: res, err: err}

            // Fail-fast: cancel remaining on first error
            if err != nil {
                mu.Lock()
                if firstErr == nil {
                    firstErr = fmt.Errorf("batch item %d (%s): %w", idx, inp, err)
                }
                mu.Unlock()
            }
        }(i, input)
    }
    wg.Wait()

    // Output in input order (deterministic)
    for _, r := range results {
        if r.output != nil {
            fmt.Println(r.output.String())
        }
    }
    return firstErr
}
```

**Key design points:**
- Semaphore limits concurrent goroutines (not unbounded)
- Results stored in indexed slice → output in input order
- Each goroutine creates its own `ModuleRuntime` (current batch mode already does this for isolation)
- `Fork()` not needed here because batch already creates fresh runtimes per item — but the thread-safe `Environment` ensures shared stdlib modules are safe

**Files:**
- `cmd/ailang/main_run.go` (~+60 LOC) — Add `--parallel` flag, `executeBatchParallel()`

### Phase 2: Eval Harness Concurrent Benchmarks (~1 day)

The `AgentBenchmarkConfig` already has `MaxConcurrent: 10` — wire it up.

```go
// internal/eval_harness/runner.go

func RunBenchmarkSuite(specs []BenchmarkSpec, config AgentBenchmarkConfig) []BenchmarkResult {
    results := make([]BenchmarkResult, len(specs))
    sem := make(chan struct{}, config.MaxConcurrent)
    var wg sync.WaitGroup

    for i, spec := range specs {
        sem <- struct{}{}
        wg.Add(1)
        go func(idx int, s BenchmarkSpec) {
            defer wg.Done()
            defer func() { <-sem }()

            // Each benchmark gets its own workspace directory
            results[idx] = RunAgentBenchmark(s, config)
        }(i, spec)
    }
    wg.Wait()
    return results
}
```

**Key design points:**
- Each benchmark already runs in its own temp directory (workspace isolation)
- Agent benchmarks spawn external processes (`ailang run`, `claude`) — naturally isolated
- `MaxConcurrent` limits parallel benchmarks (default 10, configurable)
- Results collected in order for deterministic reporting

**Files:**
- `internal/eval_harness/runner.go` (~+40 LOC) — Add `RunBenchmarkSuite()` with goroutine pool
- `cmd/ailang/eval_benchmark.go` (~+10 LOC) — Wire suite runner

### Phase 3: Coordinator Multi-Task (~1 day)

Change `Limit: 1` to honor `MaxWorktrees` config:

```go
// internal/coordinator/daemon_tasks_exec.go

func (d *Daemon) executeTaskQueue() {
    // Get pending tasks up to MaxWorktrees limit
    tasks, err := d.taskStore.ListTasks(TaskFilter{
        Status: "pending",
        Limit:  d.config.MaxWorktrees, // was: 1
    })

    var wg sync.WaitGroup
    for _, task := range tasks {
        wg.Add(1)
        go func(t Task) {
            defer wg.Done()
            d.executeTask(t) // each task gets its own worktree
        }(task)
    }
    wg.Wait()
}
```

**Key design points:**
- Each task already uses its own git worktree (isolated workspace)
- SQLite task store is already concurrent-safe
- Event broadcasting is per-task (no shared state)
- `MaxWorktrees` config already exists (default 3)

**Files:**
- `internal/coordinator/daemon_tasks_exec.go` (~+15 LOC) — Goroutine pool for task execution

### Phase 4: Documentation (~0.5 day)

Document the Go embed API's concurrency safety:

```go
// Example: concurrent AILANG calls from Go
eng := embed.New("/path/to/modules")
defer eng.Close()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(n int) {
        defer wg.Done()
        result, err := eng.Call("math", "fibonacci", n)
        // Each call gets its own Fork()'d evaluator — fully isolated
    }(i)
}
wg.Wait()
```

**Files:**
- `docs/docs/guides/go-interop.md` (~+50 LOC) — Concurrency section
- `docs/docs/guides/serve-api.md` (~+20 LOC) — Cloud Run concurrency note
- `examples/go_embed_concurrent/` (~30 LOC) — Working example

---

## Files to Modify/Create

**Phase 1 — Batch Mode:**
- `cmd/ailang/main_run.go` (~+60 LOC) — `--parallel` flag + parallel batch execution
- `cmd/ailang/help.go` (~+2 LOC) — Document `--parallel` flag

**Phase 2 — Eval Harness:**
- `internal/eval_harness/runner.go` (~+40 LOC) — `RunBenchmarkSuite()` with goroutine pool
- `cmd/ailang/eval_benchmark.go` (~+10 LOC) — Wire suite runner

**Phase 3 — Coordinator:**
- `internal/coordinator/daemon_tasks_exec.go` (~+15 LOC) — Multi-task goroutine pool

**Phase 4 — Documentation:**
- `docs/docs/guides/go-interop.md` (~+50 LOC) — Concurrency section
- `docs/docs/guides/serve-api.md` (~+20 LOC) — Cloud Run deployment note
- `examples/go_embed_concurrent/main.go` (~+30 LOC) — Example

---

## Examples

### Batch Mode

```bash
# Before: 100 documents × 50ms = 5 seconds
$ time ailang run --batch docparse/main.ail data/*.json
real    5.1s

# After: 100 documents on 8 cores ≈ 700ms
$ time ailang run --batch --parallel 8 docparse/main.ail data/*.json
real    0.7s
```

### Eval Harness

```bash
# Before: 46 benchmarks × 3s = 2.5 minutes
$ time ailang eval-run --suite medium
real    2m30s

# After: 46 benchmarks, 10 concurrent = ~30 seconds
$ time ailang eval-run --suite medium
real    0m28s
```

### Go Embed

```go
// Before: serialize or risk data races
eng := embed.New(basePath)
for _, input := range inputs {
    result, _ := eng.Call("mod", "func", input) // one at a time
}

// After: safe concurrent calls
var wg sync.WaitGroup
for _, input := range inputs {
    wg.Add(1)
    go func(in string) {
        defer wg.Done()
        result, _ := eng.Call("mod", "func", in) // Fork() per call
    }(input)
}
wg.Wait()
```

---

## Success Criteria

- [ ] `ailang run --batch --parallel 4 prog.ail a b c d` processes 4 inputs concurrently
- [ ] `--parallel 0` uses `runtime.NumCPU()`
- [ ] `--parallel 1` (default) is sequential — no behavior change
- [ ] Output order matches input order regardless of parallelism
- [ ] Eval harness runs benchmarks up to `MaxConcurrent` in parallel
- [ ] Coordinator processes up to `MaxWorktrees` tasks concurrently
- [ ] `go test -race` passes for all modified packages
- [ ] Go embed concurrency documented with example
- [ ] No regressions in sequential mode

---

## Testing Strategy

**Unit tests:**
- Batch: parallel execution with known inputs, verify output order
- Eval harness: concurrent benchmark runner with mock specs
- Environment: existing `env_concurrent_test.go` covers the foundation

**Integration tests:**
- Batch: `--parallel 4` on 20 inputs, verify all results correct
- Eval harness: run 10 benchmarks with `MaxConcurrent: 5`, verify all complete

**Race detection:**
- `go test -race` on all modified packages
- Manual: `go build -race ./cmd/ailang/ && ailang run --batch --parallel 8 ...`

---

## Deferred Decisions

- **Batch error mode** — fail-fast vs collect-all. Agent may choose (recommendation: fail-fast with `--parallel-errors=collect` flag for override).
- **Progress reporting** — batch parallel progress bar. Agent may choose to add or defer.
- **Coordinator task priority** — when running multiple tasks, should higher-priority tasks preempt? Defer to coordinator design.

## Non-Goals

- **Intra-function parallelism** — parallelizing `map(f, xs)` within a single request. Different problem (requires work-stealing scheduler).
- **Distributed execution** — spreading work across multiple machines. Out of scope.
- **Async/await in AILANG** — language-level concurrency primitives. Separate design doc.
- **Shared mutable state between parallel items** — each Fork is isolated by design.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Batch parallel increases memory usage (N runtimes) | Med | Semaphore limits N; each runtime is ~2MB |
| Eval harness parallel may hit API rate limits | Low | `RequestsPerSecond` config field exists (wire it up) |
| Coordinator parallel tasks may conflict on git worktrees | Low | Worktree manager already uses per-agent directories |
| Output interleaving in batch mode | Med | Collect results in indexed slice, output in order after all complete |

---

## Related Documents

- [M-SERVE-API-CONCURRENCY](../v0_9_4/m-serve-api-concurrency.md) — Foundation: Fork() + thread-safe Environment
- [M-SERVE-API-DX](../v0_9_4/m-serve-api-dx.md) — serve-api production readiness
- [M-PERF7: DocParse Production Pipeline](../v0_9_3/m-perf7-docparse-production-pipeline.md) — Batch mode (M-PERF7 M3)
- [M-EVAL-GUARD: Eval Process Guardrails](../../implemented/v0_5_6/m-eval-process-guardrails.md) — Eval process isolation

---

## Future Work

- **Streaming batch output** — output results as they complete instead of waiting for all (useful for long-running batches)
- **Adaptive parallelism** — auto-tune based on system load / available memory
- **Intra-request parallelism** — `parMap(f, xs)` builtin that evaluates `f` on list elements concurrently
- **Distributed eval** — spread eval benchmarks across multiple machines via coordinator

---

**Document created**: 2026-03-19
**Last updated**: 2026-03-19
