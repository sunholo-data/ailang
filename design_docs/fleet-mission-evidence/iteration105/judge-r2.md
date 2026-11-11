# Round 2 evaluation: fleet i105 after fix commit 902016622

VERDICT: PASS, 82/100.

| Category | Pts |
|---|---|
| Functionality | 29/30 |
| Test quality | 21/25 |
| Constraints | 15/15 |
| Skill text | 8/10 |
| Code quality | 4/5 |
| Process | 5/5 |
| Total | 82 |

## NB2 retracted
My round-1 NB2 was a misreading.
- I ran `--stall-seconds abc` and tested the verdict with `jq -e . o1.verdict.json >/dev/null 2>&1 && valid || INVALID`.
- That command prints INVALID for a MISSING file as well. I never checked that the file existed.
- Re-derived: `ls /tmp/fleet_i105/ev2/o1*` shows only `o1`, `o1.snapshot.ndjson` and `o1.stderr`. There is no verdict file.
- stderr shows `line 353: abc: unbound variable`. That is the pre-existing set -u abort in the poll-clamp arithmetic.
- The controller's finding (rc 0, no verdict file, on both base and head) is correct.
- Residual difference: head prints one extra stderr line, `line 158: [: abc: integer expression expected`. This is harmless noise.

## (1) B1 under load
I launched 8 concurrent copies of `tools/launchd/test_mission_pi_run_tool_stall.sh`.
- Load average was 80.05 at start and 54.96 at end. Round 1 saw 35.
- Results: every copy returned passed=16 failed=0 rc=0.

Residual exposure. The 10 s bounds mean an arm can fail only if the stub's first `tool_execution_start` takes more than 10 s to land.
- Affected: arms 1, 2 and 4. That would show as `stream_dead` rc 12.
- Not affected: arm 6 (bound 3) is safe even when the start line is late. Progress resets when the line lands and the open tool then trips the 3 s bound, still `tool_hang`.
- Arm 2 and arm 4 also have a 30 s `--max-seconds` against sleeps of 15 and 25 s. That is 5 to 15 s of headroom.
- I did not see this happen even at load 80.

## (2) Mutations of my own
Each was applied in place, then restored. `git diff --quiet` printed RESTORED each time, and `git status --short` showed only the untracked plan file.
- **M1. Poll clamp reverted to `STALL_SECONDS/3`.** Killed by `6_clamp poll_seconds=10` (expected 1). 15 passed, 1 failed. The new `poll_seconds` assertion is deterministic, with no wall time involved.
- **M2. Removed the 9-digit refusal.** Killed by `5 bad value '99999999999999999999' (rc=10)`. 15 passed, 1 failed.
- **M3. `tool_open` always false.** Killed by `1_default`, `2_raised`, `4_hang_past_bound` and `6_clamp`. 10 passed, 6 failed.

## Findings
**BLOCKING:** none.

**NON-BLOCKING:**
- The residual start-up-latency window above (10 s); consider adding a stub-ready wait.
- `--stall-seconds` is still unvalidated. That is pre-existing, out of scope per the plan, and the follow-up ticket still needs filing.
- The extra `line 158` stderr line is cosmetic.
- Rule (d) in the skill text (2x measured, floor 600) is actionable; I made no change to my round-1 assessment of it.

## Strongest objection to my own verdict
The fix removes the observed flake by widening a margin, not by synchronising on the stub having started. A latency spike of 10 s or more would still produce a false red. 8 passes in 8 at load 80 is strong evidence that the window is rare, not that it is closed. The suite also now takes longer, with sleeps of 15 to 25 s per arm. That cost is acceptable, and the rest of the change remains correct and mutation-resistant.
