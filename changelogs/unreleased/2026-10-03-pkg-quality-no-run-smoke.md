### Fixed — `pkg quality --no-run` reported the un-run smoke as a PUB015 failure (2026-10-03)

`--no-run` discovers `_smoke.ail` without executing it, but the smoke section carried `passed: false`
with nothing to say it never ran, so a package whose smoke passes got a `PUB015 _smoke.ail failed`
gate and exit 2. The smoke section now carries `not_run: true` and `notes: "not run (--no-run)"`, and
an un-run smoke is an info-level `PUB015` badge ("present but not run"), like the existing `PUB012`
badge for tests that were not run. A smoke that ran and failed still gates. (#1305)
