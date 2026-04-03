# Phase 2A Benchmark Results: Evaluator vs Native Go

**Date:** 2026-04-03
**Machine:** Apple M2, darwin/arm64
**Method:** `go test -bench -benchmem -count=3`, startup time excluded (compile once, evaluate in loop)
**Gate document:** [m-bytecode-vm.md](m-bytecode-vm.md) §9

## Results

| Workload | Native Go | Evaluator | Ratio | Verdict |
|----------|-----------|-----------|-------|---------|
| fib(30) — recursive | 2.6ms | 18.1s | **~6,900x** | FAIL (>10x) |
| map/filter 5K list | 53μs | 10.1s | **~191,000x** | FAIL (>10x) |
| Pattern match (tree depth 12) | 583μs | 250ms | **~429x** | FAIL (>10x) |
| Closure/curried HOFs (1K) | 1.1μs | 485ms | **~441,000x** | FAIL (>10x) |
| String pipeline (500 lines) | 66μs | 798ms | **~12,100x** | FAIL (>10x) |
| Game step (50 entities × 200 frames) | 16μs | 2.04s | **~127,000x** | FAIL (>10x) |
| Cross-boundary (fib15 × 100) | 192μs | 2.75s | **~14,300x** | FAIL (>10x) |

### Allocation Analysis

| Workload | Native allocs/op | Evaluator allocs/op | Ratio |
|----------|-----------------|--------------------:|------:|
| fib(30) | 0 | 445M | ∞ |
| map/filter 5K | 1 | 1.25M | 1.25M× |
| Pattern match | 8,191 | 3.32M | 405× |
| Closure/curried | 0 | 354K | ∞ |
| String pipeline | 1,163 | 222K | 191× |
| Game step | 0 | 6.86M | ∞ |
| Cross-boundary | 0 | 32.8M | ∞ |

### Latency Measurements

| Workload | p95 Latency | Target (16ms/frame) | Status |
|----------|-------------|--------------------:|--------|
| Game step | 2.4s | 16ms | **149x over budget** |
| Cross-boundary | 3.9s | N/A | — |

## Analysis

### Why the ratios are so large

The AILANG evaluator is a **tree-walking interpreter**: every expression node dispatches through Go interface method calls, creates `eval.Value` wrapper objects, and boxes/unboxes on every operation. This is the expected behavior for a language prioritizing correctness and semantic transparency over raw speed.

Key overhead sources:
1. **Value boxing**: Every integer, boolean, and string is heap-allocated as an `eval.Value` interface
2. **Function call overhead**: Each AILANG function call creates a new environment, closure capture, and argument list
3. **Recursive list operations**: `map`, `filter`, `foldl` are implemented recursively with cons-cell allocation
4. **No TCO for non-tail positions**: `fib(n-1) + fib(n-2)` can't be tail-call optimized

### Comparison to design doc thresholds

From [m-bytecode-vm.md §9](m-bytecode-vm.md):

| Rule | Condition | Result |
|------|-----------|--------|
| Ship evaluator | All workloads < 5x native Go | **NOT MET** — all exceed 400x |
| Build bytecode (hot loops) | Hot loops > 10x native Go | **TRIGGERED** — all exceed 10x |
| Build bytecode (high priority) | Miss across the board | **TRIGGERED** — 7/7 workloads fail |

## Decision

**BUILD THE BYTECODE VM.**

The evaluator exceeds the 10x threshold on every workload, most by 3-5 orders of magnitude. This is consistent with tree-walking interpreter performance characteristics and confirms that a bytecode VM is necessary for compute-intensive AILANG programs.

### Priority Assessment

This is a **"miss across the board"** result per the design doc decision rules, which means bytecode VM should be built with **higher priority**. However, the evaluator remains the correct choice for:
- Interactive REPL (single-expression evaluation)
- Effect-heavy programs (IO, Net, FS — dominated by external call latency)
- Correctness reference (semantic authority per §3.6)

### Recommended Next Steps

1. Proceed with **Phase 2B: Bytecode Foundation** from [m-bytecode-vm.md §10](m-bytecode-vm.md)
2. Target initial VM for Tier 1 features (arithmetic, control flow, closures, builtins)
3. Use these benchmarks as regression targets — bytecode should achieve < 5x native Go on pure workloads
4. Keep evaluator as fallback for Tier 3 features (effects, row polymorphism, reflection)

## Reproducibility

```bash
# Run these benchmarks yourself:
make bench-phase2a        # Full run (-count=3, ~5 min)
make bench-phase2a-quick  # Quick run (-count=1, ~2 min)

# Or directly:
go test -run='^$' -bench='Benchmark(Native|Eval)_' -benchmem -count=3 \
  ./internal/eval/ -timeout=600s
```

Benchmark source: [internal/eval/phase2a_bench_test.go](../../internal/eval/phase2a_bench_test.go)
AILANG programs: [benchmarks/runtime/](../../benchmarks/runtime/)
