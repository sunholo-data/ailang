### Added — Pipe operator (`|>`) (`M-PIPE-OPERATOR` / `M-AI-PREFERRED-LANGUAGE M5a`)

- Added left-associative pipe operator `x |> f` (lowest operator precedence, sharing band 1 with `\`), desugaring to LHS-first `Let` + `App` in `internal/elaborate` so evaluation and effect order are strictly left-to-right.
- Works across evaluator, `--strict-bytecode` (0 VM fallbacks), and `ailang fmt` (round-trip idempotent, preserving parentheses around inline lambda stages in chains).
