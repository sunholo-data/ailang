# M-FLOAT-ORD-ONE-SEMANTICS — one answer to ordered float comparisons with NaN, on every path

**Status**: Planned
**Target**: v0.49.1
**Priority**: P1 — correctness bug: the same comparison answers differently depending on backend and lowering path, violating A1 parity
**Estimated**: ~1 day (parity-test first, then a one-site semantics swap)
**Dependencies**: None. Follows [M-FLOAT-EQ-ONE-SEMANTICS](../../implemented/v0_43_2/m-float-eq-one-semantics.md) (#1274), which fixed the same class of divergence for `==` and explicitly listed "Float `Ord` with NaN" as a Non-Goal
**Tracking**: found by Stapledon mission iteration 5 (designer role, WD photometry design); reproduced first-party by the controller (see V1)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same expression, same answer regardless of backend or lowering path — removes an existing A1 violation |
| A2: Replayability | 0 | No trace/replay change |
| A3: Effect Legibility | 0 | Pure semantics; no effects touched |
| A4: Explicit Authority | 0 | No capability surface |
| A5: Bounded Verification | 0 | Verification unchanged (SMT residual documented, not worsened) |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Matches every model prior and reference language; the position-dependent trap (generic IEEE, direct total-order) is exactly what AI-generated code cannot defend against |
| A8: Minimal Syntax | +1 | No syntax change at all; net code deletion (`compareFloat` and redundant guards) |
| A9: Cost Visibility | 0 | No cost surface |
| A10: Composability | +1 | Refactoring a comparison into a helper or generic no longer changes its answer |
| A11: Structured Failure | 0 | No error-path change |
| A12: System Boundary | 0 | No boundary crossing |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): fixes an existing nondeterminism-across-backends; introduces none
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): aligns with machine/model priors, not human convenience

## Problem Statement

AILANG has **one** float `Ord` implementation on a total order (NaN greatest) and **six** on IEEE 754, and they disagree on NaN. Which one runs is decided by backend and lowering, not by the program.

Verified live 2026-09-30 (`n = 0.0 / 0.0`, interpreter vs `--bytecode` VM, full transcripts in V1):

| Expression | evaluator | `--bytecode` VM | IEEE 754 |
|---|---|---|---|
| `n < 1.0` | false | false | false — agree, **by coincidence** (see V2) |
| `n > 1.0` | **true** | false | false ✗ |
| `n >= 1.0` | **true** | false | false ✗ |
| `n <= 1.0` | false | false | false — agree, by coincidence |
| `n == n` | false | false | false — agree (fixed by #1274) |
| `n != n` | true | true | true — agree (the NaN test) |
| `gt2(n, 1.0)` — monomorphic helper `func gt2(p: float, q: float) -> bool = p > q` | **true** | false | false ✗ |
| `(\p. \q. p > q)(n)(1.0)` — lambda | **true** | false | false ✗ |
| `above[a](n, 1.0)` — generic `func above[a](x: a, hi: a) -> bool = x > hi` | false | false | false — agree, **via a different mechanism** (V4) |
| range guard `if x > hi then A else B` with `x = n` (monomorphic) | takes **A** | takes B | B ✗ |
| `maximumFloat([n, 1.0])` (std/list) | **NaN** | 1.0 | — std-defined ✗ |
| `minimumFloat([n, 1.0])` (std/list) | 1.0 | 1.0 | agree |
| `sortBy(cmp, [2.0, n, 1.0])` with `cmp` built from `<` | `[1.0, 2.0, NaN]` | `[2.0, NaN, 1.0]` | — comparator answers differ ✗ |

The evaluator disagrees **with itself**: direct `n > 1.0` is true (total-order dictionary, V2/V3) but the same comparison through a generic function is false (op-lowering shim, V4). The VM disagrees with the evaluator on every non-generic row. `<`/`<=` agree today only because the total order (NaN greatest ⇒ `n < x` false) and IEEE (all false) happen to coincide there — the prior doc's V9 observed this coincidence for `<` and correctly did not extend it to `>`/`>=`.

**Impact.**
- **Determinism (A1) and parity.** `ailang run` and `ailang run --bytecode` are documented as executing the same program; a pure function guarding a range with `x > hi` takes a different branch per backend for NaN input. The REPL uses the same registry-backed evaluator, so it inherits the total order.
- **AI-generated code.** Every model prior (Python, JS, Go, Rust, Haskell `Ord Double`) and every benchmark reference program is IEEE: a model writing `x > hi` as a NaN-safe guard is wrong only on the interpreter, and only sometimes (generic helpers are already IEEE there — the trap is position-dependent, the worst kind for synthesis).
- **Mission-visible.** Stapledon's sunholo/relativity 0.3.0 design works around it with an explicit `c == c` NaN test before range guards. The workaround stays valid after this fix (`!=`/`==` semantics are unchanged, #1274), but the guard itself becomes backend-independent.

**Root cause (single site).** `internal/types/dictionaries.go` `registerOrdFloat` (v: `compareFloat` at lines 297–315) defines floats' `Ord` instance on a total order — `-Inf < finite < +Inf < NaN`, `NaN == NaN` for the order — and derives `lt/lte/gt/gte` (and `min`/`max`) from it. The run pipeline injects this registry into the evaluator (`internal/pipeline/pipeline_module_phases.go:100` → `internal/runner/run.go:586` `SetDictionaryRegistry`), and monomorphic float comparisons elaborate to `DictApp(DictRef(Ord, float), "gt", …)` (`internal/elaborate/dictionaries.go:91–108`), which `evalDictRef` resolves from the registry (`internal/eval/eval_patterns.go:301–380`). Every other implementation site is IEEE (V4–V8).

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The divergence reproduces first-party | Live runs of the issue's `nan.ail` and of `blast.ail`/`lambda.ail`/`generic.ail`/`zeros.ail` (written for this doc) on this machine's binary: interpreter `false true true false false`, VM `false false false false false`; helper/lambda `true`/`true` vs `false`/`false`; generic `false` vs `false`; extremum and sortBy rows as in the table above | Confirmed (transcripts above; every row in the problem table comes from these runs) |
| V2 | The `Ord[Float]` dictionary is a total order with NaN greatest | Read `internal/types/dictionaries.go` `registerOrdFloat` (lines 289–355): `compareFloat` returns `+1` when `x` is NaN ("NaN is greatest"), `0` for NaN/NaN; `gt`/`gte` are `compareFloat > 0` / `>= 0` ⇒ `NaN > x` true, `NaN >= x` true | Confirmed — this is the interpreter's `true`/`true` |
| V3 | Interpreter monomorphic comparisons route through the registry | Read `internal/elaborate/dictionaries.go:68–108` (resolved BinOp → `DictApp(DictRef(Ord, float), method)`), `internal/eval/eval_patterns.go:301–380` (`evalDictRef` looks up `prelude::Ord::float::<method>` in `e.registry`), `internal/pipeline/pipeline_module_phases.go:100–101,429` and `internal/runner/run.go:585–586` (pipeline `NewDictionaryRegistry` → `SetDictionaryRegistry`) | Confirmed — mechanism read end-to-end, not inferred from output |
| V4 | Generic comparisons are IEEE on the evaluator, via op lowering — not the registry | Live: `above[a](n, 1.0)` → false on both. Read `internal/eval/eval_operations.go:287–345` (`evalIntrinsic` maps `core.OpGt` → `applyBinOp`) and `applyBinOp` float branch (native `<`/`>`/`<=`/`>=` + `types.FloatEq` for eq) — native Go float ops are IEEE | Confirmed. Same position-dependence the #1274 doc found for `==` (its V2/table row 3) |
| V5 | The VM is IEEE on every comparison shape | Read `internal/vm/vm.go:719` `compare` (float case: native `lhs.Flt < rhs.Flt` / `<=`) and `internal/bytecode/compiler/expr.go:257–283` `compileCmp` (`Gt → OpLt` with operand swap, `Gte → OpLe` with swap; comment documents the lowering) | Confirmed — NaN is false on all four ops because both operands pass through native comparison |
| V6 | Go codegen is IEEE | Read `internal/gen/golang/codegen_dictionaries.go:127–141` `generateOrdDictionary`: `Lt/Gt/Lte/Gte` emit native Go `<` `>` `<=` `>=` on `float64` | Confirmed. (Its `Eq`/`Neq` `reflect.DeepEqual` residual is already in docs/LIMITATIONS.md and is out of scope here) |
| V7 | Both builtin registries are IEEE | Read `internal/eval/builtins_comparison.go:66–130` (`lt/le/gt/ge_Float` with explicit `math.IsNaN → false` guards) and `internal/builtins/math_comparison.go:32–38` (native ops) | Confirmed |
| V8 | The typed evaluator is IEEE | Read `internal/eval/eval_typed.go:687–742` (`evalLess/evalGreater/evalLessEq/evalGreaterEq` use native Go operators) | Confirmed |
| V9 | `registerOrdFloat`'s `min`/`max` are unreachable from surface syntax today | `grep -rn '"min"|"max"' internal/elaborate/*.go` → empty (no DictApp ever carries method min/max); `grep -rn '_Ord_' internal/vm/ internal/eval/builtins.go` → empty (no VM builtin); `grep -rn 'maximumBy|minimumBy|.min|.max' std/*.ail examples/*.ail` → empty. `internal/gen/lower/expr.go:666–681` lowers only `Ord` `lt/lte/gt/gte` to binops; `evalDictRef`'s `methodNames` for `Ord` does include `min`/`max` (`eval_patterns.go:321`), so the closures must stay registered or every `Ord` DictRef fails with "missing dictionary method" | Confirmed — dead-but-load-bearing; fix must keep them registered (D2) |
| V10 | No existing test or example pins the total-order NaN behavior | `grep -rn 'NaN|nan' internal/types/dictionary_test.go` → key-format tests only; `grep -rln NaN --include='*_test.go' internal/ cmd/` → 10 files, all about `show`/`sqrt`/dedup/eq-parity (read the hits); `examples/float_nan.ail` (read in full) pins only `==`/`!=`/`isNaN`/set-ops; `tests/binops_float.ail` (read) has no NaN rows | Confirmed — no deliberate test-text changes beyond new rows |
| V11 | `sortBy` is stable on both engines, so post-fix orders match | Read `internal/vm/builtins_hof.go:213–214` ("Stable, matching the evaluator's `_list_sortBy` (#1318)") and `std/list.ail:61–69` (sortBy delegates to `_list_sortBy`); today's sortBy divergence comes from comparator answers, not the algorithm | Confirmed by reading; a parity row will pin it |
| V12 | `_array_f_argmax`'s NaN-loses policy is deliberate and separate | Read `internal/builtins/array_float.go:154–162` (`if math.IsNaN(a[best]) && !math.IsNaN(x) || x > a[best]`) — a documented builtin policy, not the `Ord` dictionary | Confirmed — deliberately unchanged |
| V13 | `maximumFloat`/`minimumFloat` are hand-written comparisons, not dict methods | Read `std/list.ail:283–305`: `if x > m then Some(x) else Some(m)` / `if x < m …` — their post-fix behavior equals the VM's today (else-branch keeps the running extreme) | Confirmed |
| V14 | Signed-zero ordering and `Inf` vs `NaN` already agree on both engines | Live `zeros.ail` (5 concatenated `show` results, in order): `0.0 < -0.0`, `-0.0 < 0.0`, `-0.0 <= 0.0`, `0.0 >= -0.0`, `+Inf > NaN` → `false`, `false`, `true`, `true`, `false` on **both** backends | Confirmed (non-goal) |
| V15 | Contract verification cannot see NaN ordered comparisons | Read `docs/LIMITATIONS.md` row (SMT `Real` has no NaN, #1274 residual, added 2026-09-25) | Confirmed — stays a documented residual; guard with `std/math.isNaN` |
| V16 | This machine's binary predates the reported versions | `ailang version` → v0.47.1 while repo `std/VERSION` is v0.49.0 (transcripts carry the mismatch warning); the issue reports v0.47.2 and v0.49.0 affected | Confirmed — divergence live-reproduced here on v0.47.1; fix targets v0.49.1 sources |
| V17 | No other engine consumes the registry | Read `internal/runner/batch.go:88–89` (same `SetDictionaryRegistry` injection); `grep -rn 'DictionaryRegistry' internal/vm/` → empty (VM never touches the registry — it lowers dict methods to binops per V5); `grep -rn 'SetDictionaryRegistry|NewDictionaryRegistry' cmd/ailang/` → empty (no CLI path wires the registry anywhere else) | Confirmed — `registerOrdFloat` is interpreter/REPL-only, so the fix cannot move VM behavior |

## Goals

**Primary Goal:** one NaN rule for `<` `<=` `>` `>=` on floats — IEEE 754, every comparison false when either operand is NaN — identical on every lowering path and both backends.

**Success Metrics:**
- Every divergent row of the problem table gives the same answer on the evaluator and the VM, and the direct/generic rows are consistent with each other
- The accidental `<`/`<=` agreement becomes principled agreement (same mechanism, not coincidence)
- A parity test pins all four operators × both operand orders × {direct, helper, lambda, generic} plus extremum and sortBy rows, with no documented-divergence rows
- Mutation: reverting `registerOrdFloat` alone must fail the parity test

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1: IEEE everywhere (all NaN ordered comparisons false) vs total order everywhere** | Observable semantics of every float comparison, both backends, std extremum results | **human** — same values question as #1274's D1, where IEEE was ruled for `==` (Mark, 2026-09-25); this doc defaults to that precedent and needs only ratification | design | med |
| D2: Registry `min`/`max` float semantics (unreachable today, V9) → NaN-propagating (`math.Min`/`math.Max`), replacing the current half-total-order (`max(nan,x)=nan` but `min(nan,x)=x`) | Last consumer of `compareFloat`; if left, a future surface exposing them re-imports the divergence | agent | sprint | low |
| D3: Named authorities `types.FloatLt/Lte/Gt/Gte` (mirror `types.FloatEq`) vs inline native ops at the registry | One grep-able rule for docs/simplicity vs minimal added surface | agent | sprint | low |
| D4: Where the fix's parity rows live (new `ord_parity_test.go` vs extending `eq_parity_test.go`) | Test architecture only | agent | sprint | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1 — IEEE or total order.** Default in this doc: **IEEE everywhere**, per the #1274 precedent (VM, Go codegen, both builtin registries, the typed evaluator and the op-lowering shim are already IEEE — option A flips exactly one site; option B flips five engines against every reference language and against the working Stapledon workaround pattern). Sprint-executor pauses if the approver does not confirm.

**D1 options:**
- **(A) IEEE everywhere (recommended).** All four ordered comparisons are false when either operand is NaN. Matches the VM, Go codegen, both builtin registries, the typed evaluator, the deferred shim, Python/JS/Go/Rust/Haskell, and every benchmark reference program. Cost: `Ord[Float]` stops being a lawful total order around NaN (transitivity fails), exactly as Haskell accepts an unlawful `Ord Double` for this reason; NaN sorts as "equal to everything" under user comparators built from `<` (stable sort keeps that deterministic and engine-identical, V11).
- **(B) Total order everywhere (NaN greatest).** Flip the VM, Go codegen, builtins and shim to `registerOrdFloat`'s rule. Cost: five engines change against the model prior (`x > hi` no longer guards NaN out — the exact bug report's complaint inverts), `maximumFloat([nan, 1.0])` becomes NaN on both, and the `c == c` / `c != c` NaN idiom keeps working but ordered guards keep surprising. Rejected by asymmetry: one site vs five, and the one site is the interpreter-only outlier.

## Solution Design

### Overview

Make IEEE the one rule for float ordered comparisons, named in one place, and route the lone outlier through it. The fix is deliberately narrow: the interpreter's `Ord[Float]` dictionary swaps its four comparison closures (and its two extremum closures) from the total-order `compareFloat` to IEEE; nothing else changes semantics, because every other site is already IEEE (V4–V8).

### Architecture

**Components:**
1. **Named authority (mirrors `types.FloatEq`):** `internal/types/float_ord.go` defines `FloatLt/FloatLte/FloatGt/FloatGte(a, b float64) bool` with the doc-comment trail (which sites use them, the #1274 lineage, why IEEE). `internal/types` is already imported by `eval` and `vm` (established by the prior doc's V10) — no new import edge.
2. **The dictionary:** `registerOrdFloat`'s `lt/lte/gt/gte` become the four named functions; `min`/`max` become NaN-propagating `math.Min`/`math.Max` (D2); `compareFloat` and its total-order comments are deleted. `Eq[Float]` (`FloatEq`) is untouched.
3. **Routing the already-IEEE sites through the names (recommended, D3):** `internal/eval/builtins_comparison.go` `lt/le/gt/ge_Float` drop their redundant `IsNaN` guards and call the named functions; `internal/builtins/math_comparison.go` likewise; `internal/vm/vm.go` `compare`'s float case calls `types.FloatLt/FloatLte` (mirroring `runtimeEq`'s `types.FloatEq` call). Go codegen stays as-is (native operators, V6) — its generated dictionaries are self-contained by design.
4. **Parity test:** `cmd/ailang/ord_parity_test.go` (or rows appended to `eq_parity_test.go`, D4), modeled exactly on `TestEqEvaluatorVMParity` (pure program, `--relax-modules`, both backends, "a future divergence should be fixed, not recorded").

### Implementation Plan

**Phase 1: Parity test, red first (~1.5h)**
- [ ] Write the parity test with every problem-table row (4 ops × NaN; both operand orders `n > x` / `x > n`; direct, helper, lambda, generic; `not (n <= x)`; `maxf`-style guard branch choice; `maximumFloat`/`minimumFloat` with NaN in both list positions; `sortBy` with NaN; non-NaN regression rows: ordinary floats, ints, strings, `0.0 < -0.0`/`>=` signed zeros)
- [ ] Record the current divergent rows as the pre-fix baseline (they must fail on the evaluator side before the fix)

**Phase 2: The one-site semantics swap (~1.5h)**
- [ ] `internal/types/float_ord.go`: the four named functions + doc trail
- [ ] `internal/types/dictionaries.go` `registerOrdFloat`: IEEE closures, `math.Min`/`math.Max` for min/max, delete `compareFloat` and its comments; update the `registerEqFloat`-style comment to cite this doc
- [ ] Unit test in `internal/types` (e.g. `float_ord_test.go`): NaN false on all four, both orders; `min`/`max` NaN-propagate; ordinary values unchanged

**Phase 3: Route remaining sites, docs, examples (~1.5h)**
- [ ] `internal/eval/builtins_comparison.go` + `internal/builtins/math_comparison.go`: drop redundant guards, call the named functions
- [ ] `internal/vm/vm.go` `compare`: float case through `types.FloatLt/Lte`
- [ ] Extend `examples/float_nan.ail` with ordered-comparison checks (`nan < 1.0` … `nan > 1.0` all false, both engines; guarded-range example) — `TestFloatNaNExample` (`cmd/ailang/eq_containers_example_test.go:71`) keeps pinning it
- [ ] Teaching-prompt row (verified with `ailang check`): ordered float comparisons are IEEE; test for NaN with `std/math.isNaN` or `x != x`
- [ ] CHANGELOG entry in `changelogs/v0.32-current.md` naming the behavior change (interpreter `NaN > x`/`>=` flips true → false; `maximumFloat` with NaN-leading lists changes on the interpreter)
- [ ] `docs/LIMITATIONS.md`: extend the SMT-Real residual's wording from "== is IEEE" to also cover ordered comparisons (V15)

### Files to Modify/Create

**New files:**
- `internal/types/float_ord.go` — named IEEE authorities (~25 LOC)
- `internal/types/float_ord_test.go` — unit rows (~40 LOC)
- `cmd/ailang/ord_parity_test.go` — evaluator/VM parity (~110 LOC)

**Modified files:**
- `internal/types/dictionaries.go` — `registerOrdFloat` rewrite, `compareFloat` deleted (~40 LOC touched, net −15)
- `internal/eval/builtins_comparison.go` — 4 closures simplify (~12 LOC)
- `internal/builtins/math_comparison.go` — 4 registrations (~6 LOC)
- `internal/vm/vm.go` — `compare` float case (~4 LOC)
- `examples/float_nan.ail` — ordered-comparison checks (~10 LOC)
- `changelogs/v0.32-current.md`, `docs/LIMITATIONS.md`, teaching prompt — doc rows

## Examples

### Example 1: the repro (issue's `nan.ail`)

**Before (divergent):**
```
$ ailang run --quiet --caps IO --entry main nan.ail        → false true true false false   (interpreter, WRONG)
$ ailang run --quiet --bytecode --caps IO --entry main nan.ail → false false false false false (VM, IEEE)
```

**After (identical, IEEE):**
```
$ ailang run --quiet --caps IO --entry main nan.ail        → false false false false false
$ ailang run --quiet --bytecode --caps IO --entry main nan.ail → false false false false false
```

### Example 2: a range guard stops being backend-dependent

```ail
-- before: takes the "high" branch on the interpreter, the "low" branch on the VM
func clampHigh(x: float, hi: float) -> float = if x > hi then hi else x
clampHigh(nan, 1.0)   -- interpreter: 1.0 (NaN > 1.0 was true → clamps!)
                     -- VM:          NaN (guard false, NaN flows through)
-- after: NaN on BOTH engines — the guard behaves like every reference language
```

## Success Criteria

- [ ] Every parity row agrees between evaluator and VM; direct and generic rows agree with each other (no documented-divergence rows, ever)
- [ ] Mutation run: reverting `registerOrdFloat` (only) fails the parity test; reverting the `min`/`max` closures fails the unit test
- [ ] `make test-core`, `make check-boundaries`, `TestFloatNaNExample`, `TestEqEvaluatorVMParity`, `stdlib_list_depth_test.go` (sortBy stability row) all green
- [ ] `examples/float_nan.ail` extended and pinned; teaching prompt updated with an `ailang check`-verified row
- [ ] CHANGELOG names the interpreter-visible behavior change

## Conflict Surface

Touches `internal/types`, `internal/eval`, `internal/vm` — the mandatory enumeration.

### Syntactic positions touched

None. No grammar production, AST node, or typechecker surface changes. The change is entirely in the **runtime semantics of the float case** of the four ordered comparisons on existing paths: the dictionary closures in `registerOrdFloat` (`internal/types/dictionaries.go:318–355`), the builtin twins (`internal/eval/builtins_comparison.go`, `internal/builtins/math_comparison.go`), and the VM's float `compare` case (`internal/vm/vm.go:731–737`).

### What else lives here

| Position (float operands of `<` `<=` `>` `>=`) | Existing behavior | Effect of this change |
|---|---|---|
| `Eq[Float]` dictionary (`registerEqFloat`, `types.FloatEq`) | IEEE since #1274 | Untouched |
| NaN detection idioms `x != x`, `std/math.isNaN` | true for NaN on both engines | Untouched (and remains the only NaN test — ordered comparisons never detected NaN on the VM) |
| `Ord[Int]` / `Ord[String]` / `Ord[Bool]` dictionaries | native comparisons, no NaN concept | Untouched |
| Constant-pool dedup (`bytecode.Value.Equal`, NaN==NaN "for dedup", D3 of #1274) | dedup semantics | Untouched — `Ord` closures never feed it |
| `_array_f_argmax` NaN-loses policy (`internal/builtins/array_float.go:159`) | deliberate builtin semantics (V12) | Untouched — separate code path |
| `show(nan)` / JSON encoding of NaN | "NaN" formatting | Untouched |
| `sortBy` with a user comparator built from `<` | comparator returns 0 vs NaN on the VM today; −1/0 on the interpreter (total order) | Comparator answers become IEEE-identical on both engines; both `_list_sortBy` implementations are stable and matching (#1318, V11), so the resulting order is engine-identical and pinned by a parity row |
| std/list `maximumFloat`/`minimumFloat` (hand-written `>`/`<`, V13) | interpreter NaN-leading lists differ from VM | Interpreter converges on the VM's existing behavior (running extreme kept when comparison is false) |
| Signed zeros / ±Inf operands | already agreeing on both engines (V14) | Untouched |

### Disambiguation strategy

n/a — no parse or type-disambiguation change. Dictionary dispatch itself is unchanged: `DictApp(DictRef(Ord, float), "gt", …)` still resolves through `evalDictRef`; only the closure body it resolves *to* changes. The elaborator (`internal/elaborate/dictionaries.go:68–108`) emits the same nodes.

### Programs that MUST still work (regression fixtures)

- `examples/float_nan.ail` — pinned by `cmd/ailang/eq_containers_example_test.go:71` `TestFloatNaNExample` (extended, not weakened, by this doc)
- `cmd/ailang/eq_parity_test.go` `TestEqEvaluatorVMParity` — all rows, including the NaN `==`/`!=` rows (Eq untouched)
- `tests/binops_float.ail` — no NaN rows (V10), all ordinary comparisons
- `cmd/ailang/stdlib_list_depth_test.go` — sortBy stability row (no NaN)
- `examples/eq_containers.ail` (26 checks, no NaN)

### Deliberately changes (intentional incompatibilities)

- Interpreter: `nan > x` and `nan >= x` flip **true → false** (the bug); `nan < x`, `nan <= x` stay false but for the IEEE reason instead of the total-order reason
- Interpreter: monomorphic range guards and helper/lambda comparisons match the VM; `maximumFloat`/`minimumFloat` with NaN-leading lists converge on the VM's answer
- Interpreter: registry `min`/`max` float closures become NaN-propagating (unreachable from surface syntax today, V9)
- Under IEEE, `Ord[Float]` is not a lawful total order around NaN (accepted, as in Haskell) — documented in Non-Goals, not "fixed"

## Testing Strategy

**Unit tests:**
- `internal/types/float_ord_test.go`: all four operators × {NaN left, NaN right, NaN both, ordinary, ±Inf, signed zeros}; `min`/`max` NaN-propagation
- Registry-level: `registerOrdFloat` closures answer IEEE (the mutation anchor for the parity test)

**Integration tests:**
- `cmd/ailang/ord_parity_test.go`: both backends on the full problem table (Phase 1), including `maximumFloat`/`minimumFloat`/`sortBy` rows — these prove the std-visible blast radius is pinned, not just the operators

**Manual testing:**
- Re-run the issue's exact `nan.ail` on both backends; diff against the table

## Deferred Decisions

The following are intentionally left open for the implementer:

- D3 exact form — named `types.FloatLt/…` functions (recommended, `FloatEq` symmetry) vs inline native operators with a comment; either is acceptable, but the rule must be stated in one comment trail if inlined
- D4 — new `ord_parity_test.go` file vs appending rows to `eq_parity_test.go`; the executor may choose by file-size taste
- Whether `internal/eval/eval_typed.go` (V8) also names the rule — it is IEEE already and unreached by the repro; touching it is optional

## Quorum

Optional per the skill. D1 is the same values question #1274's D1 settled for `==` (IEEE, ruled by Mark, 2026-09-25), and every in-repo premise is verified in the log above — a reviewer can refute a premise, but D1 is a ratification of an existing ruling, not a new call. The pre-sprint `ailang design-quorum` step remains available to the approver; D1 is the only item a reviewer could plausibly block on, and it is flagged for human confirmation in Design Freeze.

## Non-Goals

**Not attempted in this feature:**
- Float `Ord` **laws** — under IEEE, `Ord[Float]` is not a lawful total order around NaN; accepted deliberately (as Haskell does), not "fixed"
- SMT contract verification of NaN comparisons — verifier model gap (`float` as `Real`, docs/LIMITATIONS.md); unchanged
- Go-codegen `Eq` `reflect.DeepEqual` residual — already documented in LIMITATIONS, out of scope (V6)
- `_array_f_argmax` NaN-loses policy — deliberate builtin semantics, unchanged (V12)
- Signed-zero / Inf edge ordering — already agreeing on both engines (V14)
- Unifying VM/evaluator record/tuple comparison beyond floats (pinned elsewhere, agrees today)

## Timeline

**Day 1 (≈4.5h):**
- Phase 1 — parity test, record red rows (morning)
- Phase 2 — the one-site swap + unit tests (midday)
- Phase 3 — routing, examples, prompt, CHANGELOG, LIMITATIONS (afternoon)
- Full `make test-core`, mutation runs, commit

**Total: ~1 day** (2× the raw estimate of 4.5h with review/buffer ⇒ plan half a week of calendar)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| A program relies on interpreter `NaN > hi` clamping NaN (e.g., using `x > hi` to filter NaN) | Low | No test, example, or std module does (V10); CHANGELOG names it; the `x != x`/`isNaN` idiom keeps working on both engines |
| `min`/`max` registry closures change behavior (D2) | Very low | Unreachable from surface syntax today (V9); the closures stay registered (removal would break every `Ord` DictRef) |
| Hidden consumer of `compareFloat` ordering | Low | It is a local closure used only by `registerOrdFloat` (V2 grep); compiler enforces with the deleted symbol |
| Parity test flakiness from stdlib version mismatch (V16) | Low | Test uses repo-pinned std via the same harness as `eq_parity_test.go` |

## Related Documents

<!-- Auto-populated by the create-script search on "float ord one semantics"; its top matches (compile-error/codegen-bool-slice/anthropic-sandbox, ≤1.00 SimHash) are keyword noise — neural embeddings were not running in this environment. The genuinely related docs below were found by targeted reading of the code and design_docs/; the duplicate gate passes: nothing queued or shipped covers float Ord NaN. -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_43_2/m-float-eq-one-semantics.md](../../implemented/v0_43_2/m-float-eq-one-semantics.md) — the direct precedent: same divergence class for `==`, D1 ruled IEEE (Mark, 2026-09-25), `types.FloatEq` as the single named rule; its Non-Goals explicitly deferred float `Ord` with NaN to this doc
- [design_docs/implemented/v0_42_0/m-eq-derive-containers.md](../../implemented/v0_42_0/m-eq-derive-containers.md) — the sprint that established evaluator/VM parity testing (`cmd/ailang/eq_parity_test.go`)
- [docs/LIMITATIONS.md](../../../docs/LIMITATIONS.md) — residuals this doc extends (SMT `Real`, Go-codegen Eq)

**Planned (check for overlap):**
- (none — no planned doc touches float comparison semantics; auto-search top matches are unrelated, scores are SimHash keyword noise)

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- IEEE 754-2019 — NaN unorderedness: every comparison except `!=` is false when an operand is NaN
- Issue report (this task): interpreter `false true true false false` vs VM `false false false false false`, Stapledon iteration 5, workaround `c == c` NaN test in sunholo/relativity 0.3.0
- `cmd/ailang/eq_parity_test.go` — the parity-test pattern this doc reuses

## Future Work

- Surface an `Ord.compare` for floats only with an explicit `totalOrder` escape hatch (IEEE 754 totalOrder), should a determinism-critical consumer need a lawful order — out of scope until a real consumer exists
- Extend parity testing to `-0.0` in structured containers if the dedup story ever revisits canonical keys

---

**Document created**: 2026-09-30
**Last updated**: 2026-09-30
