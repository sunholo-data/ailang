### Fixed — Gate 4 prescribes canonical mission STATUS headings (2026-10-16)

Gate 4 now gives every mission the production STATUS shape, its source regex, and validation
and repair steps. This prevents malformed charter stamps from failing the archive ratchet only
after a later rotation. [Plan: `skill:gate4-status-stamp-shape-unprescribed`](../../design_docs/planned/sprint-plan-gate4-status-stamp-shape.md).
