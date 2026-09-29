### Changed — `## [Unreleased]` stays empty between releases; entries go in fragments only (2026-09-29)

`make check-changelog` (CI and the pre-push `ci-quick`) now refuses any `### ` section written directly
under `## [Unreleased]` in the active changelog, naming its line. Fragments (#1382) alone did not change
the habit: the day they landed, five entries were still written into the shared block, and each one
conflicts with every other open branch. The five were moved into `changelogs/unreleased/`. The release
commit still folds fragments and renames the heading in one step, so the rule holds at every commit.
