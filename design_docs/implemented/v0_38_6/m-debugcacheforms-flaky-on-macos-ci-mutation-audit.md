# Mutation-Audit Record — M-DEBUGCACHEFORMS-FLAKY-ON-MACOS-CI

**Sprint:** `v1_iter353_debugcacheforms` · **Milestone:** M2 · **Executor:** pi (deepseek-v4-flash)
**Worktree:** fenced sprint worktree, HEAD `d9cf6a706` (darwin/arm64) · **Restore tool:** `/usr/bin/shasum -a 256`

Audit of the six-mutant drill run against the fixed `capturePipelineStderrWith` helper. Every mutant
was copy-restored (`.bak`) and its restoration hash asserted equal to the pre-mutation hash of the
fixed form. All `go vet ./internal/pipeline/` invocations below returned `rc=0` immediately before
the corresponding test run (the mutant landed and builds; a non-compiling red is not a kill).

Pre-mutation (fixed-form) hash of `internal/pipeline/pipeline_module_phases_test.go`:

```
280368400e3d5c049413733d527678e09154ce27a28cf3c68c0ad195e28546ad  internal/pipeline/pipeline_module_phases_test.go
```

## Mutant rows

### MUT-1 (H1 reorder — the HEAD defect) — five consecutive runs

**Exact edit:** in the `defer` teardown, move `_ = r.Close() // 5. reader …` from after the bounded-wait
`select` to immediately after `_ = w.Close()` (above the `opts.release()` hook and the bounded wait,
matching the plan's M1 table row).

**Landing hash:** `a13d9007cff29919e88b0cc87966f36f99590b140aac55051b9826d664325797` (≠ pre-mutation) · **vet rc=0**

**Test runs** (`go test ./internal/pipeline/ -run TestCapturePipelineStderr_GatedReaderLosesNothing -count=1` ×5): `rc=1` on **all five** runs.
Load-bearing failure (iteration 0):

```
panic: capturePipelineStderr: copier failed: read |0: file already closed [recovered, repanicked]
```

**Restore:** copy-restore to fixed form; hash `280368400e…546ad` asserted equal to pre-mutation → OK.

### MUT-2a (H1 seam) — one run

**Exact edit:** delete the `if opts != nil && opts.wrap != nil { src = opts.wrap(r) }` branch so `src` stays `r` (the seam ignoring the wrapper).

**Landing hash:** `d4eb52f9363cf14c20128488ac61c5dda9151e976b3de01acb507fb43c6f51af` (≠) · **vet rc=0** · **test rc=1**

```
pipeline_stderr_capture_test.go:99: iteration 0: reader wrapper never invoked (reads=0); the seam is broken
```

**Restore:** OK (hash equal).

### MUT-2b (H1 seam) — one run

**Exact edit:** delete the `if opts != nil && opts.release != nil { opts.release() … }` hook body so the gate is never opened.

**Landing hash:** `bd118be2ad75318e639f957bc763c7fc7395b0d6ad87360ae315a54fe13bb046` (≠) · **vet rc=0** · **test rc=1**

```
panic: capturePipelineStderr: copier did not drain within 2s after the write end was closed (f passed os.Stderr to a still-running subprocess, or an injected reader never released)
```

**Restore:** OK (hash equal).

### MUT-3a (H1 teardown) — one run

**Exact edit:** remove the `defer` closure wrapper; teardown runs inline after `f()` instead of deferred.

**Landing hash:** `37edc2654ff3f2bf4efb60e449789eaef430974859a58632ac51076964f7e05d` (≠) · **vet rc=0** · **test rc=1**

```
pipeline_stderr_capture_test.go:118: os.Stderr was not restored after a panicking f()
```

**Restore:** OK (hash equal).

### MUT-3b (H1 teardown) — one run, declared non-hermetic

**Exact edit:** replace the bounded `select`/`time.After(timeout)` wait with a bare receive `copyErr := <-copied`
(deadline kept referenced as `_ = timeout` so the mutant builds).

**Landing hash:** `32b9a9cf78370fa18747eeac71aa26bdf7b3a31e038fe881639701085af43943` (≠) · **vet rc=0**

**Test** (`go test ./internal/pipeline/ -run TestCapturePipelineStderr_DrainTimeoutIsLoud -count=1 -timeout 60s`):
red via `go test`'s own timeout, `panic: test timed out after 1m0s`, goroutines parked on channel receive — **rc=1** (non-hermetic, red in ≤60 s, declared as such in the design doc and plan).

**Restore:** OK (hash equal).

### MUT-4 (H1 teardown, r2 astra) — one run, recorded as **V21**

**Exact edit:** in the copier goroutine, discard the copy error: `_, _ = io.Copy(&buf, src); copied <- nil`.

**Landing hash:** `4fc52002e41529368134f52434473e198e8c621f5e8744a39152db11638e706e` (≠) · **vet rc=0** · **test rc=1**

```
pipeline_stderr_capture_test.go:172: expected the helper to panic on a non-nil copy error
```

The sentinel read error is swallowed, a partial capture returns as success, the "panics with the
sentinel" assertion is red.

**Restore:** OK (hash equal).

## V21 block (for transcription into the design doc's Verification Log)

- **MUT-4 / `TestCapturePipelineStderr_CopyErrorIsLoud`:** green at the fixed form (`--- PASS … (0.00s)`);
  red under MUT-4 (`rc=1`, failure line `expected the helper to panic on a non-nil copy error`).
- **MUT-1 panic text observed** (exact, iteration 0): `capturePipelineStderr: copier failed: read |0: file already closed`

## Fixed-form AC2 output tail

```
--- PASS: TestCapturePipelineStderr_GatedReaderLosesNothing (0.00s)
--- PASS: TestCapturePipelineStderr_PanicRestoresStderr (0.00s)
--- PASS: TestCapturePipelineStderr_DrainTimeoutIsLoud (0.05s)
--- PASS: TestCapturePipelineStderr_CopyErrorIsLoud (0.00s)
PASS
ok  	github.com/sunholo-data/ailang/internal/pipeline	0.374s
```

## Summary

| Mutant | kind | landed (sha≠) | vet rc | test rc | killer test |
|--------|------|---------------|--------|---------|-------------|
| MUT-1 (×5) | reorder | ✓ | 0 | 1×5 | GatedReaderLosesNothing |
| MUT-2a | seam | ✓ | 0 | 1 | GatedReaderLosesNothing |
| MUT-2b | seam | ✓ | 0 | 1 | GatedReaderLosesNothing |
| MUT-3a | teardown | ✓ | 0 | 1 | PanicRestoresStderr |
| MUT-3b | teardown | ✓ | 0 | 1 (60 s timeout) | DrainTimeoutIsLoud |
| MUT-4 | teardown | ✓ | 0 | 1 | CopyErrorIsLoud |

All six mutants are red; every restoration hash equals the pre-mutation hash. No mutant survived;
nothing was tuned. All gates green at the fixed form (AC1/AC2/AC4/AC5 + whole-package `go test`).

## Round-1 judge corrections (sonnet, own worktree at 36a5cc8ee; each reproduced by the controller before recording)

- Red SETS measured with `-skip` (rule 3j: a single-test criterion is unsatisfiable for a broad-blast mutant):
  MUT-1 = {GatedReaderLosesNothing, TestPipelineModulePhases_DebugCacheFormsAndCounters} — the controller
  reproduced the second member: under MUT-1 the production test is red LOCALLY on the first run, because the
  read-on-closed-fd error that HEAD swallowed is now propagated. MUT-2a = {GatedReaderLosesNothing,
  DrainTimeoutIsLoud, CopyErrorIsLoud}. MUT-2b, MUT-3a, MUT-4 are sole-killers. MUT-3b is non-hermetic.
- `GatedReaderLosesNothing`'s post-loop check `os.Stderr == nil` could not fail (rc=0 with the helper's restore
  step skipped — mutant landed, vet rc=0). Fixed in r3: the test records the original stderr and asserts
  `os.Stderr == original`; against the same restore-skip mutant it is now rc=1
  (`os.Stderr was not restored to the original after the loop`), fixed form rc=0 with four `--- PASS:` lines.
- MUT-3b exits rc=1 (not the plan's rc=2); a regressed bounded wait would hang CI's un-scoped `internal/pipeline`
  run to go test's default 10-minute timeout — loud, not silent, and declared.

