# `ailang test` recompiles the whole module twice per inline test case — #1328's 24x slowdown is 82 full compiles of session.ail

- **Date**: 2026-10-03
- **Class**: bug (performance regression)
- **Recommend**: direct-fix
- **Searched**: `#1328`, `session.ail`, `ARTIFACT_TOO_LARGE`, `maxArtifactBlobBytes`, `ExtractFunctionBinding`, `ExtractPureClusterForFunction`, `per-test`, `re-elaborat` across design_docs/ (only hit: `docs-mission-iter17-issue-inventory.md`, an inventory row with no analysis); `git log --grep` for test-path perf / cache-cap commits since 2026-09-15 (none touch the inline-test path); `ailang-core-backlog.md` (no row)
- **Estimate**: ~40-80 lines in internal/testing/executor.go + runner.go, plus a regression test that counts pipeline runs

Issue: [#1328](https://github.com/sunholo-data/ailang/issues/1328) — motoko's `src/core/session.ail` (6,339 lines, 41 inline tests): `ailang test` 98 s on v0.33 → 2,336 s on v0.44.1, while `ailang check` stays ~30 s.

## Mechanism (verified at origin/dev 790169359)

`Runner.runTest` (internal/testing/runner.go:115 and :126) runs, **for every inline test case**:

1. `Executor.ExtractFunctionBinding` (internal/testing/executor.go:426) → `pipeline.Run` (executor.go:476) on the module file;
2. `Executor.ExtractPureClusterForFunction` (executor.go:624) → `pipeline.Run` (executor.go:646) on the same file.

Both pass `Filename: e.modulePath`, and the module pipeline loads source from disk — `src.Code` is read only by `pipeline_single.go` and a telemetry attribute (`grep -rn 'src\.Code' internal/pipeline/`). So the `stripNonPureFunctions` output that `ExtractFunctionBinding` builds is **dead for any file with a `module` line** (it only takes effect on the module-less temp-file branch, executor.go:450-464), and the two calls compile byte-identical input. The comment at executor.go:636-638 already notes this for the cluster call.

Each inline test therefore costs two full module compilations. Normally the on-disk compile cache absorbs the second and later ones. session.ail's type info exceeds the 16 MiB blob cap (`maxArtifactBlobBytes = 16 << 20`, internal/pipeline/cache_artifacts.go:27; motoko's logs show `CACHE_WRITE_FAILED ... coretypeinfo.gob: ARTIFACT_TOO_LARGE`), so the cache never stores it and every compile starts from scratch: 41 tests × 2 × ~28 s ≈ 2,300 s, which matches the reported 2,336 s.

**Measured** (synthetic 400-function module, origin/dev binary): with the cache on, 4 tests take 0.49 s and 16 tests take 0.99 s (`check` takes 0.37 s). With `AILANG_NO_CACHE=1`, which simulates the over-cap case, 4 tests take 2.70 s and 16 tests take 11.54 s (`check` takes 0.47 s). That is about 0.73 s per test, roughly two compiles each, and the cost grows linearly with the number of tests.

## Fix (no design decision needed)

Compile the module under test **once per `ailang test` file run**, inside the process, and reuse the `pipeline.Result` for every test case and property. The input is the same file each time, so memoizing on `(modulePath, cfg)` inside `Executor` is behaviour-preserving. The evaluators that consume the result are built fresh per test (`newHarnessEvaluator`), so sharing the Core program does not share mutable state. Delete the dead strip on the module path, or keep it only on the module-less branch where it is live, so readers do not assume per-function isolation that does not exist. Add a regression test that counts `pipeline.Run` calls for an N-test module (expect 1, not 2N).

Independently, consider a separate change: raise or remove the 16 MiB cap for `coretypeinfo.gob`, or warn once at INFO level when a module under test cannot be cached. After the memoization fix that cap costs one compile per run instead of 2N, so it no longer blocks #1328.

Not to be confused with `test-executor-module-env-miswiring.md` (#1261, now fixed by e69077211): that doc covered how the env is built. This one is about how many times the module is compiled.
