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
- a generic definition that compares or adds a container, such as
  `pure func less[a](x: Option[a], y: Option[a]) -> bool = x < y` or `xs + xs` on `[a]`. These
  used to check and then failed at every call site (`Ord[Option[int]]`), or at lowering.

Compile-cache key `v13 -> v14`.
