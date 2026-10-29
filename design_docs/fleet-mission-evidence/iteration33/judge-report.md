# Judge report: driver:exit-path-notices-unbounded (fleet i33)

Judge worktree /Users/voightkampff/.ailang-worktrees/fleet-i33-judge @ 5c8ac0daf, base 06d524c59.

## Verdict: PASS, 90/100

| Rubric area | Score | Note |
|---|---|---|
| Acceptance criteria met | 36/40 | AC1 (full `make test-launchd-drivers`) UNMEASURED; the notify suite has 2 failures that are pre-existing at base |
| Correctness / robustness | 25/25 | helper reads the right output, non-fatal, bounded, bash 3.2 and set -u safe |
| Tests / mutation strength | 19/20 | 5 of 6 drills killed by the intended arm; M6 stopped by time limit |
| Scope / text preservation / docs | 10/15 | all in allowlist, text preserved; report does flag the 2 pre-existing fails; deduction for unmeasured full make target and G2 note below |

## Blocking findings
None.

## Non-blocking findings
1. `test_driver_notify.sh` exits rc=1 (113 pass, 2 fail) in the judge tree. The failing arms are `lane sets the digest routing note` and `no routing note when healthy` (`_mc_reset_hint: command not found`). The identical 2 fail on a pristine `git archive 06d524c59` copy (96 pass, 2 fail), so they are pre-existing and unrelated. The sprint report mentions them. `make test-launchd-drivers` therefore cannot be green under the plan's AC1 wording until they are fixed (follow-up).
2. Unbounded sends remain outside the mission driver path: `nightly-eval.sh`, `nightly-lang-eval.sh`, `cron-kicker.sh`, `os-rotation-filler.sh`, `rig-watchdog-reconcile.sh` (plan already files `nightly-eval:unbounded-sends`). `tools/launchd/lib/*.sh` contains no `messages send` / `issue comment` calls. mission-control.sh lines 240 (spool drain), 299 and 353 are inside `_mc_bounded` calls (wrapped on the previous line); 334, 368 and the slot-notify gh are bounded.
3. Worst case wall clock, both channels hung, default NOTIFY_TIMEOUT=30: per call about 30 + 2 (poll granularity) + 2 (grace) + up to 2 (post-kill wait) = ~36 s, so ~72 s per path. Acceptable for an exit path, but longer than the plan's implied ~64 s.
4. Titles A/B/C are not date-stamped, so a prod duplicate is refused and only logged as "already in the inbox" (documented follow-up `driver:exit-notice-titles-undated`). Net effect: after the first-ever send each distinct title is effectively delivered once. Note in changelog, fine.
5. Mutation-drill scratch copies in /tmp make the unrelated arm `wiring: pi controller loads controller-bash-cap.ts` fail (path relative to driver dir); this is an artifact of the copy location, not of the change.

## Per-AC table
| AC | Result | Evidence |
|---|---|---|
| 1 make test-launchd-drivers green unpiped | UNMEASURED (not run, ~10 min); component suites below | notify suite rc=1 only for 2 base-pre-existing fails |
| 2 `bash -n` | PASS (all four suites loaded and ran; driver ran under 3 suites) | |
| 3 G1/G2 zero violations, selected>=6 | PASS | `G1 ... (selected 7, need >=6)`, G2 PASS in /tmp/fleet_i33_judge_test_driver_notify.log |
| 4 E1-E3 <=14 s, rc 1/143/1 | PASS | E1-E3 PASS in log (and killed by M1/M4) |
| 5 gate0 suite green, F8 | PASS | scripts/test_mission_gate0_self_notices.sh rc=0, 108 passed 0 failed; `grep -c -F` of the F8 literal in the driver = 1 |
| 6 no title/body/exit change | PASS | manual diff review: all three titles and message bodies, GH bodies (incl. the multi-line `_ref_reserve` body and F8 literal) are byte-identical; `exit 1` / `exit "$RC"` unchanged; only `2>/dev/null` removed |
| 7 scope | PASS | changed paths: .ailang/state/sprints/*, changelogs/unreleased/*, design_docs/planned/*, tools/launchd/{mission-control,test_controller_capacity,test_driver_notify,test_mission_heartbeat}.sh. All inside allowlist |
| 8 red->green evidence | PASS | red log and green log in .ailang/state/sprints; independently re-shown by my mutation drills |

Other suites (unpiped, /bin/bash): test_controller_capacity rc=0 (15 cases, fail=0); test_mission_heartbeat rc=0 (25 arms); gate0 rc=0.

## Item 2/3/4 analysis
- Every send/comment in mission-control.sh is bounded (G1 source guard selected 7 logical lines, all contain `_mc_bounded`; independently confirmed by grep).
- F8 literal appears exactly once, as the 4th argument of the rc-fail `_mc_exit_notice` call.
- Dropping `2>/dev/null`: `_mc_bounded` redirects the child to a temp file (`>"$out_f" 2>&1`), so nothing reaches the driver stderr; output surfaces only in the tailed log WARNING (300 bytes).
- `_mc_notice_suppressed` reads the global `MC_BOUNDED_OUT`, which is set by the send just executed (in the failing branch, `_out` is a copy taken from the same variable before the call). It therefore sees THIS send. On timeout (rc 124) the partial output is in MC_BOUNDED_OUT; the "duplicate message exists" test would not match an empty output, correctly logging a WARNING.
- MISSION_GH_ISSUE unset: `${MISSION_GH_ISSUE:-}` returns 0 before the gh call; set -u safe. `MSG_FROM`, `MISSION_REPO` are defined at driver top (MISSION_REPO default at :74); NOTIFY_TIMEOUT at :918, before the first call site (:2058+). `local`, `${var:+}`, no bash 4 features; ran under /bin/bash 3.2.
- Store scoping: `env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=...` per command; AILANG_STORAGE is never exported. Matches CLAUDE.md (per-store override, not the plane switch) and `_mc_notify`. It does change where these notices land (local SQLite -> prod Firestore) which is correct and is disclosed in the changelog. Risk: a second consumer expecting local inbox for these titles would no longer see them; none found.
- Slot-verdict block gets the same scoping; heartbeat regex updated accordingly (suite green).

## Mutation drills (scratch copies in /tmp/fleet_i33_mut, via MC_NOTIFY_DRIVER; tracked files untouched)
| Mutation | rc | Killing arms |
|---|---|---|
| M1 helper's AILANG send made bare (no `_mc_bounded`, no env) | 1 | E1, E2, E3 (hang), E7a-c, E9, G1, G2 |
| M2 drop `env AILANG_STORAGE_MESSAGING=...` prefix in `_mc_exit_notice` | 1 | E7a, E7b, E7c, G2 |
| M3 drop `⚠️ ` from F8 gh body prefix | 1 | E6, G3a |
| M4 helper's gh call made bare | 1 | E1, E2, E3, G1 |
| M5 alter late-kill title | 1 | E5 |
| M6 drop env scoping on SLOT NOTIFY send | UNMEASURED | run cut by the tool time limit before the summary; not counted |

(All mutant runs also show the 3 base/copy-path artifact fails noted above.)

## Commands run
- `git diff 06d524c59 5c8ac0daf` (+ --name-only), grep for `ailang messages send|gh issue comment` in tools/launchd and lib
- `/bin/bash tools/launchd/test_driver_notify.sh > /tmp/fleet_i33_judge_notify.log` (logs: /tmp/fleet_i33_judge_test_*.log) -> rc=1 (2 pre-existing)
- same for test_controller_capacity.sh, test_mission_heartbeat.sh, scripts/test_mission_gate0_self_notices.sh -> rc=0 each
- `git archive 06d524c59 | tar -x -C /tmp/base33; /bin/bash tools/launchd/test_driver_notify.sh` -> 96 pass, 2 fail (same two)
- python3 string-replace to produce /tmp/fleet_i33_mut/m1..m6.sh; `MC_NOTIFY_DRIVER=<copy> /bin/bash tools/launchd/test_driver_notify.sh > m*.log`
- `grep -c -F '⚠️ Mission iteration **FAILED to complete** (rc=' tools/launchd/mission-control.sh` -> 1
