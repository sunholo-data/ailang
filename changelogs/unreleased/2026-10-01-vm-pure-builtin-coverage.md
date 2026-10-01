### Fixed — whole-number float literals in `test` blocks evaluated as Int (#1448)

- `test "…" { root(4.0) == 2.0 }` failed with `_math_sqrt: expected FloatValue … got *eval.IntValue`.
  `ailang test` prints a folded test body back to source and re-parses it, and `ast.Literal.String()`
  spelled `float64(4)` as `4`, which re-parsed as an int. Fractional literals (`2.25`) were unaffected.
- `ast.FormatFloat` is now the one canonical float spelling (`4.0`, `1e+20`; NaN/±Inf print as
  `(0.0 / 0.0)` / `(±1.0 / 0.0)` since they have no literal form). `Literal.String()` and the source
  formatter both use it, so the older `Executor.EvaluateExpression` path is fixed too.

### Fixed — `--strict-bytecode` runs 121 more pure builtins, and says *why* for the rest (#1447)

- **Generic adapter.** A pure registry builtin with no native VM implementation, whose signature has no
  type variables, Map, function or ADT parameters, now runs on the VM: arguments convert to evaluator
  values, the builtin's registered Go `Impl` runs (not the tree-walking evaluator), and the result converts
  back (`Option`/`Result` results included). This covers 121 builtins, among them `std/string`, `std/bytes`,
  `std/crypto`, `std/regex`, datetime, URL, vector, `list.range` and the bitwise Int operators `^ & << >>` (#1450 — the Stapledon PRNG blocker). They take the indices after the native
  `OpBuiltinCall` entries, so no opcode or image format changed.
- **Honest message.** `compiler: effectful builtin "__list_reverse" not yet wired (Phase 2E)` is now
  `compiler: pure builtin "__map_size" has no VM implementation (map-value: the VM has no Map value)`.
  Effectful builtins keep the Phase 2E message.
- **Ratchet.** `TestPureBuiltinCoverage` puts every pure builtin in one bucket (native 96, adapted 121,
  unported 33, each with its computed reason). It fails when a new pure builtin lands in none, or an
  allowlisted one becomes covered.
- **Lockstep enforced.** The compiler/VM builtin name tables now live in `internal/bytecode` (the contract
  between them). The `validateBuiltinTables` check that comments cited, but which did not exist, now runs at
  `vm` package init. The VM↔evaluator value converters moved from `internal/runner` to `internal/vm`
  (`BytecodeToEval`/`EvalToBytecode`) — one implementation for the bridge and the adapter.

### Added — native VM ports of polymorphic list builtins (#1447)

- `std/list` `reverse`, `take`, `drop`, `zip`, `contains`, `head` and `extract` now run natively on the VM, so
  they work under `--strict-bytecode` for any element type, ADTs and closures included (the adapter cannot
  convert those). Semantics match the evaluator exactly; a parity table checks each against the registered
  `Impl` (edge `n`, negative offsets, empty lists, nested lists).
- `examples/runnable/vm_strict_pure_builtins.ail` prints the same line under the interpreter, `--bytecode`
  and `--bytecode --strict-bytecode`. The Stapledon repro `reverse([1,2,3])` now runs strict.
- VM coverage of pure builtins after this sprint (250 callable): native 96, adapted 121, unported 33 (Map values,
  polymorphic arrays, XML/HTML node types, two closure-taking list ops).
- Found while testing it, filed separately: `show()` of an ADT renders `<adt#0 6>` on the VM, but
  `Some(6)` in the interpreter (#1453, pre-existing).
