# M4 bounded documentation trials

Prepared 2026-09-08 under M-MISSION-ITERATION-RELIABILITY M4. Mark's actual
approval was “yep please continue” to the reliability plan/execute request.
These are concrete bounded tasks within that approved smoke series, not a claim
of separate per-task user quotes or completed acceptance.

| Trial | Useful deliverable | Frozen design | Frozen plan |
| --- | --- | --- | --- |
| A | Explain resource accounting to operators | [Budget accounting](budget-accounting/design.md) | [Plan](budget-accounting/sprint-plan.md) |
| B | Explain what evaluator evidence does and does not establish | [Review packet](review-packet/design.md) | [Plan](review-packet/sprint-plan.md) |

Each independently edits only `docs/docs/guides/mission-iteration.md`, from the
same reliability implementation commit. Neither imports or depends on the other's
output. Parent controller supplies that full immutable base commit in each work
item before dispatch, after implementation is committed. Do not use moving HEAD
as the frozen input. The design and plan files are imported prerequisites with
attended-session registry provenance `gpt6-astra`; the parent records their exact
Git object references/hashes and existing content-bound authority separately.
These markdown files alone are not executable authority.

Each task requires a real author commit and an independent evaluator, all criteria
settled, exact hard-check receipts, and the normal untracked `stage-result.json`.
Evaluator vendor independence includes imported designer/planner provenance and
actual author identity. Parent freezes concrete compatible routes before inference;
there is no fallback route implied by these briefs. No merge, publication, code
changes or scheduling expansion is part of either task.

Combined new M4 series spend ceiling remains $5, including the existing-candidate
review. Suggested maximum reservations: existing candidate $2; A $1.50; B $1.50.
For A/B, reserve author $1 and evaluator $0.50 each, total 220,000 fresh tokens
(author 120,000; evaluator 100,000), total 2,400 seconds (1,200 each). These are
upper bounds, not permission to spend more than the controller's remaining series
balance. Parent may freeze tighter limits in the specs; any increase beyond these
reservations requires reviewing the remaining $5 series allowance first. Stop at
exhaustion; no automatic repeated attempt or budget increase.

Record per task: candidate and acceptance digests, hard-check receipts, author and
evaluator routes, wall time, fresh/cache usage separately, metered spend separately
from list-price-equivalent/unknown figures, repeated tool calls, cleanup outcome,
and attended interventions. Replay accepted work with zero new dispatches. Preserve
failed attempts and report incomplete adoption if either task fails; three bounded
examples establish a smoke result, not a statistical productivity claim.
