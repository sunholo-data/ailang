# Implementation completion: M-NAMED-TEST-EFFECTFUL-HELPER

Branch: `coordinator/task-708afb41`. Issue: #1640. Execution: sequential.

## Result

Named tests and forall properties retain all module-level functions. The exact plain
`export func inc(x: int) -> int { x + 1 }` regression passes on evaluator and bytecode.
Unused effectful helpers stay bound without executing. Effectful named-test/property
bodies fail with their original identity, source location, required effects, purity
contract, and capability-granted exported-entry workaround. Batch notices and engine
fallback summaries no longer expose generated signatures or temporary paths for
these failures. Invalid module helpers retain their compiler errors and receive
original declaration locations.

The legacy evaluator fallback now compiles a pure entry rather than a free
expression; the new regression showed that retaining helpers otherwise allowed an
effectful helper to reach runtime instead of enforcing the pure-body contract.
Strict VM compile failures count as user compile failures, preserving truthful D1
fallback classification. Binding extraction keeps its existing strip policy.

## Validation

- `go test -p 2 ./internal/testing -coverprofile=...`: PASS, 83.8% statement coverage.
  Isolated pre-change archive: PASS, 82.1%. Strip policy and mapper functions: 100%.
- Focused strip, batch, property, effect-diagnostic, position, and engine-parity
  tests: PASS. Regressions failed before their respective implementation changes.
- `go test -p 2 ./cmd/ailang`: attempted, FAIL after 440 seconds. There are
  33 failing top-level tests: 29 show direct CGO/sqlite failures; four additional
  failures concern recursion subprocesses, mission heartbeat cancellation,
  package-ratchet warning output, and process-supervisor descendant termination.
  These paths are unchanged by this sprint. Isolated pre-change tests reproduce
  `TestMissionHeartbeatRenewsThenCancelsWorker`,
  `TestWarnSilentRatchet_WarnWhenVersionBumpsWithoutMessage`, and
  `TestRunPolicy_TimeoutKillsDescendants` with the same failures. The recursion
  baseline passes; its isolated current-tree recheck also PASSes (27.2 seconds),
  showing the earlier failure was transient; concurrent load may have contributed. This full
  package result is not reported as green.
- `go test -p 2 ./cmd/ailang -run '^TestTestCommand'`: PASS (15.9 seconds), covering
  the CLI test command and bytecode flags. Fresh CLI regression verification also
  passes across all requested engine modes.
- `make test-core`: attempted; eight brain/sqlite tests in `internal/effects` fail
  because the binary uses `CGO_ENABLED=0` and sqlite requires CGO. Other core
  packages pass. No C compiler was installed, per dispatch constraints.
- `make lint`: PASS, zero issues. Initial five-minute runs timed out; final run used
  a ten-minute CLI timeout via a wrapper outside the repository, without changing
  repository lint configuration.
- `make check-boundaries`, `make check-file-sizes`, `make fmt-check`: PASS.
- Freshly built CLI: pure example checks and tests PASS on both engines; exported
  fixture verification PASS with `FS,IO` and fails without FS. Reporter repro fails
  truthfully under evaluator, bytecode, and strict bytecode while its pure sibling
  passes. Invalid helper reports its real function and original file:3:1 location.
- Example manifest `--ci`: PASS, 213 modules checked, zero drift; existing stale
  example warnings remain. New `examples/tests/` assets were verified directly,
  rather than repeating the unrelated `examples/runnable/` suite.
- Internal short-module fixtures used relaxed module checking for CLI validation;
  public examples use canonical module paths. Docs/prompt effect-claim sweep agrees
  with the documented pure-body contract.
- Full `make test` was not run. CI owns the full suite on a properly provisioned job.

## Milestones and handoff

- M1: `6132aef2`, preserve helpers and add plain-export/property regressions.
- M2: `88a4efe6`, honest diagnostics and engine/fallback regressions.
- M3: final documentation, examples, seeded fallback/location coverage, validation,
  sprint-state updates, and coordinator artifacts in the final local commit.

All work is local. No push or PR creation was attempted. The coordinator should use
[M-NAMED-TEST-EFFECTFUL-HELPER-pr.md](M-NAMED-TEST-EFFECTFUL-HELPER-pr.md) as the PR body;
it contains `Closes #1640`. Original design/plan paths were retained for existing
coordinator references and their statuses updated to reflect implementation.

Independent evaluation: PASS, 97/100, round 1. Report:
`.ailang/state/evaluations/eval_M-NAMED-TEST-EFFECTFUL-HELPER_round_1.json`.
The literal broad CLI/core green criterion remains unmet because of the recorded
CGO and preexisting failures; no introduced regression was found.

Created/modified path inventories accompany this report. Estimated sprint size:
420 LOC; actual implementation/docs/example additions are 473 LOC, with 42
deletions (metadata and design-status updates excluded), recorded in sprint JSON. Historical
velocity and full-session timing were unavailable; no synthetic timing is claimed.
