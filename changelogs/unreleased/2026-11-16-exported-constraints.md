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
- **BREAKING, silent:** an imported function's `Num`/`Fractional` constraint now types the
  caller's numeric literals, so a program that already passed `check` can **print different
  numbers with no diagnostic**. With `export pure func pick(c, x) = if c then x else 0.5` in a
  library, an importer's `show(pick(true, 3))`, `pick(true, 3) + 1` and `pick(true, 7) / 2`
  printed `3 4 3` and now print `3.0 4.0 3.5` on every engine: `x` is `Fractional`, so the
  literals are floats and the last `/` is float division, not integer division. This is what
  the same code already printed when `pick` was declared in the calling module.
- **Interface surface:** `ailang iface` gains `"constraints": ["Eq[a]"]` on an export that has
  any (omitted otherwise; variable names as in `"type"`). The module interface digest (an
  importer's compile-cache key moves when a dependency gains or loses a constraint) and the
  registry's signature-sensitive **v2 interface hash** (`interface_hash_v2`, `pkg quality`,
  `HashProjection`) and signature set include them, so **the v2 hash changes for a package
  with a constrained export**. The lockfile `interface_hash` (v1) is computed from the
  manifest alone and does **not** change. Exports without constraints hash exactly as before.
  Operational: a publisher and a registry validator on different AILANG versions (either side
  of this change) disagree on the v2 hash of such a package and the publish is refused with
  `PUB005`, so the validator must be deployed in step with this release. Measured: no package
  in sunholo-data/ailang-packages or the motoko fork has a constrained export.
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
