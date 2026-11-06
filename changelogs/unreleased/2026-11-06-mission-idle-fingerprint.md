### Added — Product missions skip unchanged no-pick fires (2026-11-06)

Before the controller starts, the driver snapshots a fire's inputs with the shared
`idle-fingerprint.sh --snapshot` helper. The inputs are the newest commit outside the
mission's own record paths, decision-ledger statuses, allowlisted directives, unread
`mission-<name>` mail and red checks on the branch tip. A completed, human-gated no-pick fire
then promotes that snapshot with `--record`. `--record` never recomputes, so an input that
changes during the fire still wakes the next one.

While nothing changes, later fires exit before any probe or controller and append an IDLE
slot verdict. A failed read, a dry run or a record older than 24 hours gets a full fire.
Fleet and triage keep their own idle checks.

Harness ticket `driver:idle-fire-writes-full-block-report` (World iterations 341–358).
