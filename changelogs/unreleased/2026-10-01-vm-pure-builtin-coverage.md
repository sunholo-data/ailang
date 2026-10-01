### Fixed — whole-number float literals in `test` blocks evaluated as Int (#1448)

- `test "…" { root(4.0) == 2.0 }` failed with `_math_sqrt: expected FloatValue … got *eval.IntValue`.
  `ailang test` prints a folded test body back to source and re-parses it, and `ast.Literal.String()`
  spelled `float64(4)` as `4`, which re-parsed as an int. Fractional literals (`2.25`) were unaffected.
- `ast.FormatFloat` is now the one canonical float spelling (`4.0`, `1e+20`; NaN/±Inf print as
  `(0.0 / 0.0)` / `(±1.0 / 0.0)` since they have no literal form). `Literal.String()` and the source
  formatter both use it, so the older `Executor.EvaluateExpression` path is fixed too.
