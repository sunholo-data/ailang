### Changed — triage mission: lands its own record PR (D-TRIAGE-81)

- The triage loop may now merge its own `triage/*-record` PR. The diff must touch only
  `design_docs/triage-mission*.md`, the ledger must validate, and every required check must be
  green on the head; the merge uses `--match-head-commit`. It merges no other PR and makes no card
  decisions. Mark ruled this attended on 2026-11-09, after 8 stacked draft record PRs had kept 30
  decisions off `dev`.
