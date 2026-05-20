---
paths:
  - "examples/**/*.ail"
  - "stdlib/**/*.ail"
  - "stdlib/**"
  - "prompts/**"
  - "tests/**/*.ail"
---

# Writing AILANG Code

| Command | Purpose | Size |
|---------|---------|------|
| `ailang prompt` | Full syntax reference (0-shot generation) | ~1600 lines |
| `ailang devtools-prompt` | Toolchain/CLI reference | ~600 lines |
| `ailang agent-prompt` | Minimal agent coding guide | ~180 lines |

## Canonical syntax source

**Run `ailang prompt` for the authoritative, up-to-date syntax reference.** Do NOT pattern-match from Haskell/Elm/OCaml training data — AILANG has diverged (e.g. `++` is list-only since v0.13.0; strings use `"${expr}"` interpolation). When in doubt about a construct, `ailang prompt | grep <topic>` is the source of truth.

## Quick reminders

- **Relaxed Module Matching:** `ailang run --relax-modules` or `AILANG_RELAX_MODULES=1`
- **Type parameters:** Use `[T]` NOT `(T)`
- **Match in blocks:** Known parser bug — extract to helper function
- **Flags before filename:** `ailang run --caps IO,FS --entry main module.ail`

**See also**: `docs/LIMITATIONS.md` | `examples/`

Use the `use-ailang` skill for guided AILANG development.
