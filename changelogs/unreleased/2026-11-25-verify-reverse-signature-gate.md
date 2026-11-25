### Added — `ailang verify` encodes `std/list.reverse`

`reverse` is an uninterpreted function for each list sort. Each call carries the facts every reversal has: the result has the same length, and its first and last elements swap. A model that rests on its value is inconclusive, and witness replay decides it. Before, every caller was skipped ("unsupported application").

### Changed — a helper with an unencodable signature no longer skips callers it does not reach directly

A helper whose signature has a type variable, a tuple, a function type or `bytes` now skips only the functions that call it **directly**. Reached through another helper, it is never defined, and the helper above it is called opaquely. Before, it skipped every function that reached it at any depth. Opaque helpers are now used in two further cases:
- when a helper's `map`/`filter`/`foldl` cannot be specialised;
- when a helper's signature names an anonymous record.

Downstream gap map: 8 newly verified (among them `decodeBlocks`, `blocksToJson`, `generateQmd`, `texFlushText` and motoko's `compress_output`), 3 move from skipped to inconclusive, no regressions.

### Fixed — a record with a field of no SMT sort was declared without that field (a false VERIFIED)

A record with a tuple-typed field, such as `{ tag: int, pairs: [(int, int)] }`, was declared without that field. Two values differing only there were then modelled as equal, so `requires { r1.tag == r2.tag } ensures { result } { r1 == r2 }` reported VERIFIED. Such a record is no longer declared, and a function needing it is skipped. This applied both to named record types and to inline records inside an ADT. No downstream contract had relied on it.

### Fixed — empty-list sorts

A `[]` that is passed as a `foldl` accumulator, or hoisted out of a record update (`{ st | buf: [] }`), now takes the declared sort. It was `(Seq Int)` or the list's sort, which made Z3 reject the script.
