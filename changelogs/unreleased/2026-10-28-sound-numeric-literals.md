### Fixed — an integer literal used at type float type-checked, then crashed or came out as an int (2026-10-28)

An integer literal is `Num a => a`. When inference resolved it at `float` (`f >= 0`, `f + 1`,
`2 * f`, `func z() -> float { 0 }`, `(1, f)`, `[1, 2.5]`, `make(3, 0)` for an `Array[float]`),
`ailang check` passed but the runtime still built an int. Both engines then failed with
`expected float arguments`, `missing dictionary method: prelude::Fractional::Int::add`,
`vm: arith ADD: type mismatch Float vs Int`, or `scatterAdd: arg 0 element 0 must be float`.
Where nothing checked the value, the program returned the wrong value without an error:
`-> float { 0 }` printed `0`, and `(1, 2)` at `(float, float)` printed `(1, 2)`. Dictionary
elaboration, where the type checker and the runtime meet, now rewrites an integer literal
resolved at Float into a float literal (`internal/elaborate/dictionaries.go`), so both engines
see `0.0`. Tuples, arrays and record updates were never traversed by that pass, so they now are.

### Fixed — a local `let` holding an integer literal was generalized over a type the runtime cannot vary (2026-10-28)

`let w = 70; w / (h * h)` and `let inc = \x. x + 1; inc(1.5)` type-checked as polymorphic
bindings, but the runtime has no dictionary passing for let-polymorphism. The literal stayed an int
at every instantiation, and the float use crashed. A type variable that carries an integer literal's
`Num` constraint is no longer generalized at a let. This is Haskell's monomorphism restriction,
limited to integer literals (`internal/types/literal_restriction.go`). The binding's uses decide
its type, so both programs above now run. Using such a binding at both `int` and `float` is now a
type error at `ailang check`; before, the float use crashed at runtime. Literal-free
let-polymorphism is unchanged: `let add = \x. \y. x + y` still works at both types (M-POLY-ARITH).
A module declaration's own binding (a top-level `func` or `let`) keeps full generalization.

### Changed — an int and a float reaching one numeric operator now fail with a named cause (2026-10-28)

What the restriction does not reach is an integer literal inside a generic top-level function or a
module-level `let` that is used at float, for example `func incG[a](x: a) -> a { x + 1 }` with
`incG(2.5)`. It still fails at runtime. The evaluator now reports `numeric dictionary method "add"
received an int and a float: an integer literal inside a generic function or a module-level binding
was used at type float ...`, with the fix to apply, in place of `expected float arguments`.
Tracking: the integer-literal restriction for module declarations needs per-declaration defaulting
of the exported scheme.
