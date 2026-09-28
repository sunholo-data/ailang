### Changed — changelog entries are fragments; the macOS launchd CI job runs only for rig-shell changes (2026-09-28)

Releasing had turned into a chore, and neither pipeline was the cause: the GitHub release workflow
runs about 6 minutes and the Cloud Build deploy 5–10. Measured on 2026-09-28, the time went on PR CI
(25–45 min per push) multiplied by forced rebases:

- **Changelog fragments.** Every PR added its entry at the same line under `## [Unreleased]` in
  `changelogs/v0.32-current.md`, so each merge conflicted with every other open PR. #1369 alone was
  rebased four times, each restarting CI. New entries now go in
  `changelogs/unreleased/YYYY-MM-DD-<slug>.md`, which cannot conflict. At release,
  `scripts/changelog_fold.sh` (a new release-manager step 1.5) inserts them newest-first under
  `## [Unreleased]` and removes them, so every downstream reader sees one file as before.
  `make check-changelog` validates fragments (`--check`), and `make test-check-changelog` runs a
  7-arm self-test. Removing the `sort -r` ordering fails its first arm. The sprint-executor and
  release-manager skills (in both `.claude/` and `.agents/`) and the coding-standards rule now point
  writers at fragments. The sprint evaluator's changelog grep includes them.
- **launchd bash-3.2 job path-filtered.** It needs a macOS runner, which queued 38 minutes for a
  6-minute run and gated every PR. The `changes` job gains a `shell` output that is true only when a
  PR touches a path its suite reads: `tools/launchd/`, `tools/pi-extensions/`, `scripts/hooks/`, the
  mission scripts, the mission-control skill, `.claude/settings.json`, `design_docs/v1-mission.md`,
  `make/test.mk` or `ci.yml` itself. Pushes to `dev` still run it.
