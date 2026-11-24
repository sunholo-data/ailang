### Fixed — `ailang verify` verifies through self-recursive helpers

A contracted function calling a self-recursive helper with no `ensures` (an index scanner such as
`allDigitsFrom(s, i)`) was skipped with a misleading "SMT declaration closure could not resolve a
required sort": recursive callees were refused outright, although the contracts guide said they
were unrolled. They are now unrolled to depth 3 below an uninterpreted stub, like a `map`/`filter`
specialisation:

- a proof through the helper holds for every value the stub could return (labelled `bounded: depth 3`);
- a counterexample through it is `inconclusive` until witness replay confirms it (`countUp(2, 0)`
  breaking `result <= 1` is replayed to a VIOLATION);
- a solver `unknown`/timeout through it is a skip, as before, not a new failure;
- a recursive helper *with* an `ensures` is still used through its contract (unbounded), and mutual
  recursion is still skipped.

The inconclusive reason names the helper's unrolling instead of a frontier of the caller.

Gap map: skipped → verified 9, skipped → inconclusive 4 (docparse template scanners; replay could
not confirm a violation), no regressions.
