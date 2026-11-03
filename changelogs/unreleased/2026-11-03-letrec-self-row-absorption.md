### Fixed — A local recursive effectful `letrec` failed to type-check (#2074, 2026-11-03)

Since the #1958 M1 union fix, a local `letrec` lambda that is both recursive and effectful (the
standard read/print loop) failed with `same row variable with different extensions`:

```ailang
letrec output = \k. if k > 3 then () else { println(show(k)); output(k + 1) };
```

Two gaps in fixpoint absorption (a recursive call contributes only its labels to its own row):

- It ran only when **both** operands of a union carried distinct open tails. Once `println(…)`
  resolved to a closed `{IO}`, the block was `{IO} ∪ {...ρ}` with `ρ` the call's own row, so the
  lambda published `{IO | ρ}` and the binding needed `ρ ~ {IO | ρ}`. A single tail is now absorbed
  too, when it is a self row; any other single tail is unchanged.
- A curried binding (`\i. \acc. … loop(i + 1)(acc')`) calls its own **inner** arrow, whose row was not
  recognised as a self row. The check now walks the binding's returned arrows.

Top-level recursive functions, pure local recursion and non-recursive effectful lambdas were never
affected. Soundness is unchanged: the same loop inside a `pure func` is still rejected. Check sweep
over 1,066 `.ail` files (this repo, `ailang-packages`, `ailang-parse`), v0.57.0 vs this fix: 0 status
flips; a positive control of the 7 regressed programs flips all 7.
