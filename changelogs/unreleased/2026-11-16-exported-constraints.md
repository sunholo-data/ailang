### Changed — BREAKING: class constraints cross the module boundary (#2290, #2291, #2312) (2026-11-16)

An exported function's class constraints (`Num`, `Fractional`, `Eq`, `Ord`) were dropped from
its exported scheme, and an importer instantiated the scheme without them. A caller in another
module could pass an ADT, a string or a lambda to an exported `add[a](x: a, y: a) -> a = x + y`,
`lt[a]` or `eqq[a]`: `check` passed, both engines crashed at run time, and on the VM `==` on two
lambdas printed `false`. The same calls inside the defining module were rejected.

- The exported scheme keeps its constraints (`internal/iface/builder.go`), and each use of an
  imported function re-emits them, as a same-module use does (`inferVarGlobal`). A violating
  call is the new **`TC_CONSTRAINT_001`**, located at the call, naming the function and its
  module, and reported by `check --json`:
  `` TC_CONSTRAINT_001: `add` from lib/boxes requires Num for the type it is used at here: No instance for Num[string] in scope ``.
- **BREAKING:** a program whose cross-module call breaks a constraint now fails `check`. That
  includes calls that ran correctly by accident: `eqq({a: 1}, {a: 1})` on an anonymous record,
  or `eqq([Full(1)], …)` on an ADT without `deriving (Eq)`, are rejected exactly as they are
  when `eqq` is declared in the calling module.
- **Interface surface:** `ailang iface` gains `"constraints": ["Eq[a]"]` on an export that has
  any (omitted otherwise; variable names as in `"type"`). The interface digest and the package
  `interface_hash` include them, so **`interface_hash` changes for every export with a
  constraint**, and an importer's compile-cache key moves when a dependency gains or loses
  one. Exports without constraints hash exactly as before.
- **#2291:** `std/list`'s `contains`, `member`, `dedup`, `union`, `intersect` and `difference`
  compare elements with `==` but carried no `Eq`, so `contains([\x. x + 1], \x. x + 1)` passed
  `check` and returned `false`. Their builtins now carry `Eq` on the element type
  (`BuiltinSpec.EqVars`), and the six std functions export `Eq[a]`. A list of functions, of
  anonymous records, or of an ADT without `deriving (Eq)` is now rejected with
  `TC_CONSTRAINT_001`. `.stdlib-golden/list` is re-frozen; no other std module changes.
- **#2312 (#2221 part 4):** defaulting now finds ambiguous type variables with the shared
  free-variable walk, which sees a variable under an open record. `let f = \p. {p | x: p.x + 1}`
  then `f({x: 1.5, z: 1})`, and a top-level `func addv(p, q) = {p | v: p.v + q.v}` used at
  float, were `No instance for Fractional[int]`; they now check and run. A module
  declaration's integer literal under an open record is still defaulted to `int`, so
  `func g(p) = {p | x: p.x + 1}` used at float stays a check error rather than the run-time
  error of a literal in a generic top-level function.
