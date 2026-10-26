### Fixed — soundness: two false-VERIFIED paths in `ailang verify` (#1960, #1961) (2026-10-26)

`ailang verify` proved two kinds of contract that are false at runtime. False VERIFIED is the worst verifier
outcome, because an agent stops looking for the bug. This is milestone M0 of
`design_docs/planned/m-smt-well-sorted-preflight.md`.

- **Int `/` and `%` now truncate toward zero, matching the runtime (#1960).** They were encoded as SMT-LIB's Euclidean
  `div`/`mod`, so `x % 2` with `ensures { result >= 0 }` VERIFIED although `parity(-7) == -1`. All Int division and
  remainder emitters now share one encoding: `div_Int`/`mod_Int` (direct and curried), `OpDiv`/`OpMod`, BinOp `/`/`%`
  and DictApp `div`. Contracts that hold only under truncation, such as `x / 2 * 2 >= x` for negative `x`, now
  VERIFY where they used to give a counterexample. Division by zero stays unconstrained, because the runtime raises
  `RT001`.
- **Each call to a contract-only callee now has its own result (#1961).** A callee seen only through its contract
  (recursive, for example) used to be one shared constant for every call. So `total(a) - total(a + 1)` "proved" equal
  to 0. Such a callee is now one uninterpreted function. Its `requires => ensures` axiom is instantiated per call term,
  at that call's arguments. `total(a) - total(a)` still proves 0. The callee's `requires` now guards its `ensures`
  instead of being ignored, and a callee `ensures` that names its own parameter is bound correctly. It used to fail as
  `unknown constant n`. Turning `requires` into a call-site obligation is still open
  (`ailang-core-triage/verify-call-site-precondition-not-checked.md`).
- Counterexample output no longer prints an uninterpreted function's interpretation as a value.

Measured against the code before this change:

| Corpus | VERIFIED→not | Other flips |
|--------|--------------|-------------|
| `examples/` + `std/` (515 files) | 0 | 0 |
| `ailang-packages` (45 packages) | 0 | 1 ERROR→VERIFIED |

The ERROR→VERIFIED flip is `discord.validEmbed`. Its callee `embedBudget` has an `ensures` that names parameter `e`,
which used to be left free.
