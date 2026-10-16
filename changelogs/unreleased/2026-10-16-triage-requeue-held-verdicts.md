### Changed — triage re-verifies held verdicts (D-TRIAGE-11)

- The triage mission's Gate 2 selection now re-queues every item whose newest shadow-ledger row
  is an actionable verdict (`VERIFIED`, `CLOSE-*`) that was never acted on. Iteration 1's ten
  actionable verdicts were otherwise skipped by every later iteration because the items had not
  changed. The item is re-verified against current `origin/dev`, never acted on from the old verdict.
