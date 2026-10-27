# DEV SMOKE — motoko 835c738b

Counts of lines for each `.md` file directly in `design_docs/planned/v0_55_0/` (subdirectories excluded), computed by `count.ail` in this directory (AILANG program, `FS` + `IO` effects only).

| File | Lines |
|---|---|
| m-ailang-executor-gemini-enterprise.md | 289 |
| m-ailang-lane-executor-shadow-sprint-plan.md | 143 |
| m-ailang-lane-executor-shadow.md | 303 |
| m-crew-core-cold-compile-followup.md | 439 |
| m-effect-latent-nested-callbacks.md | 470 |
| m-effect-row-open-closed-soundness.md | 412 |
| **Total** | **2056** |

## Notes

- Line counting convention: newline-terminated lines count as-is; a final
  unterminated line (if any) also counts as one line. All six files end with
  a trailing newline, so the count equals the number of `\n` characters.
- Sorted by file name (lexicographic).