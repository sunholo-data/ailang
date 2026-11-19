# Fix round 1 — fleet i105 `--tool-stall-seconds` (executor report)

Commit: `902016622a02023807747c58832dc6a42cd6afbc`
Subject: `fix(mission-pi-run): harden tool-stall suite against start-up latency; valid verdict JSON`
Files: `scripts/mission_pi_run.sh` (12 lines changed), `tools/launchd/test_mission_pi_run_tool_stall.sh` (60 lines changed). Untracked `design_docs/planned/sprint-plan-pi-tool-stall-seconds.md` left out of the commit.
Branch `fleet/i105-tool-stall-seconds`, not pushed.

## Status
- B1 (flake): fixed. Timing arms moved to a 10 s stall bound; arm 6 asserts `poll_seconds`.
- NB1 (item 2 `poll_seconds`): fixed, asserted by arm 6.
- NB2 (item 3, invalid JSON for `--stall-seconds abc`): **NOT fixed. Blocked, see below.**
- NB3 (item 4, >9-digit `--tool-stall-seconds`): fixed, arm 5 covers it.

## Item 1 — start-up latency (B1)
Timing arms now use stall 10 (was 4). Tool sleeps and bounds were rescaled so each arm keeps its discrimination:
| Arm | Before (stall/tool/sleep) | After (stall/tool/sleep) | Purpose kept |
|---|---|---|---|
| 1_default | 4 / (=4) / 8 | 10 / (=10) / 20 | open tool killed as tool_hang at the bound |
| 2_raised | 4 / 20 / 8 | 10 / 30 / 15 | open tool silent longer than stall bound survives |
| 3_model_silence | 4 / 20 / 12 | 10 / 30 / 20 | no open tool → stall bound applies (kills tool-bound leak) |
| 3b_closed_tool | 4 / 20 / 12 | 10 / 30 / 20 | closed tool → stall bound applies (kills "any start" leak) |
| 4_hang_past_bound | 4 / 6 / 15 | 10 / 12 / 25 | finite tool bound still kills |
Arm 6 keeps stall 60 / tool 3 (its clamp timing is unchanged).
Sequential wall time of the suite: 80 s at host load ~40 (target ~90 s).

## Item 2 — deterministic arm 6
- Runner: new verdict key `"poll_seconds"` = the effective `POLL_SECONDS` after the clamp (numeric via `json_num`).
- Arm 6 now asserts `poll_seconds == 1` (stall 60, tool 3, default poll 10 → clamp 3/3 = 1). The wall-clock `elapsed <= 9` assertion is removed.

## Item 3 — valid verdict JSON
- Done: `json_num` helper. `stall_seconds`, `tool_stall_seconds`, `poll_seconds` are emitted as bare numbers when all digits (leading zeros dropped, so `08` → `8`), else `null`. Checked by hand: `--stall-seconds 08 / 0 / 600` all produce valid JSON with the right values.
- **Not done: `--stall-seconds abc`.** Measured on HEAD (f993cadf0) and on this commit: the run aborts inside the poll-clamp arithmetic (`scripts/mission_pi_run.sh` poll clamp, `abc: unbound variable` under `set -u`), exits **rc 0**, and writes **no verdict file**. The pi lane never runs. So:
  - The evaluator's observation ("invalid JSON with rc 0") does not reproduce; the actual behaviour is no verdict at all.
  - A `jq -e .` arm for `abc` cannot pass without changing the value's handling: refusing it (rc 14 launch_failed, like the other bad values) or defaulting it. You ruled out refusal, and a default is a silent fallback.
  - I did not commit a failing arm. The 5b arm I wrote was removed before the commit.
  - Decision needed: (A) refuse non-numeric `--stall-seconds` with rc 14 + verdict (changes HEAD's rc-0-no-verdict behaviour; I recommend it), or (B) keep it and assert the real behaviour. Your call.

## Item 4 — 9-digit cap
- `--tool-stall-seconds` values of 10+ characters are refused in the same `case` as the other bad values: rc 14 `launch_failed`, verdict written. `99999999999999999999` added to arm 5's list (passes).

## Acceptance (unpiped, rc recorded)
| Command | rc | Result |
|---|---|---|
| `/bin/bash tools/launchd/test_mission_pi_run_tool_stall.sh` (alone) | 0 | passed=16 failed=0 |
| six concurrent copies (`fix_par_1..6.log`) | 0 ×6 | passed=16 failed=0 each, no FAIL lines |
| `/bin/bash scripts/test_mission_pi_run.sh` | 0 | passed=36 failed=0 |
| `/bin/bash tools/launchd/test_mission_pi_run_commits.sh` | 0 | passed=9 failed=0 |
| `/bin/bash tools/launchd/test_mission_pi_run_provider_quota.sh` | 0 | passed=29 failed=0 |
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | 0 | 5 passed, 0 failed |

Syntax checked with `/bin/bash -n` on both files. Host load during the concurrent run was about 23–34.

## Mutation re-check (in place, restored with `git checkout -- scripts/mission_pi_run.sh`, each followed by `git diff --quiet`)
| Mutant | Edit | Suite | Killed by |
|---|---|---|---|
| M-b tool_open never applied in loop | loop line 388 → `if false; …` | rc 1, 13/3 | 2_raised, 4_hang_past_bound, 6_clamp |
| M-c tool bound applied unconditionally | loop line 387 → `BOUND="$TOOL_STALL_SECONDS"` | rc 1, 14/2 | 3_model_silence, 3b_closed_tool |
| M-d tool_open = any start seen | line 259 `"${e:-0}" ]` → `0 ]` | rc 1, 15/1 | 3b_closed_tool |
| M-i poll clamp on STALL/3 | lines 356–357 `MIN_BOUND / 3` → `STALL_SECONDS / 3` | rc 1, 15/1 | 6_clamp (`poll_seconds=10`, expected 1) |

All four are killed. Each restore printed `RESTORED`, and the final `git diff --quiet` printed `RESTORED`.

## Notes
- The untracked plan file is still untracked and out of the commit.
- The `--stall-seconds abc` finding is new information for the evaluator: its "invalid JSON" claim does not reproduce on HEAD.
