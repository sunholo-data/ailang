### Changed — BREAKING: a constructor pattern whose type cannot be named is `TC_MATCH_007` (#2303, #2272)

A constructor can be matched by name without importing it or its type (#323), but only against a
value whose type says which type the constructor belongs to. When nothing fixed that type — a
generic or unannotated parameter, a let-bound lambda, a function literal, a lambda passed to a
generic function, a pattern nested in a tuple, a contract clause, a test body — a final `_ =>`
arm let the match through, and the back ends guessed the type: the bytecode VM took the first
loaded type declaring the name, the Go emitter whichever its map yielded. With lib/x's
`Color = Red | Green | Blue` and lib/y's `Light = Green | Red`,
`isRed[a](x: a) = match x { Red => 1, _ => 0 }` printed `[1, 0, 0, 1]` on the evaluator and
`[1, 0, 1, 0]` on `--bytecode`/`--strict-bytecode` (exit 0), and the emitted Go panicked.

Such a pattern is now `TC_MATCH_007` at the pattern, on `check` (also `--json`, code
`TC_MATCH_007`) and every path that compiles the program, whatever the other arms are. The
suggestion names the loaded types declaring the constructor: import the one you mean, or
annotate the value matched. The same code covers a by-name constructor matched against a value
whose type has no such constructor (an arm that can never match), and replaces the
`TC_MATCH_002` "cannot be checked" message for a by-name pattern on an untyped value without a
catch-all.

The back ends no longer guess: the VM lowering, the bytecode compiler and the Go emitter take a
constructor pattern's type from the type checker's stamp only and refuse a pattern without one.
The bytecode compiler's tag-to-type inference (`inferADTFromTags`) and the Go emitter's
bare-name and `Option` fallbacks are gone.

#2272: the arity check no longer compares an unplaced by-name pattern with every loaded type
declaring the name (a verdict that changed when an unrelated module was added to the import
closure); such a pattern is `TC_MATCH_007` before its arity is compared with anything.

Measured: 0 of 12,153 programs that pass `check` on dev (983 tracked `.ail` files and 14,813
banked passing eval solutions, 12,153 of which check) contain such a pattern; bytecode
disassembly is identical for every one of them.

### Fixed — `compile --emit-go` emitted imported type declarations in Go map order (#2359)

The Go emitter (and the experimental `--emit-go-v2` path) gathered the type declarations of
imported modules by ranging a map of loaded modules, so the emitted `types.go` — and which of two
same-named types was kept — could differ from one run to the next. They are now gathered in
module-path order. Other map-ordered output in the Go emitter (record literals) is #2360.
