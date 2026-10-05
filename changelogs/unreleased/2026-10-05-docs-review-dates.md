### Added — review dates on the published docs, with a warn-only CI gate (2026-10-05)

Every page under `docs/docs/` (plus `docs/LIMITATIONS.md`, `TESTING.md`, `VISION.md`) now carries a
`reviewBy:` date in its frontmatter, and a page that has been read against the code also carries
`reviewed:`. `make check-doc-review` runs in CI: an overdue or undated page is a `::warning::`
annotation and never fails a PR, because it arrives by the calendar and not by the change under test.
A malformed date, or `reviewed` later than `reviewBy`, fails. `make docs-review-overdue` lists what is
due, oldest first, as a backlog for the docs mission. The first 135 dates are staggered seven a week
from 2026-11-02, with the pages that have gone longest without a commit due first. Pages that sync
scripts regenerate are exempt. Convention: `.claude/rules/docs-review.md`.
