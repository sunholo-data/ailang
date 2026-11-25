### Added — verify: findIndex and the std/json accessors are checked

- **`findIndex` with a literal or named predicate is specialised like `any`** (M-VERIFY-SOUNDNESS-COVERAGE). The chain computes the index of the first match (-1 when there is none), and the call site rebuilds the `Option[int]`. Beyond the unrolling depth the index is unknown but still -1 or a position in the list, so a contract like `i >= 0 && i < length(xs)` for a found `i` is proved for lists of any length.
- **`std/json`'s `get`, `getString`, `getObject`, … now reach the solver.** They find a key through the private `findInList`, which calls `findIndex` and the raw string equality builtin `_str_eq`, now encoded as `=`.
- Downstream corpora: 2 skips now verify (`agui` `eventJson`, `decodeRuns`). Five more get past `findIndex` and now stop at blockers already listed in the coverage table (list patterns, a list in a callee signature, a literal nested inside a pattern).

### Fixed — verify: a helper inside an imported body is collected from its own module

- **A same-module call inside an imported function's body was looked up in the caller's module**, so `std/json.get`'s call to its private `findInList` was never collected as a callee. The caller was skipped with `it was not collected as a callee of the function being verified`. The collector now looks a plain call up in the module whose body it is walking. The existing identity gate still refuses any call whose bare name would bind to a different module's function.
