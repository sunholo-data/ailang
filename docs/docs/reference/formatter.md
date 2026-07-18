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
| 2 | Usage, read, comment, parse, print, round-trip, or write error |

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

## Comments (Phase 1 refusal)

The AILANG AST currently carries no comment or trivia information, so a naive
reprint would silently delete comments. Phase 1 is therefore comment-non-
preserving but **not comment-destructive**:

- Before parsing, `fmt` detects any real comment with a lossless lexical scan
  that distinguishes comment introducers (`--`, `//`) from comment-like text
  inside strings, character literals, regex literals, and quasiquote templates.
- If a file contains any comment, `fmt` prints
  `path: comments are not yet supported by ailang fmt` to stderr, exits with code
  2, and leaves the file **byte-identical**. It never removes a comment.

This keeps the formatter immediately useful for generated and comment-free code
while never risking user text.

### Phase 2 (planned): lossless comment preservation

A separately-scheduled Phase 2 will attach comments losslessly without changing
the grammar. The lexer will expose comments and exact byte spans through a
separate lossless token stream, and AST nodes (or a parallel syntax-envelope
index) will gain full source spans. Comments will then attach deterministically:

- A comment after a node on the same line attaches as a **trailing** comment to
  the nearest preceding node.
- A comment before the next node with no intervening blank line attaches as a
  **leading** comment to that node.
- A comment separated by a blank line, between siblings, or before a closing
  delimiter attaches as a **floating** comment to the smallest enclosing ordered
  list at that boundary.

Until Phase 2 lands, the Phase 1 refusal is mandatory and is never weakened to a
warning.
