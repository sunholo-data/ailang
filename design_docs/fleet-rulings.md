# Fleet-wide rulings

Rulings in this file bind **every** mission. Each charter links here, and nothing in a charter or a mission
skill overrides a RESOLVED, unsuperseded row. Design: `m-fleet-rulings-ledger` (D-FLEET-25 A, Mark attended 2026-11-16).

- **Rows** use the decision-ledger table: `scripts/mission_decisions.sh --check --file design_docs/fleet-rulings.md`.
  A new ruling arrives OPEN and is resolved only by Mark, attended:
  `scripts/mission_answer.sh --id D-FR-<n> --file design_docs/fleet-rulings.md`. A resolved row is never re-answered.
  To revise one, add a new ID plus a `supersedes` line, in one reviewed PR. A transcription row cites its source row.
- **Detectors** (the block after the table) are what `ailang mission doctor` checks. One per line:
  `<ID> forbid <registry|skills|charters> "<literal, 8-200 bytes>"` or `<ID> supersedes <lower ID>`; at most 64.
  A hit is a `note`, never drift: the doctor reports and does not block.
- Do not quote a forbidden literal outside a decision-ledger block in any charter, registry file or mission skill.
  Paraphrase it instead.

<!-- decision-ledger:start -->
| ID | Status | Ruling | Evidence |
|---|---|---|---|
| D-FR-1 | RESOLVED | World runs the shared driver; it is not a fork. Text calling it a fork is stale. | Transcribed from D-FLEET-25 (fleet-mission.md, Mark attended 2026-11-16). |
| D-FR-2 | RESOLVED | generator≠judge is a PREFERENCE at vendor level, never a blocker (model-level self-review stays refused). | Transcribed from V1 D-62/D-63 (v1-mission.md, Mark attended 2026-09-08); skill fixed in 349e311b9. |
<!-- decision-ledger:end -->

<!-- fleet-rulings-detectors:start -->
D-FR-1 forbid registry "WORLD IS A FORK"
D-FR-1 forbid charters "WORLD IS A FORK"
D-FR-2 forbid skills "generator≠judge guard (HARD"
<!-- fleet-rulings-detectors:end -->
