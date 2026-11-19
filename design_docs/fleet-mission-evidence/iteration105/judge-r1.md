# Evaluation: fleet i105 `--tool-stall-seconds` (commits e653d19a5, f993cadf0)

VERDICT: FAIL, 68/100. The runner change is sound and resolves the ticket. The new test suite flakes on a loaded host, and that flake is blocking.

## Score
| Category | Pts | Note |
|---|---|---|
| Functionality / ticket resolved | 27/30 | Open tool gets its own bound. Model silence keeps `--stall-seconds`. Defaults are unchanged. The verdict JSON can go invalid on a garbage `--stall-seconds` (NB2). |
| Test quality / reproducibility | 12/25 | Arms are well designed and kill mutants. But arm 1 (`1_default`) flakes about 50% at load 35 (B1). |
| Constraints (scope, defaults, markers, mirror, bash 3.2) | 15/15 | Every constraint was checked; see below. |
| Skill text accuracy / actionability | 8/10 | Rule (d) is concrete. See NB3. |
| Code quality / simplicity | 4/5 | One `tool_open` helper replaces the inline count (the relabel's old `_starts > TOOL_CALLS` check was identical, since TOOL_CALLS counts `_end`). Runner is 524 lines. |
| Process / honesty of report | 2/5 | The report claims "x3 concurrent, 15/0 each" and "no arm survives". It never mentions arm 1's load sensitivity. |
| Total | 68 | |

## Reproduced
- `/bin/bash tools/launchd/test_mission_pi_run_tool_stall.sh` at load 20: rc 0, passed=15 failed=0. (My `PIPESTATUS` capture printed empty; the `passed=15 failed=0` line was visible.)
- `/bin/bash scripts/test_mission_pi_run.sh`: passed=36 failed=0.
- `/bin/bash tools/launchd/test_agents_skills_sync.sh`: 5 passed, 0 failed.
- Scope, mirror, handshake-span and defaults. I checked the diff only, not the sync or marker tests directly beyond the sync suite above.
  - Diff files are all inside the write scope.
  - The only skill hunks are at old lines 431, 439, 487 and 532. The handshake span ends at the END marker, and the new line sits after it.
  - Defaults 600 / 1800 / 10 / 50 are untouched.
  - With no new flag or env, the tool bound equals the effective stall bound, so kill time and verdict are the same. The verdict JSON gains three keys.
  - No bash 4 constructs.

## BLOCKING
**B1. `1_default` flakes under load.**

Repro, from the worktree at load 35:
`for i in 1 2 3 4 5 6; do /bin/bash tools/launchd/test_mission_pi_run_tool_stall.sh 2>&1 | grep -E '^(FAIL|passed)'; done`

Output, 3 of 6 runs:
```
FAIL: 1_default (rc=12 verdict=stream_dead bound=4; expected rc=18 verdict=tool_hang bound=4)
FAIL: 1_default verdict keys (stall_seconds/tool_stall_seconds/hung_tool)
passed=13 failed=2
```
The other 3 runs passed 15/15. During my mutation runs, mutants D and E also showed `1_default` failing in the same way, which is how I found it.

Cause (inferred from the rc 12 symptom; I did not trace it directly). The stall clock starts at launch, and the stub is slow to emit `tool_execution_start` under load. With `--stall-seconds 4`, the 4 s bound expires before the start line is in `--out`. The kill then sees no open tool and reports `stream_dead` instead of `tool_hang`. Arm 4 (bound 6) and arm 6 (bound 3, poll 1) are exposed to the same risk.

Impact: the suite is in `LAUNCHD_SUITES`. On a loaded rig it will randomly redden `make test-launchd-drivers`, and the repo already has a history of dev going red from such gates.

Fix: widen the arm 1 bounds (for example stall 12, tool sleep 20). Alternatively, have the stub wait for the start line to be flushed before the clock matters, or retry the arm once and treat a repeated failure as real.

## NON-BLOCKING
- **NB1.** Executor deviation 6 (the full `make test-launchd-drivers` was not run) is fine for the controller to cover. But "x3 concurrently green" is weak evidence, as shown by B1.
- **NB2.** The new verdict keys interpolate `$STALL_SECONDS` and `$TOOL_STALL_SECONDS` unquoted.
  - `--stall-seconds abc` now produces INVALID verdict JSON with rc 0 (`jq -e .` fails). Before, `stall_seconds` was not in the JSON.
  - The arithmetic errors (`abc: unbound variable`) on that input are pre-existing.
  - The JSON invalidity is new, but only for garbage input.
  - `--tool-stall-seconds 99999999999999999999` passes validation and then prints `[: integer expression expected` with the run continuing.
  - Cheap fix: validate or quote these values.
- **NB3.** Rule (d) says "never below 600" and "2x measured". The ticket's measurement (396-573 s, killed at 600 s under load 30-60) implies a bound of at least 1146 s. The rule gives the agent no guidance on the `--max-seconds` it must also raise beyond the 3600 rule, but it does say to size `--max-seconds`. Acceptable.
- **NB4.** An open tool with a lower tool bound than the stall bound is allowed and handled by the clamp (arm 6). Documented only in a comment.
- **NB5.** The plan said 520 lines; the runner is 524.

## Mutations run by me (in place, new suite only, restored each time; `git diff --quiet` printed RESTORED each time, and `git status` showed only the untracked plan file)
- **A.** Gate the check on `IDLE -ge STALL_SECONDS` instead of `MIN_BOUND`. Killed by `6_clamp` (rc 0 at 31 s). 13/2.
- **B.** `STALL_BOUND_APPLIED=$STALL_SECONDS` always. Killed by `4_hang_past_bound` and `6_clamp` (bound 4 / 60 instead of 6 / 3). 13/2.
- **D.** Remove the `0*` leading-zero refusal. Killed by `5 '0'` and `5 '08'`. `1_default` also failed, which was the flake, not the mutant.
- **E.** `-ge` changed to `-gt` on the bound comparison. The only failures were `1_default` (rc 12) and its keys check. These are the same failures as the load flake, so the suite cannot distinguish an off-by-one from load noise. The mutant effectively survived, or at best is ambiguous.

## Does it resolve the ticket?
Yes. A silent `make free-nav-stop-test`-shaped tool call can now be given `--tool-stall-seconds` of roughly 1200 s. Model-stall detection (`reasoning_stall` / `stream_dead`) stays at 600 s, so the filer's blunt `--stall-seconds 1500` workaround is no longer needed. This holds subject to B1 on the test side.

## Strongest objection to my own verdict
The flake only shows at load about 35, well above the ~16-20 stated for the rig, and at load 20 the suite passed 15/15. The product code is correct and the skill text is good. A reasonable evaluator could PASS at about 75 and file the arm-1 widening as a follow-up. I chose FAIL because the controller will run the gate under parallel load, and a 50% red on a gate suite is exactly what makes dev red. The fix is a one-line change to the test, after which I would score this about 80.
