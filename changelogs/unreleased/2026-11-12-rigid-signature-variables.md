### Changed — BREAKING: signature type variables are rigid (TC_RIGID_001, #2273)

A type variable written in a top-level `func`'s signature — listed in `[a]` or used implicitly
(`func f(x: elem) -> elem`) — now means *any type*, and the body must work for every one. Until
now the body could quietly narrow it, so the signature promised nothing: `func toInt[a](x: a) -> int = x`,
`func f[a](x: a) -> a = 1` and `func pick[a, b](x: a, y: b) -> a = y` type-checked, and a
misspelled or miscased type name (`state` for `State`, `-> (stack, int)` with `stack` a parameter)
became a variable the body silently pinned. Each is now `TC_RIGID_001` at `check`, in human,
`--json` and `--format agent` output, naming the variable, what the body needs it to be and where.
A type the message tells you to write is one you can paste (`{n: int | r}`, `[int]`); when the body
fixes only its shape, the message gives the shape (`(_) -> _`) instead. A "did you mean the type
`State`?" hint is given only when it agrees with what the body needs.

- An integer literal used at a signature variable (`func bump[a](x: a) -> a = x + 1`) is rejected:
  a literal has no value of an arbitrary type, and the runtime gave it an `int` (a float call
  failed at run time). Write `int` or `float` in the signature.
- Class constraints are still inferred and attached to the scheme (`func add[a](x: a, y: a) -> a = x + y`
  is fine): there is no syntax for writing one.
- A `let x: a` or a function literal's `(y: a)` inside the body names the signature's `a`; their
  own new variables stay flexible. A lowercase name the module declares, or imports by name, as a
  type is not made rigid (that it is the type is #2211/#2240); one an import merely exports is.
- Numeric defaulting at an inner `let` no longer defaults a variable an enclosing parameter owns
  (it is not ambiguous there). It used to pin a signature's `a` to `int` behind the exported scheme.
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
- Measured: 4 of 11,180 banked passing AI solutions newly fail, all wrong or over-general
  annotations; std/ and examples unchanged; `.stdlib-golden` unchanged. Downstream: one site,
  ailang-packages `packages/agui/input.ail:54` (`decodeAll[f, b](f: f, …)` calls `f`; the fix
  `f: (Json) -> Result[b, string]` checks on both versions). Known gaps are listed in
  docs/LIMITATIONS.md.
