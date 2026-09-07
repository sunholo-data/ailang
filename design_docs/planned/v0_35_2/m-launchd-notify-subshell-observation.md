# M-LAUNCHD-NOTIFY-SUBSHELL-OBSERVATION: Restore Notification Test Observability

**Status**: Planned — mission iteration 344, single revision after quorum round 1
**Target**: v0.35.2
**Priority**: P0 (inherited dev-CI regression)
**Estimated**: 2 hours, including mutation validation and independent evaluation
**Dependencies**: Existing notification repair `63a0d2b32`; base `c308b2a0a84edb593609af5e7f4bb3b7bc402014`
**Lane**: Mission harness; test-only correction, with the production notification contract preserved
**Created**: 2026-09-07

## Problem Statement

At the exact base, `make test-launchd-drivers` passes all 54 pin-root checks and then fails
with **20 passing / 7 failing** notification checks. The actual fixture is
`tools/launchd/test_driver_notify.sh`; the reported name
`test_mission_control_notifications.sh` does not exist at this base.

Commit `63a0d2b32` retained direct sends, added canonical message-store environment settings,
captured their diagnostic output, and added a drain for previously spooled failures. It did
**not** replace direct notification with spool-only delivery. Its `_out=$(ailang ... 2>&1)`
executes the test's `ailang()` function in a subshell. That stub records calls by assigning to
`TRACE`, so the record disappears when the subshell exits. The unchanged GitHub stub still
records successfully. All seven red checks depend on the missing AILANG call or its title.

The product's diagnostic capture is legitimate. The test observation mechanism is wrong;
weakening call/title assertions or removing product output capture would hide the defect.
The broader pattern is incomplete standalone-driver labs: `9f267cf1f` repaired missing
`STATE_DIR` inputs in this same suite. Repair the common spy boundary and cover the newly
introduced retry/store/drain behavior in this existing CI fixture.

## Verification Log

Measured first-party in the isolated iteration-344 worktree at the base above. Tests use
stubbed channels; these measurements did not send production notifications.

| ID | Claim / command or read | Observed |
|---|---|---|
| V1 | `git status --short`; `cat std/VERSION`; `/bin/bash --version` | Clean before design creation; v0.35.1; Bash 3.2.57 on arm64 Darwin. |
| V2 | `make test-launchd-drivers` | Exit 2: pin-root 54/0; driver-notify 20/7; make stops before the remaining suites. |
| V3 | Read `make/test.mk:59-83` and all of `tools/launchd/test_driver_notify.sh` | Target invokes `test_driver_notify.sh`; 27 existing assertions. `test_mission_control_notifications.sh` read fails with ENOENT; existing fixture is the positive control. |
| V4 | Read `git diff 63a0d2b32^ 63a0d2b32 -- tools/launchd/mission-control.sh` | Only production file changed. Direct send changed to command substitution; canonical environment, error-tail capture, drain function and post-pin call were added. |
| V5 | `/bin/bash -c 'TRACE="parent"; ailang() { TRACE="sent"; }; result=$(ailang); printf "capture: %s\n" "$TRACE"; ailang; printf "direct: %s\n" "$TRACE"'` | `capture: parent`, then `direct: sent`; independent shell-mechanism reproduction. |
| V6 | Read suite `run()`, `run_drift()`, drift-j lab; `rg -n 'TRACE=|ailang\(\)' tools/launchd/test_*.sh` | These three labs in `test_driver_notify.sh` use mutable TRACE spies. The seven failures are pin AILANG/title, lane AILANG/title, drift-a, drift-c, drift-g. Healthy silence and GitHub checks pass. |
| V7 | Read `_mc_notify` at driver lines 167-203 | Three attempts, sleeps 5 then 10 seconds; output captured with stderr; final failure logged with last 300 bytes, newline-flattened; failed payload appended to mission spool; GitHub handled separately. Existing test executes real backoff. |
| V8 | Read `_mc_drain_notices` at lines 145-165 and call at 972 | Moves current spool to a draining file, delivers each entry with original timestamp prefix, re-appends failed rows, removes draining file, logs sent/kept counts; direct call follows pin-decision block. |
| V9 | `rg -n 'spool|drain' tools/launchd/test*.sh` | Zero matches; positive controls are `_mc_drain_notices` definition/call and spool references in production driver. Existing launchd test bodies contain no spool/drain assertions. |
| V10 | `git log -5 --oneline -- tools/launchd/test_driver_notify.sh`; read `git show 9f267cf1f` | Prior test-only repair addressed missing lab state, preserved per-arm isolation, and demonstrated mutation controls. Its removed drift-j change was dead code: avoid incidental drift-j refactoring here. |
| V11 | `ailang docs search --stream planned/implemented --neural --timeout 15s --limit 5 'launchd notification spool tests'` (separate invocations with actual stream names) | Partial searches timed out after 27/187 planned and 44/200 implemented embeddings. Best planned 0.45 (motoko discovery refusal plan); best implemented 0.43. No returned duplicate threshold reached. Targeted text search supplied relevant driver/workbench docs below; search is not claimed exhaustive. |
| V12 | Read numbered driver lines 153-154 and 183-184 | Both drain and direct-send sites set `AILANG_MESSAGES_STORE=gcp` and `AILANG_MESSAGES_PROJECT="${AILANG_MESSAGES_PROJECT:-ailang-multivac}"` on the command. Thus unset **or empty** project selects `ailang-multivac`; nonempty caller project is retained. Neither site assigns `AILANG_STORAGE`. Tests will observe these values inside the stub, not infer them from its exit code. |
| V13 | Read numbered fixture lines 52-55, 87-88, 125-130; `/bin/bash -c 'false; actual=$?; printf "%s" ""; printf "actual=%s fixture_after_printf=%s\n" "$actual" "$?"'` | `run()` sources the emit block, then `printf`, then expands `RC:$?`; the two “block still exits 0” assertions therefore inspect printf's status. Shell control prints `actual=1 fixture_after_printf=0`. `run_drift()` expands `DECISION_RC:$?` immediately after the decision block and does not have this masking defect. |
| V14 | In `/bin/bash`, `eval` the unchanged `_mc_notify` obtained with `awk '/^_mc_notify\(\) \{/,/^\}/' tools/launchd/mission-control.sh` and unchanged fixture stubs obtained with `sed -n '38,43p' tools/launchd/test_driver_notify.sh`; call notify with title `Mission v1: driver ran UNPINNED`, body `body`, normal fixture GH/repo/sender variables; then reset TRACE and call the **same** AILANG stub directly with the same send arguments | Real notify records only `GH:issue comment 635 --repo sunholo-data/ailang --body body`; direct control records `AILANG:messages send controlplane body --title Mission v1: driver ran UNPINNED --from mission-control`. Both use the actual source/stub, not a retyped notification implementation; neither invokes a real external command. Together with V2 and the call-site mapping below, this isolates lost observation in the real fixture. |
| V15 | Read complete driver functions 145-203; `rg -n '_mc_bounded|_mc_notify|_mc_drain_notices' tools/launchd/mission-control.sh`; read helper 496 onward and `git show 63a0d2b32^:tools/launchd/mission-control.sh` notify body | Direct AILANG send, drain AILANG send, and notify GitHub post have **no driver-level per-call timeout**. Notify/drain call sites are direct. Positive control: `_mc_bounded` exists and wraps other messaging calls at 1817/1820. Parent commit already has unbounded direct sends/posts; the new drain at base is also unbounded. Three retries bound the count, not each call's duration. This is a separate existing production defect, not an effect of the proposed test edits. No claim is made about internal CLI/network-library deadlines. |

### Exact causal mapping of the seven baseline failures

Line numbers below are at the recorded base. `T` means `tools/launchd/test_driver_notify.sh`;
`P` means `tools/launchd/mission-control.sh`. All rows use P:183-184, the captured direct send
inside `_mc_notify`; P:185 accepts the stub's successful exit. The only AILANG observation
in the relevant fixture stubs is assignment to `TRACE` (T:40-41 or T:71-72). Capture loses
that assignment as measured in V14; each row's failed predicate requires that lost record.

| Exact failing assertion / source | Lab and production emit call | Lost value required by assertion; surviving control in V2 |
|---|---|---|
| `fires on both channels (ailang)` — T:114, pin section | `run(pin_block)`; P:1578 | `AILANG:messages send controlplane`; adjacent GitHub post check passes |
| `titled as UNPINNED` — T:116, pin section | `run(pin_block)`; P:1578 | Exact lowercase `driver ran UNPINNED` in AILANG title; GH body has differently capitalized `Driver ran UNPINNED`, so does not satisfy this predicate |
| `lane fires on ailang` — T:139 | `run(lane_block)`; P:1549 | `AILANG:messages send controlplane`; lane GitHub and log checks pass |
| `lane keeps its own title` — T:141 | `run(lane_block)`; P:1549 | Exact lowercase `executor/planner lane degraded` in AILANG title; GH body's initial `Executor/planner` differs |
| `drift-a: first threshold notice reaches both channels with path/count` — T:148 | `run_drift(pinned,170,25,absent)`; P:1596 | Ordered AILANG record before GH; decision rc 0, state 170, GH/path/count survive |
| `drift-c: doubling notifies` — T:152 | `run_drift(pinned,340,25,170)`; P:1596 | AILANG call record; decision rc 0 and state 340 survive |
| `drift-g: STALE keeps original notice only` — T:160 | `run_drift(STALE,170,25,absent)`; P:1578 | Exact lowercase `driver ran UNPINNED` title; decision rc 0 and GH body survive |

V2 supplies the actual failed rows and surviving trace contents; V4 identifies the changed
capture site; V14 reproduces the causal boundary with the actual notify function and fixture
stub. This establishes a test-observation defect without claiming production delivery was
measured against live services.

## Goals and High-Impact Decisions

Primary goal: make the existing suite observe real notification calls across shell boundaries,
restore its original 27 checks, and prevent regressions in the repaired notification path.

| Decision | Reason | Owner | Freeze / cost |
|---|---|---|---|
| Preserve current production behavior | The red set is explained by the observation boundary, not failed sends | Agent designer, reviewed by evaluator | Design / low |
| Persist test observations in per-arm files | Works through command substitution and stdout/stderr redirection | Executor | Design / low |
| Extend the existing suite | Already wired into the Bash 3.2, Go-less CI target | Executor | Design / low |

- [x] Production changes are outside this fix unless new evidence demonstrates a separate defect; report and re-scope such evidence.
- [x] Keep all original behavior assertions and their meaning, including healthy silence and source-clone paths.
- [x] Preserve per-arm temporary state; all channel calls remain stubbed and sleep is observed without waiting.

## Solution Design

Modify **only `tools/launchd/test_driver_notify.sh`** for implementation (approximately
100–180 added/changed lines; executor may organize helpers to keep it smaller). Documentation
and sprint metadata accompany it. `mission-control.sh` and `make/test.mk` require no landed edit.

1. Introduce a file-backed call trace inside the suite's temporary lab. In the send-capable
   `run()` and `run_drift()` labs, make `ailang`, `gh`, and `log` append ordered observations
   to this file; replay it after the block. Preserve `AILANG:`, `GH:`, and `LOG:` records used
   by the existing checks. Keep stub command output separate from its trace, because production
   intentionally captures or discards command output. The drift-j path does not send at base;
   leave it alone unless sharing the helper eliminates duplication without weakening coverage.
2. Record messaging environment **inside** the AILANG stub, alongside command arguments.
   Verify direct and drained sends see `AILANG_MESSAGES_STORE=gcp`, default project
   `ailang-multivac` for both unset and empty input, and the caller's nonempty explicit project
   override when supplied (V12). Verify the
   caller's store/project values and `AILANG_STORAGE` are unchanged after each function.
3. Add a `sleep` stub that logs requested durations and returns immediately. For retry scenarios,
   maintain attempt state in a file as well: a shell counter has the same subshell defect.
   Cover success first try, failure then success, and all three failures. On permanent failure,
   assert the error-tail content, exactly one spool row, flattened multiline body, and continued
   GitHub attempt. Capture the block's actual status immediately after sourcing; the current
   `RC:$?` after `printf` measures `printf`, not the block.
4. Extract `_mc_drain_notices` from the real driver using the established guarded awk pattern.
   Cover absent/empty spool, all-success delivery, mixed success/failure, all-failed retention,
   and recovery on the next drain. Assert original timestamps, titles, bodies, sender, exact
   retained row set, sent/kept counts, and cleanup of the draining artifact. A second successful
   drain must not resend delivered rows. Verify v1 and one non-v1 namespace do not collide.
   Keep GitHub out of drain assertions except a negative assertion: drains are controlplane-only.
5. Assert the live driver has exactly one top-level `_mc_drain_notices` invocation after the
   pin-decision end marker. This narrowly scoped wiring check complements behavioral function
   tests and detects deletion of the preflight call without starting a live mission.

Use `mktemp -d` and cleanup traps for suite-owned artifacts. Avoid parallel arms sharing trace
or retry state. A trace-write failure must produce a test failure, not an apparent successful
stub. Product functions and emit blocks must remain extracted from source, never copied into tests.

## Conflict Surface

| Surface | Existing occupant / behavior to preserve | Verification |
|---|---|---|
| Captured AILANG stdout/stderr | `_out` contains provider output; caller cannot rely on it being printed | Separate file spy and diagnostic-tail assertions |
| Shell state and episode markers | Fresh pin/lane arms avoid accidental dedupe; drift state persists only within its arm | All original checks; per-arm trace/state paths |
| Ordered trace output | drift-a expects decision, AILANG then GitHub; titles appear in AILANG arguments | Replay file after decision/state output, preserving channel order |
| Retry state | Each command substitution forks; first-attempt failure must advance across forks | File counter, exact attempt/backoff checks |
| Message environment | Per-command canonical store; project override; separate `AILANG_STORAGE` | Inspect inside stub and after calls |
| Spool filesystem | Mission names select distinct TSV files; failures survive next drain | Exact content/count and cross-namespace controls |
| Go-less Bash 3.2 CI | Target executes shell suites then syntax-checks drivers | `/bin/bash`, no Go or new dependencies |

Fixtures that must still work: pin notice healthy/degraded/failed-channel/unset-issue,
lane healthy/degraded, and drift-a through drift-j in `test_driver_notify.sh`; all 54
`test_pin_root.sh` checks; every remaining suite in `make test-launchd-drivers`.
Intentional changes are limited to test instrumentation and added assertions. No language
syntax, runtime behavior, notification payload, retry timing, or mission policy is changed.

## Examples

Before: `_out=$(ailang ...)` sends successfully, but the stub's `TRACE` assignment vanishes;
the pin arm reports “fires on both channels (ailang)” as failed.

After: the same production call appends an AILANG record to an isolated trace file; the pin
arm observes its recipient/title, while the captured stdout remains available for diagnostics.
During a simulated outage the trace records three sends and backoffs `5`, `10`; one spool
row remains and a later successful drain removes it after asserting timestamped delivery.

## Acceptance Criteria and Test Plan

- [ ] Original 27 assertions pass with the unchanged production driver; record exact new total.
- [ ] New direct-send/store/retry/error/spool/drain/wiring cases above pass; every spy is local.
- [ ] Failed send and failed GitHub delivery remain non-aborting, measured from the actual block status.
- [ ] All tests in `make test-launchd-drivers` pass, including Bash 3.2 syntax checks; record duration and exit code.
- [ ] Mutation controls below produce named behavioral failures; clean copies parse and all applied mutations are proven absent afterward.
- [ ] Documentation updated with observed counts, results, limitations, and scope; no implementation claim based solely on a future test plan.

Run the focused suite first, then these small independent mutants against temporary copies
of the driver/fixture tree (or a fixture-supported source override). Keep the production
worktree untouched and prove each mutation landed by diff/hash and `bash -n`:

| Mutant | Required failure |
|---|---|
| Replace actual AILANG send command with a successful no-op, retaining function/block extraction markers | Named send/call-count/title assertion fails; extraction failure is insufficient |
| Change canonical store assignment to `local` in direct and/or drain send | Corresponding inside-stub environment assertion fails |
| Suppress re-appending a failed drain row | Retained-row/recovery assertion fails |
| Delete only top-level drain invocation | Wiring assertion fails, with function extraction still valid |

Record exact outcomes, not predicted counts. Return to the unmutated suite and rerun once;
verify production driver hash equals its initial value. Each focused run gets a bounded
30-second budget after sleep stubbing; full target gets a bounded 10-minute budget with
progress reporting, because pin-root/probe suites do real work. An exhausted budget is an
explicit incomplete check. These budgets bound **test executions only**; they do not add a
production timeout or establish a production duration guarantee (V15). Tests must not contact
Firestore, GitHub, model providers, or launchd.

## Timeline, Risks, and Deferred Decisions

One coherent milestone: roughly 30 minutes instrumentation, 30 minutes new cases, 30 minutes
mutations, and 30 minutes full verification/evaluation buffer. Executor may choose helper
names and whether a test-only source override is cleaner than a temporary tree for mutants.

Main risks are a spy still relying on subshell-local counters, dropped trace ordering, and
an accidentally vacuous mutant. File state, the original ordered checks, and named red sets
address them. Retry timing is asserted rather than slept. Crash recovery for an interrupted
drain, concurrent spool writers, TSV-format redesign, bounded production network calls, and
episode-gating expansion are outside this regression and are not claimed solved here.

### Separate production follow-up for the controller's queue

**Candidate: bound mission notification network calls.** V15 establishes that existing direct
send/post calls and the base's spool drain have no driver-level per-call deadline. A stuck
command can therefore prevent its next retry or next spool row from being reached; finite
retry count is insufficient. This is a production defect acknowledged by this design, not
an accepted duration guarantee. The existing `_mc_bounded` helper is relevant prior art,
but choosing timeout values, handling process termination, and preserving failed spool records
need their own design and hanging-command tests.

Disposition under the controller's Gate-2 rule 3f/c: **queue that follow-up separately while
keeping this CI-red repair test-only**. The baseline seven failures arise with immediate,
successful local stubs (V14), so production timeout changes are unnecessary to resolve them.
The controller owns adding this candidate to mission queue/log; this designer is authorized
to change only this document and has not claimed an external queue write. A5 below receives
no improvement credit for production boundedness. This follow-up is not an acceptance
criterion requiring production edits in the current sprint.

## Axiom Compliance

Harness-scoped scoring; no language-support claim is made.

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Stable observations independent of subshell variable lifetime |
| A2 Replayability | 0 | Existing production replay contract preserved |
| A3 Effect Legibility | +1 | Tests distinguish captured output from observable channel calls |
| A4 Explicit Authority | 0 | All external channels remain stubbed |
| A5 Bounded Verification | 0 | Test runs have explicit budgets, but production network calls remain unbounded at driver level (V15); no production boundedness improvement is claimed |
| A6 Safe Concurrency | 0 | Per-arm isolation retained; no production concurrency change |
| A7 Machines First | +1 | CI failures identify broken behavior rather than hidden spy state |
| A8 Minimal Syntax | 0 | No language change |
| A9 Cost Visibility | 0 | No billing change |
| A10 Composability | 0 | Existing test target retained |
| A11 Structured Failure | 0 | Production failure contract preserved and checked |
| A12 System Boundary | 0 | Production boundaries unchanged |

**Net +3**; hard gates A1/A3/A4/A7 have no negative score. This score describes the proposed
test-only change, not certification that the existing production driver meets every axiom.

## Related Documents and Duplication Check

- [Driver pin rollout](../m-driver-pin-rollout.md): defines the existing caller-emits contract
  and references this test lane; its parked rollout policy is not reopened.
- [Mission loop workbench](../v0_36_0/m-mission-loop-workbench.md): configuration/topology
  registry work, distinct from repairing the notification spy in an existing suite.
- [Motoko refusal sprint plan](../m-motoko-discovery-arm-discriminating-refusal-sprint-plan.md):
  neural match 0.45; relevant mutation discipline, but a different process-discovery instrument.
- Historical commits `63a0d2b32` and `9f267cf1f` are the production change and closest prior
  fixture repair. The sampled search had no duplicate-threshold match; targeted driver search
  and source history establish this correction's distinct scope.

## Handoff

Design author changes only this document and makes no Git writes. Route to planner, executor,
and independent evaluator under mission-control's unattended authority after the design gate.
Quorum round 1 was blocked by all three reviewers: Sol raised production boundedness, Gemini
requested project-default and status-capture evidence, and GLM requested real-fixture causal
correlation. This is the single permitted revision: V12-V15, the exact causal table, the
separate production follow-up, and corrected A5/net score address those objections. Re-review
must supply the verdict; this document does not self-approve. The measurements support keeping
the test-only correction separate from production timeout work.
