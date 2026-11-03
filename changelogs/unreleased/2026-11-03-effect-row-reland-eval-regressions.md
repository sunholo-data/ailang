### Fixed — valid programs rejected by the #1958 re-land, found by the pre-release sweep of generated code (2026-11-03)

The v0.57.0 pre-release sweep checked 15,447 unique eval-generated AILANG programs against v0.56.1.
16 valid programs that v0.56.1 accepts were rejected by the effect-row re-land. There were two
causes:

- **An opened lambda tail was generalized into a type scheme.** A function returning an
  unannotated lambda (`pure func pipe(f: int -> int, g: int -> int) -> int -> int = \x. g(f(x))`)
  unified the lambda's opened row tail with its return annotation's tail. `let`/`letrec`
  generalization then quantified that tail before it was closed, so every use site instantiated a
  tail that no scope owns. Depending on the program, this surfaced as `internal invariant: missing
  or malformed ApplicationEffects`, as `distinct unresolved effect tails`, or as a spurious
  `Missing effects`. An opened tail that is still unresolved is now closed to `{}` before
  generalization, which is the value it was given later anyway.
- **A recursive call's own row was missed from the third call on.** The row unifier's substitution
  is not idempotent (ρ1 ↦ {...ρ3}, ρ3 ↦ {...ρ6}), and the union resolver followed only one link.
  A local IO `letrec` whose recursive calls sit three or more branches deep then failed with
  `same row variable with different extensions`. #2074 fixed the shallow case; the resolver now
  follows the whole chain.

After the fixes, the newly rejected programs in that sweep are exactly the intended ones: effectful
lambdas passed to a pure `std/list` callback slot.
