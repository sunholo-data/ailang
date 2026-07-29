# Archived design docs

Docs moved out of `planned/` during the 2026-07-29 attended triage (Mark's directive).
Criteria for archiving — BOTH had to hold:

1. **No living reference**: the doc's stem appears nowhere in `design_docs/v1-mission.md`'s
   Queue/clause sections (verified by stem, not filename, per the empty-search-is-a-claim rule), OR
2. **Stale duplicate**: an authoritative copy already exists under `implemented/` (the
   `planned/` twin was the leftover).

Archived docs keep their original sub-folder (`archive/<era>/<name>.md`). Historical
mission-log entries referencing them are left as-is — this README is the old→new map.
Nothing here is deleted; unarchive with `git mv` if an item earns its way back
(demand evidence per the program's routing rules).

Also in this pass: `m-bytecode-vm-parity-bugs.md` moved `planned/v0_29_0/` → `planned/v1_0_0/`
(active clause-2 item, Mark-decided A+B scope — it was mis-foldered in the old era, not stale).

NOTE for future triage: the remaining `planned/v0_29_0/` docs are NOT dead — the charter's
living queue references nearly all of them (mostly post-v1 / normal-road). Do not bulk-archive
that folder without a per-doc charter-section check.
