### Fixed — Preserve absent quorum reviewers' recovered vetoes

Tier-1 quorum reviews now retain complete top-level verdict tokens from invalid responses,
including truncated output and valid JSON that violates the review contract. Recovered rejects
block synthesis while remaining named absences; recovered passes are audit only and never
count as votes, external signal or seats. JSON syntax diagnostics report the byte offset and
a centred UTF-8-safe window, including Claude transport envelopes. The mission-control remedy
now distinguishes budget absences from invalid responses: revise recovered objections directly,
or allow one same-cap retry when no verdict was recovered, without raising the invalid cap.

Design: [M-QUORUM-SALVAGE](../../design_docs/planned/m-quorum-salvage-retry.md); issue #941.
