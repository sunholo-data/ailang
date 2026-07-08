# EVAL CANDIDATE — `dense_operator_program` persistent failure analysis

**Type**: Eval-driven candidate analysis (precursor to formal design doc)
**Source**: M-EVAL-OS-LONGITUDINAL feedback loop (manual run, pre-Phase-4 automation)
**Date**: 2026-05-23
**Status**: Diagnosed — language gap confirmed

## Summary

`dense_operator_program` failed in **all 3 trials** (gemma4:26b, opencode, smoke tier) with `logic_error` and ~37K tokens each time. The consistency across trials (37K, 36K, 37K) made it the cleanest candidate from the rotation. Manual deep-dive revealed the root cause is **a real AILANG language gap**, not a model capability issue: the benchmark requires bitwise operators (`<<`, `>>`, `&`, `|`, `^`) which AILANG does not have.

## Evidence

### Failure pattern (3 trials, gemma4:26b)

| Run | Setting | Wall | Tokens | Category | Code written |
|---|---|---|---|---|---|
| p=4 #1 (overnight) | 1h 39m batch | 16m32s | 36,888 | logic_error | placeholder only |
| p=2 #2 (overnight) | 1h 49m batch | 12m45s | 37,465 | logic_error | placeholder only |
| (earlier smoke tier) | bumped timeouts | 23m50s | 148,251 | compile_error | broken AILANG |

**`agent_turns`**: 2 in both overnight runs. **`files_modified`**: null. The model attempted in 2 turns, gave up, never wrote a solution to disk. The earlier smoke-tier run blew more tokens because it actually wrote AILANG code (with invented operators) and the compile-error loop drove iteration.

### Benchmark requirements

From `benchmarks/dense_operator_program.yml`:

```yaml
task_prompt: |
  Write a program in <LANG> that computes and prints the following values, one per line:

  1. (5 << 3) + (16 >> 2)     -- left-shift 5 by 3, right-shift 16 by 2, then sum
  2. (12 & 10) | (4 ^ 6)      -- bitwise AND, OR, XOR composition
  3. The number of integers in [1..20] where (n != 0) && ((n % 3 == 0) || (n % 5 == 0))
  4. ((100 == 100) && (50 < 75)) -- print "true" or "false"

expected_stdout: |
  44
  10
  9
  true
```

Lines 1 and 2 use **bitwise operators**: `<<`, `>>`, `&` (AND), `|` (OR), `^` (XOR).

### AILANG operator support (lexer scan)

| Operator | Token | Status |
|---|---|---|
| `<<` | not lexed as binop | ❌ Missing |
| `>>` | not lexed as binop | ❌ Missing |
| `&` | AMPERSAND (used for record/row syntax, NOT bitwise) | ❌ Missing |
| `\|` (single) | not lexed as bitwise OR | ❌ Missing |
| `^` | XOR (need to confirm — present at lexer.go:242 but semantics unclear) | ❌ Likely missing |
| `&&` | AND | ✅ Logical AND |
| `\|\|` | OR | ✅ Logical OR |
| `==`, `!=`, `<`, `>` | comparisons | ✅ |
| `%` (modulo) | already supported | ✅ |

### Stdlib search

```bash
grep -rE "bitand|bitor|bitxor|shl\b|shr\b" std/ | wc -l   # → 0 matches
```

No bitwise functions provided by stdlib either.

## Root cause

**The benchmark cannot be solved in AILANG today.** Lines 3 and 4 of the task use only supported operators (`%`, `==`, `!=`, `<`, `&&`, `||`) and are solvable. Lines 1 and 2 require operators that don't exist in the language at all.

This is exactly the kind of finding M-EVAL-OS-LONGITUDINAL's feedback loop is designed to surface. **The benchmark itself is now diagnostic of the language**, not of model capability.

## Recommended responses (need decision)

Three options, in increasing scope:

### Option A: Mark the benchmark Python-only

Edit `benchmarks/dense_operator_program.yml`:

```diff
- languages: ["python", "ailang", "moonbit", "aver"]
+ languages: ["python"]   # AILANG lacks bitwise ops — see candidate analysis
```

**Cost**: ~1 line of YAML. **Tradeoff**: removes a benchmark from the AILANG rotation; doesn't fix the underlying gap.

### Option B: Add bitwise operators to AILANG (full language change)

A real design doc — call it `M-BITWISE-OPS` — would specify:

- Token additions: `LSHIFT`, `RSHIFT`, `BITAND`, `BITOR`, `BITXOR`, possibly `BITNOT`
- Operator precedence: where in the precedence ladder
- Type rules: `Int → Int → Int` for binary; `Int → Int` for unary
- Stdlib bridge functions for AI agents that don't know the syntax: `bitShl`, `bitShr`, `bitAnd`, `bitOr`, `bitXor`
- AST + parser + types + evaluator + teaching prompt updates
- Examples in `examples/runnable/bitwise.ail`

**Cost**: ~300 LOC across parser/types/eval + examples + prompt. **Tradeoff**: adds language complexity; precedence ambiguity (does `a & b == c` parse as `a & (b == c)` or `(a & b) == c`?); but the language gains a capability that's table-stakes for systems programming benchmarks.

The NERD-hypothesis framing in the benchmark description suggests this was always intended as a *test of whether AILANG had the operators*. Adding them turns a 0% pass benchmark into a measurable one.

### Option C: Add stdlib functions only (no syntax change)

Halfway: add `std/bits` with `shl`, `shr`, `band`, `bor`, `bxor`, `bnot` as functions. Model would write:

```ailang
let r = band(12, 10) `bor` bxor(4, 6)
```

instead of `(12 & 10) | (4 ^ 6)`.

**Cost**: ~80 LOC stdlib + tests + teaching-prompt update + benchmark-prompt update (must say "use std/bits functions"). **Tradeoff**: functional but ergonomically worse than native operators; benchmark wording would need updating.

## Recommendation

**Option A immediately** (1-line YAML change unblocks the benchmark from being a false positive in the rotation) + **Option B as a follow-up design doc** when language-design bandwidth allows.

The benchmark's stated purpose was to test the NERD-hypothesis that "ambiguous operators hurt LLM tokenization." If AILANG adds the operators (Option B), the benchmark becomes a real refutation test. If we mark it Python-only (Option A only), we lose that experimental signal.

## Follow-up actions if this becomes a formal design doc

1. Confirm Option A is acceptable as a stopgap (no impact on rotation pass rate calculation — the benchmark is filtered out)
2. Decide whether `M-BITWISE-OPS` makes the v0.23.0 cut or moves to v0.24.0
3. Open question: does `^` (lexer.go:242) get repurposed? Currently it has some token assignment — need to check whether it's already in use as a different operator (probably function composition or list-cons)

## Related

- [M-EVAL-OS-LONGITUDINAL design doc](./m-eval-os-longitudinal.md) — the feedback loop this candidate originated from
- `benchmarks/dense_operator_program.yml` — the benchmark spec
- `internal/lexer/lexer.go:242` — `^` token handling (needs investigation for Option B/C conflict surface)
- The NERD-hypothesis framing in the benchmark description hints at what we're testing
