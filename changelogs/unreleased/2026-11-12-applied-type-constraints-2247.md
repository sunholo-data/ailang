### Changed — BREAKING: a class constraint on an applied type is checked, not dropped (#2247)

`ailang check` accepted an int literal where the parameter was `Box[a]`, `[a]` or `Option[a]`
(`pure func g[a](o: Box[a]) -> int = …` then `g(7)`): the literal's `Num[Box[α]]` constraint was
neither ground (so never looked up) nor a bare type variable (so never deferred to a caller), and the
top-level boundary dropped it. The evaluator then failed with `RT002 … (scrutinee: int)` and the VM
with `GET_TAG on Int`. The same constraint with a ground argument (`Num[Box[int]]`) was always rejected.

Context reduction now covers every class, not only `Eq`: at a declaration boundary a constraint whose
type has a constructor at its head must be solved by an instance, and only a constraint on a type
variable is deferred to the caller. `Eq` keeps its structural decomposition (`None == None`,
`[] == []`, `xs == ys` on `[a]` still check). Newly rejected, with the existing
`No instance for C[T]` diagnostic:

- an int/float literal passed where `Box[a]`, `[a]`, `Option[a]`, `Result[a, e]` is expected (the
  issue's repros, and the same through let-bound, inline and `letrec` lambdas);
- a definition that can never be called successfully, **even if it is never called**: a generic
  function that compares or adds a container, such as
  `pure func less[a](x: Option[a], y: Option[a]) -> bool = x < y`, `xs + xs` on `[a]`, or a lambda
  that passes a literal where a `Box[a]` is expected. These used to check and then failed at every
  call site (`Ord[Option[int]]`), or at lowering; an unused one went unnoticed.
- a higher-kinded parameter (`func k[f, a](x: f[a])`) used with arithmetic or a literal: `f[a]`
  reaches the checker with `f` as a type constructor, and no class but `Eq` has an instance with an
  applied head, so `Num[f[a]]` can never be solved. Higher-kinded heads are not supported.

### Fixed — class constraints see through type aliases (#2247)

A constraint is now judged by the type an alias stands for when that solves it: with
`type Id[a] = int`, `x + 1` on `x: Id[a]` is `Num[int]`. Expansion follows alias chains (including
`Id[Id[int]]`), aliases imported from another module, and aliases inside lists, tuples and type
arguments. Correct programs that were rejected before now check: `type Score = int` with `x + 1` or
`x == y` (`No instance for Num[Score]`), `Eq[Id[a]]`, `Num[Id[string]]`, `Same[int]` with
`type Same[a] = a`. `type Meters[u] = float` with `m * 2.0`, which checked and then crashed at run time
on the Int dictionary, now prints the right result.

An expansion is used only when it solves the constraint outright (a ground constraint an instance
resolves); otherwise the constraint is judged, and reported, as written. Hence:

- **Limitation:** a polymorphic operator on an alias of a type variable is rejected, as before:
  `type Id[a] = a` with `x + x` on `x: Id[b]` is `No instance for Num[Id[b]]`. Write `x: b` instead.
  (Expanding it to `Num[b]` deferred the constraint while the operator was still lowered from the
  alias type, to the Int implementation.)
- An alias whose body names a type variable it does not bind (`type Count = integer`) still reports
  `No instance for Num[Count]`.
- Diagnostics name the type as written (`Num[Tagged[t]]`, `Eq[L[int]]` for a self-referential
  `type L[a] = [L[a]]`).
- Expansion is bounded: an alias is never expanded inside its own expansion, and a node budget stops a
  legitimately huge expansion (a 2^20-leaf nest of pair aliases). Either way the constraint is judged as
  written. A self-referential alias such as `type T2[a] = (T2[a], T2[a])` checks as fast as on dev.
- Record aliases and newtypes are never expanded: their `deriving (Eq)` is keyed by their module, and
  the record path of Eq synthesis is keyed by bare name. The existing record-alias Eq failures are #2318.
