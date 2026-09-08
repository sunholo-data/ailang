# Live trial: containment passes; evaluator completion fails

**Implementation:** `48e72ef7e`, independent code review 94/100, zero code blockers.
**Live adoption:** incomplete. The existing guide still has no accepted independent verdict.
Do not mark the whole sprint complete or start unattended/fleet adoption from these results.

## Existing candidate review

Original accepted product: `58f9fd4bc2d5003dd9f04760edb8054a4367c114`.
Successor: `docs-canary-guide-review-3`, prepared by the new command from the original
failed author/evaluator work item. Original baseline, author provenance, criteria,
checks and accepted artifact hash survived preparation. New committed approval:
`97e695be542c6580d603ff3d6f9729bd4ecc6541`. Exact input/manifest in `trials/inputs/`.

| Measurement | Observed |
| --- | --- |
| Route | pi-or-deepseek-v4-flash, OpenRouter deepseek-v4-flash-0731:floor |
| Author dispatches | 0 |
| Evaluator dispatches | 1 |
| Evaluator wall time | 362,068 ms |
| Fresh input / output | 93,659 / 6,365 |
| Fresh total / limit | 100,024 / 100,000 |
| Cache reads / creation | 504,123 / 0, reported separately |
| Metered cost | $0.016704388 |
| New trial series cap / spent | $5 / $0.016704388 |
| Turns / tool calls | 19 / 26 (25 completed) |
| Exact repeated calls | 0; this does not exclude semantically repeated investigation |
| Final result | execution_failed, token guard (finish reason thrash_aborted) |
| Evaluator verdict / stage-result.json | Neither produced |

The retained Pi user-message event contains the full 20,716-character contract and
packet, including the guide criteria and validation command. This establishes the
local harness input, not correctness of downstream provider prompt handling.
The actual calls repeatedly list directories, inspect Git status/log/HEAD and mission
context with different command strings. They do not run the bound guide validator
or settle the frozen guide criteria. The model also attempts a malformed revision
suffix. Two tool results are errors; the rest complete. No compaction is reported.
Thus removing exact duplicate reads is insufficient, and this trial does not support
another token-cap increase as a solution. The remaining issue is evaluator convergence
or prompt/history delivery on this route, not a demonstrated shortage of useful work time.

## Cleanup and replay evidence

The new supervisor returned exit 5 and **automatically restored** both the previously
absent Docs marker and runtime binding after verifying its session stopped. The
installation record says `restored`, process phase `exited`, no cleanup_pending.
No attended process kill, binding restore, marker deletion or database edit was needed.

A second owned activation replayed the same terminal failed item: exit 5, the exact
one receipt journal and its hash unchanged, zero new dispatches, and both baseline
files absent afterward. This proves failed-terminal replay containment, not the
completed-success replay required by M4. Read-only status still works with `--activation`.
All original work-item, child and acceptance row hashes, and both historical input
file hashes, are unchanged. Evidence lives in `trials/results/`.

## Remaining gate and prepared work

Two independent small documentation tasks are fully frozen, approved by the delegated
M4 scope, and dry-run valid from `e6b54faf0c857553d55cfac171d3f0dd49c7d6b9`:
Budget accounting and Review evidence packet. They remain undispatched because the
plan requires successful existing-candidate acceptance and completed replay first.
Their separate $1.50 ceilings do not authorize bypassing that gate or automatic retries.

The next concrete investigation should isolate Pi/DeepSeek prompt and conversation
handling against a small evaluator contract, then choose a reviewed successor strategy
or explicit independent route. Preserve this failure; do not relabel it pass, clear its
state manually, or silently raise its immutable allowance. No merge or publication occurred.
