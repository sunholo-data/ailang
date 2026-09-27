# M-NUMERICS-VEC-ARRAY-INGEST: Unboxed float arrays, bulk update, binary numeric ingest

**Status**: IMPLEMENTED (2026-09-27, all four phases; see "Implementation Record"). Was: Planned, **ratified by Mark 2026-09-27** (attended session): the quorum block after round 2 is overridden on the strength of the round-2 revisions, and all five Design Freeze items are decided as recommended (D1 = D, D2 = `std/array`, D3 = (i), D4 = both widths named, D5 = error). Sprint plan: `m-numerics-vec-array-ingest-sprint-plan.md`.
**Target**: v0.45.0 (Phase 0 can ship in a patch)
**Priority**: P2. The quick fixes already closed most of the reported gap (see "What already shipped").
**Estimated**: Phase 0: 0.5 day. Phase 1: 3–4 days. Phase 2: 2–3 days. Phase 3: 1–2 days.
**Dependencies**: M-NUMERICS-QUICK (landed with this doc). Interacts with the D-19 cons-cells
programme (`m-list-cons-cells-decomposition.md`); see Conflict Surface.
**Source**: email-parse dogfood, message `inbox_1790498369833_8034dce8` (2026-09-27), triage
`task-8034dce8`.
**Quorum trigger**: #1 fires (the doc has Design Freeze items), so the doc goes to quorum before planning.
**Revision (quorum round 1, 2026-09-27)**: both present reviewers blocked, and both were right.
(1) gpt6-astra: the risk table allowed shipping Phase 1 evaluator-only with a fallback,
contradicting the no-fallback success criterion. That mitigation is removed: VM support is part
of Phase 1's definition of done, and if it cannot be built, Phase 1 does not ship. (2) gemini-3-1-pro:
the doc rejected specialising `ListValue` (D-19 collision) but never considered specialising
`ArrayValue`, which does not collide with D-19 and needs no new public type. That is now option D
and the recommendation. The opaque `FloatVec` (B) is kept as the alternative.
**Revision (quorum round 2, 2026-09-27)**: both present reviewers blocked again, and both were right.
(1) gpt6-astra: the primary goal promised O(1) single-element updates, but the recommended D3 (i)
cannot deliver that, since `updateMany` is O(n + k). The goal now states what each D3 option buys,
and the O(1) metric is attached to options (ii)/(iii) only. (2) gemini-3-1-pro: the doc deferred the
`ArrayValue` silent-fallback audit to implementation. The audit is done (V13–V17): `set` is the
only silent path, and `unsafeGet` fails with a Go `panic` rather than a typed error. Both are now
in Phase 0. Round 3 is not run (the process allows one re-quorum). Status is left for a human.

## What already shipped (M-NUMERICS-QUICK, same change as this doc)

The report asked for (a) `range`, (b) a vector type with dot/axpy/scale, (c) in-place arrays and
(d) binary ingest. Part of (a) and (b) needed no new language surface and landed first:

- `std/list.range(start, end)`, from a native `_list_range`.
- `std/embedding.dot`/`scale`/`add_vectors`/`euclidean_distance` on native loops, and a new strict
  `axpy`.
- `std/json.filterNumbers`/`allNumbers`/`filterStrings`/`allStrings` made linear (they were
  quadratic).
- The teaching prompt now names `range` and `std/embedding`. The stdlib reference no longer
  describes `std/embedding` as a host-model call.

That changes the premise of the remaining work. Measured with the new stdlib:

| Workload | Before | After quick fixes |
|---|---:|---:|
| One SGD step, 5 classes × 768 dims (reporter's shape) | ~84 ms (reported); 24.5 ms (`zipWith` update, measured, V9) | **~0.1 ms** (`dot` + `axpy`, V9) |
| 1,000 × 768-dim `dot` | 4.74 s | 0.04 s (V8) |
| `Json` → `[float]`, 1,252 × 768 | 4.0 s / 1.1 GB | 0.76 s (0.31 s is decode) / ~1 GB (V10) |

**So this doc is no longer about speed.** The remaining costs are memory (about 262 B per element
for a `[float]`, V7, against 8 B for a float64), updates that copy the whole array (V1), and
getting numbers in without a `Json` tree (V10). Each phase below is gated on a measured need,
not on the original 500× number.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Kernels are sequential loops with a fixed summation order; no parallel reductions (Non-Goal) |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | All proposed ops are pure. D3 option (ii) would add an effect and is not recommended |
| A4: Explicit Authority | 0 | Ingest decodes `bytes` the caller already holds; FS stays with `std/fs` |
| A5: Bounded Verification | 0 | The float kernels are opaque to the SMT encoder, the same as non-trivial `[float]` ops today |
| A6: Safe Concurrency | 0 | Immutable values, no shared mutation |
| A7: Machines First | +1 | One obvious fast way to do vector math; the prompt names `foldl(zipWith(...))` as wrong |
| A8: Minimal Syntax | +1 | No syntax and, under D, no new type: kernels on the existing `Array[a]` |
| A9: Cost Visibility | +1 | Documented O(n) per op and 8 B per element, no hidden quadratic helpers |
| A10: Composability | 0 | Explicit `toList`/`fromList` at the boundary |
| A11: Structured Failure | +1 | Length mismatch and bad bytes are errors; fixes the silent out-of-bounds `set` (V2) |
| A12: System Boundary | +1 | A typed binary boundary with the lossy width in the name, not JSON-only |

**Net: +6.** No −1 on A1, A3, A4 or A7.

## Problem Statement

**Current state after the quick fixes. Every row is measured or read in the Verification Log.**

- **Memory.** `[float]` is a list of boxed `*eval.FloatValue`. A 10M-element list peaks at 2.5 GB
  (~262 B per element including the source range and GC headroom, V7). The reporter's 1,252 × 768
  matrix is ~1 M floats, which is fine. Ten times that is not.
- **Updates copy.** `std/array.set` copies the whole array (V1). The canonical "update weight i"
  loop is O(n²), exactly what the module's docstring recommends it for ("performance-critical
  code").
- **Silent failure.** `ArrayValue.Set` returns the array **unchanged** on an out-of-bounds index
  (V2). A training loop with an off-by-one computes on stale weights and reports nothing. That
  violates CLAUDE.md principle 2, and it is the one item here that is a defect, not a feature gap.
- **Ingest.** The only numeric boundary is JSON, which materialises a `Json` tree (decode alone:
  0.31 s / 285 MB for 9.5 MB, V10). Packed float32 encode/decode builtins already exist
  (`_embedding_encode`/`_embedding_decode`, V4), but **they are not exported**. User code cannot
  call `_`-prefixed builtins, and `std/embedding` only uses them inside `cosine_encoded`.
- **Naming.** General float-vector math lives in a module called `embedding`. The reporter checked
  `std/list`, `std/iter`, `std/array` and `std/math` and missed it, and the reference page
  described it as a host-model call (now fixed).

**Impact.** It affects small-scale numerics inside AILANG programs: a quick classifier, a
similarity pass, feature scaling. The reporter's own conclusion frames this doc: AILANG is the
typed, effect-checked shell, and bulk numerics belong in the engine (DuckDB). The goal is to make
the *small* case pleasant, not to compete with BLAS.

## Goals

**Primary goal:** a program can hold and load ~10M floats at 8–16 B per element, and update them
without an O(n) copy per element, without leaving pure AILANG.

What "update" means depends on D3, and the metric is chosen to match:

| D3 option | Per-update cost | Metric |
|---|---|---|
| (i) bulk-only (recommended) | O(n + k) per batch of k writes | Applying 100k writes to a 3,840-element array **as one batch** (`updateMany`/`scatterAdd`) takes < 50 ms. Sequential, dependent single updates stay O(n) each; (i) does not solve that, and says so |
| (ii) effect-scoped mutable buffer | O(1) | 100k sequential single-element writes < 50 ms |
| (iii) uniqueness-based reuse | O(1) when unique, O(n) otherwise | same as (ii), when the array is uniquely referenced |

The reporter's workload (an SGD step) is a whole-vector update, which `axpy` already covers
(~0.1 ms per step, V9). The recommendation is (i) because no real program yet needs dependent
single-element updates. Choosing (i) means accepting that limit on purpose.

**Success metrics:**
- A 10M-element float array peaks at < 200 MB (from ~2.5 GB).
- The D3 metric in the table above, for the option chosen.
- Loading 1,252 × 768 floats from binary takes < 50 ms and < 50 MB above baseline.
- The quick-fix SGD benchmark (~0.1 ms per step) does not regress.
- Zero silent out-of-bounds writes.

## High-Impact Decisions

| # | Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|---|
| D1 | **Representation**: (A) keep `[float]` plus kernels (done); (B) new opaque `FloatVec` value backed by `[]float64`; (C) specialise `ListValue` internally for homogeneous floats; (D) **specialise `ArrayValue`**: an `Array[float]` is backed by an unboxed `[]float64`, invisibly | B adds a public type. C edits the list representation the D-19 programme owns. D reuses the existing `Array[a]` type and `std/array` surface, and needs the evaluator to pick the backing store at construction | human | design | high |
| D2 | **Where vector math lives**: under D, float kernels on `Array[float]` in `std/array` (`dot`, `axpy`, ...); under B, a new `std/vec`. `std/embedding` keeps its `[float]` API either way | Public API; moving it later breaks importers | human | design | med |
| D3 | **Updates**: (i) bulk-only API (`generate`, `updateMany(v, [(i, x)])`, `scatterAdd`), no mutation; (ii) an effect-scoped mutable buffer; (iii) invisible reuse when a value is uniquely referenced (FBIP-style) | (ii) is a new effect (A3/A4 surface). (iii) needs uniqueness information the Go runtime does not track | human | design | high |
| D4 | **Binary format**: float32 LE (exists, lossy for float64), float64 LE, or both; raw or with a header | A lossy default would silently change values at the boundary | human | design | med |
| D5 | **`std/array.set` out of bounds**: error (recommended) or keep the no-op | Behaviour change for existing programs | human | design | low |
| D6 | Phase 1 kernel list (dot, axpy, scale, add, sub, mul, sum, max, argmax, norm) | Surface size | agent | compile | low |

### Design Freeze

- [x] D1 representation. Recommendation: **D**. There is no new public type, `std/array` finally
  fits its "performance-critical" charter, and it doesn't collide with D-19. B only if D's runtime
  backing-store switch proves unsound (see Conflict Surface, item 3).
- [x] D2 module. Recommendation (with D): **float kernels in `std/array`** on `Array[float]`;
  `std/embedding` keeps its `[float]` API.
- [x] D3 update model. Recommendation: **(i)**. Open (ii) or (iii) only if a real program needs
  element-wise mutation that bulk ops can't express.
- [x] D4 format. Recommendation: **both, named by width** (`decodeF64LE`, `decodeF32LE`), so the
  lossy one says it's lossy.
- [x] D5. Recommendation: **error**.

## Solution Design

### Phase 0: defect and exposure (no new type, 0.5 day)

1. `ArrayValue.Set` / `_array_set`: an out-of-bounds index becomes an error naming the index and
   the length (D5). This is the only silent path in `std/array` (audit, V13–V17).
2. `_array_unsafe_get`: out of bounds returns a typed runtime error instead of a Go `panic` (V16).
   It stays a loud failure; it just becomes an ordinary AILANG error instead of a crash.
3. Export the float codec from `std/embedding`: `encodeF32LE`/`decodeF32LE` over the existing
   builtins, plus F64 siblings. This gives `[float]` a binary boundary before any new type exists.
4. `std/array` header: stop recommending it for performance-critical code that does repeated
   `set`; point to bulk ops.

### Phase 1: unboxed `Array[float]` (if D1 = D)

- `eval.ArrayValue` gains an unboxed backing store: `Floats []float64`, used when every element is a
  float. `Elements []Value` stays for everything else. Exactly one of the two is populated.
- **Who picks the store:** the constructors, `make(n, v)` (v is a float) and `fromList(xs)` (every
  element is a `FloatValue`). A type-correct program can never put a non-float into an
  `Array[float]`, so the store never has to change after construction. An empty array uses the
  boxed store; `fromList([])` has nothing to pack.
- `get`/`set`/`length`/`toList`/`show`/`==` read either store. `==` on floats is IEEE (#1274).
- New `std/array` kernels, typed on `Array[float]`: `dot`, `axpy`, `scale`, `add`, `sub`, `mul`,
  `sum`, `argmax` (D6). A length mismatch is an error (the `axpy` rule). They also accept a boxed
  float array, reading it element by element.
- **VM: required, not optional.** `bytecode.Value` and `internal/runner/bridge.go` have no Array at
  all today (V11), so `std/array` programs under `--bytecode` already fall back. Phase 1 adds
  `TagArray` with a float-unboxed variant to the bytecode values and the bridge. It is part of the
  definition of done: a Phase 1 that only works on the evaluator does not ship.
- **Go codegen:** `[]float64` for `Array[float]`.

**Option B (if D is rejected):** the same kernels on a new opaque `FloatVec` type in `std/vec`,
with the same VM requirement.

### Phase 2: updates (if D3 = i)

- `std/array.updateMany(arr, [(int, a)]) -> Array[a]`: one copy, k writes, any element type.
- `std/array.scatterAdd(arr: Array[float], idx: [int], xs: [float])`, the gradient-accumulate shape.
- If a program still needs single-element updates in a loop after this, that is the evidence
  for D3 (ii) or (iii), in a follow-up doc.

### Phase 3: ingest (D4)

- `std/array.decodeF64LE(bytes) -> Result[Array[float], string]` and `decodeF32LE`, plus encoders,
  producing the unboxed store directly. A bad byte length is an `Err` naming the count.
- `std/json.decodeFloatArray(s) -> Result[Array[float], string]`: a native parse of a flat JSON number
  array that never builds a `Json` tree. This fits the reporter's actual data (JSON exported from
  DuckDB).
- Matrices are `[Array[float]]` (rows). No 2-D type here.

### Conflict Surface (touches `internal/eval`, `internal/vm`, `internal/types`)

1. **Positions extended:** `eval.ArrayValue`'s representation (a second backing store), every
   `_array_*` builtin that reads `Elements`, `bytecode.Value` (new Array tag), the bridge, `show`/`==`
   on arrays, Go codegen for `Array[float]`.
2. **Existing constructs in those positions:** every `ArrayValue` consumer (`grep -rn
   "ArrayValue" internal/`). Each must read both stores; a missed one silently sees an empty
   `Elements`. **Mitigation:** make `Elements` private behind accessor methods in the same change,
   so the compiler finds every consumer instead of grep. `ListValue` and D-19 are untouched, which
   is why D beats C.
3. **Is the store choice sound?** It relies on `Array[float]` never receiving a non-float element
   after construction. The type system guarantees that for typed programs. Untyped paths that build
   an `ArrayValue` directly (builtins, JSON decode, the bridge) must go through the same
   constructor. The accessor refactor in item 2 enforces this. If any path can't, fall back to B,
   where the type is nominal and nothing is shared.
4. **Programs that must still work:** `examples/runnable/stdlib_embedding.ail`, every example
   importing `std/array` (`grep -l "std/array" examples/`), `cmd/ailang/stdlib_numeric_test.go`,
   `cmd/ailang/eq_parity_test.go`, `cmd/ailang/stdlib_list_depth_test.go`,
   `examples/runnable/recursion_quicksort.ail`, all on both backends.
5. **Intentional incompatibility:** D5 only. Under D, no type changes; under B, none either (nominal type).

## Examples

```ailang
-- valid today, after M-NUMERICS-QUICK, and stays valid:
import std/embedding (dot, axpy)
func step(w: [float], x: [float], lr: float, y: float) -> [float] =
  axpy(0.0 - lr * (dot(w, x) - y), x, w)
```

```ailang
-- Phase 1 + 3, option D (valid since M-NUMERICS-VEC-ARRAY; see examples/runnable/array_float_kernels.ail):
import std/array (make, decodeF32LE, dot, axpy)  -- Array is a builtin type, not an export
```

## Success Criteria

- [x] Phase 0: an out-of-bounds `set` is an error (test); the F32/F64 codecs round-trip (test).
- [x] Phase 1: every Goals metric measured with `/usr/bin/time -l` and recorded; evaluator and VM run
  `Array[float]` programs with **no fallback** (the no-fallback check from `stdlib_numeric_test.go`).
  This is a ship gate: no evaluator-only release.
- [x] The quick-fix SGD benchmark (~0.1 ms per step) does not regress.
- [x] Prompt: `std/array` in the canonical import block, with the float kernels and the lossy F32 codec marked as lossy.
- [x] `make verify-stdlib` re-frozen; changelog; `docs/docs/reference/stdlib.md` row.

## Implementation Record (2026-09-27)

Sprint plan: `m-numerics-vec-array-ingest-sprint-plan.md`, milestones M1–M7, one commit each.

### Goals, measured (`/usr/bin/time -l`, Mac Studio, evaluator and `--bytecode`)

| Metric | Target | Evaluator | VM |
|---|---|---:|---:|
| 10M-element `Array[float]` peak RSS (`make(10000000, 1.5)`) | < 200 MB (was ~2.5 GB as `[float]`) | 137 MB | 132 MB |
| 100k writes to a 3,840-element array in one batch (D3 i) | < 50 ms | `updateMany` ~1 ms, `scatterAdd` < 1 ms | same |
| 1,252 × 768 float64 from a binary file | < 50 ms, < 50 MB above baseline | 11 ms end to end (read 5–7, base64 2–3, decode ≤ 2); +43 MB | same, no fallback |
| Same data as a 19.5 MB JSON file (`decodeFloatArray`) | none (was ~4 s via the `Json` tree) | 50 ms | 49 ms |
| SGD, 1,000 steps × 768 dims | ~0.1 ms/step, no regression | `[float]` 20 ms, `Array[float]` 9 ms | 19 ms / 10 ms |
| Silent out-of-bounds writes | 0 | `set`/`updateMany`/`scatterAdd` all error | same |

### What differed from the design

1. **The VM needed no native `_array_*` builtins.** Builtins missing from the VM table make the
   calling std function `EvalOnly`, and it runs through `EvalInterop`. So the ship gate came down to
   the bridge: `bytecode.TagArray`, whose packed variant shares the `[]float64` with the evaluator
   (an O(1) crossing), plus `show`/`==` in the VM. A boxed array still converts element by element.
2. **Two more bridge gaps closed.** Ingest was still falling back because `bytes` could not cross
   (the VM had no bytes value) and a `Result` from an evaluator call could not either. `TagBytes` now
   crosses both ways. `Option`/`Result` from `std/option`/`std/result` cross from the evaluator to
   the VM with fixed declaration-order ordinals (`bytecode.StdADTTag`, pinned to the `.ail`
   sources by a test). ADTs going from the VM to the evaluator are still M-BYTECODE-2E scope.
3. **No `Eq` on arrays.** `==` on `Array[float]` is a type error ("No instance for Eq[Array[float]]"),
   so parity is tested through `toList`. The Go-level equality reads both stores anyway.
4. **The `std/array` kernels are strict; the `std/embedding` ones stay lenient.** A length mismatch
   is an error on `Array[float]`. The `[float]` versions keep their common-prefix rule.
5. **`std/fs.readFileBytes` returns base64 text, not `bytes`.** A binary load is therefore
   read → `fromBase64` → `decodeF64LE`, and the base64 string is a transient 1.33× copy. It still
   fits the budget. A direct bytes reader would be a separate `std/fs` change.
6. The doc's proposed import `import std/array (Array, ...)` was wrong: `Array` is a builtin type
   and not an export (`IMP010`). Fixed above.

### Tests and mutation checks

- `cmd/ailang/stdlib_array_test.go` (out-of-bounds `set` on both backends; the `[float]` codecs,
  VM-native), `stdlib_array_packed_test.go` (11 parity rows), `stdlib_array_kernels_test.go`
  (16 kernel rows + 7 error rows), `stdlib_array_ingest_test.go` (12 rows). Every table asserts the VM
  ran it natively.
- `internal/eval/value_array_test.go`, `internal/builtins/array_float_test.go` (`-count=20`),
  `internal/bytecode/std_adt_tag_test.go`.
- Mutations: reverting D5 fails the out-of-bounds test; making `Get` read only the boxed store fails
  the packed parity test; removing the Array case from the bridge fails all 11 VM parity rows; removing
  the bytes case fails the 4 ingest rows that use bytes.

## Testing Strategy

- Unit tests per kernel: semantics, no input mutation, length-mismatch error, NaN behaviour.
- End-to-end memory checks as ratios at two sizes, not absolute numbers, on both backends.
- Mutation checks: remove the bridge tag and watch the VM rows fail; make one `_array_*` builtin read
  only `Elements` and watch the unboxed rows fail; revert D5 and watch the out-of-bounds test fail.

## Deferred Decisions

- Agent: the exact D6 kernel list, `show` truncation length, error wording.
- Agent: whether `std/embedding` delegates to the `Array[float]` kernels internally (invisible to callers).

## Non-Goals

- BLAS/SIMD, parallel reductions (a fixed summation order keeps results bit-reproducible, A1), GPU.
- Matrix types, autodiff, an ML library.
- Arrow/Parquet readers. Columnar data stays in DuckDB (`sunholo/duckdb`); Phase 3 covers the
  "export a float column, load it" boundary.
- Changing `ListValue`'s representation (D-19's job).

## Timeline

Phase 0: 0.5 day. Phase 1: 3–4 days. Phase 2: 2–3 days. Phase 3: 1–2 days. Each phase ships
separately, behind its measurement gate.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Two ways to hold floats (`[float]`, `Array[float]`), and models pick the wrong one | One prompt rule: `[float]` + `std/embedding` for small vectors, `Array[float]` when memory or updates matter; explicit `toList`/`fromList` |
| VM Array support balloons | It is a ship gate, not a fallback: Phase 1 does not ship without it. If it overruns badly, the fallback is to stop at Phase 0 and re-plan, not an evaluator-only release |
| A consumer reads `Elements` and misses the unboxed store | Accessor refactor makes it a compile error (Conflict Surface, item 2) |
| The float32 codec silently loses precision | D4: the width is in the name, and F64 is offered alongside |
| Merge conflicts with D-19 in `eval/value.go` | D touches `ArrayValue` only, not `ListValue`; coordinate merge order with the cons-cells owner anyway |

## Verification Log

| # | Claim | How checked | Result |
|---|---|---|---|
| V1 | `std/array.set` copies the whole array | `internal/eval/value.go:130-139` `ArrayValue.Set`: `make` + `copy` | Confirmed |
| V2 | Out-of-bounds `set` is silent | same lines: `return a // Out of bounds, return unchanged` | Confirmed (defect) |
| V3 | No mutable or in-place API in std | `grep -niE "inplace\|in-place\|mutable" std/*.ail`: every hit states immutability (array.ail:19,70,73,80; list.ail:4; map.ail:5) | Confirmed none |
| V4 | A float32 codec exists but is not exported | `internal/builtins/embedding.go:45,82,94,109` (float32 LE); `grep '^export' std/embedding.ail` shows no encode/decode; `_`-prefixed builtins fail in user code with `undefined variable` (observed 2026-09-26) | Confirmed |
| V5 | No `FloatVec`/`Vec` type or `std/vec` module | `grep -rnwE "FloatVec\|Vec" internal std` empty; no `std/vec.ail` | Confirmed free |
| V6 | `decodeFloatArray`/`decodeF32` names unused | grep over `std/` and `internal/` | Confirmed free |
| V7 | `[float]` peaks at ~262 B per element at 10M | `length(map(\i. 1.5, range(0, 10000000)))`, `/usr/bin/time -l`: 2,503 MB | Measured 2026-09-27 |
| V8 | Quick-fix `dot`: 4.74 s → 0.04 s for 1,000 × 768 | the same program on the old and new stdlib | Measured 2026-09-27 |
| V9 | SGD step ~0.1 ms with `dot` + `axpy`; 24.5 s for 1,000 steps with a `zipWith` update | 1,000 steps, 5 × 768; both converge to 5.0 | Measured 2026-09-27 |
| V10 | JSON decode 0.31 s / 285 MB for 9.5 MB; conversion was the other 3.7 s | decode-only vs decode+convert runs | Measured 2026-09-27 |
| V11 | VM bridge carries Int/Float/Bool/Unit/String/List/Tuple/Record, rejects closures, has no Array | `internal/runner/bridge.go` value switch (~L120-175) | Confirmed |
| V12 | `ArrayValue` holds `Elements []Value` only (boxed), no typed store | `internal/eval/value.go:103-105` | Confirmed |
| V13 | Audit: `ArrayValue` has exactly four methods (`Type`, `String`, `Get`, `Set`) | `grep -n "func (a \*ArrayValue)" internal/eval/*.go` → value.go:107,108,122,130 | Confirmed |
| V14 | `Get` out of bounds is loud: returns `(nil, false)`, and `_array_get` turns it into `array_get: index N out of bounds` | value.go:122-127; `internal/builtins/array.go:181-182` | Confirmed, not silent |
| V15 | `getOpt` out of bounds returns `None` explicitly (documented) | `std/array.ail:62` | Confirmed, by design |
| V16 | `unsafeGet` out of bounds calls Go `panic` | `internal/builtins/array.go:236-238` | Confirmed: loud, but a crash, not a typed error (Phase 0 item 2) |
| V17 | `make` with negative size errors; `append` has no index | `internal/builtins/array.go:114-115`, :478-484 | Confirmed, not silent |

## Related Documents

- `design_docs/planned/m-list-cons-cells-decomposition.md`: the D-19 list representation
  programme. Option C would collide with it.
- `design_docs/planned/m-list-cons-quadratic.md`: superseded context for `::` costs (#676).
- Issues #1318 (per-element recursion in std/list, fixed) and #676 (cons quadratic, open).

## References

- email-parse dogfood report, 2026-09-27 (`inbox_1790498369833_8034dce8`).

## Future Work

- D3 (ii)/(iii) if Phase 2's bulk ops prove insufficient on a real program.
- A `Matrix` type only if a second real workload needs one.
