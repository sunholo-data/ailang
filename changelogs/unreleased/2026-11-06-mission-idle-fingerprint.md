### Added — Product missions skip unchanged no-pick fires (2026-11-06)

A completed, human-gated no-pick fire can record its inputs with the shared
`idle-fingerprint.sh --record` helper. Subsequent identical fires exit before probes or
controllers and append an IDLE slot verdict. Product commits, ledger rulings, allowlisted
directives, unread mission mail or red branch-tip checks wake the loop; failed reads and
records older than 24 hours force a full fire. Fleet and triage retain their own checks.
Harness ticket `driver:idle-fire-writes-full-block-report` (World iterations 341–358).
