### Added — simplicity audit tracks the v1.0 codebase-health numbers (2026-11-12)

`make simplicity-audit` now records, week over week, the numbers the 2026-11-09
codebase-health handover (§2) measured by hand (workstream 5):

- `unreachable_funcs` — `deadcode@v0.28.0 -test=false ./cmd/...`, `third_party/`
  excluded (counted apart), top packages in the detail. Gated `le`, ratchet at 1,301.
- `prod_loc_core` / `prod_loc_platform` / `prod_loc_platform_pct` — tracked non-test Go
  under `internal/`, split by the core list read from `.claude/rules/architecture.md`.
  Informational.
- SonarCloud gate + security/reliability/maintainability ratings, CodeQL high+critical
  open, Dependabot open by severity, secret scanning on/off — new
  `tools/simplicity_external.sh`. Optional and offline-safe: no network, no `gh`
  credential, or `--offline` records `null` with a `reason`, never 0, never a failed run.
- `cloud_run_services_used` — behind `--services` (gcloud, read-only, minutes): 30-day
  2xx request classes and distinct IPs per Cloud Run service across prod/dev/test.

`audit.sh` passes `--offline`/`--services` through, shows `—`/`?` (never WORSE) for a
row the old snapshot lacks, and lists the new rows under "what to look at".
