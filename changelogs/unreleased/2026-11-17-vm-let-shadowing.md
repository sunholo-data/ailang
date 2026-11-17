### Fixed — a `let` inside a block used as an argument or operand captured its siblings, on every engine (#2388) (2026-11-17)

**This changes program output on all engines (evaluator, `--bytecode`, `--strict-bytecode`) for
the programs it affects; they printed a wrong answer before.** `add({ let r = 2; r * 10 }, r)`
gave 22 instead of 21: ANF completion (`extractLetBindings`, `internal/elaborate/core.go`) moved
the block's `let r = 2` out over the whole enclosing expression, so the sibling read of the outer
`r` saw the inner one. The same happened for a block, or a parenthesised `let … in`, as a call
argument (read by an earlier or a later sibling), an operator operand, a list, tuple or record
element, an `if` condition, a match scrutinee, a string interpolation, an argument of a
higher-order call (the lambda beside it captured the inner value), and the value of a `let … in`.
`ailang verify` saw the same wrong Core and reported false violations for such functions.

ANF completion now moves only its own temporaries (`$tmpN`, which no program can name); a `let`
the program wrote stays nested in the value it belongs to. No name is changed, so error messages
and the stdlib interfaces are unchanged.

### Changed — BREAKING (narrow): a binding whose value is a `let … in` or block now follows ML's value restriction (2026-11-17)

A `let` nested in a binding's value used to be floated out of it in the `let … in` form, and kept
in the block form, so the two forms typed differently. Both now follow ML's non-expansive rule:
`let x = v1 in v2` (or `{ let x = v1; v2 }`) is a value, and the binding is generalised, when
`v1` and `v2` both are. What moves:

- `let f = { let k = 1; \x. x }` used at two types was a type error; it now checks.
- such a block-bound lambda used at both int and float (`k(1)`, `k(1.5)`) was monomorphic and
  printed `2.0` for `k(1)`; it is now generalised and prints `2`, as `let k = \x. x + x` always did.
- **rejected now:** a `let … in` value whose inner `let` is a call or an effect,
  `let f = (let c = mk(0) in \x. x)`, used at two types. It checked before only because the
  inner let was floated away; the block form was already rejected. Write the inner `let` before
  the binding (`let c = mk(0); let f = \x. x`).

### Fixed — VM: a shadowing `let` inside a block overwrote the binding it shadowed (#2375) (2026-11-17)

On `--bytecode` and `--strict-bytecode`, `let r = 1; let s = { let r = 2; r * 10 }; r` read
`2`: the lowering hoists a nested let-chain's bindings into the enclosing statement list and the
compiler keeps one name-to-register map per function, so the inner `r` replaced the outer one for
the rest of the function. The evaluator printed `1`. The same leak hit a closure built after the
block (it captured the inner value), a closure built inside it, a function parameter shadowed in
the body, a tail-recursive loop (one shape never terminated), a local `letrec`, records, strings
and functions, three-deep nesting, and named-test bodies under `ailang test --bytecode`; a block
let named like a top-level function made the later call fail with "CALL on non-closure".

The VM lowering now renames, per top-level function, every local binder that would share a name
with an earlier binder or with a free reference (`uniqueLocals`, `internal/gen/lower/uniquify.go`),
so the flat name space is sound by construction for block lets, `letrec` groups, match-pattern
variables (guards included), lambda parameters and closure captures. Code without shadowing
lowers unchanged.

### Fixed — compile passes that resolve a variable by name ignored lexical scope (2026-11-17)

Three pipeline passes looked a variable up by NAME in a map that lambda parameters and pattern
variables did not shadow:

- monomorphisation (`specialize_expr.go`) rewrote a call of a parameter `f` into a call of an
  enclosing `let f = \x. x`: `let f = \x. x; let g = \f. f(3); g(\z. z + 1)` printed `3`, not
  `4`, on every engine (also through a match arm's `Some(f)`, and before a letrec member was
  processed);
- operator lowering (`op_lowering.go`) gave a pattern variable or parameter of a type parameter
  the type of ANY earlier `let` of its name (the map was never cleared, across declarations):
  `let x = 1; match o { Some(x) => x == k }` lowered `==` to `eq_Int` and failed on strings, and
  `/` on ints became float division under `--strict-bytecode`;
- Var type resolution (`resolve_vars.go`) had the same blind spot and deleted the outer entry
  after an inner `let` instead of restoring it.

Each pass now scopes its map: a parameter, a pattern variable or a letrec member hides an outer
binding of its name, and an inner `let` restores the outer one after its body.

On a fixed-seed corpus of 360 generated shadowing programs, each checked against a twin with every
binder renamed apart, dev printed a wrong answer (or failed) on every engine for 92 and on both VM
engines for 126 more; now none.
