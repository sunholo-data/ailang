# Changelog fragments

**Write new changelog entries here, not in `changelogs/v*-current.md`.**

One file per change: `changelogs/unreleased/YYYY-MM-DD-<slug>.md` (lowercase slug), holding one or
more complete `### ...` sections, exactly as they should read in the release notes:

```markdown
### Fixed — `ailang foo` crashed on empty input (2026-09-28)

What broke, why, and what changed. Link the design doc or issue.
```

Why: every PR used to add its entry at the same line under `## [Unreleased]` in the active file, so
any merge conflicted with every other open PR, and each rebase restarted a 25–45 min CI run. Separate
files cannot conflict.

At release, `scripts/changelog_fold.sh` (run by release-manager before it renames `## [Unreleased]`)
inserts every fragment newest-first under `## [Unreleased]` and deletes it, so the release notes,
the broadcast and `ailang docs search` read one file exactly as before. `make check-changelog` runs
`scripts/changelog_fold.sh --check`, which refuses a misnamed fragment, one that does not open with a
`### ` heading, or one carrying a `#`/`##` heading. This README is never folded.
