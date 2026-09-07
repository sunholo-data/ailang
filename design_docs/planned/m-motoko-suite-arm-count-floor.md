# Arm-count floor for the motoko connection-probe self-test suite

**Status**: Planned (revision 2 — quorum round 2 BLOCKED, see Quorum verification log)
**Mission row**: motoko queue row 6s

## Problem

`tools/eval/test_motoko_connection_probe.sh` (1212 lines) carries a *production-side* drift gate
at lines ~1165-1184: it counts refusal branches in the probe under test
(`expected_refusal_branches=28`), refuses when the number moves, and carries an anti-vacuity guard
("a counter that returns zero is a broken instrument, not a clean result"). There is NO symmetric
gate on the TEST side. The only arm-count assertion in the whole suite is `if (( arms == 0 ))` at
line 1208 (fatal). So deleting any single test arm leaves the suite rc=0 and green, and the loss
is invisible to CI.

CONTROLLER'S FIRST-PARTY MEASUREMENT, 2026-09-07, at base 878939117 — treat as established fact:
- pristine run:  `PASS: 59 probe self-test arms ran`, rc=0
- mutant: delete the single line `pass_arm "refusal-branch count still matches ..."` (chosen
  because it matches none of the three refusal-branch counter patterns, so no other gate moves)
- mutant run:    `PASS: 58 probe self-test arms ran`, rc=0  <-- STILL GREEN. The defect.
- tree restored, sha256 byte-identical, `git status --porcelain` clean.

## Design

Add an `expected_arms` drift gate beside `expected_refusal_branches`, placed between the existing
`if (( arms == 0 ))` check (line 1208) and the terminal `echo "PASS: $arms ..."` line (line 1212).
The gate reads the runtime counter `$arms` (the same value the terminal line prints), so it measures
the arms that actually EXECUTED — not the 18 static `pass_arm` call sites, which undercount because
several arms run inside loops.

### The shell (exact lines to insert)

```bash
# Arm-count drift gate. Every arm above proves a branch that EXISTS goes red when neutered —
# a removal proves the check FIRES; only an addition proves it LOOKS. Deleting any single arm
# leaves the suite rc=0 and green, so the coverage claim is a one-time manual count that
# silently rots on the next edit. Count the arms and refuse when the number moves.
# Host-scoped: four arms are Darwin-only (see Host-scoping), so the expected count differs
# off-Darwin. This mirrors the suite's own skip_run_lane_fixture / Darwin-only discipline.
if [[ "$host_os" == Darwin ]]; then
  expected_arms=60
else
  expected_arms=56   # 60 - 4 Darwin-only arms (run_lane fixture, SIGKILL-escalation, 2 REAL_LSOF)
fi
# This gate's own pass_arm is itself an arm. It runs BEFORE the check below, so by the time the
# comparison executes, $arms already includes this arm: the check is a bare equality against a
# fully-observed count, never a count plus an assumption about a line that might be deleted.
pass_arm "arm-count still matches the set this suite covers ($((arms + 1)))"
if (( arms != expected_arms )); then
  echo "not ok - arm-count drift: suite ran $arms arms, this suite is written for $expected_arms." >&2
  echo "         Add an arm for the new case (or delete a stale one), then update expected_arms." >&2
  exit 1
fi
```

`expected_arms=60` (Darwin) = 59 original arms + this gate's own `pass_arm`. The gate must remain
the LAST arm (immediately before the terminal line); any arm added to the suite lands before it and
is counted.

### Exact final ordering of the lines around the gate

The gate sits between the existing anti-vacuity guard and the terminal line, in this exact order:

```
1208  if (( arms == 0 )); then
1209    echo "not ok - zero test arms ran" >&2
1210    exit 1
1211  fi
      # --- gate inserted here ---
      if [[ "$host_os" == Darwin ]]; then expected_arms=60; else expected_arms=56; fi
      pass_arm "arm-count still matches the set this suite covers ($((arms + 1)))"
      if (( arms != expected_arms )); then
        echo "not ok - arm-count drift: suite ran $arms arms, this suite is written for $expected_arms." >&2
        echo "         Add an arm for the new case (or delete a stale one), then update expected_arms." >&2
        exit 1
      fi
      # --- end gate ---
1212  echo "PASS: $arms probe self-test arms ran"
```

The gate's own `pass_arm` runs BEFORE the comparison, so `$arms` is 60 (Darwin) when the bare
`(( arms != expected_arms ))` executes, and the terminal line prints the SAME `$arms` the gate
checked. The `+1` construction is gone: the gate asserts on the count it actually observes, never
on a count plus an assumption about a line that may have been deleted.

### Decision: EXACT-EQUALITY, not floor (>=)

The gate uses `!=` (exact equality), matching the refusal-branch gate. Reasoning:

- **The addition mutant only kills an exact gate.** With a floor `arms >= expected_arms`, a removal
  (59→58) fires the gate, but an addition (59→60) passes — the gate would only ever prove it FIRES,
  never that it LOOKS. Requirement (b) demands an addition-shaped mutant that kills the gate; only
  exact equality delivers it.
- **The charter row's cost is already paid.** "an exact arm-count gate reds on every legitimate arm
  addition, a maintenance tax the branch-count gate already charges and that this loop has judged
  worth paying once." The refusal-branch gate already charges this exact tax (28, `!=`). The arm-count
  gate charges the same tax for the same benefit: a legitimate arm addition reds the suite and forces
  a one-line `expected_arms` bump, exactly as a legitimate refusal addition forces an
  `expected_refusal_branches` bump. The loop has judged that tax worth paying once; this design pays
  it a second time, consistently.

### Self-reference (the trap this design must not ship broken)

The gate is itself an arm: its own `pass_arm` increments `$arms`. Two failure modes are avoided:

1. **Naive `expected_arms=59` with a post-gate check** would immediately red on the pristine run,
   because the gate's own arm makes the count 60. The design instead sets `expected_arms=60` and
   runs the gate's own `pass_arm` BEFORE the comparison, so the check sees the fully-observed 60.
2. **A pre-gate check that ignores the gate's own arm** would measure 59 and pass, but then the
   terminal would print 60 while `expected_arms` says 59 — a silent off-by-one that rots. Placing
   the comparison AFTER the gate's own `pass_arm` makes the checked count and the terminal count the
   same variable at the same value (60), so the two can never drift apart.

The ordering — `pass_arm` first, then the bare equality — is the single point of self-reference
coupling; it is documented in the comment directly above the `pass_arm`.

### Anti-vacuity

The gate carries NO redundant `if (( arms == 0 ))` guard. The existing line-1208 check
(`if (( arms == 0 )); then echo "not ok - zero test arms ran" >&2; exit 1; fi`) supplies the
anti-vacuity property for this gate, and it is sufficient: it sits BEFORE the gate's own `pass_arm`,
so a zero or non-numeric `$arms` exits 1 there before the gate's arm can run. A zero counter is
"instrument failure, not a verdict" — it reds loudly at line 1208 rather than passing. Documenting
this rather than re-guarding keeps the property explicit and removes the unreachable dead code.

### Host-scoping (objection 4)

Four arms do not run on a non-Darwin host, so a single exact `expected_arms` would red loudly there:

- line 762: `if (( skip_run_lane_fixture )); then echo "UNINFORMATIVE: run_lane fixture arm ..."` —
  the run_lane fixture arm (arm 36) does not run off Darwin.
- line 980: the same guard skips the run_lane SIGKILL-escalation arm.
- line 999: `if [[ "$host_os" == Darwin ]]` runs two REAL_LSOF containment arms
  (`expect_failure`/`expect_success`); its `else` prints only `UNINFORMATIVE: REAL_LSOF containment
  arms are Darwin-only`.

The gate scopes `expected_arms` by host (60 Darwin / 56 non-Darwin), mirroring the discipline the
suite already applies to those arms via `skip_run_lane_fixture` and the Darwin-only blocks. This is
position (a). Evidence that both gating hosts are Darwin: `.github/workflows/ci.yml` runs the
`launchd drivers (bash 3.2)` job on `macos-latest` DELIBERATELY (the job's own comment: a suite
green on bash 5 proves nothing about the rig's bash 3.2), and the rig is Darwin. The stakes are
higher than a rig-local suite: `make/test.mk:72` runs the probe inside the `test-launchd-drivers`
target, and `.github/workflows/ci.yml:602` runs `make test-launchd-drivers` in the `launchd drivers
(bash 3.2)` job on a Darwin runner — so the gate runs in CI on every push, on a Darwin runner, and
a wrong `expected_arms` is a repo-blocking red. That is precisely why the count had to be measured
rather than derived.

## Milestones

- **M1** — Insert the gate shell above between line 1208 and line 1212; set `expected_arms=60`
  (Darwin) / `56` (non-Darwin). Run the suite on the rig; confirm `PASS: 60 probe self-test arms
  ran`, rc=0. Independently committable.
- **M2** — Mutation matrix validation: apply the removal mutant (delete one arm line), the addition
  mutant (add one `pass_arm` line), and the self-arm mutant (delete the gate's OWN `pass_arm` line)
  in turn; confirm each reds the gate with rc=1; restore the tree byte-identical. Independently
  committable.

## Test plan

| Assertion | Mutation that kills it | Expected observable |
|---|---|---|
| Drift gate FIRES on removal | Delete one `pass_arm` line (e.g. line 1184) | `not ok - arm-count drift: suite ran 59 arms...`, rc=1 |
| Drift gate LOOKS on addition | Add one `pass_arm "extra"` line before the gate | `not ok - arm-count drift: suite ran 61 arms...`, rc=1 |
| Gate's OWN arm is observed (objection 1) | Delete the gate's own `pass_arm` line | `not ok - arm-count drift: suite ran 59 arms...`, rc=1 — proves the gate asserts on the count it OBSERVES, not on a `+1` assumption |
| Anti-vacuity fires on zero | Force `arms=0` before line 1208 | `not ok - zero test arms ran`, rc=1 (line 1208, before the gate) |
| Pristine run stays green | None | `PASS: 60 probe self-test arms ran`, rc=0 |

## Conflict Surface

Every other gate whose counts my new lines could move, and the verdict. The search identifies four
direct grep readers of the TEST file (`$0`) — the wall-clock literal census — but exhaustiveness
remains unverified (see Residual below):

- **Refusal-branch gate (lines 1165-1184)** greps `$probe` (the probe file), NOT `$0` (the test
  file). My new lines live in the test file, so they cannot move its counts. Verified anyway: the
  three refusal patterns (`instrument_failure "`, `\|\| usage$`, `echo "process-tree discovery`)
  each match 0 of my proposed lines.
- **Wall-clock literal census (lines 1186-1199)** greps `$0` (the test file) — this one CAN be
  moved by my lines. Verified: the four census patterns (`PROBE_TIMEOUT_SECS=[0-9]+`,
  `bound_secs `, `PROBE_MAX_TREE_NODES=[0-9]+`, `PROBE_MAX_TREE_NODES=`) each match 0 of my
  proposed lines, so the census counts (5, 12, 1, 5) are unchanged.
- **Existing `arms == 0` check (line 1208)** is unaffected; my gate sits after it and relies on it
  for anti-vacuity (no redundant guard).

The search identifies four direct grep readers of $0. The proposed insertion matches none of their
patterns; exhaustiveness remains unverified.

### Residual — unresolved by this revision

gpt6-astra's further ask — a full-file audit of aliases, sourced files and invoked helpers — was
NOT run. The claim is therefore downgraded from "closed" to "four direct readers found,
exhaustiveness unverified". This residual is carried into the implementation as a known limitation
rather than silently resolved.

## Verification Log

| Claim | Command | Observed |
|---|---|---|
| Set of gates reading the TEST file is closed at the 4 census greps (objection 3) | `grep -n 'grep [-A-Za-z]* .*"\$0"' tools/eval/test_motoko_connection_probe.sh` | `1186:timeout_literal_count=$(grep -Ec 'PROBE_TIMEOUT_SECS=[0-9]+' "$0" \|\| true)`; `1187:bound_secs_match_count=$(grep -c 'bound_secs ' "$0" \|\| true)`; `1188:node_literal_count=$(grep -Ec 'PROBE_MAX_TREE_NODES=[0-9]+' "$0" \|\| true)`; `1189:node_reference_count=$(grep -c 'PROBE_MAX_TREE_NODES=' "$0" \|\| true)` — exactly 4 hits, ALL the wall-clock literal census |
| Known-positive control, same call shape against the other file variable | `grep -c 'grep [-A-Za-z]* .*"\$probe"' tools/eval/test_motoko_connection_probe.sh` | `3` (the refusal-branch gate's greps) |
| Terminal line prints `$arms` | `sed -n '1206,1212p' tools/eval/test_motoko_connection_probe.sh` | `echo "PASS: $arms probe self-test arms ran"` at line 1212 |
| Only arm-count assertion is `arms == 0` at 1208 (objection 2) | `grep -nw arms tools/eval/test_motoko_connection_probe.sh` | `8:arms=0`; `81:arms=$((arms + 1))`; `82:echo "ok $arms - $1"`; `1208:if (( arms == 0 ))`; `1209:echo "not ok - zero test arms ran"`; `1212:echo "PASS: $arms probe self-test arms ran"` — exactly one assertion hit at 1208; the other hits are the counter init/increment/print and English prose in comments (lines 43, 112, 666, 974, 1013, 1110, 1149, 1154) |
| Known-positive control, same census shape | `grep -cw pass_arm tools/eval/test_motoko_connection_probe.sh` | `19` |
| Negative control (fresh invented literal) | `grep -cw zzq_no_such_token_38 tools/eval/test_motoko_connection_probe.sh` | `0` |
| `expected_refusal_branches=28` at line 1165 | `grep -n 'expected_refusal_branches=' ...` | `1165:expected_refusal_branches=28` |
| Refusal gate greps `$probe` | read lines 1165-1184 | `grep -c ... "$probe"` |
| Census greps `$0` | read lines 1186-1199 | `grep -Ec ... "$0"` |
| `pass_arm` increments `$arms` | `grep -n 'arms=\$((arms + 1))' ...` | `81:arms=$((arms + 1))` |
| `pass_arm` call sites = 18 | `grep -c 'pass_arm ' ...` | `18` |
| Refusal patterns match probe (known-positive) | `grep -c 'instrument_failure "' "$probe"` etc. | 20, 5, 3 (sum 28) |
| Census patterns match test file (known-positive) | `grep -Ec 'PROBE_TIMEOUT_SECS=[0-9]+' ...` etc. | 5, 12, 1, 5 |
| Proposed lines match 0 of all 7 gate patterns | `grep -c <pattern> /tmp/arm_gate_probe.txt` | 0 for each of the 7 |
| Four Darwin-only arms (objection 4) | `sed -n '31p;762p;980p;999,1008p' ...` | line 31 sets `skip_run_lane_fixture` off Darwin; lines 762/980 skip run_lane arms; line 999 Darwin-only REAL_LSOF block with 2 arms |
| CI runs the bash-3.2 job on macos-latest (objection 4) | `sed -n '582,589p' .github/workflows/ci.yml` | `runs-on: macos-latest`; comment: "macOS DELIBERATELY ... the rig runs bash 3.2.57" |
| Base commit / clean tree | `git log --oneline -1`; `git status --porcelain` | `878939117`; empty |
| Pristine runtime arm count = 59 | controller first-party measurement (given) | `PASS: 59 probe self-test arms ran`, rc=0 |
| Determinism — 4 consecutive pristine runs (objection 5) | run the suite 4× on the Darwin rig | baseline + rep1 + rep2 + rep3 all `PASS: 59 probe self-test arms ran`, rc=0 — 4 of 4 identical |
| The multipliers: 18 static `pass_arm` sites → 59 runtime arms (objection 5) | `grep -n 'pass_arm ' ...`; `grep -nE '^[a-z_]+\\(\\)' ...` | four helper wrappers each emit exactly ONE arm per call and are called from loops over FIXED, LITERAL enumerations: `expect_failure()` (def 367, arm 384), `expect_success()` (def 387, arm 400), `run_lane_fixture_arm()` (arm 914), `assert_measurement_failure()` (def 1108, arm 1131). Enclosing loops iterate hard-coded word lists: line 559 `for dependency in dig lsof pgrep jq ailang-stub` (5 literals); line 585/599 `for retained in treatment.driver.log treatment.lsof control.driver.log control.lsof` (4 literals); line 946 `for cap_expect in "exceeded its 2s arm cap" ...`; line 894 `for run_lane_expected_marker in "uname -sm" "dig +short ..." ...`. No arm-emitting loop iterates over a process count, directory listing, or other environment-derived cardinality. SCOPE: static reading of loop headers + 4/4 identical runs — strong evidence, not a proof for every host. Because no arm-emitting loop is environment-dependent, glm's conditional does NOT fire and EXACT EQUALITY stands |
| Non-Darwin count measured, not derived (objection 5) | run suite with PATH-shadowed `uname` returning `Linux` (control: shadowed `uname -s` printed `Linux`, `/usr/bin/uname -s` printed `Darwin` in the same call) | `PASS: 55 probe self-test arms ran`, rc=0; `grep -c UNINFORMATIVE` → `4` (exactly the 4 skipped arms); 55 + the gate's own arm = 56, so `expected_arms=56` is CONFIRMED. LABEL: SIMULATED non-Darwin host (a real Linux box would also differ in shell and tools). Fallback if a reader judges the simulation insufficient: glm's own `else echo 'UNVERIFIED host: arm-count gate skipped' >&2` shape. This doc ADOPTS the measured 56; the fallback is written down so the choice is visible |

## Quorum verification log

**Round 1 — verdict: BLOCKED (3/3 reviewers present, 0 absent) + controller reject.**

- **gpt6-astra** — the `+1` gate silently passes if its own `pass_arm` line is deleted: the
  comparison computes 59+1=60 and succeeds while the terminal prints 59, recreating the single-line
  deletion defect. *Resolved:* the assertion now runs AFTER the gate's own `pass_arm`, as a bare
  `(( arms != expected_arms ))` against the fully-observed 60; a new Test-plan row proves the
  self-arm deletion now reds.
- **gemini-3-1-pro** — the proposed inline `if (( arms == 0 ))` guard is unreachable dead code
  after the identical fatal line-1208 guard. *Resolved:* the redundant guard is deleted; the
  Anti-vacuity section documents that line 1208 supplies the property and why it is sufficient.
- **oc-glm-5-2** — the Conflict Surface enumeration of "every other gate" was unproven. *Resolved:*
  both grep commands and both results (4 hits on `$0`, 3 on `$probe`) are now in the Verification
  Log as the exhaustiveness proof, and the Conflict Surface states the set is closed at those four.
- **controller** — an exact gate on a host-dependent count would red on non-Darwin hosts. *Resolved:*
  position (a) — `expected_arms` is scoped by host (60 Darwin / 56 non-Darwin), mirroring the
  suite's existing `skip_run_lane_fixture` / Darwin-only discipline; the four arms, two Darwin
  blocks, and the macos-latest CI evidence are named in Host-scoping.

**Round 2 — verdict: BLOCKED (3/3 reviewers present, 0 absent).** This is the SECOND revision,
made under Gate 2's narrow-refinement carve-out: reviewer-authored fixes applied verbatim, no
re-quorum. The Fable-diet clause counts this second revision as an overspend and requires it be
FLAGGED.

- **oc-glm-5-2** — the 59 count was asserted, not proven stable, and the non-Darwin 56 was derived,
  not measured. *Resolved:* a Verification Log row records 4 consecutive pristine runs (4/4
  identical `PASS: 59`); a row names every loop containing a `pass_arm` call, its iteration count,
  and that each is fixed/literal (no environment-dependent loop, so EXACT EQUALITY stands); the
  non-Darwin count is now measured via a PATH-shadowed `uname` (55 + gate arm = 56, CONFIRMED),
  labeled as a simulated host with glm's `UNVERIFIED host` fallback written down.
- **gemini-3-1-pro** — the uniqueness claim used a 7-line `sed`, the wrong instrument. *Resolved:*
  replaced with a global word-boundary census (`grep -nw arms ...`) over all 1212 lines showing
  exactly one assertion hit at line 1208, plus both controls (`grep -cw pass_arm` → 19;
  `grep -cw zzq_no_such_token_38` → 0).
- **gpt6-astra** — the Conflict Surface claimed the reader set "is closed at those four" without
  proof. *Resolved:* that sentence is deleted and replaced verbatim with astra's "four direct grep
  readers of $0 ... exhaustiveness remains unverified"; the full-file audit is recorded as an OPEN
  residual carried into implementation as a known limitation.
- **controller (round 2)** — a false sentence claimed the test file is not referenced by CI or the
  Makefile. *Resolved:* deleted; replaced with the correct stakes — `make/test.mk:72` and
  `ci.yml:602` run the probe in CI on every push on a Darwin runner, so a wrong `expected_arms` is
  a repo-blocking red, which is why the count had to be measured rather than derived.

## Acceptance criteria

1. `bash tools/eval/test_motoko_connection_probe.sh` (Darwin) → `PASS: 60 probe self-test arms
   ran`, rc=0.
2. Delete one `pass_arm` line, run → `not ok - arm-count drift: suite ran 59 arms...`, rc=1.
3. Add one `pass_arm "extra"` line before the gate, run → `not ok - arm-count drift: suite ran 61
   arms...`, rc=1.
4. Delete the gate's OWN `pass_arm` line, run → `not ok - arm-count drift: suite ran 59 arms...`,
   rc=1 (objection 1 regression).
5. `grep -c 'pass_arm ' tools/eval/test_motoko_connection_probe.sh` → `19` (18 + the gate's own arm).
6. `git status --porcelain` clean after restoring the tree; `sha256sum` byte-identical to base.
