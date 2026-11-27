### Added — Pipe `|>` Call-Argument Placeholder, Block Newline Holes & `IMP010` Auto-Move (`M-V1-RC1-READINESS-AND-DX-SWARM` M3)

- **Pipe `|>` explicit `_` call-argument placeholder (`D2=a`)**: `lhs |> f(a, _, c)` desugars to a hygienic LHS-first `let` binding in `internal/elaborate/expr_simple.go`, supporting chained data-last (`std/list`) and data-first (`std/string`) pipelines on both the evaluator and `--strict-bytecode` while preserving curried `0`-placeholder stages (`5 |> makeAdder(1)`) and rejecting multi-placeholder stages (`x |> f(_, _)`) with `TC_PIPE_001`.
- **Newline-separated `?name` / `?` typed holes in `{ ... }` blocks**: Added `lexer.QUESTION` to `peekIsLiteralStatementStart()` in `internal/parser/parser_func.go` so `{ let y = x * 2 \n ?finish }` parses `?finish` as `*ast.Hole` while keeping same-line `let y = f(x)?` propagation intact.
- **`IMP010` cross-module `ailang check --fix`**: Added `internal/quickfix/imp010_move.go` to move unambiguous single-exporter mis-imported symbols (e.g. `import std/list (map, join)` -> `import std/list (map)` + `import std/string (join)`) automatically.

### Fixed — `STRICT_FALLBACK_001` List Base-Case Exemption, Block `let` Hints & Warning File Spans

- **`STRICT_FALLBACK_001` structural list recursion base case**: Exempted `[] => Ok([])` arms in `internal/pipeline/strict_fallbacks.go` when paired with a non-empty list pattern sibling (`x :: rest` / `[x, ...rest]`), while still flagging `Err(_) => Ok([])`, `None => Ok([])`, and `_ => Ok([])`.
- **Block `let` syntax suggestions & warning spans**: Suppressed `let ... in` suggestions on `PAR014`/`PAR015`/`PAR018` inside `{ ... }` blocks in favour of `;`/newline guidance, and wired `WarningFile()` (`warningFiler`) + fallback spans on `ArgOrderWarning` (`warn_split_args.go`) and `TakeAfterFlatMapWarning` (`warn_take_after_flatmap.go`).
