### Fixed — an integer literal in a generic function runs at the type of its argument (#2378, #2333)

`func inc(x) = x + 1` has the type `Num a => a -> a`, and since #2364 an importer sees that type.
But its body was compiled once, at the default `int`, so `inc(2.5)` passed `check` and then failed
at run time:
- the evaluator said "numeric dictionary method add received an int and a float";
- the VM said "arith ADD: type mismatch Float vs Int".

The same happened for a literal under a record, tuple, list, `Option` or nested structure, for
`if x > 0 then x else 0`, `x % 3` and `x == 0` (the evaluator crashed, the VM answered `false`),
and in recursion and mutual recursion.

**Top-level functions.** A function whose numeric type variable carries an integer literal (its
own, or a callee's) is now compiled once per instance of that variable: as written, for `int`,
and as a variant typed at `float` throughout (`inc$numF`, whose literal is `1.0` and whose
operators, `show` and `%` work at float).
- Every use picks the variant its argument types name, in the module and from importers.
- A use at `int` runs exactly the code it ran before: `inc(2)` is `3`, `show(1 + 1)` is `2`,
  `10 / 4` is `2`.
- A literal-free generic (`add(a, b) = a + b`) has one body that is right at every instance, so
  it gets no variants and compiles exactly as before.
- Results are the same on the evaluator, `--bytecode` and `--strict-bytecode`.
- A contract on such a function is checked at the type of the call: every clause, `result`, and
  clauses that call other generic functions. `ailang verify` counts it once.

**Let-bound functions follow the same rule.** This covers a let whose value is a lambda, a block
of values ending in a lambda, or a `letrec` of one such binding. Each use gets its own copy,
whether the literal is the function's own or a callee's:
- `let f = \x. x + 1` gives `f(2) == 3` and `f(2.5) == 3.5`;
- `let g = { let k = 1; \u. inc(u) }` behaves the same way;
- so does `letrec s = \n. if n <= 0 then 0 else n + s(n - 1)`.

The same body now behaves the same as a top-level function, a let-bound lambda, a block-bound
lambda, and when imported. A binding whose value is computed (`if … then \u. … else …`, a call)
is a single value: its uses decide its one type. Calling it at both `int` and `float` is a
located check error that says exactly that, instead of a crash or a silently wrong number.

The exception #2364 kept is gone. It was `literalAtDecl`: `func f(p) = {p | x: p.x + 1}` used at
float gave `No instance for Fractional[int]`. That function now runs at float.

### Changed — BREAKING: a literal that no parameter carries is an `int` (#2378)

A numeric type variable of a top-level binding defaults to `int` when **both** of these hold:
- it carries an integer literal, directly or through a callee;
- it occurs in no parameter position of any function type in the binding's type.

It defaults to `float` instead when the literal is written `3.0`, or when an annotation says so.
Such a variable has no argument to tell a call its type, so it cannot be generic. Every other
variable keeps its generality.

- **Now `int`:** `let K = 3`, `func K() = 3`, `func cfg() = {lo: 1, hi: 10}`, `func one(x) = 1`,
  `len(xs)`, and the second component of `pairUp(x) = (h(x), h(1))`. `K * 0.5` is now a check
  error at the use. The message names the binding and where its literal is, and says to write
  `3.0`, to annotate it as float, or to convert. Before, it passed `check` and failed at run time.
- **Still generic:**
  - literal-free bindings, including containers of them: `let ops = (add, mul)`,
    `[add, mul]`, `{plus: add}`, `func getOps() = (add, mul)`,
    `func pick(b) = if b then Some(add) else None`;
  - a function in a result, such as `func mk() = \x. x + 1`;
  - a variable a parameter carries.
- **Exported types narrow** where the rule applies:
  - `func K() = 3` exports `() -> int` (it was `Num[a] => () -> a`);
  - `cfg` exports `() -> {hi: int, lo: int}`;
  - `W(x) = {a: x + 1, b: 0}` exports `Num[a] => (a) -> {a: a, b: int}`.
- An exported function whose literal sits under an open record
  (`export func bump(f) = {f | v: f.v + 1}`) now exports `Num[b] => ({v: b, ...a}) -> …` instead
  of `({v: int, ...a}) -> …`.
- **`TC_NUMVAR_001`:** a binding generic over more than 4 literal-carrying numeric types gets only
  the variants its own module uses. A use from another module at a combination that was not
  compiled is this check error (before, it failed at run time). It is located at that use and
  names the binding that has too many types, and the chain to it:
  "`V2` from lib/v2 has no variant for this use at float: it needs `W5` from lib/w (used at …),
  which is not compiled with 1 of its 5 numeric types at float".

**Interface surface.** `ailang iface` gains `"literals": ["a"]` on an export, listing the type
variables (named as in `"type"`) whose `Num` constraint carries an integer literal. These are
the variables the export has compiled variants over. The field is omitted when there are none,
so a literal-free generic is unchanged.
- The module interface digest includes the field, so an importer's compile-cache key moves
  when a dependency gains or loses a literal.
- The registry's v2 interface hash (`HashProjection`) and the signature set include it, so **the
  v2 hash changes for a package with a literal-carrying generic export**.
- The lockfile `interface_hash` (v1) is computed from the manifest alone and does not change.
- Operational: a publisher and a registry validator on different AILANG versions disagree on the
  v2 hash of such a package, and the publish is refused with `PUB005`. Deploy the validator in
  step with this release.

Measured against origin/dev: MEASUREMENTS.
