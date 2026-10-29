# Shadow report 2026-10-12 – 2026-10-19

5 comparisons: 4 graded to the end, 1 still in progress, 1 with base drift.

## Harnesses

| Harness | Runs | Pass (graded) | No output | Cost | Wall time | Turns | Wins |
|---|---|---|---|---|---|---|---|
| motoko | 5 | 4/4 (100%) | 0 | $0.75 | 7m30s | 60 | 2 |
| pi | 5 | 2/3 (67%) | 1 | $0.75 | 7m30s | 60 | 0 |

## Verdicts

- primary_better: 1
- shadow_better: 1
- equivalent: 1
- both_failed: 0
- ungraded: 1
  - ungraded because: apply_failed 1

## Disagreements (2)

### task-a — motoko (shadow) better

- primary pi: `ailang chains find task-a` then `ailang chains chat <chain-id>`; diff: 1 file changed, 2 insertions(+), 1 deletion(-)
- shadow motoko: task `task-a-shadow`, session `sess-task-a-shadow`, artifacts `tasks/task-a-shadow`; diff: 1 file changed, 2 insertions(+), 1 deletion(-)
- decided by: reviewer
- reviewer: shadow (confidence 0.80): primary drops the error case

### task-b — motoko (primary) better

- primary motoko: `ailang chains find task-b` then `ailang chains chat <chain-id>`; diff: 1 file changed, 2 insertions(+), 1 deletion(-)
- shadow pi: task `task-b-shadow`, session `sess-task-b-shadow`, artifacts `tasks/task-b-shadow`; diff: 1 file changed, 2 insertions(+), 1 deletion(-)
- primary PR: #12 merged (circular)
- base drift: the two runs started from different commits

