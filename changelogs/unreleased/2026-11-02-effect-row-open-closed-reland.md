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
