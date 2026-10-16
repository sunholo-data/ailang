# M-VM-LIST-TAIL-SHARE residual execution report

Approved handoff: task-caba1356, source #1740, D-TRIAGE-14.
Execution branch: coordinator/task-1cee992e. Execution mode: sequential.
Phase 1 (#1827/v0.54.10) is protected and unchanged. D-19 stays parked.

## Acceptance evidence

- M1 / AC-R1: strict-alone regression covers a pure VM entry and the evaluator-only
  Clock entry, plus successful evaluator control. Existing paired and plain tests
  cover their unchanged behavior. Routing uses the existing run configuration,
  shared by ordinary and batch runs; `ailang test` already ORs its flags.
- M2 / AC-R2: equivalent range inputs; closed-form oracles checked on every
  repetition and boundary 0/1/2 on both engines. Fake wrong-output and exit-23
  executables were both rejected; exit-23 stderr was preserved.
- M3 / AC-R3: adjacent retention row and canonical entry explain backing storage
  and capacity caps. Existing D-19 construction rows remain byte-identical.
- AC-R4: no changes to Phase 1 builtins, sharing tests or std/list.ail.

## AILANG syntax and showcase checklist

AILANG prompt version loaded: v0.16.7, before writing any AILANG.
Walk has a pure countGt helper; all benchmark entries use
`main(n: int) -> () ! {IO}`. Contracts and inline tests: skip, because the runner
checks exact closed-form outputs at 0/1/2 and every timed size/repetition.
CLI fixtures: pure main and explicit Clock effect; contracts and inline tests
skip because Go tests assert routing, diagnostics and exit status.

## Measurement and validation

Linux amd64, Go 1.26.9. Both binaries were built from this checkout at
078ba98b4fc69387728b0c031164b694aad0b6e4; the final binary includes the two CLI edits.
Baseline build: 15:30:00 UTC. Final initial build: 15:38:29 UTC; verify-examples
rebuilt the same sources at 15:40:04 UTC during the benchmark. Build metadata
changed; measured runtime sources did not. Validation ran concurrently, so these
are local wall observations, not an isolated microbenchmark or CLI-fix speedup.

At 160k, final walk VM/interpreter = 0.0824 (12.14x faster): no residual walk
slowdown. Construction at 40k remains slower on VM (2.0619x interpreter).
Historical pre-fix V6/V28 in the design doc remain separate, from different
binaries/machines; this sprint makes no new Phase 1 before/after speedup claim.

### Fresh baseline before CLI implementation

```text
AILANG 078ba98b
Commit: 078ba98
Full:   078ba98b4fc69387728b0c031164b694aad0b6e4
Built:  2026-10-16_15:30:00

The AI-First Programming Language
Copyright (c) 2025-2026

sha256 a38a20c79779052a55dd92dde41187c9322c46fefd49c751655cbd2afc832643
40000 interpreter [1.1590789610000911, 1.1929560089999995, 1.4006856030000563] min 1.1590789610000911
40000 vm [0.29406524599994555, 0.1997686170000179, 0.20996627600004558] min 0.1997686170000179
80000 interpreter [2.629945152000005, 2.7649894859999904, 3.103524069000059] min 2.629945152000005
80000 vm [0.3942132900000388, 0.3288466149999749, 0.3832935909999833] min 0.3288466149999749
160000 interpreter [5.901319201999968, 5.897910280000019, 6.199063531999968] min 5.897910280000019
160000 vm [0.4170735390000573, 0.5182514439999295, 0.379691868000009] min 0.379691868000009
```

### Checked final min-of-three run

```text
binary: /workspace/task-1cee992e/bin/ailang
AILANG 078ba98b-dirty
Commit: 078ba98
Full:   078ba98b4fc69387728b0c031164b694aad0b6e4
Built:  2026-10-16_15:38:29

The AI-First Programming Language
Copyright (c) 2025-2026
runs: 3; monotonic wall seconds include startup/compilation/input construction
walk n=40000 interpreter: min=1.582403 samples=[1.582403285000055, 1.9852253330000167, 1.9166514339999594]
walk n=40000 vm: min=0.220888 samples=[0.2844353379999802, 0.22088778499994532, 0.4016224860000648]
VM/interpreter=0.1396
walk n=80000 interpreter: min=2.917953 samples=[3.3842559240000583, 3.203188993000026, 2.9179525230000536]
walk n=80000 vm: min=0.270558 samples=[0.2705575969999927, 0.30163361300003544, 0.32187366900006964]
VM/interpreter=0.0927
walk n=160000 interpreter: min=4.922467 samples=[5.006682533000003, 6.379944272999978, 4.922466922000012]
walk n=160000 vm: min=0.405566 samples=[0.40556605999995554, 0.4875578779999614, 0.4755776380000043]
VM/interpreter=0.0824
foldl n=40000 interpreter: min=1.188424 samples=[1.2826629040000626, 1.2048513590000312, 1.1884238860000096]
foldl n=40000 vm: min=0.113886 samples=[0.1234214339999653, 0.180080650999912, 0.11388585499992132]
VM/interpreter=0.0958
foldl n=80000 interpreter: min=1.538203 samples=[2.1783208239999112, 1.7995542169999226, 1.5382028280000668]
foldl n=80000 vm: min=0.112600 samples=[0.16695636899999045, 0.12206189300002279, 0.11260043400000086]
VM/interpreter=0.0732
foldl n=160000 interpreter: min=3.161836 samples=[3.1618363579999595, 3.202525172000037, 5.125849401999972]
foldl n=160000 vm: min=0.220797 samples=[0.37579102500001227, 0.2207968150000852, 0.29755526899998586]
VM/interpreter=0.0698
build n=10000 interpreter: min=1.103379 samples=[1.1033791439999732, 1.6115143199999693, 1.2955269249999901]
build n=10000 vm: min=1.772965 samples=[1.772965003999957, 2.0153222379999534, 1.9100020600000107]
VM/interpreter=1.6069
build n=20000 interpreter: min=4.275709 samples=[4.399627054000007, 4.504113891999964, 4.275708940999948]
build n=20000 vm: min=6.096410 samples=[6.096410274999926, 6.4259769809999625, 6.373372277000044]
VM/interpreter=1.4258
build n=40000 interpreter: min=8.733037 samples=[19.696934764999924, 8.733036513000002, 17.676316702999998]
build n=40000 vm: min=18.006722 samples=[18.006721718999984, 20.190747459000022, 24.201047736999953]
VM/interpreter=2.0619
```

### Final checks

Passed: focused CLI bytecode/strict tests; existing sharing and builtin-error
regressions; required three example stdout comparisons; benchmark module checks,
Bash syntax, all actual runner checks and fake-binary failure diagnostics;
file-size/changelog gates. New Go tests use filepath and do not assert rendered
paths, require external executables, or compare platform-dependent golden text.

`make verify-examples`: 243 passed, 5 failed, 9 skipped. All five failures are
unchanged programs with unexported main. Reproduced on a baseline binary built
using a Go overlay restoring BOTH modified production files from 078ba98b:

```text
examples/runnable/adt_list_fields.ail baseline exit 1 : main not exported
examples/runnable/if_else_chain.ail baseline exit 1 : main not exported
examples/runnable/poly_adt_result_type.ail baseline exit 1 : main not exported
examples/runnable/record_in_match.ail baseline exit 1 : main not exported
examples/runnable/record_width_subtyping.ail baseline exit 1 : main not exported
Baseline binary uses Go overlay restoring BOTH modified production files from 078ba98b; all other production sources unchanged.
```

Manifest import drift: 0; one stale missing lambda_expressions.ail entry is a
warning. The anticipated manifest-statistics mismatch did not occur here.

`make lint` retry passed: 0 issues (GOMEMLIMIT=768MiB, GOGC=25, GOMAXPROCS=2).
`make test-changed`: failed in cmd/ailang (372.235s): deep recursion
through-map/over-limit subprocesses exited -1 under concurrent validation;
policy deadline test could not find `ps`; supervisor-death test reported surviving
worker subprocesses. A baseline-source retry (GOFLAGS overlay inherited by the
CLI helper's build, restoring both production files from 078ba98b) reproduced
both policy failures; all deep-recursion cases passed in that lower-contention
baseline retry. Final changed-source deep-recursion retry passed (25.857s), matching the
baseline retry and resolving the resource-sensitive recursion failures.

Separate `make test-core` (GOMAXPROCS=2, GOFLAGS=-p=1, GOMEMLIMIT=1GiB):
effects nested-stop failure and pipeline signal-killed recur exactly as before
CLI edits. Other core packages passed, including evaluator, VM-related focused
checks, parser/types/loader/runtime/format. These unrelated failures are left
for CI/environment follow-up; this sprint does not claim an all-green gate.

## Friction

The toolchain was present at /usr/local/go/bin but absent from PATH; commands
explicitly include it. First builds compiled uncached Go dependencies.
Session startup's velocity display required missing `bc`; sprint validation
itself passed with three real milestones. No new workflow scripts were added.

The initial baseline core run (before CLI edits) failed
`TestRuntimeNestedStopUsesRemainingDeadline` (nested fixture survived bounded
stop) and killed the pipeline test process. Initial repository lint exited 137,
and the first baseline-overlay linker was killed under memory pressure. The
baseline-overlay rebuild succeeded on retry with GOGC=50. Final checks use lower
Go concurrency / memory limits; these infrastructure observations are recorded
separately from semantic test results.

### Baseline CLI gate reproduction

```text
--- FAIL: TestRunPolicy_DeadlineStopsExecutionWithoutProtocolGrace (0.00s)
    run_policy_deadline_test.go:58: policy fixture inventory: exec: "ps": executable file not found in $PATH
--- FAIL: TestRunPolicy_SupervisorDeathStopsWorker (2.64s)
    run_policy_lifetime_test.go:96: policy worker/subprocess survived lost supervisor: [16565 16575]
FAIL
FAIL	github.com/sunholo-data/ailang/cmd/ailang	29.085s
FAIL
```

## Completion and handoff

All three implementation milestones are complete. Independent evaluation is
requested with broader red gates explicitly disclosed. No merge or release was
performed. Active work took approximately 23 minutes (including cold builds and
validation); the plan budgeted about four active hours plus review buffer.
Production routing/help changed by two lines; 38 Go test lines and 105 benchmark/
README lines were added. Remaining additions are limitation/release documentation,
progress records and this evidence report. No representation or Phase 1 edits.

The initial overlapping validation jobs caused avoidable memory pressure. Run
large lint/link/benchmark jobs sequentially on this 8 GiB executor in future;
GOMEMLIMIT=768MiB allowed lint to finish with zero issues.

Evaluator handoff attempted via sprint-executor's prescribed `ailang messages send`.
The CLI refused: no agent serves `sprint-evaluator` and it is not human-triage.
`ailang coordinator agents` lists only `coordinator`; `messages inboxes` has none.
No forced send was made. Independent evaluation remains pending through the
coordinator's next stage; implementation is committed and ready for that stage.
