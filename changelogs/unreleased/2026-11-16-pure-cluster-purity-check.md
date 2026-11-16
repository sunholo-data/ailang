### Fixed — `ailang test`: inline tests on an effectful function now say why they cannot run (2026-11-16)

The inline-test runner builds a "pure cluster" (the function under test plus every binding it
references) through `Executor.ExtractPureClusterForFunction`. That extractor never checked
purity. Its purity-checking twin, `ExtractPureCluster`, was called only from tests. So an
effectful function such as `func work(x: int) -> int ! {IO} tests [...]` went into a cluster
labelled pure. The test then failed at runtime with
`effect 'IO' requires capability, but none provided / Hint: Run with --caps IO`, and
`ailang test` has no `--caps` flag.

- The live path now goes through `ExtractPureCluster`, the one implementation of the purity
  gate. The harness takes that cluster directly instead of rebuilding a third copy of the
  dependency closure.
- An effectful function under test fails with
  `cannot test 'work': it has effects [IO] (inline tests run without capabilities; only pure functions can be tested inline)`.
  The runner reports this `*PurityError` as the test's failure and no longer falls back to
  the single-binding harness.
- The gate reads the effect row of the function under test, not the effect row of every
  dependency. If the effect system is sound, any effect a call can perform is already in that
  row. A pure function may reference an effectful one without calling it, as
  `examples/runnable/explicit_empty_callback_contracts.ail` does. The old closure-wide check
  rejected that example, which passes today.
- No pass/fail changes across the 64 tracked `.ail` files with inline tests (before/after
  sweep).
