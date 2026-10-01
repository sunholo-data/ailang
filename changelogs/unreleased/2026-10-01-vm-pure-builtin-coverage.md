### Fixed — whole-number float literals in `test` blocks evaluated as Int (#1448)

- `test "…" { root(4.0) == 2.0 }` failed with `_math_sqrt: expected FloatValue … got *eval.IntValue`.
  `ailang test` prints a folded test body back to source and re-parses it, and `ast.Literal.String()`
  spelled `float64(4)` as `4`, which re-parsed as an int. Fractional literals (`2.25`) were unaffected.
- `ast.FormatFloat` is now the one canonical float spelling (`4.0`, `1e+20`; NaN/±Inf print as
  `(0.0 / 0.0)` / `(±1.0 / 0.0)` since they have no literal form). `Literal.String()` and the source
  formatter both use it, so the older `Executor.EvaluateExpression` path is fixed too.

### Fixed — `--strict-bytecode` runs 81 more pure builtins, and says *why* for the rest (#1447)

- **Generic adapter.** A pure registry builtin with no native VM implementation, whose signature has no
  type variables, Map, function or ADT parameters, now runs on the VM: arguments convert to evaluator
  values, the builtin's registered Go `Impl` runs (not the tree-walking evaluator), and the result converts
  back (`Option`/`Result` results included). This covers 81 builtins, among them `std/string`, `std/bytes`,
  `std/crypto`, `std/regex`, datetime, URL, vector and `list.range`. They take the indices after the native
  `OpBuiltinCall` entries, so no opcode or image format changed.
- **Honest message.** `compiler: effectful builtin "__list_reverse" not yet wired (Phase 2E)` is now
  `compiler: pure builtin "__map_size" has no VM implementation (map-value: the VM has no Map value)`.
  Effectful builtins keep the Phase 2E message.
- **Ratchet.** `TestPureBuiltinCoverage` puts every pure builtin in one bucket (native 82, adapted 81,
  unported 40, each with its computed reason). It fails when a new pure builtin lands in none, or an
  allowlisted one becomes covered.
- **Lockstep enforced.** The compiler/VM builtin name tables now live in `internal/bytecode` (the contract
  between them). The `validateBuiltinTables` check that comments cited, but which did not exist, now runs at
  `vm` package init. The VM↔evaluator value converters moved from `internal/runner` to `internal/vm`
  (`BytecodeToEval`/`EvalToBytecode`) — one implementation for the bridge and the adapter.
