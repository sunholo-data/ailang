### Changed — `ailang verify` checks a callee's `requires` at each call (M-VERIFY-CALL-SITE-REQUIRES)

A `✓ VERIFIED` used to mean "ensures holds whenever requires holds" for the function itself, while
calls it made to a function with a `requires` were never checked: `caller(n) requires { n < 0 }`
returning `one(n)` with `one`'s `requires { n > 0 }` verified. Now each call in a verified function's
body must meet its callee's `requires`, under the branch and `let` that reach the call. A call that
may break it is `✗ VIOLATION` with reason code `requires_violation` ("a call to one may break its
requires") and a counterexample. The postcondition is proved assuming callee preconditions hold at
their calls, which those checks then establish (decision D1(a), ruled 2026-11-24).

- A callee whose `requires` cannot be encoded makes its callers `skipped` (the call cannot be
  checked), never verified.
- Not yet checked: calls inside a helper's body (checked when the helper is itself verified), a
  lambda passed to `map`/`filter`/`foldl`, a recursive function's unrolled body, and `requires`/`ensures`.
- `ailang verify --json` results gain `requires_checked` (callees whose calls were checked);
  `make verify-gap-map` reports results that reach a `requires` only where calls are not checked.

Gap map across the eight downstream corpora: no verdict changed. All 11 verified results that call a
function with a `requires` had every such call checked and still verify.
