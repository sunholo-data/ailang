### Fixed — `make test-codegen` passes again and runs in CI

`tests/codegen-harness` (`make test-codegen`) had stopped compiling: `list_ops.ail` annotated
`zip` as `[[int]]` (it returns `[(int, int)]`). It had also never run in CI, despite the
"CI harness" commit that added it, and it tracked a stray compile-cache manifest. It is now
`tests/codegen-harness/run.sh`, which does three things, all from a temp dir, cache included:

1. **build:** every `tests/codegen-harness/*.ail` emits Go that builds.
2. **parity:** `parity/stdlib_parity.ail` calls 27 std functions (list, string, math, option)
   through `--emit-go`; the generated Go's `summary()` must print exactly what
   `ailang run --entry summary` prints.
3. **known failures:** `known_failures/known_failures.tsv` lists the Go-emitter bugs the
   harness hit, one case per issue (#2241, #2259, #2260, #2261, #2262, #2263, #2264). Each
   case must still fail with its recorded message. A case that starts to pass, fails
   differently, or is not listed fails the harness, so the list cannot rot.

The harness programs that hit an emitter bug moved to the known-failure list rather than being
rewritten around it (`string.length`, `startsWith`/`endsWith` on a module-level `let`). The "Test
AILANG -> Go Codegen Pipeline" workflow now runs `make test-codegen`, and also runs on changes
to `std/**`, `tests/codegen-harness/**` and the Go helper specs
(`internal/builtins/registry_codegen*.go`). No emitter change.
