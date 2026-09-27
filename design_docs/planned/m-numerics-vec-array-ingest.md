# M-NUMERICS-VEC-ARRAY-INGEST: Native float vectors, in-place update, binary numeric ingest

**Status**: Planned. The Design Freeze items below need human decisions before any phase starts.
**Target**: v0.45.0 (Phase 0 can ship in a patch)
**Priority**: P2. The quick fixes already closed most of the reported gap (see "What already shipped").
**Estimated**: Phase 0: 0.5 day. Phase 1: 3–4 days. Phase 2: 2–3 days. Phase 3: 1–2 days.
**Dependencies**: M-NUMERICS-QUICK (landed with this doc). Interacts with the D-19 cons-cells
programme (`m-list-cons-cells-decomposition.md`); see Conflict Surface.
**Source**: email-parse dogfood, message `inbox_1790498369833_8034dce8` (2026-09-27), triage
`task-8034dce8`.
**Quorum trigger**: #1 fires (the doc has Design Freeze items), so the doc goes to quorum before planning.

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
| A5: Bounded Verification | 0 | FloatVec is opaque to the SMT encoder, the same as non-trivial `[float]` ops today |
| A6: Safe Concurrency | 0 | Immutable values, no shared mutation |
| A7: Machines First | +1 | One obvious fast way to do vector math; the prompt names `foldl(zipWith(...))` as wrong |
| A8: Minimal Syntax | +1 | No syntax: library functions and one opaque type |
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

**Primary goal:** a program can hold, update and load ~10M floats at 8–16 B per element with O(1)
per-element update cost, without leaving pure AILANG.

**Success metrics:**
- A 10M-element float vector peaks at < 200 MB (from ~2.5 GB).
- 100k single-element updates to a 3,840-element weight vector take < 50 ms (each is O(n) today).
- Loading 1,252 × 768 floats from binary takes < 50 ms and < 50 MB above baseline.
- The quick-fix SGD benchmark (~0.1 ms per step) does not regress.
- Zero silent out-of-bounds writes.

## High-Impact Decisions

| # | Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|---|
| D1 | **Representation**: (A) keep `[float]` plus kernels (done); (B) new opaque `FloatVec` value backed by `[]float64`; (C) specialise `ListValue` internally for homogeneous floats | B adds a public type. C edits the list representation the D-19 programme owns | human | design | high |
| D2 | **Where vector math lives**: `std/embedding` as is; a new `std/vec` with `std/embedding` as thin wrappers; or `std/list` | Public API; moving it later breaks importers | human | design | med |
| D3 | **Updates**: (i) bulk-only API (`generate`, `updateMany(v, [(i, x)])`, `scatterAdd`), no mutation; (ii) an effect-scoped mutable buffer; (iii) invisible reuse when a value is uniquely referenced (FBIP-style) | (ii) is a new effect (A3/A4 surface). (iii) needs uniqueness information the Go runtime does not track | human | design | high |
| D4 | **Binary format**: float32 LE (exists, lossy for float64), float64 LE, or both; raw or with a header | A lossy default would silently change values at the boundary | human | design | med |
| D5 | **`std/array.set` out of bounds**: error (recommended) or keep the no-op | Behaviour change for existing programs | human | design | low |
| D6 | Phase 1 kernel list (dot, axpy, scale, add, sub, mul, sum, max, argmax, norm) | Surface size | agent | compile | low |

### Design Freeze

- [ ] D1 representation. Recommendation: **B**. C collides with D-19, and A is where we are now.
- [ ] D2 module. Recommendation: **`std/vec`** for FloatVec; `std/embedding` keeps its `[float]` API.
- [ ] D3 update model. Recommendation: **(i)**. Open (ii) or (iii) only if a real program needs
  element-wise mutation that bulk ops can't express.
- [ ] D4 format. Recommendation: **both, named by width** (`decodeF64LE`, `decodeF32LE`), so the
  lossy one says it's lossy.
- [ ] D5. Recommendation: **error**.

## Solution Design

### Phase 0: defect and exposure (no new type, 0.5 day)

1. `ArrayValue.Set` / `_array_set`: an out-of-bounds index becomes an error naming the index and
   the length (D5). Audit the other `ArrayValue` methods for the same silent fallback.
2. Export the float codec from `std/embedding`: `encodeF32LE`/`decodeF32LE` over the existing
   builtins, plus F64 siblings. This gives `[float]` a binary boundary before any new type exists.
3. `std/array` header: stop recommending it for performance-critical code that does repeated
   `set`; point to bulk ops.

### Phase 1: `FloatVec` (if D1 = B, D2 = std/vec)

- `eval.FloatVecValue{Data []float64}`, immutable. Every op allocates one result slice: O(n),
  8 B per element, no per-element boxes.
- `std/vec`: `fromList`, `toList`, `length`, `get` (bounds-checked, returns `Option`), `zeros(n)`,
  `fill(n, x)`, `generate(n, \i. ...)`, plus the D6 kernels. A length mismatch is an error
  everywhere (the `axpy` rule).
- Type: an opaque builtin `TCon "FloatVec"`. `show` prints `FloatVec[n]` and the first k values.
  `==` is elementwise IEEE (NaN ≠ NaN, #1274).
- **VM:** add `bytecode.TagFloatVec` and teach `internal/runner/bridge.go` to carry it. The bridge
  today carries Int/Float/Bool/Unit/String/List/Tuple/Record and rejects closures (V11). Without
  this, every `--bytecode` program touching FloatVec falls back to the evaluator, the failure #1318
  hit with `sortBy`.
- **Go codegen:** `[]float64`.

### Phase 2: updates (if D3 = i)

- `std/vec.updateMany(v, [(int, float)]) -> FloatVec`: one copy, k writes.
- `std/vec.scatterAdd(v, idx: [int], xs: [float])`, the gradient-accumulate shape.
- The same `updateMany` for `std/array`.
- If a program still needs single-element updates in a loop after this, that is the evidence
  for D3 (ii) or (iii), in a follow-up doc.

### Phase 3: ingest (D4)

- `std/vec.decodeF64LE(bytes) -> Result[FloatVec, string]` and `decodeF32LE`, plus encoders. A bad
  byte length is an `Err` naming the count.
- `std/json.decodeFloatArray(s) -> Result[FloatVec, string]`: a native parse of a flat JSON number
  array that never builds a `Json` tree. This fits the reporter's actual data (JSON exported from
  DuckDB).
- Matrices are `[FloatVec]` (rows). No 2-D type here.

### Conflict Surface (touches `internal/eval`, `internal/vm`, `internal/types`)

1. **Positions extended:** a new `eval.Value` kind, `bytecode.Value` tags, the bridge, `show`/`==`,
   `std/json.encode` of a FloatVec (error or number array, decided in Phase 1), Go codegen types.
2. **Existing constructs in those positions:** `ListValue` (every list builtin type-switches on it),
   `ArrayValue`, and the D-19 cons-cells programme, which is changing `ListValue`. Option C would
   edit that struct and is rejected for this reason. Option B adds a sibling kind in a new file and
   leaves `ListValue` alone.
3. **Disambiguation:** FloatVec is nominal. Nothing converts implicitly to or from `[float]`, so no
   existing program changes type.
4. **Programs that must still work:** `examples/runnable/stdlib_embedding.ail`,
   `cmd/ailang/stdlib_numeric_test.go`, `cmd/ailang/eq_parity_test.go`,
   `cmd/ailang/stdlib_list_depth_test.go`, `examples/runnable/recursion_quicksort.ail` on both
   backends.
5. **Intentional incompatibility:** D5 only.

## Examples

```ailang
-- valid today, after M-NUMERICS-QUICK, and stays valid:
import std/embedding (dot, axpy)
func step(w: [float], x: [float], lr: float, y: float) -> [float] =
  axpy(0.0 - lr * (dot(w, x) - y), x, w)
```

```ailang
-- proposed (Phase 1 + 3), NOT valid yet:
import std/vec (FloatVec, decodeF32LE, dot, axpy, zeros)
```

## Success Criteria

- [ ] Phase 0: an out-of-bounds `set` is an error (test); the F32/F64 codecs round-trip (test).
- [ ] Phase 1: every Goals metric measured with `/usr/bin/time -l` and recorded; evaluator and VM run
  FloatVec programs with no fallback (the no-fallback check from `stdlib_numeric_test.go`).
- [ ] The quick-fix SGD benchmark (~0.1 ms per step) does not regress.
- [ ] Prompt names `std/vec` in the canonical import block, with the lossy F32 codec marked as lossy.
- [ ] `make verify-stdlib` re-frozen; changelog; `docs/docs/reference/stdlib.md` row.

## Testing Strategy

- Unit tests per kernel: semantics, no input mutation, length-mismatch error, NaN behaviour.
- End-to-end memory checks as ratios at two sizes, not absolute numbers, on both backends.
- Mutation checks: remove the bridge tag and watch the VM rows fail; revert D5 and watch the
  out-of-bounds test fail.

## Deferred Decisions

- Agent: the exact D6 kernel list, `show` truncation length, error wording.
- Agent: whether `std/embedding` delegates to `std/vec` internally (invisible to callers).

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
| FloatVec duplicates `[float]` APIs and models pick the wrong one | One prompt rule: `[float]` + `std/embedding` for small vectors, FloatVec when memory matters; explicit `toList`/`fromList` |
| VM bridge work balloons | If `TagFloatVec` exceeds ~1 day, ship Phase 1 evaluator-only with the fallback, and say so in the changelog |
| The float32 codec silently loses precision | D4: the width is in the name, and F64 is offered alongside |
| Merge conflicts with D-19 in `eval/value.go` | Option B puts the new type in its own file; coordinate merge order with the cons-cells owner |

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
