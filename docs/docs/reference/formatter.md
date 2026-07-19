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

## Adoption

`fmt` is opt-in everywhere. Nothing forces canonical form: `make ci` does not
gate on it, no hook is mandatory, and drift is reported rather than rejected.
The pieces below let harness and CI authors integrate the formatter safely.

### Exit-code contract for hooks

Any wrapper around `ailang fmt --write` (a post-edit hook, a per-edit harness
step) keys its behavior on the exit code:

| Exit | Meaning | Hook interpretation |
|---:|---|---|
| 0 | Formatted, or every file canonical (`--check`) | Success. The file may have been rewritten (`--write`). An optional one-line confirmation is fine. |
| 1 | `--check` only: at least one non-canonical file | Drift signal — CI reports it. Post-write hooks use `--write` and never see exit 1. |
| 2 | Genuine operational error: usage, read, print, round-trip, envelope, or write failure (an unrecovered panic also exits 2) | **Surface `fmt`'s stderr to the agent as advisory context, then exit 0.** Never silent, never blocking. |
| 3 | Input does not parse | **The ONLY silent case.** Agents write partial files mid-edit; the hook defers until the file parses again (clause 5). |

### Contract clauses

Harness authors rely on five properties:

1. **Idempotence** — `fmt(fmt(x)) == fmt(x)` byte-for-byte. Hooks may fire
   repeatedly (after every edit, in loops) with no oscillation, and compose
   with other write-then-read tooling.
2. **Meaning preservation** — `Parse(fmt(x)) ≡ Parse(x)`. A hook can never
   change what the code does; formatting is safe to interleave anywhere in an
   edit loop.
3. **Atomicity** — `--write` validates in memory and replaces atomically per
   file; a mid-hook crash never leaves a truncated file.
4. **Non-blocking AND non-silent** — every wrapper MUST exit 0 regardless of
   `fmt`'s exit code (a formatter must never wedge a turn or a commit), but MUST
   capture `fmt`'s stderr (never `2>/dev/null`) and surface every non-exit-3
   failure as advisory context. Discarding diagnostics conflates "not
   parseable yet" with read/round-trip/envelope/write/panic failures — a silent
   fallback, which this project's axioms forbid.
5. **Never format what doesn't parse** — `--write` on a not-yet-parseable file
   exits **3** and writes nothing; the hook defers silently until the file
   parses. This is the sole sanctioned silence: mid-edit parse failure is an
   expected state, not a fault.

### CI drift check: `make fmt-check-ail`

```bash
make fmt-check-ail
```

Runs `ailang fmt --check` over `examples/**/*.ail` and `stdlib/**/*.ail`. It
exits **0** when every file is canonical and **1** (listing the drifted paths)
when any file has drifted. It is a **standalone** target — it is deliberately
**not** wired into `make ci`, so it never gates the build. (The name is
`fmt-check-ail`, not `fmt-check`, because `make fmt-check` is already the Go
`gofmt` CI gate.) Run it locally, or add it as an advisory (non-required) CI
step. Gating CI on canonical form would require a one-time mass-reformat
commit, which is a separate decision.

### Claude Code PostToolUse hook

`scripts/hooks/format_ail.sh` formats `.ail` files after every `Edit`/`Write`.
It is non-blocking (always exits 0), non-silent (surfaces every non-exit-3
failure via `hookSpecificOutput.additionalContext`), and bounded (a portable
`date +%s` deadline plus a SIGTERM→grace→SIGKILL kill guard — no GNU `timeout`
dependency, so a hung `fmt` can never wedge a turn). Register it opt-in in
`.claude/settings.json`, alongside the Go hook:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/scripts/hooks/format_go.sh"
          },
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/scripts/hooks/format_ail.sh"
          }
        ]
      }
    ]
  }
}
```

The hook requires `jq` on `PATH`; a missing `jq` surfaces an advisory note and
no-ops rather than silently skipping.

### Motoko per-edit contract (cross-repo)

The Motoko harness already returns a per-edit `typecheck` field on every edit
result. The adoption contract for the fork: **after an edit whose `typecheck`
is green, run `ailang fmt --write <file>`; on exit ≠ 0, keep the file
as-written, continue, and attach `fmt`'s captured stderr to the edit result**
(the same channel the `typecheck` field rides on) — never discard it (clause
4). Note the asymmetry with the Claude Code hook: because this contract fires
only after a green type-check (which implies the file parses), **exit 3 is
unexpected here** and is surfaced like any other failure rather than silently
deferred. Formatting only type-checked files means the model always sees its
own code in canonical form on the next read — passive convergence pressure with
zero added failure modes and zero swallowed diagnostics. This is a
motoko-fork change; only the contract text lives here.
