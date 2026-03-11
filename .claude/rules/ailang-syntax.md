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

## Critical Syntax Notes

- **Import Aliasing (v0.4.8):** `import std/list as List (map, filter)`
- **Relaxed Module Matching:** `ailang run --relax-modules` or `AILANG_RELAX_MODULES=1`
- **Nullary functions (v0.4.6+):** Call with empty parens: `getArgs()`, `readLine()`
- **Type parameters:** Use `[T]` NOT `(T)`
- **Match in blocks:** Known parser bug — extract to helper function
- **Flags before filename:** `ailang run --caps IO,FS --entry main module.ail`

**See also**: `docs/LIMITATIONS.md` | `examples/`

Use the `use-ailang` skill for guided AILANG development.
