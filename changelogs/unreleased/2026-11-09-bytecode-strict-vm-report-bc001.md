### Added — `ailang check --strict-bytecode`: BC001 for every function the strict VM would refuse (2026-11-09)

A function the bytecode compiler cannot lower is tagged evaluator-only and run through
the evaluator bridge. Under `--strict-bytecode` there is no bridge. Until now the first
sign of this was a `vm: CALL: … is evaluator-only … but no interop bridge is wired`
trap in the middle of the run, and a plain `ailang check` said nothing.

`ailang check --strict-bytecode <file>` now compiles the image too. It reports one
positioned `BC001` for each evaluator-only function the file declares, whether or not
anything calls it, and for each imported function those functions reach. The check
then exits 1. Human, `--json` and `--format agent` output all carry the
`file:line` and the compiler's reason.

An evaluator-only function that an import brings in but the program never reaches is
not reported, because it can never trap. Importing `std/list`, for example, does not
fail on `takeMap`. Combining the flag with `--package`, `--verify` or a directory is
refused. A plain `check` is unchanged. Error page: `docs/docs/reference/errors/bc001.md`.
Design: `design_docs/planned/v0_53_1/m-bytecode-strict-256-reg-ceiling.md` (Phase A).

### Fixed — strict runs and tests fail before dispatch; `--bytecode` no longer falls back silently (2026-11-09)

- `ailang run --bytecode --strict-bytecode` lists every evaluator-only function the
  entry can reach as `BC001` and fails before the entry runs, so no side effect
  happens first. Before, it trapped at the first such call. **Behaviour change:**
  reachability is static. A strict run whose input never takes the branch that calls
  an evaluator-only function (for example a `--help` path in a program that can call
  `std/ai.call`) used to succeed and is now rejected. Strict mode now means "this
  program runs entirely on the VM", whatever the input. Use `--bytecode` without
  `--strict-bytecode` to keep the bridge.
- `ailang test --strict-bytecode` does the same for each test body, naming the helper
  instead of the VM trap.
- `ailang run --bytecode` without `--strict-bytecode` prints a stderr warning naming
  those functions: the exact count and the first ten. Stdout is unchanged. The
  whole-program "bytecode path unavailable; falling back to evaluator" warning used to
  be hidden by `--quiet`, which is `run`'s default. It is now always printed.

The census over `std/` and `examples/` (601 files, 538 checked) found 232 unique
evaluator-only functions. None hit the 256-register ceiling after the v0.54.13/14
list-literal and else-if chain fixes. The remainder are unwired effectful builtins
(133), callers passing values the bridge cannot carry (62), budget/mode frames (22),
closure-arg builtins (11) and 4 others. See
`design_docs/planned/v0_53_1/m-bytecode-strict-256-reg-ceiling-census.md`.
