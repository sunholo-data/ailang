### Changed — BREAKING: a constructor pattern whose type cannot be named is `TC_MATCH_007` (#2303)

A constructor can be matched by name without importing it or its type (#323) — a library module
that uses `Some` without `import std/option`, say. Its type has to come from the value matched.
When nothing fixed it, a final `_ =>` arm let the match through and the back ends guessed the
type: the bytecode VM took the first loaded type declaring the name, the Go emitter whichever its
map yielded. With lib/x's `Color = Red | Green | Blue` and lib/y's `Light = Green | Red`,
`isRed[a](x: a) = match x { Red => 1, _ => 0 }` printed `[1, 0, 0, 1]` on the evaluator and
`[1, 0, 1, 0]` on `--bytecode`/`--strict-bytecode` (exit 0), and the emitted Go panicked.

Such a pattern is now `TC_MATCH_007`, located at the constructor, on `check` (also `--json`) and
every path that compiles the program, whatever the other arms are. Still rejected, by design:

- a constructor matched by name against a signature type variable (`x: a`, also in a lambda
  passed to a generic function or in a contract clause) — "a is a type variable of the
  signature": write the value's type instead, and import it;
- a by-name constructor on an unannotated top-level parameter (`func isRed(x)`), alone or nested
  in a tuple, in a module that does not import the type — import the type you mean;
- a by-name constructor matched against a value whose type has no such constructor (an arm that
  can never match) — remove the arm.

A let-bound function that matches by name is decided by its uses (as for an ambiguous import,
#2228): used at one type it runs; used at two it is `TC_AMBIGUOUS_CTOR_001`. The message for a
by-name pattern on an untyped value without a catch-all moves from `TC_MATCH_002` to
`TC_MATCH_007`.

The back ends no longer guess: the VM lowering, the bytecode compiler and the Go emitter take a
constructor pattern's type from the type checker's stamp only and refuse a pattern without one
(the bytecode compiler's tag-to-type inference and the Go emitter's bare-name and `Option`
fallbacks are gone).

### Fixed — a constructor matched by name had untyped arguments (#2353, #2272)

A by-name constructor pattern bound its arguments to fresh type variables tied to nothing, so
`match nth([1, 2], 0) { Some(v) => v ++ ["a"], None => [] }` passed `check` and crashed on every
engine, and a constructor nested in one (`Some(Red)`, `MkBox(Red)`, `Some(Some(x))`) could not
have its type fixed even when the value's type was annotated. A by-name pattern is now decided by
the value matched: its type is the value's, its arguments are typed from that type's constructor
at the value's type arguments, and a by-name pattern nested in it is decided the same way. Using a
bound variable at the wrong type is a `check` error.

#2272: the arity check no longer compares a by-name pattern with every loaded type declaring the
name. `\o. match o { Some(x) => x, _ => 0 }` applied to an Option, with an unrelated
`Some(int, int)` loaded, was `TC_MATCH_006`; it is accepted and runs the same on every engine.

Constructor patterns are located at the constructor's name in every diagnostic (a nullary one had
no position; one with arguments pointed at its `(`).

Measured over 983 tracked `.ail` files, 14,814 banked passing eval solutions and 968 files of six
downstream repositories: no `check` verdict changes (16,765 programs; three `TC_MATCH_006` messages
move from the `(` to the constructor name). Bytecode is identical except where a variable bound under a by-name
pattern now has its real type (14 downstream files: a record field read by position instead of by
name, a string no longer passed through `show` in interpolation).

### Fixed — `compile --emit-go` emitted imported type declarations in Go map order (#2359)

The Go emitter (and the experimental `--emit-go-v2` path) gathered the type declarations of
imported modules by ranging a map of loaded modules, so the emitted `types.go` — and which of two
same-named types was kept — could differ from one run to the next. They are now gathered in
module-path order. Other map-ordered output in the Go emitter (record literals) is #2360.
