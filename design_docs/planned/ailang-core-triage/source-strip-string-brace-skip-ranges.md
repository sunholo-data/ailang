# source_strip test skip-range scan miscounts braces inside string literals

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: "skip range", "source strip", "test block", "named test", "strip test" across design_docs/ — no existing doc rules on the strip-range computation. Closest are `design_docs/implemented/v0_29_0/m-named-test-blocks.md` and `design_docs/planned/v0_33_1/m-named-test-body-check-semantics.md`, which govern test-block parsing/semantics, not the stripper.
- **Estimate**: `~20-40 lines in internal/testing/source_strip.go`

`testAndPropertySkipRanges` (internal/testing/source_strip.go) scans raw runes for
`{`/`}` from each test/property decl's start line, with no string-literal or
comment awareness. An unbalanced brace inside a string literal in a test body
(e.g. a malformed JSON payload `"{\"partial"`) keeps the depth scan from ever
returning to 0, so the skip range collapses to the decl's first line and the
rest of the block leaks into the stripped base for every subsequent test —
turning one bad literal into `PAR_NO_PREFIX_PARSE` failures in all other tests
of the module, reported at synthesized `_namedtest_body_*.ail:1:1` positions far
from the real source.

Why design-doc rather than direct-fix: the fix is a real decision with more than
one acceptable shape. Options reported and plausible: (a) string-aware scanning
in the stripper (handles strings but not comments/interpolation), (b) reuse the
lexer token stream so string/comment boundaries are authoritative, (c) fall back
to the AST decl spans instead of re-scanning. The choices differ in
lexer-coupling and whether the stripper keeps duplicating lexical knowledge;
that is a disagreement-surface (row 3), and the change exceeds
DIRECT_FIX_MAX_LINES regardless. Workaround in the meantime: keep brace-balanced
string literals in test bodies.
