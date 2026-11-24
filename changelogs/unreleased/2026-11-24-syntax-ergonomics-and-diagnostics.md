### Added — Syntax Ergonomics (V1 & V2), `std/list` helpers, and Agent Diagnostics

- **Destructuring in `let` and lambdas** (`M-SYNTAX-ERGONOMICS-V1` / `V2`):
  - Irrefutable tuple and record destructuring in `let` bindings (`let (a, b) = pair`, `let {name, age} = person`, `let {port: p, ...} = cfg`, block and `in` forms).
  - Irrefutable tuple and record destructuring in lambda parameters (`\(k, v). ...`, `\acc (k, v). acc + v`, `\{age, ...}. age`), preserved verbatim by `ailang fmt` and compiled with 0 VM fallbacks under `--strict-bytecode`.
- **Module-qualified type names & block ergonomics**:
  - Qualified type annotations (`S.Set[int]`, `M.Map[string, int]`, `J.Json`) resolve through module import aliases.
  - Newline-leading `(`, `()`, `{`, and `[` inside `{ }` blocks start new statements instead of being parsed as calls/indexing on the preceding expression.
  - Optional trailing `;` on top-level `let` bindings (`let MOD = 1000000007;`).
  - Contextual contract/test keywords (`requires`, `ensures`, `invariant`, `property`, `test`) are accepted in unambiguous record field positions (`{ requires: [Req] }`, `pkg.requires`).
- **Stdlib & effects**:
  - Added iterative $O(n)$ `takeWhile` and `dropWhile` to `std/list`.
  - Added one-way `AI[mode=routeable]` → `AI[mode=fixed]` effect subsumption so routeable callers can invoke fixed-mode helpers.
- **High-signal parser & import diagnostics** (`M-AGENT-DIAGNOSTICS-ERGONOMICS`):
  - Added dedicated diagnostics `PAR_IF_BRANCH_BLOCK` (unbraced multi-statement `then` branch), `PAR_LOCAL_FUNC` (nested named `func` in block), `PAR_HASH_COMMENT` (`#` comments vs `#[...]` arrays), `PAR_PIPE_OPERATOR` (`|>` instead of misleading bitwise-OR `PAR016`), and `PAR_MATCH_ARM_BLOCK`.
  - Added camelCase/snake_case word-overlap fallback to `IMP010` suggestions (e.g. `nthChar` → `charAt`, `parseInt` → `stringToInt`).
- **Fixes**:
  - `ailang eval-suite` now prefers `os.Executable()` over a stale `ailang` binary on `PATH`.
  - `ailang test` now binds top-level value `let` declarations in its tree-walking evaluator harness (matching `ailang run` and `--strict-bytecode`).
  - Fixed `ailang fmt` crash on comments inside single-expression braced function bodies.
