### Fixed — `ailang verify` checks calls and arithmetic inside helpers, lambdas and recursion

A verified function's callee `requires`, int64 overflow and NaN checks used to
cover only the function's own body. A helper that divided by zero, a helper that
called another function with a bad argument, overflow inside a `map` lambda, and
a recursive call that broke the function's own `requires` all left the result
`✓ VERIFIED`.

They are now checked at each call, under the caller's branch and `let`, through
any depth of helpers:

- a helper (or a helper's helper) that may divide by zero is `✗ VIOLATION`
  (`float_nan_or_inf`); one that may break a callee's `requires` is
  `requires_violation`; its int `+ - *` that may wrap gets the overflow note;
- a lambda passed to `map`/`filter`/`foldl` and a recursive helper are checked
  to the unrolling depth (3), and a verified result says so with an
  `obligations_checked_to_depth` note;
- a recursive contracted function's calls to itself must meet its own
  `requires`. With that checked, the unrolling's obligations follow by
  induction, so no depth note. A model of a broken self-call is inconclusive
  (the reason names it), since replay confirms only `ensures` failures.

The postcondition query no longer assumes the int64 range obligations (they
are checked separately); assuming them made Z3 time out on one docparse
function. Witness replay re-solves with int parameters in the int64 range when
the solver's first model used a value no caller can pass.

Across the eight downstream corpora (`make verify-gap-map`), no verdict
changed.
