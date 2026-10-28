### Changed — bytecode VM: no allocation per call or per tail call; inline int arithmetic (M-VM-QUICK-WINS)

- **Shared zero-capture closures.** Every call to a known top-level function used to allocate a closure. A function that captures nothing now has one closure per VM. Closure equality is structural, so no result changes.
- **Tail calls reset the frame in place.** `TAIL_CALL` moves its arguments to the front of the register slab and zeroes the rest, with no argument slice allocated per step.
- **Inline int fast path** for `+ - *`, `<`, `<=` and `==` when both operands are ints. Division and modulo keep their single zero-divisor check.
- Measured end to end under `ailang run --strict-bytecode`, best of 3, against the same-day baseline:
  - `fib(32)`: 0.51 s → 0.33 s (−35%)
  - 10M-step tail-recursive loop: 1.10 s → 0.55 s (−50%)
  - map + fold over 2M ints: 0.18 s → 0.16 s
  - VM allocations: `fib(25)` 11.66 MB → 12.7 KB, 1M tail loop 176 MB → 3.8 KB.
- **New instruments.**
  - perf-sweep now banks `vm_fib25`, `vm_tailloop1m` and `vm_mapfold200k` (`ns/op`, `B/op`) from `go test ./internal/runner -bench BenchmarkVM`.
  - `tools/vm_parity_diff.sh` banks each `examples/runnable` file's outcome under `--bytecode` and `--strict-bytecode`, and fails on any change.
  - `scripts/verify_bytecode_parity.go` gains `--strict` and `--bin`.
