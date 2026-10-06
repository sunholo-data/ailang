### Fixed — Testing guide documented syntax and settings that do not exist (2026-10-06)

`docs/docs/guides/testing.md` now matches the current `ailang test`; every `ailang` example in it
was run with `ailang test`. Unit tests use `test "x" { expr }` (not `= expr`), inline tests use
`tests [ (input, expected), … ]`, properties use `property "x" { forall(…) => pred }` with `[int]`
list types, and preconditions are `if … then … else true` (there is no `==>` or `where`). Removed:
the `AILANG_TEST_RUNS/SEED/MAX_SIZE/MIN_INT/MAX_INT` environment variables (never implemented; the
case count is a fixed 100 and seeding is `--seed N` / `--random-seed`), `@generator` custom
generators, function-typed binders (no generator), and the JUnit report claim. The shrinking,
human and JSON output examples are now real `ailang test` output.

### Fixed — `ailang test --help` taught test syntax that does not parse (2026-10-06)

The help footer showed `test "name" = expression` and `property "name" (x: int) = ...`, which is probably where the testing guide's wrong syntax came from. It now shows the real forms, all three verified: `test "name" { expression }`, `property "name" { forall(x: int) => expr }` and inline `tests [(input, expected)]`.
