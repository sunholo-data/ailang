# M-VM-LIST-TAIL-SHARE — residual sprint plan

Date: 2026-10-16. Status: planned, awaiting sprint execution approval.
Design: [m-vm-list-tail-share.md](m-vm-list-tail-share.md).
Source: #1740 (stapledons_godot); D-TRIAGE-14; design handoff task-585cec8d.
Target: next release after current dev v0.54.12; the v0.53.0 directory is historical.
Duration: 2 days including review/validation buffer, approximately 4 hours active work.
Estimated LOC: 230 (implementation, tests, benchmark and docs). Risk: low to medium.

## Scope and current state

Phase 1 shipped in #1827/v0.54.10. Do not reproduce rejected task-fc7451cd/PR #1750's M2.
Only AC-R1 (strict flag), AC-R2 (checked benchmark), AC-R3 (retention docs) remain.
AC-R4 is a no-redo guard; AC-R5 covers focused validation and the release note.
No changes to `internal/vm/builtins.go`, `internal/vm/builtins_list_poly.go`,
`internal/vm/list_tail_share_test.go`, or `std/list.ail`. Re-run existing sharing tests only.
`::`/`++`, OpCons, list representation and arena work remain parked on D-19 in
[m-list-cons-quadratic](../m-list-cons-quadratic.md).

## Planning measurement (performed before milestone decomposition)

Binary: `/usr/local/bin/ailang`, v0.54.12, full build identity
`3af754c4703d659e11ae3c2dac9b512af93e7797-dirty`, built 2026-10-16T08:32:09Z.
Checkout: `482a9a4e`, shallow snapshot; this is an available current-dev binary,
not a fresh build from this checkout. Go and make are absent here. Executor must
build its own binary and repeat measurements, retaining binary/commit provenance.
Do not compare absolute times across machines or claim these are checkout-built results.

Temporary module `walk` was type-checked with `ailang check` (temporary-path warning only).
The measured kernel is `countGt(xs, thr, acc) = match xs { [] => acc,
x :: r => countGt(r, thr, if x > thr then acc + 1 else acc) }`.
`main(n: int) -> () ! {IO}` prints `countGt(range(0, n), 0, 0)`.
Command: `ailang run --quiet [--bytecode] --caps IO --args-json N walk.ail`.
Python `time.perf_counter` around each subprocess; three runs, wall time includes startup,
compilation and range construction. Every exit code and exact integer stdout was checked
against N−1 on both engines. These are consumption benchmarks, not cons-construction runs.

| N | Engine | Three wall times (s) | Minimum (s) |
|---|---|---|---|
| 40,000 | interpreter | 0.489199, 0.536580, 0.490314 | 0.489199 |
| 40,000 | vm | 0.073602, 0.071322, 0.070488 | 0.070488 |
| 80,000 | interpreter | 0.901381, 0.936304, 0.918880 | 0.901381 |
| 80,000 | vm | 0.097989, 0.099402, 0.100385 | 0.097989 |
| 160,000 | interpreter | 1.782754, 1.806863, 1.796755 | 1.782754 |
| 160,000 | vm | 0.153712, 0.153622, 0.148465 | 0.148465 |

At 160k the VM takes 0.148465 s versus 1.782754 s interpreter: **12.01× faster**
(VM/interpreter = 0.0833). There is no remaining walk slowdown against the interpreter.
VM doubling ratios are 1.39 and 1.52, consistent with startup plus linear traversal;
these three wall measurements alone are not a proof of asymptotic complexity.
The original pre-fix V6/V28 tables remain historical references, not same-machine speedups.
D-19's construction work is a separate unresolved limitation, not a residual walk defect.

## Capacity and reuse audit

`analyze_velocity.sh 7` found one shallow snapshot commit and no usable recent LOC
metrics; its historical changelog matches are not velocity evidence. No empirical
LOC/day claim is possible. Budget 115 LOC/day, three bounded sessions of 60–90 minutes,
and approximately 25% contingency within two days. These are estimates, not observed throughput.
Existing CLI regression helpers and `bench/vm_hof_callbacks/run.sh` supply the patterns;
the latter discards output and only echoes errors, so reuse its min-of-N structure
while requiring fatal errors and output checks.

Registry decisions: M1 `none` (core CLI flag wiring), M2 `none` (repository engine
regression fixture using existing std/list), M3 `none` (repository limitation/release docs).
No package-like capability is being added, so registry search is inapplicable; no new
package or dependency is justified for any milestone.

AILANG prompt version loaded: v0.16.7 (`ailang prompt` loaded in full;
`ailang prompt --version-active`); CLI binary version is separately v0.54.12.

## Milestones

### M1: Strict-alone run selects the VM

Estimate: 70 LOC (5 implementation/help, 65 test); 60–75 minutes. Dependencies: none.
Files: `cmd/ailang/main_run_exec.go`, `cmd/ailang/main_run.go`, new
`cmd/ailang/run_strict_bytecode_test.go` using existing CLI helpers. Current source files
are 178 and 302 lines; no companion split is needed near the 800-line limit.
Example: a minimal pure VM-runnable entry and an evaluator-only entry created by the
CLI test (start with the existing Clock fixture in `run_bytecode_test.go`; confirm it
still triggers strict failure, otherwise select an actually unsupported entry).

- Set RunConfig.BytecodeMode to `bytecodeMode || strictBytecode`; retain StrictBytecode.
- Help explicitly says strict implies bytecode. Search related run entry paths to ensure
  they converge on this configuration; do not alter `ailang test`'s existing OR.
- Acceptance: strict-alone succeeds on the VM fixture and fails nonzero with
  `bytecode execution failed` on the evaluator-only fixture; evaluator mode succeeds
  on that fixture. Existing paired strict, plain bytecode and test strict behavior stays green.
- Assert routing through deterministic diagnostics/configuration or an unsupported fixture,
  never elapsed time. The successful strict-alone fixture guards the positive path.
- Validation: focused `go test ./cmd/ailang -run 'TestCLI_RunBytecode|Test.*Strict|Test.*Bytecode'`
  plus `make test-changed` after the final combined changes.
Risk: the historical unsupported fixture becomes VM-runnable; verify it before relying on it.

### M2: Commit equivalent-input benchmarks with fatal correctness checks

Estimate: 130 LOC (45 AILANG, 75 runner, 10 usage notes); 75–90 minutes. Dependencies: M1
(for strict-alone coverage in the final runner invocation).
Create `bench/vm_tail_walk/{walk.ail,foldl.ail,build.ail,run.sh,README.md}`.
Modules follow the repository's canonical module-path convention and are checked before runs.

- Walk and foldl both use `range(0, n)` and count `x > 0`, output max(n−1, 0).
  Build uses the same range then `foldl(\acc x. x :: acc, [], xs)` and prints length N;
  it is the explicitly quadratic D-19 contrast, not a construction optimization.
- Runner accepts binary path and positive run count, resolves binary path before chdir,
  reports binary version, sizes, engines and min-of-N wall seconds. Use a monotonic timer.
- Default walk/foldl N = 40k/80k/160k; bounded build sizes = 10k/20k/40k.
  Add boundary correctness runs N=0,1,2 without using tiny timings to infer scaling.
- Run both interpreter and explicit VM on identical inputs. For every repetition require
  successful exit, exact expected stdout and VM/interpreter output equality. Preserve stderr
  on failures, fail immediately, and never include a failed run in timing minima.
- Acceptance: an incorrect-output fake binary and a nonzero-exit fake binary both make
  the runner fail; actual binary passes. Run a strict-alone smoke after M1, not a timing assertion.
- Type-check all modules, run `bash -n`, execute min-of-3 on an executor-built binary.
  Record historical pre-fix V6/V28 separately from current measurements and state residual
  interpreter ratio. No flaky CI wall-time threshold. Mechanical copy regression protection
  remains the existing identity/cap/isolation Go tests, which are re-run without editing them.
Risk: construction contrast dominates runtime; keep the smaller size grid and configurable runs.

### M3: Document retention and finish focused validation

Estimate: 30 LOC docs; 60–75 minutes including final checks. Dependencies: M1, M2.
Files: `docs/LIMITATIONS.md`, `docs/docs/reference/limitations.md`,
`changelogs/unreleased/2026-10-16-vm-list-tail-share-residual.md` (opening `### `).
Example: reuse the M2 benchmark and explain a small retained suffix/prefix of a large source.

- Add adjacent retention row and canonical entry: a live small tail/take/drop view can
  retain the whole source backing allocation until all views are unreachable. Capacity
  caps prevent append writes into the source; they do not release the backing allocation.
- Acceptance: D-19's existing construction rows remain unchanged; text does not claim
  cons/append construction was fixed. Link the benchmark and distinguish landed Phase 1
  from the newly changed strict-alone CLI behavior in the release note.
- Re-run existing sharing tests: `go test ./internal/vm -run 'TestList(Tail|Take|Drop)'`.
  Diff evaluator/VM stdout for `examples/first_non_repeat.ail`,
  `examples/record_cons_pattern.ail`, and `examples/url_route_dispatch.ail`; preserve
  list head/tail/take/drop edge-case and builtin error-text regressions from the design.
  Run `make test-changed`, `make check-file-sizes`, `make check-changelog`,
  `make verify-examples`, and benchmark module checks. CI owns the full `make test` suite.
  If the known dev example-manifest statistics mismatch recurs, record exact baseline
  evidence and report it separately; do not expand this sprint into a manifest repair.
- Acceptance: all new targeted checks pass, pre-existing failures are evidenced, protected
  Phase 1 files have no diff, all AC-R1–R5 evidence is recorded in the sprint report.
Risk: docs drift; inspect the adjacent rows in both documents rather than relying on line numbers.

## AILANG module checklist

| Module / demo | Contracts | Effects | Inline tests |
|---|---|---|---|
| walk.ail | skip: timed benchmark; exact harness oracle N−1 | include: main(n: int) -> () ! {IO}, pure countGt | skip: runner checks 0/1/2 and all measured N |
| foldl.ail | skip: harness oracle N−1 checks equivalence | include: main(n: int) -> () ! {IO}, pure fold step | skip: same runner oracle |
| build.ail | skip: construction contrast with length oracle N | include: main(n: int) -> () ! {IO}, pure builder | skip: same runner oracle |
| M1 CLI fixtures | skip: routing tests, not API contracts | include: pure success entry, explicit Clock or supported fixture effect row | skip: Go CLI tests assert routing and diagnostics |

## Day-by-day execution and handoff

Day 1: M1 then M2, with a fresh binary and benchmark evidence. Day 2: M3, combined focused
validation, report and sprint evaluation; remaining time is contingency, not extra scope.
Three milestones total 230 LOC. No parallel agents are needed.

This planning task changes documents and sprint state only. Coordinator approval/merge of
the plan is the execution handoff; execution then uses sprint-executor and sprint-evaluator.
Do not start implementation or self-approve this newly produced plan during planning.
