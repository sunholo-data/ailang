### Changed — BREAKING: test bodies and contracts are type-checked (#2232, #2248) (2026-11-12)

Two kinds of code were never type-checked, so `ailang check` passed programs with type errors in them:

- **Contracts (#2248).** A `requires`/`ensures` clause was inferred only to give op-lowering its types,
  and an inference error was printed only under `--debug-compile`. A clause containing `forall` always
  failed inference (the checker had no rule for the quantifier), so nothing inside it was checked:
  `requires { forall i: 0..2 => i + "x" }` passed `check`. Clauses are now checked like a body: a
  clause must be a `bool` over the parameters (and `result`, in `ensures` only), `forall i: lo..hi =>
  p` types with `int` bounds and a `bool` body, and `==` needs `Eq` exactly as in a body (an enum
  compared in an `ensures` needs `deriving (Eq)`). A failure is `TC_CONTRACT_001`, also when the
  contract is in an imported module.
- **Test bodies (#2232).** Named `test "…" { }` blocks, `property "…" { }` blocks, inline
  `properties [ … ]` and inline `tests [ … ]` rows are part of no compiled program: `ailang test`
  compiles them separately, and `check` never looked at them. The module being checked or run now
  elaborates each one in the form `ailang test` compiles it (the same `assert` fold, now
  `ast.FoldTestBody`) and type-checks it; test and property bodies are checked pure, a property or a
  test with no `assert` must end in a `bool` (the runner fails anything else), and a row
  `(inputs, expected)` must type as the call the harness makes (one shared construction,
  `ast.RowInput`/`RowCallArgs`: `((a, b), r)` and `(((a, b)), r)` both call `f(a, b)`) returning the
  expected value's type. A
  failure is `TC_TEST_001`. A dependency's test blocks and rows are not checked when it is imported.

Also:

- **`forall` outside a contract is `TC_FORALL_001`** instead of an uncoded "type inference not
  implemented for *core.Forall" error.
- **The missing-`Eq` message says what to change:** "Add `deriving (Eq)` to type X (declared at
  file:line)", or, for a type from the standard library or an installed package, that the package
  must add it. For `()` it no longer suggests `deriving (Eq)` (#2282).
- **`ailang test` reports a module that does not compile once, as itself,** instead of one "failed to
  extract function binding" per test, also when the error is in one of its imports. On that path
  `ailang test --json` has a single synthetic `tests[]` entry (`module <m> does not compile`, failed)
  and no `named_test_batch_failures`, so a tool that counts failures sees 1 where it used to see one
  per test.

All three codes are structured reports, so `check --json` shows them with a line (also in directory
mode).

**What newly fails** is code that was already wrong: a contract that uses a name it never imports
(`length`), `==` on a type without `deriving (Eq)`, a contract that is not a `bool`, a row whose
expected value has the wrong type, a test whose final expression is not a `bool`. A module with an
ill-typed contract fails `run`, `test` and its importers too, not only `check`; published packages
with such contracts are being republished.

**Intended: a module that does not pass `check` does not run.** A module's own ill-typed test block,
property or row fails `run`, `run --bytecode`, `compile --emit-go` and `serve-api` when that module is
the entry (root), although the same module loaded as a dependency compiles and runs (its test bodies
are not checked there). `serve-api` treats every module path on its command line as a root, so one
ill-typed test in any served module stops the server from starting.
