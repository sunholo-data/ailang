# Published Benchmark Results (ai-coding-lang-bench)

Source: https://github.com/mame/ai-coding-lang-bench
Agent: Claude Code (Opus 4.6, high effort mode)
Trials: 20 per language, v1 + v2

## Results Summary

| Language | v1 Time (avg) | v2 Time (avg) | Total Cost | v1 Pass | v2 Pass | LOC (v1) |
|----------|--------------|---------------|------------|---------|---------|----------|
| Ruby | 73.1s | - | $0.36 | 20/20 | 20/20 | ~200 |
| Python | 74.6s | - | $0.38 | 20/20 | 20/20 | ~250 |
| JavaScript | 81.1s | - | $0.39 | 20/20 | 20/20 | ~220 |
| Go | 101.6s | - | $0.50 | 20/20 | 20/20 | ~280 |
| Java | 92.2s | - | $0.48 | 20/20 | 20/20 | ~350 |
| TypeScript | 88.5s | - | $0.45 | 20/20 | 20/20 | ~240 |
| Rust | 113.7s | - | $0.54 | 20/20 | 18/20 | ~320 |
| Perl | 100.2s | - | $0.46 | 20/20 | 20/20 | ~200 |
| OCaml | 120.5s | - | $0.58 | 20/20 | 19/20 | ~280 |
| Haskell | 174.0s | - | $0.74 | 20/20 | 19/20 | ~300 |
| C | 142.3s | - | $0.63 | 20/20 | 18/20 | ~400 |
| Lua | 95.8s | - | $0.44 | 20/20 | 20/20 | ~220 |
| Scheme | 156.2s | - | $0.68 | 19/20 | 17/20 | ~350 |
| Python/mypy | 98.4s | - | $0.48 | 20/20 | 20/20 | ~260 |
| Ruby/Steep | 130.1s | - | $0.57 | 20/20 | 18/20 | ~210 |

## Key Findings

1. **Dynamic languages are faster**: Ruby/Python ~73-75s vs Haskell ~174s
2. **Type checking adds overhead**: Python vs Python/mypy, Ruby vs Ruby/Steep
3. **Cost correlates with time**: More turns = more tokens = more cost
4. **Pass rate high across board**: Most languages 20/20 on v1, slight drops on v2

## AILANG Comparison Points

AILANG is closest to:
- **Haskell** (static types, pure FP, algebraic effects) — time/cost target
- **Elixir** (zero training data, functional) — data scarcity comparison
- **Scheme** (niche, functional, parenthesized) — novelty comparison

AILANG advantages over these:
- Self-documenting via `ailang prompt` (1851-line machine-readable spec)
- Explicit effect system reduces debugging
- Concise syntax (fewer LOC expected)

AILANG disadvantages:
- Zero training data in any model
- Missing some CLI tool primitives (exit codes, mutable state)
- Novel syntax requires discovery overhead
