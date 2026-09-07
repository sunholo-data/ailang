# Arm-count floor for the motoko connection-probe self-test suite

**Status**: Planned (revision 3 — round-3 revision under attended ruling D-MOTOKO-CARVEOUT-1)
**Mission row**: motoko queue row 6s

> **Ruling this revision answers** — D-MOTOKO-CARVEOUT-1 (attended, 2026-09-07): *"overrule the
> carve-out. A partially applied requested audit or controller-substituted remedy does not qualify as
> a verbatim narrow refinement. Perform the residual audit and run a fresh independent quorum before
> row 6s can land. The prior controller routing was incorrect; this does not itself reject the code."*
> The M1 code (bd4ee3d87) stands as shipped for the Darwin branch. This revision (1) folds the
> residual audit gpt6-astra asked for into the doc, with commands and outputs, and (2) replaces the
> round-2 non-Darwin branch — a controller-substituted remedy — with oc-glm-5-2's own remedy 2. The
> Darwin gate body is byte-for-byte what landed; only the non-Darwin branch and the evidence change.

## Problem

`tools/eval/test_motoko_connection_probe.sh` (1212 lines at base 878939117; 1232 on the sprint
branch with M1 applied) carries a *production-side* drift gate at lines 1165-1184: it counts
refusal branches in the probe under test (`expected_refusal_branches=28`), refuses when the number
moves, and carries an anti-vacuity guard ("a counter that returns zero is a broken instrument, not
a clean result"). There is NO symmetric gate on the TEST side. At base, the only arm-count assertion
in the whole suite is `if (( arms == 0 ))` at line 1208 (fatal). So deleting any single test arm
leaves the suite rc=0 and green, and the loss is invisible to CI.

CONTROLLER'S FIRST-PARTY MEASUREMENT, 2026-09-07, at base 878939117 — treat as established fact:
- pristine run:  `PASS: 59 probe self-test arms ran`, rc=0
- mutant: delete the single line `pass_arm "refusal-branch count still matches ..."` (chosen
  because it matches none of the three refusal-branch counter patterns, so no other gate moves)
- mutant run:    `PASS: 58 probe self-test arms ran`, rc=0  <-- STILL GREEN. The defect.
- tree restored, sha256 byte-identical, `git status --porcelain` clean.

## Design

Add an `expected_arms` drift gate beside `expected_refusal_branches`, placed between the existing
`if (( arms == 0 ))` check (line 1208) and the terminal `echo "PASS: $arms ..."` line (line 1212 at
base). The gate reads the runtime counter `$arms` (the same value the terminal line prints), so it
measures the arms that actually EXECUTED — not the 18 static `pass_arm` call sites at base, which
undercount because one `expect_failure` site runs inside a five-literal loop and four helper
wrappers each emit one arm per call (see "Where the 60 arms come from").

### The shell (exact lines to insert)

The Darwin body is the M1 gate exactly as landed in bd4ee3d87. The non-Darwin branch is
oc-glm-5-2's remedy 2, in glm's own shape (`if Darwin ... else echo 'UNVERIFIED host ...' >&2;
exit 1; fi`), because glm's remedy 1 — measuring on a real Linux host — is measured unavailable on
this rig (Verification Log, "Container runtimes"). The round-2 `expected_arms=56` branch is gone: it
rested on a PATH-shadowed `uname`, a simulated host, which is the controller-substituted remedy the
ruling names.

```bash
# Arm-count drift gate. Every arm above proves a branch that EXISTS goes red when neutered —
# a removal proves the check FIRES; only an addition proves it LOOKS. Deleting any single arm
# leaves the suite rc=0 and green, so the coverage claim is a one-time manual count that
# silently rots on the next edit. Count the arms and refuse when the number moves.
# Host-scoped: the expected count is MEASURED on Darwin only (this rig and the macos-latest CI
# runner). No Linux host was reachable to measure the non-Darwin count, so that branch refuses
# loudly instead of asserting a derived number (design doc: Host-scoping).
# The exact count rests on two measured preconditions (design doc: Preconditions):
#   P1 every self-re-exec sub-mode of this suite exits before this tail — a child starts from
#      arms=0 and would red this gate for a reason unrelated to arm drift;
#   P2 the loopback-socket arm above stays UNINFORMATIVE on both gating hosts — it is the one
#      environment-conditional arm in the suite.
if [[ "$host_os" == Darwin ]]; then
  expected_arms=60
  # This gate's own pass_arm is itself an arm. It runs BEFORE the check below, so by the time the
  # comparison executes, $arms already includes this arm: the check is a bare equality against a
  # fully-observed count, never a count plus an assumption about a line that might be deleted.
  pass_arm "arm-count still matches the set this suite covers ($((arms + 1)))"
  if (( arms != expected_arms )); then
    echo "not ok - arm-count drift: suite ran $arms arms, this suite is written for $expected_arms." >&2
    echo "         Add an arm for the new case (or delete a stale one), then update expected_arms." >&2
    exit 1
  fi
else
  echo 'UNVERIFIED host: arm-count gate skipped' >&2
  exit 1
fi
```

`$host_os` is the suite's existing `uname -s` reading at line 20
(`host_os=$(uname -s 2>/dev/null || printf '%s\n' unknown)`), the same variable that already scopes
the Darwin-only arms at lines 31 and 999. `expected_arms=60` (Darwin) = 59 original arms + this
gate's own `pass_arm`. The gate must remain the LAST arm (immediately before the terminal line); any
arm added to the suite lands before it and is counted.

### Exact final ordering of the lines around the gate

Line numbers are the base file's (878939117). The gate sits between the existing anti-vacuity guard
and the terminal line, in this exact order:

```
1208  if (( arms == 0 )); then
1209    echo "not ok - zero test arms ran" >&2
1210    exit 1
1211  fi
      # --- gate inserted here (comment block elided) ---
      if [[ "$host_os" == Darwin ]]; then
        expected_arms=60
        pass_arm "arm-count still matches the set this suite covers ($((arms + 1)))"
        if (( arms != expected_arms )); then
          echo "not ok - arm-count drift: suite ran $arms arms, this suite is written for $expected_arms." >&2
          echo "         Add an arm for the new case (or delete a stale one), then update expected_arms." >&2
          exit 1
        fi
      else
        echo 'UNVERIFIED host: arm-count gate skipped' >&2
        exit 1
      fi
      # --- end gate ---
1212  echo "PASS: $arms probe self-test arms ran"
```

On Darwin the gate's own `pass_arm` runs BEFORE the comparison, so `$arms` is 60 when the bare
`(( arms != expected_arms ))` executes, and the terminal line prints the SAME `$arms` the gate
checked. The `+1` construction is gone: the gate asserts on the count it actually observes, never
on a count plus an assumption about a line that may have been deleted. On a non-Darwin host the
`else` branch exits 1 before the terminal line, so no `PASS:` line is ever printed there.

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
- **glm's condition for exact equality holds, with one stated precondition.** oc-glm-5-2 (round 2):
  "If any loop is environment-dependent, the gate cannot use exact equality." No arm-emitting LOOP is
  environment-dependent (Verification Log, "Loop inventory"). One arm — the loopback-socket arm at
  line 640 — is environment-CONDITIONAL outside any loop, and is measured absent on both gating hosts
  (Preconditions, P2). Exact equality therefore stands on the measured hosts, and the doc says what
  it depends on rather than leaving it to be discovered.

### Where the 60 arms come from

Every runtime arm on the Darwin rig, attributed to its static source (base line numbers; the
instrument is `grep -nE '^[[:space:]]*(expect_failure|expect_success|assert_measurement_failure|run_lane_fixture_arm|pass_arm) '`
on the base file, cross-checked against the 60 `ok` lines of each measured run):

| Source | Static sites | Runtime arms (Darwin rig) | Why they differ |
|---|---|---|---|
| top-level `pass_arm` | 14 at base (427, 591, 604, 640, 727, 757, 954, 1031, 1054, 1070, 1091, 1106, 1184, 1198) + the gate's own | 14 | line 640 is skipped on both gating hosts (P2); the gate adds one |
| `expect_failure` | 34 | 37 | line 563 sits in the 5-literal loop at 559 (+4); line 929 runs inside the `cap_report=$( { ... } 2>&1 )` subshell at 927-931, so its `pass_arm` increments the subshell's copy of `$arms` and never reaches the parent (−1) |
| `expect_success` | 4 (434, 436, 582, 1010) | 4 | — |
| `assert_measurement_failure` | 3 (1134-1136) | 3 | — |
| `run_lane_fixture_arm` | 2 (917, 983) | 2 | — |
| **Total** | | **60** (59 at base, before the gate's own arm) | matches `grep -c '^ok '` = 60 on each of 3 consecutive runs |

Five arms are conditional. Four are decided by the host alone: 917 and 983 via `skip_run_lane_fixture`
(set at line 31 when `$host_os != Darwin`), and 1006/1010 inside the `if [[ "$host_os" == Darwin ]]`
block at 999. The fifth, line 640, is decided by the environment beyond the host: line 612 requires
`uname -s == Darwin` AND `nc` AND `lsof` on PATH, then the arm runs only if a loopback listener
stays alive and `lsof` samples an ESTABLISHED peer within a 5 s deadline (lines 612-647); otherwise
one of two `UNINFORMATIVE UNDER SANDBOX` lines prints and no arm is emitted.

### Preconditions the exact count depends on

These are the two findings of the residual audit that are not conflicts but DEPENDENCIES. A future
change that breaks either reds the gate with a message about "arm-count drift" when the cause is
something else, so they are stated here and in the comment above the gate.

**P1 — every self-re-exec sub-mode exits before the tail.** The suite re-executes ITSELF as a child
process 14 times (`/bin/bash "$0"`, lines 958, 1007, 1011, 1035, 1038, 1041, 1045, 1057, 1063, 1074,
1078, 1084, 1122) in three sub-modes. A child starts from `arms=0` (line 8). It does not reach the
gate only because each sub-mode terminates early:

| Sub-mode | Terminator | Lines | Reaches the gate? |
|---|---|---|---|
| `PROBE_SELFTEST_ARM_CAP_SECS=invalid` | `exit 1` after the `^[1-9][0-9]*$` test | 11-14 | no |
| `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1` | `exit 0` | 181-183 | no |
| `PROBE_SELFTEST_DERIVATION_ONLY=1` | `exit 0` | 353-354 | no |

`grep -nE '^[[:space:]]*exit 0'` returns exactly 3 hits — 183, 204, 354 — and 204 is inside the
canonical-pgrep fixture heredoc (lines 200-206), not a suite path. The suite also refuses to recurse
if any sub-mode variable leaks into the arm section (three refusals, lines 988, 992, 996). So today
no child reaches line 1208. A future sub-mode added WITHOUT an early exit would run a partial arm set
in the child and red either the zero-arm guard (if it ran nothing) or this gate (if it ran some
arms) — for a reason that has nothing to do with arm drift. Anyone adding a sub-mode must add its
early exit, and this is the place that says so.

**P2 — the loopback-socket arm (line 640) stays UNINFORMATIVE on both gating hosts.** Measured:
- rig, 3 consecutive runs on the sprint branch: each output carries exactly one `UNINFORMATIVE`
  line, `UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer; fixture arm remains
  authoritative`, and 60 `ok` lines.
- CI `macos-latest` runner, latest green `dev` run (workflow run 34163313150, job 101869378522,
  commit b8ee24580, pre-gate): the SAME `UNINFORMATIVE` line, then `PASS: 59 probe self-test arms
  ran`. Instrument: `gh run view --job 101869378522 --log`.

Both gating hosts agree, so `expected_arms=60` is right on both. If a gating host ever DOES sample
the peer, the count becomes 61 and the gate reds with a drift message for a non-drift reason. The
design does not change the gate's shape for this; it states the dependency and asks the round-3
quorum to rule on whether the stated precondition is sufficient. (Designer-raised in round 3; not a
reviewer remedy. The obvious alternative — adjusting `expected_arms` by an observed flag — is a
count-plus-a-conditional, the shape round 1 rejected, and is not adopted here.)

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

### Host-scoping (objection 4, revised under glm remedy 2)

Four arms do not run on a non-Darwin host (917, 983, 1006, 1010 — see "Where the 60 arms come
from"), so a single exact `expected_arms` would red loudly there. Round 2 answered this with
`expected_arms=56`, a number obtained by running the suite under a PATH-shadowed `uname` that
returned `Linux` — a SIMULATED host. The ruling names that as a controller-substituted remedy. This
revision makes NO claim about the non-Darwin count. oc-glm-5-2 offered two remedies:

1. **Measure on a real Linux host.** UNAVAILABLE on this rig: `docker`, `podman`, `colima`, `lima`,
   `nerdctl` and `orb` are all absent from PATH, `docker info` exits 127, and the control (`git`) is
   present at `/opt/homebrew/bin/git`, so the probe can see a positive (Verification Log, "Container
   runtimes"). No Linux host is reachable from this loop.
2. **Refuse explicitly on an unmeasured host.** ADOPTED, in glm's own shape: the non-Darwin branch
   prints `UNVERIFIED host: arm-count gate skipped` to stderr and exits 1.

**The consequence, stated plainly:** on a non-Darwin host the suite EXITS 1 and prints no `PASS:`
line. That is a loud refusal on an unmeasured platform, which is glm's stated purpose — the
alternative is a derived constant that could be silently wrong. Note that the message says the
GATE is skipped but the exit code withholds the SUITE's verdict; that is glm's shape verbatim and
is the intended reading: a suite whose last gate cannot run has not passed. What it costs:

- **No CI leg goes red.** The suite is run by exactly one CI job: `.github/workflows/ci.yml:602`
  (`make test-launchd-drivers`) in the `launchd drivers (bash 3.2)` job, `runs-on: macos-latest`
  (line 589), and the job's own comment says macOS is DELIBERATE — a suite green on bash 5 proves
  nothing about the rig's bash 3.2. No `make` target depends on `test-launchd-drivers` (neither
  `test` at `make/test.mk:27` nor `ci` at `make/ci.mk:11`), so the `ubuntu-latest` and
  `windows-latest` jobs never reach it.
- **A Linux contributor running the suite by hand** gets the explicit `UNVERIFIED host` refusal and
  rc=1 instead of a possibly-wrong count. To lift it, someone with a Linux host measures the count
  there (glm remedy 1) and replaces the `else` branch with a measured `expected_arms`.

The stakes on Darwin are unchanged: `make/test.mk:72` runs the probe inside `test-launchd-drivers`,
CI runs that target on every push on a Darwin runner, and a wrong `expected_arms` is a repo-blocking
red. That is precisely why the Darwin count is measured on BOTH gating hosts (P2) rather than
derived.

## Milestones

- **M1** — LANDED as bd4ee3d87 with the round-2 shell (`expected_arms=56` in the `else` branch).
  Measured on the sprint branch: `PASS: 60 probe self-test arms ran`, rc=0 (controller first-party,
  plus 3 consecutive designer runs this iteration). The Darwin body is unchanged by this revision.
- **M1b** — Replace the `else` branch with glm remedy 2 (the refusal) and the comment block with the
  one above; no other line moves. Re-run pristine on Darwin: `PASS: 60`, rc=0. Independently
  committable.
- **M2** — Mutation matrix validation: apply the removal mutant (delete one arm line), the addition
  mutant (add one `pass_arm` line), the self-arm mutant (delete the gate's OWN `pass_arm` line), and
  the refusal mutant (delete the `exit 1` in the `else` branch, exercised under a shadowed `uname`)
  in turn; confirm each reds with rc=1; restore the tree byte-identical. Independently committable.

## Test plan

| Assertion | Mutation that kills it | Expected observable |
|---|---|---|
| Drift gate FIRES on removal | Delete one `pass_arm` line (e.g. line 1184) | `not ok - arm-count drift: suite ran 59 arms...`, rc=1 |
| Drift gate LOOKS on addition | Add one `pass_arm "extra"` line before the gate | `not ok - arm-count drift: suite ran 61 arms...`, rc=1 |
| Gate's OWN arm is observed (objection 1) | Delete the gate's own `pass_arm` line | `not ok - arm-count drift: suite ran 59 arms...`, rc=1 — proves the gate asserts on the count it OBSERVES, not on a `+1` assumption |
| Anti-vacuity fires on zero | Force `arms=0` before line 1208 | `not ok - zero test arms ran`, rc=1 (line 1208, before the gate) |
| Non-Darwin refusal is LOUD (glm remedy 2) | Delete the `exit 1` in the `else` branch | Pristine, under a PATH-shadowed `uname -s` → `Linux`: `UNVERIFIED host: arm-count gate skipped` on stderr, rc=1, no `PASS:` line. Mutant: rc=0 and `PASS: 55 probe self-test arms ran` — the refusal became silent, the mutant is killed. The shadowed `uname` here is a TEST INSTRUMENT that exercises the branch; it measures no Linux count and no number in this doc rests on it |
| Pristine run stays green | None | `PASS: 60 probe self-test arms ran`, rc=0 |

## Conflict Surface

Every other machinery that reads the TEST file, and whether the insertion moves its inputs or
results. Round 2 could name only the four direct `grep ... "$0"` readers and said exhaustiveness was
unverified. The residual audit gpt6-astra asked for has now been run: a full-file review of all
1212 lines at base 878939117 (not a sample), tracing path aliases, sourced files, self-invocations,
external readers by name, and gates that would catch the file by glob. Its inventory:

**In-file readers of the suite's own text**

- **Refusal-branch gate (lines 1165-1184)** greps `$probe` (the probe file), NOT `$0` (the test
  file). The three refusal patterns (`instrument_failure "`, `\|\| usage$`, `echo "process-tree
  discovery`) each match 0 of the proposed lines. NO.
- **Wall-clock literal census (lines 1186-1199)** greps `$0` — the one reader the insertion CAN
  move. The four census patterns (`PROBE_TIMEOUT_SECS=[0-9]+`, `bound_secs `,
  `PROBE_MAX_TREE_NODES=[0-9]+`, `PROBE_MAX_TREE_NODES=`) each match 0 of the proposed lines, so the
  census counts (5, 12, 1, 5) are unchanged. NO.
- **Existing `arms == 0` check (line 1208)** reads the runtime counter, sits BEFORE the gate, and
  supplies its anti-vacuity. Unaffected.
- **The self-re-exec class — NEW in this audit.** `grep -nE '\$0|BASH_SOURCE'` returns 18 hits, of
  which the four census greps are only the tail. Line 4 is a path alias (`script_dir=$(... dirname
  -- "$0" ...)`) used only to locate the probe (line 5); it never reads the suite's text. The other
  14 are the suite re-executing ITSELF (`/bin/bash "$0"`) in three sub-modes. None reads the
  suite's text or counts anything, so the insertion moves no input or result — but a child that
  reached the tail would red the gate on its own partial `$arms`. That is precondition P1, not a
  conflict; see Preconditions.
- **Sourced files: none.** `grep -nE '^[[:space:]]*(source|\.)[[:space:]]+'` → 0 hits;
  `grep -c 'source'` → 0 in the file; control `grep -c 'probe'` → 73 in the same file.

**External readers, complete enumeration (25 files name the suite; 2 are machinery)**

| File | What it does | Moved by the insertion? |
|---|---|---|
| `make/test.mk:72` | `@/bin/bash tools/eval/test_motoko_connection_probe.sh` — runs it, inside `test-launchd-drivers` | Runs it. Not count-based. NO |
| `make/test.mk:75` | `@/bin/bash -n tools/eval/test_motoko_connection_probe.sh` — syntax check | Parses it. NO |
| `scripts/test_check_referenced_paths.sh:46` | fixture `check19: ; @bash tools/eval/test_motoko_connection_probe.sh` | Asserts the PATH exists and is tracked; never reads contents. NO |
| the other 23 | design docs, mission logs, changelogs, sprint JSONs, a retro | prose. NO |

`.github/workflows/ci.yml:602` runs `make test-launchd-drivers` on `macos-latest` (line 589).

**Gates that would catch the file by glob rather than by name** (13 `scripts/check_*.sh`
enumerated; the ones that scan by glob):

| Gate | Scope, measured | Verdict |
|---|---|---|
| `make check-file-sizes` | `for file in $(find internal cmd -name "*.go")` (`make/code-health.mk:167`), cap 800 | Go only, `internal`/`cmd` only. NO |
| `make shellcheck-autopush` | `AUTOPUSH_SHELL_SCRIPTS := scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh` (`make/code-health.mk:34`) | Two named files. NO |
| `scripts/check_context_docs.sh` | CLAUDE.md / `.claude/rules` / `SKILL.md` | Not a context doc. NO |
| `scripts/check_no_personal_email.sh` | pattern-refusal on emails | Insertion adds none. NO |
| `scripts/check_tmpfile_hygiene.sh` | makefiles | Not a makefile. NO |
| `scripts/check_home_isolation.sh` | pattern-refusal on `$HOME` | Insertion touches none. NO |

**Verdict.** Additional readers found: one class (14 self-re-exec sites plus the path alias at
line 4), two external machinery references, zero glob gates. None has its inputs or results moved
by the insertion. The self-re-exec class establishes precondition P1. Exhaustiveness is now
supported by a full-file review with per-class commands and controls (Verification Log, round-3
rows), not by four `$0` greps.

**Why the existing census and zero-arm guard cannot do this job** (gpt6-astra's question). The
wall-clock census counts LITERALS in the file's text (`PROBE_TIMEOUT_SECS=…`, `bound_secs `), so
it is blind to whether an arm EXECUTED; deleting a `pass_arm` line moves none of its four patterns
(measured: the mutant that deleted line 1184 left the census at 5, 12, 1, 5 and the suite green).
The `arms == 0` guard tests a single boundary — it distinguishes "nothing ran" from "something ran"
and says nothing about HOW MUCH ran, which is exactly the 59→58 transition measured as the defect.
Neither reads the runtime counter against an expected value; only the new gate does.

## Verification Log

Two trees are cited. **Base** = 878939117 (the pre-gate file, 1212 lines, read via
`git show 878939117:tools/eval/test_motoko_connection_probe.sh`). **Sprint** = bd4ee3d87 in
worktree `.wt-motoko-iter39-armcount` (M1 applied, 1232 lines). Lines 1-1211 are identical in
both (`git diff 878939117 bd4ee3d87 -- tools/eval/test_motoko_connection_probe.sh` → one hunk,
`@@ -1209,4 +1209,24 @@`, 20 insertions). Rows marked INHERITED were taken in rounds 1-2 and
not re-run this iteration; every other row was run in round 3.

### Rows inherited from rounds 1-2 (base 878939117)

| Claim | Command | Observed |
|---|---|---|
| Four direct grep readers of `$0` (objection 3, round 1) | `grep -n 'grep [-A-Za-z]* .*"\$0"' tools/eval/test_motoko_connection_probe.sh` | `1186:timeout_literal_count=$(grep -Ec 'PROBE_TIMEOUT_SECS=[0-9]+' "$0" \|\| true)`; `1187:bound_secs_match_count=$(grep -c 'bound_secs ' "$0" \|\| true)`; `1188:node_literal_count=$(grep -Ec 'PROBE_MAX_TREE_NODES=[0-9]+' "$0" \|\| true)`; `1189:node_reference_count=$(grep -c 'PROBE_MAX_TREE_NODES=' "$0" \|\| true)` — exactly 4 hits, ALL the wall-clock literal census. This proves only what it searched for; the full reader inventory is in the round-3 rows |
| Known-positive control, same call shape against the other file variable | `grep -c 'grep [-A-Za-z]* .*"\$probe"' tools/eval/test_motoko_connection_probe.sh` | `3` (the refusal-branch gate's greps) |
| Terminal line prints `$arms` | `sed -n '1206,1212p' tools/eval/test_motoko_connection_probe.sh` | `echo "PASS: $arms probe self-test arms ran"` at line 1212 |
| Only arm-count assertion at base is `arms == 0` at 1208 (objection 2, round 2 — gemini's global-grep remedy, kept) | `grep -nw arms tools/eval/test_motoko_connection_probe.sh` (base) | `8:arms=0`; `81:arms=$((arms + 1))`; `82:echo "ok $arms - $1"`; `1208:if (( arms == 0 ))`; `1209:echo "not ok - zero test arms ran"`; `1212:echo "PASS: $arms probe self-test arms ran"` — exactly one assertion hit at 1208; the other hits are the counter init/increment/print and English prose in comments (lines 43, 112, 666, 974, 1013, 1110, 1149, 1154). Re-run on the sprint tree in round 3: the same hits plus the gate's own lines (1215, 1216, 1221, 1224, 1226, 1227, 1228, 1232), i.e. the gate at 1227 is now the second and only other assertion |
| Known-positive control, same census shape | `grep -cw pass_arm tools/eval/test_motoko_connection_probe.sh` | `19` at base; `21` on sprint (re-run round 3) |
| Negative control (fresh invented literal) | `grep -cw zzq_no_such_token_38 tools/eval/test_motoko_connection_probe.sh` | `0` (both trees) |
| `expected_refusal_branches=28` at line 1165 | `grep -n 'expected_refusal_branches=' ...` | `1165:expected_refusal_branches=28` |
| Refusal gate greps `$probe` | `awk 'NR>=1165&&NR<=1184 && /grep/' ...` (re-run round 3) | `1169:actual_instrument_failures=$(grep -c 'instrument_failure "' "$probe")`; `1170:actual_usage_refusals=$(grep -cE '\|\| usage$' "$probe")`; `1171:actual_echo_refusals=$(grep -c 'echo "process-tree discovery' "$probe")` |
| Census greps `$0` | read lines 1186-1199 | `grep -Ec ... "$0"` |
| `pass_arm` increments `$arms` | `grep -n 'arms=\$((arms + 1))' ...` | `81:arms=$((arms + 1))` |
| Refusal patterns match probe (known-positive) | `grep -c 'instrument_failure "' "$probe"` etc. | 20, 5, 3 (sum 28) |
| Census patterns match test file (known-positive) | `grep -Ec 'PROBE_TIMEOUT_SECS=[0-9]+' ...` etc. | 5, 12, 1, 5 |
| Four host-determined Darwin-only arms (objection 4) | `awk 'NR==31 \|\| NR==762 \|\| NR==980 \|\| (NR>=999&&NR<=1014)' ...` (re-run round 3) | line 31 `if [[ "$host_os" != Darwin ]]` sets `skip_run_lane_fixture`; lines 762/980 `if (( skip_run_lane_fixture ))` skip the two run_lane arms; line 999 `if [[ "$host_os" == Darwin ]]` encloses `expect_failure` (1006) and `expect_success` (1010); its `else` (1013) prints only `UNINFORMATIVE: REAL_LSOF containment arms are Darwin-only` |
| CI runs the bash-3.2 job on macos-latest (objection 4) | `sed -n '580,590p' .github/workflows/ci.yml` (re-run round 3) | `runs-on: macos-latest` at 589; comment: "macOS DELIBERATELY, and it is the whole point of this job: the rig runs bash 3.2.57 ... A suite green on bash 5 proves nothing" |
| Pristine runtime arm count at base = 59 | controller first-party measurement, 2026-09-07, base 878939117 | `PASS: 59 probe self-test arms ran`, rc=0 |
| INHERITED — determinism at base, 4 consecutive pristine runs (glm i, round 2) | run the base suite 4× on the Darwin rig, round 2, base 878939117 | baseline + rep1 + rep2 + rep3 all `PASS: 59 probe self-test arms ran`, rc=0 — 4 of 4 identical. Not re-run this iteration; superseded on the sprint tree by the round-3 determinism row (60, 3/3 + the controller's run) |
| INHERITED, CORRECTED in round 3 — loop inventory (glm ii, round 2) | round 2 read loop headers statically | Round 2 named `expect_failure`, `expect_success`, `run_lane_fixture_arm`, `assert_measurement_failure` as "called from loops" at lines 559, 585/599, 894, 946. Round 3 re-read every loop body: only the 559 loop's BODY emits arms; 585/599/894/946 are assertion loops FOLLOWED by a single `pass_arm`. The conclusion (no environment-dependent arm-emitting loop) is unchanged; the description was imprecise. See the round-3 "Loop inventory" row |
| DEMOTED — SUPPORTING CONTEXT ONLY, no number in this doc rests on it | round 2 ran the base suite with a PATH-shadowed `uname` returning `Linux` (control: shadowed `uname -s` → `Linux`, `/usr/bin/uname -s` → `Darwin` in the same call) | `PASS: 55 probe self-test arms ran`, rc=0; `grep -c UNINFORMATIVE` → `4`. Round 2 adopted 55 + 1 = 56 as the non-Darwin `expected_arms`. Ruling D-MOTOKO-CARVEOUT-1 names this a controller-substituted remedy: a shadowed `uname` is a simulated host, not glm's "Linux host". This revision ships NO non-Darwin count; the row is retained only as the trail for why the 4 UNINFORMATIVE lines (762, 980, 1013, 646) match the 4 host-determined arms |

### Round-3 rows (worktree `.wt-motoko-iter39-armcount`, 2026-09-08)

| Claim | Command | Observed |
|---|---|---|
| Both trees, and where they differ | `git show 878939117:tools/eval/test_motoko_connection_probe.sh \| wc -l`; `wc -l tools/eval/test_motoko_connection_probe.sh`; `git diff 878939117 bd4ee3d87 -- tools/eval/test_motoko_connection_probe.sh \| grep '^@@'` | `1212`; `1232`; one hunk `@@ -1209,4 +1209,24 @@` — lines 1-1211 identical, so base line numbers hold on the sprint tree up to the gate |
| **Reviewed range: the whole file** (astra) | full read of base lines 1-1212 plus the per-class greps below | not a sample |
| Self-re-exec class: 18 `$0` hits, 14 are re-execs (astra) | `grep -nE '\$0\|BASH_SOURCE'` on base | `4:script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)`; `958: env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash "$0"`; `1007`, `1011` (`PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 ... /bin/bash "$0"`); `1035, 1038, 1041, 1045, 1057, 1063, 1074, 1078, 1084, 1122` (`PROBE_SELFTEST_DERIVATION_ONLY=1 ... /bin/bash "$0"`); `1186-1189` (census). Count `18`. Same count `18` on sprint (the gate adds none) |
| Every sub-mode child exits before the tail (P1) | `grep -nE '^[[:space:]]*exit 0'` on base and sprint; `awk 'NR>=8&&NR<=14 \|\| NR>=181&&NR<=183 \|\| NR>=353&&NR<=354'` | `exit 0` at `183`, `204`, `354` only, both trees. `8:arms=0`; `11-14`: `if [[ ! "$ARM_CAP_SECS" =~ ^[1-9][0-9]*$ ]]; then echo "not ok - PROBE_SELFTEST_ARM_CAP_SECS must be a positive integer" >&2; exit 1; fi`; `181-183`: `if [[ "${PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY:-0}" == 1 ]]; then echo "REAL_LSOF containment check passed: $REAL_LSOF"; exit 0`; `353-354`: `if [[ "${PROBE_SELFTEST_DERIVATION_ONLY:-0}" == 1 ]]; then exit 0`. Line 204 sits inside the canonical-pgrep heredoc (200-206: `sleep ... while ... echo "$1" ... fi / exit 0 / EOF`) |
| Anti-recursion refusals exist | `grep -n 'refusing to recurse'` on base | `988` (`PROBE_SELFTEST_DERIVATION_ONLY`), `992` (`PROBE_SELFTEST_MEASUREMENT_FAILURE`), `996` (`PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY`) — three, each `exit 1` |
| No sourced files (astra) | `grep -nE '^[[:space:]]*(source\|\.)[[:space:]]+'` on base; `grep -c 'source'`; control `grep -c 'probe'` | `0` hits; `0`; control `73` in the same file |
| External readers: 25 files, 2 machinery (astra) | `grep -rl 'test_motoko_connection_probe' . --exclude-dir=.git \| grep -v tools/eval/test_motoko_connection_probe.sh`; `grep -n 'test_motoko_connection_probe' make/test.mk scripts/test_check_referenced_paths.sh` | `25` files (list in Conflict Surface); `make/test.mk:72: @/bin/bash tools/eval/test_motoko_connection_probe.sh`; `make/test.mk:75: @/bin/bash -n tools/eval/test_motoko_connection_probe.sh`; `scripts/test_check_referenced_paths.sh:46:check19: ; @bash tools/eval/test_motoko_connection_probe.sh` |
| Negative control for the enumeration instrument | `grep -rl 'zzq_no_such_suite_iter39' . --exclude-dir=.git` | no files, rc=1 — the instrument can return empty and does |
| The suite runs in exactly one CI job, on macOS | `grep -n 'test-launchd-drivers\|runs-on:' .github/workflows/ci.yml` | `602: run: make test-launchd-drivers`; `runs-on:` at 18/489/512/536/606/635 (`ubuntu-latest`), 392 (`windows-latest`), 589 (`macos-latest`) — the launchd job at 589-602 is the only one that reaches the suite |
| No make target pulls in `test-launchd-drivers` | `grep -nE '^[a-z-]+:.*test-launchd-drivers' make/*.mk Makefile`; control `grep -nE '^(test\|ci):' make/*.mk` | `rc=1`, no hits; control: `make/test.mk:27:test: build test-pi-extensions ...`, `make/ci.mk:11:ci: deps fmt-check ...` (neither lists it) |
| Glob gates cannot reach `tools/eval/*.sh` (astra) | `grep -n -A3 '^check-file-sizes' make/*.mk`; `grep -n 'AUTOPUSH_SHELL_SCRIPTS :=' make/*.mk`; `ls scripts/check_*.sh \| wc -l` | `make/code-health.mk:167: for file in $$(find internal cmd -name "*.go")`; `make/code-health.mk:34:AUTOPUSH_SHELL_SCRIPTS := scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh`; `13` check scripts (scopes in Conflict Surface) |
| **Determinism on the sprint tree — 3 consecutive pristine runs** (glm i) | `for i in 1 2 3; do /bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/iter39_run$i.txt 2>&1; echo "rc=$?"; done`, sequential, rig, 2026-09-08 | run1 `PASS: 60 probe self-test arms ran` rc=0 69 s; run2 `PASS: 60` rc=0 69 s; run3 `PASS: 60` rc=0 66 s. `grep -c '^ok '` → `60`, `60`, `60`; `grep -c UNINFORMATIVE` → `1`, `1`, `1`. With the controller's first-party run: 4 of 4 identical at 60 |
| **Loop inventory — every loop, which bodies emit arms** (glm ii) | `grep -nE '^[[:space:]]*(for\|while) '` on base; bodies read at 455-481, 559-606, 620-647, 880-916, 944-955 | 20 loop headers (`grep -cE` → `20`). Exactly ONE body emits arms: `559:for dependency in dig lsof pgrep jq ailang-stub; do` → `563: expect_failure "dependency gate rejects missing $dependency"` — 5 literals, fixed. `585`/`599` (`for retained in treatment.driver.log treatment.lsof control.driver.log control.lsof`) assert-or-exit and are FOLLOWED by one `pass_arm` (591, 604). `894` (inside `run_lane_fixture_arm`, marker list) sets a flag; one `pass_arm` follows at 914. `946` (`for cap_expect in ...`) asserts; one `pass_arm` follows at 954. `455` (`for tool in awk bash ...`) builds symlinks, no arm. The remaining loops (58, 63, 68, 72, 115, 124, 173, 201, 232, 482, 500, 554, 628, 806) are pid/wait/argument loops with no arm-emitting call. No arm-emitting loop iterates a process count, directory listing, or other environment-derived cardinality → glm's conditional does not fire |
| **Attribution: 60 runtime arms from 61 static sites** (glm ii) | `grep -nE '^[[:space:]]*(expect_failure\|expect_success\|assert_measurement_failure\|run_lane_fixture_arm\|pass_arm) ' \| grep -v '()'` on base; `awk 'NR>=918&&NR<=945'` | `18 pass_arm` (4 helper-internal: 384, 400, 914, 1131; 14 top-level), `34 expect_failure`, `4 expect_success`, `3 assert_measurement_failure`, `2 run_lane_fixture_arm`. Line 929 `expect_failure "synthetic hang for the report path"` is inside `927:cap_report=$( {` … `931:} 2>&1 )` — a subshell, by design ("Drive it through expect_failure in a SUBSHELL", comment at 924); `grep -c 'synthetic hang' /tmp/iter39_run1.txt` → `0`. Sum on Darwin with the gate: 14 + (34 − 1 + 4) + 4 + 3 + 2 = 60 = measured |
| **P2: the loopback arm is skipped on the rig** | `awk 'NR>=612&&NR<=647'` on base; `grep -n UNINFORMATIVE /tmp/iter39_run1.txt`; `grep -c 'live synthetic child' /tmp/iter39_run1.txt` | `612:if [[ $(uname -s) == Darwin ]] && command -v nc >/dev/null && command -v lsof >/dev/null; then` … `640: pass_arm "live synthetic child connection is sampled and classified"` / `642: echo "UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer; fixture arm remains authoritative"` / `646: echo "UNINFORMATIVE UNDER SANDBOX: live synthetic socket arm requires darwin nc+lsof; ..."`. Run output line `34:UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer; ...`; label count `0`. Same in runs 2 and 3 |
| **P2: the loopback arm is skipped on the CI macos-latest runner too** | `gh run list --workflow ci.yml --branch dev --limit 3`; `gh run view 34163313150 --json jobs`; `gh run view --job 101869378522 --log \| grep -E 'probe self-test arms\|UNINFORMATIVE'` | run `34163313150` (dev `b8ee24580`, success, 2026-09-07T21:29Z); job `101869378522` (`launchd drivers (bash 3.2)`); log: `UNINFORMATIVE UNDER SANDBOX: loopback socket sampling yielded no peer; fixture arm remains authoritative` then `PASS: 59 probe self-test arms ran`. Both gating hosts agree: 59 pre-gate, so 60 with it |
| The gate-bearing commit has not itself run in CI | `git ls-remote --heads origin 'sprint/motoko-iter39-armcount*'`; `gh run list --workflow ci.yml --branch sprint/motoko-iter38-changelog-arms` | no such remote head; no runs. The CI evidence above is therefore for the pre-gate file; the first CI run of the gate happens when row 6s lands |
| **Container runtimes absent — glm remedy 1 unavailable** (glm iii) | `for c in docker podman colima lima nerdctl orb git; do command -v "$c" \|\| echo "$c: absent"; done; docker info; echo rc=$?` | `docker: absent`, `podman: absent`, `colima: absent`, `lima: absent`, `nerdctl: absent`, `orb: absent`; `git: present (/opt/homebrew/bin/git)`; `docker info` rc=`127` |
| Proposed round-3 lines match 0 of all 7 gate patterns | gate block saved to `/tmp/iter39_r3/gate_r3.sh`; `/usr/bin/grep -Ec` per pattern | `instrument_failure "` 0; `\|\| usage$` 0 (control: `5` on the probe); `echo "process-tree discovery` 0; `PROBE_TIMEOUT_SECS=[0-9]+` 0; `bound_secs ` 0; `PROBE_MAX_TREE_NODES=[0-9]+` 0; `PROBE_MAX_TREE_NODES=` 0; control `grep -c expected_arms` → `4` |
| `pass_arm` call-site counts on the sprint tree (criterion 5) | `grep -c 'pass_arm ' ...`; `grep -n 'pass_arm ' ... \| grep -vc ':[[:space:]]*#'` | `20` bare / `19` executable (18 at base + the gate's own). The round-3 comment block adds no `pass_arm ` token, so both counts are unchanged under M1b |
| Non-Darwin refusal, exercised (Test plan row 5) | scratch copy of the sprint file with the round-3 gate (`/tmp/iter39_r3/probe_r3.sh`, `bash -n` ok), run with `PATH=/tmp/iter39_r3/shadow:$PATH PROBE_UNDER_TEST=<probe>`; control in the same call: shadowed `uname -s` → `Linux`, `/usr/bin/uname -s` → `Darwin` | control line printed `shadow uname -s => Linux; /usr/bin/uname -s => Darwin`; stderr `UNVERIFIED host: arm-count gate skipped`; `rc=1`; `grep -c '^PASS:'` → `0`; 55 `ok` lines; 4 `UNINFORMATIVE` lines (646 "requires darwin nc+lsof", 762, 980, 1013) — the refusal is loud and withholds the verdict. This exercises the branch; it measures no Linux count |
| Non-Darwin refusal mutant (Test plan row 5) | same, with the `exit 1` after the `UNVERIFIED host` echo replaced by `:` (`diff` → line 1237 only) | stderr still prints `UNVERIFIED host: arm-count gate skipped`, then `PASS: 55 probe self-test arms ran`, `rc=0` — the refusal went silent and the suite passed on an unmeasured host. Mutant KILLED by criterion 6 |
| Round-3 gate shape on plain Darwin | `PROBE_UNDER_TEST=<probe> /bin/bash /tmp/iter39_r3/probe_r3.sh` | `PASS: 60 probe self-test arms ran`, `rc=0`, 60 `ok` lines, one `UNINFORMATIVE` line (642, "yielded no peer") — the if/else re-shaping leaves the Darwin path identical to the landed gate |

## Quorum verification log

**Surfaces per round** (protocol requirement from round 3 on — where each round's objections
landed, so the loop can tell a doc that needs splitting from one that needs revising):
- Round 1: gate shell (self-reference, dead guard), Conflict Surface proof, Host-scoping — one
  file, one gate.
- Round 2: entirely the EVIDENCE surface (Verification Log instruments: local `sed` vs global
  `grep`; single vs repeated measurement; derived vs measured count; grep-census vs full-file
  audit) — no objection touched the Darwin gate's logic.
- Round 3 (this revision): Verification Log (astra's audit; glm i, ii) and the non-Darwin branch of
  the gate shell (glm iii). Still one file, one gate; nothing here indicates a split.

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

**Round 3 — revision under attended ruling D-MOTOKO-CARVEOUT-1 (2026-09-07); goes to a fresh
independent quorum.** An independent evaluator failed the round-2 iteration 68/100 on the
carve-out call, and the owner ruled that a partially applied audit and a controller-substituted
remedy do not qualify as verbatim narrow refinements. What round 2 left unresolved, per reviewer,
and how this revision resolves it — with the reviewer's own remedy in each case:

- **gpt6-astra** — *Left unresolved by round 2:* only the cosmetic first sentence of astra's fix was
  applied; the full-file audit was filed as an open residual and never run. *Resolved in round 3, by
  astra's remedy:* the audit was run by the controller, first-party, at base 878939117, over the
  whole file (reviewed range 1-1212, not a sample), tracing the path alias (line 4), sourced files
  (none), invoked helpers and self-invocations (14 `/bin/bash "$0"` re-execs in three sub-modes),
  external readers by name (25 files, 2 machinery) and glob gates (6 that scan by glob, none in
  scope). Every row carries its command, output and a known-positive control. Conflict Surface is
  rewritten from that inventory; the additional reader class found (self-re-exec) is recorded with
  whether the insertion changes its inputs or results (it does not) and with the precondition it
  imposes (P1). Astra's second question — why the census and zero-arm guard cannot detect runtime
  arm drift — is answered at the end of Conflict Surface. The prior objection is marked resolved
  only now that the audit is recorded, per astra's own wording.
- **gemini-3-1-pro** — *Left unresolved by round 2:* nothing; the global `grep -nw arms` census
  with two controls was gemini's own remedy and is KEPT unchanged. *Round 3:* the row was re-run on
  the sprint tree and annotated — the base result stands (one assertion at 1208), and on the sprint
  tree the gate at 1227 is now the second and only other assertion, which is the intended state.
- **oc-glm-5-2** — *Left unresolved by round 2:* (i) and (ii) were applied and are kept; (iii) was
  answered with a THIRD option glm never offered — a PATH-shadowed `uname` simulating Linux, whose
  count (56) was adopted as measured. *Resolved in round 3, by glm's remedy 2:* glm's remedy 1 (a
  real Linux host) is measured unavailable on this rig (six container runtimes absent, `docker info`
  rc=127, control `git` present), so the non-Darwin branch is now glm's explicit refusal, in glm's
  own shape: `else echo 'UNVERIFIED host: arm-count gate skipped' >&2; exit 1; fi`. The design
  ships NO non-Darwin count; the shadowed-`uname` row is demoted to supporting context and is the
  basis for no number. The consequence (rc=1 on a non-Darwin host; no CI leg affected because the
  only job that runs the suite is `macos-latest`) is stated in Host-scoping. In addition, (i) was
  re-measured on the sprint tree (3 consecutive runs, 60/60/60) and (ii) was corrected: only the
  559 loop's body emits arms, the other loops round 2 named are assertion loops followed by one
  arm; the 34-site vs 37-arm `expect_failure` gap is fully attributed (5-literal loop +4, line-929
  subshell −1). One environment-CONDITIONAL arm outside any loop was found (line 640, loopback
  socket) and is measured absent on both gating hosts; it is stated as precondition P2 and put to
  this quorum rather than silently absorbed.

## Acceptance criteria

1. `bash tools/eval/test_motoko_connection_probe.sh` (Darwin) → `PASS: 60 probe self-test arms
   ran`, rc=0.
2. Delete one `pass_arm` line, run → `not ok - arm-count drift: suite ran 59 arms...`, rc=1.
3. Add one `pass_arm "extra"` line before the gate, run → `not ok - arm-count drift: suite ran 61
   arms...`, rc=1.
4. Delete the gate's OWN `pass_arm` line, run → `not ok - arm-count drift: suite ran 59 arms...`,
   rc=1 (objection 1 regression).
5. `grep -n 'pass_arm ' tools/eval/test_motoko_connection_probe.sh | grep -vc ':[[:space:]]*#'` → `19` (18 + the gate's own arm).
   The bare `grep -c` counts the gate's own explanatory comment as well as its call, returning 20; the criterion is about executable call sites, so the command must exclude comment lines (measured: 20 bare / 19 executable / 18 at base).
6. Non-Darwin path (glm remedy 2): with `uname -s` returning anything but `Darwin`, the suite
   prints `UNVERIFIED host: arm-count gate skipped` on stderr, exits 1, and prints NO `PASS:` line.
   Deleting the `exit 1` in the `else` branch makes it exit 0 with `PASS: 55 ...` — that mutant must
   be killed. The pass criterion is the refusal, not any count.
7. `git status --porcelain` clean after restoring the tree; `sha256sum` byte-identical to base.
