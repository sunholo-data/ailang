### Added — `std/list` (`all`, `enumerate`, `filterMap`, `partition`) and `std/result` (`getOrElse`)

- Added true $O(n)$ iterative combinators to `std/list`:
  - `all(p, xs)` — short-circuiting universal predicate (vacuously `true` on `[]`)
  - `enumerate(xs)` — pairs each element with its 0-based index `[(0, x0), ...]`
  - `filterMap(f, xs)` — maps `a -> Option[b]` and keeps `Some` values in order
  - `partition(p, xs)` — splits into `(matching, nonMatching)` evaluating `p` once per element
- Added `getOrElse(r, default)` to `std/result` for parity with `std/option.getOrElse`.
- All 5 combinators run with 0 VM fallbacks under `--strict-bytecode`.
