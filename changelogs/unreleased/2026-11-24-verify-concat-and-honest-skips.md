### Changed — `ailang verify`: std/list.concat encoded; two errors become honest skips

- `std/list.concat(xs, ys)` is encoded as `(seq.++ xs ys)` for any element type. Before, every
  caller was skipped: the callee's signature is generic, and its bare name collided with
  `std/bytes.concat`. An empty-list operand of `++`/`concat` now takes the other operand's sort;
  `concat([], xs)` over strings was a Z3 sort error.
- A record whose declaration could not be encoded (e.g. a field of type `[(int, int)]`) is now
  `skipped` with a reason, not an `error` (docparse `emptyParseState`).
- Skip reasons name tuple and function types in AILANG syntax (`list[(int, string)]`), not Go
  type names (`list[*ast.TupleType]`).

Gap map against the M0 baseline: error 1 → 0, no other verdict changed. The callers `concat`
blocked are large parsers with further blockers (`foldChars`, `bytes`/`Option` helpers, effects).
