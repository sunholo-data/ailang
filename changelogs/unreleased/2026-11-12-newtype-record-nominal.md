### Changed — BREAKING: a single-constructor type wrapping a record is not that record (#2239) (2026-11-12)

`type W = W({ x: int })` (the M-STREAM-DX/M4 "newtype" shape) was registered as an
alias of its record so that `w.x` worked, and the unifier expanded it into the record
whenever the two met. So a bare record was accepted where a `W` was expected (run time:
`RT002` / `GET_TAG on Record`), a `W` could be returned as a record, and two such types
over the same record reached each other through any record-typed value — under the
bytecode VM, `Meters` passed as `Feet` printed a wrong value with exit 0 (#2233, #2234).

The semantics are now the ones the teaching prompt and the docs already describe:
`type P = { x: int }` is a structural record alias; `type W = W({ x: int })` is an ADT,
built with `W({...})` and taken apart with `match w { W(r) => ... }`, and its fields can
be read directly (`w.x`). Nothing else converts between a `W` and a record.

How: a newtype never expands in the unifier. Field access is a deferred constraint
(`types/field_access_deferred.go`, the same shape as the deferred constructor choice)
decided once the receiver's type is known — the field of the record a newtype wraps,
or the usual `{f: α | r}` record equation — so `map(\t. t.name, items)` and
`\w. if w.x > 0 then w else other` keep working on a `W`. A let-bound field reader
(`let getN = \r. r.name`) carries its undecided access in its scheme — the field's
type is quantified with it, so `let describe = \p. show(p.x)` stays usable at
`{x: 1}` and `{x: "s"}` — and stays polymorphic over records and newtypes; an access
nothing decides is read as a record access at the end of its top-level declaration,
as before (its scheme is exported).

Newly rejected, with `TC_NEWTYPE_001` (`check --json` reports it as itself; page
`docs/docs/reference/errors/tc_newtype_001.md`): a record where a `W` is expected, a `W`
where a record is expected (a record-typed parameter, return annotation or `let`
annotation), and a record update `{ w | x: 1 }` on a `W` (which crashed at run time on
both engines). Also a `W` passed to a **top-level** function whose parameter type was
inferred from field reads (`pure func getx(r) = r.x`; `getx(W({x: 2}))` worked by
accident before): that function takes records, and the message says to annotate the
parameter as `W`. A let-bound or inline reader still takes both. Migration: wrap with the constructor, `W({...})`; unwrap with
`match w { W(r) => r }`; update with `match w { W(r) => W({ r | x: 1 }) }`; or declare
the type as a record alias if it is meant to be interchangeable with its record.
