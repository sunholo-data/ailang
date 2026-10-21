### Fixed — Routing evidence heading recognition (2026-10-21)

The routing checker now recognises colon, period, parenthetical and bulleted headings, and stops bulleted blocks at sibling fields while retaining fleet role bullets; hyphenated compounds with allowed model prefixes produce no findings only when their suffix contains no further `gpt` (case-insensitive). See the [sprint plan](../../design_docs/planned/sprint-plan-routing-check-heading-shapes.md).
