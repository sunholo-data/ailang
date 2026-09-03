# Sprint Plan — M-MOTOKO-GROUP-KILL-AND-LSOF-CONTAINMENT

**Design doc**: `design_docs/planned/m-motoko-group-kill-and-lsof-containment.md` (462 lines, quorum-cleared after two rounds)
**Mission**: motoko, iteration 34, queue row **6o**
**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-motoko-iter34-sprint`, branch `sprint/motoko-iter34-group-kill-lsof`, base `origin/dev` `55891002f`
**Milestones**: 4 · **Test rows**: 14 (T1-T14) · **Estimated**: ~3 h implementation + ~50 min of suite/drill wall-clock
**Risk**: low-medium (one file changed; the risk is a refactor regression in the 145-line hoist, and it has a named killer)

---

## 0. Executor contract — read this first

- **The executor performs NO git write operations.** No `git add`, no `git commit`, no `git stash`, no `git checkout`, no `git branch`, no `git reset`, no `git clean`, no `git push`. Read-only git (`git status`, `git diff`, `git show`) is fine and is encouraged for self-checking.
- **The controller builds the commits.** The controller reconstructs one commit per milestone from the worktree state and runs the acceptance gate at each milestone boundary. That is why **every milestone below must leave the suite GREEN** — a milestone that lands the tree red is a planning error, not an execution detail.
- **Exactly one file may be modified**: `tools/eval/test_motoko_connection_probe.sh`. Plus, in M4 only, `changelogs/v0.32-current.md`.
- **`tools/eval/motoko_connection_probe.sh` MUST NOT BE MODIFIED.** Its sha256 must still be
  `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99` when the sprint ends.
  Re-measured by the planner on this worktree at `55891002f`; the control (`test_motoko_connection_probe.sh` →
  `e1d56346b08cd14dd16f2b826268f90c8b36a84f6399d8c1b8d7a4037ce71b58`) proves the instrument discriminates.
- **All mutants live in `/tmp`.** The tree is never edited to make a mutant. Every probe mutant is driven with
  `PROBE_UNDER_TEST=<copy>`; every suite mutant is a `/tmp` copy of the suite run with
  `PROBE_UNDER_TEST="$PWD/tools/eval/motoko_connection_probe.sh"` (mandatory — a `/tmp` suite copy derives
  `script_dir` from `$0`, so its default `$probe` points at a file that does not exist).

---

## 1. Base state — measured by the planner, this worktree, `55891002f`, clean

| Fact | Command | Observed |
|---|---|---|
| **THE acceptance gate**, green at base | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, 43 `ok`, 0 `not ok`, last line `PASS: 43 probe self-test arms ran` |
| Both files parse | `bash -n` on each | rc=0, rc=0 |
| Probe hash (must not move) | `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e024…aabc99` |
| Suite hash (control — a different file, a different hash) | `shasum -a 256 tools/eval/test_motoko_connection_probe.sh` | `e1d56346…e71b58` |
| `run_bounded` definition | `grep -n '^run_bounded()'` | opens at **88**, closing `}` at **139**; `report_arm_cap()` opens at **141** |
| Arm 42 | `sed -n '740,741p'` | `expect_failure "descendant discovery stub refusal carries its own message" …` / `run_live PROBE_TEST_DESCENDANT_FAILURE=1` |
| Insertion gap | `sed -n '742,752p'` | 742 blank; 743 comment; 749-752 the suite-scope `PROBE_MAX_TREE_NODES` guard |
| Refusal-branch gate | `grep -n 'expected_refusal_branches='` | test:770 `=28` |
| New identifiers unallocated | `grep -c` for `getconf`/`/usr/bin/getconf PATH`/`run_lane_fixture_arm`/`no-probe` | suite: 1 (a comment) / 0 / 0 / 0 |
| Shell | `/bin/bash --version` | GNU bash **3.2.57(1)-release**, arm64-apple-darwin25 |
| Suite shell options | `sed -n '2p'` | `set -uo pipefail` — **no `-e`**, so `gate_rc=$?` after `run_bounded` is safe |

### Gates that are RED AT BASE and are therefore NOT acceptance criteria anywhere in this plan

`make test-launchd-drivers`, `/bin/bash tools/launchd/test_fmt_ab_schedule.sh` (rc=1,
`instrument failure: FMT_AB_TESTABLE_FUNCTIONS marker extraction … produced no text`), and the CI checks
`test` and `launchd drivers (bash 3.2)`. All three are **one inherited defect**: `c8c841e24` deleted the
`# BEGIN/END FMT_AB_TESTABLE_FUNCTIONS` markers from `tools/launchd/nightly-eval.sh`. V1 owns it; PR #1030 is
in flight. **Nothing in this plan depends on any of them going green.** A gate red at base measures the repo,
not the change. Every acceptance command below is scoped to `/bin/bash tools/eval/test_motoko_connection_probe.sh`,
to `bash -n`, to `shasum -a 256`, or to targeted `grep`/`sed` on the two probe files.

---

## 2. The relocation ruling — the RELOCATED gate wins

The design doc went through two quorum rounds. Round 2 relocated the `REAL_LSOF` containment gate. **Where the
doc still reads as though the gate sits after test:28, the RELOCATED version wins.** Concretely:

- **Doc line 175** (Solution Design → Overview, bullet 2) still says *"Insert a Darwin-only containment gate
  directly after the non-Darwin skip (test:28)"*. **This sentence is stale and must be ignored.**
- **Doc §(b) lines 209-249** and **doc line 299** (Files to Modify) carry the round-2 ruling and are authoritative:
  the gate goes **after `run_bounded`'s definition and before arm 1**, and it runs `/usr/bin/getconf PATH`
  **through `run_bounded`** with an explicit `PROBE_GETCONF_CAP_SECS` cap and **three distinct refusals**
  (timeout / non-zero exit / empty output).

**Exact insertion point, resolved by the planner:** `run_bounded`'s body ends with `}` at **test:139**; test:140
is blank; `report_arm_cap()` opens at **test:141**. The gate block and the marker early-exit go **between test:139
and test:141** — after the closing brace, before `report_arm_cap()`. That is the earliest point at which
`run_bounded` is callable, so the marker-exit inner run still executes only variable assignments and function
definitions (cheap, per the doc's V36).

Reason the relocation is necessary and not stylistic (doc V45): at test:28 `run_bounded` is not yet defined, so
the gate as originally written *could not have called it*, and a hung `getconf` would have hung the suite before
arm 1 with no deadline. Handling empty/non-zero output does not handle non-termination.

---

## 3. What the planner found wrong or inconsistent in the design doc

Reported rather than silently planned around. Four findings; the plan carries the corrections.

| # | Finding | Severity | Disposition in this plan |
|---|---|---|---|
| **F1** | **Doc line 175 still places the gate at test:28**, contradicting the round-2 relocation at §(b)/line 299. | medium (an executor reading top-down implements the unbounded version) | §2 above rules explicitly. All milestone tasks reference test:139/141, never test:28. |
| **F2** | **AC11 is arithmetically wrong for the relocated gate.** AC11 requires `grep -n '/usr/bin/getconf PATH' tools/eval/test_motoko_connection_probe.sh` to yield **1 hit**. The round-2 gate code contains that literal **4 times**: once as the `run_bounded` argument (doc:219) and three times inside the refusal messages (doc:222, 226, 231). AC11 was written against the pre-relocation gate and was not updated. | **high — an executor implementing the doc correctly would fail AC11** | AC11 is **restated** in §6 as AC11a/AC11b (see below). The 4-hit count is the correct post-change value. |
| **F3** | **T12's killer column names T13 and T14 — neither exists.** The doc's Test Plan table runs T1-T12 with no T13/T14, yet T12 says the empty-output and non-zero branches are "covered by T13/T14". The round-2 gate adds **three** refusal branches; only one (timeout) has a specified drill. Two of three new refusals ship unpinned. | medium | **T13 (non-zero exit) and T14 (empty output) are written out in full in §7** and are milestone-M3 acceptance. This discharges a promise the doc makes by name; it is not new scope. |
| **F4** | **`PROBE_GETCONF_CAP_SECS` has no validation, unlike every other cap knob in this suite.** `ARM_CAP_SECS` is validated at test:10-13 precisely because an invalid value poisons `run_bounded`'s arithmetic. Measured by the planner on this shell: `/bin/bash -c 'set -uo pipefail; f(){ local c=$1; local d=$(( $(date +%s) + c )); }; f abc'` → `/bin/bash: abc: unbound variable` and the shell **exits**. So `PROBE_GETCONF_CAP_SECS=abc` would abort the suite with a bare `unbound variable` and **no `not ok -` line** — the one failure shape this suite's conventions forbid. | medium | **Planner addition, doc-silent, labelled as such**: M3 adds a 4-line validation mirroring test:10-13. It adds **no arm**, so arm counts, AC3, AC4 and AC12 are all unchanged and the controller can drop it with zero re-planning. Pinning it with an arm would make the suite 47 arms and break three pinned ACs, so the pinning arm is a **follow-up queue row**, not this sprint. |

Two further notes that are *not* defects, recorded so nobody re-litigates them:
- AC11's phrase "above the first `pass_arm` call" still holds by line order: the first textual `pass_arm "` is at
  test:168 (inside `expect_failure`), which is below the test:139/141 insertion point.
- The doc's LOC figure (`+~85 / −~10`, prototype 874 vs 795 lines) was measured on the **pre-relocation** gate.
  The bounded gate is ~13 lines longer. Expect **+~100 / −~10**, final suite ~890-900 lines. Not a defect, but
  do not treat 874 as a target.

---

## 4. Milestones

Four milestones. Each is independently committable and each leaves the suite green. Arm counts move
43 → 43 → 44 → 46 → 46. The pre-existing refusal-branch-count arm renumbers 43 → 43 → 44 → 46 as new arms land
ahead of it; that is expected and is not a regression.

### M1 — Hoist the 6i fixture block into `run_lane_fixture_arm`, one call, behaviour-identical

**Goal**: pure refactor. 43 arms in, 43 arms out. Arm 36 unchanged in name, message, fixture, evidence line and
position. This exists so the kill arm in M2 is a *second call*, not a 145-line copy (doc D5).

**Edits** — `tools/eval/test_motoko_connection_probe.sh` only:
1. Wrap the body currently at **test:539-681** (the `skip_run_lane_fixture` guard at 536-538 STAYS where it is;
   the closing `fi` at 682 STAYS) into a function `run_lane_fixture_arm`.
2. Prologue:
   `local variant=$1 fixture_secs=$2 grace_allowance=$3 expected_refusal=$4 arm_name=$5; shift 5; local run_lane_extra_env; run_lane_extra_env=("$@")`.
   Do **not** `local`-ise `run_lane_*` state vars or `active_fixture_dir` — the nested
   `run_lane_fixture_harness` and the EXIT trap read them, and bash's dynamic scoping is what makes the
   `local` list above visible to the nested harness.
3. Apply exactly the seven substitutions the doc enumerates (doc table at lines 181-189):
   - `run_lane_fixture_secs=2861` → `run_lane_fixture_secs=$fixture_secs`
   - `run_lane_outer_cap_secs=$(( run_lane_timeout_secs + 10 ))` → `$(( run_lane_timeout_secs + grace_allowance + 10 ))`
   - `"$tmp_dir/run-lane.<x>"` → `"$tmp_dir/run-lane-$variant.<x>"`; `"$tmp_dir/run-lane-outer.<x>"` → `"$tmp_dir/run-lane-$variant-outer.<x>"`
   - `PROBE_STUB_STATE="$tmp_dir/lane-run-lane"` → `…/lane-run-lane-$variant"`
   - `env PATH="$live_bin" AILANG_BIN=ailang-stub \` → `env ${run_lane_extra_env[@]+"${run_lane_extra_env[@]}"} PATH="$live_bin" AILANG_BIN=ailang-stub \`
   - the `grep -Fq -- "INSTRUMENT FAILURE: lane treatment exceeded ${run_lane_timeout_secs}s sampling deadline"` → `grep -Fq -- "$expected_refusal"`
   - the literal arm name in the `not ok` echo and in `pass_arm` → `"$arm_name"`
   `run_lane_timeout_secs=2` and `run_lane_ready_cap_secs=5` stay literal inside the body.
4. Replace the hoisted block with one call, in arm 36's current position:
   ```
   run_lane_fixture_arm term 2861 0 "INSTRUMENT FAILURE: lane treatment exceeded 2s sampling deadline" \
     "production run_lane timeout kills wrapper grandchild"
   ```

**Gotcha to respect**: `${run_lane_extra_env[@]+"${run_lane_extra_env[@]}"}` is mandatory. bash 3.2 under
`set -u` treats a bare empty-array expansion as unbound, and M1's only call passes **no** extra env — so the
plain `"${arr[@]}"` form aborts the suite on the very first call.

**Acceptance (all must hold before M1 is considered done):**

| # | Command | Required |
|---|---|---|
| M1-G1 (**gate**) | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, 43 `ok`, 0 `not ok`, `PASS: 43 probe self-test arms ran` |
| M1-G2 | `grep -n '^ok 36' /tmp/s.out` | `ok 36 - production run_lane timeout kills wrapper grandchild` — same number, same name |
| M1-G3 | `grep 'run_lane evidence' /tmp/s.out` | exactly ONE line, containing `fixture-2861`, `survivors=0`, `outer_cap_fired=no`, `cleanup=0`, `markers=yes` |
| M1-G4 | `bash -n` on both files | rc=0, rc=0 |
| M1-G5 | `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e024…aabc99` |
| M1-G6 | `grep -c 'run_lane_fixture_arm'` / `grep -c 'run_lane_fixture_secs=2861'` / `grep -c 'run_lane_fixture_secs=\$fixture_secs'` | ≥ 2 (def + one call) / **0** / **1** |
| M1-G7 | **T2 drill** (§7) | rc=1, 35 `ok`, sole `not ok` names arm 36 with `survivors=1` — identical to base |
| M1-G8 | **T3 drill** (§7) | rc=1, 35 `ok`, sole `not ok` names arm 36 with `survivors=1` |

M1 is the milestone most likely to go wrong (the hoist rewrites 145 lines of indentation). M1-G3 and M1-G7 are
what make a subtle behaviour change visible; do not skip them to save 90 seconds.

---

### M2 — The SIGKILL-escalation arm: second call, behind arm 42

**Goal**: complete sub-item **(a)**. Suite goes 43 → 44 arms. New arm 43 is the first killer the probe's
`kill -9 "-$pid"` group escalation has ever had.

**Edits**:
1. Immediately **after test:741** (`run_live PROBE_TEST_DESCENDANT_FAILURE=1`, arm 42's continuation line) and
   **before the suite-scope `PROBE_MAX_TREE_NODES` guard comment at test:743**, insert:
   ```
   if (( skip_run_lane_fixture )); then
     echo "UNINFORMATIVE: run_lane SIGKILL-escalation arm requires real lsof for cwd survivor checks"
   else
     run_lane_fixture_arm kill 2863 5 "INSTRUMENT FAILURE: lane treatment exceeded its bounded termination deadline" \
       "production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild" PROBE_TEST_IGNORE_TERM=1
   fi
   ```
2. Add a one-line comment on the `PROBE_TEST_IGNORE_TERM` knob at the stub (test:293) noting it now has two
   callers (test:363 and this arm) — the doc's shared-machinery note.

**Why this position** — see §5. Do not move it earlier to "group it with arm 36".

**Acceptance:**

| # | Command | Required |
|---|---|---|
| M2-G1 (**gate**) | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, **44** `ok`, 0 `not ok`, `PASS: 44 probe self-test arms ran` |
| M2-G2 | `grep -n '^ok 4[34]' /tmp/s.out` | `ok 43 - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild`; `ok 44 - refusal-branch count still matches the set this suite covers (28)` |
| M2-G3 | `grep 'run_lane evidence' /tmp/s.out \| grep -o 'fixture-[0-9]*\|survivors=[0-9]*\|outer_cap_fired=[a-z]*' \| paste -sd' ' -` | `fixture-2861 outer_cap_fired=no … survivors=0 fixture-2863 outer_cap_fired=no … survivors=0` — in that order |
| M2-G4 | `grep -c 'timeout=yes' /tmp/s.out` | 2 (both fixtures reached their refusal; for 2863 `timeout=yes` now means the **bounded-termination** message, not the sampling-deadline one) |
| M2-G5 | **T1 drill** — the headline | rc=1, **42** `ok`, **sole** `not ok` = `production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild (outer_rc=0 survivors=1 cleanup=0 probe_rc=…)` |
| M2-G6 | **T2 drill** | rc=1, 35 `ok`, sole `not ok` still arm 36 — 6i's coverage intact, new arm masked by fail-fast |
| M2-G7 | **T8 drill** | rc=1, 43 `ok`, sole `not ok` = arm 43 with `survivors=0` |
| M2-G8 | `shasum -a 256 tools/eval/motoko_connection_probe.sh`; `bash -n` ×2 | hash unchanged; rc=0, rc=0 |

**Cost**: the arm spends the probe's real 5 s grace window. Suite time ~49 s → ~58-60 s. Its 17 s outer cap
(`2 + 5 + 10`) must NOT fire — `outer_cap_fired=no` in M2-G3 is the check.

---

### M3 — The bounded `REAL_LSOF` containment gate, the marker exit, and its two arms

**Goal**: complete sub-item **(b)**. Suite goes 44 → 46 arms.

**Edits**:
1. **Between test:139 (`run_bounded`'s closing `}`) and test:141 (`report_arm_cap()`)** insert the gate and the
   marker exit exactly as the doc's §(b) code block (doc lines 214-248) specifies: Darwin-only `if`;
   `gate_cap_secs=${PROBE_GETCONF_CAP_SECS:-5}`;
   `run_bounded "$tmp_dir/gate.out" "$tmp_dir/gate.err" "$gate_cap_secs" -- /usr/bin/getconf PATH`;
   `gate_rc=$?`; **three distinct refusals** —
   - `199` → `not ok - REAL_LSOF containment: /usr/bin/getconf PATH exceeded its ${gate_cap_secs}s cap; instrument failure, not a verdict`
   - non-zero → `not ok - REAL_LSOF containment: /usr/bin/getconf PATH exited ${gate_rc}; instrument failure, not a verdict`
   - empty `$(cat "$tmp_dir/gate.out")` → `not ok - REAL_LSOF containment: /usr/bin/getconf PATH produced no text; instrument failure, not a verdict`
   — then `IFS=: read -ra standard_path_entries`, the string-equality loop against `${REAL_LSOF%/*}`, and the
   containment refusal
   `not ok - REAL_LSOF resolved outside getconf PATH: $REAL_LSOF is not in any of $standard_path; an ambient lsof would serve as the survivor oracle`.
   Every refusal `exit 1`.
   Then, **outside** the Darwin `if`, the marker early-exit:
   `if [[ "${PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY:-0}" == 1 ]]; then echo "REAL_LSOF containment check passed: $REAL_LSOF"; exit 0; fi`.
   The marker exit being outside the `if` is load-bearing: a mutant that deletes the gate must still exit here
   rather than run 43 arms one level deep (doc V25).
2. **Planner addition (doc-silent — see F4)**: immediately above the `run_bounded` call, validate the knob,
   mirroring test:10-13:
   ```
   if [[ ! "$gate_cap_secs" =~ ^[1-9][0-9]*$ ]]; then
     echo "not ok - PROBE_GETCONF_CAP_SECS must be a positive integer" >&2
     exit 1
   fi
   ```
   No new arm. If the controller prefers to drop this, delete those four lines; **no acceptance number in this
   plan changes**.
3. **After the M2 kill-arm block** (still ahead of the test:743 `PROBE_MAX_TREE_NODES` guard), insert the tail
   recursion guard and the two re-exec arms exactly as the doc's block at lines 264-285: the
   `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY leaked into the arm section; refusing to recurse` guard, then the
   Darwin `if` building `$tmp_dir/hostile-lsof/lsof` (a `#!/bin/bash` + `exit 1` script, `chmod +x`) and
   `$tmp_dir/benign-dir`, then:
   ```
   expect_failure "REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH" "resolved outside getconf PATH" \
     env PATH="$hostile_lsof_dir:$PATH" PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 PROBE_UNDER_TEST="$tmp_dir/no-probe" /bin/bash "$0"
   expect_success "REAL_LSOF containment accepts a leading directory without an lsof" \
     env PATH="$benign_dir:$PATH" PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 PROBE_UNDER_TEST="$tmp_dir/no-probe" /bin/bash "$0"
   ```
   with the `else echo "UNINFORMATIVE: REAL_LSOF containment arms are Darwin-only, as is the gate they pin"` branch.
   **`PROBE_UNDER_TEST="$tmp_dir/no-probe"` stays on the two arm env lines and NEVER at suite scope** — that is
   option A, the 2.07× → ~5,000× margin fix from quorum round 1, and the suite already polices exactly this
   discipline for `PROBE_MAX_TREE_NODES` at test:743-752.
4. Update the comment block at test:30-38: its last sentence, `Hardening that resolution against an ambient
   hijack is tracked as charter row 6o.`, now points at a gate that exists. Rewrite it to name the gate and its
   two arms.

**Acceptance:**

| # | Command | Required |
|---|---|---|
| M3-G1 (**gate**) | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, **46** `ok`, 0 `not ok`, `PASS: 46 probe self-test arms ran` |
| M3-G2 | `grep -n '^ok 4[3-6]' /tmp/s.out` | `ok 43 - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild`; `ok 44 - REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH`; `ok 45 - REAL_LSOF containment accepts a leading directory without an lsof`; `ok 46 - refusal-branch count still matches the set this suite covers (28)` |
| M3-G3 | **T4 drill** | rc=1, **43** `ok`, sole `not ok` = `REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH unexpectedly succeeded`; elapsed ≈ 58 s (a post-hoc OBSERVATION that no inner run happened — **not** a bound; the bound is `run_bounded`'s 120 s + 5 s per arm) |
| M3-G4 | **T5 drill** | rc=1, 43 `ok`, sole `not ok` = `… ahead of getconf PATH lacked expected message: resolved outside getconf PATH` |
| M3-G5 | **T6 drill** | suite refuses **before arm 1**: zero `^ok ` lines, stderr `not ok - REAL_LSOF resolved outside getconf PATH: /usr/sbin/lsof is not in any of /usr/bin:/bin:/usr/sbin:/sbin; …` |
| M3-G6 | **T7 drill** | rc=1, 44 `ok`, sole `not ok` = `REAL_LSOF containment accepts a leading directory without an lsof failed`, with `not ok - classification fixture: missing loopback` and `no-probe: No such file or directory` in the captured stderr; arm 44 stays green; the tail guard is NOT reached |
| M3-G7 | **T12 drill** (timeout branch) | zero `^ok ` lines; stderr contains `REAL_LSOF containment: /usr/bin/getconf PATH exceeded its 2s cap; instrument failure, not a verdict`; total elapsed **< 10 s** |
| M3-G8 | **T13 drill** (non-zero branch) | zero `^ok ` lines; stderr contains `REAL_LSOF containment: /usr/bin/getconf PATH exited 3; instrument failure, not a verdict` |
| M3-G9 | **T14 drill** (empty-output branch) | zero `^ok ` lines; stderr contains `REAL_LSOF containment: /usr/bin/getconf PATH produced no text; instrument failure, not a verdict` |
| M3-G10 | **T10 vacuity control** | rc=0, 46 `ok` — the append-shaped mutant is VACUOUS and must be rejected as evidence |
| M3-G11 | `grep -c 'PROBE_UNDER_TEST="\$tmp_dir/no-probe"'` ; `grep -c 'no-probe'` | 2 ; 3 (two env lines + the comment) |
| M3-G12 | `shasum -a 256 tools/eval/motoko_connection_probe.sh`; `bash -n` ×2 | hash unchanged; rc=0, rc=0 |

---

### M4 — Changelog, full acceptance sweep, three consecutive green runs

**Goal**: discharge the doc's acceptance table end to end and record the drill outputs the doc lists as sprint
deliverables. Touches only `changelogs/v0.32-current.md` — so the suite is green here by construction.

**Edits**: one entry under the motoko/eval-tooling heading in `changelogs/v0.32-current.md`, naming both
sub-items, the three new arms, the +11 s suite cost, and the fact that the production probe was not modified.

**Acceptance** — the doc's AC table, with F2's correction:

| # | Command | Required after |
|---|---|---|
| AC1 | `bash -n tools/eval/motoko_connection_probe.sh; bash -n tools/eval/test_motoko_connection_probe.sh` | rc=0, rc=0 |
| AC2 | `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99` — unchanged |
| AC3 (**gate**) | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc=0, 46 `ok`, 0 `not ok`, `PASS: 46 probe self-test arms ran`, **three consecutive runs, sequential (never in parallel — the suite is load-sensitive)**; record `sysctl -n vm.loadavg` with each |
| AC4 | `grep -n '^ok 4[3-6]' /tmp/s.out` | the four names in M3-G2 |
| AC5 | `grep 'run_lane evidence' /tmp/s.out \| grep -o 'fixture-[0-9]*\|survivors=[0-9]*\|outer_cap_fired=[a-z]*' \| paste -sd' ' -` | `fixture-2861 outer_cap_fired=no … survivors=0 fixture-2863 outer_cap_fired=no … survivors=0` |
| AC6 | T1 drill | rc=1, 42 `ok`, sole `not ok` arm 43 with `survivors=1` |
| AC7 | T2 drill | rc=1, 35 `ok`, arm 36 — unchanged from base |
| AC8 | T4 drill | rc=1, 43 `ok`, sole `not ok` arm 44 `unexpectedly succeeded`; elapsed ≈ 58 s recorded as an **observation**, not a bound |
| AC9 | `grep -c 'expected_refusal_branches=28' tools/eval/test_motoko_connection_probe.sh` | 1 |
| AC10 | `grep -c 'run_lane_fixture_arm'` ; `grep -c 'run_lane_fixture_secs=2861'` ; `grep -c 'run_lane_fixture_secs=\$fixture_secs'` | ≥ 3 (def + two calls) ; 0 ; 1 |
| **AC11a** *(restated — see F2)* | `grep -n -- '-- /usr/bin/getconf PATH' tools/eval/test_motoko_connection_probe.sh` | **exactly 1 hit**, and its line number is **< 168** (the first textual `pass_arm "`) and **> 139** (`run_bounded`'s closing brace) |
| **AC11b** *(restated — see F2)* | `grep -c '/usr/bin/getconf PATH' tools/eval/test_motoko_connection_probe.sh` | **4** — one `run_bounded` argument plus three refusal-message literals. The doc's "1" is stale and was written against the pre-relocation gate. |
| AC12 | new-arm order in `/tmp/s.out` | line numbers strictly increasing 42 → 43 → 44 → 45 → 46 |
| AC13 | `grep -c 'PROBE_UNDER_TEST="\$tmp_dir/no-probe"'` ; `grep -c 'no-probe'` | 2 ; 3 |
| AC14 *(planner)* | `git status --short` | shows exactly `tools/eval/test_motoko_connection_probe.sh` and `changelogs/v0.32-current.md` as modified, and the design doc + this plan + the sprint JSON as untracked/modified. **Nothing else. And no commit was made.** |

---

## 5. Arm placement and ordering — where each new arm lands, and why

The suite is **fail-fast**: `expect_failure` exits 1 on either failure path and every hand-rolled arm ends in
`exit 1`, so the first red terminates the run and every later arm is unreached.

Several arms are **wall-clock bounded** — the `PROBE_TIMEOUT_SECS=4` `run_live` arms (26-34), arm 36's 5 s
readiness cap, the `refusing live path` arm. Iteration 33 measured that inserting a forking arm **ahead** of them
gave **4 reds in 19 runs at position 26**, against **0 in 5 at position 42** and **0 in 17 at base**.

**Therefore, all three new arms go BEHIND arm 42** (`descendant discovery stub refusal carries its own message`,
test:740-741) and **ahead of** the suite-scope env guards (test:743-752, 754-764) and the refusal-branch gate
(test:766-790). Arm 42 was itself placed as "the last forking arm" by iteration 33; the new arms extend that
tail, so **every wall-clock-bounded arm begins at exactly the offset it has at base**. The mechanism is
unreachable by construction rather than argued from a flake rate.

| New arm | Position | Cost | Why there |
|---|---|---|---|
| 43 — SIGKILL escalation | first after arm 42 | ~9 s (spends the probe's real 5 s grace window) | The only expensive new arm. Last-but-three means it tips nothing else. It forks a probe + wrapper + grandchild; ahead of the 4 s-deadline arms this is exactly the iteration-33 failure shape. |
| 44 — hostile PATH | after 43 | ~15 ms (one bash startup + the gate) | Must run after 43 so a kill-arm regression is reported by name rather than masked. Cheap enough that its position is not load-bearing, but it is grouped with 45 for readability. |
| 45 — benign PATH | after 44 | ~15 ms | The same-shape negative control for 44. Must run after 44 so that if the gate is simply broken, the *hostile* arm is the first red — the informative one. |
| 46 — refusal-branch count | unchanged, last | ~0 | Pre-existing. It renumbers 43 → 46; that is the only visible effect on an existing arm. |

**Function locality vs call ordering**: `run_lane_fixture_arm` is *defined* where the 6i block lives today (inside
the first `skip_run_lane_fixture` guard, ~test:536); its second call is ~200 lines later. A reader following arm 36
finds the function beside it. The function's header comment must name the second call site and say why it is far
away — otherwise the next editor "tidies" the second call up next to the first and silently re-creates the
iteration-33 flake.

**Honest residual**: arm 43 is itself wall-clock bounded (5 s readiness cap, 17 s outer cap) and shares arm 36's
exposure to host contention. Placing it last means it tips nothing else — not that it cannot flake. That exposure
is row 6p's (derive bounds from an in-test stimulus), not this row's.

---

## 6. Mutation-drill discipline — the four rules this suite has already broken three times

Every drill in §7 obeys all four. A drill that violates any of them produces **no evidence** and must be re-run.

**Rule 1 — an APPEND-shaped mutant is VACUOUS BY CONSTRUCTION.** `expect_failure`'s matcher is
`grep -Fq -- "$expected"` over the **whole** captured stderr (test:163). Appending text to a message leaves the
substring intact and the arm green. **Every mutation must CHANGE or REMOVE matched text.** T10 exists purely as
the control that demonstrates this, and its expected result is *green*.
*Sole exception*: T9, where the oracle is a **count** (`grep -c 'instrument_failure "'`), not a substring match —
there an addition is precisely what the gate is built to catch. Say so out loud when reporting T9.

**Rule 2 — mutants are `cp -p` copies, never `sed > file`.** A `sed > file` mutant lands mode 644; the suite
invokes the probe directly at test:201 (`"$probe" --classify-fixture …`), so the run reds at **arm 1 on the file
MODE**, not on the mutation, and the drill silently measures nothing. Use `cp -p` (preserves 755), then
`sed -i ''` in place, then **assert the mode**: `ls -l <mutant>` → `-rwxr-xr-x`. Drive every probe mutant with
`PROBE_UNDER_TEST=<copy>` so the tree is never edited.
For **suite** mutants the same `cp -p` hygiene applies, and additionally
`PROBE_UNDER_TEST="$PWD/tools/eval/motoko_connection_probe.sh"` is **mandatory**: a `/tmp` suite copy derives
`script_dir` from `$0`, so its default `$probe` points into `/tmp` where no probe exists.

**Rule 3 — read WHICH ARM failed, never the rc alone.** rc=1 is produced by every arm, by the arm cap, by an
instrument failure and by a startup refusal. Every row in §7 names the exact `not ok` substring that must appear,
and where relevant the `ok` count that must precede it. Report the substring, not "it went red".

**Rule 4 — assert the mutation took, with a same-family control that did NOT move.** Before running any mutant:
`bash -n <mutant>` → rc=0; the intended count moves; a sibling count stays put; `diff` shows only the intended
lines. A mutant whose count did not move is a no-op run reported as a survivor — the exact shape that made V6's
"survives" reading meaningful only because V7/V8 moved.

---

## 7. Test rows — mutation, file, and the exact `not ok` text it must produce

The "kills which mutation" column is the one claim in a sprint nobody ever checks. Each row below names **the
mutation**, **the file it is applied to**, and **the `not ok` substring** that must appear. If the substring
differs, the row FAILED even if rc=1.

**Mutant scaffolding, once:**
```
m=$(mktemp -d /tmp/iter34-mut.XXXXXX)          # probe mutants
s=$(mktemp -d /tmp/iter34-smut.XXXXXX)         # suite mutants
g=$(mktemp -d /tmp/iter34-getconf.XXXXXX)      # fake getconf fixtures for T12/T13/T14
PRISTINE="$PWD/tools/eval/motoko_connection_probe.sh"
```

| # | Mutation (exact) | File mutated | Run as | Must red with `not ok` line containing | Killer status | Milestone | Measured before the sprint? |
|---|---|---|---|---|---|---|---|
| **T1** | probe:261 `kill -9 "-$pid"` → `kill -9 "$pid"`. Assert: group-`-9` count 1→**0**, group-TERM count stays **1**, mode `-rwxr-xr-x`, `diff` = line 261 only | `motoko_connection_probe.sh` (copy) | `PROBE_UNDER_TEST=$m/probe-9only.sh /bin/bash tools/eval/test_motoko_connection_probe.sh` | `production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild (outer_rc=0 survivors=1 cleanup=0 probe_rc=` — with **42** `ok` before it | **SOLE killer.** This is the sprint's headline: at base this mutant SURVIVES (rc=0, 43 ok) | **M2** | Yes — doc V6 (survives at base) / V24 (reds on the prototype) |
| **T2** | probe:252 **and** 261 both → single-PID form. Assert: both group counts → **0** | probe (copy) | `PROBE_UNDER_TEST=$m/probe-both.sh …` | `production run_lane timeout kills wrapper grandchild (` … `survivors=1` — with **35** `ok` before it | Arm 36 reds first and **masks** arm 43 by fail-fast. Proves 6i's coverage is intact after the hoist | **M1 + M2** | Yes — doc V7, V26 |
| **T3** | probe:252 `kill -TERM "-$pid"` → `kill -TERM "$pid"` **only**. Assert: group-TERM 1→**0**, group-`-9` stays **1** | probe (copy) | `PROBE_UNDER_TEST=$m/probe-termonly.sh …` | `production run_lane timeout kills wrapper grandchild (` … `survivors=1` — **35** `ok` before it | Arm 36 is the TERM half's sole killer; arm 43 need not be. Arm 43 is unreached (fail-fast) and would be green by construction — TERM is a no-op on a TERM-immune tree | **M1** | Yes — doc V8 |
| **T4** | Delete the containment gate block (the `if [[ "$host_os" == Darwin ]]; then … fi` containing `resolved outside getconf PATH`); **keep** the marker exit. Assert: `grep -c 'resolved outside getconf PATH'` 2→**1** (the arm's expected-substring survives) | `test_motoko_connection_probe.sh` (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-nocontain.sh` | `REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH unexpectedly succeeded` — **43** `ok` before it | **SOLE killer** of the gate's existence. Elapsed ≈ 58 s proves no recursion happened | **M3** | Yes — doc V25 (pre-relocation gate) |
| **T5** | Gate message text: `resolved outside getconf PATH` → `resolved OUTSIDE the standard path`, in the **echo only**. Assert: the echo's literal moved; the arm's expected-string literal did NOT | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-msg.sh` | `REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH lacked expected message: resolved outside getconf PATH` | Sole killer (arm 44) of the message text | **M3** | No — predicted from V20's matcher. **Run it.** |
| **T6** | Invert the containment predicate: `[[ "$standard_path_entry" == "$real_lsof_dir" ]]` → `!=` | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-invert.sh` | The suite refuses **before arm 1**: `not ok - REAL_LSOF resolved outside getconf PATH: /usr/sbin/lsof is not in any of /usr/bin:/bin:/usr/sbin:/sbin;` with **zero** `^ok ` lines | **NOT an arm killer** — the gate's own startup refusal is the red. Listed so nobody expects arm 44/45 to name it | **M3** | No — predicted from V15. **Run it.** |
| **T7** | Remove the marker early-exit block alone (gate intact). Assert: `grep -c 'REAL_LSOF containment check passed'` 1→**0** | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-nomarker.sh` | Arm 44 stays green; arm 45 reds: `REAL_LSOF containment accepts a leading directory without an lsof failed`, and the captured stderr contains `not ok - classification fixture: missing loopback` **and** `no-probe: No such file or directory` | Arm 45 sole killer. The leaked inner run costs ~20 ms (V37) inside the arm's 120 s `run_bounded` cap (V35) — this is the option-A scoping earning its keep. The tail recursion guard is NOT reached | **M3** | Mechanism yes (V37, on HEAD); the drill itself **no** — the `/tmp` prototype did not survive. **Run it; expected outer elapsed ≈ 60 s.** |
| **T8** | Drop `PROBE_TEST_IGNORE_TERM=1` from arm 43's call line. Assert: `grep -c 'PROBE_TEST_IGNORE_TERM=1'` 2→**1** (test:363's copy survives) | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-noignore.sh` | `production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild (outer_rc=0 survivors=0` — red because `timeout=no`: the probe emits the probe:272 sampling-deadline string, not probe:267's | Arm 43 sole killer of its own env line. Proves the arm really reaches the escalation rather than passing for the wrong reason | **M2** | No — predicted from V6 (that run IS this configuration on the base fixture). **Run it.** |
| **T9** | Append one line to the probe copy that the counter matches but the runtime ignores, e.g. `: # instrument_failure "synthetic drift probe"`. Assert: `grep -c 'instrument_failure "'` 20→**21**; `bash -n` rc=0; mode 755 | probe (copy) | `PROBE_UNDER_TEST=$m/probe-drift.sh /bin/bash tools/eval/test_motoko_connection_probe.sh` | `refusal-branch drift: probe has 29 refusal branches,` — with **45** `ok` before it | Arm 46 sole killer. **The one legitimate append-shaped mutation** (Rule 1's exception: the oracle is a count, not a substring match) | **M4** | No — the gate's anti-vacuity floor is at test:778-782. **Run it.** |
| **T10** | **Vacuity control.** APPEND ` (x)` to the gate's containment message: `…would serve as the survivor oracle (x)` | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-append.sh` | **Nothing. rc=0, 46 `ok`.** The arm stays green because `grep -Fq` still finds the substring | **REJECTED as evidence — by design.** This row is the proof that Rule 1 is real on THIS suite, not a story about it | **M3** | Established in principle by iteration 33. **Run it here anyway** — it costs 60 s and it is what makes T5 credible |
| **T11** | Neuter `set -m` in the probe's `run_lane` | probe (copy) | `PROBE_UNDER_TEST=$m/probe-nomonitor.sh …` | Arm 36 via its `INSTRUMENT DEGRADED` branch, per 6i's IV3 | Arm 36; unchanged by this design. Optional regression confirmation only | **M1** (optional) | No — cited from 6i's doc |
| **T12** | In the suite copy, point the gate's `run_bounded` argument at a slow fixture: `sed -i '' "s#-- /usr/bin/getconf PATH#-- $g/getconf PATH#"`, where `$g/getconf` is `#!/bin/sh` + `sleep 30`, mode 755. Assert: `grep -c -- '-- /usr/bin/getconf PATH'` 1→**0**, `grep -c -- "-- $g/getconf PATH"` = **1**, and `grep -c '/usr/bin/getconf PATH exceeded'` stays **1** (the message literal must NOT move — that is what makes the expected text exact) | test suite (copy) | `PROBE_GETCONF_CAP_SECS=2 PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-slowgetconf.sh` | `REAL_LSOF containment: /usr/bin/getconf PATH exceeded its 2s cap; instrument failure, not a verdict` — with **zero** `^ok ` lines before it, and total elapsed **< 10 s** (proving the cap terminated it rather than the 30 s sleep completing) | **SOLE killer of the timeout branch.** Distinguishes timeout from the non-zero and empty branches (T13/T14) | **M3** | No — the drill is specified by the doc as a sprint deliverable. **Run it.** |
| **T13** *(planner — discharges F3)* | Same shape as T12 but the fixture is `#!/bin/sh` + `exit 3`; no `PROBE_GETCONF_CAP_SECS` override | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-rc3getconf.sh` | `REAL_LSOF containment: /usr/bin/getconf PATH exited 3; instrument failure, not a verdict` — **zero** `^ok ` lines before it | **SOLE killer of the non-zero branch.** The doc names T13 in T12's killer column but never writes it; this is that row | **M3** | No. **Run it.** |
| **T14** *(planner — discharges F3)* | Same shape; fixture is `#!/bin/sh` + `exit 0` printing nothing | test suite (copy) | `PROBE_UNDER_TEST="$PRISTINE" /bin/bash $s/suite-emptygetconf.sh` | `REAL_LSOF containment: /usr/bin/getconf PATH produced no text; instrument failure, not a verdict` — **zero** `^ok ` lines before it | **SOLE killer of the empty-output branch.** Without it, the doc's own Risks-table mitigation ("Explicit instrument-failure branch on empty output") is an unpinned claim | **M3** | No. **Run it.** |

**Which write does each read?** T1 reads probe:261 through the 2863 fixture's cwd oracle. T2/T3/T11 read probe:252
and `set -m` through the 2861 fixture's oracle. T4/T5/T6 read the gate's presence, text and predicate through a
real `command -p -v` under a real prepended directory. T7 reads the absent-probe scoping on the arm's own env
line, through arm 1 of the inner run, with the tail guard as the depth backstop. T8 reads arm 43's own env line.
T9 reads the refusal-branch counter. T12/T13/T14 read the three bounded-gate refusal branches independently.
**No new arm observes a value set alongside the mechanism it pins.**

---

## 8. UNMEASURED at plan time — what the sprint must MEASURE vs what is a PREDICTION

### Must be measured DURING the sprint (these are deliverables, not predictions)

| Item | Where |
|---|---|
| **T5** — the gate-message drill | M3-G4 |
| **T6** — the inverted-predicate drill (startup refusal, zero `ok` lines) | M3-G5 |
| **T7** — the marker-removal drill on the REAL tree (only the *mechanism* is measured, on HEAD, at V37; the drill itself was lost with the `/tmp` prototype) | M3-G6 |
| **T8** — the dropped-`PROBE_TEST_IGNORE_TERM` drill | M2-G7 |
| **T9** — the refusal-branch-drift drill | M4 |
| **T10** — the vacuity control | M3-G10 |
| **T12 / T13 / T14** — all three bounded-gate refusal branches. **None has ever been run**; the gate itself has never existed outside this doc, since the surviving prototype predates the round-2 relocation | M3-G7/8/9 |
| **The relocated gate's green-path cost.** V36 timed the *pre-relocation* gate body (5 ms, a direct `getconf` call). The relocated gate forks `run_bounded`, which adds a background job, a `jobs -p` check and a poll loop. **Re-time it.** | M3 |
| **The three-consecutive-green requirement at the real arm count (46), sequential, with `sysctl -n vm.loadavg` recorded per run** | AC3 |
| **Final suite wall-clock.** The doc predicts ~60 s (+11 s over base) from the pre-relocation prototype. Record the actual. | AC3 |
| **Final line count** of the suite (doc predicts 874; the bounded gate makes that stale — expect ~890-900) | M4 |

### Predictions carried from the doc, NOT re-measurable in this sprint

1. **GitHub `macos-latest` runner behaviour** — that `/usr/bin/getconf PATH` returns the same four entries and
   that `command -p -v` behaves identically. Consequence if wrong: the gate refuses at startup on the runner with
   its own explicit message naming both strings. Fail-loud, not silent. **And note: the CI leg that would show
   this (`launchd drivers (bash 3.2)`) is red at base for an unrelated reason, so this sprint gets no CI signal
   on it either way.** Say that in the sprint report rather than implying CI validated anything.
2. **Non-Darwin behaviour** — skipped by construction (`host_os != Darwin`); no Linux host was available.
3. **The "would be green" half of T3** — fail-fast hides it; it follows from TERM being a no-op on a TERM-immune
   tree (V9), not from a run.
4. **Arm 43's flake rate under heavy load** — 3/3 at load ≤ 2.9 is not a rate. Iteration 33's readiness-cap red
   at load 39-46 applies to arm 43 as much as to arm 36.
5. **Triple test-side mutant recursion** (marker exit + absent-probe scoping + tail guard all removed) —
   depth-unbounded, each level inside its parent's 120 s + 5 s cap. Not a drill; the same exposure the
   pre-existing test:721 re-exec has always had.
6. **`/bin/realpath` on every supported macOS** — present here; its portability is why the gate is un-normalised
   by choice and normalisation is deferred.

---

## 9. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| The 145-line hoist silently changes arm 36's behaviour | **medium** — the highest single risk in this sprint | The substitutions are enumerated line by line; M1-G3 requires the 2861 evidence line byte-comparable; M1-G7/G8 require 6i's mutants to still red on arm 36 |
| The empty-array expansion aborts the suite on M1's first call | medium | `${run_lane_extra_env[@]+"${run_lane_extra_env[@]}"}` is mandatory on bash 3.2 under `set -u`; M1-G1 catches it instantly |
| An executor implements the gate at test:28 from the stale doc line 175 | **medium** | §2 rules explicitly; every task references test:139/141; AC11a pins the line number range |
| AC11 as written in the doc fails a correct implementation | **high if unaddressed** | Restated as AC11a/AC11b (F2) |
| Arm 43 flakes under host contention | low-medium | Placed last-but-three so it tips nothing else; caps unchanged in kind; residual is row 6p's |
| A drill runs concurrently with another and both flake | medium | **All suite runs strictly sequential.** Never background two. Record load average with each |
| A suite mutant is run without `PROBE_UNDER_TEST` and reds at arm 1 on a missing probe, and that is read as the drill's result | medium | Rule 2 and every §7 run column state it; Rule 3 requires reading the arm name, which would immediately show `classification fixture: missing loopback` |
| The executor commits | — | §0. The controller commits. The executor runs no git write |

---

## 10. Files, LOC, and rollback

| File | Change | Est. |
|---|---|---|
| `tools/eval/test_motoko_connection_probe.sh` | gate + marker exit between test:139 and 141; the 6i block hoisted into `run_lane_fixture_arm` with one call in place; kill-arm call, recursion guard and two containment arms after test:741; comment refresh at test:30-38 | **+~100 / −~10** (795 → ~890-900 lines) |
| `tools/eval/motoko_connection_probe.sh` | **NONE.** sha256 must remain `f0b5e024…aabc99` | 0 |
| `changelogs/v0.32-current.md` | one entry (M4) | +~8 |
| New files | none | 0 |

**Rollback**: every milestone is one file's worth of additive change plus one localised hoist. If M1's hoist
misbehaves in a way the drills cannot localise, the correct move is to revert M1 in the working tree and land M3
alone — sub-item (b) has **no dependency** on sub-item (a) and is independently valuable. M2 depends on M1; M3
depends on neither.

---

## 11. Handoff

- **Plan**: `design_docs/planned/m-motoko-group-kill-and-lsof-containment-sprint-plan.md`
- **Progress JSON**: `sprint_m-motoko-group-kill-and-lsof-containment.json` (repo root of this worktree)
- **THE acceptance gate, at every milestone boundary**: `/bin/bash tools/eval/test_motoko_connection_probe.sh`
  → rc=0, `PASS: <N> probe self-test arms ran` (N = 43 / 44 / 46 / 46).
- **The executor performs no git write operations. The controller builds the commits.**

SPRINT_PLAN_PATH: design_docs/planned/m-motoko-group-kill-and-lsof-containment-sprint-plan.md
SPRINT_JSON_PATH: sprint_m-motoko-group-kill-and-lsof-containment.json
