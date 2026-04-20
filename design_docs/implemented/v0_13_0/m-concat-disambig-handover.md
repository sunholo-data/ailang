# M-CONCAT-DISAMBIG Sprint Handover

**Status**: M1-M5 complete and committed. M6-M8 pending.
**Branch**: `dev`
**Last commit**: `089228f3` (M5_GO_TEST_MIGRATION)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-CONCAT-DISAMBIG.json`

## What's done (M1-M5)

### M4 (commit `bc0f444f`) — compiler change
- `typechecker_operators.go`: `++` now unifies both operands with `[a]`; string heads produce the helpful error.
- `op_lowering.go` / `op_table.go`: `OpConcat` default is `List`; `Types: [List]`.
- `std/string.ail`: added `concat: [string] -> string` via `_str_join(xs, "")`; `repeat` migrated off string `++`.
- Stdlib + internal fixture migrations: `std/net.ail`, `std/jwt.ail`, `internal/dashboard_transforms/*.ail`, `internal/apiserver/templates/web_app/api/handlers.ail`, `internal/pkg/testdata/multi_module/src/core.ail`, inline .ail strings in `internal/repl/*_test.go` + `internal/apiserver/server_test.go`.
- New tests: `internal/pipeline/concat_string_error_test.go` — `TestConcatStringErrorMessage`, `TestConcatListPolymorphic`, `TestStdlibConcatSignature`.
- Net compiler LOC reduction: ~82 lines.

### M5 (commit `089228f3`) — Go test migration
- Deleted from `concat_operator_test.go`: `TestConcatRecursiveString`, `TestConcatConcreteString`, `TestConcatStringVarPlusConcreteString`, `TestConcatNestedRecursion`, `TestConcatMixedTypes` (previously skipped).
- Dropped string-concat cases from `TestOpLowering_Concat` and `TestConcatOperator_TypeGuidedLowering`.
- `TestOpLowering_FallbackPath` now expects `concat_List` as fallback default.
- `go test ./...`: zero failures. `make lint`: clean (two pre-existing `bytecode/compiler/` warnings are not from this sprint).

## What remains

### M6_EXAMPLE_MIGRATION — where I stopped

**The hard blocker**: 133 `.ail` files under `examples/` match string-`++` heuristics (`grep -lE '" \+\+ |\+\+ "|\+\+ show\(|\+\+ intToStr\(|\+\+ floatToStr\(|\+\+ toString\(|\+\+ _str_' examples/ --include='*.ail'`). 87/165 are currently failing `make verify-examples`. Some of those failures pre-date this sprint (unrelated); the rest are from the new `++` restriction.

**Strategy**: don't edit all 133 files blindly — many `++` uses are still valid (list concat). Instead:

1. Run `make verify-examples` and capture output: which examples fail with `lists only` in the error message? Those are the migration targets. The rest (failing for other reasons, e.g. AI capability unavailable) are out of scope.
2. For each failing file, the migration pattern is one of:
   - `"prefix " ++ show(x) ++ " suffix"` → `"prefix ${show(x)} suffix"`
   - `"prefix " ++ name` → `"prefix ${name}"`
   - Multiple concatenations: prefer single `"${a}${b}${c}"` interpolation or `concat([a,b,c])` for list-driven building.
3. Leave `examples/archive/` and `examples/experimental/` untouched — they're historical/broken-on-purpose.
4. Some files may have genuine list `++` mixed with string `++` — verify each edit is string-context before changing.

**Files confirmed to fail with `lists only` error** (from spot checks):
- `examples/runnable/string_chars.ail:65:35`
- `examples/runnable/stdlib_demo.ail:10:22`
- (Full list needs a complete `make verify-examples` run)

**Helpful commands once shell is working**:
```bash
# Identify which examples fail with the new error message only
for f in $(find examples/runnable -name '*.ail'); do
  err=$(ailang run --entry main --caps IO,FS,Net,Clock,Random "$f" 2>&1 | grep "lists only" | head -1)
  [ -n "$err" ] && echo "$f"
done
```

### M7_BENCHMARK_PROMPT_DOCS

**Benchmarks**: scan `benchmarks/` for string `++` usage in prompts/expected outputs. Replace with interpolation in the prompt text and expected output.

**Active teaching prompt**: `prompts/devtools/v0.8.0.md` + its compact variant, plus `cmd/ailang/prompts/devtools/v0.8.0.md` (embedded copy). Update the operator section to:
- Mark `++` as list-only.
- Show `"${x}"` interpolation as the primary string-building pattern.
- Mention `concat([parts])` and `join(sep, parts)` as alternatives.

**Docs sweep** (~30 files under `docs/docs/`):
- `docs/docs/reference/language-syntax.md` — operator table.
- `docs/docs/guides/getting-started.md` — any string-building examples.
- Stdlib reference pages for `std/string` — add `concat` entry, note `++` is list-only.
- CHANGELOG is already updated under `[Unreleased] — v0.13.0 in progress`.

### M8_FINAL_VALIDATION

1. `make ci` must be clean.
2. `make verify-examples` must show 0 string-`++` failures (other pre-existing failures OK but note in retro).
3. Run eval subset on at least `type_unify`, `config_file_parser`, `graph_bfs` benchmarks — compare against v0.12.1 baseline.
4. Move `design_docs/planned/v0_13_0/m-concat-disambiguation.md` → `design_docs/implemented/v0_13_0/`.
5. Use `scripts/finalize_sprint.sh M-CONCAT-DISAMBIG v0_13_0`.
6. Send handoff message to release-manager for v0.13.0 cut.

## Known pre-existing failures unrelated to this sprint

- `internal/bytecode/compiler/lambda.go::withBound` — lint "unused"
- `internal/bytecode/compiler/expr.go::freeIfTemp` — lint "unused"

These were flagged by `make lint` before M4; do not touch under this sprint.

## Uncommitted local changes unrelated to sprint

Two files are modified but not staged and not part of this sprint:
- `cmd/ailang/prompts/python.md` (Python version pinning documentation)
- `cmd/ailang/prompts/v0.11.4.md` (+25 lines, unknown)

Leave them alone unless the next session explicitly wants to commit them separately.

## Environment note

Shell permissions were intermittent during the M6 attempt — `find`, and some `bash -c` invocations over the examples directory were being denied. If it recurs, the workaround is:
- Use the **Grep** tool (files_with_matches mode) for discovery.
- Use the **Edit** tool per file with `replace_all: false` and clearly unique old_string.
- Avoid `find | while read` shell patterns; prefer `Glob` or `Grep`.
