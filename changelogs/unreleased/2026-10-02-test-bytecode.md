### Added — `ailang test --bytecode` and `--strict-bytecode` (#1487)

- Named-test bodies (`test "name" { ... }`) can now run on the bytecode VM. The semantics match
  `ailang run --bytecode`: a body runs on the VM when it compiles, and anything the VM cannot run falls
  back to the evaluator. That covers a body that does not lower, an evaluator-only entry and a VM runtime
  error. `--strict-bytecode` makes each of those a failure of that test, naming the flag. It implies
  `--bytecode`.
- The report is the same on both engines: pass/fail, error text, assert diagnostics, property
  counterexamples and seeds. Each fallback reruns the body on the evaluator, so a failing body reports
  the evaluator's error. Functions the compiler leaves evaluator-only are called through a bridge into the
  module-scoped test harness. `--max-recursion-depth` bounds the VM's frame stack as well.
- stderr gets one line per run with the counts of bodies that ran on the VM, fell back, or failed under
  strict mode, plus the first fallback reason. stdout, including `--json`, does not change.
- Speed, measured on a float sweep fixture (100 × 2,000-point `1/sqrt(1 - v²)` sweeps): the test went from
  7.37 s on the evaluator to 0.14 s on the VM. The whole run went from 8.7 s to 0.2 s wall-clock. A
  10^4-point sweep test went from 368 ms to 15 ms.
- **Not covered:** inline `tests [...]` tables and property harnesses still run on the evaluator under both
  flags. They are Core built in Go, with no surface source or type info to lower.
- Parity tests: `internal/testing/engine_parity_test.go` runs `testdata/engine_parity/*.ail` on both
  engines. Those fixtures include failing tests and asserts, a runtime error, a non-bool body, an ADT match,
  a bridged helper, inline tests and a seeded failing property. The end-to-end CLI test is
  `cmd/ailang/test_bytecode_test.go`.
