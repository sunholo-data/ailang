### Fixed — rigid signature variables: follow-ups from the #2273 review

- A record row the author names `_r1` (or any `_r` + digits) is a named row, rigid like `| r` or
  `| _rest`. Before, it was taken for the parser's anonymous `...` row by its name, so
  `f(p: {x: int | _r1}) -> {x: int | _r1} = {x: 1}` was accepted with a closed scheme. Anonymity is
  now a mark the parser sets on the row it creates for `...` (`ast.TypeVar.Anonymous`).
- `TC_RIGID_001` names the culprit when a deferred field access (#2284) is involved. A reader
  applied to the rigid value, or a rigid row read through a let-bound copy, is "(field access)" at
  the access. A type an access passes on is blamed where it is pinned, for example
  "(return type annotation)". Before, the message had no culprit.
- A `requires`/`ensures` clause that makes two signature variables one type now suggests keeping
  them apart "in the clause", not "in the body".

### Changed — tests

- The generic-body matrix (`cmd/ailang/generic_matrix_test.go`) now checks five forms for every
  operator group: a rigid function, an unannotated one, a let-bound lambda and an annotated function
  literal inside a rigid function, and a let-bound lambda inside a monomorphic function. Each runs
  against its monomorphic twin on the evaluator and the strict VM. A `show` inside an inner lambda
  printed at the wrong type is now caught.
- A `%` between an int and a float (`h(x) = x % 2; h(7.5)`) is pinned to fail loudly on both
  engines. `mod_Int` still turns two floats into float modulo on every call, because restricting
  that to generic bodies would change the frozen `$builtin` interface.
- The codegen harness pins #2264's current symptom exactly: the generated Go returns `<function>`
  where the evaluator prints 6. It used to panic.
