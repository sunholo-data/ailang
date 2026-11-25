### Added — Machine-applicable quick-fix edits in `ailang check --json` and `ailang check --fix`

- `ailang check --json` now emits structured `edits` (`file`, `start_line`, `start_col`, `end_line`, `end_col`, `new_text`, `title`) for compiler-verified mechanical fixes:
  - `PAR018` (`let rec` → `letrec`)
  - `PAR_HASH_COMMENT` (`#` → `--`)
  - `IMP010` stdlib symbol renames (`nthChar` → `charAt`) and redundant auto-imported builtin removal (`show`)
  - Unambiguous missing stdlib imports (`undefined variable: foldl` → `import std/list (foldl)`)
- Added `ailang check --fix <file>` to apply non-overlapping mechanical fixes in place (up to 4 passes) before the final check.
