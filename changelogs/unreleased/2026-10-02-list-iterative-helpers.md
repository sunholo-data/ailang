### Fixed — std/list helpers no longer recurse per element (#1518, #1501)

- `any`, `findIndex` and `foldr` now delegate to new iterative builtins (`_list_any`, `_list_findIndex`,
  `_list_foldr`), with native ports on the bytecode VM. Their `[x, ...rest]` walks copied the tail at every
  step on the VM, so `any` over 320,000 elements took 117 s there (#1518). `foldr` was not tail recursive and
  failed `RT_REC_003` past 10,000 elements. `any` and `findIndex` still stop at the first match.
- `maximumInt`, `minimumInt`, `maximumFloat`, `minimumFloat`, `maximumString` and `minimumString` were
  non-tail recursive. They failed `RT_REC_003` on the interpreter and `vm: stack overflow` on
  `--strict-bytecode` at 50,001 elements (reported by stapledons_godot). They are now a right fold that makes
  the same comparisons in the same order, so ties and NaN resolve exactly as before.
- `foldlE` and `forEachE` walk the list by index in a tail loop (O(n), was O(n²) on the VM). `mapE`,
  `filterE` and `flatMapE` split the index range in halves: recursion depth is O(log n) and the total copy
  is O(n log n). Before, they failed `RT_REC_003` past 10,000 elements. Effects still run left to right, once
  per element.
- The same audit covered other stdlib modules. `std/json.get`, `std/sem`'s key lookup and `std/jwt`'s
  audience check now use `findIndex`/`any`. `std/fs.walk` lists each directory with `flatMapE`, so a
  directory with more than 10,000 entries no longer fails.
- Timings at 320,000 elements are interpreter / `--bytecode` / `--strict-bytecode`:

  | helper | before | after |
  |---|---|---|
  | `any` | 0.90 / 117 / 126 s | 0.50 / 0.08 / 0.07 s |
  | `findIndex` | 1.6 / 123 / 126 s | 0.77 / 0.08 / 0.08 s |
  | `maximumInt`, `foldr` | RT_REC_003 / RT_REC_003 / stack overflow | 0.5–0.7 / 0.09 / 0.09 s |
  | `foldlE` | 0.98 / 110 / 105 s | 2.1 / 0.18 / 0.12 s |
  | `forEachE` | 0.58 / 103 / 102 s | 1.3 / 0.10 / 0.16 s |
  | `mapE`, `filterE`, `flatMapE` | RT_REC_003 / RT_REC_003 / stack overflow | 4.7–5.3 / 0.3–0.4 / 0.3 s |

  The index loops make `foldlE`/`forEachE` about 2x slower on the interpreter, which matched `[x, ...rest]`
  without copying. They are about 600x faster on the VM.

### Added — `std/list.mapAccumL`: build a list from running state in one pass (#1501)

- `mapAccumL(f, s0, xs)` with `f: (state, elem) -> (output, newState)` returns `(outputs, finalState)`. The
  output list is one allocation. It is the linear replacement for a `foldl` that conses onto a list
  accumulator. Native on both engines. Example: `examples/runnable/mapAccumL_running_total.ail`.

### Changed — list-building cost guidance tells the truth (#1501)

- The std/list header, the `concat` and `foldl` comments, the no-loops guide and the `RT_REC_003` message
  used to recommend foldl and prepend-then-reverse as the O(n) way to build lists. They now say that foldl is
  O(n) only if the step is O(1). A step that conses onto or concatenates to a list accumulator is O(n) per
  element, which makes the fold O(n²). To build a list, use map/filter/flatMap or `mapAccumL`.
- Deferred: a new teaching-prompt version with the same caveat (prompt-manager lane), and the VM
  `CallClosure` frame-allocation investigation (design doc Phase 3). Design:
  [M-FOLDL-CONS-COST-MODEL](../design_docs/implemented/v0_51_1/m-foldl-cons-cost-model.md).
