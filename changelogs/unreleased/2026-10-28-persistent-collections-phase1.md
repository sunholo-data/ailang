### Added — structured Map keys: tuples, records, ADT constructors, unit and float (2026-10-28)

`Map` keys may now be tuples, records, ADT constructors, unit and floats, nested to any depth, in
both the evaluator and the bytecode VM. Before this, such a map type-checked and then failed at
runtime with `unsupported map key type: *eval.TupleValue`, so a visited set over `(row, col, dir)`
could not be written. The key grammar now lives in one leaf package, `internal/pcoll`, which both
engines drive through a `KeyWriter`; a cross-engine golden test keeps their keys byte-identical.
Existing int, string and bool keys are byte-identical to v0.11.0 (`i:%d`, `s:…`, `b:%t`), so every
existing map keeps its iteration order, including lexical int order (`[10, 2, 9]`). Composite keys
sort element by element; floats sort numerically; constructors sort by name. NaN is rejected as a
key, and `-0.0` and `0.0` are the same key because they compare equal.
[m-persistent-collections](design_docs/planned/v0_55_0/m-persistent-collections.md) Phase 1a.

### Fixed — `lookup`, `member` and `remove` silently answered for keys that cannot be keys (2026-10-28)

Called with a key that cannot be encoded (a function, a list), `std/map.lookup` returned `None`,
`member` returned `false` and `remove` returned the map unchanged, in both engines, with no error.
`insert` already rejected such a key, so the answer never came from data. All four operations now
fail with an error naming the type (`unsupported map key type: function`).

### Added — `std/set`: an immutable set type (2026-10-28)

`Set[a]` with `empty`, `fromList`, `insert`, `remove`, `member`, `size`, `toList` (canonical
order), `union`, `intersect` and `difference`. Elements may be anything a Map key may be, so a
visited set over grid states is `Set[(int, int, Dir)]`. It is built over today's copy-on-write Map
runtime, so `insert` and `remove` are still O(n) per call (20,000 sequential inserts take about
5.6 s); the module header says so, and the persistent-map phase of
[m-persistent-collections](design_docs/planned/v0_55_0/m-persistent-collections.md) changes the
cost without changing the API. Import the functions qualified (`import std/set as S`) and the type
by name (`import std/set (Set)`); `S.Set[...]` is not valid type syntax.
