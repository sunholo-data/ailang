# Attended canary authority — 2026-09-14 (evaluator-only successor, raised allowance)

Locator: `mark-docs-canary-review-4-approved-20260914`

Mark instructed, in an attended session, after being shown the measured requirement and the
recommended figure:
> ok lets try clsoe m4 finally

This records the attended instruction, not a model self-approval. It authorizes ONE
evaluator-only successor to the terminal work item `docs-canary-guide-review-3`, at a
**150,000-token** allowance, and nothing else. No new author work, no merge, no publication.

## What is being approved, and on what evidence

`docs-canary-guide-review-3` was killed by the token guard **four times**, across two
models and two harness configurations:

| Run | Route | Processed | Cap |
|---|---|---|---|
| review-2, 09-08 | `pi-or-deepseek-v4-flash:floor` | 100,216 | 100,000 |
| review-3, 09-08 | `pi-or-deepseek-v4-flash` | 100,024 | 100,000 |
| review-3, 09-14 | `pi-or-minimax-m3`, session gate armed | 101,131 | 100,000 |
| review-3, 09-14 | `pi-or-minimax-m3`, gate isolated | 101,542 | 100,000 |

The final run is the one that justifies the number: with the session-protocol gate removed
the stage spent **zero** calls on refusals, repeated **zero** calls, emitted only 1,720
output tokens in 36 seconds, reached `progress_status: "final report"`, and still needed
101,542. There is no waste left to reclaim — the reading load the contract prescribes
(the candidate diff plus the production spec, authority and CLI source) simply exceeds
100,000.

150,000 is ~1.48x the measured requirement. The margin covers the two variances actually
observed: model choice (deepseek and minimax differed ~1.5% on the same contract) and the
production source growing. It is deliberately not larger — a genuinely looping stage must
still be stopped.

The cost ceiling stays **$2**: the most expensive of the four runs cost $0.209.

The timeout is raised 1200s → **1800s** because the frozen spec caps a stage at the work
item's own limits and 1200s left no headroom over the 36s–172s observed; it remains bounded.

## Bound artifact

Prerequisite authority for the designer and planner artifacts is unchanged and already
recorded in `authority.md` (locator `mark-docs-canary-work-item-guide-approved-20260907`).
This document adds only the accepted AUTHOR artifact for the evaluator-only continuation.

| Role | Artifact | Commit | SHA-256 of committed content |
| --- | --- | --- | --- |
| author (accepted) | `docs/docs/guides/mission-iteration.md` | `58f9fd4bc2d5003dd9f04760edb8054a4367c114` | `70bccdfdaa6a8eecdc895addc3ed779e11eb5a0b480ca16b0afd27dd29dc9f3c` |

Digest verified first-party against the committed blob, not copied from tooling output.

## What this does NOT authorize

Raising the allowance again, relabelling any earlier failure, clearing runtime state by
hand, dispatching either of the two additional frozen briefs (`review-packet.json`,
`budget-accounting.json` — both still declare 100,000 for their evaluator and would hit the
same wall), merging the candidate, or publishing the guide. Runtime acceptance is separately
required.
