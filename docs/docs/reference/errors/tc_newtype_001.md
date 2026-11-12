---
reviewed: 2026-11-12
reviewBy: 2027-02-12
---

# TC_NEWTYPE_001: A single-constructor type is not its record

**Code**: `TC_NEWTYPE_001`
**Category**: Type checker
**Since**: the release after v0.60.2 (#2239)

## What this error means

AILANG has two ways to name a record shape, and they mean different things:

| Declaration | What it is |
|---|---|
| `type P = { x: int }` | a **record alias**: another name for the record type. A `P` and any `{ x: int }` are the same type. |
| `type W = W({ x: int })` | an **ADT** with one constructor, `W`, wrapping a record: a new type. A `W` is built with `W({...})` and is not a record. |

Field access works on both: `w.x` reads the field of the record a `W` wraps (the teaching
prompt's "Newtype Record Access"). That is the only thing the two forms share. This error is
reported where a `W` meets a record type: a bare record passed where a `W` is expected, a `W`
passed where a record is expected, or a record update `{ w | x: 1 }` on a `W`.

Up to v0.60.2 the checker converted between the two silently, so a bare record was accepted
as a `W` (and failed at run time with `RT002` or `GET_TAG on Record`), and two different
single-constructor types over the same record could reach each other through a record-typed
value — under the bytecode VM one silently printed the other's value.

## Example

```ailang error=TC_NEWTYPE_001
module units/main

type Meters = Meters({ v: int })

pure func describe(m: Meters) -> string = "${show(m.v)} m"

export func main() -> () ! {IO} = println(describe({ v: 3 }))
```

```text
Error: type error in units/main (decl 1): ... TC_NEWTYPE_001: cannot unify Meters (defined in
units/main) with the record type { v: int }: Meters is a type with one constructor wrapping the
record { v: int }, not that record, and the two do not convert implicitly.
  Suggestion: build a Meters with its constructor, Meters({...}), or take the record out with
  match v { Meters(r) => r }; field access (v.v) works on a Meters directly.
```

## How to fix

- **Build the value with its constructor** where a `W` is expected: `describe(Meters({ v: 3 }))`.
- **Take the record out** where a record is expected: `match m { Meters(r) => r }`.
- **Update** by rebuilding: `match m { Meters(r) => Meters({ r | v: r.v + 1 }) }`.
- If the type should be interchangeable with its record, declare it as a **record alias**,
  `type Meters = { v: int }`, instead.

```ailang
module units/fixed

type Meters = Meters({ v: int })

pure func describe(m: Meters) -> string = "${show(m.v)} m"

pure func grow(m: Meters) -> Meters = match m { Meters(r) => Meters({ r | v: r.v + 1 }) }

export func main() -> () ! {IO} = println(describe(grow(Meters({ v: 3 }))))
```

## What is not an error

- Field access on a `W`, including through a lambda whose parameter type is inferred:
  `map(\t. t.name, items)` over a list of `Item({...})` values.
- Record aliases are structural, as before.

## Related diagnostics

- `TC_NOMINAL_001` — two different nominal types that share a name (for example two modules'
  `type W = W({ x: int })`).
