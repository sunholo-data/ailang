# M-IMPORT-ALIAS: Import aliases — ALREADY SUPPORTED (doc retracted)

**Status**: RETRACTED — original claim was false (verified 2026-06-03)
**Target**: n/a
**Priority**: P4 (near-non-issue — 0 real recent eval failures)
**Estimated**: 0 days
**Dependencies**: None

> **⚠️ RETRACTION (2026-06-03):** This doc originally claimed "AILANG has no import
> aliases" and proposed implementing them. **Both claims were false** — caught by
> running `ailang check` (the verification step that was skipped on first write).
>
> **VERIFIED — import aliases ALREADY work:**
> ```ailang
> import std/list as L                      -- ✓ COMPILES (module alias)
> import std/list (length as listLen)       -- ✓ COMPILES (per-name rename)
> import std/list (map) as L                -- ✗ rejected (selective import + module
>                                           --   alias COMBINED — the only unsupported form)
> ```
>
> **Frequency was also wrong.** The earlier "6% of recent compile failures" came from a
> broken detection heuristic (flagged any `as <word>` in an import line). Re-examined:
> **0 of 16 flagged recent failures actually fail on an import alias** — all 16 fail for
> unrelated reasons (parse errors elsewhere, `++` string-concat, undefined vars). The
> import line in those programs compiles fine.
>
> **And `log_file_analyzer` does NOT fail on aliases.** The original "9/9 frontier models,
> import alias is the primary cause" assertion was wrong — those failures are `++` and
> other parse errors. See `m-prompt-string-concat-plusplus.md` (the real, but largely-solved,
> cause).

## What (if anything) remains

The only genuinely-unsupported form is `import M (f, g) as X` — combining a selective
import list with a module-level alias. This is:
- Rare (no model produced it in any recent eval failure)
- Arguably nonsensical (if you've selected `f, g`, what would the alias `X` refer to?)

**Recommendation: take no action.** Optionally add a one-line note to the prompt's import
section that you can use `import M as X` OR `import M (f as g)` but not both combined. Even
that is probably unnecessary given zero observed failures.

## Axiom Compliance

Not applicable — no change proposed (doc retracted). The feature already exists.

## Lesson

This doc is preserved (rather than deleted) as a record of a process failure: it was
hand-written without running `ailang check` to verify the core claim, and without reading
the related-doc search. Both the "unsupported" claim and the "6% frequency" were false.
The fix — `ailang check` on a 3-line temp file — takes 10 seconds and would have prevented
the entire doc. Always verify language claims before asserting them in a design doc.

## Related Documents

- `m-type-constraints.md` — the OTHER hand-written doc with a false core claim (also corrected:
  AILANG has typeclasses with inferred constraints, contrary to the original draft)
- `m-prompt-string-concat-plusplus.md` — the actual (largely-solved) cause of log_file_analyzer failures
