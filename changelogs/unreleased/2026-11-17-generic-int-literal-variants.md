### Fixed — an integer literal in a generic top-level function runs at the type of its argument (#2378, #2333)

`func inc(x) = x + 1` has the type `Num a => a -> a`, and since #2364 an importer sees that
type, but its body was compiled once, at the default `int`: `inc(2.5)` passed `check` and failed
at run time ("numeric dictionary method add received an int and a float" on the evaluator,
"arith ADD: type mismatch Float vs Int" on the VM). The same for a literal under a record, tuple,
list, `Option` or a nested structure, for `if x > 0 then x else 0`, `x % 3`, `x == 0` (the
evaluator crashed, the VM said `false`), recursion and mutual recursion.

A top-level binding generic over a numeric type that occurs in a **parameter** type is now
compiled once per instance of it: as written, for `int`, and as a variant typed at `float`
throughout (`inc$numF`: its literal `1.0`, its operators, `show` and `%` at float). Every use
picks the variant its argument types name, in the module and from importers (the variant is
derived from the exported type, so interfaces do not list it), and a use at `int` runs exactly
the code it ran before: `inc(2)` is `3`, `show(1 + 1)` is `2`, `10 / 4` is `2`. Results are the
same on the evaluator, `--bytecode` and `--strict-bytecode`. A contract on such a function is
checked at the type of the call (every clause, `result`, clauses calling other generic
functions); `ailang verify` counts it once.

A let-bound lambda that calls such a function (`let h = \u. inc(u)`) gets a copy per use:
`h(2)` is `3`, `h(2.5)` is `3.5`. A literal-free one (`let dbl = \u. add(u, u)`) is unchanged. One
with its own integer literal (`let f = \x. x + 1`) is one value whose uses decide its type, as
before. Patterns are unchanged: `match x { 0 => … }` makes `x` an `int`.

The exception #2364 kept (`literalAtDecl`: `func f(p) = {p | x: p.x + 1}` used at float was
`No instance for Fractional[int]`) is gone: that function now runs at float.

### Changed — BREAKING: a numeric constant or result-only literal is an `int` (#2378)

A numeric type variable of a top-level binding that occurs in no parameter type has no argument
to tell a call its type, so it is no longer generic: it defaults to `int` (to `float` when the
literal is written `3.0`, or by an annotation).

- `let K = 3` is an `int`: `K * 0.5` is a check error at the use, whose hint says to write `3.0`,
  annotate (`let K: float = 3`), or convert. Before, it passed `check` and failed at run time.
- Exported types narrow: `func K() = 3` exports `() -> int` (was `Num[a] => () -> a`),
  `func cfg() = {lo: 1, hi: 10}` exports `() -> {hi: int, lo: int}`, `W(x) = {a: x + 1, b: 0}`
  exports `Num[a] => (a) -> {a: a, b: int}`, `pairUp(x)` returning `(h(x), h(1))` exports
  `(a) -> (a, int)` (as its result printed before). Their interface hashes change.
- An exported function whose literal sits under an open record
  (`export func bump(f) = {f | v: f.v + 1}`) now exports `Num[b] => ({v: b, ...a}) -> …` instead
  of `({v: int, ...a}) -> …`.
- `TC_NUMVAR_001`: a binding generic over more than 4 parameter-borne numeric types gets only the
  variants its own module compiles; another module's use at a combination it did not compile is
  this check error, located at that use (it failed at run time before).
- A let-bound recursive lambda (`letrec`) that calls a literal-carrying generic function at its
  own variable is one value: used at both int and float it is a type error at `check` (the float
  call failed at run time before).

Measured against origin/dev 35f5ad6d6; the numbers were identical against 950a0cc54:
- 14,857 banked eval programs: the `check` verdict and error code are identical on all of them.
  11,328 pass on both, 0 are newly rejected, and 0 hit `TC_NUMVAR_001`.
- 136 passing programs are touched by the change, and a random control of 400 is not. Both sets
  print byte-identical output on the evaluator and on `--strict-bytecode`.
- A 340-program defaulting sweep (102 groups of top-level, let and cross-module forms):
  - 298 print the same output;
  - 24 that failed at run time now run;
  - 18 are newly accepted;
  - 0 are newly rejected.
  The top-level, let and cross-module forms of a group now agree in 102 of 102 groups, against
  81 of 102 before.
- 1,004 tracked `.ail` files: 0 verdict or interface changes. The 605 that have a `main` print
  the same output on both engines.
- `ailang iface` over std, the tracked modules and 622 ailang-packages files: 0 diffs.
- An `int` loop runs in the same time as before.
