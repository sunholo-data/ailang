### Fixed — three flaky CI tests now measure what they protect (2026-11-16)

Test-only. No production code changed.

- **Debug.log retention.** `TestMemprobeDebugLogDoesNotAccumulate` (cmd/ailang) compared the
  child's peak RSS against a 30 MB bound. macOS CI read +34 and +38 MB for a correct binary. It is
  replaced by `TestDebugLogDoesNotAccumulateDuringRun` (internal/runner), which counts the
  `Debug.log` lines a run holds. The test runs `runner.Run` in-process with stdout and stderr on
  one file. Before marker k is printed, every line logged ahead of it that passes `--log-level`
  must already be in the file. A run that streams holds 0 lines. The pre-F6 shape (no sink, one
  shared accumulator) holds all 40 and all 20,040 lines until the run ends. The test covers a
  single run at `--log-level error` and `debug`, and each item of a batch. Strict bytecode is not
  covered, because the VM does not wire `Debug.log` yet. The `debuglog.ail` memprobe fixture is
  removed.
- **showPreview allocation.** `TestShowPreviewDoesNotMaterialise` asserted fewer than 4,096 bytes
  per preview. Every preview costs about 4,040 bytes, and `testing.Benchmark` charges every
  allocation the process makes in its window to a `b.N` of only a few dozen. Windows CI read
  4,288. The test now checks that allocation does not scale with the value: a preview of 1M
  elements against one of 10k, ratio at most 1.5. The measured ratio is 1.00. A materialising
  preview reads about 200x.
- **CLI-boundary budgets.** `TestWorkerCLIProcessExitLifecycle` ran `go build` and about 34
  subtests under one shared 120 s context. On a slow runner its last subtest
  (`original-select-handler-false`) failed after 0.00 s, and the failure named a subtest that had
  done nothing wrong. Three sibling tests (`TestWorkerLifecycleExistingFixtures`,
  `TestTerminalCLIPTYLifecycle`, `TestWorkerTerminalPTYLifecycle`) also built their own binary,
  and one of them shared the same context. The runner package now builds `ailang` once per test
  process (`ailangCLIBinary`, `sync.Once`), outside every subtest budget. Each probe runs under
  its own deadline (`runCLIProbe`). An overrun fails the subtest that overran and names it, its
  budget and how long it ran. The package builds the CLI once instead of four times.
