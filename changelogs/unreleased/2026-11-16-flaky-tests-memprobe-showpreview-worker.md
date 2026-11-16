### Fixed — three flaky CI tests now measure what they protect (2026-11-16)

Test-only. No production code changed.

- **Debug.log retention.** `TestMemprobeDebugLogDoesNotAccumulate` (cmd/ailang) compared the
  child's peak RSS against a 30 MB bound. macOS CI read +32 to +47 MB for a correct binary. It is
  replaced by `TestDebugLogDoesNotAccumulateDuringRun` (internal/runner), which runs
  `runner.Run` in-process and uses two instruments.
  - Output order: with stdout and stderr on one file, every passing line logged before marker k
    must already be in the file, at every marker, and the output must interleave.
  - Retention: the test wraps the registered `Debug.log` operation to capture the contexts the
    program actually runs on. These are the cloned `Fresh()` contexts, not the runner's own.
    Each must end the run holding no entries.

  The pre-F6 shape (no sink, one shared accumulator) fails both instruments. A context that
  retains in the clone only, while still streaming every line, fails the retention instrument.
  The new `TestClonedContextWithSinkRetainsNothing` (internal/effects) pins the same contract at
  the unit level. Rows: a single run at `--log-level error` and at `debug`, and each item of a
  batch. Strict bytecode is not covered, because the VM does not wire `Debug.log` yet. The
  `debuglog.ail` memprobe fixture is removed.
- **showPreview allocation.** `TestShowPreviewDoesNotMaterialise` asserted fewer than 4,096 bytes
  per preview. Every preview costs about 4,040 bytes, and `testing.Benchmark` charges every
  allocation the process makes in its window to a `b.N` of only a few dozen. Windows CI read
  4,288. The test now makes two checks:
  - allocation does not scale with the value: a 1M-element preview against a 10k-element one,
    ratio at most 1.5 (measured 1.00);
  - each preview stays under 16 KiB, so a constant-size over-render is also caught.

  A materialising preview reads about 200x. Rendering a fixed 16k-character prefix reads
  70 KB per preview.
- **CLI-boundary budgets.** `TestWorkerCLIProcessExitLifecycle` ran `go build` and about 34
  subtests under one shared 120 s context. On a slow runner its last subtest
  (`original-select-handler-false`) failed after 0.00 s, and the failure named a subtest that had
  done nothing wrong. Three sibling tests (`TestWorkerLifecycleExistingFixtures`,
  `TestTerminalCLIPTYLifecycle`, `TestWorkerTerminalPTYLifecycle`) also built their own binary,
  and one of them shared the same context. The runner package now:
  - builds `ailang` once per test process (`ailangCLIBinary`, `sync.Once`), outside every
    subtest budget, instead of four times;
  - runs each probe under its own deadline (`runCLIProbe`). An overrun fails the subtest that
    overran and names it, its budget and how long it ran. A probe cut short by the package
    `-timeout` is reported as that, not as its own overrun;
  - names the build dir by pid. Each first build removes the dirs left by processes that died
    without running TestMain's cleanup.
