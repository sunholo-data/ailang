### Fixed — an integer literal in a generic top-level function runs at the type it is used at (#2378, #2333)

`func inc(x) = x + 1` has the type `Num a => a -> a`, and since #2364 an importer sees that
type, but its body was compiled once, at the default `int`: `inc(2.5)` passed `check` and failed
at run time ("numeric dictionary method add received an int and a float" on the evaluator,
"arith ADD: type mismatch Float vs Int" on the VM). The same for a literal under a record, tuple,
list, `Option` or a nested structure, for a top-level `let K = 3` used as `K * 0.5`, and for
`if x > 0 then x else 0`, `x % 3`, `x == 0`, recursion and mutual recursion.

A top-level binding generic over `Num a` is now compiled once per instance of `a`: as written, for
`int`, and as a variant typed at `float` throughout (`inc$numF`, its literal `1.0`, its operators,
`show` and `%` at float). Every use picks the variant its instantiation names, in the module and
from importers (the variant is derived from the exported type, so interfaces do not list it), and
a use at `int` runs exactly the code it ran before: `inc(2)` is `3`, `show(1 + 1)` is `2`,
`10 / 4` is `2`. Results are the same on the evaluator, `--bytecode` and `--strict-bytecode`.
A contract on such a function is checked at the type of the call. `ailang verify` counts a
contract once, on the function as written.

A let-bound lambda inside a declaration is one run-time value: a literal there (or, now, a use
of a generic top-level binding, `let g = \y. inc(y)`) keeps it monomorphic, so its uses decide its
type, as before. Patterns are unchanged: `match x { 0 => … }` makes `x` an `int`.

The exception #2364 kept (`literalAtDecl`: a declaration's literal under an open record defaulted
to `int`, so `func f(p) = {p | x: p.x + 1}` used at float was `No instance for Fractional[int]`) is
gone; that function is now generic and runs at float.

### Changed — BREAKING: three narrow cases (#2378)

- An exported function whose integer literal sits under an open record
  (`export func bump(f) = {f | v: f.v + 1}`) now exports `Num[b] => ({v: b, ...a}) -> {v: b, ...a}`
  instead of `({v: int, ...a}) -> {v: int, ...a}`: its interface (and `interface_hash_v2`) changes.
- A binding generic over more than 4 independent numeric types (a record constant of nine integer
  fields) gets only the variants its own module uses, so another module may use it only at int: a
  float use from another module is the new `TC_NUMVAR_001`, located at the use (it failed at run
  time before).
- A let-bound lambda that calls a generic top-level binding (`let g = \y. inc(y)`) is monomorphic:
  used at both int and float it is a type error at `check` (the float call failed at run time
  before).

Measured against origin/dev: of 11,328 banked passing solutions none is newly rejected, and the
107 that define or use a generic numeric binding print the same output on the evaluator and
`--strict-bytecode`; 989 tracked repo `.ail` files keep their `check` verdict and `ailang iface`
output; no std module has a generic numeric binding, so no std interface changes. Int code runs
the same code at the same speed (a 3M-iteration generic loop: 5.03 s both on the evaluator,
503/505 ms on the VM), and the float variant runs as fast as a hand-written float twin.
