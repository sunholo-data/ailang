# Sprint plan: M-SMT-WELL-SORTED-PREFLIGHT M0 (soundness first)

**Design doc**: [m-smt-well-sorted-preflight.md](m-smt-well-sorted-preflight.md), milestone **M0 only**
**Issues**: #1960 (P0), #1961 (P0)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-SMT-PREFLIGHT-M0.json`
**Estimate**: 1 day, about 350 LOC (150 implementation, 200 tests and fixtures)
**Risk**: medium. Some existing VERIFIED results may stop verifying. That is intended: each one was resting on an
unsound encoding, and every flip is listed in the PR.
**Ruling**: Mark, 2026-10-26: fix the P0s now. Truncating Int `/` and `%` is the required sound choice (F1). The freeze
items for M1–M4 (`encoder_gap`, ai-check exit codes, pre-flight shape) are **not** ruled and are out of scope.

## Status: complete (2026-10-26)

- [x] M0a: truncating Int `/` and `%` (981a7aee9)
- [x] M0b: per-call-site contract callee results (5410d0b2f)
- [x] Mutation: each fix reverted from a scratch copy fails its tests
- [x] Flip audit: examples/ + std/ have 0 flips; ailang-packages has 1 ERROR→VERIFIED (`discord.validEmbed`, which
  had a callee parameter left free) and 0 VERIFIED→not
- [x] Gates: `make test-changed`, `make lint`, `make verify-examples`, `make check-changelog`,
  `make check-file-sizes`

## Scope

| In | Out (later milestones or other docs) |
|----|--------------------------------------|
| S1: truncating Int `div`/`mod` on every emitter (`div_Int`/`mod_Int`, `OpDiv`/`OpMod`, BinOp `/`/`%`, DictApp `div`) | Type-directed operator resolution (M1), including `mod_Float` and Float `/` via BinOp |
| S2: contract-as-spec callees become one uninterpreted function each, constrained per call | Discharging the callee's `requires` as a call-site obligation (`verify-call-site-precondition-not-checked.md`) |
| Model parser ignores non-nullary `define-fun`s, so a UF does not print as a junk counterexample line | Pre-flight sort checker and `encoder_gap` (M3) |

## M0a: truncating Int division and remainder (#1960)

- One helper renders `a / b` and `a % b` with Go semantics:
  `div` = `ite(sign(a) == sign(b), div(|a|,|b|), -div(|a|,|b|))`; `mod` = `a - b * div`.
  The operands are bound with a `let`, so each operand is encoded once and never captured.
- Every Int div/mod emitter routes through it. Division by zero stays uninterpreted. The runtime raises RT001 there, so
  any value is sound for partial correctness.
- Tests (TDD, red first): encoding unit tests for each emitter; Z3-backed tests that `parity` and `halfDown` give a
  counterexample with a negative `x`; true controls (`halfNonNeg`, parity bounded in `(-2, 2)`) still VERIFY.

## M0b: per-call-site contract callee results (#1961)

- Replace the shared `$xmod_result_<f>_0` constant with `(declare-fun <f>__spec (<param sorts>) <ret>)` and one
  axiom, `forall params. requires(params) => ensures(params, result := <f>__spec(params))`, with the call term as the
  trigger. Z3 therefore instantiates the callee contract once per distinct call term, with that call's arguments.
  `f(a) - f(a)` still proves 0, which is the design's F4 choice.
- `requires` **guards** the ensures. It is not asserted as an axiom, which would be unsound, and it is not an
  obligation, which belongs to the other doc. If a `requires` cannot be encoded, the `ensures` are dropped and the
  result is left unconstrained. That is weaker, and sound.
- Callee parameters are bound by the quantifier, so no callee parameter name is left free. This fixes bucket B3.
- Tests: `gap` gives a counterexample with `a >= 0`; `atLeast` (an ensures that mentions the callee parameter)
  VERIFIES; `f(a) - f(a) == 0` VERIFIES; a caller that may violate the callee's `requires` no longer inherits its
  `ensures`.

## Validation

- Mutation: revert each fix from a scratch copy and confirm its tests fail.
- Flip audit: run `ailang verify` across `examples/` and `std/`, and read-only across `ailang-packages`, before and
  after. Explain every VERIFIED→not-VERIFIED flip.
- Gates: `make test-changed`, `make lint`, `make verify-examples`, `make check-changelog`, `make check-file-sizes`.
  Never the full `make test`.
- Changelog fragment `changelogs/unreleased/2026-10-26-smt-truncating-divmod-per-call-callee.md` (### Fixed).

## AILANG syntax gate

The plan writes only `.ail` **test fixtures** (contracts are their whole point), not showcase modules. Each fixture is
type-checked with `ailang check` and verified with `ailang verify`. Contracts: include. Effects: skip, because the
fixtures are pure. Inline tests: skip, because these are verifier fixtures pinned by Go e2e tests.

## Registry reuse

M0a: none (internal verifier encoding). M0b: none (internal verifier encoding).
