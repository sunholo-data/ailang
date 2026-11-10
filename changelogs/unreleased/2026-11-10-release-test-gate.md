### Added — Release binaries wait on the tagged commit's CI (2026-11-10)

`release.yml` published binaries with no test dependency: v0.57.4 (`dec3e8756`) shipped with its
`Build macos-latest` test run red. A new first job, `ci-gate`, now runs
`tools/ci/require_checks.sh`, and every build, sign and publish job `needs:` it.

- The gate reads the commit's recorded check-runs; it does not re-run tests. It pages through all
  of them, and every GitHub Actions run of each required name must conclude `success`, so a matrix
  that emits two runs under one name needs both green. A pending or missing check is polled for up
  to 90 minutes, then fails closed. Any completed run that is not `success` fails at once, with
  the job URL.
- The required names are one data file, `tools/ci/release_required_checks.txt`. It ships with the
  `ci.yml` jobs `test`, `checks`, `lint`, `build`, `test-windows`, `govulncheck (vuln gate)` and
  `float determinism (arm64)`, plus `build.yml`'s per-OS jobs `Build ubuntu-latest`,
  `Build windows-latest` and `Build macos-latest` (ruled by Mark, 2026-11-10). With the `ci.yml`
  names alone the gate would not have stopped v0.57.4, whose tag commit had a red macOS build. A
  flaky per-OS job now blocks a release until it is re-run.
- Override: an **annotated** tag whose message has a line
  `release-gate-override: https://github.com/<owner>/<repo>/issues/<n> <reason>` waives the red
  checks; it does not waive missing or pending ones. The override is printed at the top of the
  release notes. A malformed marker is an error. The gate ignores a marker that reaches it through
  a lightweight tag (that text is a commit message) or a tag that points at another commit.
- Dry run: `workflow_dispatch` with a `sha` input runs only the gate. Nothing is built or published.
- Self-test `tools/ci/test_require_checks.sh` (27 arms, a fake `gh`, no network) joins
  `make test-launchd-drivers`.

Design: `design_docs/planned/v1_0_0/m-v1-release-gates.md` §1 (M2).
