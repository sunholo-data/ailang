# Evaluation — M-DAEMON-TASK-EXEC-RUN-UNTESTED (v1 mission iteration 352, round 1)

- **Sprint:** `v1-iter352-daemon-task-exec` on `sprint/v1-iter352-daemon-task-exec-untested`, PR #1113
- **Commit under review:** `c501dcb62` (M3, the tip)
- **Evaluator:** independent sprint-evaluator, worktree `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter352-eval` (detached at `c501dcb62`)
- **Date:** 2026-09-08

## Verdict

**SCORE: 85/100 — PASS** (bar is 70)

The production change is genuinely minimal (an 8-line interface widening), it is exercised by a
test that fails to compile when M1 alone is reverted (true non-vacuity), and every mutation named
in the design doc that I re-ran independently does turn the gate red. But I found two things the
sprint's own evidence trail got wrong (a fabricated/incorrect sha256 and a misreported mutation
outcome, both in the M3 mutation-audit doc, both repeated in the PR description's "byte-identical
by sha256" claim), and two real surviving mutants on the exact call the sprint exists to cover,
neither found by the executor's own 9-mutation audit because it never varied `analyzed` or
`maxRetries`, only `opts`.

## Score breakdown

| Category | Points | Notes |
|---|---|---|
| Tests Pass | 20 / 20 | Build + full `internal/coordinator` suite green; no hard fail |
| Lint Clean | 10 / 10 | `gofmt -l`, `go vet`, `golangci-lint run` all clean on changed files |
| Acceptance Criteria | 30 / 30 | All 5 design-doc acceptance items independently verified |
| Code Quality | 10 / 15 | Two real surviving mutants on the covered call (see below) |
| Documentation | 6 / 15 | No CHANGELOG entry; audit doc has two evidence-integrity defects |
| Design Fidelity | 9 / 10 | Implementation matches design almost exactly; audit-doc defects cost 1 |
| Regression Surface Coverage | N/A | Not triggered — diff touches only `internal/coordinator/` |
| Performance Verification | N/A | Not a perf sprint |
| **Total** | **85 / 100** | **PASS** |

## The production diff, derived independently from `git diff a071ad981..c501dcb62 -- internal/`

```
internal/coordinator/daemon.go                     |   8 +-
internal/coordinator/daemon_tasks_exec_run_test.go | 100 +++++++++++++++
2 files changed, 107 insertions(+), 1 deletion(-)
```

That is the entire diff (confirmed with `git diff a071ad981..c501dcb62 --stat` including
non-`internal/` paths — the only other file in range is the mutation-audit doc itself). It adds:

- a package-level `taskExecutor` interface with one method, `ExecuteWithRetry`
- `Daemon.executor` retyped from `*TaskExecutor` to `taskExecutor`
- one new test file, `daemon_tasks_exec_run_test.go` (100 lines, one test function)

No other production file changed. This matches the design doc's own "Conflict Surface" claim of
5 total references to `d.executor` (1 declaration + 4 call/nil-check/assign sites), which I
independently reproduced:

```
$ grep -rn "\.executor\b" internal/coordinator/*.go | grep -v _test.go
internal/coordinator/daemon_tasks_exec_run.go:335:	result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)
internal/coordinator/daemon_tasks_exec.go:37:	if d.executor == nil {
internal/coordinator/daemon_tasks_init.go:274:		d.executor = taskExecutor
internal/coordinator/daemon.go:522:			if d.executor != nil {
```
(4 usages + the declaration itself = 5, matching V9/the M1 commit message.)

## Non-vacuity: M1-hunk-alone reversion (the test the task specifically demanded)

I reverted **only** `daemon.go` to its pre-`a7ec8a4b2` state (the exact hunk M1 introduced),
keeping M2's test file in place, using `git archive a7ec8a4b2^:internal/coordinator/daemon.go`
copied over the working file — no branch/remote git command used.

```
$ go build ./internal/coordinator/    # production code alone
rc=0   (the concrete *TaskExecutor path still compiles without the interface)

$ go test ./internal/coordinator/ -run TestExecuteTask_HandsDefaultBasedOptionsToExecutor -count=1 -v
internal/coordinator/daemon_tasks_exec_run_test.go:50:13: cannot use cap (variable of type
*capturingExecutor) as *TaskExecutor value in struct literal
FAIL	github.com/sunholo-data/ailang/internal/coordinator [build failed]

$ go test ./internal/coordinator/... -count=1
(same build failure — the WHOLE package's test binary fails to build)
```

**RED, confirmed, via M1's own hunk alone** — not via reverting the whole sprint. File restored
from `/tmp/daemon.go.orig` and diffed clean against the tree before continuing (`git diff` on
`daemon.go` empty afterward). This is the strongest possible form of red: a compile failure, not
merely a failing assertion.

## Mutation testing — independently re-run, plus two I added myself

All mutations applied to a backed-up copy of `daemon_tasks_exec_run.go`, run against
`TestExecuteTask_HandsDefaultBasedOptionsToExecutor -v`, then reverted and diffed clean against
`/tmp/daemon_tasks_exec_run.go.orig` after every single mutation.

| # | Mutation | My result | Matches audit doc? |
|---|---|---|---|
| M1 | `opts := DefaultExecuteOptions()` → `opts := &ExecuteOptions{}` | **RED** — `RetryBaseDelay = 0s, want 1s` | Yes, exact match |
| M2 | `opts.Timeout = agentConfig.GetEffectiveTimeout()` → `5*time.Minute` | **RED** — `Timeout = 5m0s, want 15m` | Yes, exact match |
| M3 | delete `opts.AgentConfig = agentConfig` | **RED** — `AgentConfig = 0x0, want 0x...` | Yes, exact match |
| M4 (plan's exact edit) | `d.executor.ExecuteWithRetry(...)` → `&ExecuteResult{Success:true}, error(nil)` | **RED, but a BUILD FAILURE** — `declared and not used: analyzed` | Yes — audit doc correctly flags this as a plan defect, independently reproduced |
| M4 (corrected, `_ = analyzed` added) | same + `_ = analyzed` | **RED** — `ExecuteWithRetry was never called (cap.opts == nil)` | Yes, exact match |
| S1 | `opts.IdleTimeout = agentConfig.GetEffectiveIdleTimeout()` → `3*time.Minute` | **RED** — `IdleTimeout = 3m0s, want 90s` | Yes, exact match |
| S2 | `opts.Workspace = workspacePath` → `""` | **RED** — `Workspace = "", want ".../ailang-test-workspace"` | Yes, exact match |
| S3 | `opts.ObservatoryContext = obsContext` → `nil` | **RED, but a BUILD FAILURE** — `declared and not used: obsContext` | **NO — audit doc claims a clean test-assertion failure (`ObservatoryContext = <nil>, want non-nil...`); actual result is a compile error.** See BLOCKING/NON-BLOCKING finding #2 below. |
| G2 (plan's stated gate, `go build`) | `daemon.go:88` `taskExecutor` → `*TaskExecutor` | `go build ./internal/coordinator/` → rc=0 (does not compile `_test.go`) | Yes — audit doc correctly flags this as a plan defect |
| G2 (correct gate, `go test`) | same field revert | **RED** — `cannot use cap ... as *TaskExecutor value` | Yes, exact match (this is the same experiment as my M1-hunk-alone check above) |
| **maxRetries (mine, not in plan or audit)** | `ExecuteWithRetry(taskCtx, analyzed, opts, 2)` → `..., 99)` | **SURVIVED** — target test PASS, full `internal/coordinator` suite PASS | Not attempted by the audit; `capturingExecutor` never even stores this argument |
| **AnalyzedTask.Content (mine, not in plan or audit)** | `Content: directive,` → `Content: "MUTATED-CONTENT",` | **SURVIVED** — target test PASS, full `internal/coordinator` suite PASS | Not attempted; `cap.task` is captured by the fake but **never asserted anywhere in the test** |

## Finding 1 (NON-BLOCKING, but should be fixed before this sets a precedent): two surviving mutants on the exact call the sprint exists to cover

`capturingExecutor.ExecuteWithRetry` captures both `opts` *and* `task` (`c.task = task`), but the
test only ever reads `cap.opts`. The `task` argument (the `*AnalyzedTask` built from lines
111–125, including the `Content: directive` field — the actual prompt handed to the executor) and
the `maxRetries` literal (`2`, not stored by the fake at all) are unexercised by any assertion in
the new test **and** by the rest of the `internal/coordinator` package (verified: `go test
./internal/coordinator/... -count=1` stays green with both mutations applied).

This matters because the design doc's problem statement frames the gap as "the `ExecuteOptions`
construction ... and the `ExecuteWithRetry` call at line 335" being uncovered, and the PR
description says the test "executes `executeTask` and asserts the `ExecuteOptions` actually handed
to `ExecuteWithRetry`" — true, narrowly, but the framing reads as broader coverage of "the call"
than what's actually verified. In fairness, Design §3 ("What the first test asserts") is itself
scoped only to `ExecuteOptions`, so no explicit promise was broken — but the `cap.task` field
existing in the fake with zero assertions on it is a strong signal the intent was broader than
what shipped. I'm filing this as a finding per instruction item 8 (under-statement is as much an
error as over-statement): the executor's own audit ran 9 mutations, all of them variations on
`opts`, and reported "every assertion is individually load-bearing" — true for the assertions that
exist, but the audit never asked whether there *should* be more assertions on the arguments the
fake already had in hand.

**Suggested fix (cheap):** add `if cap.task.Task.Content != task.Content { t.Fatalf(...) }` and
capture/assert `maxRetries` in the fake. Small, and it would have caught both of my mutations.

## Finding 2 (NON-BLOCKING but an evidence-integrity defect): the M3 mutation-audit doc misreports its own S3 result, and its baseline sha256 for the test file is wrong

The audit doc's central methodology claim, repeated in the PR body, is: *"Every mutation was
applied one at a time, the gate run, the result recorded, then reverted and verified
**byte-identical** to the baseline shasum."* I checked this two ways and both broke:

**(a) The claimed sha256 for `daemon_tasks_exec_run_test.go` does not match the actual file, at
any point in its history:**

```
$ shasum -a 256 internal/coordinator/daemon_tasks_exec_run_test.go
eb61961932c2ac725b3166542a481aae79894fd858cb4382d891e367259c6f58

$ git show c6ed3a1e1:internal/coordinator/daemon_tasks_exec_run_test.go | shasum -a 256   # M2 commit
eb61961932c2ac725b3166542a481aae79894fd858cb4382d891e367259c6f58

$ git show c501dcb62:internal/coordinator/daemon_tasks_exec_run_test.go | shasum -a 256   # HEAD (M3)
eb61961932c2ac725b3166542a481aae79894fd858cb4382d891e367259c6f58
```

All three agree with each other. **None of them match** the value recorded in the audit doc's
baseline table and repeated as the "after (reverted)" value on every mutation row and in the
"Final tree state" table: `6d6f247e3783e2c9516bddf397c2f743f97a9591c74c81430bfb9900c3d049d9`. The
test file itself never changed between M2 and M3 (`git diff c6ed3a1e1..c501dcb62 --
daemon_tasks_exec_run_test.go` is empty) — so this isn't drift, the recorded hash was simply wrong
from the moment it was written. The two other file hashes in the same table (`daemon.go`,
`daemon_tasks_exec_run.go`) **do** match what I measured, so this isn't a wholesale fabrication —
it's one wrong value sitting inside an otherwise-accurate table, which is arguably worse for
trust calibration than if the whole table were suspect.

**(b) The S3 row's claimed observed output is not what the mutation actually produces:**

The audit doc's S3 row claims: `--- FAIL` / `..._test.go:93: ObservatoryContext = <nil>, want
non-nil with TaskID "task-exec-1"`. I applied the identical edit (`opts.ObservatoryContext =
obsContext` → `opts.ObservatoryContext = nil`) and got:

```
# github.com/sunholo-data/ailang/internal/coordinator [...test]
internal/coordinator/daemon_tasks_exec_run.go:202:6: declared and not used: obsContext
FAIL	github.com/sunholo-data/ailang/internal/coordinator [build failed]
```

`obsContext` is declared at line 202 and otherwise only read at line 245 (the mutated line), so
nulling out the only read makes it write-only → Go's "declared and not used" — the *same class* of
defect the audit doc **correctly** caught for mutation M4 (the plan's edit leaving `analyzed`
unused). The audit doc applies that scrutiny to M4 but not to its own S3 row.

Neither of these two defects changes the bottom-line conclusion (the mutation still turns the gate
red either way, and I have no reason to think the M1/M2/M3/S1/S2/M4-corrected/G2 rows are
similarly wrong — I reproduced all of those exactly). But "verified byte-identical by sha256" is a
specific, falsifiable claim made in both the audit doc and the PR description, and it is false for
at least one file; and one of nine mutation rows reports a mechanism that isn't what happens. Per
instruction item 7, this costs points in Documentation even though the code is fine.

## Finding 3 (NON-BLOCKING, informational): PR #1113's "test" CI job is currently failing, but not because of this diff

```
$ gh pr checks 1113
test    fail    6m47s   .../job/101964086878
```

I fetched the job log (`gh api repos/sunholo-data/ailang/actions/jobs/101964086878/logs
--allow-escape-sequences`) rather than trusting the red X. The actual failing step is:

```
##[group]Run make check-file-sizes / check-boundaries / check-referenced-paths / check-git-exec
...
git exec site absent from baseline: internal/mission/iteration/artifacts.go (1)
git exec site absent from baseline: internal/mission/iteration/review_packet.go (1)
git exec AST total: 49; regex total: 49; fixture total: 2
make: *** [make/code-health.mk:185: check-git-exec] Error 1
##[error]Process completed with exit code 2.
```

This is `make check-git-exec` failing on a baseline drift in `internal/mission/iteration/*.go` —
files this sprint's diff never touches (its diff is limited to `internal/coordinator/`). The
`verify-examples-trace` output that appears later in the same log (2 of 217 examples red:
`runnable/ai_call.ail`, `runnable/claude_haiku_call.ail`) is a red herring — that step is
explicitly `|| true`-suppressed in `ci.yml` and both failing examples are AI-provider-call
examples that plausibly fail in CI for lack of network/API access, unrelated to this diff too.
This matches the project's own stated acceptance philosophy ("a gate already red at base measures
the repo, not this change") — I did not chase this further since it's out of this sprint's diff
surface, but the controller should know the PR is not actually mergeable-green right now for a
reason that has nothing to do with M1/M2/M3.

`Build windows-latest`, `test-windows`, and `docs-build` were still `pending` at last check —
**UNMEASURED**.

## Spot-checks on the design doc's Verification Log (item 7)

I independently re-ran or re-derived the following rows rather than trusting them; all matched
except where noted:

- V2–V11 (file length, function boundaries, caller count, `d.executor` reference sites, options
  construction lines 241–246, call at line 335): **all confirmed** via direct `grep -n`/`sed -n`
  against the actual file.
- V16 (`MockStore.GetCostByProvider` returns empty map), V19 (`DefaultExecuteOptions` values), V20
  (`GetEffectiveTimeout`/`GetEffectiveIdleTimeout` defaults): **confirmed** by reading the actual
  source.
- V26–V29 (AILANG_CONFIG precedence, `os.IsNotExist` → `DefaultBudgetsConfig`, no caching, default
  limits): **confirmed** by reading `agent_config.go` directly — the hermeticity mechanism the
  whole test rests on is real and works as described.
- V30/V31 (script-agent path never dereferences a nil `worktreeMgr`): **confirmed** — the deref at
  line 156 sits inside `} else if worktreeMgr != nil {` at line 149, only reached when
  `isScriptAgent` is false.
- V33/V34 (lines 368/640 guarded independently of `isScriptAgent`): **confirmed**, though the
  doc's claim column cites "line 372"/original-reviewer's line while the guard `if` itself is at
  368 — the doc's own observed-output cell already gives the corrected line, so this is not a
  fresh defect, just an artifact of quoting a prior reviewer's citation verbatim.
- The M1 commit message's claim that `TestExecuteTaskQueueSurvivesClosedStore` passes on the M1
  tree alone: **confirmed** via `git archive a7ec8a4b2 | tar -x` into a scratch directory (no
  branch/checkout operation) and running the test there directly — PASS.

## Hermeticity — adversarial check (item 5)

This machine (`mark@aitanalabs.com`'s) genuinely has a real `~/.ailang/config.yaml`
(`/Users/voightkampff/.ailang/config.yaml`, 6087 bytes, last modified Aug 26), so this was a live
adversarial test, not a hypothetical. I confirmed:

- `defaultConfigPath()` checks `AILANG_CONFIG` before `os.UserHomeDir()` (`agent_config.go:14-22`
  — read directly).
- `LoadBudgetsConfigFrom` on a nonexistent path returns `DefaultBudgetsConfig(), nil` via
  `os.IsNotExist` (`agent_config.go:214-217` — read directly).
- `LoadBudgetsConfig`/`LoadBudgetsConfigFrom` have no cache — `os.ReadFile` runs on every call, no
  `sync.Once`, no package-level cache var (`agent_config.go:205-213` — read directly), so
  `t.Setenv`'s per-test scoping is real, not accidentally shared across tests.
- Ran the target test at `-count=5 -race` and `-count=3 -p=8`: all PASS, no flakiness, no data
  races reported.

I did **not** independently re-run the audit doc's own two-run M5 experiment (fixed-spend store
wrapper + hostile external YAML) end-to-end — reproducing it requires temporarily rewriting the
test file's fixture and an out-of-band `AILANG_CONFIG`-pointed hostile config, which I judged lower
value than the mutations above given I'd already verified the underlying mechanism (`AILANG_CONFIG`
precedence, `os.IsNotExist` fallback, no caching) directly from source. **UNMEASURED**: I take the
audit doc's Run A/Run B GREEN/RED result on faith given the mechanism checks out, but I did not
reproduce those two specific `go test` invocations myself.

## Design fidelity

The implementation is a very close match to the design doc: same interface, same field-type
change, same fixture shape (script agent, `t.Setenv` first line, six assertions), same milestone
boundaries. No scope creep, no silently dropped requirements, no production-behavior change beyond
the field's static type (verified: `*TaskExecutor` is still the only production implementor,
assigned unchanged at `daemon_tasks_init.go:274`). The doc's own quorum log (3 reviewers, 2 rounds,
all objections answered with verification-log rows) is unusually rigorous for a 3-day, two-file
sprint. The 1-point deduction is solely for the audit-doc defects above, since M3 (the mutation
audit) is itself a committed design deliverable.

## Documentation completeness

- **CHANGELOG.md: not updated.** Checked `changelogs/v0.32-current.md` (the "Unreleased" section
  that other 2026-09-07/09-08 mission-related entries use) — no entry for this sprint anywhere.
  `.claude/rules/coding-standards.md` states "Every change requires: 1. CHANGELOG.md ... Semantic
  versioning, grouped by category" with no carve-out for internal-only/no-behavior-change sprints,
  and neither the design doc, sprint plan, nor sprint JSON mentions CHANGELOG at all. This is a
  real gap against a standing project rule, though I'd weight it as minor given the change is
  genuinely internal (a test-only diff plus a field-type widening with zero production behavior
  change) — a reasonable case exists that it doesn't warrant a public-facing entry, but that case
  was never made in the docs, it was just silently skipped.
- **Example files:** not applicable — no new language feature, no `.ail` surface.
- **Design doc status:** still "planned" — correct for a doc under first-round review, not a
  defect; this evaluator doesn't move it on a report, only on an actual PASS action taken by the
  controller.

## What I could NOT measure (explicit)

- **M5 two-run isolation experiment** (fixed-spend `MockStore` wrapper + hostile external YAML,
  Run A GREEN / Run B RED): mechanism verified from source (AILANG_CONFIG precedence, no caching,
  `os.IsNotExist` fallback, block-condition arithmetic in `daemon_tasks_budget.go`), but the two
  specific `go test` invocations were not re-run by me. **UNMEASURED.**
- **`Build windows-latest`, `test-windows`, `docs-build`** on PR #1113: still `pending` at last
  check (`gh pr checks 1113`). **UNMEASURED.**
- **Full repo `make test`** (build + all Go unit tests + pi-extension suite): not run end-to-end
  due to time budget; I instead ran `go build ./internal/coordinator/`, `go build ./cmd/ailang/`,
  and `go test ./internal/coordinator/... -count=1` directly, which is the actual surface this
  sprint touches. **UNMEASURED at whole-repo scope**, though the CI `Build ubuntu-latest` /
  `Build macos-latest` / `lint` / `Analyze Go` jobs on PR #1113 all report SUCCESS independently.
- **check-git-exec / check-file-sizes / check-boundaries** on this specific tree: not re-run
  locally (Finding 3 already establishes the one red CI gate is unrelated to this diff via job-log
  attribution, so I did not spend further budget chasing it locally).

## Bottom line

The core deliverable holds up under adversarial pressure: the seam is real, the test is genuinely
non-vacuous against M1 alone (a compile failure, the strongest possible red), the hermeticity
mechanism is real and independently verified from source, and 7 of the 9 audited mutations
reproduce exactly as claimed. But the sprint's own "prove everything" methodology has two concrete
holes in its own evidence (a wrong sha256 baseline and a misreported S3 mechanism, both stated as
verified in the PR description), and its mutation coverage stopped one field short of the
argument list it had already instrumented for (`cap.task` captured, never asserted) — which let two
real mutations on the exact covered call survive undetected by both the new test and the full
package suite. None of this is severe enough to fail the sprint, but all of it should be fixed
before this audit format is treated as a template for future sprints.
