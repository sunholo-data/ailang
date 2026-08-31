# Sprint plan — M-MOTOKO-CONNECTION-PROBE-RUN-LANE-HARNESS

**Design authority:** `design_docs/planned/m-motoko-connection-probe-run-lane-harness.md`

**Mission iteration:** 29

**Planner lane:** `codex:gpt-5.6-sol anthropic-fallback:fail-closed:planner-lane-field-missing`

**Planned base:** detached `HEAD` at `0b35abd5d0e7` on 2026-08-30

**Duration:** one bounded iteration, about 4–6 hours

**Estimate:** about 100 LOC, all in the shell self-test (roughly 25 LOC fixture/helper and 75 LOC behavioral arm/evidence)

**Risk:** medium — small diff, but real process ownership, timing, cleanup, and bash 3.2 semantics are load-bearing

The design doc wins wherever this plan disagrees. This plan adds execution order, measured gate
baselines, and the exact mutation/evidence discipline. It does not authorize a production behavior
change.

## 1. Scope and completion boundary

The sprint modifies only `tools/eval/test_motoko_connection_probe.sh`. It extends the existing
hermetic live-path fixture, refactors the existing cwd-survivor query into one parameterized helper,
and adds one behavioral arm that launches the full production probe through `run_lane`.

Expected production diff in `tools/eval/motoko_connection_probe.sh`: **zero**. `PROBE_UNDER_TEST`
already lets the self-test execute an unmutated or temp-copy-mutated production script. If the
executor discovers that a production hook is necessary, stop and return to design approval; do not
grow this sprint.

Out of scope: real Ollama/model work, GPU access, OpenRouter calls, real DNS, required socket binds,
changes to routing/classification, Go helpers, make/CI wiring, and edits to the approved design doc.
The pre-existing optional loopback-socket observation may remain `UNINFORMATIVE`; it is not an
acceptance signal for this sprint.

## 2. Planner measurements and gate baselines

All measurements below were run on `Darwin arm64` with `/bin/bash 3.2.57(1)-release` and real
`lsof` resolved by `command -p -v lsof` to `/usr/sbin/lsof`.

| Command / fact | Baseline on 2026-08-30 | Gate ruling |
|---|---|---|
| `/bin/bash -n tools/eval/motoko_connection_probe.sh` | rc=0 | mandatory before and after every mutant |
| `/bin/bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0 | mandatory after implementation and final restore |
| `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0; 40 arms; final `PASS: 40 probe self-test arms ran` | primary regression gate; supported baseline is green |
| `command -p -v lsof` plus executable check | rc=0; `/usr/sbin/lsof` | mandatory precondition for the new arm; never fall back to the PATH stub |
| `make test-launchd-drivers` | rc=2 at base after 54 pin-root and 27 notify checks passed; `tools/launchd/test_mission_routing.sh` then reported 4 unrelated planner-allowlist failures | **red at base, so not locally discriminating today**; retain as the macOS CI-facing integration target and do not attribute this pre-existing routing red to the sprint |
| CI wiring | `.github/workflows/ci.yml:536-555`: job `launchd drivers (bash 3.2)`, `runs-on: macos-latest`, asserts bash major 3, 15-minute cap, runs `make test-launchd-drivers` | authoritative platform context; no CI edit needed |

The exact probe self-test is the executor's local green gate. The aggregate make target must still
contain the probe self-test and both syntax checks. Completion also requires a green macOS aggregate
result after the unrelated routing baseline is repaired, or an evaluator/controller record that the
only remaining failures are byte-identical to the four measured baseline routing failures and that
the target has not yet reached this sprint's member. A red-at-base aggregate is never evidence for or
against the new arm.

Velocity calibration is the immediately preceding row-6g change: 86 insertions / 9 deletions across
the production probe and self-test, completed in one mission iteration. This follow-up is narrower
(one modified file, no new production mechanism), so 4–6 hours including two actual mutation runs is
conservative.

## 3. Milestones

| ID | Milestone | LOC | Time | Depends on |
|---|---|---:|---:|---|
| M1 | Real-`lsof` helper and ready-capable driver fixture | ~35 | 1.5 h | — |
| M2 | Full-production `run_lane` timeout arm | ~50 | 2 h | M1 |
| M3 | Actual mutants, evidence table, and final gates | ~15 comments/evidence plumbing | 1.5–2.5 h | M2 |

### M1 — Real-`lsof` helper and ready-capable fixture

**Owned file:** `tools/eval/test_motoko_connection_probe.sh`

Tasks:

1. Before constructing or using the hermetic `live_bin`, resolve `REAL_LSOF` with
   `command -p -v lsof`, require an absolute executable, and separately determine whether the host
   is the supported Darwin platform. Missing/non-executable real `lsof` is a named hard failure on
   Darwin; only unsupported non-Darwin platforms set a run-lane-arm skip flag. Neither path may
   select the later PATH `lsof` stub.
2. Replace `orphan_fixture_pids` with `fixture_sleep_pids <fixture_dir>`. The helper must invoke
   `"$REAL_LSOF" -a -c sleep -d cwd` and filter exact canonical cwd. Update the existing
   `run_bounded` orphan arm and cleanup to use the parameterized helper unchanged in behavior.
3. Extend the existing `ailang-stub` with a mode guarded by
   `PROBE_TEST_RUN_LANE_GRANDCHILD_CWD`. In that mode it canonicalizes and verifies cwd, starts a
   distinctive long `sleep`, atomically publishes a ready file through the supplied same-directory
   temporary path, and waits. The ready payload contains exactly usable `wrapper_pid`, `child_pid`,
   and canonical `cwd` fields.
4. Keep the ordinary treatment/control stub path unchanged when the new environment variable is
   absent. No real binary, DNS, socket, GPU, Ollama, or network call may escape the hermetic PATH.

Acceptance:

- The existing wrapper-grandchild arm remains green through `fixture_sleep_pids`.
- A hostile PATH `lsof` stub remains responsible for production TCP samples, while cwd checks use
  the absolute real `lsof`; marker evidence distinguishes both calls.
- Readiness is published only after `cd`/`pwd -P` verification and child spawn, using `mv` from the
  same directory.
- On Darwin with missing/non-executable real `lsof`, fail by the exact name
  `not ok - run_lane fixture arm requires real lsof on Darwin CI target`.
- On unsupported non-Darwin platforms, output names
  `UNINFORMATIVE: run_lane fixture arm requires real lsof for cwd survivor checks`; this one arm
  does not call `pass_arm` and does not claim a behavioral `ok`.
- Both bash syntax gates remain rc=0.

### M2 — Full-production `run_lane` timeout arm

**Owned file:** `tools/eval/test_motoko_connection_probe.sh`

Tasks:

1. Create a unique canonical fixture cwd, ready path, ready-temp path, marker path, and captured
   stdout/stderr paths under the self-test temp directory. Register cleanup before launch.
2. Add a small run-lane fixture harness that launches
   `/bin/bash "$probe" treatment control <artifact> numeric_modulo` asynchronously with hermetic
   PATH, `AILANG_BIN=ailang-stub`, `PROBE_TIMEOUT_SECS=2`, and the grandchild fixture env. Run this
   harness through existing `run_bounded`, or one shared parameterized process-group runner reused
   by `run_bounded`, with emergency cap `RUN_LANE_OUTER_CAP_SECS=$(( PROBE_TIMEOUT_SECS + 10 ))`.
   Production `run_lane` remains the first and only cleanup mechanism before that later emergency
   deadline.
3. Wait for readiness inside the harness under its own positive integer cap (default 5 seconds).
   On expiry, write a named readiness-failure marker and return non-zero; the surrounding shared
   process-group runner owns emergency termination if the harness itself does not return.
4. Parse and validate every ready field: integer wrapper/child PIDs, exact canonical cwd, distinct
   relevant PIDs. Before the production timeout, require `kill -0 "$child_pid"` to succeed after
   reading the ready file. Do not call `fixture_sleep_pids` for pre-timeout readiness; reserve it for
   the post-production-timeout survivor verdict.
5. The reused/shared outer runner must record a distinct emergency-cap marker/result and require
   that the cap did not fire for the baseline or either expected mutant. If the cap fires, it must
   TERM, then KILL, the harness-owned process group and perform a bounded reap; report a named
   harness leak/hang failure, never a successful production-timeout result.
6. Accept the probe's non-zero exit only if captured stderr contains
   `INSTRUMENT FAILURE: lane treatment exceeded 2s sampling deadline` (or, for the job-control
   mutant only, the design-authorized named degraded process-group line). Marker evidence must show
   the production CLI shape and calls to stub `uname`, `dig`, `ailang-stub`, `pgrep`, and PATH
   `lsof`.
7. After probe exit, collect survivor PIDs before cleanup. Baseline requires exactly zero. Always
   cleanup and re-query until zero under a final bound before pass/fail is returned.
8. Emit a stable TAP diagnostic carrying `ready`, wrapper/child fields, timeout/degraded marker,
   pre-timeout `kill -0` result, distinct outer-cap result, survivor count, and probe rc. These are
   the evidence source for IV1–IV3; the temp directory disappears at suite exit.

Acceptance:

- On supported macOS, the suite adds exactly one behavioral `ok` arm (current 40 → expected 41).
- Ready file observed = yes; wrapper PID, child PID, and verified cwd are present.
- Ready child passes `kill -0` before timeout; the fixture-cwd `lsof` oracle is used only after the
  production timeout/cleanup completes.
- The emergency cap is exactly `PROBE_TIMEOUT_SECS + 10`, uses `run_bounded` or its one shared
  process-group runner, records a distinct result, and does not fire.
- Baseline timeout marker is observed, probe rc is non-zero as expected, post-timeout survivor count
  is 0, cleanup count is 0, and overall suite rc is 0.
- The arm reaches the full production live form and production `run_lane`; no copied timeout helper
  can satisfy it.
- The arm has no required socket bind, GPU, Ollama, real DNS, or real network dependency.

### M3 — Mutation proof and final validation

Run mutants against temp copies only. Do not edit and restore the worktree production probe. For
each copy: record pristine and mutant SHA-256, assert exactly the intended production `run_lane`
occurrences changed, run production `bash -n` (must stay rc=0), then invoke the self-test with
`PROBE_UNDER_TEST=<absolute-temp-copy>`.

| Row | Actual probe under test | Readiness / timeout required | Emergency outer cap | Survivor and suite requirement |
|---|---|---|---|---|
| IV1 baseline | unmutated production probe | ready=yes with wrapper PID, child PID, verified cwd; ready child passes `kill -0`; exact treatment sampling-deadline line | distinct result says did not fire | post-timeout survivors=0; suite rc=0; supported macOS arm count 41 |
| IV2 mutant A | in production `run_lane` only, `kill -TERM "-$pid"` → `kill -TERM "$pid"` and `kill -9 "-$pid"` → `kill -9 "$pid"` | mutation assertions exact; syntax rc=0; ready=yes; child passes `kill -0`; timeout=yes | distinct result says did not fire | post-timeout survivors >0 before cleanup; named run-lane arm fails; suite rc non-zero; cleanup returns survivors to 0 |
| IV3 mutant B | neuter production `run_lane`'s `set -m` only; do not touch self-test `run_bounded` | mutation assertions exact; syntax rc=0; ready=yes; child passes `kill -0`; timeout=yes or exact `INSTRUMENT DEGRADED` group-ownership refusal | distinct result says did not fire | post-timeout survivors >0 before cleanup, or the named degraded branch plus an independently non-zero run-lane arm; suite rc non-zero; cleanup returns survivors to 0 |

Mutant discipline:

1. Copy production probe into a `mktemp -d` directory and make it executable.
2. Record pre-SHA and expected source occurrence counts.
3. Apply one mutation; assert SHA differs and the old/new parsed text counts changed exactly as
   prescribed. A mutation that did not land is no evidence.
4. Require `/bin/bash -n <mutant>` rc=0. A syntax-red mutant is no evidence.
5. Run the full self-test with `PROBE_UNDER_TEST`; record ready fields, pre-timeout `kill -0`,
   timeout/degraded line, distinct non-firing outer-cap result, post-timeout survivors before test
   cleanup, failing arm text, and suite rc.
6. Delete only the explicit temp directory. Verify the worktree production probe SHA never changed.
7. Re-run IV1 and both syntax gates after both mutants.

Final gates:

```text
/bin/bash -n tools/eval/motoko_connection_probe.sh
/bin/bash -n tools/eval/test_motoko_connection_probe.sh
/bin/bash tools/eval/test_motoko_connection_probe.sh
make test-launchd-drivers
```

The first three must be green locally on supported macOS. The fourth is the macOS bash-3.2 CI-facing
gate. Because its planner-routing member is rc=2 at this sprint's measured base, compare any local
failure against the four recorded baseline failures and never call that aggregate a new regression
without a changed failure. The official macOS CI leg must ultimately run it with a 15-minute cap.

Record IV1–IV3 actual evidence in the sprint JSON milestone notes and executor/evaluator handoff.
Do not edit the approved design doc merely to fill its template rows.

## 4. Pause points and risks

- **Pause:** baseline targeted self-test or either syntax gate is red before editing.
- **Pause:** real `lsof` is missing/non-executable or cannot observe a known fixture-cwd sleep on
  supported macOS; this is a named hard failure, not an uninformative skip.
- **Pause:** readiness is flaky under a 5-second cap in two consecutive clean runs. Measure before
  widening; do not let the ready cap approach the production timeout semantics silently.
- **Pause:** either actual mutant stays green. Strengthen the same arm; do not add a second bespoke
  survivor helper or weaken the acceptance row.
- **Pause:** implementation appears to require a production hook or changes a refusal-branch count.
  Return to design approval.

Main risks and mitigations:

- PID reuse / unrelated sleeps: readiness names the exact child PID for `kill -0`; the later unique
  canonical fixture-cwd survivor query disambiguates unrelated processes.
- Vacuous zero survivors: atomic readiness plus pre-timeout `kill -0` of the ready child; reserve
  real-`lsof` enumeration for the post-production-timeout survivor verdict.
- Killing the self-test's own group: production guard remains unchanged; mutant B must fail through
  degraded/survivor behavior, never by sending an unsafe negative-PID signal.
- Emergency-cap safety: reuse `run_bounded` or one shared process-group runner so cap firing performs
  TERM/KILL only against the harness-owned process group and bounds the reap.
- Cleanup pollution: shared helper, cleanup after recording the post-timeout verdict on every exit
  path, bounded final zero-survivor check.
- CI shell drift: the existing macOS job asserts bash major 3 before the aggregate target.

## 5. Definition of done

- M1–M3 criteria are satisfied with no scope expansion and no production diff.
- IV1, IV2, and IV3 contain actual—not predicted—readiness, pre-timeout `kill -0`,
  timeout/degraded, distinct non-firing outer-cap, post-timeout survivor, cleanup, syntax, and
  suite-rc evidence.
- Supported baseline is 41/41 and both actual mutants are killed for the stated behavior.
- Existing 40 arms remain green; the existing orphan arm uses the same absolute-real-`lsof` helper.
- Final syntax and targeted self-test gates are green; macOS bash-3.2 aggregate status is recorded
  with the red-at-base routing caveat handled explicitly.
- `git status --short` contains no mutation backups, temp artifacts, or production probe edit.
