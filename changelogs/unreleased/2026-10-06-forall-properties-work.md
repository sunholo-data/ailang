### Fixed — `forall` properties never evaluated; they now run every case (#624) (2026-10-06)

Every `property "…" { forall(…) => … }` used to fail on its first generated case with `evaluation failed: empty program`, or with a parse error in a synthesized `_test.ail` when the predicate called a module function. The old path spliced each generated value into the source as a literal and compiled a module-less program: one compile per case, with no imports and no module scope.

How it works now:
- A forall property is one function of its binders, `pure func __namedtest_prop_<k>(n: int, …)`. It is compiled once, in the same per-file batch as the named tests, and every generated case is a call with the values as arguments.
- Measured: int, float, bool, string, list, tuple, record and same-module ADT binders all run their 100 cases. Predicates may call module functions.

Shrinking:
- Shrinking now repeats until no binder shrinks further, with a budget of 1,000 calls. `forall(n: int) => n < 5` reports `property failed on input: [5]`; the old single pass would have stopped at a value like `[100]`.
- The generated stream follows `--seed`; a test now pins this.

Failures:
- A property that does not type-check fails alone with `property does not compile: …`, pointing at `<file>:<line> (property)`, and the rest of the file still runs.

Docs:
- The testing guide's quick-start and property sections now use the real syntax (`property "x" { forall(x: int) => … }`, `test "x" { … }`) and list the binder types that have generators. The guide had documented a `property "x" (x: int) = …` form that does not parse.
