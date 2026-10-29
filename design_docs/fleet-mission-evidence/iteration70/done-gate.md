# Iteration 70 — done-gate evidence

Ticket: `pi:provider-error:no-retry` (stapledon, filed 2026-10-07).
Branch `fleet/i70-provider-error-retry-seam`; base `96d69f05b6e84bd6666e8fa91ff948f803b67d92`; head `2c456d263241d0ea0cc028faf70a5e68bad296aa`; PR #2022.

## Reality check (controller, first-party)
- The ticket's event is the stapledon fire ending 2026-09-29T00:52Z. Its pi session (`~/.pi/agent/sessions/--Users-voightkampff-dev-sunholo-data-stapledons-godot--/2026-09-29T00-46-20-125Z_01a0eaa0-….jsonl`) ends `"stopReason":"error",…,"errorMessage":"Provider finish_reason: error"`.
- #1391 (`7eaf9bece`, 2026-09-29 09:24 +0200 = 07:24Z) added `|Provider finish_reason: error` to `TRANSIENT_SIG` (`tools/launchd/mission-control.sh:1160`). That is about 6.5 h after the event and 8 days before the ticket was filed. The driver loop (`:2552-2600`) checks `RUNTIME_QUOTA_SIG` first, then retries a `TRANSIENT_SIG` match on the same rung: `TRANSIENT_RETRIES=3` total attempts, `TRANSIENT_BACKOFF=45` × attempt (45 s, 90 s). The pi controller runs under a fixed `--session-id`; `_mc_pi_session_exists` switches `-p` to `MC_PI_RESUME_PROMPT` on a retry (`:2469-2480`).
- pi 0.85.1 print mode writes the error to stderr: `dist/modes/print-mode.js:115-117`, `console.error(assistantMsg.errorMessage …); exitCode = 1`. The driver launches pi with `>>"$LOG" 2>&1`.
- The only prior test was `tools/launchd/test_driver_notify.sh` case 13, a membership grep.
- Live retry evidence: none. `grep -c 'transient API error'` = 0 in every `/tmp/ailang-mission-*.log` (rotated), and no `verdict=RETRIED` row exists in any `~/.ailang/state/mission-*-slot-verdicts.log`. Since #1391, pi controllers ran 4 COMPLETED fires and mostly `PAUSED-NO-CAPACITY` ones, so the absence is uninformative, not a negative.
- Companion ticket `stall:watchdog:slow-on-hung-tool` (not picked): ask 1 is built by #1391 (`tools/pi-extensions/controller-bash-cap.ts` clamps every pi `bash` call to 540 s and is loaded with `-e` at driver `:2480`). Ask 2, a shorter no-progress threshold for pi controllers, changes the watchdog thresholds that D-FLEET-4/-7 fixed. Since #1391 there was one pi-controller KILLED verdict (fleet 2026-10-04T04:28Z, the ticket's own second event) and no recurrence.

## Gates
- `/bin/bash tools/launchd/test_controller_capacity.sh`: base rc 0 (15 cases); head rc 0 (16 cases). The 15 base PASS lines are unchanged.
- `make test-launchd-drivers` on head `2c456d263`: rc 0, `run-suites: all 52 suites passed`. The executor's run at `5d5d5850f` was also rc 0 with 52 suites.
- shellcheck on the test file (controller's version): base 64, head 70. All 6 added findings are SC2319, the suite's existing `expect … $?` idiom. The plan's 61/66 came from another shellcheck version; the delta matches.
- `make check-changelog`: rc 0 (executor and judge).
- `find ~/.pi -maxdepth 4 -name '*ctl-capacity*'`: empty.

## Controller dry-runs (world: idle, armed; real HOME; `AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED=2c456d263…`)
- Healthy: rc 0, `DRY RUN ok: mission=world`, `lanes=DEGRADED(2)`. Codex is over its daily ration (probe rc 75), so planner → opus and executor → `claude:claude-haiku-5-5`.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i70`): rc 0, `codex model 'bogus-model-i70' unusable`, executor → `claude:claude-haiku-5-5`, `DRY RUN ok`.
- The worktree-sweep log hash (`0b63a24e99ae5c66`) and the pin listing hash (`612a30193219489f`) were unchanged; no world pidfile was written.

## Mutation drills (scratch copies; each asserted applied)
| Mutant | Who ran it | Red |
|---|---|---|
| M1 literal removed from `TRANSIENT_SIG` | executor, judge | new case (rc, same rung, retry line, sid, resume, verdict) |
| M2 `2>&1` dropped from the pi launch | executor, judge, controller | new case, 8 assertions (controller: 15 pass, 1 fail) |
| M3 `_mc_pi_session_exists` always false | executor, judge | resume assertion only |
| M4 fresh session id per attempt | executor, judge | sid + resume |
| M5 `TRANSIENT_RETRIES` default 1 | executor, judge | new case + `transient-unchanged` |
| D1 literal added to `RUNTIME_QUOTA_SIG` | judge | demotion |
| D2 retries default 2 | judge | only via the `1/3` log text |
| D3 backoff 0 | judge | **survives** (suite forces backoff 0) |
| D4 grep without `-i` | judge | survives (harmless: exact case) |
| T4b ROOT not canonicalised | judge (planner's C1) | resume |
| T5a `script()` writes no `.stderr` marker | judge r1 survived → controller r2 | red after `2c456d263` |
| T5b fake ignores the marker | judge r1 survived → controller r2 | red after `2c456d263` |
| T5b + M2 | judge r1 survived (16/16) → controller, judge r2 | red after `2c456d263` |
| T6 `MISSION_TRANSIENT_RETRIES` unset dropped, ambient =1 | judge | red |
| LIE fake prints stdout but labels `stderr` | judge r2 | survives (a double edit; only real pi can refute it) |

Still unpinned (judge round 2, non-blocking): the 45 s/90 s backoff values, the 3-attempt bound beyond the `1/3` log text, and the driver's logical-vs-physical session slug (latent; live mission workdirs are physical paths).
Outside the ticket: `Provider finish_reason: network_error` and `content_filter` (pi-ai `mapStopReason`) do not match the literal. Widening `TRANSIENT_SIG` is policy class (HD-2a; compare D-FLEET-26).
