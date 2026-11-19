# executor round 2 — fleet i109 pi-run orphan reap (base 7dbba8b4d)

Started 2026-11-12. Steps appended as they complete.

## Step 1 — unit arms for F, C, B, G (in progress)
Added in tools/launchd/test_mission_pi_run_orphan_reap.sh: fixture mode `deaf` (ignores TERM, no child);
arms U6 (F), U7 (C), U8 (B), U9 (G). Overrides: U7 stubs `ps -A` as a shell function returning a canned table;
U8/U9 override `_pr_birth` (U9 by call count: 1st call real = TERM pass, 2nd call bogus = KILL pass).
Baseline with unmutated lib: passed=15 failed=0 (11 old + 4 new), rc 0.

| mutant | change | killed by (designated) | other reds | suite |
|---|---|---|---|---|
| F | drop `b[root]==rootb` | U6 | none | rc 1, 14/15 |
| C | drop "snapshot lists $$" | U7 | none | rc 1, 14/15 |
| B | drop pre-TERM birth recheck | U8 | U9 (its call-count assumes the TERM-pass call exists) | rc 1, 13/15 |
| G | drop pre-KILL birth recheck | U9 | U8 (AH8 is then KILLed at the post-grace pass) | rc 1, 13/15 |

Mutant scratch copies: /tmp/fleet109/exec/mut2/m<N>, driver /tmp/fleet109/exec/mutate2.sh, output mut2-FCBG.txt.
Step 1 DONE.

## Step 2 — remove dead post-grace _pr_ledger_update (in progress)
Step 2 DONE: removed the post-grace `_pr_ledger_update` from `_pr_reap`; header + `_pr_reap` comment now
say why (the closure reaches a grace-born child through its live parent). U5 renamed and re-described
("child of a live member, forked after TERM, is reached and KILLed"); it still passes via the closure.
Plan mutant 7 (delete post-grace refresh) is now not a mutant: the line no longer exists.

Step 3 DONE: `_pr_reap` prints `WARNING: pi reap could not read the process table; nothing could be
reaped, ...` to stderr when the reap-time snapshot fails; a failed post-TERM re-read prints a second,
distinct WARNING (only when something was TERMed). Arms: U10 (ps -A stubbed to fail; plus unstubbed
positive control with no WARNING) and U11 (3rd `ps -A` fails: post-TERM WARNING, TERMed fixture reaped).

Step 4 DONE: `_pr_exe PID` = `ps -o comm=` (executable name, trailing blanks trimmed, 120 chars max),
used for both TERM and KILL rows. Key `command` kept. Runner header item 6, changelog fragment updated.
Arms: U12 (argv secret absent from the row; row == `perl`), R1 now asserts the child row is `perl` and
contains no `hostile`. Bug found on the way: comm= output carried a trailing blank ("perl "), fixed in _pr_exe.

## Step 5 — mutant table on the FINAL code (driver /tmp/fleet109/exec/mutate2.sh; copies in mut2/m<N>)

| mutant | change | killing arm(s) | suite |
|---|---|---|---|
| F | drop `b[root]==rootb` in _pr_members | U6 (designated); no other | rc 1, 17/18 |
| C | drop "snapshot must list $$" | U7 | rc 1, 17/18 |
| B | drop pre-TERM birth recheck | U8 (designated); U9 also red (its call-count assumes the TERM-pass call) | rc 1, 16/18 |
| G | drop pre-KILL birth recheck | U9 (designated); U8 also red (AH8 is KILLed at the post-grace pass) | rc 1, 16/18 |
| 1 | delete per-poll `_pr_ledger_update` in runner | R1, R2, R3 | rc 1, 15/18 |
| 2 | wrap reap in `if OUTCOME != finished` | R1 (R0 stays green as control) | rc 1, 17/18 |
| 6 | delete KILL pass | U5, U9, R2 | rc 1, 15/18 |
| 9 | report JSON built with printf | U4, R0, R1, R2 | rc 1, 14/18 |
| W (new) | silence the unreadable-table WARNING | U10 | rc 1, 17/18 |
| E (new) | store argv again instead of comm | U12, R1 | rc 1, 16/18 |

Plan mutant 7 retired (line removed; see step 2). Baseline on the unmutated final code: 18/18.
Mutant E had first been run with a broken sed quoting in the driver (no-op, rc 0); fixed and re-run, red as shown.

## Gates (final tree, commit 0d0d73a4a)
- new suite: 18/18 rc 0 — run 1 and run 2 (gates.txt), plus a third run after a comment-only header edit (final-new.txt)
- existing: scripts/test_mission_pi_run.sh rc 0; test_mission_pi_run_{commits,sandbox,provider_quota,tool_stall}.sh rc 0, 0 FAIL lines
- git diff --check: clean
- make check-changelog: rc 0 (4 fragments valid)
- make check-file-sizes: rc 0 (limit 800; runner is 556 lines)
- Not run: make test-launchd-drivers (per instruction). Not pushed.

## Commit
0d0d73a4a fix(fleet-i109): pin the reaper's safety guards with kill arms; command = executable only
