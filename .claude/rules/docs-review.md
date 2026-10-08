---
paths:
  - "docs/docs/**"
  - "docs/LIMITATIONS.md"
  - "docs/TESTING.md"
  - "docs/VISION.md"
---

# Docs Review Dates

Every published docs page carries a review date in its frontmatter:

```yaml
---
title: ...
reviewed: 2026-10-05   # the day you read the page against the current code
reviewBy: 2027-01-05   # when it must be read again
---
```

- **New page:** add both keys. Default to three months out; up to six for a stable
  reference page. Use `reviewBy: never` only for a point-in-time record.
- **Reviewing a page:** read it against the code, fix what drifted, then set
  `reviewed` to today and move `reviewBy` forward. Only bump the dates after an
  actual read. Moving the date without reading the page defeats the point.
- **Editing a page for another reason:** leave the dates alone unless you checked the
  whole page.

`make check-doc-review` runs in CI. An overdue or undated page only produces a
`::warning::`. A malformed date, or `reviewed` later than `reviewBy`, fails.
`make docs-review-overdue` lists what is due, oldest first.

Pages a sync script regenerates are exempt and must not be dated by hand: the next
sync wipes the date. The exempt list is `generated()` in `scripts/check_doc_review.sh`.
Add a page there when you add a generator.
