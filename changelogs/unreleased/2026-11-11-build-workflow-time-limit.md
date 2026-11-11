### Fixed — the per-OS build jobs are no longer cancelled mid-suite (2026-11-11)

`build.yml` gave each OS job 30 minutes for the full test suite. Runs on `dev` take 9 to 18 minutes
but have reached 27, and a pull request with heavier tests crossed the limit: the job was cancelled
with no failing test, which the release gate reads as "not green". The limit is now 45 minutes, the
same as `ci.yml`'s `test` job. The matrix also sets `fail-fast: false`, so one OS failing or flaking
no longer cancels the other two.
