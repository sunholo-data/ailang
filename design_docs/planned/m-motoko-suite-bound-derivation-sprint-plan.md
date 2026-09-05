# Sprint plan: M-MOTOKO-SUITE-BOUND-DERIVATION

**Design:** `design_docs/planned/m-motoko-suite-bound-derivation.md`

**Execution base verified:** detached HEAD `5f425d35568cc8b2f04c849ee34635dce8ec7f30`; the only difference from design base `087fbea631a0b80556baa034b499fbdae33e76d2` is the committed design document. The suite is byte-identical at both commits (`sha256 c46f07d50457653d6453b76cf6501127c914320275e7f050c42db511beaae417`).

**Implementation scope:** `tools/eval/test_motoko_connection_probe.sh` only. `tools/eval/motoko_connection_probe.sh` is a read-only test dependency and must not be edited.

**Milestones:** 3, each independently green and controller-committable.

**Risk:** high: this is a timing-sensitive Bash 3.2 suite and the two last quorum fixes are not integrated consistently in the design.

## Non-negotiable execution protocol

- The executor runs **no git write operations**: no `git add`, `git commit`, `git stash`, `git checkout`, `git switch`, `git branch`, `git push`, or reset/clean operation. Commits are the **CONTROLLER'S job**, exactly one commit after each milestone.
- At every milestone boundary the executor runs the syntax check, the bounded full suite, the milestone-specific checks, then snapshots the full post-milestone implementation file. It then stops for the controller to inspect, commit, and run the suite again.
- Snapshots are cumulative representations of every sprint-owned implementation file modified so far. Since the implementation scope contains one file, each boundary must contain `.snap/M<k>/tools/eval/test_motoko_connection_probe.sh`. Do not recursively snapshot `.snap/` itself. Preserve mode with `cp -p`.
- New self-test arms go **after** the last existing containment arm (`expect_success "REAL_LSOF containment accepts..."`, current lines 817-818) and **before** the suite-scope `PROBE_MAX_TREE_NODES` guard (current line 823). This is after all existing wall-clock-bounded work, including the SIGKILL-escalation arm at current lines 798-799. No new arm may be inserted ahead of those arms.
- Any socket/network observation made in the sandbox is **UNINFORMATIVE UNDER SANDBOX**. The fixture-backed assertions and all non-socket timing/derivation assertions remain usable; never report the live socket sub-arm as a network result.
- M1 has a controller/CI hold point. After the M1 commit, the controller must read the `launchd drivers (bash 3.2)` log. M2 must not start until its measured `r`, floor state, `r_real`, `p_obs`, and bookend line are recorded. If `r < 100/s`, the controller must decide whether to change `SCALE_MAX`/the 15-minute job budget before authorizing M2.

## Verified current state

| Fact | Command actually run at planning time | Observation |
|---|---|---|
| Base suite | bounded artifact-poll loop around `/bin/bash tools/eval/test_motoko_connection_probe.sh` | `rc=0`; `46` `ok` lines; `0` `not ok` lines with `PASS: 46 probe self-test arms ran` as the same-block positive control; `59` wall seconds; one socket line labelled `UNINFORMATIVE UNDER SANDBOX` |
| Shell floor | `/bin/bash --version` | `GNU bash, version 3.2.57(1)-release (arm64-apple-darwin25)` |
| Syntax | `/bin/bash -n` on the suite and production probe | rc 0 for both |
| CI reachability | `rg -n 'make test-launchd-drivers' .github/workflows/*.yml`; inspect `.github/workflows/ci.yml:583-602` and `make/test.mk:59-74` | exactly one workflow call, from job `launchd drivers (bash 3.2)`; `runs-on: macos-latest`; `timeout-minutes: 15`; target invokes this suite and `/bin/bash -n` |
| No direct workflow invocation | `rg -l --fixed-strings 'test_motoko_connection_probe.sh' .github/workflows \| wc -l`, with the `make test-launchd-drivers` hit above as positive control | `0`: the single CI path is indirect through `make/test.mk` |
| Current literals | `rg -c 'PROBE_TIMEOUT_SECS=[0-9]+' ...`; `rg -c 'PROBE_MAX_TREE_NODES=[0-9]+' ...` | `9`; `2` |
| Helper absent | `rg -c 'derive_bound\|measure_fork_rate\|bound_secs\|classify_drift' ... || true`, with `rg -c run_bounded ...` in the same block | no helper matches; positive control `run_bounded=10` |
| Production probe unchanged from design base | SHA-256 and `git diff 087fbea...HEAD -- tools/eval/motoko_connection_probe.sh` | hash `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99`; empty diff |

The measured base for all milestone comparisons is therefore **rc 0 / 46 ok / 59 wall seconds**, not the design log's exact 57-second observation.

## Design corrections the executor must follow

These are corrections, not optional interpretations:

1. **M1 AC-8 changes the M1 arm total.** The M1 body still says “eight self-arms” and AC-1 says `46 + 8`. AC-8 adds three distinct helper-failure arms. Correct M1 total: **57** (`46 + 8 + 3`).
2. **M1 must integrate the real-op measurement.** Section 4.8.2 says the first runner `r_real`/`p_obs` arrives in M1, while the M1 body omits it and its AC-1 regex uses the old diagnostic. M1 will measure and publish both rates; M2 will turn on the threshold refusal and add its two-sided arm. The canonical diagnostic from M1 onward is:
   `# bound derivation: r=<n>/s r_real=<n>/s p_obs=<d.dd> reference=400/s scale=<1-4> arm_cap=<n>s node_ceiling=<n> floor=<state>`.
3. **The startup anchor cannot measure the current `live_bin/pgrep`.** Derivation is inserted after current line 179, but `live_bin/pgrep` is only written at current lines 294-303. Create the canonical pgrep-stub file once beside the stimulus before derivation, measure that file, and later copy that exact file to `$live_bin/pgrep` instead of maintaining a second heredoc. This makes “measured executable” and “executable used by the walk” byte-identical without reordering existing arms.
4. **AC-8 needs an injection hook that the body never specifies.** Add one validated, test-only enum such as `PROBE_SELFTEST_MEASUREMENT_FAILURE=exit1|nonexec|timer`; unset means normal. `exit1` rewrites the minimal stimulus to exit 1, `nonexec` removes its execute bit, and `timer` kills the helper's timer after launch. Each mode is used only by a bounded `PROBE_SELFTEST_DERIVATION_ONLY=1` recursion. The caller must propagate the helper stderr and exit non-zero before `derive_bounds`; no mode may print a derived-bound line.
5. **Arm placement in the conflict table is wrong.** “After 793” is before the current SIGKILL-escalation arm at 798-799 and before containment recursions at 813-818. Put every new arm after current line 821, so no new fork/exec work can push any existing wall-clock arm later.
6. **`p_obs` is not the design's `P_PROXY`.** Section 3 defines `P_PROXY` as a ratio of degradation factors across two load conditions. Section 4.8.2's one-moment rates can only yield a contemporaneous throughput ratio. Moreover, its chosen “real op” is the suite's Bash pgrep stub, not `/usr/bin/pgrep`, `date`, or real `lsof`; it measures the executable used by the hermetic walk but cannot observe heterogeneity among the real OS operations named in the objection. Implement the literal AC-6 contract as `p_obs = max(r, r_real) / min(r, r_real)`, rounded to hundredths with Bash integer arithmetic, and gate that diagnostic proxy at `4.70`; do not claim that it measures the two-condition degradation ratio. The forced controls `800:100 -> 8.00` and `400:100 -> 4.00` were re-run under Bash 3.2 and discriminate the `> 4.70` comparison. Closing the stronger semantic gap would require a load step or an independently justified baseline and is outside this approved one-file design.
7. **Several surrounding prose claims remain stale after 4.8.2.** The old 2.8 s/4.9% cost becomes about **4.2 s**, not the carve-out's stated 3.8 s: all three measurements (startup stimulus + startup canonical-pgrep + stimulus bookend) use the approximately 1.4 s `run_bounded` path. That is 7.4% of the design's 57-second base, or 7.1% of the measured 59-second base. Section 4.3's “P_PROXY ... by nothing in the code” and section 8's “budgeted, not bounded” no longer describe the literal M2 gate.
8. **`bound_secs` and the derived arm cap are ordered incorrectly in the prose.** Cleanup and the getconf/measurement `run_bounded` calls can execute before the insertion point after line 179. Define `bound_secs` near the top, before `cleanup_fixture_sleeps`, so `${BOUND_SCALE:-1}` is callable on every early/EXIT path. Initially set `ARM_CAP_SECS` to the base/explicit override; only reassign its default to the scaled value after `derive_bounds` has set `BOUND_SCALE`. Calling `bound_secs` at current line 9 would call a not-yet-defined function.

## Reusable bounded suite runner

Use this in a fresh Bash 3.2 shell at every boundary and for every full-suite variant. It backgrounds the suite only with a completion artifact and polls that artifact inside a `date +%s` deadline. `RUN_*` paths remain available for the checks that follow.

```bash
run_suite_bounded() {
  local label=$1 cap=$2 run_dir pid start now rc ok_count not_ok_count
  shift 2
  run_dir=$(mktemp -d "${TMPDIR:-/tmp}/motoko-${label}.XXXXXX") || return 1
  RUN_OUT=$run_dir/stdout
  RUN_ERR=$run_dir/stderr
  RUN_DONE=$run_dir/done
  start=$(date +%s)
  ( "$@" >"$RUN_OUT" 2>"$RUN_ERR"; printf '%s\n' "$?" >"$RUN_DONE" ) &
  pid=$!
  while [[ ! -f "$RUN_DONE" ]]; do
    now=$(date +%s)
    if (( now >= start + cap )); then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
      echo "RUN_RESULT label=$label timeout=${cap}s out=$RUN_OUT err=$RUN_ERR"
      return 124
    fi
    sleep 1
  done
  wait "$pid"
  rc=$(sed -n '1p' "$RUN_DONE")
  now=$(date +%s)
  ok_count=$(grep -c '^ok ' "$RUN_OUT" || true)
  not_ok_count=$({ grep -h '^not ok ' "$RUN_OUT" "$RUN_ERR" || true; } | wc -l | tr -d ' ')
  echo "RUN_RESULT label=$label rc=$rc ok=$ok_count not_ok=$not_ok_count wall=$((now-start))s out=$RUN_OUT err=$RUN_ERR"
  return "$rc"
}
```

For a green boundary, `not_ok=0` is valid only beside the known-positive `ok` count and final `PASS:` line from the same `RUN_OUT`.

---

## M1 — measure, derive, publish, bookend, and fail loudly

**Goal:** add all measurement/derivation instrumentation, including the late AC-8 failure paths and the published contemporaneous `p_obs`, while leaving every existing capacity bound at its current literal value and leaving the floor disabled by default.

**Expected size/time:** about 180-250 shell lines including 11 arms; one day. Expected full-suite time on this host is roughly 64-70 seconds, but arm count and verdict—not a narrow time band—are the boundary gate.

### Exact edits

All edits are in `tools/eval/test_motoko_connection_probe.sh`.

1. At current line 9 (`ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-120}`), introduce `ARM_CAP_BASE=120`, keep `ARM_CAP_SECS` at the unscaled base/explicit override, and retain the current positive-integer validation unchanged. Immediately after that validation and before `fixture_sleep_pids`, define `bound_secs() { printf '%s\n' "$(( $1 * ${BOUND_SCALE:-1} ))"; }`. M1 must have **zero capacity-bound** `bound_secs` consumers; the definition exists early so M2's pre-derivation/EXIT paths are valid.
2. Immediately after the containment-only early exit ending at current line 179 and before `report_arm_cap()`:
   - write and chmod the minimal `stimulus.sh`;
   - write a canonical pgrep-stub executable containing the exact current lines 294-303 behavior;
   - add Bash-3.2-compatible `measure_fork_rate`, with increment only after successful execution and distinct return statuses 71 (missing/non-executable), 72 (stimulus exit), and 73 (timer wait failure);
   - validate and apply `PROBE_SELFTEST_MEASUREMENT_FAILURE` before measurement;
   - add a caller that uses `run_bounded ... 10 -- measure_fork_rate 1 ...`, distinguishes rc 199 from other non-zero statuses, copies the helper diagnostic to stderr, and exits before derivation on failure;
   - honor `PROBE_SELFTEST_FORK_RATE` and `PROBE_SELFTEST_REAL_OP_RATE` using `${VAR+x}` so empty values reach validation. When the fork-rate override is set and the real-op override is absent, use the fork-rate override for both **only in this self-test path**, keeping existing forced derivation arms fast and deterministic. A normal invocation measures both executables independently;
   - validate both rates as positive decimal integers before division;
   - calculate `P_OBS_HUNDREDTHS` as the rounded symmetric throughput ratio and format `P_OBS` with two decimal places using integer arithmetic/`printf`; no `bc`, float arithmetic, `${var^^}`, associative arrays, `mapfile`, `$EPOCHREALTIME`, or `date +%N`;
   - add `FORK_RATE_REF=400`, `SCALE_MAX=4`, `NODE_CEILING_FACTOR=16`, `BOUND_FLOOR_ENFORCED=${PROBE_SELFTEST_BOUND_FLOOR_ENFORCED:-0}`, its exact 0/1 validation, `derive_bounds`, and `classify_drift`; `bound_secs` is already defined near the top;
   - make `derive_bounds` print the corrected diagnostic format from Design Correction 2 and implement the loud disabled-floor/enforced-floor branches;
   - exit immediately after the diagnostic when `PROBE_SELFTEST_DERIVATION_ONLY=1`.
3. At current pgrep-stub heredoc lines 294-303, remove the duplicated heredoc and `cp -p` the already measured canonical pgrep-stub into `$live_bin/pgrep`. Retain the final chmod list. Verify the installed file with `cmp -s`; a zero result is paired with `[[ -x ... ]]` as the positive control.
4. After the containment arms ending at current line 821, add the original eight M1 arms in this order: slowed stimulus; forced `abc`; forced `0`; forced empty; disabled floor at 99; enforced floor at 99 plus boundary success at 100; invalid flag 2; drift classifier normal/loud/invalid. Each recursive call is already bounded by `expect_failure`/`expect_success`; direct helper/classifier calls must go through `run_bounded` too.
5. Immediately after those eight, add AC-8's three separate recursion arms: stimulus exits 1, stimulus non-executable, and killed timer. Use a small assertion helper that requires non-zero rc, exactly one named refusal, and zero derived lines. The zero derived-line observation must be reported beside the positive refusal count from the same captured files.
6. Beside the current containment-only leak guard at lines 802-805 (which shifts downward after insertion), add guards rejecting leaked `PROBE_SELFTEST_DERIVATION_ONLY` and `PROBE_SELFTEST_MEASUREMENT_FAILURE` before normal arm execution. Do not move the existing containment arms earlier.
7. After the refusal-branch gate (current line 869) and before `arms == 0`, run the stimulus measurement a second time through `run_bounded`, honor the same fork-rate override, and call `classify_drift "$BOUND_SCALE" "$bookend_rate"`. Do not repeat the real-op measurement at bookend.

### Boundary commands and expected observation

```bash
/bin/bash -n tools/eval/test_motoko_connection_probe.sh
/bin/bash -n tools/eval/motoko_connection_probe.sh
run_suite_bounded M1 300 /bin/bash tools/eval/test_motoko_connection_probe.sh
[ "$(grep -Ec '^# bound derivation: r=[1-9][0-9]*/s r_real=[1-9][0-9]*/s p_obs=[0-9]+\.[0-9][0-9] reference=400/s scale=[1-4] arm_cap=(120|240|360|480)s node_ceiling=[1-9][0-9]* floor=DISABLED$' "$RUN_OUT")" -eq 1 ]
[ "$(grep -Ec '^# (bound drift: .* drift=none|BOUND_DRIFT_DURING_RUN: )' "$RUN_OUT")" -eq 1 ]
grep '^PASS: 57 probe self-test arms ran$' "$RUN_OUT"
m1_bound_consumers=$(rg -c '\$\(bound_secs ' tools/eval/test_motoko_connection_probe.sh || true)
m1_bound_helper=$(rg -c '^bound_secs\(\)' tools/eval/test_motoko_connection_probe.sh || true)
printf 'm1_bound_consumers=%s positive_helper=%s\n' "$m1_bound_consumers" "$m1_bound_helper"
[ "$m1_bound_consumers" -eq 0 ] && [ "$m1_bound_helper" -eq 1 ]
```

Expected: syntax rc 0; bounded suite `rc=0 ok=57 not_ok=0`; exactly one diagnostic; exactly one bookend line; exact `PASS: 57...`; the final `rg` shows only the function definition/comments/assertions and **no capacity-bound call site**. If the socket arm is unavailable, its observation remains explicitly `UNINFORMATIVE UNDER SANDBOX` and does not change 57.

Controller/CI hold after snapshot and commit: run the sole `launchd drivers (bash 3.2)` job, record `r`, `r_real`, `p_obs`, floor line/state, and bookend. Absence of the derivation and floor-state evidence blocks M2. A CI/network check attempted only in the sandbox is uninformative.

### M1 test plan

| # | Test/arm | Mutation killed | Mechanical mutation on a temporary suite copy |
|---|---|---|---|
| M1-T1 | Full-run diagnostic uniqueness | Remove publication | Delete the single `echo "# bound derivation:` statement; bounded copy must fail the exact-one diagnostic check. |
| M1-T2 | Full-run diagnostic uniqueness | Duplicate publication | Duplicate that echo line; bounded copy stays otherwise green but diagnostic count becomes 2. |
| M1-T3 | Full-run bookend uniqueness | Remove the second measurement | Delete only the bookend `run_bounded`/`classify_drift` block; drift-line count becomes 0 while the derivation line remains the positive control. |
| M1-T4 | `bound derivation responds to a slowed stimulus` | Contaminate ambient stimulus | Insert `sleep 0.05` before `exit 0` in the canonical minimal stimulus text; ambient and slowed rates collapse and the arm reds. |
| M1-T5 | Same slowed-stimulus arm | Constant counter | Replace `n=$((n + 1))` with `n=1`; both readings become constant and the arm reds. |
| M1-T6 | Same slowed-stimulus arm | Measure the wrong executable | Replace `"$stim"` in the loop with `/usr/bin/true`; slow and ambient fixtures become indistinguishable and the arm reds. |
| M1-T7 | Forced `FORK_RATE=abc` recursion | Add nonnumeric fallback | Insert `r=${r:-400}` or rewrite nonnumeric input to 400 before validation; the recursion unexpectedly succeeds. |
| M1-T8 | Forced `FORK_RATE=0` recursion | Add zero fallback | Insert `(( r < 1 )) && r=1` before validation; the expected `'0'` refusal disappears. |
| M1-T9 | Forced empty-rate recursion | Conflate unset and empty | Replace `${PROBE_SELFTEST_FORK_RATE+x}` with `-n`; empty no longer reaches the `'<empty>'` refusal. |
| M1-T10 | Disabled floor at 99 | Silent clamp | Delete only the `# BOUND_FLOOR_NOT_ENFORCED` echo or clamp before the branch; rc remains 0 but the exact loud line is absent. |
| M1-T11 | Enforced floor at 99 and boundary at 100 | Remove/reverse ceiling comparison | Delete `BOUND_SCALE > SCALE_MAX` handling or change `>` to `<`; 99 succeeds or 100 refuses, so the paired arm reds. |
| M1-T12 | Invalid flag 2 recursion | Widen accepted flag values | Change `^[01]$` to `^[0-2]$`; the recursion no longer emits `must be 0 or 1`. |
| M1-T13 | Drift classifier `(1,800)`, `(1,399)`, `(1,abc)` | Remove/reverse loud classification or turn annotation into verdict | Delete the loud branch, reverse `k_end > k_start`, or add `return 1` to the loud branch; one of the paired rc/text checks reds. |
| M1-T14 | Helper failure: stimulus exits 1 | Restore unconditional counting/`|| true` | Change the guarded invocation back to `"$stim" ... || true` followed by unconditional increment; the arm observes rc 0/derived output instead of refusal 72. |
| M1-T15 | Helper failure: non-executable stimulus | Remove executable guard | Delete `[ -f "$stim" ] && [ -x "$stim" ]`; the named status-71 message disappears (zero derived lines are checked beside the expected-message positive count). |
| M1-T16 | Helper failure: killed timer | Hide `wait` failure | Replace `wait "$timer"; trc=$?` with `wait "$timer" ... || true` and force `trc=0`; a shortened window publishes a bound, so the arm reds. |

### M1 snapshot and handback

```bash
mkdir -p .snap/M1/tools/eval
cp -p tools/eval/test_motoko_connection_probe.sh .snap/M1/tools/eval/test_motoko_connection_probe.sh
cmp -s tools/eval/test_motoko_connection_probe.sh .snap/M1/tools/eval/test_motoko_connection_probe.sh
```

Expected `cmp` rc 0; positive control: both files exist, are executable, and have the same non-zero byte count. Executor stops. Controller inspects, commits M1, reruns the bounded suite, and obtains the CI observation before authorizing M2.

---

## M2 — wire wall-clock bounds, enforce floor, and gate the observed throughput proxy

**Goal:** make all must-not-fire wall-clock bounds consume `BOUND_SCALE`, make the under-floor refusal the default, add literal/order gates, and add the literal two-sided `p_obs` refusal required by carve-out AC-6.

**Dependencies:** controller has accepted M1's local and CI evidence; if the CI runner measured below 100/s, a recorded controller decision has resolved the scale/CI-budget issue.

**Expected size/time:** about 60-100 shell lines including two new counted arms/gates; one day plus the bounded loaded comparison. Expected full-suite arm count: **59**.

### Exact edits

All edits remain in `tools/eval/test_motoko_connection_probe.sh`.

1. Flip only the literal default to `BOUND_FLOOR_ENFORCED=${PROBE_SELFTEST_BOUND_FLOOR_ENFORCED:-1}`. Update the existing disabled-floor arm to pass explicit `...ENFORCED=0`; update the enforced-floor arm so its 99 refusal has no flag override. This flip and the first consumer below are one milestone/controller commit.
2. Leave the pre-derivation line-9/base assignment in place. Immediately after `derive_bounds` returns successfully, reassign `ARM_CAP_SECS=${PROBE_SELFTEST_ARM_CAP_SECS:-$(bound_secs "$ARM_CAP_BASE")}`. This is the first scaled consumer; it executes only after `BOUND_SCALE` exists. Preserve the already-validated explicit override verbatim. The derivation-only early exit follows this reassignment.
3. Apply `$(bound_secs 5)` to both `cleanup_fixture_sleeps` deadlines (current lines 56 and 66), and to `run_bounded`'s terminate grace (current line 118). These functions can execute before derivation or from the EXIT trap, so `bound_secs` must retain `${BOUND_SCALE:-1}`.
4. Replace only the four must-not-fire lane literals at current lines 361, 409, 422, and 435 with `PROBE_TIMEOUT_SECS="$(bound_secs 4)"`. Leave literal pins 0, 2, 1, 1, and 60 unchanged.
5. Scale `cap_elapsed > 10` (current line 533), `run_lane_ready_cap_secs=5` (588), and the outer `+ 10` margin (589). Keep `cap_secs_fixture=2`, `run_lane_timeout_secs=2`, orphan cap 1, report-path cap 2, socket 5s caps, and getconf cap literal.
6. Immediately after computing `discovery_killer_lane_secs=$((ARM_CAP_SECS + 30))`, add an assertion that it is strictly greater than `ARM_CAP_SECS`; on violation print the exact lane/arm values and exit 1. Do not increment `arms` separately: this assertion belongs to the existing discovery arm.
7. Add `P_PROXY_MAX_HUNDREDTHS=470`. After validating/calculating both rates but **before** setting/printing a derived bound, refuse when `P_OBS_HUNDREDTHS > P_PROXY_MAX_HUNDREDTHS` with `instrument failure, not a verdict: observed proxy spread <d.dd> exceeds 4.70`. Equality is allowed. No derived line may be emitted on refusal.
8. In the post-wall-clock self-arm section, add one counted arm which runs two bounded derivation-only recursions: `800/100` must refuse with `8.00 exceeds 4.70` and zero derived lines; `400/100` must exit 0 and emit exactly one diagnostic containing `p_obs=4.00`.
9. After the existing refusal-branch gate and before the bookend, add one counted wall-clock literal census arm: numeric `PROBE_TIMEOUT_SECS` literals must equal 5; `bound_secs` matches must be at least 8. Code review must enumerate 11 actual capacity-bound consumers (arm cap, two cleanup waits, run-bounded grace, four lane caps, elapsed ceiling, readiness cap, outer grace); do not assert that the raw grep count is 11 because the census command itself contains the token. If either counter is zero, print `instrument failure, not a verdict`; pair every zero refusal with the positive other counter. Then `pass_arm` once.

### Boundary commands and expected observation

```bash
/bin/bash -n tools/eval/test_motoko_connection_probe.sh
run_suite_bounded M2-base 900 /bin/bash tools/eval/test_motoko_connection_probe.sh
grep '^PASS: 59 probe self-test arms ran$' "$RUN_OUT"
rg -c 'PROBE_TIMEOUT_SECS=[0-9]+' tools/eval/test_motoko_connection_probe.sh
rg -c 'bound_secs ' tools/eval/test_motoko_connection_probe.sh
run_suite_bounded M2-k2 900 env PROBE_SELFTEST_FORK_RATE=200 /bin/bash tools/eval/test_motoko_connection_probe.sh
grep -E '^# bound derivation: r=200/s r_real=200/s p_obs=1\.00 reference=400/s scale=2 arm_cap=240s node_ceiling=3200 floor=enforced$' "$RUN_OUT"
run_suite_bounded M2-floor-high 30 env PROBE_SELFTEST_FORK_RATE=99 PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash tools/eval/test_motoko_connection_probe.sh
run_suite_bounded M2-proxy-high 30 env PROBE_SELFTEST_FORK_RATE=800 PROBE_SELFTEST_REAL_OP_RATE=100 PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash tools/eval/test_motoko_connection_probe.sh
run_suite_bounded M2-proxy-low 30 env PROBE_SELFTEST_FORK_RATE=400 PROBE_SELFTEST_REAL_OP_RATE=100 PROBE_SELFTEST_DERIVATION_ONLY=1 /bin/bash tools/eval/test_motoko_connection_probe.sh
```

Expected: base and k2 suite runs `rc=0 ok=59 not_ok=0`, exact PASS line; numeric timeout count `5`; raw `bound_secs` count at least 8 and code review finds the 11 consumers enumerated above; k2 diagnostic exactly as shown and fixed 1s/2s must-fire arms still say 1s/2s. `M2-floor-high` exits 1 with the floor refusal and no disabled-floor line. `M2-proxy-high` exits 1 with `8.00 exceeds 4.70` and zero derived lines beside one positive refusal line. `M2-proxy-low` exits 0 with exactly one `p_obs=4.00` diagnostic.

Run the design's loaded A/B evidence only after the ordinary boundary is green. The experiment must be a single bounded background job with a completion artifact and a TERM trap that kills every recorded spinner. Interleave ten derived/control pairs under 64 `yes` processes; cap the whole experiment at 2400 seconds. Record each rc, diagnostic, start/end `uptime`, and final spinner-survivor count. Success is all 20 runs completed, control (`FORK_RATE=800`, hence k=1) has at least one red, derived has zero reds, and spinner survivors are zero. If the control has zero reds, fewer than ten pairs complete, the deadline fires, or a p_obs instrument refusal occurs, report the evidence as **UNINFORMATIVE**, not a pass. Any live socket observation inside those suites is also uninformative under sandbox.

Use this exact bounded experiment shape (run from the repository root after reloading `run_suite_bounded` in the shell):

```bash
run_m2_load_evidence() {
evidence_dir=$(mktemp -d "${TMPDIR:-/tmp}/motoko-M2-load.XXXXXX") || exit 1
experiment_done=$evidence_dir/done
experiment_summary=$evidence_dir/summary
set -m
(
  experiment_rc=0
  spinner_file=$evidence_dir/spinners
  : >"$spinner_file"
  cleanup_load() {
    while IFS= read -r load_pid; do kill "$load_pid" 2>/dev/null || true; done <"$spinner_file"
    while IFS= read -r load_pid; do wait "$load_pid" 2>/dev/null || true; done <"$spinner_file"
    survivors=0
    while IFS= read -r load_pid; do kill -0 "$load_pid" 2>/dev/null && survivors=$((survivors + 1)); done <"$spinner_file"
    printf 'spinner_survivors=%s\n' "$survivors" >>"$experiment_summary"
    printf '%s\n' "$experiment_rc" >"$experiment_done"
  }
  trap cleanup_load EXIT
  trap 'experiment_rc=124; exit 124' TERM INT
  i=0
  while (( i < 64 )); do
    yes >/dev/null &
    printf '%s\n' "$!" >>"$spinner_file"
    i=$((i + 1))
  done
  derived_reds=0
  control_reds=0
  completed_pairs=0
  i=1
  while (( i <= 10 )); do
    uptime >>"$experiment_summary"
    /bin/bash tools/eval/test_motoko_connection_probe.sh >"$evidence_dir/derived.$i.out" 2>"$evidence_dir/derived.$i.err"
    derived_rc=$?
    (( derived_rc == 0 )) || derived_reds=$((derived_reds + 1))
    env PROBE_SELFTEST_FORK_RATE=800 /bin/bash tools/eval/test_motoko_connection_probe.sh >"$evidence_dir/control.$i.out" 2>"$evidence_dir/control.$i.err"
    control_rc=$?
    (( control_rc == 0 )) || control_reds=$((control_reds + 1))
    printf 'pair=%s derived_rc=%s control_rc=%s\n' "$i" "$derived_rc" "$control_rc" >>"$experiment_summary"
    grep '^# bound derivation:' "$evidence_dir/derived.$i.out" >>"$experiment_summary" || true
    grep '^# bound derivation:' "$evidence_dir/control.$i.out" >>"$experiment_summary" || true
    uptime >>"$experiment_summary"
    completed_pairs=$i
    i=$((i + 1))
  done
  printf 'completed_pairs=%s derived_reds=%s control_reds=%s\n' "$completed_pairs" "$derived_reds" "$control_reds" >>"$experiment_summary"
) &
experiment_pid=$!
experiment_group_safe=0
if jobs -p 2>/dev/null | grep -qx -- "$experiment_pid" && kill -0 "-$experiment_pid" 2>/dev/null; then experiment_group_safe=1; fi
set +m
if (( experiment_group_safe == 0 )); then
  kill "$experiment_pid" 2>/dev/null || true
  wait "$experiment_pid" 2>/dev/null || true
  echo 'UNINFORMATIVE: load experiment lacked a distinct process group'
else
  experiment_start=$(date +%s)
  while [[ ! -f "$experiment_done" ]] && (( $(date +%s) < experiment_start + 2400 )); do sleep 2; done
  if [[ ! -f "$experiment_done" ]]; then
    kill -TERM "-$experiment_pid" 2>/dev/null || true
    term_start=$(date +%s)
    while kill -0 "-$experiment_pid" 2>/dev/null && (( $(date +%s) < term_start + 10 )); do sleep 1; done
    kill -9 "-$experiment_pid" 2>/dev/null || true
    wait "$experiment_pid" 2>/dev/null || true
    echo 'UNINFORMATIVE: load experiment exceeded 2400s'
  else
    wait "$experiment_pid" 2>/dev/null || true
    cat "$experiment_summary"
  fi
fi
}
run_m2_load_evidence
```

### M2 test plan

| # | Test/arm | Mutation killed | Mechanical mutation on a temporary suite copy |
|---|---|---|---|
| M2-T1 | Forced k=2 full suite | Leave a must-not-fire 4s lane literal | Revert any one of the four `$(bound_secs 4)` call sites to literal `4`; the literal census becomes 6 and reds. |
| M2-T2 | Existing 1s/2s must-fire refusal arms under k=2 | Scale a must-fire stimulus | Wrap current `PROBE_TIMEOUT_SECS=1` or `=2` in `bound_secs`; expected refusal text changes to 2s/4s and its existing arm reds. |
| M2-T3 | Default floor 99 refusal plus boundary 100 success | Leave floor disabled/remove ceiling | Restore default 0 or delete `BOUND_SCALE > SCALE_MAX`; no-override 99 succeeds instead of refusing, while 100 is the positive boundary control. |
| M2-T4 | Explicit disabled-floor recursion after flip | Delete diagnostic-only path | Remove the `...ENFORCED=0` branch or loud line; explicit 99 no longer exits 0 with exactly one loud line. |
| M2-T5 | Timeout literal census | Add a new capacity literal | Insert `: PROBE_TIMEOUT_SECS=4` after the arm section; syntax stays valid, count moves 5 to 6, census arm reds. |
| M2-T6 | Census anti-vacuity | Break the counter | Change its search token to `PROBE_TIMEOUT_SECS_ZZZ=`; zero is rejected beside positive `bound_secs` count rather than accepted as clean. |
| M2-T7 | Discovery ordering at k=4 | Hardcode lane deadline | Replace `ARM_CAP_SECS + 30` with literal 150 and run with forced rate 100; gate prints `lane deadline 150 is not above arm cap 480`. |
| M2-T8 | Bounded loaded A/B comparison | Remove all effective scaling | Make `bound_secs` echo its input unchanged; once the k1 control proves load sufficient, derived reds instead of remaining zero. Apply only to a temporary copy and retain the 2400s outer artifact deadline. |
| M2-T9 | Plain diagnostic plus high proxy refusal | Hollow/omit real-op observation | Set `r_real=$r` or delete `r_real`/`p_obs` fields; high forced case becomes 1.00 or the plain field-count check fails. |
| M2-T10 | Paired high `8.00` refusal and low `4.00` success | Widen/remove/reverse proxy gate | Change threshold 470 to 4700, delete the comparison, or reverse `>` to `<`; high stops refusing and/or low starts refusing, so the pair identifies the direction. |

### M2 snapshot and handback

```bash
mkdir -p .snap/M2/tools/eval
cp -p tools/eval/test_motoko_connection_probe.sh .snap/M2/tools/eval/test_motoko_connection_probe.sh
cmp -s tools/eval/test_motoko_connection_probe.sh .snap/M2/tools/eval/test_motoko_connection_probe.sh
if cmp -s .snap/M1/tools/eval/test_motoko_connection_probe.sh .snap/M2/tools/eval/test_motoko_connection_probe.sh; then
  echo 'not ok - M2 snapshot did not change'
  exit 1
fi
```

Expected: current file equals M2 snapshot; M1 and M2 snapshots differ, with both non-empty executable files as positive controls. Executor stops. Controller inspects, commits M2, and reruns the bounded suite.

---

## M3 — derive the discovery-arm node ceiling and preserve both scope gates

**Goal:** replace only the discovery wall-clock arm's hardcoded 50000 node limit with `NODE_CEILING`; keep the must-fire node=3 arm literal and production probe untouched.

**Expected size/time:** about 10-25 shell lines; less than one day. Expected full-suite arm count remains **59** because the M2 census arm is extended rather than duplicated.

### Exact edits

1. On the discovery arm's existing per-command env line (current line 519), replace only `PROBE_MAX_TREE_NODES=50000` with `PROBE_MAX_TREE_NODES="$NODE_CEILING"`. Keep it on that env line; never assign/export `PROBE_MAX_TREE_NODES` at suite scope. Keep `PROBE_TEST_PGREP_LOOP_DELAY=1` unchanged. Add `PROBE_TEST_MARKER="$tmp_dir/pgreploop.marker"` on that same arm and, immediately after the arm, count `^pgrep ` marker lines, refuse on zero or `>= 800`, and print `# discovery walk marker_count=<n> node_ceiling=<n>`. This makes the required delay-less manual control observable without persisting another arm.
2. Extend M2's literal census arm: numeric `PROBE_MAX_TREE_NODES` literals must equal exactly 1, the must-fire `PROBE_MAX_TREE_NODES=3` arm. Add a same-block positive control that total `PROBE_MAX_TREE_NODES=` references are at least 3 before accepting a zero/changed literal result.
3. Do not edit `tools/eval/motoko_connection_probe.sh`. Its expected SHA-256 remains `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99`, and the refusal-branch suite arm remains `(28)`.

### Boundary commands and expected observation

```bash
/bin/bash -n tools/eval/test_motoko_connection_probe.sh
run_suite_bounded M3-k2 900 env PROBE_SELFTEST_FORK_RATE=200 /bin/bash tools/eval/test_motoko_connection_probe.sh
grep '^PASS: 59 probe self-test arms ran$' "$RUN_OUT"
grep -E '^# bound derivation: r=200/s r_real=200/s p_obs=1\.00 reference=400/s scale=2 arm_cap=240s node_ceiling=3200 floor=enforced$' "$RUN_OUT"
rg -c 'PROBE_MAX_TREE_NODES=[0-9]+' tools/eval/test_motoko_connection_probe.sh
run_suite_bounded M3-scope 900 env PROBE_MAX_TREE_NODES=50000 /bin/bash tools/eval/test_motoko_connection_probe.sh
shasum -a 256 tools/eval/motoko_connection_probe.sh
git diff --quiet 087fbea631a0b80556baa034b499fbdae33e76d2 -- tools/eval/motoko_connection_probe.sh
```

Expected: k2 suite `rc=0 ok=59 not_ok=0`, diagnostic node ceiling 3200, discovery arm retains the wall-clock message, numeric node literal count `1`; suite-scope run exits 1 with `PROBE_MAX_TREE_NODES is set at suite scope`; production hash matches and read-only diff is empty. Pair the empty diff with the non-empty suite diff/hash as its positive control.

Verify the empty production diff and positive suite diff in one block:

```bash
if ! git diff --quiet 087fbea631a0b80556baa034b499fbdae33e76d2 -- tools/eval/motoko_connection_probe.sh; then
  echo 'not ok - production probe changed'
  exit 1
fi
if git diff --quiet 087fbea631a0b80556baa034b499fbdae33e76d2 -- tools/eval/test_motoko_connection_probe.sh; then
  echo 'not ok - positive control: suite has no implementation diff'
  exit 1
fi
```

For the delay-less leg-A check, run the exact temporary-copy pair below. It mutates no worktree file and preserves both result directories printed by `run_suite_bounded`:

```bash
mutation_dir=$(mktemp -d "${TMPDIR:-/tmp}/motoko-M3-mutation.XXXXXX") || exit 1
cp -p tools/eval/test_motoko_connection_probe.sh "$mutation_dir/delayless.sh"
perl -pi -e 's/PROBE_TEST_PGREP_LOOP_DELAY=1/PROBE_TEST_PGREP_LOOP_DELAY=0/' "$mutation_dir/delayless.sh"
run_suite_bounded M3-delayless 300 env PROBE_UNDER_TEST="$PWD/tools/eval/motoko_connection_probe.sh" PROBE_SELFTEST_FORK_RATE=200 PROBE_SELFTEST_REAL_OP_RATE=200 /bin/bash "$mutation_dir/delayless.sh"
grep -E '^# discovery walk marker_count=([1-9][0-9]?|[1-7][0-9][0-9]) node_ceiling=3200$' "$RUN_OUT"
cp -p "$mutation_dir/delayless.sh" "$mutation_dir/factor-one.sh"
perl -pi -e 's/NODE_CEILING_FACTOR=16/NODE_CEILING_FACTOR=1/' "$mutation_dir/factor-one.sh"
run_suite_bounded M3-factor-one 300 env PROBE_UNDER_TEST="$PWD/tools/eval/motoko_connection_probe.sh" PROBE_SELFTEST_FORK_RATE=200 PROBE_SELFTEST_REAL_OP_RATE=200 /bin/bash "$mutation_dir/factor-one.sh"
grep -F 'process-tree discovery exceeded 200 nodes' "$RUN_ERR"
```

Expected: delay-only control rc 0 with the marker count 1-799; factor-one mutant rc non-zero and the exact node-ceiling message, not a green wall-clock result. Do not change the worktree's delay.

### M3 test plan

| # | Test/arm | Mutation killed | Mechanical mutation on a temporary suite copy |
|---|---|---|---|
| M3-T1 | Forced k=2 full suite diagnostic/discovery arm | Keep hardcoded 50000 | Restore `PROBE_MAX_TREE_NODES=50000` on the discovery env line; node literal census moves 1 to 2 and reds. |
| M3-T2 | Delay-less discovery comparison | Shrink factor to 1 | In a temporary copy change the discovery delay 1 to 0 and `NODE_CEILING_FACTOR=16` to 1; forced 200 run emits `exceeded 200 nodes`. The control copy changes delay only and remains wall-clock-shaped with fewer than 800 markers. |
| M3-T3 | Suite-scope override gate | Promote derived ceiling to global environment | Add `export PROBE_MAX_TREE_NODES="$NODE_CEILING"` before arms; ordinary suite reaches the existing scope guard and reds. |
| M3-T4 | Node literal census/anti-vacuity | Add another literal or break grep | Add `: PROBE_MAX_TREE_NODES=50000` to move count to 2, or corrupt the census token to produce zero; both refuse, with total-reference count as positive control. |
| M3-T5 | Production-probe hash/diff and refusal-count arm | Touch production behavior/refusal inventory | Apply any one-line comment change to a temporary production-probe copy to prove hash discrimination; an actual production edit makes the base diff non-empty, while a refusal addition also moves `(28)`. No production edit is permitted in the worktree. |

### M3 snapshot and final handback

```bash
mkdir -p .snap/M3/tools/eval
cp -p tools/eval/test_motoko_connection_probe.sh .snap/M3/tools/eval/test_motoko_connection_probe.sh
cmp -s tools/eval/test_motoko_connection_probe.sh .snap/M3/tools/eval/test_motoko_connection_probe.sh
if cmp -s .snap/M2/tools/eval/test_motoko_connection_probe.sh .snap/M3/tools/eval/test_motoko_connection_probe.sh; then
  echo 'not ok - M3 snapshot did not change'
  exit 1
fi
```

Expected: current file equals M3 snapshot; M2 and M3 differ; M1/M2/M3 snapshot files all exist with full post-boundary content. Executor stops. Controller inspects, commits M3, and runs the final bounded suite at that commit.

## Final success conditions

- Three controller commits, one per milestone; zero executor git writes.
- `.snap/M1`, `.snap/M2`, and `.snap/M3` each contain the full corresponding version of the sole modified implementation file.
- Final suite: Bash 3.2 syntax green, rc 0, 59 ok arms, exact PASS line; any socket result still labelled uninformative under sandbox.
- M1 CI observation recorded before M2; M2 loaded evidence reported pass only with a red k1 control and zero derived reds, otherwise explicitly uninformative.
- Final numeric literals: 5 `PROBE_TIMEOUT_SECS=<digits>` and 1 `PROBE_MAX_TREE_NODES=<digits>`.
- `tools/eval/motoko_connection_probe.sh` remains byte-identical to the verified base; refusal count remains 28.
- The plan's `p_obs` gate is reported honestly as a contemporaneous two-class throughput diagnostic, not as a measurement of the two-condition degradation-factor ratio defined as `P_PROXY` in section 3 of the design.
