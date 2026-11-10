---
paths:
  - "docs/docs/**"
  - "docs/LIMITATIONS.md"
  - "docs/TESTING.md"
  - "docs/VISION.md"
  - "README.md"
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

## Code blocks: the doc-sample gate

`make check-doc-samples` (CI) runs `ailang check` on every ```` ```ailang ```` block in
`docs/docs/**`, README.md, docs/LIMITATIONS.md and the active prompt. Inline blocks are
fine; a failing untagged one is not. Tag a block with words after the language:

| Fence | Use for | Gate |
|---|---|---|
| ```` ```ailang ```` | a complete program | must check (a missing `module` line is synthesised) |
| ```` ```ailang file=pkg/mod.ail ```` | one file of a multi-file example | staged so sibling blocks can import it, then checked |
| ```` ```ailang parse ```` | a fragment using names defined elsewhere | must parse (`ailang fmt`) |
| ```` ```ailang sketch ```` | pseudo-code, elided bodies, notation | not run; counted per page |
| ```` ```ailang repl ```` | a REPL transcript or REPL input | not run; counted |
| ```` ```ailang error=PAR015 ```` | a deliberate anti-example | must FAIL with that code; passing is a failure |

Never fence AILANG as `typescript` or leave it untagged: that is a failure too (fence
CLI output as `text`). Known failures sit in `scripts/doc_samples_baseline.txt`, keyed
by page + content hash; it only shrinks. A block that now passes must leave it, and
editing a listed block re-keys it, so you fix or tag what you touch. Reproduce one block:
`make check-doc-samples DOC_SAMPLES_ARGS='-only <page>:<line> -keep /tmp/ds'`.
