### Changed — triage mission: decision packets ask answerable questions (D-TRIAGE-49)

- New gate step §0b (decompose sweep): a ledger row that bundles several design choices ("review
  D1/D3/D4/D6 before dispatch") is split into one OPEN row per choice. Each child carries the design
  doc's recommended option, stated concretely. The parent is marked DECOMPOSED and dispatch stays
  held until every child is answered.
- Packet self-check (§7a): every numbered item must change something when the answer is "yes".
  A "review X" recommendation is no longer allowed, and items that cannot be one line go on a
  separate, uncounted "Needs a read" line, so "all yes" never covers them. The first packet
  (2026-11-04) had 4 of its 5 items in that form.
