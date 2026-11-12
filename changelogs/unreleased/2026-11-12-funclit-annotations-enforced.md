### Changed — BREAKING: function-literal annotations are enforced (#2238) (2026-11-12)

The parameter and return annotations of an inline function literal were dropped
during elaboration, so `let f = func(x: int) -> int { 0 }; f("str")` type-checked
and every engine ran it, and `func(x: int) -> int { "s" }` returned a string
typed as an int. An explicit `! {}` on a literal was dropped too, so
`func(x: int) -> int ! {} { println("x"); x }` was accepted. Only top-level `func`
declarations recorded their annotations.

Every function form now hands its annotations to the type checker through one
path (`elaborate.recordSignatureAnnots`): top-level `func`, `func(x: T) -> R { … }`
literals (in a `let`, a `letrec`, a module-level `let`, or passed as an argument),
and `func(x: T) => e` lambdas. An unannotated parameter is still inferred. A
literal's parameters are typed as a top-level func's are (the annotation is the
parameter's type), except a parameter whose annotation contains a callback with
a concrete effect contract (`cb: (int) -> () ! {IO}`): that one is an equality on
a fresh variable, as a `let` annotation is, because the contract's open tail
otherwise leaks into the enclosing function when the callback is passed on to
an effect-polymorphic callee (#2295, which top-level funcs still have). A row
variable a literal declares or takes in its parameters (`func(g: int -> () ! {e})
-> () ! {e} { g(1) }`) is the literal's own effect polymorphism: it is charged
where the literal is called, not to the function around it. An explicit `! {}`
on a literal is a declaration of purity and is enforced, including for a literal
stored in a record, list or tuple. The REPL prompt now passes all of these
annotations (and `let` annotations) to the checker; it passed none.

Newly rejected (all previously unsound), with the migration for each:

- Calling an annotated literal with an argument of another type:
  `cannot unify type constructors: int vs string`. Fix the call or the annotation.
- A literal whose body does not have its declared return type:
  `return type annotation at …`. Fix the body or the annotation.
- A **closed record** annotation given a wider record — the most likely
  real-world hit: `sortBy(func(a: {key: int}, b: {key: int}) -> int { … }, rows)`
  where `rows` have four fields is `record field mismatch: expected 1 fields, got 4`.
  Write the open row the body needs: `func(a: {key: int | r}, b: {key: int | r}) -> int`.
- Reading a field the annotated record does not have (`func(ctx: PureCtx) -> int
  { ctx.ports.emit(…) }` with `PureCtx = {name: string}`):
  `record field 'ports' not found in concrete record`. Annotate the record the body
  actually reads.
- A literal's effect contract violated by its body: `! {IO}` calling an
  `FS`-contract callback, `! {}` calling an `IO` callback or printing, an
  unannotated-effect literal inside a `pure` function that invokes an `IO`
  callback: `lambda at … uses effects not declared in its annotation`. Declare
  the effects the body performs (or a row variable `! {e}`).

Measured: of 11,181 banked AI-written programs that check on dev (14,692
deduped), 4 are newly rejected, all genuinely ill-typed (a closed `{key: int}`
annotation given 4-field records; a function applied where an int is expected;
`jnum` given an int where a float is declared; `Option[Json]` returned where
`Json` is declared). In motoko-fork, 3 DST fixtures that document the
missing-field hole and 21 that document `! {}` effect escapes (its
run_declared_vs_performed.sh group 3, "accepted bare") become compiler-rejected.
