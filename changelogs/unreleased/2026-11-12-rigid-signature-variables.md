### Changed — BREAKING: signature type variables are rigid (TC_RIGID_001, #2273)

A type variable written in a top-level `func`'s signature — listed in `[a]`, used implicitly
(`func f(x: elem) -> elem`), or a named record row (`{name: string | r}`; an anonymous `...` stays a
flexible wildcard) — now means *any type* (any
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
  `requires { x + 1 > 0 }` is `TC_CONTRACT_001` carrying `TC_RIGID_001`, and so is
  `ensures { x == "s" || true }`. A clause that only needs a class the body does not
  (`idc[a](x: a) -> a ensures { result == x } { x }`) is still accepted, as before: callers are not
  held to it, and there is no syntax to declare it (#2337; neither the repo's 117 contract files nor
  651 banked contract programs have such a clause).
- A generic body (a signature's `a`, an unannotated function, a let-bound lambda) is compiled
  once at its defaulted type (`Num a` ⇒ `int`, #2333); the evaluator and the VM now give it the
  run-time value's answer everywhere: `show` stays dynamic on defaulted nodes and `mod_Int` is
  corrected like the Num methods. `label[a](x: a, y: a) -> string { … show(x * y) … }`,
  `pure func h(x) = show(x + x)` and `m[a](x: a, y: a) -> a = x % y` crashed when called with
  floats. `compile --emit-go` still compiles such a body at int (#2332). An inner `let` no longer
  defaults a variable an enclosing parameter owns.
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
- Measured (base = dev 5db322d71): 4 of 11,168 banked passing AI solutions newly fail, all wrong
  or over-general annotations (the row, literal and contract rules add none); 0 of 700 passing repo
  `.ail` files; `.stdlib-golden` unchanged. Downstream: ailang-packages `agui` 0.2.2 and `discord`
  0.5.1 and the motoko fork's `src/core` (71 files) check as before; the one site found earlier
  (`agui` `decodeAll[f, b](f: f, …)`) was fixed upstream. Known gaps are listed in docs/LIMITATIONS.md.
