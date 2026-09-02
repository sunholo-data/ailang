# M-MOTOKO-DISCOVERY-ARM-DISCRIMINATING-REFUSAL: make the real wall-clock discovery refusal observable and reachable

**Status**: REVISED per attended human ruling D-MOTOKO-6N-1 (Mark Edmondson, 2026-09-01, commit `878e0a5a0`) — awaiting design quorum round 3. Rounds 1-2 were BLOCKED 3/3; the ruling selected §D4 (scoped ceiling), now applied throughout. See Quorum Verification Log.
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
addition-shaped mutant moves the branch-count gate; a scoping mutant (the ceiling override moved from
arm :449's `env` line to suite scope) reds a named arm, proving the override is LOCAL, not global.

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
| :449 "descendant discovery refuses on the real wall-clock deadline" | real walk, `PROBE_MAX_TREE_NODES=50000` scoped to its own `env` line (D4) | `process-tree discovery deadline expired (wall clock)` | tests the real wall-clock branch specifically; the scoped ceiling makes the node bound structurally unreachable inside the wall-clock window (see (c)) |
| 40 "descendant discovery refuses on the node-count ceiling" | real walk, `MAX_TREE_NODES=3` | `process-tree discovery exceeded 3 nodes` (unchanged) | tests the node-ceiling branch specifically |

Each arm now asserts a distinct observable, so no two arms can pass for the same reason.

### (c) Making arm :449 reach the wall-clock branch deterministically — D4, per the attended ruling

The wall clock fires on **wall time** (`date +%s > deadline`, nominally ~1s with `PROBE_TIMEOUT_SECS=1`);
the node ceiling fires on **iteration count** (`visited > MAX_TREE_NODES`). Which fires first is a race
that depends on machine speed. Note that `date +%s` has 1-second granularity, so the true wall-clock
window is **0-2s**, not 1s: the deadline is one whole second past the start second, and the in-loop
check can trip anywhere from almost immediately (walk starts just before the clock rolls) to just under
2s in.

**Position (supersedes the round-1 and round-2 positions, per attended ruling D-MOTOKO-6N-1):** arm
:449's own `env` line carries a scoped override, **`PROBE_MAX_TREE_NODES=50000`** — the §D4 design.
Every other arm keeps the default 4096: the override is a per-command `env`-line assignment, which does
not persist past the one probe invocation, and nothing is exported at suite scope (the gate refuses if
it is — see (f) and T4). This is the synthesis the iteration-31 evaluator recorded in §D4 and Mark
Edmondson selected, attended, on 2026-09-01.

**Why the scoped form answers both rejections at once.** Round 1 rejected the original
`PROBE_MAX_TREE_NODES=1000000` override: effectively unbounded, it turns a wall-clock regression into a
hang that only the arm cap can end (at the measured stub rates below, 1,000,000 iterations ≈ 1,532-2,106s,
so the 600s widened cap ALWAYS fires first — a cap-shaped timeout, not a message-shaped red). Round 2
rejected removing the override: at the default 4096 the race stays live, and a fast CI runner could hit
the ceiling inside the window and spuriously RED a clean tree. A scoped middle value is neither: it is
bounded, so a regression still fail-fasts on the ceiling's own message inside the cap (answering round
1), and it is far enough out that the ceiling is structurally unreachable inside the wall-clock window —
the race is removed by arithmetic, not bet on (answering round 2) — and because it rides the arm's own
`env` line, it is not a global raise (the residue of round 1's objection that a global change perturbs
every arm's fail-fast).

**Deriving 50,000.** Inputs: the corrected E9 stub rates, measured against this suite's own `pgrep`
stub (474.9 / 652.7 / 648.6 iter/s), and the 0-2s worst-case window.

- **Default 4096 is too close.** 4096 / 652.7 ≈ 6.3s and 4096 / 474.9 ≈ 8.6s — a 6.3-8.6x margin
  against the nominal 1s window, but only **3.1-4.3x against the true 2s worst case** (losing the race
  needs just 4096 / 2s = 2,048 iter/s). That is the "6-9x margin holding on a CI host nobody has
  measured" that the ruling declines to bet on — and the honest worst-case number is smaller still.
- **50,000 removes the race, and iteration 32 measured a HARD physical bound under it (V25).** A
  ceiling C wins the race only if the runner sustains ≥ C/2 iter/s inside the window. C = 50,000
  requires **25,000 iter/s** — **38x** the fastest rate ever observed for this stub (652.7 iter/s) and
  **125x** the contended rate re-measured at this base (200 iter/s, V23). The bound is stronger than a
  rate extrapolation: the walk spawns **one process per iteration by construction**
  (`pgrep -P "$current"`), and on this machine `/usr/bin/true` — no bash, no script, the absolute
  physical floor — spawns at **205-250/s** (V25). 25,000 iter/s is therefore **~100x the bare-spawn
  ceiling of the hardware**, not merely 38x a stub benchmark. **This refutes `gpt5-6-sol`'s round-3
  objection** ("a sufficiently fast runner can exceed 25,000 iterations/s") by measurement rather than
  by argument: no machine that must fork a process per iteration reaches it.
- **50,000 keeps the backstop bounded — CORRECTED at iteration 32 (V23/V24), and this leg is weaker
  than first written.** With the wall clock dead (the T1 regression this arm exists to catch), the
  ceiling ends the walk in 50,000 / rate seconds. At the QUIET rates (474.9-652.7 iter/s) that is
  **77-105s**, inside the 600s widened cap with ≈ 5.7x headroom — the figure this doc originally
  carried. At the CONTENDED rates re-measured at this base (**181-200 iter/s**, V23, under load
  average 20.59 on 16 CPUs, V24) it is **250-276s**, i.e. only **≈ 2.2x** under the cap. The honest
  statement is therefore a range, not a point: **backstop headroom is 2.2x-5.7x depending on ambient
  load**, and the 5.7x figure is the quiet-machine end of it, not the operating point.
  **What degrades, and what does not.** On a runner more than ~2.2x slower than this already-contended
  measurement, T1 stops being a message-shaped red (`process-tree discovery exceeded 50000 nodes`) and
  becomes a **cap-shaped** red at 600s. It does **not** stop being a red: T1's verdict is unchanged,
  only its diagnosticity. That is the correct reading of round 1's objection, which was aimed at
  `PROBE_MAX_TREE_NODES=1000000` — a value at which the cap fires ALWAYS (1,532-2,106s at quiet rates,
  4,000-5,500s at contended ones), so the ceiling's own message was unreachable in every configuration
  rather than in a contended tail.
- **Feasible interval — RE-DERIVED at iteration 32 with the full observed rate range, and the result
  is that the doc's original interval was an artefact of using ONE rate for BOTH legs.** The two legs
  pull in opposite directions and each must take its own worst case: the race leg is worst at the
  FASTEST observed rate, the backstop leg at the SLOWEST. Race: a ≥ 30x margin at 652.7 iter/s needs
  C ≥ 30 x 2 x 652.7 ≈ **39,200**. Backstop: inside the 600s cap at 5x contention of the slowest
  observed rate (181 iter/s, V23) needs C ≤ 600 x 181 / 5 ≈ **21,700**. **At those two margins the
  interval is EMPTY** — the original `[≈39,000, ≈57,000]` was non-empty only because both legs were
  evaluated at quiet-machine rates. The original derivation is retained above as history; this is the
  correction.
- **Why 50,000 nonetheless stands, and what it costs.** The two margins are not symmetric in
  consequence. Violating the RACE margin produces a **spurious red on a clean tree** — a false
  accusation, the failure round 2 rejected and the one this arm exists to avoid. Violating the
  BACKSTOP margin produces a **correct red, less precisely diagnosed** (cap-shaped instead of
  message-shaped) in a configuration that is *already* a regression. Given an empty interval, the doc
  chooses to satisfy the leg whose violation is a wrong verdict and to accept, explicitly, a
  degradation in the leg whose violation is only a wrong explanation. At C = 50,000 the race margin is
  **38x** at the fastest observed rate and **125x** at the contended rate (V23), while the backstop is
  **2.2x-5.7x** under the cap (V24). **This is a stated trade, not a satisfied constraint**, and it is
  flagged as such for the round-4 reviewers rather than presented as an interval membership.

**What the arm still fail-fasts on:** a dead or mis-scoped wall clock (T1 — red on a ceiling-message
mismatch in ~77-105s at measured rates), a walk with BOTH bounds dead (the 600s cap, as before), and
the compound case of a wall-clock regression on a runner more than ~5.7x slower than this machine's
slowest measured rate (cap-shaped red — acceptable, because that configuration is already a
regression; a clean wall clock reds nothing).

**Failure-mode framing (unchanged from the iteration-31 correction, evaluator finding 2):** the arm's
verdict depends on the wall clock firing — T1 is the demonstration — and the arm asserts a message only
the wall-clock branch can emit, so if the ceiling somehow won anyway, the arm would RED loudly on a
clean tree rather than pass vacuously. D4 does not change that shape; it moves the ceiling far enough
out that the spurious red cannot happen at any measured or plausibly extrapolated rate, while keeping
it close enough in to remain a real backstop.

Measured (iteration 31), on the machine E7/E8/T1 all ran on — the controller's own pin worktree; no row
here establishes that it is the same machine as any other referenced in this doc (evaluator finding 4).
At the DEFAULT ceiling, the wall clock is what fires (E8: minimal fix only, no ceiling change → rc=0,
41 ok, 0 not ok, 50s), and the iteration-31 evaluator measured the D4 scoped override as free on the
happy path (41/41 ok, 47.1s, indistinguishable from baseline). The original
`PROBE_MAX_TREE_NODES=1000000` mechanism is **not needed for the arm to pass** and stays deleted; D4
re-introduces a ceiling override at 50,000, scoped, for the reasons derived above — not because the arm
needs it to pass, but so that its passing does not depend on an unmeasured runner losing a live race.

The 5x widened arm cap (test_motoko_connection_probe.sh:447-452) is **kept**. **The justification is the
CI incident the code comment already cites (run 33001432738, 2026-08-26), NOT the timings below**
(evaluator finding 3: neither run comes near even the un-widened 120s cap, so they bear on nothing).
For the record: E8 clean = 50s whole-suite, T1 mutant = 44s whole-suite. The widening exists for a
CONTENDED CI runner, which is
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
- **assert the target file exists BEFORE any counter runs** — `[[ -f "$probe" ]] || { echo "not ok -
  refusal-branch gate: \$probe does not resolve to a file; instrument failure, not a verdict" >&2; exit 1; }`
  — so that no `grep` in this gate can ever fall through to reading stdin,
- extend the anti-vacuity floor to require `actual_echo_refusals != 0`,
- `expected_refusal_branches` moves **24 → 27** (19 + 5 + 3).

The count moves because the gate now covers the previously-unguarded echo refusals, not because this
work adds a new refusal branch (it does not — it only re-words two existing messages).

The gate section also gains a **ceiling-locality guard** (new in this revision, in service of D4's
"scoped" claim): before the drift count, the suite refuses if `PROBE_MAX_TREE_NODES` is set in the
suite's own scope at gate time —

```bash
if [[ -n "${PROBE_MAX_TREE_NODES:-}" ]]; then
  echo "not ok - PROBE_MAX_TREE_NODES is set at suite scope; the ceiling override must stay on arm env lines" >&2
  exit 1
fi
```

A per-command `env`-line assignment (arm :449's D4 override, arm 40's `=3`, the invalid-value arm)
never persists into the suite's shell, so this is invariantly quiet on a correct tree. It fires on
exactly two leak shapes: a future edit that promotes the override to a file-global assignment or
`export`, and an ambient `PROBE_MAX_TREE_NODES` in the caller's environment. The second is deliberate
fail-loud hermeticity, not collateral: an ambient override silently re-parameterises every arm that
does not pin its own ceiling, which un-hermeticizes the suite. The guard lives inside the existing gate
flow (like the anti-vacuity floor), so the arm count stays 41. Without it, nothing in the suite would
distinguish "scoped to one arm" from "global" — the happy path is byte-identical either way, which is
why T4 (Testing Strategy) is required to prove this guard LOOKS.

## Design Freeze

- The three `descendant_pids` branches keep distinct messages; no two branches share a string.
- Arm :449 asserts `process-tree discovery deadline expired (wall clock)` and carries
  `PROBE_MAX_TREE_NODES=50000` on its own `env` line (§D4, applied per attended ruling D-MOTOKO-6N-1);
  every other arm keeps the default 4096 as its fail-fast backstop; the override is never exported or
  assigned at suite scope, and the gate's ceiling-locality guard refuses if it is; the 5x widened cap
  is kept.
- The refusal-branch gate counts echo-based process-tree refusals and `expected_refusal_branches=27`,
  and gains the ceiling-locality guard (see (f)).
- No change to the probe's production refusal *outcome* (exit 1, caller wrapper); only the diagnostic
  stderr strings at lines 182/187 change.
- No fix to #971; no runner-only speculative change; no widening into row 6o.

## Verification Log

Base commit: **a223e7274** (`git log --oneline -1` → `a223e7274 M2: pipeline.BuildCanonicalJSON + hidden internal-dump-iface (iteration 312) (#998)`). All rows measured in this worktree at that commit unless noted. Rows V1-V11 are re-derived here; E1-E8 and T1 are controller-measured, iteration 31, and cited as such.

**Rows V12-V21 are the iteration-32 revision's rows**, measured at **`48817dcdd`** (= `origin/dev` at
revision time) in the iteration-32 revision worktree. The Success Criteria are re-baselined on them.
Where a V12+ row repeats an earlier row's command, the V12+ observation is the one the criteria cite.

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
| V12 | Iteration-32 worktree is at `48817dcdd` and clean | `git log --oneline -1`; `git status --porcelain \| wc -l` | `48817dcdd docs(mission): file D-53 — the 4 UNCLASSIFIED docs from the D-51 inventory (#1006)`; `0` |
| V13 | Baseline suite is green at `48817dcdd` (re-baselines S1) | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0; `ok` lines 41; `not ok` lines 0; 54.1s wall; last line `PASS: 41 probe self-test arms ran` |
| V14 | Both files parse at `48817dcdd` | `bash -n tools/eval/motoko_connection_probe.sh`; `bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0; rc=0 |
| V15 | #971's mechanism still absent at `48817dcdd`, so decision (e) holds | `grep -c 'PROBE_TREE_DISCOVERY_SECS'` in each file; same-file control `grep -c 'PROBE_TIMEOUT_SECS'` | 0 occurrences in `tools/eval/motoko_connection_probe.sh` (grep rc=1) and 0 in `tools/eval/test_motoko_connection_probe.sh` (grep rc=1); control: 3 in the probe, 12 in the test |
| V16 | PR #971 still OPEN/MERGEABLE at revision time | `gh pr view 971 --json state,mergeable,headRefOid,updatedAt` | `OPEN`, `MERGEABLE`, head `8a384e81b7e9f0f6b4b4588b462fb35ef74b7bf4`, updated `2026-08-31T06:57:30Z` |
| V17 | Touched positions unchanged at `48817dcdd`: arm :449 block spans :447-:452 (save/widen/arm/restore), stub arm drive at :358, node-ceiling arm at :698-:701 with `PROBE_TIMEOUT_SECS=60 PROBE_MAX_TREE_NODES=3`, gate `expected_refusal_branches=24` at :707 | `grep -n '_arm_cap_saved\|ARM_CAP_SECS=\|real wall-clock deadline\|expected_refusal_branches\|node-count ceiling\|PROBE_TEST_DESCENDANT_FAILURE=1' tools/eval/test_motoko_connection_probe.sh`; `sed -n '440,475p;694,702p'` of the same file | :447 `_arm_cap_saved=$ARM_CAP_SECS`; :448 `ARM_CAP_SECS=$(( ARM_CAP_SECS * 5 ))`; :449 the wall-clock arm asserting `process-tree discovery failed`; :450 its env line `env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=1 PROBE_TEST_PGREP_LOOP=1`; :452 restore; :358 `run_live PROBE_TEST_DESCENDANT_FAILURE=1`; :698-:701 the node-ceiling arm; :707 `expected_refusal_branches=24` |
| V18 | No suite-scope assignment/export of the ceiling exists at base (the locality guard's precondition) | `grep -cE '^(export )?PROBE_MAX_TREE_NODES=' tools/eval/test_motoko_connection_probe.sh`; same-file anchored control `grep -c '^ARM_CAP_SECS='` | 0 occurrences in `tools/eval/test_motoko_connection_probe.sh` (grep rc=1); control 3 (rc=0; lines 9, 448, 452). Note: the first control attempted, `grep -c 'export '`, is itself 0 in this file (it contains no exports at all), so the anchored-assignment control is the sound same-scope positive |
| V19 | The D4 token is absent at base (S6's base value) | `grep -c 'PROBE_MAX_TREE_NODES=50000' tools/eval/test_motoko_connection_probe.sh`; same-file control `grep -c 'PROBE_MAX_TREE_NODES=3 '` | 0 occurrences in `tools/eval/test_motoko_connection_probe.sh` (grep rc=1); control 1 (rc=0, the node-ceiling arm's env line) |
| V20 | Gate components at `48817dcdd` re-derive V5/V6 | `grep -c 'echo "process-tree discovery'`, `grep -c 'instrument_failure "'`, `grep -cE '\|\| usage$'`, all on `tools/eval/motoko_connection_probe.sh` | 3; 19; 5 (all rc=0; 19 + 5 = 24, matching :707; + 3 echo refusals = the new 27) |
| V21 | The default-ceiling line exists verbatim at :126; the `-F` form is the sound instrument for it | `grep -Fc 'MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}' tools/eval/motoko_connection_probe.sh`; control `grep -c 'PROBE_MAX_TREE_NODES'` same file | `-F` count 1 (rc=0); control 2 (rc=0; lines 126, 128). Measured trap: the same pattern WITHOUT `-F` returns 0 (rc=1) on this rig's grep — the `${...:-...}` text misparses as regex — so S7 below mandates `grep -F` |

| V22 | The pgrep stub is BYTE-IDENTICAL between the rate-measurement commit `a223e7274` and this doc's base `48817dcdd`, so oc-glm-5-2's named mechanism ("the stub implementation may have changed") cannot be the cause of any rate difference | `git show a223e7274:tools/eval/test_motoko_connection_probe.sh > /tmp/old`; `sed -n '250,266p'` each side; `cmp -s`; `shasum -a 256`; whole-file control `cmp -s /tmp/old <file>` | stub region **IDENTICAL**, sha256 `abffefd3622943b01fb598688c2ffc32b6f6409b04097548873119cffba9cd7d` on BOTH sides; whole-file control: **SAME** (the file did not change at all between the two commits) |
| V23 | Stub rate RE-MEASURED at `48817dcdd` as oc-glm-5-2 asked — and it is **3.3-3.6x SLOWER** than the iteration-31 figures the 50,000 derivation rests on | extract the stub verbatim from `sed -n '255,261p' tools/eval/test_motoko_connection_probe.sh` (sha `3d487e95f2494ea3`), sanity-check it echoes its last arg under `PROBE_TEST_PGREP_LOOP=1` (`out=12345`), then 3 trials of N=2000 invocations timed with `date +%s` | **181 / 200 / 200 iter/s** (11s / 10s / 10s), against iteration 31's **474.9 / 652.7 / 648.6 iter/s**. Same machine, byte-identical stub (V22) — so the variable is neither the code nor the machine |
| V24 | The cause of V23's spread is **ambient load**, not the instrument: a finer clock agrees, and the rig is heavily contended | `uptime`; `sysctl -n hw.ncpu`; `ps -A -o %cpu,comm \| sort -rn \| head`; then the same spawn loop timed with python `time.perf_counter()` instead of `date +%s` | load averages **20.59 / 18.07 / 16.96** on **16** CPUs, with a 100%-CPU editor helper and five ~48% `bash` processes (sibling missions); `date`-timed `/usr/bin/true` **200-250 spawns/s** vs `perf_counter`-timed **205.2 / 205.8 / 219.2 spawns/s** — the two clocks AGREE, so `date +%s` granularity is refuted as the explanation |
| V25 | **gpt5-6-sol's round-3 objection is REFUTED by measurement**: no runner can sustain the 25,000 iter/s that a 50,000 ceiling would need to win the race | same spawn loop against `/usr/bin/true` — no bash, no script, the ABSOLUTE physical floor per iteration on this machine — plus the spawn-free control loop to prove the harness is not the bound | `/usr/bin/true` = **200-250 spawns/s** (`date`) / **205.2-219.2** (`perf_counter`); spawn-free control = **200,000 iter/s**, so the loop harness is ~1000x away from being the limiter. 25,000 iter/s is **~100x** the bare-spawn ceiling of this machine. The walk spawns one process per iteration BY CONSTRUCTION (`pgrep -P "$current"`), so this bound is structural, not incidental |
| V26 | `gemini-3-1-pro`'s round-4 premise is REFUTED: `$probe` cannot be unset, so the gate's `grep` cannot fall through to stdin | `grep -n 'probe='`; `grep -cE '^[[:space:]]*probe='`; `grep -c '"$probe"'`; `head -5 ... | grep -n 'set '`; same-file control `grep -cE '^[[:space:]]*live_bin='` | assigned once at `:5` as `probe=${PROBE_UNDER_TEST:-$script_dir/motoko_connection_probe.sh}` (a `:-` default, so never unset); `set -uo pipefail` at `:2`; **27** uses of `"$probe"` in the file, including the two sibling counters in this same gate; control = **1**. Hazard class still closed defensively by the `[[ -f "$probe" ]]` assertion added to Phase 3 |

## Solution Design

### Overview

The fix is small and confined to two shell files. In the probe, the two `descendant_pids` refusal
messages that currently share a string are made distinct, so the real wall-clock branch has a unique
observable. In the test, arm :449 asserts that unique observable and carries `PROBE_MAX_TREE_NODES=50000`
on its own `env` line (§D4, per the attended ruling), making the node ceiling structurally unreachable
inside the wall-clock window on that arm while every other arm keeps the default 4096; the 5x widened
cap is kept; and the refusal-branch gate is extended to count the echo-based refusals it previously
ignored and to refuse if the ceiling override leaks to suite scope.

### Architecture

- **Probe (`motoko_connection_probe.sh`)** — `descendant_pids` (lines 178-209) is the only function
  touched. Its three refusal branches emit three distinct messages. The caller (`sample_tree`, line
  217) is unchanged: it still collapses any `descendant_pids` failure to
  `instrument_failure "process-tree discovery failed"`. Production refusal *outcome* is unchanged.
- **Test (`test_motoko_connection_probe.sh`)** — arm :449 (lines 447-452) and the refusal-branch gate
  (lines 707-722) are the only regions touched. Arm :449's assertion changes AND its `env` line gains
  `PROBE_MAX_TREE_NODES=50000` (a per-command assignment scoped to that one probe invocation — not an
  export, not a suite-scope variable); the gate gains the echo counter, the extended anti-vacuity
  floor, the new expected total, and the ceiling-locality guard.

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
- Test lines 447-452 (arm :449 assertion + its `env` line) and lines 707-722 (refusal-branch gate +
  ceiling-locality guard). Positions re-verified at `48817dcdd` (V17).

**What must still work:** all 41 arms pass (V2); both files parse under `bash -n` (V3, V4); the node
ceiling arm (40) still asserts `process-tree discovery exceeded 3 nodes`; the stub arm (:357) still
asserts the caller wrapper; the run_lane fixture arm still passes; the gate arm still passes with the
new count.

**What deliberately changes:** the two probe stderr strings at lines 182/187 (cosmetic diagnostic
change; the refusal outcome is unchanged); arm :449's assertion and its `env` line (gains
`PROBE_MAX_TREE_NODES=50000`); the gate's count (24 → 27), anti-vacuity floor, and new
ceiling-locality guard.

**Shell constraints:** both files are shell; `bash -n` is the only syntax gate, and the CI shell is GNU
bash 3.2.57 on arm64-apple-darwin25. No bash-4+ constructs are introduced (no associative arrays, no
`${var^^}`, no `mapfile`). The added `grep -c` and arithmetic are bash-3.2-safe.

## Implementation Plan

**Phase 1 — probe messages (~15 min).** In `motoko_connection_probe.sh`, change line 182 to
`echo "process-tree discovery deadline expired (test stub)" >&2` and line 187 to
`echo "process-tree discovery deadline expired (wall clock)" >&2`. `bash -n` must stay rc=0.

**Phase 2 — arm :449 (~20 min).** In `test_motoko_connection_probe.sh`, in the arm :449 block
(lines 447-452, verified V17): change the `expect_failure` expectation string to `process-tree
discovery deadline expired (wall clock)`, and add exactly one token, **`PROBE_MAX_TREE_NODES=50000`**,
to the arm's `env` line at :450, after `PROBE_TIMEOUT_SECS=1`. The line becomes:

```bash
  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=1 PROBE_MAX_TREE_NODES=50000 PROBE_TEST_PGREP_LOOP=1 \
```

This is the D4 env change (attended ruling D-MOTOKO-6N-1): a per-command assignment scoped to this one
probe invocation — no `export`, no suite-scope assignment anywhere in the file. The
`_arm_cap_saved` / `ARM_CAP_SECS=$(( ARM_CAP_SECS * 5 ))` widening is KEPT (see decision (c); the 600s
cap is now also the T1 backstop's outer bound).

**Phase 3 — refusal-branch gate (~35 min).** In `test_motoko_connection_probe.sh`, add
`actual_echo_refusals=$(grep -c 'echo "process-tree discovery' "$probe")`, extend the anti-vacuity floor
to require it non-zero, set `expected_refusal_branches=27`, and add it to the total. Immediately before
the drift count, add the ceiling-locality guard from decision (f) (refuse if
`${PROBE_MAX_TREE_NODES:-}` is non-empty at suite scope).

**Phase 4 — baseline + mutation validation (~2-3 hours).** Run the baseline suite (must stay 41 ok), then
run the T1 (wall-clock) mutant, the T2 (node-ceiling-only) mutant, the T3 addition-shaped mutant, and the
T4 scoping mutant (see Testing Strategy), recording each mutant's red set AS RUN — the iteration-31 red
sets were measured under the old default-ceiling design and are predictions here, not results.

## Files to Modify/Create

**Modified:**
- `tools/eval/motoko_connection_probe.sh` — lines 182, 187 (distinct refusal messages).
- `tools/eval/test_motoko_connection_probe.sh` — arm :449 (assertion + `PROBE_MAX_TREE_NODES=50000` on
  its `env` line at :450); refusal-branch gate (echo counter, anti-vacuity,
  `expected_refusal_branches=27`, ceiling-locality guard).

**Created:** none.

## Success Criteria

Each criterion names a command and its measured result on the **unmodified** tree at **`48817dcdd`**
(re-baselined for this revision, per the rule that a criterion must be baselined on the tree that will
run it; a command that cannot run cleanly at base is broken). S6 is the one criterion whose value is
DESIGNED to move (0 → 1); every other criterion must hold its base value.

| # | Command | Result at base (`48817dcdd`) | Must hold after |
|---|---|---|---|
| S1 | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, 41 ok, 0 not ok, 54.1s, `PASS: 41 probe self-test arms ran` (V13) | rc=0, 41 ok, 0 not ok |
| S2 | `bash -n tools/eval/motoko_connection_probe.sh` | rc=0 (V14) | rc=0 |
| S3 | `bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0 (V14) | rc=0 |
| S4 | `grep -c 'echo "process-tree discovery' tools/eval/motoko_connection_probe.sh` | 3 occurrences in that file (V20) | 3 (the two re-worded messages keep the count) |
| S5 | `grep -c 'instrument_failure "'` + `grep -cE '\|\| usage$'` + S4's echo count, all on `tools/eval/motoko_connection_probe.sh` | 19 + 5 + 3 = 27 (V20) | 19 + 5 + 3 = 27, and the gate's `expected_refusal_branches` equals this sum |
| S6 | `grep -c 'PROBE_MAX_TREE_NODES=50000' tools/eval/test_motoko_connection_probe.sh` | 0 occurrences in that file (V19; grep rc=1; same-file control `PROBE_MAX_TREE_NODES=3 ` = 1) | **exactly 1**, and it is on arm :449's `env` line — more than 1 means the override spread to another arm; 0 means D4 was not applied |
| S7 | `grep -cE '^(export )?PROBE_MAX_TREE_NODES=' tools/eval/test_motoko_connection_probe.sh` | 0 occurrences in that file (V18; grep rc=1; same-file anchored control `^ARM_CAP_SECS=` = 3) | 0 — **this is the criterion that REDS if the scoping leaks to suite scope**; its runtime twin is the gate's ceiling-locality guard, exercised by T4 |
| S8 | `grep -Fc 'MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}' tools/eval/motoko_connection_probe.sh` (the `-F` is mandatory — V21 measured the non-`-F` form returning a false 0 on this rig's grep) | 1 occurrence in that file (V21) | 1 — the probe's default stays 4096, so every arm without its own override keeps it |

## Testing Strategy

Each row names the OBSERVABLE and confirms it is produced **by** the mechanism, not alongside it.

**Re-derivation notice (this revision):** iteration 31's T1/E7 measurements were taken under the OLD
design — arm :449 at the **default** ceiling. Under D4's scoped ceiling the mutants traverse different
code-path lengths (a dead wall clock now runs to 50,000 nodes, not 4,096), so those numbers **do not
transfer**. Every "expected" column below is a PREDICTION the executor must PRODUCE BY RUNNING on the
modified tree; the iteration-31 figures are cited only as the prior design's measurements.

| # | Mutant | Kills which mutation | Predicted observable (executor must produce by running) | Produced by |
|---|---|---|---|---|
| T1 | E2 mutant: neuter the in-loop wall clock (`if false && (( $(date +%s) > deadline ))`) | the vacuity this doc fixes | suite reds; arm :449 is the failing arm with `lacked expected message: process-tree discovery deadline expired (wall clock)`; the arm's stderr shows `process-tree discovery exceeded 50000 nodes`; the arm runs **~250-276s at the CONTENDED stub rates re-measured at this base (V23: 181-200 iter/s)**, and ~77-105s only at iteration-31's quiet rates (474.9-652.7 iter/s) — cap headroom is correspondingly **2.2x-5.7x**, not 5.7x. *(Corrected at round 4 on `oc-glm-5-2`'s objection: (c) had been corrected and this row had not, so the executor's primary reference for the load-bearing acceptance carried a premise the doc's own V23 had already refuted — understating runtime ~3x and overstating headroom ~2.5x. The executor must therefore allow for a multi-minute T1 and must NOT read a >105s run as a hang.)* (well under the 600s widened cap), so whole-suite time will be MATERIALLY LONGER than iteration 31's 44s — that figure was measured at the default ceiling and does not transfer | the wall-clock mechanism — the wall clock is dead, so its unique message cannot appear; the SCOPED ceiling (50,000) ends the walk with its own message |
| T2 | node-ceiling-only mutant: neuter `if (( visited > MAX_TREE_NODES ))` | the two arms being separable | arm :449 still passes (wall clock fires inside its 0-2s window; the scoped override is irrelevant when the ceiling is dead); arm 40 reds with `lacked expected message: process-tree discovery exceeded 3 nodes` — its walk now ends on its own wall clock (`PROBE_TIMEOUT_SECS=60`), so expect that arm to take ~60s; iteration 31's E7 shape (39 ok, 1 not ok, only arm 40) is the prediction, its timings are not | the wall-clock mechanism for :449 (alive); the node-ceiling mechanism for arm 40 (dead, so its unique message cannot appear) |
| T3 | addition-shaped mutant: add a new `instrument_failure "..."` refusal branch to the probe | the gate LOOKS (a removal proves a check FIRES; only an addition proves it LOOKS) | gate reds with `refusal-branch drift: probe has 28 refusal branches` | the gate mechanism — the count moved 27 → 28 and the gate refuses |
| T4 | scoping-locality mutant: DELETE `PROBE_MAX_TREE_NODES=50000` from arm :449's `env` line and ADD `export PROBE_MAX_TREE_NODES=50000` near the top of the test file — the token survives, only its SCOPE moves | the "scoped" claim itself — D4 without locality is just a global raise, which round 1 rejected | a NAMED red: the gate's ceiling-locality guard fires with `PROBE_MAX_TREE_NODES is set at suite scope; the ceiling override must stay on arm env lines`; statically, S7 flips 0 → 1 and S6's arm-line count drops to 0 | the locality-guard mechanism — and ONLY it: on the happy path a global export is behaviorally indistinguishable from the scoped form (arm 40's own `env`-line `=3` overrides any export; the wall clock still fires first on every arm), so without the guard this mutant passes the whole suite. A removal-shaped mutant (just deleting the override) proves nothing here on a green run for the same reason; the moved-scope shape is what proves the guard LOOKS |

T1 is the load-bearing acceptance: it is red at base (V8) and must be green after this work. T2 proves
the wall-clock arm no longer depends on the node ceiling. T3 proves the extended gate still catches new
refusal branches. T4 proves the D4 override is LOCAL — that the design's word "scoped" is enforced, not
decorative. All four red sets are to be produced by the executor on the modified tree.

## Quorum Verification Log

| Round | Artifact | Verdict | Reviewers present | Surface every objection landed on |
|---|---|---|---|---|
| 1 | `.ailang/state/mission-quorum/m-motoko-discovery-arm-discriminating-refusal-2026-09-01T00-35-20Z.json` | **BLOCKED** ($0.0806) | 3/3 external (`gpt5-6-sol`, `gemini-3-1-pro`, `oc-glm-5-2`) — **no absentees** | `PROBE_MAX_TREE_NODES=1000000`: makes one outcome likely rather than guaranteed; forces a 120s hang on regression; efficacy shown only where the race is absent |
| 2 | `.ailang/state/mission-quorum/m-motoko-discovery-arm-discriminating-refusal-2026-09-01T00-43-50Z.json` | **BLOCKED** ($0.0781) | 3/3 external — **no absentees** | the same surface from the other side: removing the override leaves the wall-clock-versus-ceiling race in place, so a fast CI runner could hit the ceiling and the arm would spuriously RED on a clean tree |
| 3 | `.ailang/state/mission-quorum/m-motoko-discovery-arm-discriminating-refusal-2026-09-01T15-31-29Z.json` | **BLOCKED** ($0.1403) | 3/3 external — **no absentees** (`.synthesis.absent_reviewers` = `[]`, cross-checked `[.reviewers[]|select(.present==false)]` = `[]`) | **`gemini-3-1-pro` FLIPPED TO PASS — the first pass in three rounds.** The two rejects again localise on the ceiling surface, and BOTH are *premise* objections rather than design objections: `gpt5-6-sol` — "a sufficiently fast runner can exceed 25,000 iter/s, so the arm is still race-dependent"; `oc-glm-5-2` — "the rates feeding the 50,000 derivation were measured at `a223e7274`, the doc is baselined at `48817dcdd`, and no row re-measures them; the stub at :254-262 may have changed" |
| 3b | controller measurement pass (no reviewer spend) | **OBJECTIONS MEASURED, NOT FORWARDED** | — | Per the rule that a reviewer's objection is a claim too, both round-3 objections were RUN rather than routed to a designer. `gpt5-6-sol`'s is **REFUTED** (V25: 25,000 iter/s is ~100x this machine's bare `/usr/bin/true` spawn ceiling, and the walk forks one process per iteration by construction). `oc-glm-5-2`'s named mechanism is **REFUTED** (V22: the test file is byte-identical between the two commits — the stub cannot have changed) but its underlying ask is **SATISFIED AND PARTLY UPHELD** (V23/V24: re-measured at this base the rate is 181-200 iter/s, 3.3-3.6x slower, from ambient load alone — which leaves the race leg stronger and the backstop leg weaker than the doc claimed, and makes the doc's feasible interval EMPTY at its own stated margins). The derivation in (c) is corrected accordingly and the residual is stated as a trade, not hidden. |
| 4 | `.ailang/state/mission-quorum/m-motoko-discovery-arm-discriminating-refusal-2026-09-01T15-41-57Z.json` | **BLOCKED** ($0.0776) | **2/3 external — `gpt5-6-sol` ABSENT (budget), so this round is N-1** (recorded, not silently passed; the degradation cannot manufacture a false pass here because the verdict is BLOCKED either way) | **THE SURFACE MOVED.** For the first time in four rounds NEITHER objection touches the ceiling, the race, the value or the design. Both are internal-consistency defects introduced by the round-3/4 controller revision: `gemini-3-1-pro` — Phase 3's `grep -c ... "$probe"` rests on an unverified premise and could read stdin unbounded; `oc-glm-5-2` — (c) was corrected with V23's rates but the **T1 row was not**, so the load-bearing acceptance still quoted the superseded 77-105s / 5.7x figures |
| 4b | narrow-refinement carve-out APPLIED (no reviewer spend, no designer run) | **FIXES APPLIED VERBATIM → route to planner** | — | Both round-4 objections (a) carry a concrete reviewer-authored fix and (b) dispute completeness, not design DIRECTION — the carve-out's two conditions. **`oc-glm-5-2`'s fix applied verbatim**: the T1 row now carries the contended figures (~250-276s, 2.2x-5.7x) and an explicit instruction not to read a >105s run as a hang. **`gemini-3-1-pro`'s premise is REFUTED by measurement** (V26) — `probe` is assigned at `test_motoko_connection_probe.sh:5` with a `:-` default, the file runs under `set -u` (line 2), and `"$probe"` is already used **27** times including by the two sibling counters in this very gate — so it cannot be unset and `set -u` would abort rather than hang. The named HAZARD is nonetheless real and cheap to close, so the fix is applied in substance rather than argued away: Phase 3 now asserts `[[ -f "$probe" ]]` before any counter runs. Using a literal path instead, as the objection proposed, was rejected as it would make the new counter inconsistent with the two beside it. |

Per the mission-control rule on repeated blocks, the surface each round's objections landed on was
tracked rather than the round count. Both rounds localise on ONE surface (the race), and **no
reviewer flipped to pass in either round** — so the disposition is not SPLIT. The one revision and
the one re-quorum the protocol allows are spent, and no remaining objection carries a concrete
reviewer-authored fix that does not simply re-propose the mechanism the previous round rejected, so
the narrow-refinement carve-out does not apply. **Disposition: PARK `needs-human-review`.** This is a
JUDGMENT park, not a capacity park: nothing unblocks it on a clock.

**Round-3 disposition (iteration 32).** Round 3 BLOCKED, so the doc is NOT force-passed. What changed
is that both surviving objections are *premise* objections, and the controller measured them instead of
buying another revision round (V22-V25). One is refuted outright; the other's mechanism is refuted while
its ask is satisfied by re-measurement — and the re-measurement partly upholds it, which is recorded in
(c) rather than argued away. **The design DIRECTION is now unobjected**: `gemini-3-1-pro` passes, and
neither reject proposes a different design — both dispute the evidence behind one constant. Under the
narrow-refinement rule this is the controller's to fix verbatim; the doc nonetheless goes to a **fourth
round** rather than straight to planning, because the corrected derivation contains a controller-authored
judgement (which leg to satisfy when the interval is empty) that ought to be checked by someone other
than its author.

**Superseded 2026-09-01 (retained as history):** the park was resolved by the attended human ruling
D-MOTOKO-6N-1 (commit `878e0a5a0`), which chose option (B) — hold for D4, scope the override to arm
:449's `env` line, remove the race structurally, and spend one more iteration and a third quorum round
"as the right price for ending the argument instead of deferring it to a red CI leg". The third round
is authorized by that ruling, not by the controller re-opening a spent revision budget.

### E9 — the controller's refutation of the reviewers' shared premise, AND ITS CORRECTION

All three reviewers rest on one empirical premise: that a CI runner might process 4096
`descendant_pids` iterations inside the 1-second wall-clock window, so the node ceiling would win and
the arm would spuriously red.

**E9 as first measured (WRONG INSTRUMENT — recorded because the corrected number is what a human
must read).** The controller benchmarked the **system** `pgrep` at ~79 iterations/second (3 trials:
79.1 / 78.8 / 79.0; spawn-free control loop 11,433 iter/s), giving 4096 iterations ≈ 52s and a
claimed ~52x margin.

**E9 CORRECTED (iteration 31 evaluator, reproduced first-party by the controller).** Arm :449 sets
`PATH="$live_bin"`, and this suite installs its **own** `pgrep` stub at
`test_motoko_connection_probe.sh:254-262` — a bash script that echoes its last argument. The walk
therefore never calls system `pgrep`. Benchmarked against the suite's actual stub: **474.9 / 652.7 /
648.6 iter/s**, i.e. 4096 iterations ≈ **6.3-8.6 seconds** against a 1-second window — a **~6-9x
margin, not ~52x**. The wrong instrument, re-run for the side-by-side, reads 92.1 iter/s / 44.5s;
negative control confirms the stub is what resolves first on the arm's PATH. The evaluator's own
independent instrument (isolating arm 33) measured ~455 iter/s / ~9s, which agrees.

**What survives and what does not.** The *conclusion* survives and never rested on E9: E8 and T1
measured the arm's behaviour DIRECTLY rather than inferring it from a rate. What does not survive is
the SIZE of the margin offered as grounds for reconsidering three unanimous rejections — it is ~6x
smaller than stated. A ~6-9x local margin on this hardware is materially weaker evidence about a
different, historically contention-prone CI host than a ~52x one, and the human decision should be
taken on the corrected number. It was: the 2026-09-01 attended ruling cites exactly this corrected
margin ("a 6-9x margin holding on a CI host nobody has measured") as the thing not to bet on.

### D4 — a design the doc, its author and all three reviewers missed (evaluator, iteration 31) — **APPLIED**

Scoping `PROBE_MAX_TREE_NODES` to arm :449's own `env` line — rather than raising it globally or
deleting the override entirely — makes the ceiling structurally unreachable inside the wall-clock
window while leaving every other arm at the default. The evaluator measured it as free on the happy
path (41/41 ok, 47.1s, indistinguishable from baseline). A middle value keeps a bounded fail-fast
backstop rather than either the ~6-9s the default gives or the ~600s the old widened cap gave. This
is the synthesis both quorum rounds were circling and neither reached. **The synthesis is the
iteration-31 evaluator's.** It was recorded here rather than applied, because the revision budget was
spent and applying it unilaterally would have been a controller-invented resolution to a blocked
quorum.

**Status: APPLIED in this revision (iteration 32), by attended human ruling D-MOTOKO-6N-1 — Mark
Edmondson, 2026-09-01, commit `878e0a5a0` (author `mark@aitanalabs.com`, provenance verified by the
controller as an attended identity, not the fleet bot). The ruling, verbatim:**

> **(B) HOLD for D4. Scope `PROBE_MAX_TREE_NODES` to that one arm's env line and remove the race
> structurally rather than betting on a 6-9x margin holding on a CI host nobody has measured. The
> reviewers blocked twice for the same reason and D4 is what all three were circling; one more
> iteration and a third quorum round is the right price for ending the argument instead of deferring
> it to a red CI leg. Take it as a normal queue pick under row 6p, as the loop proposed.**

The concrete value (50,000) and its derivation are in decision (c); the locality enforcement (gate
guard, S6/S7, T4) is in decisions (c) and (f).

## Deferred Decisions

- **Whether the CI runner's iteration rate is anywhere near the D4 bound.** Under D4 this entry changes
  character: it is no longer "whether the node ceiling ever wins the race" — on arm :449 the ceiling is
  structurally unreachable inside the wall-clock window at any measured or plausibly extrapolated rate
  (it would take ≥ 25,000 stub iterations/second, ~38x the fastest rate measured; decision (c)) — but
  the runner's ACTUAL rate has still never been measured, and only CI can measure it (decision (d)).
  What remains deferred is the empirical confirmation, not the design bet. If the invariant somehow
  failed anyway, the arm REDS loudly with `process-tree discovery exceeded 50000 nodes` in its stderr —
  the designed failure shape, diagnosable on sight; the measurement that settles it is the wall-clock
  arm's stderr on the "launchd drivers (bash 3.2)" leg of the first PR carrying this work.
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

One iteration (~3-4 hours): Phase 1 (~15 min) → Phase 2 (~20 min) → Phase 3 (~35 min) → Phase 4 baseline
+ mutation validation (~2-2.5 hours) → doc/verification polish (~1 hour). Four edit sites: probe lines
182/187, arm :449 (assertion + one env token), gate (counter + locality guard). Phase 4 is longer than
iteration 31's runs because T1's dead-wall-clock walk now runs to the scoped ceiling (~250-276s contended, ~77-105s quiet — V23; at
measured rates, per arm) and T4 is new. If mutation validation exceeds the budget, the echo-count gate
extension is the first cut (per Deferred Decisions) — the ceiling-locality guard is NOT cuttable, since
it is what makes the ruling's word "scoped" enforced rather than asserted.
