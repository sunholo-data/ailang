### Added — `ailang check --json` and `--format agent` now carry warnings (2026-11-10)

Pipeline warnings (the literal-only non-exhaustive-match warning, `STRICT_FALLBACK_001`,
`LIST_TAKE_AFTER_FLATMAP`, the split-argument-order hint) reached only a human's stderr:
`check --json` returned `"errors": []` and `--format agent` returned `ok: <file>`, so an
AI author on either channel could not see them. Both formats now carry them, additively:

- `--json` gains a `warnings` array (omitted when empty) of entries shaped like `errors`
  — `code`, `message`, `file`, `line`, `column`, `suggestion`. The non-exhaustive-match
  warning is code `ELB004` and now reports the file path rather than the module name.
- `--format agent` prints one `warn: file:line:col CODE: message → suggestion` line per
  warning **after** the `ok:` / error lines, so a reader of the first line is unaffected.

Known gap, unchanged here: on a warm compile cache the elaborator's warnings are not
re-derived, so a second `check` of an unchanged file shows none in any format
(`AILANG_NO_CACHE=1` shows them). Milestone M3 (warnings half) of
design_docs/planned/v1_0_0/m-v1-match-exhaustiveness.md (#2146).
