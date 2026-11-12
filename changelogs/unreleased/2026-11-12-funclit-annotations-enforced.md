### Changed — BREAKING: function-literal annotations are enforced (#2238) (2026-11-12)

The parameter and return annotations of an inline function literal were dropped
during elaboration, so `let f = func(x: int) -> int { 0 }; f("str")` type-checked
and every engine ran it, and `func(x: int) -> int { "s" }` returned a string
typed as an int. Only top-level `func` declarations recorded their annotations.

Every function form now hands its annotations to the type checker through one
path (`elaborate.recordSignatureAnnots`): top-level `func`, `func(x: T) -> R { … }`
literals (in a `let`, a `letrec`, a module-level `let`, or passed as an argument),
and `func(x: T) => e` lambdas. An unannotated parameter is still inferred. A
literal's parameter annotation is enforced as an equality on the parameter's
variable — the way a `let` annotation is — rather than by seeding the
parameter's type as a top-level `func` does: seeding made a callback parameter
annotated `! {IO}` and passed on to an effect-polymorphic callee leak its open
effect tail into the enclosing function (a pre-existing effect-row defect that
top-level `func`s share), which rejected correct programs.

Newly rejected (all previously unsound): calling an annotated literal with an
argument of another type, and a literal whose body does not have its declared
return type. The diagnostics are the existing unification errors
(`... failed to unify parameter 0: cannot unify type constructors: int vs string`,
`... return type annotation at ...`). Migration: fix the call or the annotation.
