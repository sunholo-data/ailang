### Changed — BREAKING: signature type variables are rigid (TC_RIGID_001, #2273)

A type variable written in a top-level `func`'s signature — listed in `[a]`, used implicitly
(`func f(x: elem) -> elem`), or a record row (`{name: string | r}`) — now means *any type* (any
further fields), and the body and its `requires`/`ensures` clauses must work for every one. Until
now the body could quietly narrow it, so the signature promised nothing: `func toInt[a](x: a) -> int = x`,
`func f[a](x: a) -> a = 1`, `func pick[a, b](x: a, y: b) -> a = y` and
`func nick(p: {name: string | r}) -> string = p.nick` type-checked, and a misspelled or miscased type
name (`state` for `State`, `-> (stack, int)` with `stack` a parameter, `map` imported as a value)
became a variable the body silently pinned. Each is now `TC_RIGID_001` at `check`, in human,
`--json` and `--format agent` output, naming the variable, what the body needs it to be and where.
A type the message tells you to write is one you can paste (`{n: int | r}`, `[int]`); when the body
fixes only its shape, the message gives the shape (`(_) -> _`) instead. A "did you mean the type
`State`?" hint is given only when it agrees with what the body needs; a name #2286 reads as a type
variable (`TC_UNBOUND_TYPE_002`) carries that reading in the error too.

- An integer literal used at a signature variable (`func bump[a](x: a) -> a = x + 1`) is rejected:
  a literal has no value of an arbitrary type, and the runtime gave it an `int` (a float call
  failed at run time). Write `int` or `float` in the signature.
- Class constraints are still inferred and attached to the scheme (`func add[a](x: a, y: a) -> a = x + y`
  is fine): there is no syntax for writing one.
- Inside a declaration, an annotation (a `let x: T`, or a function literal's `(y: T) -> U`) that
  names an enclosing signature variable means that rigid variable; its own new variables are
  equalities on fresh variables, as before (#2289), so `let id = func(y: t) -> t { y }` stays
  let-polymorphic.
- Contract clauses see the parameters at their rigid types: `ident[a](x: a)` with
  `requires { x + 1 > 0 }` is `TC_CONTRACT_001` carrying `TC_RIGID_001`.
- Numeric defaulting no longer pins a signature variable behind the exported scheme: an inner
  `let` leaves a variable an enclosing parameter owns alone, and `show` of a value typed through
  a signature variable stays dynamic when the generic body is compiled at its defaulted type
  (`label[a](x: a, y: a) -> string { … show(x * y) … }` called at `float` printed with intToStr and
  crashed).
- Builtin schemes quantify all their type variables (#2305): `_list_length : list[a] -> int` and
  27 others left `a`/`b`/`k`/`v` free, so all their uses shared one variable, and
  `both[a, b](xs: [a], ys: [b]) = _list_length(xs) + _list_length(ys)` was wrongly rejected.
- Each declaration's signature variables are now its own (they no longer share a name with every
  other `a` in the module). Two dev bugs go with that: `sh[a](x: a) -> string = show(x)` next to
  `add[a]` was compiled at `int` and failed on a float, and **`compile --emit-go`** emitted a generic
  export's wrapper at another declaration's defaulted type — `ident[a]` next to `add[a]` was
  `func Ident(x int64) int64` and is now `func Ident(x interface{}) interface{}`. Go code calling
  such a wrapper with a typed value still compiles; code that relied on the `int64` result type
  needs a type assertion.
- Measured: MEASURED_PLACEHOLDER. Known gaps are listed in docs/LIMITATIONS.md.
