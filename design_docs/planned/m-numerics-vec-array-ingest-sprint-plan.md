# Sprint plan: M-NUMERICS-VEC-ARRAY-INGEST

**Design doc**: `design_docs/planned/m-numerics-vec-array-ingest.md` (ratified by Mark 2026-09-27,
all five Design Freeze items as recommended: D1 = D, D2 = `std/array`, D3 = (i), D4 = both widths, D5 = error)
**Target**: v0.45.0 (M1 can ship in a patch release)
**Estimated**: ~1,500 LOC including tests, 7 milestones, ~6 working days. The doc says 7–10 days
across its phases; the saving comes from the VM finding below.
**Risk**: medium, concentrated in M2 (every `ArrayValue` consumer) and M4 (VM).

## What the code survey changed (2026-09-27)

- **The VM does not need native `_array_*` builtins.** Builtins missing from the VM's
  `BuiltinTable` lower to `OpBuiltinTrap`, so the std function that calls them is compiled
  `EvalOnly` and runs on the evaluator through `EvalInterop`
  (`internal/vm/interop.go`). The arguments and results cross `internal/runner/bridge.go`.
  Array programs fall back today only because the bridge has no Array case (doc V11). The VM work
  is therefore: a `TagArray` value, the bridge in both directions, and `show`/`==` on it in the VM.
- **Crossing cost.** For an unboxed `Array[float]`, the bridge can share the `[]float64` slice,
  because both sides treat it as immutable. That makes a crossing O(1). A boxed array is converted
  element by element, which is O(n) per crossing, the same as lists today. It is documented, not fixed here.
- **Go codegen already maps `Array[T]` to `[]T`** (`internal/gen/lower/typeres.go:145`), so
  `Array[float]` is already `[]float64`. New builtins need a codegen entry, or they are rejected at
  compile time in the usual way.
- `ArrayValue.Elements` has 9 consumer files (`grep -rln ArrayValue internal/`). That is small
  enough to make the field private in M2, and the compiler then finds every consumer.

## Milestones

### M1: Phase 0, the defect and the codec export (~150 LOC, 0.5 day)
- `ArrayValue.Set` returns `(*ArrayValue, bool)`; `_array_set` out of bounds returns an error
  naming the index and the length (D5). Update the metadata LongDesc.
- `_array_unsafe_get` out of bounds returns an error instead of calling `panic`.
- `std/embedding`: export `encodeF32LE`/`decodeF32LE` (over `_embedding_encode`/`_embedding_decode`),
  plus new `encodeF64LE`/`decodeF64LE` builtins. A bad byte length is an error.
- `std/array` header: stop recommending repeated `set` for hot loops; point to bulk ops.
- **Tests**: an out-of-bounds `set` fails (evaluator); the F32 codec round-trips 1.5 exactly and
  0.1 lossily; the F64 codec round-trips exactly; a bad length is rejected.
- **Acceptance**: `go test ./internal/builtins/ ./internal/eval/ ./cmd/ailang/ -run 'Array|Codec|Numeric'`
  green; `make verify-stdlib` re-frozen.

### M2: Phase 1a, the unboxed store on the evaluator (~350 LOC, 1 day)
- `eval.ArrayValue{elems []Value; floats []float64}`, with constructors `NewArray(elems)` (packs
  when every element is a `*FloatValue` and the slice is non-empty) and `NewFloatArray([]float64)`.
  Accessors: `Len`, `Get`, `Set`, `Elements()` (materialises boxed values on demand), `Floats()`
  (returns `(slice, ok)`).
- Every consumer moves to the accessors (the compiler lists them): builtins/array, list, show,
  canonical_key, eval show_bounded, eval_expressions (array literals), embed/convert, observatory.
- `==`/`show` are identical for both stores. `show` of a packed float uses `FloatValue`'s formatting.
- **Tests**: packing is chosen by `make`/`fromList`/literals; `set` of a float keeps it packed;
  `show`/`==`/`toList` agree across both stores; empty arrays stay boxed.
- **Mutation check**: make `Get` read only `elems`, and watch the packed rows fail.

### M3: Phase 1b, float kernels in `std/array` (~300 LOC, 1 day)
- Builtins `_array_f_dot/axpy/scale/add/sub/mul/sum/argmax`, typed on `Array[float]`, exported
  from `std/array` as `dot`, `axpy`, `scale`, `add`, `sub`, `mul`, `sum`, `argmax` (D6). A length
  mismatch is an error; `argmax` of an empty array is an error. They read a boxed float array too.
- The names collide with nothing in `std/array`. Callers importing both `std/embedding` and
  `std/array` alias one, which the prompt says.
- **Tests**: semantics, no input mutation, NaN behaviour, errors.

### M4: Phase 1c, VM parity as a ship gate (~300 LOC, 1 day)
- `bytecode.TagArray` + `ArrayObj{Elems []Value; Floats []float64}`, with `show` and `==` in the VM.
- Bridge: packed arrays share the slice in both directions; boxed arrays convert per element.
- **Tests**: a `TestStdArray…` table on both backends with the no-fallback assertion from
  `stdlib_numeric_test.go`. Mutation check: remove the bridge case and watch the VM rows fail.
- **Measurements** (`/usr/bin/time -l`, recorded in the design doc): 10M-element `Array[float]`
  peak RSS < 200 MB; the SGD benchmark unchanged at ~0.1 ms per step.

### M5: Phase 2, bulk updates (~150 LOC, 0.5 day)
- `updateMany(arr, [(int, a)]) -> Array[a]`, one copy and k writes. An out-of-bounds index is an error.
- `scatterAdd(arr: Array[float], idx: [int], xs: [float]) -> Array[float]`. A length mismatch or
  out-of-bounds index is an error.
- **Metric**: 100k writes to a 3,840-element array in one batch, < 50 ms.

### M6: Phase 3, ingest (~250 LOC, 1 day)
- `std/array.decodeF64LE/decodeF32LE(bytes) -> Result[Array[float], string]` and
  `encodeF64LE/encodeF32LE(Array[float]) -> bytes`, building the packed store directly.
- `std/json.decodeFloatArray(s) -> Result[Array[float], string]`: a native parse of a flat JSON
  number array with no `Json` tree.
- **Metric**: 1,252 × 768 floats loaded from binary in < 50 ms and < 50 MB above baseline.
- VM: `Result` is an ADT, and ADTs don't cross the bridge yet (M-BYTECODE-2E scope). Check whether
  the std wrapper stays VM-native. If it cannot, record that as the one known fallback, with evidence.

### M7: Docs, prompt, example, release notes (~100 LOC, 0.5 day)
- `examples/runnable/array_float_kernels.ail` (verified on both backends); the teaching prompt
  rule "`[float]` + `std/embedding` for small vectors, `Array[float]` when memory or updates
  matter", with F32 marked lossy; a `docs/docs/reference/stdlib.md` row; CHANGELOG; the design
  doc moved to `design_docs/implemented/v0_45_0/` with the measurements filled in.

## Dependencies

M1 → M2 → {M3, M5} → M4 (the VM test tables cover M3 and M5 as well) → M6 → M7.

## Workflow

Work happens in the worktree `worktree-m-numerics-vec-array`, with one commit per milestone and a PR to `dev`.
