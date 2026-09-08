# Controller's residual audit — full-file reader inventory for `tools/eval/test_motoko_connection_probe.sh`

Run 2026-09-07/08 by the motoko iteration-39 controller, at base `878939117` (the base the
round-2 doc names), in worktree `.wt-motoko-iter39-armcount`. This is the audit `gpt6-astra`
demanded in quorum round 2 and which round 2 did NOT run. Every row is a command with its
observed output; empty/negative results carry a known-positive control in the same call (rule 3a).

## Reviewed ranges

The whole file, 1..1212 at base (1232 on the sprint branch with M1 applied). Not a sample.

## A. In-file readers of the suite's OWN text

| Reader | Lines | Reads | Can the insertion move it? |
|---|---|---|---|
| Wall-clock literal census | 1186-1189 | `grep -Ec/-c ... "$0"` ×4 | Patterns are `PROBE_TIMEOUT_SECS=[0-9]+` (exact `!=5`), `bound_secs ` (min `<8`), `PROBE_MAX_TREE_NODES=[0-9]+` (exact `!=1`), `PROBE_MAX_TREE_NODES=` (min `<3`). The proposed lines match **0** of the four. NO. |
| Refusal-branch gate | 1165-1184 | `grep -c ... "$probe"` ×3 | Reads the PROBE, a different file. NO. |
| `arms == 0` guard | 1208 | `$arms` runtime counter | Sits BEFORE the gate; supplies its anti-vacuity. Unaffected. |

**Additional readers found by this audit that round 2 did not name — the self-re-exec class.**
`grep -nE '\$0|BASH_SOURCE' ` returns **18** hits, of which the four census greps are only the tail.
The other **14** are the suite re-executing ITSELF as a child process:

- line 4 — `script_dir=$(... dirname -- "$0" ...)`: a **path alias**, used only to locate the probe
  (`probe=${PROBE_UNDER_TEST:-$script_dir/motoko_connection_probe.sh}`, line 5). It never reads the
  suite's text.
- line 958 — `env PROBE_SELFTEST_ARM_CAP_SECS=invalid /bin/bash "$0"`
- lines 1007, 1011 — `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1 ... /bin/bash "$0"`
- lines 1035, 1038, 1041, 1045, 1057, 1063, 1074, 1078, 1084, 1122 —
  `PROBE_SELFTEST_DERIVATION_ONLY=1 ... /bin/bash "$0"`

**Why this matters and why it does NOT break the gate — the child terminators, measured:**

| Sub-mode | Terminator | Line | Reaches the gate? |
|---|---|---|---|
| `PROBE_SELFTEST_ARM_CAP_SECS=invalid` | `exit 1` after the `^[1-9][0-9]*$` test | 12-14 | no |
| `PROBE_SELFTEST_LSOF_CONTAINMENT_ONLY=1` | `exit 0` | 181-183 | no |
| `PROBE_SELFTEST_DERIVATION_ONLY=1` | `exit 0` | 353-354 | no |

`grep -nE '^[[:space:]]*exit 0'` returns exactly **3** hits — 183, 204, 354 — i.e. every
`exit 0` in the file is a sub-mode terminator or the fixture heredoc at 204; **no** child path
falls through to line 1208. The suite additionally carries explicit anti-recursion refusals at
lines 987-988 and 995-996 ("… leaked into the arm section; refusing to recurse"), so a leaked
sub-mode variable reds loudly rather than reaching the tail.
**This is the load-bearing new fact: a child re-exec increments its OWN `arms` from 0 and would
red an exact-equality gate — it does not, only because all three sub-modes exit before the tail.**
The design MUST state this dependency, because a future sub-mode added without an early exit
would red the arm-count gate for a reason unrelated to arm drift.

`grep -nE '^[[:space:]]*(source|\.)[[:space:]]+'` → **0 sourced files**; control
`grep -c 'source'` → **0** repo-wide in this file, so the suite includes nothing.

## B. EXTERNAL readers — complete enumeration, not truncated

`grep -rl 'test_motoko_connection_probe' .` (excluding `.git/` and the file itself) → **25 files**.
Twenty-three are prose (design docs, mission logs, changelogs, sprint JSONs, a retro). The two
that are machinery:

| File | Refs | What it does | Can the insertion move it? |
|---|---|---|---|
| `make/test.mk` | 2 | line ~72 `@/bin/bash tools/eval/test_motoko_connection_probe.sh` (runs it, inside `test-launchd-drivers`), line ~76 `@/bin/bash -n …` (syntax check) | Runs it / parses it. Not count-based. NO. |
| `scripts/test_check_referenced_paths.sh` | 1 | line 46 fixture `check19: ; @bash tools/eval/test_motoko_connection_probe.sh` | Asserts the PATH exists and is tracked; never reads contents. NO. |

`.github/workflows/ci.yml:602` runs `make test-launchd-drivers`, so the suite runs in CI on every
push, on `macos-latest`. Negative control: `grep -rc 'zzq_no_such_suite_iter39'` → no file, so the
enumeration instrument can return empty and does.

## C. Gates that would catch the file by GLOB rather than by name

| Gate | Scope, measured | Verdict |
|---|---|---|
| `make check-file-sizes` | `for file in $(find internal cmd -name "*.go")`, cap 800 | **Go only, `internal`/`cmd` only.** `tools/eval/*.sh` is out of scope. NO. |
| `make shellcheck-autopush` | `AUTOPUSH_SHELL_SCRIPTS := scripts/hooks/push_dev_on_stop.sh scripts/hooks/test_push_dev_on_stop.sh` | Two named files. NO. |
| `scripts/check_context_docs.sh` | `git ls-files`, then CLAUDE.md / `.claude/rules` / `SKILL.md` | Not a context doc. NO. |
| `scripts/check_no_personal_email.sh` | `git ls-files \| grep -E "$SCOPE_RE"` | Pattern-refusal on emails, not a count. Insertion adds none. NO. |
| `scripts/check_tmpfile_hygiene.sh` | `MK_ROOT` makefiles, `MK_FILES_EXPECTED=12` | Makefiles. NO. |
| `scripts/check_home_isolation.sh` | `SCAN_ROOT=$REPO_ROOT`, pattern-refusal | Insertion touches no `$HOME`. NO. |
| remaining `scripts/check_*.sh` | 13 enumerated total; the 6 above are all that scan by glob | NO. |

## D. Verdict

**Additional readers found: 1 class (the 14 self-re-exec sites + 1 path alias at line 4), plus 2
external machinery references and 0 glob gates.** None has its inputs or results moved by the
proposed insertion. The self-re-exec class does not move a count but does establish a
**precondition** the design must state: every sub-mode child exits before line 1208.
Exhaustiveness is now supported by a full-file review with per-class commands and controls,
rather than by four `$0` greps.

## E. The question astra also asked

*"Explain why the existing census and zero-arm guard cannot provide runtime arm-count drift
detection."* The wall-clock census counts LITERALS in the file's text (`PROBE_TIMEOUT_SECS=…`,
`bound_secs `), so it is blind to whether an arm EXECUTED; deleting a `pass_arm` line moves none
of its four patterns. The `arms == 0` guard tests a single boundary — it distinguishes "nothing
ran" from "something ran" and says nothing about *how much* ran, which is exactly the 59→58
transition measured as the defect. Neither reads the runtime counter against an expected value.

## F. glm's remedy 1 is measured UNAVAILABLE on this rig

`docker`, `podman`, `colima`, `lima`, `nerdctl`, `orb` — all **absent**; `docker info` fails; the
control (`git`) is present, so the probe can see a positive. There is no Linux host reachable from
this loop, so of glm's two named remedies only the second — the explicit `UNVERIFIED host` gate —
is available. This is a choice BETWEEN the reviewer's own two options on measured availability,
not a third option.
