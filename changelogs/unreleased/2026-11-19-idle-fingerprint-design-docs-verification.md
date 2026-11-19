### Fixed — mission idle pre-check woke on design_docs/verification record-only commits (2026-11-19)

`tools/launchd/lib/idle-fingerprint.sh` excluded a mission's per-iteration records from `head=` only
under the repo-root `verification/<name>-iter*`. World now writes them under
`design_docs/verification/<name>-iter*`, so every record-only commit (e.g. world iter 414,
`4299cdb6`) moved `head=` and the driver ran a full fire on unchanged state. Two slots were lost.
The helper now excludes both locations. The exclusion stays narrow: other `design_docs/verification/`
dirs (feature `w-*`, `world-attended-*`) still wake the loop. The idle-fingerprint test adds the
new record arm, a non-iter negative control and per-exclusion mutants. Ticket
`driver:idle-fingerprint-head-ignores-design-docs-verification`.
