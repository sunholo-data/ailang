# Sprint plan: pi-runner pre-dirty verdict (fleet iteration 20)

**Ticket:** `pi-runner:verdict-blind-to-commits-and-predirty`, the **pre-dirty half only**. The
commit-blind half landed in #1329 (`tools/launchd/test_mission_pi_run_commits.sh`).
**Lane:** harness fix inside fleet write scope (`scripts/mission_*`, `scripts/test_mission_*`,
`make/test.mk`, `changelogs/`; D-FLEET-5). Mechanical, not routing/quota/billing policy.
**Base:** origin/dev `a12a319b5`. **Size:** one milestone, about 1–2 h.

## Problem (verified at a12a319b5)

`scripts/mission_pi_run.sh` grants `ok` (rc 0) when `DIFF_LINES > 0 || COMMITS > 0` (~l.408), where
`DIFF_LINES=$(git status --porcelain | wc -l)` is read **after** the run (~l.346). `BASE_HEAD` is
captured before launch (~l.247), but nothing records the worktree's state at that point. A tree that
is already dirty when pi starts therefore reads `ok` even if pi did nothing. That is the false green
the script exists to prevent, and l.347 says so itself ("a pre-dirty tree still reads ok vacuously").

**The existing suite depends on the bug.** `scripts/test_mission_pi_run.sh` TEST 1 (happy path) and
TEST 5 (slow-but-working) both build the repo with `mkrepo … dirty` and use a stub that writes
nothing, yet expect `ok`. They pass only because of the pre-dirty hole. After the fix they become
rc 10, so their stubs must do real work (see M1 step 5). This is the strongest evidence that the hole
is real.

## Design

### Pre-run snapshot

Add one function next to `mtime_of` and take the snapshot **right after `BASE_HEAD`**, before `set -m`
and before the launch:

```bash
# Content fingerprint of the worktree relative to its CURRENT HEAD: what porcelain lists,
# the bytes of every tracked change, and the bytes of every untracked file. rc!=0 = no git.
worktree_fingerprint() {
  ( cd "$1" || exit 1
    git rev-parse --git-dir >/dev/null 2>&1 || exit 1
    { git status --porcelain=v1 -z --untracked-files=all
      printf '\0--diff--\0'
      if git rev-parse -q --verify HEAD >/dev/null; then
        git diff --binary --no-ext-diff --no-textconv HEAD
      else   # unborn branch (commits case E): no HEAD to diff against
        git diff --binary --no-ext-diff --no-textconv --cached
        git diff --binary --no-ext-diff --no-textconv
      fi
      printf '\0--untracked--\0'
      git ls-files -o --exclude-standard -z | xargs -0 git hash-object --
    } | git hash-object --stdin )
}
PRE_FP=$(worktree_fingerprint "$WORKDIR") || preflight_fail 14 launch_failed "cannot fingerprint worktree before launch"
PREDIRTY_FILES=$(git -C "$WORKDIR" status --porcelain 2>/dev/null | wc -l | tr -d ' ')
```

- **Binary-safe and portable.** Use `git diff --binary`. The final digest is `git hash-object --stdin`,
  so the function needs no `shasum` or `sha256sum` and behaves the same on darwin and on the CI Linux
  legs. The whole stream is piped and never held in a shell variable. All of this runs under
  bash 3.2: no arrays, `declare -A`, `mapfile` or `${v,,}`.
- **Bounded cost.** `--exclude-standard` drops ignored trees such as `node_modules` and build output,
  so the cost is one pass over tracked changes plus one hash of each untracked file, done twice.
  `xargs -0 git hash-object --` with no input prints nothing on both BSD and GNU xargs (checked on the
  rig, see below).
- **Diff against `HEAD`, not `BASE_HEAD`.** This keeps commits case H green: a clean tree followed by
  `git reset --hard HEAD~1` stays byte-identical relative to its own HEAD, so it still reads rc 10.
  Commits are judged only by the existing `COMMITS` path.
- **Prototype checked on the rig** (bash 3.2.57, BSD xargs, scratch repo): the clean tree gave the same
  fingerprint twice; editing a file that was already dirty changed it; editing an untracked file
  changed it; `rm -rf .git` gave rc 1.

### Post-run comparison and verdict

At ~l.346, keep `DIFF_LINES` as it is: it is the post-run porcelain count, and `gate-3-route.md` and
`test_mission_pi_run_provider_quota.sh` read it as `worktree_changed_files`. Then add:

```bash
WORKTREE_CHANGED=false
POST_FP=$(worktree_fingerprint "$WORKDIR") && [ "$POST_FP" != "$PRE_FP" ] && WORKTREE_CHANGED=true
```

- If the post-run fingerprint fails, for example because pi ran `rm -rf .git` (commits case I),
  `WORKTREE_CHANGED` stays `false`. That matches today's porcelain-0 reading, so case I stays rc 10.
- Change the `ok` condition (~l.408) to
  `[ "$WORKTREE_CHANGED" = true ] || [ "$COMMITS" -gt 0 ]`. **`DIFF_LINES` no longer grants `ok`.**
- On a clean tree this matches today's behaviour: a clean tree has a fixed fingerprint, so any dirt
  changes it, and an edit that pi then reverts reads rc 10, as it does today.

### Decision: reverting a pre-existing dirty edit reads `ok`

If pi runs `git checkout -- f.txt` on a tree that was dirty before launch, the tree really did
change, so the verdict is `ok`. The reasons:

1. This guard exists to catch runs where **nothing happened**. Judging whether a change was useful is
   the evaluator's job.
2. Telling a revert apart from other work would mean storing per-file baselines and reasoning about
   intent, which is more complexity for a case that has never been measured.
3. Readers can still see the case: `predirty_files > 0`, `worktree_changed_since_start: true` and
   `worktree_changed_files == 0`.

The same holds for `reset --hard` on a pre-dirty tree. Arm 7 pins this decision.

### Verdict JSON and log line

- Add `"predirty_files": $PREDIRTY_FILES` (the pre-run porcelain count) and
  `"worktree_changed_since_start": $WORKTREE_CHANGED` (a boolean) next to `worktree_changed_files`.
- Keep `worktree_changed_files` as the post-run porcelain count. Its consumers read it on
  `wall_timeout` to see partial work.
- In the closing `echo`, add `, N pre-dirty` and `changed since start: true/false`.

### Header and comment edits

- **Step 5:** "Asserts the worktree CHANGED relative to a pre-launch content fingerprint, or that
  commits were made since launch. A tree that is already dirty at launch does not count as work."
- **Exit 0:** "pi finished, and either the worktree content differs from its pre-launch fingerprint or
  commits were made since launch."
- **Exit 10:** "pi finished, the worktree is byte-identical to its pre-launch state (dirty or clean),
  and no commits were made. This is the false green in its pure form."
- **l.347:** delete "a pre-dirty tree still reads ok vacuously" and replace it with a pointer to the
  fingerprint ("pre-dirty: judged against PRE_FP, see worktree_fingerprint").

### Out of scope

- An `--out` or `--verdict` path inside `--workdir` would itself change the fingerprint. Porcelain
  already had the same problem, and every caller (gate-3-route, mission-lane-check) uses `/tmp`.
- Ignored files are not counted. This matches porcelain today.

## Milestone M1: fingerprint verdict, arms, wiring

1. Add `worktree_fingerprint`, the `PRE_FP` and `PREDIRTY_FILES` capture, the post-run comparison, the
   new `ok` condition, the two JSON fields, the log-line text and the header edits, all in
   `scripts/mission_pi_run.sh`.
2. In `scripts/test_mission_pi_run.sh`, add a `field()` helper. It can be the jq one-liner
   `jq -r ".$2" "$1"` (jq is already a dependency) or the sed form from the commits suite. Then add the
   arms below, using the existing `mkstub` and `mkrepo` fixtures plus `--max-seconds 30 --stall-seconds 10`.
3. Extend `mkrepo` with a third mode, `dirty-untracked`, which commits `f.txt` and leaves an untracked
   `u.txt` containing `u0`. This is needed for arm 6.
4. Export `MISSION_PI_POLL_SECONDS=1` at the top of the test, as the commits suite does, so the new arms
   finish in about 1 s each instead of about 3 s.
5. **Fix TEST 1 and TEST 5.** Their stubs must write a file, for example `echo work > work.txt;`
   before the `printf`, so they keep asserting `ok` for honest reasons. Do this in the same commit as
   the fix, and say in the commit message that these tests passed only because of the bug.
6. Wire the suite into `make/test.mk` `test-launchd-drivers` with
   `@$(LAUNCHD_SUITE) scripts/test_mission_pi_run.sh`, placed after the three `test_mission_pi_run_*`
   lines. **Today this suite runs in no gate:** `grep -n test_mission_pi_run make/*.mk Makefile` lists
   only the three `tools/launchd/test_mission_pi_run_*.sh` suites. `make/test.mk` is fleet-allowed by
   `test_mission_scope_guard.sh`. The cost is about 45 s, mostly the stall arms (TEST 3, 4, 4b and 5).
7. Write the changelog fragment `changelogs/unreleased/2026-10-05-pi-runner-predirty-verdict.md`, a
   single section that opens with `### Fixed — pi runner verdict read ok on a pre-dirty worktree
   (2026-10-05)`. The body covers the cause (porcelain was read only after the run), the change (a
   pre-launch content fingerprint; `predirty_files` and `worktree_changed_since_start` in the verdict),
   the revert decision, and the fact that TEST 1 and TEST 5 had been passing only because of the bug.

### Test arms and the mutation that kills each one

The executor must run every mutation against a **scratchpad copy** of the script, never with
`git checkout <file>`, and confirm that the named arm goes red. Each mutation is applied alone.

| # | Arm | Expected | Mutation that kills it |
|---|-----|----------|------------------------|
| 1 | pre-dirty (`f.txt` = changed) + stub `:` (no-op) | rc 10 `empty_worktree`; `predirty_files`=1; `worktree_changed_since_start`=false | **M-a:** revert the `ok` condition to `DIFF_LINES > 0 \|\| COMMITS > 0`, making the snapshot comparison dead code. Arm 1 reads rc 0. |
| 2 | pre-dirty + stub `echo n > new.txt` (new untracked file) | rc 0 `ok`; `worktree_changed_since_start`=true | **M-b:** fingerprint = `git diff HEAD` only, with the porcelain and untracked sections dropped. The new untracked file is invisible, so arm 2 reads rc 10. |
| 3 | pre-dirty + stub `echo changed2 > f.txt` (further edit to the already-dirty file) | rc 0 `ok` | **M-c:** fingerprint = porcelain only, with the diff-content section dropped. ` M f.txt` is identical before and after, so arm 3 reads rc 10. This is the arm that justifies hashing content. |
| 4 | clean + stub `:` (control, unchanged) | rc 10 `empty_worktree`; `predirty_files`=0 | **M-d:** initialise `WORKTREE_CHANGED=true`, or put anything time-varying such as `date` or a stat-based index read into the fingerprint. Arm 4 reads rc 0. |
| 5 | clean + stub commits (`echo c > f.txt; git add f.txt; git commit`) (control) | rc 0 `ok`; `commits_since_start`=1; `worktree_changed_since_start`=false | **M-e:** drop `\|\| COMMITS > 0` from the `ok` condition. The clean, committed tree has an unchanged fingerprint, so arm 5 reads rc 10. |
| 6 | pre-dirty-untracked (`u.txt`=u0) + stub `echo u1 > u.txt` | rc 0 `ok` | **M-f:** drop the `--untracked--` hash section. `?? u.txt` is identical in porcelain and `git diff HEAD` cannot see untracked files, so arm 6 reads rc 10. |
| 7 | pre-dirty + stub `git checkout -- f.txt` (revert) | rc 0 `ok`; `worktree_changed_files`=0; `predirty_files`=1 | Pins the revert decision. Any special case that treats "now clean" as no work, or **M-a′** (`ok` requires `DIFF_LINES > 0 && WORKTREE_CHANGED`), reads rc 10. |
| J | JSON fields on arms 1 and 3 | `predirty_files` = 1 on both; `worktree_changed_since_start` false on arm 1, true on arm 3 | **M-g:** hard-code either field, or emit `DIFF_LINES` as `predirty_files`. Arm 1 would then report 1 and arm 3 would still report 1, so `worktree_changed_since_start` is the one that catches it. |

Regression guards that must stay green and are not new: commits suite cases **H** (clean + reset
`HEAD~1` → rc 10) and **I** (`rm -rf .git` → rc 10). Case H fails if the fingerprint diffs against
`BASE_HEAD`; case I fails if a failed post-run fingerprint is treated as "changed". Also
`test_mission_pi_run_sandbox.sh` and `test_mission_pi_run_provider_quota.sh` must stay green; their
stubs write before they assert, so no change is expected.

## Acceptance commands, with the pristine baseline recorded at a12a319b5 on the rig (bash 3.2.57)

| Command | Baseline (pristine) | Required after M1 |
|---------|---------------------|-------------------|
| `/bin/bash scripts/test_mission_pi_run.sh` (unpiped) | rc 0, `passed=12 failed=0` | rc 0, `passed=21+ failed=0`: 12 existing, with TEST 1 and 5 fixed, plus arms 1–7 and J |
| `make test-launchd-drivers` (unpiped) | rc 0, ends with `launchd drivers: tests + bash 3.2 syntax OK`. Does **not** include `scripts/test_mission_pi_run.sh` today. | rc 0, and the log shows the `scripts/test_mission_pi_run.sh` `passed=` line |
| `/bin/bash -n scripts/mission_pi_run.sh` / `scripts/test_mission_pi_run.sh` | rc 0 / rc 0 | rc 0 / rc 0 |
| `shellcheck scripts/mission_pi_run.sh scripts/test_mission_pi_run.sh` (`/opt/homebrew/bin/shellcheck`) | rc 1 from 7 pre-existing notes and infos: runner SC2015 l.192 and SC1091 l.250; test SC2016 ×4 and SC2015 ×1. No warnings or errors. | **No new findings** beyond those 7 (compare with `shellcheck -f gcc`). Shellcheck is not a CI gate for these files. |
| `make check-changelog` | not run for the baseline | rc 0 with the new fragment |
| Mutation table above | — | every mutation, applied alone on a scratchpad copy, turns its named arm red |

Run every gate **unpiped**, because a piped `make` reports the pipe's exit code.
