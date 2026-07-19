# `ailang fmt` — Canonical Source Formatter

`ailang fmt` rewrites AILANG source into one canonical textual form. AILANG's
grammar intentionally accepts several spellings for statement sequencing (an
equation body with semicolons, a braced semicolon sequence, and a braced
newline sequence all parse to the same AST). `fmt` collapses those equivalent
spellings to a single stable representation so repositories and generated
programs do not drift.

The formatter is a developer and agent convenience, **not a correctness gate**.
It changes presentation only: `Parse(fmt(x))` is structurally identical to
`Parse(x)` for every file it accepts.

## Usage

```bash
ailang fmt <file.ail>          # write canonical source to stdout (exactly one file)
ailang fmt --write <files...>  # rewrite each file in place
ailang fmt --check <files...>  # list files that are not canonical (never writes)
```

- **Default (stdout) mode** accepts exactly one file and writes the canonical
  source to stdout, leaving the file unchanged.
- **`--write`** accepts one or more files. It validates every input first
  (comment preflight, parse, format, and round-trip verification in memory); if
  any file fails, **no file is changed**. Each changed file is then replaced
  individually and atomically (a same-directory temporary file plus `os.Rename`),
  preserving the original file mode. Cross-file crash atomicity is not claimed.
- **`--check`** accepts one or more files, prints each non-canonical path to
  stdout, and never modifies files. "Canonical" means byte-equal to formatter
  output, including the final newline. This mode is CI-friendly.
- `--write` and `--check` are mutually exclusive.

### Exit codes

| Code | Meaning |
|---:|---|
| 0 | Formatting succeeded; or every file is canonical in `--check` mode |
| 1 | `--check` found at least one non-canonical file (no operational error) |
| 2 | Operational error: usage, read, print, round-trip, envelope, or write |
| 3 | Input parse error: the file does not parse |

Exit **3** is distinct from exit **2** so tooling can tell a *mid-edit,
not-yet-parseable* file (defer and retry) apart from a *genuine formatter
failure* (surface it). Consumers that only test "nonzero = failed" are
unaffected.

The formatter is **fail-closed**: on any error it makes no write and never falls
back to the original source.

## Canonical form

- **Sequences use braces and newlines.** A multi-expression block prints one
  expression per line, with the final value expression bare and no sequence
  semicolons:

  ```ailang
  func compute() -> int {
    let x = 1 + 2
    let y = x * 3
    y + 10
  }
  ```

- **Single-expression equation bodies use `= expr`.** A `func f() = expr` stays
  on one line.
- **`let ... in` stays explicit.** An explicit `let name = value in body`
  expression is preserved as written; only statement-sequence lets (which have no
  `in`) are printed as newline statements.
- **Layout is fixed:** two-space indentation, LF line endings, exactly one final
  newline, one blank line between top-level declarations, and module → imports →
  declarations in original AST order. Imports, symbols, match cases, record
  fields, and declarations are never reordered.
- **Parentheses are precedence-driven.** Redundant source parentheses are
  dropped; parentheses are reconstructed wherever omitting them would change the
  parsed expression.
- **Literals are escaped canonically** through a single escaping routine.

The formatter performs no semantic rewrites: no constant folding, import/decl
sorting, dead-code removal, alpha-renaming, or annotation inference.

## Comments (lossless preservation)

`fmt` preserves comments losslessly: every input comment appears in the output
exactly once, deterministically placed. Comments are collected by a lossless
lexical scan that distinguishes real introducers (`--`, `//`) from comment-like
text inside strings, character literals, regex literals, and quasiquote
templates, then attached to structural boundaries via a formatter-owned
**token-anchored envelope** (the AST grammar is unchanged). Attachment is
deterministic:

- A comment after a node on the **same source line** attaches as a **trailing**
  comment to that node.
- A comment directly above the next node with **no intervening blank line**
  attaches as a **leading** comment to that node.
- A comment separated by a blank line, between siblings, or before a closing
  delimiter attaches as a **floating** comment to the smallest enclosing ordered
  list at that boundary.
- Comments before the module attach at the file's top boundary; consecutive
  comments preserve source order and blank-line grouping.

### Fail-closed carve-outs (never lossy)

A comment that cannot be placed on a stable boundary is **refused fail-closed**
(exit 2, file left byte-identical) rather than dropped or relocated:

- A comment inside a `${...}` string-interpolation hole (interior interpolation
  attachment is deferred; the refusal makes silent deletion structurally
  impossible).
- A comment interior to an expression the formatter emits inline (e.g. inside a
  top-level `let ... in` chain that collapses onto fewer lines).

These are the honest boundaries of lossless coverage: on the example corpus the
interpolation-comment rate is 0% and the inline-interior rate is ~15% of
parse-valid files; the remaining ~85% format losslessly and idempotently. The
formatter never removes or moves a comment.

### NFC normalization

Comment bytes are emitted verbatim from the **NFC-normalized** source (the same
normalization boundary the lexer applies to all input). For NFC input — the
entire example corpus — output comment bytes are byte-identical to the source. A
non-NFC input file has its comment bytes NFC-normalized on first format, after
which `fmt` is idempotent.
