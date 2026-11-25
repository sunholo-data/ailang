# M-VERIFY-NUMERIC-MODEL — int64 overflow and float NaN/∞ (program M4)

**Status**: Implemented, shipped in v0.61.0 (see §Result). **Program**: [M-VERIFY-SOUNDNESS-COVERAGE](../../planned/v0_61_0/m-verify-soundness-coverage-program.md) M4 (gaps S2, S3).
**Rulings** (Mark, 2026-11-24): D2 = **(a′)**, D4 = **(b′)**, on the measurements recorded in the
program doc (§D2/D4 measurements).
**Quorum**: no trigger fires. D2/D4 are ruled; the change is local to `internal/smt` plus one CLI
flag; no cost/KPI/banked schema; premises in-repo.

## Problem

`int` is proved as an unbounded integer and `float` as a real number. `inc(x) requires { x >= 0 }
ensures { result > x } { x + 1 }` is VERIFIED, yet `inc(9223372036854775807)` wraps to the
minimum. `x / y` with `y` possibly 0 is VERIFIED, yet at runtime it is ±Inf or NaN.

## Design

Both checks are obligations on the M3 channel (`codegen_obligations.go`): path-scoped through call
facts, emitted only for the verified function's own body, and checked by one extra query each
after the postcondition verified. They use two reserved obligation names, `%int64` and `%nan`.

**Int (D2 a′).** Every int `+`, `-`, `*` and unary `-` in the body emits
`(=> (and (in64 l) (in64 r)) (in64 (op l r)))`, with `in64(e) = -2^63 <= e <= 2^63-1`. The
operand guard makes the check per operation and needs no assumption about where the operands
came from (parameters, record fields, list elements, callee results). By induction over the
expression, if every operation keeps in-range operands in range, every value is in range.
- Default: a sat obligation leaves the result VERIFIED and attaches
  `caveats: [{kind: "int_overflow", detail, model}]`. The CLI prints a note. The exit code is unchanged.
- `--int-model=int64` (on `ailang verify`, with or without `--package`; `ai-check` and
  `check --verify` use the default and carry the caveats in their JSON): the result is a `counterexample`
  with reason code `int_overflow`.
- An undecided overflow query never changes the verdict by default (caveat
  `int_overflow_undecided`). Under `--int-model=int64` it is `unknown`.

**Float (D4 b′).** Inputs are assumed non-NaN and finite. The operations that turn such inputs into
NaN or ±Inf are checked: real `/` and `%` need a non-zero divisor, `sqrt` a non-negative argument,
`log` a positive one. A sat obligation is a `counterexample` with reason code `float_nan_or_inf`.
Every VERIFIED result whose encoding uses `Real` carries caveat `float_inputs_assumed_non_nan`.
Overflow of a float product to ±Inf is not modelled (the encoding is the reals) and stays disclosed.

**Evidence.** `numeric_model` gains `int_overflow` (`reported` or `checked`) and `float_nan`
(`operations_checked`). The `Scope:` line says so.

**Scope** (same as M3): arithmetic inside `requires`/`ensures` is not checked. Arithmetic inside
helpers, HOF lambdas and recursion is checked since the follow-up (obligations inside definitions,
see the program doc); in recursive helpers and lambdas only to the unrolling depth. Int `/` and `%` by zero are runtime errors, not values, so they are
outside the partial-correctness model and not checked. `pow` is not checked.

## Verification log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Int arithmetic reaches the encoder as `DictApp` add/sub/mul/neg on `Num Int`, or as `$builtin` add_Int/sub_Int/mul_Int/neg_Int | `encodeDictApp` sortAgnostic branch; `BuiltinToSMTOp` (`types.go:263-297`) |
| V2 | Every real division and remainder goes through `applyBinaryOp` | `encodeBuiltinOp`, `encodeBinaryArgs` both end in `applyBinaryOp`; `divModOp` returns `/` and `smtRealModTrunc` for Float |
| V3 | sqrt/log are encoded in one place | `encodeMathCall` (`codegen_math.go`) |
| V4 | The reason codes are unused | `git grep -n "int_overflow\|float_nan_or_inf"` outside design docs: none |
| V5 | Blast radius, default mode: no verdict changes. Under `--int-model=int64`, 64/590 changed in the prototype | program doc §D2/D4 measurements |

## Acceptance

- `inc` VERIFIED with an `int_overflow` caveat by default, and VIOLATION (`int_overflow`) under `--int-model=int64`;
  a bounded `inc` (`x < 1000`) has no caveat.
- A record-field or list-element operand does not produce a caveat by itself (operand guard).
- `ratio(a, b) requires { b >= 0.0 }` is VIOLATION `float_nan_or_inf`; with `b > 0.0` it verifies.
- `sqrt(x)` under `x >= 0.0` verifies; without it is a violation.
- Mutation: drop each emission; its test fails.
- Gap map, default mode: no verdict regressions.

## Result (2026-11-24)

- `codegen_numeric_obligations.go` (emission), `verify_obligations.go` (caveats, verdicts),
  `--int-model` on `ailang verify`; `numeric_model` evidence fields.
- Refinement found while testing: an obligation's sat model is downgraded to inconclusive only when
  a fact carrying *that* obligation mentions an uninterpreted builtin
  (`CallObligation.DependsOnUninterpreted`). Before it, `sqrt(x)` with `x` possibly negative read
  inconclusive although the failing clause is `x >= 0`.
- `numeric_model_test.go` pins both modes. Mutation-tested: dropping the int emission, the operand
  guard, the division check, the sqrt check, or the uninterpreted-scoping each fails the test.
- Gap map, default mode: no verdict changed; 56 `int_overflow`, 5 `int_overflow_undecided`, 12
  `float_inputs_assumed_non_nan` caveats on verified results. `--int-model=int64`: 61 change
  (56 → counterexample, 4 → unknown, 1 → skipped). The prototype without the operand guard and with
  bounded parameters gave 64.
