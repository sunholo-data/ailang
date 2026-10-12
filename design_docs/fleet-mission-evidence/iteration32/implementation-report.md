# Iteration 32 — Implementation report: M-QUORUM-ZERO-SIGNAL

**Sprint**: M-QUORUM-ZERO-SIGNAL (M1_GUARD_EXTERNAL_COUNT)
**Design doc**: [m-quorum-external-zero-signal-guard.md](../../planned/m-quorum-external-zero-signal-guard.md)
**Sprint plan**: [sprint-plan-quorum-zero-signal-guard.md](../../planned/sprint-plan-quorum-zero-signal-guard.md)
**Fleet ticket**: `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651)
**Branch**: `fleet/i32-quorum-zero-signal`
**Base HEAD at start**: `2d104f404` (tree clean)

## Intentional behavior change

Controller-only runs change from **proceed → blocked**.

Before: `synthesize` incremented the same presence count for a non-nil controller that it then tested for zero signal. A run where every external reviewer was absent (or none was supplied) therefore proceeded whenever the in-session controller passed. The zero-signal guard was vacuous on the invocation the mission-control skill mandates.

After: the count is `externalPresentCount` and counts present external reviewer outcomes only. The controller increment is removed. When no external reviewer is present, the guard fires and the run is `blocked` with the unchanged objection text:

`no reviewer produced a verdict (all absent) — refusing to proceed on zero signal`

Unchanged by design:
- Controller veto. A controller reject still blocks with `controller (in-session): <note>`, and the controller keeps its distinct artifact entry.
- N-1 degradation. One present passing external reviewer plus any named absences still proceeds.
- Tier-2 monotonicity. `SynthesizeWithTier2` still only assigns `SynthBlocked`, so a Tier-2 pass cannot rescue a zero-signal Tier-1 result.
- Accounting. Absence records, cost and token totals (including absent outcomes), and JSON fields are unchanged.
- The one external consumer, `cmd/ailang/design_quorum.go:89`, maps `SynthBlocked` to exit 3 and needed no change.

Resume path for a blocked controller-only run: obtain an admitted present external reviewer. The guard is not weakened.

## Commands and observed output

All commands were run in the foreground from the worktree root.

### Step 1 — pre-fix baseline

```
$ git status --short ; git rev-parse --short HEAD ; go test ./internal/mission/quorum -count=1 ; echo rc=$?
2d104f404
ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.363s
rc=0
```

Tree was clean at start; the baseline was green.

### Step 2 — new tests written

- `quorum_test.go`: `TestSynthesize_ExternalSignalMatrix` (table-driven, 11 subtests covering the full controller × signal matrix, including the empty-slice rows), plus helpers `costedAbsent`, `costedPresent` and the constant `zeroSignalObjection`. Absent outcomes carry non-zero cost and tokens so the accounting assertion is non-vacuous. Also added the `reflect` import.
- `seating_test.go`: `TestRunSeatedQuorum_AllAbsentControllerPassBlocks`, using the existing `poolRunner` and `Selection{Primary:a, Reserve:b, Benched:z}` with all absent and a controller pass. A helper `hasCall` is used for the fallback check. It was named `hasCall` because a `contains` helper already exists in `call_test.go`; the first draft collided and `go vet` caught it.
- `tier2_test.go`: `TestSynthesizeWithTier2_PassCannotRescueZeroSignalTier1` (all-absent Tier-1, controller pass, present Tier-2 pass, called directly through `SynthesizeWithTier2`) and the positive control `TestSynthesizeWithTier2_OnePresentPassStillProceeds`. Also added the `reflect` import.

```
$ go vet ./internal/mission/quorum; echo rc=$?   # after writing tests, before fix
rc=0
$ gofmt -l internal/mission/quorum; echo done
done
```

### Step 3 — pre-fix RED run (new tests, unfixed source)

```
$ go test ./internal/mission/quorum -count=1 -run 'TestSynthesize_ExternalSignalMatrix|TestRunSeatedQuorum_AllAbsentControllerPassBlocks|TestSynthesizeWithTier2_PassCannotRescueZeroSignalTier1|TestSynthesizeWithTier2_OnePresentPassStillProceeds' -v
    --- PASS: TestSynthesize_ExternalSignalMatrix/two_absent,_controller_nil
    --- FAIL: TestSynthesize_ExternalSignalMatrix/two_absent,_controller_pass
        verdict = "proceed", want "blocked"
        blocking objections = [], want ["no reviewer produced a verdict (all absent) — refusing to proceed on zero signal"]
    --- FAIL: TestSynthesize_ExternalSignalMatrix/two_absent,_controller_reject
        blocking objections = ["controller (in-session): controller veto"], want [controller veto, zero-signal]
    --- PASS: TestSynthesize_ExternalSignalMatrix/empty_slice,_controller_nil
    --- FAIL: TestSynthesize_ExternalSignalMatrix/empty_slice,_controller_pass
        verdict = "proceed", want "blocked"
    --- FAIL: TestSynthesize_ExternalSignalMatrix/empty_slice,_controller_reject
        blocking objections = ["controller (in-session): controller veto"], want [controller veto, zero-signal]
    --- PASS: TestSynthesize_ExternalSignalMatrix/one_pass,_one_absent,_controller_nil
    --- PASS: TestSynthesize_ExternalSignalMatrix/one_pass,_one_absent,_controller_pass
    --- PASS: TestSynthesize_ExternalSignalMatrix/one_pass,_one_absent,_controller_reject
    --- PASS: TestSynthesize_ExternalSignalMatrix/one_reject,_one_absent,_controller_nil
    --- PASS: TestSynthesize_ExternalSignalMatrix/one_reject,_one_absent,_controller_pass
--- FAIL: TestSynthesize_ExternalSignalMatrix
    seating_test.go:161: synthesis = proceed, want blocked: controller pass is not external signal
--- FAIL: TestRunSeatedQuorum_AllAbsentControllerPassBlocks
    tier2_test.go:158: verdict = "proceed", want blocked: a Tier-2 pass must not rescue zero Tier-1 signal
--- FAIL: TestSynthesizeWithTier2_PassCannotRescueZeroSignalTier1
--- PASS: TestSynthesizeWithTier2_OnePresentPassStillProceeds
FAIL
FAIL	github.com/sunholo-data/ailang/internal/mission/quorum	0.419s
FAIL
```

Result: the required red cases fail (controller-pass/all-absent, both empty-slice controller-pass and controller-reject rows, the seated regression, and the Tier-2 regression). The nil-controller and one-present rows pass before and after, as the plan predicted. The positive control also passes pre-fix, which is expected.

One row was not predicted by the plan's list: the two-absent/controller-reject row also fails pre-fix. Today the controller veto suppresses the zero-signal objection. The matrix expects both objections, and the fix supplies the second one by construction.

### Step 4 — production change

`internal/mission/quorum/quorum.go` only:
- Renamed the local count to `externalPresentCount`, incremented only in the reviewer loop.
- Removed the controller increment. The controller block keeps its reject veto and objection unchanged.
- The guard keeps its position and exact objection text, now testing `externalPresentCount == 0`.
- No early return was added. `synthesize` still has its single return at the function end.
- Updated the `RunQuorum` and `synthesize` contract comments to state the external minimum.
- The inline count was kept. `seating.go:presentCount` was not reused, because it operates on a built `*QuorumResult`, while `synthesize` runs before any result exists (plan registry decision).

### Step 5 — green runs after the fix

```
$ go test ./internal/mission/quorum -count=1 ; echo rc=$?
ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.462s
rc=0

$ go test -race ./internal/mission/quorum -count=1 ; echo rc=$?
ok  	github.com/sunholo-data/ailang/internal/mission/quorum	2.223s
rc=0

$ go vet ./internal/mission/quorum ; echo rc=$?
rc=0

$ gofmt -l internal/mission/quorum ; echo rc=$?
rc=0        (no output lines: nothing listed)
```

Post-fix matrix (all 11 subtests PASS), from the verbose run after the fix:

```
--- PASS: TestSynthesize_ExternalSignalMatrix
    --- PASS: .../two_absent,_controller_nil
    --- PASS: .../two_absent,_controller_pass
    --- PASS: .../two_absent,_controller_reject
    --- PASS: .../empty_slice,_controller_nil
    --- PASS: .../empty_slice,_controller_pass
    --- PASS: .../empty_slice,_controller_reject
    --- PASS: .../one_pass,_one_absent,_controller_nil
    --- PASS: .../one_pass,_one_absent,_controller_pass
    --- PASS: .../one_pass,_one_absent,_controller_reject
    --- PASS: .../one_reject,_one_absent,_controller_nil
    --- PASS: .../one_reject,_one_absent,_controller_pass
ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.318s
```

Bounding control: the pre-fix full-suite run (step 1, `ok … 0.363s`) and the post-fix full-suite run (`ok … 0.462s`) are both green. The full suite stays green across the change.

Changelog gate:

```
$ scripts/changelog_fold.sh --check ; echo rc=$?
✓ 1 changelog fragment(s) valid; changelogs/v0.32-current.md has '## [Unreleased]'
rc=0
```

The `## [Unreleased]` note refers to `changelogs/v0.32-current.md`, which predates this change and was not edited here.

### Step 6 — mutation drill MU1

Procedure: backed up the fixed `quorum.go` to `/tmp/quorum.go.fixed` (sha256 `ea9bb2953636a8b7d10af1aa6770a2dcbfd6c14ca8d13c765a50a45b85047c66`). Re-inserted `externalPresentCount++` as the first statement inside `if controller != nil {`, then ran the tests.

```
$ go test ./internal/mission/quorum -count=1 -run 'TestSynthesize_ExternalSignalMatrix' -v   (MU1 mutated)
--- FAIL: TestSynthesize_ExternalSignalMatrix
    --- PASS: .../two_absent,_controller_nil
    --- FAIL: .../two_absent,_controller_pass
    --- FAIL: .../two_absent,_controller_reject
    --- PASS: .../empty_slice,_controller_nil
    --- FAIL: .../empty_slice,_controller_pass
    --- FAIL: .../empty_slice,_controller_reject
    --- PASS: .../one_pass,_one_absent,_controller_nil
    --- PASS: .../one_pass,_one_absent,_controller_pass
    --- PASS: .../one_pass,_one_absent,_controller_reject
    --- PASS: .../one_reject,_one_absent,_controller_nil
    --- PASS: .../one_reject,_one_absent,_controller_pass
FAIL
FAIL	github.com/sunholo-data/ailang/internal/mission/quorum	0.420s

$ go test ./internal/mission/quorum -count=1   (MU1 mutated, full suite)
--- FAIL: TestSynthesize_ExternalSignalMatrix
--- FAIL: TestRunSeatedQuorum_AllAbsentControllerPassBlocks
--- FAIL: TestSynthesizeWithTier2_PassCannotRescueZeroSignalTier1
FAIL
FAIL	github.com/sunholo-data/ailang/internal/mission/quorum	0.401s

$ diff /tmp/quorum.go.fixed internal/mission/quorum/quorum.go   (MU1 mutated)
171a172
> 		externalPresentCount++
```

The named controller-pass/all-absent cases and the empty-slice/controller-pass case flip to FAIL, as required. The empty-slice/controller-reject and two-absent/controller-reject rows also fail, because the mutated increment suppresses the zero-signal objection. The seated and Tier-2 regressions also fail.

Restore, by copy from the backup:

```
$ cp /tmp/quorum.go.fixed internal/mission/quorum/quorum.go ; sha256sum internal/mission/quorum/quorum.go
ea9bb2953636a8b7d10af1aa6770a2dcbfd6c14ca8d13c765a50a45b85047c66  internal/mission/quorum/quorum.go
$ go test ./internal/mission/quorum -count=1 -run 'TestSynthesize_ExternalSignalMatrix' -v   (restored)
--- PASS: TestSynthesize_ExternalSignalMatrix   (all 11 subtests PASS)
ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.318s
$ go test ./internal/mission/quorum -count=1 ; echo rc=$?   (restored)
ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.361s
rc=0
$ diff /tmp/quorum.go.fixed internal/mission/quorum/quorum.go && echo identical
identical
```

Caveat: the MU1 invocation pipes `go test` through `grep`, so the exit status for the mutated runs was not captured as a number. The FAIL lines in the output are the evidence for that step.

### Scope check

```
$ git diff --stat
 internal/mission/quorum/quorum.go       | 31 ++++++-----
 internal/mission/quorum/quorum_test.go  | 95 +++++++++++++++++++++++++++++++++
 internal/mission/quorum/seating_test.go | 36 +++++++++++++
 internal/mission/quorum/tier2_test.go   | 40 ++++++++++++++
 4 files changed, 190 insertions(+), 12 deletions(-)
```

Plus the untracked changelog fragment `changelogs/unreleased/2026-10-12-quorum-external-zero-signal.md`. Nothing outside `internal/mission/quorum/**` and that changelog fragment was touched.

## Acceptance criteria status

- [x] Matrix passes; new controller-only tests failed before the fix (red run above) and pass after.
- [x] Pre-fix and post-fix full-suite green pair (bounding control).
- [x] Seated replacement works and all three absences are named (`a`, `b`, `z`).
- [x] Controller, reviewer and Tier-2 rejects keep their vetoes. The one-present pass still proceeds.
- [x] Existing serialization, accounting and escalation tests pass; `TestSynthesis_AbsentReviewersMarshalsAsEmptySlice` is in the green full suite.
- [x] `go test` and `go test -race` pass. `go vet` and `gofmt -l` are clean.
- [x] Contract comments and this report describe the controller-only behavior change.
- [x] Absent-pair/controller-pass row asserts model/reason pairs in input order, the controller entry, and totals equal to the supplied cost and token sums, including absent outcomes.
- [x] Two-absent/controller-reject row asserts both objections in order: controller note, then zero-signal.
- [x] `AbsentReviewers` is a non-nil empty slice for all-present inputs (existing serialization test, green).
- [x] Seated regression asserts blocked, three named absences, and fallback attempted.
- [x] Tier-2 regression asserts blocked with the zero-signal objection retained. The positive control proceeds.
- [x] MU1 mutation evidence recorded above, with restore confirmed by sha256 and by diff.

## Not verified here

- No live provider quorum was run. All runners are in-process stubs, consistent with the plan.
- The `-race` run and the full suite are green, but the wider repo (`make test`, `make ci`) was not run. Only the package named in the plan was in scope.
