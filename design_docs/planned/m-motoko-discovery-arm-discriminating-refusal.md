# M-MOTOKO-DISCOVERY-ARM-DISCRIMINATING-REFUSAL: make the real wall-clock discovery refusal observable and reachable

**Status**: Planned
**Target**: v0.34.1
**Priority**: P1 (High — a REQUIRED-adjacent CI gate is vacuous: arm 33 of the 41-arm self-test cannot fail for the reason its name claims)
**Estimated**: 1 iteration (~3-4 hours implementation + mutation validation)
**Dependencies**: Row 6n of the motoko mission queue (issue #975). No new dependency; the two shell files are self-contained and the suite is hermetic.

## Problem Statement

`tools/eval/motoko_connection_probe.sh` is the motoko provider-connection probe. Its self-test
`tools/eval/test_motoko_connection_probe.sh` runs 41 arms and is a REQUIRED-adjacent CI gate
(GitHub Actions job "launchd drivers (bash 3.2)").

Arm 33 of that set (test_motoko_connection_probe.sh:449) is named
"descendant discovery refuses on the real wall-clock deadline" and asserts the generic caller-side
wrapper message `process-tree discovery failed`. That arm **cannot fail for the reason its name
claims**: every discovery failure funnels into that one wrapper string, so the arm passes whether
the walk was ended by the wall clock, the node ceiling, or the test stub.

**Current state (measured, iteration 31):**

- `descendant_pids` has three refusal branches and only two distinct messages:
  - line 182 — `PROBE_TEST_DESCENDANT_FAILURE=1` test stub → `process-tree discovery deadline expired`
  - line 187 — the REAL in-loop wall clock → `process-tree discovery deadline expired` (identical string)
  - line 195 — the node ceiling → `process-tree discovery exceeded $MAX_TREE_NODES nodes` (unique)
- the caller collapses all three (line 217): `sample_tree -> instrument_failure "process-tree discovery failed"`.
- Arm at :357 ("descendant discovery deadline refuses at the caller") drives the STUB (line 182) and
  asserts the wrapper. Arm at :449 drives the real walk and asserts **the same wrapper**.
- The node ceiling is the only one of the three branches with a message unique to itself.

**Impact:** the CI gate gives false confidence. Arm 33 claims to pin the real wall-clock deadline but
cannot discriminate the branches at all: each refusal branch independently suffices for the arm to pass
(measured — E2: neuter the wall clock alone → suite still green, arm 33 still ok; E7: neuter the node
ceiling alone → suite still green, arm 33 still ok). The gate cannot detect a regression in the
wall-clock branch, and it cannot distinguish the wall-clock branch from the node-ceiling branch at all.

## Goals

**Primary Goal:** Give the real in-loop wall-clock refusal in `descendant_pids` an observable that
**only it** can produce, and make arm :449 actually **reach** that branch deterministically, so the
arm can fail for the reason its name claims.

**Secondary Goal:** Close the refusal-branch-count gate gap so the `descendant_pids` echo-based
refusals (the exact branches this defect is about) are covered by the anti-vacuity gate, not silently
excluded from it.

**Success Metrics:** the baseline suite stays green (41 ok); the E2 mutant (neuter the wall clock) reds
the suite with arm :449 as the failing arm; a node-ceiling-only mutant does NOT red arm :449; an
addition-shaped mutant moves the branch-count gate.

## High-Impact Decisions

### (a) Observable for the real wall-clock branch — distinct messages per branch

The three refusal branches in `descendant_pids` get three distinct, self-documenting messages:

| line | branch | message after this work |
|---|---|---|
| 182 | test stub (`PROBE_TEST_DESCENDANT_FAILURE`) | `process-tree discovery deadline expired (test stub)` |
| 187 | real in-loop wall clock | `process-tree discovery deadline expired (wall clock)` |
| 195 | node ceiling | `process-tree discovery exceeded $MAX_TREE_NODES nodes` (unchanged) |

**Why this over the alternatives:**

- **Exit-code / diagnostic channel — rejected.** `descendant_pids` returns `1` for all three branches
  and the caller collapses them to `instrument_failure "process-tree discovery failed"`. Distinguishing
  the branches by exit code would require changing the caller's handling, which is more invasive and
  changes production semantics (Non-Goal R6). The message channel is already the probe's diagnostic
  surface; we make it discriminating rather than add a new channel.
- **Making only the stub's message distinct — rejected as insufficiently robust.** It is the minimal
  diff, but it leaves the real wall-clock message as the generic `process-tree discovery deadline
  expired`, which is exactly the over-subscription-prone string this defect is about. A future branch
  that reuses that string silently re-creates the bug.
- **Distinct messages per branch — chosen.** Each branch emits a string unique to itself, so any arm
  can assert exactly the branch it intends, and no two branches share a string. The stub's message is
  never directly asserted (arm :357 asserts the caller wrapper), so changing it is safe.

### (b) Arm assertions become non-overlapping

| arm | drives | asserts after this work | still meaningful because |
|---|---|---|---|
| :357 "descendant discovery deadline refuses at the caller" | test stub | `process-tree discovery failed` (unchanged) | tests the caller collapse of a `descendant_pids` failure |
| :449 "descendant discovery refuses on the real wall-clock deadline" | real walk | `process-tree discovery deadline expired (wall clock)` | tests the real wall-clock branch specifically |
| 40 "descendant discovery refuses on the node-count ceiling" | real walk, `MAX_TREE_NODES=3` | `process-tree discovery exceeded 3 nodes` (unchanged) | tests the node-ceiling branch specifically |

Each arm now asserts a distinct observable, so no two arms can pass for the same reason.

### (c) Making arm :449 reach the wall-clock branch deterministically

The wall clock fires on **wall time** (`date +%s > deadline`, ~1s with `PROBE_TIMEOUT_SECS=1`); the node
ceiling fires on **iteration count** (`visited > MAX_TREE_NODES`). Which fires first is a race that
depends on machine speed. This design does **not** try to win that race: the arm no longer asserts on
*which* bound fired, so it does not need to. It asserts a message only the wall-clock branch can emit
(`process-tree discovery deadline expired (wall clock)`), so the arm's verdict is independent of which
bound fires first. The node ceiling stays at its **default** `MAX_TREE_NODES` (4096) as a fail-fast
backstop: if the ceiling ever won the race, the arm would RED (loudly, correctly) rather than pass
vacuously — which is the property the arm has always lacked.

Measured (iteration 31): on the machine previously claimed to exhibit the race, at the DEFAULT ceiling,
the wall clock is what fires (E8: minimal fix only, no ceiling change → rc=0, 41 ok, 0 not ok, 50s). The
`PROBE_MAX_TREE_NODES=1000000` mechanism is **not needed for the arm to pass** and is deleted.

The 5x widened arm cap (test_motoko_connection_probe.sh:447-452) is **kept**. Measured evidence: E8 clean
= 50s whole-suite, T1 mutant = 44s whole-suite. The widening exists for a CONTENDED CI runner, which is
not measured here (see decision (d)); removing it is a separate, runner-dependent change this doc has no
measurement for, and keeping it costs nothing because the arm now reds on a message rather than waiting
for a cap.

### (d) Runner risk — what cannot be established locally

The controller measured (E5) that on the CI runner under V1's PR #971 configuration, discovery did not
refuse at all — both lanes completed with `driver_rc=0` and an **empty peer set**, and the run died
downstream on the empty-peer-set guard. That is a fixture that failed to produce a live process tree to
walk, not a refusal. This is **UNINFORMATIVE-UNDER-SANDBOX**: the local hermetic suite (E1) cannot
reproduce the CI runner's contention, so the local result is not authoritative for the runner.

**The measurement that settles it** (no speculative runner fix is designed here): after this work lands
on a PR, read the "launchd drivers (bash 3.2)" leg of that PR's CI run and confirm arm :449 passes with
the new unique message. If it fails, the failure must be the expected refusal (the wall-clock message
missing), not a hang and not an empty-peer-set guard. That leg is the only place the runner's behavior
is observable.

### (e) The #971 collision

PR #971 (V1 mission's `mission/iter306-probe-deadline-race`, OPEN/MERGEABLE, head `8a384e81b`) touches
exactly the same two files. It introduces `PROBE_TREE_DISCOVERY_SECS` (a discovery deadline independent
of the lane deadline) and re-parameterises arm :449 as `PROBE_TIMEOUT_SECS=60 PROBE_TREE_DISCOVERY_SECS=1`,
leaving arm :449 asserting the generic wrapper. It is **orthogonal** to this doc's defect: it changes
*which deadline* bounds discovery, this doc changes *which message* the wall-clock branch emits and *how
the arm reaches it*. Neither supersedes the other.

- **Whichever lands first, the other re-applies its changes on top of the merged tree** (a rebase onto
  the merged head). The two changes are compatible: this doc's distinct messages compose with #971's
  `PROBE_TREE_DISCOVERY_SECS`.
- **motoko does NOT touch #971.** It is another mission's PR; motoko has no branch in its worktree list
  for it. The hand-over is a **message** to the V1 mission (that the two files have changed and the
  second-to-merge must rebase), not a rebase performed by motoko.

### (f) Refusal-branch count gate

The gate (test_motoko_connection_probe.sh:707-722) counts `instrument_failure "` (19) + `|| usage$` (5)
= `expected_refusal_branches=24`. The three `descendant_pids` echo-based refusals are **not** counted —
a gap, and precisely the branches this defect is about. The design extends the gate to count them:

- add `actual_echo_refusals=$(grep -c 'echo "process-tree discovery' "$probe")` (counts exactly the 3
  descendant_pids refusals; verified V6),
- extend the anti-vacuity floor to require `actual_echo_refusals != 0`,
- `expected_refusal_branches` moves **24 → 27** (19 + 5 + 3).

The count moves because the gate now covers the previously-unguarded echo refusals, not because this
work adds a new refusal branch (it does not — it only re-words two existing messages).

## Design Freeze

- The three `descendant_pids` branches keep distinct messages; no two branches share a string.
- Arm :449 asserts `process-tree discovery deadline expired (wall clock)`; the node ceiling stays at its
  default `MAX_TREE_NODES` (4096) as a fail-fast backstop; the 5x widened cap is kept.
- The refusal-branch gate counts echo-based process-tree refusals and `expected_refusal_branches=27`.
- No change to the probe's production refusal *outcome* (exit 1, caller wrapper); only the diagnostic
  stderr strings at lines 182/187 change.
- No fix to #971; no runner-only speculative change; no widening into row 6o.

## Verification Log

Base commit: **a223e7274** (`git log --oneline -1` → `a223e7274 M2: pipeline.BuildCanonicalJSON + hidden internal-dump-iface (iteration 312) (#998)`). All rows measured in this worktree at that commit unless noted. Rows V1-V11 are re-derived here; E1-E8 and T1 are controller-measured, iteration 31, and cited as such.

| # | Claim | Command | Observed output |
|---|---|---|---|
| V1 | Base commit is a223e7274 | `git log --oneline -1` | `a223e7274 M2: pipeline.BuildCanonicalJSON + hidden internal-dump-iface (iteration 312) (#998)` |
| V2 | Baseline suite is green (E1) | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0; `ok` lines 41; `not ok` lines 0; final `PASS: 41 probe self-test arms ran` |
| V3 | Probe parses | `bash -n tools/eval/motoko_connection_probe.sh` | rc=0 |
| V4 | Test parses | `bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0 |
| V5 | Refusal-branch gate components sum to 24 | `grep -c 'instrument_failure "' tools/eval/motoko_connection_probe.sh`; `grep -cE '\|\| usage$' tools/eval/motoko_connection_probe.sh`; sum | 19; 5; 24 |
| V6 | Exactly 3 echo-based process-tree refusals exist | `grep -c 'echo "process-tree discovery' tools/eval/motoko_connection_probe.sh` | 3 (lines 182, 187, 195) |
| V7 | Three branches, two distinct messages (E4) | `read` lines 178-217 of the probe | line 182 and 187 both emit `process-tree discovery deadline expired`; line 195 emits `process-tree discovery exceeded $MAX_TREE_NODES nodes`; line 217 collapses to `process-tree discovery failed` |
| V8 | E2 mutant (neuter wall clock) does NOT red the suite (E2) | apply `if false && (( $(date +%s) > deadline ))` at line 186; `bash -n`; run suite; restore from `cp` backup | sha256 `7a75c698…` → `e22a74eb…`; parses rc=0; suite rc=0, 41 ok, arm 33 still `ok`; restored byte-identical, `git status --porcelain` 0 lines |
| V9 | Wall-clock arm reaches the wall clock on this machine at the default ceiling | run arm :449's env directly against the probe, capture stderr, at default `MAX_TREE_NODES` | emits `process-tree discovery deadline expired` then `INSTRUMENT FAILURE: process-tree discovery failed`; rc=1. (Confirms the wall clock fires first here at the default ceiling.) |
| V10 | #971 is OPEN/MERGEABLE and touches exactly the two files; `PROBE_TREE_DISCOVERY_SECS` absent at motoko HEAD (E6) | `gh pr view 971 --json state,mergeable,author,baseRefName,headRefName,headRefOid,createdAt,updatedAt,files`; `grep -c "PROBE_TREE_DISCOVERY_SECS"` both files; control `grep -c "PROBE_MAX_TREE_NODES"`; negative control `grep -c "PROBE_ZZZ_INVENTED"` | OPEN, MERGEABLE, author sunholo-voight-kampff, base dev, head `8a384e81b…`, files = exactly the two shell files; `PROBE_TREE_DISCOVERY_SECS` = 0 in both; control `PROBE_MAX_TREE_NODES` = 2 and 3; negative control = 0 |
| V11 | E5 runner finding (controller-measured, scoped to V1's PR #971 tree, NOT motoko HEAD) | controller read GitHub Actions job 99402730557, "launchd drivers (bash 3.2)", PR #971 head `8a384e81b…` | single failing line `not ok - descendant discovery refuses on the real wall-clock deadline / lacked expected message: process-tree discovery failed`; surrounding log shows `OPENROUTER_API_KEY: UNSET`, `lane=treatment driver_rc=0 peers: []`, `lane=control driver_rc=0 peers: []`, `INSTRUMENT FAILURE: empty peer set; absence of evidence cannot prove routing`; 32 passing `ok` lines. Re-fetch attempt here returned 404 (`gh run view 99402730557`), so this row is cited as controller-measured, not re-derived. |
| E3 | Sound reading of the controller's E3 (corrected, iteration 31) | controller's E3: neuter BOTH the wall clock and the node ceiling | the walk hangs. Sound reading: with the wall clock dead, the node ceiling is what stops the walk; it says nothing about which branch wins when both are alive. The prior inference "the node ceiling fires first on the controller's machine" does NOT follow and is withdrawn. |
| E7 | Neuter the node ceiling ALONE, wall clock alive (controller-measured, iteration 31) | apply `if false && (( visited > MAX_TREE_NODES ))` at the node-ceiling site; `bash -n`; run suite; restore byte-identical | parses rc=0; effect asserted (neutered 0->1, live 1->0); suite rc=1, 39 ok, 1 not ok — the ONLY failing arm is arm 40 ("descendant discovery refuses on the node-count ceiling"); arm 33 ("real wall-clock deadline") still `ok`. Restored byte-identical. |
| E8 | Minimal fix only, NO ceiling change, default MAX_TREE_NODES=4096 (controller-measured, iteration 31) | apply only the message distinction (probe line 182 -> "(test stub)", line 187 -> "(wall clock)") and arm :449's new assertion; `bash -n` both; run suite; restore byte-identical | both files `bash -n` rc=0; effect asserted (2 distinct messages present); `/bin/bash tools/eval/test_motoko_connection_probe.sh` -> rc=0, 41 ok, 0 not ok, 50s. On the machine previously claimed to exhibit the race, at the DEFAULT ceiling, the WALL CLOCK is what fires. `PROBE_MAX_TREE_NODES=1000000` is not needed for the arm to pass. Restored byte-identical, `git status --porcelain` 0 lines. |
| T1 | Load-bearing acceptance, measured (controller-measured, iteration 31) | apply the SAME minimal fix as E8 AND the E2 mutant (wall clock neutered), ceiling untouched at default (asserted: neutered wall-clock site = 1, live node-ceiling site = 1); run suite; restore byte-identical | rc=1, 44 seconds; failing arm exactly: `not ok - descendant discovery refuses on the real wall-clock deadline / lacked expected message: process-tree discovery deadline expired (wall clock)`. Clean rc=0/50s vs mutant rc=1/44s — outcomes DIFFER, and the mutant run is FASTER than the clean one, not a 120-second hang. Restored byte-identical; both files sha256-verified; `git status --porcelain` 0 lines. |

## Solution Design

### Overview

The fix is small and confined to two shell files. In the probe, the two `descendant_pids` refusal
messages that currently share a string are made distinct, so the real wall-clock branch has a unique
observable. In the test, arm :449 asserts that unique observable; the node ceiling stays at its default
`MAX_TREE_NODES` (4096) as a fail-fast backstop; the 5x widened cap is kept; and the refusal-branch gate
is extended to count the echo-based refusals it previously ignored.

### Architecture

- **Probe (`motoko_connection_probe.sh`)** — `descendant_pids` (lines 178-209) is the only function
  touched. Its three refusal branches emit three distinct messages. The caller (`sample_tree`, line
  217) is unchanged: it still collapses any `descendant_pids` failure to
  `instrument_failure "process-tree discovery failed"`. Production refusal *outcome* is unchanged.
- **Test (`test_motoko_connection_probe.sh`)** — arm :449 (lines 449-452) and the refusal-branch gate
  (lines 707-722) are the only regions touched. Arm :449's assertion changes (no env change — the node
  ceiling stays at its default); the gate's counters and expected total change.

### Conflict Surface

**What else lives in these two shell files (must still work):**

- `motoko_connection_probe.sh` (310 lines): `instrument_failure`, `usage`, `peer_host`,
  `is_loopback_host`, `or_ip_member`, `classify_lsof`, `assert_nonempty`, `assert_treatment`,
  `assert_control`, the `--classify-fixture`/`--assert-*` dispatch, the main arg/env/platform/dependency
  gates, `retain_diagnostics`/`cleanup`, `sample_tree`, `assert_pid_scope`, `run_lane`, and the final
  treatment/control assertion flow. None of these change.
- `test_motoko_connection_probe.sh` (728 lines): the classification fixtures, the `--assert-*` arms, the
  `live_bin` hermetic toolchain, `run_live`, the usage/dependency/platform arms, the stub arm (:357), the
  node-ceiling arm (40), the run_lane fixture arm, the arm-cap and orphan-grandchild arms, and the
  `report_arm_cap` coverage arm. None of these change.

**Positions touched:**

- Probe lines 182 and 187 (the two echo messages).
- Test lines 449-452 (arm :449 assertion) and lines 707-722 (refusal-branch gate).

**What must still work:** all 41 arms pass (V2); both files parse under `bash -n` (V3, V4); the node
ceiling arm (40) still asserts `process-tree discovery exceeded 3 nodes`; the stub arm (:357) still
asserts the caller wrapper; the run_lane fixture arm still passes; the gate arm still passes with the
new count.

**What deliberately changes:** the two probe stderr strings at lines 182/187 (cosmetic diagnostic
change; the refusal outcome is unchanged); arm :449's assertion; the gate's count (24 → 27) and
anti-vacuity floor.

**Shell constraints:** both files are shell; `bash -n` is the only syntax gate, and the CI shell is GNU
bash 3.2.57 on arm64-apple-darwin25. No bash-4+ constructs are introduced (no associative arrays, no
`${var^^}`, no `mapfile`). The added `grep -c` and arithmetic are bash-3.2-safe.

## Implementation Plan

**Phase 1 — probe messages (~15 min).** In `motoko_connection_probe.sh`, change line 182 to
`echo "process-tree discovery deadline expired (test stub)" >&2` and line 187 to
`echo "process-tree discovery deadline expired (wall clock)" >&2`. `bash -n` must stay rc=0.

**Phase 2 — arm :449 (~20 min).** In `test_motoko_connection_probe.sh`, replace the arm :449 block
(lines 447-452) with a single `expect_failure` asserting `process-tree discovery deadline expired
(wall clock)`. No env change: the node ceiling stays at its default `MAX_TREE_NODES` (4096) as a
fail-fast backstop. The `_arm_cap_saved` / `ARM_CAP_SECS=$(( ARM_CAP_SECS * 5 ))` widening is KEPT (see
decision (c)).

**Phase 3 — refusal-branch gate (~30 min).** In `test_motoko_connection_probe.sh`, add
`actual_echo_refusals=$(grep -c 'echo "process-tree discovery' "$probe")`, extend the anti-vacuity floor
to require it non-zero, set `expected_refusal_branches=27`, and add it to the total.

**Phase 4 — baseline + mutation validation (~2-3 hours).** Run the baseline suite (must stay 41 ok), then
run the E2 mutant, the node-ceiling-only mutant, and the addition-shaped mutant (see Testing Strategy),
recording each mutant's red set.

## Files to Modify/Create

**Modified:**
- `tools/eval/motoko_connection_probe.sh` — lines 182, 187 (distinct refusal messages).
- `tools/eval/test_motoko_connection_probe.sh` — arm :449 (assertion); refusal-branch
  gate (echo counter, anti-vacuity, `expected_refusal_branches=27`).

**Created:** none.

## Success Criteria

Each criterion names a command and its result on the **unmodified** tree (a criterion already red at
base is broken). All are green at base and must stay green after this work.

| # | Command | Result at base (a223e7274) | Must hold after |
|---|---|---|---|
| S1 | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, 41 ok, 0 not ok, `PASS: 41 probe self-test arms ran` (V2) | rc=0, 41 ok, 0 not ok |
| S2 | `bash -n tools/eval/motoko_connection_probe.sh` | rc=0 (V3) | rc=0 |
| S3 | `bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0 (V4) | rc=0 |
| S4 | `grep -c 'echo "process-tree discovery' tools/eval/motoko_connection_probe.sh` | 3 (V6) | 3 (the two re-worded messages keep the count) |
| S5 | `grep -c 'instrument_failure "' tools/eval/motoko_connection_probe.sh` + `grep -cE '\|\| usage$'` + echo count | 24 (V5) | 27 (gate now counts the echo refusals) |

## Testing Strategy

Each row names the OBSERVABLE and confirms it is produced **by** the mechanism, not alongside it. Each
mutant's expected red set is produced by RUNNING it, not asserted.

| # | Mutant | Kills which mutation | Observable | Produced by |
|---|---|---|---|---|
| T1 | E2 mutant: neuter the in-loop wall clock (`if false && (( $(date +%s) > deadline ))`) | the vacuity this doc fixes | suite reds; arm :449 is the failing arm with `lacked expected message: process-tree discovery deadline expired (wall clock)` | the wall-clock mechanism — the wall clock is dead, so its unique message cannot appear; the node ceiling (default 4096) or the 120s cap ends the walk with a different message. **MEASURED by the controller at iteration 31: rc=1, 44s, failing arm exactly as stated; re-run required by the executor.** |
| T2 | node-ceiling-only mutant: neuter `if (( visited > MAX_TREE_NODES ))` | the two arms being separable | arm :449 still passes (wall clock fires at ~1s); arm 40 reds with `lacked expected message: process-tree discovery exceeded 3 nodes` | the wall-clock mechanism for :449 (alive, fires at ~1s); the node-ceiling mechanism for arm 40 (dead, so its unique message cannot appear). **MEASURED by the controller at iteration 31 (E7): suite rc=1, 39 ok, 1 not ok — the ONLY failing arm is arm 40; arm 33 still `ok`; re-run required by the executor.** |
| T3 | addition-shaped mutant: add a new `instrument_failure "..."` refusal branch to the probe | the gate LOOKS (a removal proves a check FIRES; only an addition proves it LOOKS) | gate reds with `refusal-branch drift: probe has 28 refusal branches` | the gate mechanism — the count moved 27 → 28 and the gate refuses |

T1 is the load-bearing acceptance: it is red at base (V8) and must be green after this work. T2 proves
the wall-clock arm no longer depends on the node ceiling. T3 proves the extended gate still catches new
refusal branches. T1 and T2 (E7) were measured by the controller at iteration 31; the executor must
re-run them on the modified tree.

## Deferred Decisions

- **Whether the node ceiling ever wins the race on a contended CI runner.** The arm no longer depends on
  which bound fires, so this is not load-bearing for the arm's verdict. If a future CI leg shows the
  node ceiling firing before the wall clock on arm :449, the arm REDS (loudly, correctly) rather than
  passing vacuously; the measurement that settles it is the wall-clock arm's stderr on the "launchd
  drivers (bash 3.2)" leg.
- **Whether the gate extension is kept.** It is in scope here (it closes the exact gap this defect is
  about). If a reviewer judges it out of scope, the count stays 24 (echo messages are not counted) and
  the gate gap is tracked as a follow-up row; the measurement that settles it is the gate arm's pass/fail
  on the suite.
- **Runner behavior under contention (E5).** Cannot be established locally; the measurement is the
  "launchd drivers (bash 3.2)" leg on the PR after this work lands (see decision (d)).

## Non-Goals

- **No change to the live probe's production semantics.** The refusal *outcome* (exit 1, caller wrapper)
  is unchanged; only two diagnostic stderr strings change.
- **No fix to #971.** It is another mission's PR; motoko does not touch it. The hand-over is a message.
- **No runner-only speculative change.** The runner risk is named and measured on CI, not fixed blind.
- **No widening into row 6o** (the SIGKILL-escalation gap). That is a separate queue row.

## Timeline

One iteration (~3-4 hours): Phase 1 (~15 min) → Phase 2 (~20 min) → Phase 3 (~30 min) → Phase 4 baseline
+ mutation validation (~2 hours) → doc/verification polish (~1 hour). The design is now SMALLER than the
original estimate: three edit sites (probe lines 182/187, arm :449 assertion, gate) and no ceiling
manipulation, so the previous ~4-6 hour estimate is reduced. If the mutation validation exceeds the
budget, the gate extension (Phase 3) is the first cut, per Deferred Decisions.
