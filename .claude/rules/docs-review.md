---
paths:
  - "docs/docs/**"
  - "docs/LIMITATIONS.md"
  - "docs/TESTING.md"
  - "docs/VISION.md"
  - "README.md"
  - "docs/src/**"
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

`make check-doc-samples` (CI, and `ci-quick` in the pre-push hook) runs `ailang check` on
every ```` ```ailang ```` block in `docs/docs/**`, README.md, docs/LIMITATIONS.md and the
active prompt; on every `.ail` file the docs import with raw-loader (examples/docs,
reference, debug as well as runnable — `make verify-examples` covers only runnable); and
on AILANG embedded as a `{`...`}` literal in docs/src. Tag a block after the language:

| Fence | Use for | Gate |
|---|---|---|
| ```` ```ailang ```` | a complete program | must check; a `module` line is synthesised only for a fragment without `main` |
| ```` ```ailang file=pkg/mod.ail ```` | one file of a multi-file example | staged so siblings can import it, then checked |
| ```` ```ailang parse ```` | a fragment using names defined elsewhere | must parse (`ailang fmt`) |
| ```` ```ailang error=PAR015 ```` | a deliberate anti-example | the FIRST error must be exactly that code |
| ```` ```ailang sketch ```` / ```` repl ```` | pseudo-code, elisions / a REPL session | not run; must be listed in `scripts/doc_samples_unchecked.txt` |

A block that defines `main` must carry its own `module` line (a reader copies it into a
file: MOD014 otherwise). Render imported files with `<CodeBlock language="ailang">`.
**Structural defects fail and cannot be listed**: AILANG fenced as `typescript`, `text`
or untagged (fence output as `text`; `not-ailang` opts a false positive out), an unknown
tag, a fence that never closes or swallows the next one, an import that does not resolve.

Known failures sit in `scripts/doc_samples_baseline.txt` (page + content hash + code).
Both lists only shrink, **as sets**, against the PR base / previous tip: any new line
fails, so fix or tag what you touch. A listed block that passes, vanishes or fails with a
different code fails too. A new sketch/repl line passes only with `# reviewed: <why>`.
The mis-fence detector is keyword hints plus a parse probe; it still misses AILANG in
other languages' fences and keyword-less snippets (`1 + 2`). Reproduce one block:
`make check-doc-samples DOC_SAMPLES_ARGS='-only <page>:<line> -keep /tmp/ds'`.
