# M-BYTECODE-2D Benchmarks: Bytecode VM vs Tree-Walking Evaluator

Milestone: **M5_BENCH** (M-BYTECODE-VM Phase 2D)
Date: 2026-04-08
Harness: [tests/golden/bytecode/bench_test.go](../../../tests/golden/bytecode/bench_test.go)

## Summary

Steady-state, per-call throughput of the register-based bytecode VM against
the tree-walking evaluator on a small curated slice of the golden corpus
plus a recursive `fib` throughput case. Compile cost is amortized outside
`b.N` — the numbers below measure dispatch + execution, not cold-start
compile time.

**Headline result**: on the recursive `fib(25)` case the bytecode VM is
**~26.3x faster** than the evaluator, with **~55x fewer heap allocations**.
`fib(30)` holds the same speedup (~25.7x, ~55x fewer allocs). Pure
arithmetic cases sit in the 24–30x range; the single degenerate case is
`identity` (4.5x) where per-call fixed overhead dominates on both sides
and the VM's register file allocation still shows up.

## Environment

- Machine: Apple M2 (`darwin/arm64`), Go default toolchain
- `benchtime=1s` for the full table, `benchtime=2s` for the dedicated fib run
- No parallelism flags — single-goroutine measurements

## Results

| Case          | Bytecode VM ns/op | Evaluator ns/op |   Speedup | Bytecode allocs/op | Eval allocs/op | Alloc reduction |
|---------------|------------------:|----------------:|----------:|-------------------:|---------------:|----------------:|
| Identity      |              51.3 |           229.5 |    **4.5x** |                  2 |              3 |            1.5x |
| AddInts       |              83.5 |           2 062 |   **24.7x** |                  2 |             66 |             33x |
| Classify      |             138.8 |           1 975 |   **14.2x** |                  2 |             63 |           31.5x |
| Clamp         |             134.4 |           3 892 |   **29.0x** |                  2 |            119 |           59.5x |
| Factorial(10) |           1 875   |          56 454 |   **30.1x** |                 29 |          1 827 |             63x |
| Fib(20)       |       4 458 493   |     115 235 458 |   **25.8x** |             65 684 |      3 601 526 |             55x |
| Fib(25)       |      49 669 842   |   1 308 723 291 |   **26.3x** |            728 494 |     39 943 593 |             55x |
| Fib(30)       |     555 340 375   |  14 276 332 583 |   **25.7x** |          8 079 167 |    442 981 918 |             55x |

Throughput cases (`-bench=BenchmarkFib -benchtime=2s`):

```
BenchmarkFib/Bytecode/Fib20-8         562     4 237 711 ns/op      10 858 298 B/op       65 683 allocs/op
BenchmarkFib/Evaluator/Fib20-8         20   110 523 004 ns/op     107 685 817 B/op    3 601 399 allocs/op
BenchmarkFib/Bytecode/Fib25-8          51    46 464 332 ns/op     120 425 560 B/op      728 476 allocs/op
BenchmarkFib/Evaluator/Fib25-8          2 1 241 420 708 ns/op   1 194 348 036 B/op   39 942 901 allocs/op
BenchmarkFib/Bytecode/Fib30-8           4   516 612 760 ns/op   1 335 544 144 B/op    8 078 951 allocs/op
BenchmarkFib/Evaluator/Fib30-8          1 13 915 022 333 ns/op  13 245 595 976 B/op  442 975 310 allocs/op
```

## Observations

### Where the speedup comes from

1. **No closure/environment allocation per call**. The evaluator builds a
   fresh `*Env` chain on every `CallFunction` and references it via pointer
   links; the VM allocates a single `Frame` with a flat `Regs []Value` slice
   and dispatches via integer register operands. For recursive workloads
   the per-call `Env.Clone` cost dominates the evaluator — hence the
   "55x fewer allocs" line on fib.

2. **Direct integer operand dispatch**. The VM reads opcodes from a
   `[]bytecode.Instruction` slice and operand registers as bytes, versus
   the evaluator walking a `core.Expr` AST with type switches at every
   node. The opcode dispatch loop in [internal/vm/vm.go](../../../internal/vm/vm.go)
   is a straight `for { switch op { case OpX: ... } }`, which Go compiles
   into a dense jump table.

3. **Arithmetic is monomorphic at the bytecode layer**. `OpAddInt`/`OpMulInt`
   are direct `int64` additions; the evaluator dispatches through dictionary
   resolution for typeclass methods (`Num.add`, `Num.mul`) on every call.
   This is why `AddInts` jumps from 4.5x (identity) to 24.7x — introducing
   one arithmetic op is enough to amortize the evaluator's per-node cost.

### Where the VM is NOT that much faster

- **`Identity`**: 51.3 vs 229.5 ns/op (4.5x). Both sides hit pure fixed
  overhead — arg marshalling, frame/env creation, one instruction body,
  return. The VM's 2 allocs/op are the frame itself plus the register
  slice. There is no obvious path to cheaper than this without pooling
  frames (tracked for M-BYTECODE-2E).

### What this does NOT measure

- **Cold-start compile time**. Both pipelines' compile cost is outside the
  `b.N` loop. In the evaluator case this is ~nothing (the module instance
  is built once by `LoadAndEvaluate`); in the bytecode case the
  `lower → compile` pass is nontrivial and would matter for one-shot
  script execution. A dedicated cold-start benchmark is Phase 2E scope.

- **Effect-trap / bridge dispatch**. All cases above are pure — no `IO`,
  no ADT pattern matching that would hit the evaluator bridge. The
  `--bytecode` CLI path already falls through to the bridge for these,
  and the M3 parity test exercises that surface. Bridge dispatch adds the
  eval path's cost back in plus one value-conversion per argument/result,
  so it will sit closer to the evaluator baseline than to the pure VM
  numbers here. Phase 2E will add a dedicated "bridge throughput"
  benchmark once closure/ADT support lands.

## Reproducing

```bash
# Full table
go test -run=^$ -bench=BenchmarkBytecodeVsEvaluator -benchmem -benchtime=1s ./tests/golden/bytecode/

# Fib throughput only (longer run for stability)
go test -run=^$ -bench=BenchmarkFib -benchmem -benchtime=2s ./tests/golden/bytecode/
```

## Sprint commit summary

> **M5_BENCH: bytecode VM measures ~26x faster than evaluator on fib(25),
> ~30x on factorial(10); ~55x fewer allocations on recursive workloads.**

This is the number that goes in the M-BYTECODE-VM Phase 2D sprint commit
message per the milestone acceptance criterion.
