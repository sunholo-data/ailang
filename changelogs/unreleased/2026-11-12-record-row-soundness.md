### Changed — BREAKING: record rows are unified soundly; missing fields and infinite record types are errors (#2210, part of #2221) (2026-11-12)

`ailang check` accepted programs that read a field a record does not have, and
could be made to run forever on a small program. Programs that now fail `check`
all failed at run time before (or never finished checking).

- **A field access through a lambda parameter is checked against the record it
  receives (#2210).** `let f = \acc. … acc.score … in f({label: "none", score: 0.0})`
  in a function declared `-> {label: string, score: float}` exported the shim
  `{score: float, ...}`, so `best().bogus` type-checked and failed with
  `record has no field: bogus`. The shim's row is now bound to the record's
  remaining fields. The same holds through `foldl`, within a module and across
  modules. New error: `TC_REC_001: field 'bogus' not found in record {label, score}`.
- **A record update keeps the other fields (#2221).** `let upd = \q. {q | x: 1}`
  applied to `{x: 0, y: "s"}` lost `y`'s type, so `.y + 1` type-checked against a
  string. It is now `No instance for Num[string]`.
- **`check` terminates on an infinite type through a record (#2221).** The occurs
  check had no record cases; `\x. {self: x} == x`, `letrec f = \x. {inner: f(x)}` and
  `\x. g(x) == x` with `g(p: {self: a, ...}) -> a` ran forever. They now fail with
  `TC004: occurs check failed: … (an infinite type)` (`TC_REC_003` when a record
  row would contain itself).
- **A user-written `| r` works (#2221).** `func bump(p: {x: int | r}) -> {x: int | r}`
  was rejected with `r has kind Row Record, but r has kind Row Record`: the
  annotation's row carried a pointer kind that did not equal the checker's, and
  every field access shared one row named `r`. Each access and update now has its
  own fresh row. The same pointer kind stopped a module exporting an annotated open
  record from being written to the compile cache (`CACHE_WRITE_FAILED … unsupported
  kind *types.KRow`); it is now cached.
- **A row variable shared by two records cannot give one of them a field
  twice.** `func f(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q` called
  with `({y: 1, x: "s"}, {x: 2})` binds `r := {x: string}` and so gave `q` the
  field `x` twice (`int` and `string`); which one a later access saw depended on
  map order. The two copies are now one field: they must agree, else
  `TC_REC_004: field 'x' would have two types, int and string`. (The definition
  itself is still accepted; a lacks constraint at definition time is not
  implemented.)
- Two closed records of the same size with different fields now say so
  (`both records have 2 fields, but not the same ones`) instead of
  `expected 2 fields, got 2`, and a record literal no longer prints its fields
  twice in messages (`{ self: α1, ...{self: α1} }`).

`check --json` reports `TC_REC_001`, `TC_REC_003`, `TC_REC_004` and `TC004` as their codes.

**Interface changes.** Two exports over shim records no longer claim to share their
other fields: `export pure func f(a, b) = a.x + b.y` renders
`({x: b, ...a},{y: b, ...c})->b` (was `...a` twice, which with bound rows would
reject `f({x: 1, z: "a"}, {y: 2, w: true})`). In `std/sem`, `encode_frame`,
`store_frame` and `store_frame_ns` now require every field they read (they named
only `id`, so `encode_frame({id: "x"})` type-checked and failed at run time), and
`create_or_update_frame` takes the full frame record. `.stdlib-golden/sem.*` is
re-frozen. The 18 in-repo packages' `interface_hash` v2 is unchanged.

**Not changed (#2221 part 4).** Defaulting keeps its own free-variable walk.
Routing it through the shared walk makes `std/sem.bump_version` polymorphic in
`ver`, and an exported scheme's class constraints do not reach the importer, so
`bump_version({ver: "s", id: "x"})` would pass `check` and fail at run time — the
same hole `export pure func inc(x: a) -> a = x + 1` has today. So
`let f = \p. { let y = {p | x: p.x + 1}; y.x }; f({x: 1.5, z: 1})` is still
rejected with `No instance for Fractional[int]` (a correct program rejected since
v0.30.0); #2221 stays open for it.
