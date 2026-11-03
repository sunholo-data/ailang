### Changed — Breaking (soundness): a lambda's effects no longer depend on how many calls its body makes (#1958, 2026-11-02)

Effect rows of sibling sub-expressions (sequences, `let`, branches, lists, binops, an application's
arguments) were combined while their operands could still be unresolved row variables. Two different
unresolved variables made the combined row close to "no effects", so a lambda with two effectful
calls published a pure type, while the same lambda with one call published `! {IO}`. Inference now
resolves those operands against the enclosing lambda's constraints before combining them. A recursive
call's own row contributes only its labels (fixpoint absorption).

Programs that now fail to check (all were accepted before):

- An IO lambda passed to a **pure** callback slot (`std/list.map`, `any`, `filter`, `foldl`, …) when
  its body makes more than one call, e.g. `map(\x. { println(show(x)); x }, xs)`. One-call lambdas
  and named functions were already rejected. Migrate to `mapE`/`filterE`/`foldlE`/`forEachE`
  (there is no `anyE`; `foldlE` covers it).
- A recursive function whose declared row omits the row of a callback it calls, e.g.
  `func bad[e](f: (int) -> () ! {e}, n: int) -> () = … { f(n); bad(f, n - 1) }`. The non-recursive
  form was already rejected. Declare the row: `-> () ! {e}`.

Design: `design_docs/implemented/v0_56_0/m-effect-row-open-closed-soundness.md` (re-land brief).

### Fixed — Breaking (soundness): an annotated effectful callback can no longer pass through a pure slot (#1958 M2 re-landed, 2026-11-02)

Row unification bound an open row's tail against a closed row without checking the open row's own
labels, so `{IO | ε} ~ {}` succeeded and `IO` disappeared. A `pure func` could then run IO through
`std/list.map`/`filter`/`foldl`/`any` with an annotated callback parameter (`g: (string) -> bool ! {IO}`),
a record field, or a mixed row `! {IO, e}`, and both engines executed it. The guard (Rémy's side
condition) is restored in both open/closed branches of `RowUnifier.UnifyRows`. The error names the
labels: `row {IO, ...ε} carries labels [IO] which the closed row {} does not permit`. Migrate to
`mapE` with a row-polymorphic parameter: `func viaE[e](g: (int) -> int ! {e}, xs: [int]) -> [int] ! {e} = mapE(g, xs)`.

v0.56.0 carried this guard and was withdrawn before publishing: it rejected valid callbacks. Two
inference fixes now ship with it: the union fix above, and **lambda open-on-use**. An unannotated
inline lambda whose row is (or resolves to) a closed row gets a fresh row tail, so it fits a slot
that permits more: `withDir(\d. 1)` fits `f: string -> int ! {IO}`, and `w(\c. println("x"))` and
the eta form `runWide(\s. loud(s), "x")` fit `! {IO, FS}`. An unconstrained tail is closed to `{}`
before the declaration is published.

Newly rejected, and valid (deferred to Phase 2b): a function value *read from a binding* whose row is
closed and narrower than the slot, e.g. `runWide(loud, "x")` with `loud ! {IO}` passed for
`! {IO, FS}`, or `let f = \s. println(s); runWide(f, "x")`. v0.56.1 accepted these only because the
unguarded unifier dropped the slot's extra labels. Pass the lambda inline (or eta-expand:
`\s. loud(s)`), declare the value with the slot's row, or make the slot row-polymorphic. None of the
4,599 `.ail` files swept for this release (this repo, the package, parse, docparse, demos, game,
Daneel and email-parse repos) hits this case.

Migration needed before bumping a pinned consumer: every program this release newly rejects in that
sweep passes an effectful lambda to a pure `std/list` `map`/`filter`/`foldl`/`any` slot (21 sites).
Each one moves to `mapE`/`filterE`/`foldlE`.
