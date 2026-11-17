### Fixed — the REPL and the wasm playground rejected a let inside an expression (#2407) (2026-11-18)

A regression from #2388: the elaborator now keeps a let the program wrote nested inside the value
it belongs to, and `elaborate.VerifyANF`, which only the REPL runs (so also the wasm
playground's `ailangEval`), still described the old floated shape. Every block or let-in used as a
call argument, an operator operand, a list/tuple/record element, an if condition, a match
scrutinee or a let's value failed with `ANF verification error … let bindings are not simple
calls`. The verifier now states the shape the elaborator produces: operands (arguments,
operator sides, elements, fields, conditions, scrutinees) are atomic, and a let's value, a lambda
or let body, a branch, a match arm and its guard may be any such expression, including another
let. It also accepts a tuple anywhere (before, `(1, 2)` and `let t = (1, 2) in t` were rejected
as an "unknown expression type"), and an if or match as a let's value. A position matrix of
24 inputs now gives the same value in the REPL, the wasm build and `ailang run` on the evaluator
and `--strict-bytecode`.

Also fixed in the REPL session, found on the way:

- **A later `let` reached closures made earlier.** After `let k = 10`, `let f = \x. k`,
  `let k = 99`, `f(3)` printed 99 (lexical scope gives 10): top-level bindings were written into
  one shared environment, so a later binding replaced the one a closure had captured. Each
  top-level `let` now binds in a new child environment; `k` is 99 for later inputs and `f` keeps
  10.
- **`:reset` lost every binding made after it,** and `println` was undefined after it: the reset
  replaced the REPL's copy of the environment but not the evaluator's, and a fresh environment
  without the builtins or the prelude's types. The session's bindings now live in a child of the
  evaluator's own environment, which `:reset` replaces in both.
- **Modules loaded through the REPL's module registry kept #2401's capture.** The registry (the
  REPL's imports, and the wasm playground's `ailangLoadModule`/`ailangCall`) compiles a module
  through its own copy of the phases and did not make local names unique before
  monomorphisation, so `let f = \x. k; let g = \k. f(3); g(99)` in a loaded module returned 99
  where `ailang run` returns the parameter. It now does, as the native pipelines do.
- **Structured types printed as `Int`.** `[1, 2] :: Int`, `{a: 1, b: "x"} :: Int`,
  `\x. x + 1 :: Int`: the type shown was taken from the expression only when it was a bare type
  constructor, and otherwise from any defaulted constraint. The REPL now shows the expression's
  type when it is ground (`[Int]`, `{a: Int, b: String}`, `int -> int`).

Tests: the matrix through the REPL and through the wasm playground's setup; `VerifyANF` against
the elaborator's output for every module under `examples/` and against a non-atomic operand in
each operand position; the session, `:reset`, registry and type-display cases; and the 360-case
generated shadowing corpus run through the module registry against its renamed-apart twin. `ailang
run`, `ailang check` and the file pipeline are unchanged: 2,985 programs (the tracked corpus and
2,000 banked solutions) give the same check verdict and the same output on the evaluator and
`--strict-bytecode` before and after.

Not fixed (pre-existing, filed separately): in the REPL, string interpolation and cons (`::`)
fail, because its runtime is a hand-kept subset of the builtins.
