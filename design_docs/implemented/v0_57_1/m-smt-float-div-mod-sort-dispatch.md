# M-SMT-FLOAT-DIV-MOD-SORT-DISPATCH: encode Float `/` and `%` by operand sort in the SMT verifier, not as truncating Int division

**Status**: Implemented (2026-11-03, attended; ships in the release after v0.57.0)
**Target**: v0.57.1 (originally v0.56.2)
**Priority**: P0 (High) — blocks publishing `sunholo/relativity` (R2-KERR, ailang-packages PR #173) via PUB006
**Estimated**: 3 days (1 day dispatch core, 1 day mod/string slice + unit tests, 1 day e2e/regression/changelog)
**Dependencies**: none new; reuses M-SMT-WELL-SORTED-PREFLIGHT M0 plumbing (S1 truncating Int encoding, `EncodeFunctionOpts.CoreTI`, `verifyStatuses` e2e helper)
**Created**: 2026-11-02. Audit base: workspace commit `d555a52a` (std/VERSION v0.56.1), `ailang` v0.56.1 binary, Z3 4.8.12.

**Parent roadmap**: this doc executes the live-defect slice (rows A1 + A2) of M1 "Type-directed operator and literal resolution" in [m-smt-well-sorted-preflight.md](../../planned/m-smt-well-sorted-preflight.md) (planned, M0 merged in v0.56.0 as PR #2019). The parent's M2–M4 and F2/F5 stay deferred there; this doc does not duplicate them. Parent M1's `resolveOperator(method, operandType)` shape is adopted as specified, scoped here to the arithmetic-div/mod family plus the `DictApp` comparison methods that share the same sort-blind map.

## Implementation Notes (2026-11-03)

Landed encoder-local as designed (D5): `internal/smt/resolve_sort.go` (sort resolution, `divModOp`,
`orderOp`, `encodeRealMod`), dispatch in `codegen_operators.go`, `mod_Float` rerouted in `types.go`.
Tests: `internal/smt/float_divmod_test.go`, `cmd/ailang/verify_float_div_e2e_test.go`. Three
deviations from the text above, each measured:

1. **The D2 expansion as written was itself ill-sorted.** `(* $ail_d (ite … (to_int q) …))`
   multiplies a Real by an Int: the same bug class. The shipped form wraps the truncation in
   `to_real`.
2. **The single parallel `let` binding `$ail_q` read the OUTER `$ail_n`/`$ail_d`**, because SMT-LIB
   parallel-let right-hand sides cannot see their siblings. The shipped form binds `q` in an inner
   `let`. `TestFloatModEncoding_InternalBinderCollision` pins it with operands named after all
   three binders.
3. **D3 holds for `DictApp` (the production shape) but not for pre-lowered `Intrinsic`/`BinOp`.**
   When neither operand carries a sort, those keep the historical Int reading, which is what
   `int_divmod_test.go` pins with untyped operands. That test had to stay unchanged, so this
   deviation was needed to satisfy both requirements. Those nodes do not reach the encoder after
   op_lowering (`internal/pipeline/contract_pipeline_test.go`). Any Float or String evidence
   (literal kind or CoreTI) still wins.

`TestEncodeDictApp` built `DictApp`s with a nil dictionary, a shape elaboration never produces;
its Int cases now carry `DictRef{Ord, Int}`. `verifyStatuses` now also rejects lowercase
`sort mismatch`, the Z3 wording the `tidalTransverse` error used, which the e2e guard missed.

**Measured:** the reporter's file goes from 1 VIOLATION + 1 ERROR to 2 VERIFIED (Z3 4.16.0, which
reproduced the reporter's r = 7/4 counterexample). Float `%`, string `<`/`>=` and Int `/` all verify
too, and three false float/string contracts are still refuted. Across all 72 tracked contract
files (333 verdicts), the old and new binaries give byte-identical verdicts. Mutation check: all 7
new tests fail against the pre-fix encoder.

## Problem Statement

`ailang verify` encodes Float `/` as **truncating Int division** and Float `%` as **Int `mod`**, regardless of operand sort. The emitted SMT-LIB is ill-formed (Int-only operators on Real terms, Real-vs-Int `0` comparisons), and what Z3 does with it is version- and structure-dependent: either a hard parse error, or a silent `to_int`-based elaboration that **refutes contracts that are true at runtime**.

Repro (from the task report, re-confirmed live on this workspace, v0.56.1 + Z3 4.8.12):

```ailang
module t

export pure func tidalRadialOrbit(r: float) -> float
  requires { r > 1.5 }
  ensures { result > 0.0 }
{
  (1.0 / (2.0 * r * r * r)) * (2.0 + 3.0 / (2.0 * r - 3.0))
}

export pure func tidalTransverse(r: float) -> float
  requires { r > 0.0 }
  ensures { result > 0.0 }
{
  1.0 / (2.0 * r * r * r)
}
```

`ailang verify t.ail` on this workspace:

```
✗ VIOLATION tidalRadialOrbit
  Counterexample: ... $p_r: Real = (/ 1610612737.0 536870912.0)   -- ≈ 3.0000000019; reporter's Z3 gave r = 7/4 (true value +0.746)
! ERROR tidalTransverse
  Z3 error (exit 1): (error "line 5 column 262: invalid constant definition, sort mismatch")
  (error "line 8 column 16: unknown constant result")
2 exported functions: 1 violations, 1 errors
```

Both verdicts are wrong. `tidalRadialOrbit` and `tidalTransverse` are true contracts (float `/` is true division; every divisor is non-zero under `requires`), yet one is *refuted* with a bogus counterexample and the other *errors*.

**Generated SMT** (`verify --verbose`, verbatim, `tidalTransverse`):

```lisp
(define-const result Real (let (($tmp9 (* 2.0 $p_r))) (let (($tmp10 (* $tmp9 $p_r))) (let (($tmp11 (* $tmp10 $p_r))) (let (($ail_n 1.0) ($ail_d $tmp11)) (ite (= (>= $ail_n 0) (>= $ail_d 0)) (div (abs $ail_n) (abs $ail_d)) (- (div (abs $ail_n) (abs $ail_d)))))))))
```

That is `encodeTruncDivMod` (`internal/smt/codegen_operators.go:86`) — the Go-truncating **Int** encoding shipped in M-SMT-WELL-SORTED-PREFLIGHT M0 (#1960) — applied to **Real** operands:

1. `(>= $ail_n 0)` compares a Real against the Int numeral `0` — sort mismatch.
2. `(div (abs $ail_n) (abs $ail_d))` applies the SMT-LIB Int-only `div` to Real terms.

**Impact:**

- **PUB006 blocks publishing.** `internal/pkg/quality.go:294` gates `pkg publish` on `Verify.Counterexample > 0` → "N contract(s) refuted by Z3". The bogus `tidalRadialOrbit` refutation is a hard publish blocker for sunholo/relativity (R2-KERR, PR #173); the ERROR verdicts are publish-quality noise for every float-division contract in the corpus.
- **The verifier reports false refutations on true contracts** — worse than noise: an AI consumer of `verify --json` (or a human reading `✗ VIOLATION`) is told the contract is false when it is true.
- **Solver-version-dependent behavior.** Feeding the two structurally-similar scripts above to Z3 4.8.12 directly: the `tidalTransverse` script hard-errors (sort mismatch → `unknown constant result`), while the `tidalRadialOrbit` script is *silently elaborated* into `to_real (div (to_int …) (to_int …))` (visible in the model) and returns `sat` with a bogus counterexample at r ≈ 3.000000002. The same ill-formed text yields either failure mode depending on surrounding structure. v0.54.18 and the reporter's Z3 hit the sort-mismatch error on both; a future Z3 could do anything.
- **Sibling defects in the same sort-blind map (live-verified today):** Float `%` (`(mod 7.5 $p_x)` → sort mismatch; `BuiltinToSMTOp["mod_Float"] = "mod"` at `internal/smt/types.go:272`) and String ordering in contracts (`DictApp` method `lt` on `Ord String` → `(declare-fun < (Int Int) Bool) supplied sort is String`).

## Root Cause

The M0 fix (#1960, PR #2019) made `encodeTruncDivMod` correct for **Int** operands and pinned it with `int_divmod_test.go` — but every mapping that reaches it is **sort-blind**:

| Core shape reaching `EncodeExpr` | Mapping site | Today | Correct |
|---|---|---|---|
| `DictApp{Dict: DictRef{Fractional, Float}, "div"}` — the *normal post-pipeline shape* for float `/` | `methodToSMT["div"] = smtIntDivTrunc` (`codegen_operators.go:117`) | trunc Int encoding on Real args | **BUG** → `(/ n d)` |
| `App(VarGlobal{$builtin.mod_Float})` — normal shape for float `%` | `BuiltinToSMTOp["mod_Float"] = "mod"` (`types.go:272`) | Int-only `mod` on Real args | **BUG** → exact Real expansion (or honest skip) |
| `DictApp{Ord String, "lt"}` etc. | `methodToSMT["lt"] = "<"` (`codegen_operators.go:124`) | Int `<` on String args | **BUG** → `str.<` |
| `Intrinsic{OpDiv}` / `BinOp{"/"}` (pre-lowering / robustness paths) | `opMap[OpDiv] = smtIntDivTrunc` (`codegen_operators.go:14,34`) | trunc Int encoding, sort-blind | sort-dispatched via CoreTI/Lit kind |
| `App(div_Int)` / `App(mod_Int)` / `DictApp{Num Int, div/mod}` / `Intrinsic/BinOp` on **Int** | same sites | trunc Int encoding | ✅ unchanged — matches Go (pinned by `int_divmod_test.go`) |
| `App(div_Float)` | `BuiltinToSMTOp["div_Float"] = "/"` (`types.go:270`) | plain Real `/` | ✅ already correct |

How float `/` reaches the encoder as `DictApp`: the pipeline (`pipeline_single.go:373`, `ElaborateWithDictionaries`) turns every operator with a resolved class constraint into a dictionary application — for `1.0 / x` the constraint is `Fractional Float → div` (live: `ailang run --debug-types` prints `Constraint: Fractional → div` for the division node), and `internal/elaborate/dictionaries.go:105-110` builds `DictRef{ClassName: "Fractional", TypeName: "Float"}`. Op-lowering then passes `DictApp` through untouched (`op_lowering.go:293`). Float `%` has **no** `Fractional` method, so it stays an `Intrinsic`, which op-lowering *does* rewrite to `App(mod_Float)` — which is why `/` and `%` fail through *different* sort-blind sites with the same root cause: **operator→SMT mappings chosen by method name only, never by operand sort.**

The sort is available at every fix site: `da.Dict.(*core.DictRef).TypeName` is already `"Float"`/`"Int"`/`"String"` for `DictApp`; `EncodeFunctionOpts.CoreTI` (already passed by `verify.go:205`) types `Intrinsic`/`BinOp` operand nodes; `core.Lit` carries its `Kind`.

## Goals

**Primary Goal:** make `ailang verify` encode Float `/` and `%` (and `DictApp` String comparisons) by operand sort, so the reporter's `t.ail` yields `✓ VERIFIED` for both functions and no emitted script contains an Int-only operator applied to Real/String terms.

**Success Metrics:**

- `t.ail` (both `tidalRadialOrbit` and `tidalTransverse`): `ERROR`/`VIOLATION` → `VERIFIED`.
- A float `%` contract (`7.5 % x` with `ensures { result >= 0.0 }`, `requires { x > 0.1 }`): `ERROR` → `VERIFIED` (or honest skip — see D2), never a Z3 sort-mismatch error.
- A String-ordering contract (`a < b` on strings): `ERROR` → `VERIFIED`.
- All Int div/mod behavior unchanged: `internal/smt/int_divmod_test.go` passes byte-identical encodings; `examples/runnable/contracts/basic.ail` (`safeDivide`) and `insurance.ail` verdicts unchanged (insurance keeps its 1 pre-existing, unrelated `applyDiscount` violation).
- Zero `sort mismatch` / `unknown constant` strings in any `verify --json` run over `examples/` + `std/` (the `verifyStatuses` e2e helper already fails on these).
- PUB006 no longer trips on sunholo/relativity's float-division contracts (re-measured by the reporter on their Z3; `float=Real` scope line unchanged).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| **D1.** Float `/` encodes as plain SMT-LIB Real `(/ n d)` (mathematical division). Division by zero stays solver-unspecified (Z3: uninterpreted `/0`), the **same documented stance as the existing Int encoding** — NOT the reporter's proposed `d ≠ 0` side condition | Determines what VERIFIED claims and the refutation contract; the side condition needs assertion-hoisting machinery that does not exist (divisors are let-bound *inside* the body expression; script-level asserts cannot see them) | human | design | high |
| **D2.** Float `%` (`mod_Float`) encodes **exactly** as the truncated-remainder expansion `(- n (* d (ite (>= q 0) (to_int q) (- (to_int (- q))))))` with `q = (/ n d)` (Go `math.Mod` semantics: sign of dividend), rather than an honest skip | Extends the VERIFIED surface (ERROR → verifiable) vs. skipping; must not introduce a to_int floor gap (Z3 `to_int` rounds toward −inf; the `ite` compensates) | human at review | design | med |
| **D3.** Dispatch sources: `DictRef.TypeName` for `DictApp`; `activeCoreTI` (installed from `EncodeFunctionOpts.CoreTI`, mirroring `activeParamRenames`) then `Lit` kind for `Intrinsic`/`BinOp`; **sort unknown → honest skip (`ErrUnencodableTypes`), never a default to Int** | "Default to Int" is the exact class that produced this bug (parent M1: "never a default to Int"); the fallback rule is the recurrence guard | agent | design | med |
| **D4.** Scope = parent M1's A1+A2 slice (`DictApp` arithmetic-div/mod + comparison methods, `Intrinsic`/`BinOp` fallbacks). Parent M2 (name resolver), M3 (preflight), M4 (CI gate), F2/F5 stay deferred in the parent doc | Routing decision; prevents both scope creep and a duplicate roadmap | human | design | low |
| **D5.** No runtime, pipeline, or language changes — fix is `internal/smt`-local | Keeps blast radius at the encoder; engines keep engine parity (tree-walker and VM already agree: float div-by-zero → `+Inf`, verified live) | agent | design | low |

### Design Freeze

- [x] D1: Real `/` without side condition; div-by-zero solver-unspecified, documented as the existing Int stance (parent M0 comment: "any value is sound for partial correctness")
- [x] D2: exact `math.Mod` expansion chosen; honest skip is the reviewer fallback if the expansion is rejected
- [x] D3: sort-unknown → skip, never Int default
- [x] D4: A1+A2 slice only; no parent-M2/M3/M4 work
- [x] D5: encoder-local; no new pipeline pass (see Pipeline Pass Coverage)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Helper naming and file placement (`resolveOperator` vs extending `methodToSMT`; `codegen_operators.go` vs a new `resolve.go`) — agent may choose
- Whether the `DictApp` String comparison fix routes through `StringBuiltinSpecial` (reusing `FlipArgs`) or a small dedicated table — agent may choose
- Test file organization (extend `int_divmod_test.go` vs new `float_divmod_test.go`) — agent may choose
- Whether `mod_Float`-by-zero (runtime NaN, outside the Real model) gets a dedicated skip reason string or shares `ErrUnencodableTypes` — agent may choose
- Changelog wording and example-manifest entries for any new fixture — agent may choose, human at review
- A `DictApp` `abs` method, if one is ever reachable (`methodToSMT` has no `abs` entry today — grep-verified) — not part of this fix; add when reachable

## Solution Design

### Overview

Make every operator→SMT mapping in `internal/smt` **sort-directed** for the div/mod family and the `DictApp` comparison methods, following parent M1: one resolution function keyed by `(method/operator, operand sort)`, with the sort taken from the node (`DictRef.TypeName`, CoreTI, or `Lit` kind) and an **honest skip** when the sort cannot be determined. Int keeps today's byte-identical truncating encodings (they are correct and pinned); Float gets Real encodings; String gets `str.*`.

### Architecture

**Components:**

1. **Sort resolution helper** (`internal/smt`, ~40 LOC): `operandSortOf(expr core.CoreExpr) string` returning `"Int"`/`"Float"`/`"String"`/`"Bool"`/`""` from (a) `activeCoreTI.Get(expr.ID())` head type, (b) `*core.Lit` kind, (c) for `*core.DictRef` its `TypeName`. Installed once per `EncodeFunction` from `opts.CoreTI` (global `activeCoreTI`, save/restore like `activeParamRenames`, `codegen.go`).
2. **Real div/mod encoders** (`codegen_operators.go`, ~30 LOC): `encodeRealDiv(n, d) = "(/ n d)"`; `encodeRealMod(n, d)` = the D2 expansion with a let-bound `q` (parallel let, same binder-hygiene pattern as `encodeTruncDivMod`, including the `$ail_n`/`$ail_d` collision test).
3. **Dispatch at the four mapping sites**:
   - `encodeDictApp` (`codegen_operators.go:117`): resolve `da.Dict.(*core.DictRef).TypeName`; `div` → Int: trunc (unchanged) / Float: `encodeRealDiv`; comparisons `lt/le/gt/ge` → String: `str.<`/`str.<=` (with flip for `gt`/`ge`, as `StringBuiltinSpecial` already does for `gt_String`); unknown sort → `ErrUnencodableTypes`.
   - `BuiltinToSMTOp` (`types.go:269-272`): drop `"mod_Float": "mod"`; keep `"div_Float": "/"`; route `mod_Float` through a new `NumericBuiltinSpecial`-style entry (or the same table) to `encodeRealMod`.
   - `encodeIntrinsic` (`:14`) and `encodeBinOp` (`:34`): `OpDiv`/`"/"` and `OpMod`/`"%"` dispatch on `operandSortOf(firstArg)`: Int → trunc (unchanged), Float → Real encoders, unknown → skip.
4. **No change** to `encodeTruncDivMod`, `applyBinaryOp`'s Int sentinel path, `encodeBuiltinOp`'s arity handling, or any runtime/eval/VM code.

### Implementation Plan

**Phase 1: Sort resolution + Float `/`** (~4 hours)
- [ ] `operandSortOf` + `activeCoreTI` install/restore in `EncodeFunction`
- [ ] `encodeRealDiv`; dispatch in `encodeDictApp`, `encodeIntrinsic`, `encodeBinOp`; keep `div_Float` → `/`
- [ ] Unit test: every emitter shape × `1.0/x`-style Real operands agrees with Go `float64` division (dyadic-exact pairs: ±7.0/2.0, 0.5/0.25, …) via Z3 `unsat`-against-runtime, mirroring `TestIntDivModEncoding_MatchesGoTruncation`
- [ ] Shape pins: Real div encoding contains no `div`/`abs`/`(>= $ail_n 0)`, encodes operands once, no binder capture

**Phase 2: Float `%` + String comparisons** (~4 hours)
- [ ] `encodeRealMod` (D2), `mod_Float` rerouting, dispatch in all sites
- [ ] `DictApp` String `lt/le/gt/ge` → `str.<`/`str.<=` (+flip)
- [ ] Unit tests: `math.Mod` agreement on sign-combination pairs; string ordering agreement with AILANG runtime string comparison
- [ ] Unknown-sort emitters return `ErrUnencodableTypes` (skip), tested

**Phase 3: e2e, regression corpus, changelog** (~4 hours)
- [ ] e2e `cmd/ailang/verify_float_div_e2e_test.go`: the reporter's `t.ail` (both functions `verified`), float `%` contract, string-ordering contract, via existing `buildAilang`/`writeTempAil`/`verifyStatuses` (fails on any `unknown constant`/`sort mismatch`)
- [ ] Regression: `int_divmod_test.go` green untouched; `examples/runnable/contracts/basic.ail`, `insurance.ail` verdicts unchanged; `make verify-examples` green; `make test-core` green
- [ ] CHANGELOG `v0.56.2` entry; `std/VERSION` untouched by this doc

### Pipeline Pass Coverage

Not applicable — no compiler pipeline pass is added or modified (D5). Contract expressions (`requires`/`ensures`) go through the same pipeline elaboration as bodies (`pipeline_single.go:373`), so the `DictApp` fix covers contracts and bodies alike; verified live: the string-ordering repro above fails from its `requires { a < b }` clause.

### Files to Modify/Create

**New files:**
- `internal/smt/resolve_sort.go` (~40 LOC) — `operandSortOf`, Real div/mod encoders (placement per Deferred Decisions)
- `internal/smt/float_divmod_test.go` (~200 LOC) — emitter matrix × Real operands, shape pins, skip tests
- `cmd/ailang/verify_float_div_e2e_test.go` (~90 LOC) — `t.ail` repro, float `%`, string ordering

**Modified files:**
- `internal/smt/codegen_operators.go` (+45/−10) — sort dispatch in `encodeDictApp`/`encodeIntrinsic`/`encodeBinOp`
- `internal/smt/types.go` (+10/−5) — `mod_Float` routing; comparison entries if table-based
- `internal/smt/codegen.go` (+10/−0) — install/restore `activeCoreTI` from `opts.CoreTI`
- `changelogs/v0.32-current.md` (+1 section) — v0.56.2 entry

## Examples

**Before (live, `verify --verbose`, v0.56.1):**

```lisp
;; tidalTransverse — sort mismatch + Int div on Reals → Z3 error
(define-const result Real (let (($ail_n 1.0) ($ail_d $tmp11)) (ite (= (>= $ail_n 0) (>= $ail_d 0)) (div (abs $ail_n) (abs $ail_d)) (- (div (abs $ail_n) (abs $ail_d))))))
;; float % — Int-only mod on Reals → Z3 error
(define-const result Real (mod 7.5 $p_x))
;; string ordering in a contract — Int < on Strings → Z3 error
(assert (< $p_a $p_b))
```

**After:**

```lisp
;; tidalTransverse — Real division; VERIFIED under requires { r > 0.0 }
(define-const result Real (/ 1.0 $tmp11))
;; float % — exact math.Mod expansion (q let-bound once); VERIFIED under requires { x > 0.1 }
(define-const result Real (let (($ail_n 7.5) ($ail_d $p_x) ($ail_q (/ $ail_n $ail_d))) (- $ail_n (* $ail_d (ite (>= $ail_q 0) (to_int $ail_q) (- (to_int (- $ail_q))))))))
;; string ordering — str.<; VERIFIED
(assert (str.< $p_a $p_b))
```

**Unchanged (Int, correct today, pinned by `int_divmod_test.go`):**

```lisp
(define-const result Int (let (($ail_n 7) ($ail_d $p_x)) (ite (= (>= $ail_n 0) (>= $ail_d 0)) (div (abs $ail_n) (abs $ail_d)) (- (div (abs $ail_n) (abs $ail_d))))))
```

## Success Criteria

- [ ] `ailang verify t.ail` (reporter's repro): `tidalRadialOrbit` and `tidalTransverse` both `VERIFIED`
- [ ] Float `%` contract: `VERIFIED` (or honest skip with reason, per D2 review outcome), never a Z3 error
- [ ] String-ordering contract: `VERIFIED`
- [ ] No Int-only `div`/`mod` applied to Real/String terms in any emitted script (unit + shape pins)
- [ ] `internal/smt/int_divmod_test.go` passes unchanged (Int encodings byte-identical)
- [ ] `make verify-examples`, `make test-core` green; `basic.ail`/`insurance.ail` verdicts unchanged
- [ ] All tests passing; 90%+ coverage on new code
- [ ] Documentation updated (CHANGELOG v0.56.2 entry; this doc moves to `implemented/` on completion)

## Conflict Surface

### Syntactic positions touched

No grammar/parser/AST change (D5). The touched positions are **Core-expression encoder sites** in `internal/smt`:

- `encodeDictApp`'s `methodToSMT` map (method names: `div`, plus `lt`/`lte`/`gt`/`ge`/`eq`/`neq`)
- `encodeIntrinsic`'s `opMap` (`OpDiv`, `OpMod`) and `encodeBinOp`'s `opMap` (`"/"`, `"%"`)
- `BuiltinToSMTOp` entries `div_Int`/`div_Float`/`mod_Int`/`mod_Float`

### What else lives here

| Position | Existing valid forms | Shape |
|---|---|---|
| `DictApp` with `DictRef{Num, Int}` `div`/`mod` | Int truncating encoding | unchanged (correct vs Go) |
| `DictApp` with `Fractional/Num, Float` `add`/`sub`/`mul`/`neg` | `+`/`-`/`*`/`-` (sort-agnostic Real ops) | unchanged |
| `DictApp` with `Ord String` `lt`… | currently Int `<` (**the A1 bug**) | → `str.<` |
| `App(div_Float)` (lowered float `/` when dict-elab didn't take it) | plain `(/ n d)` | unchanged (already correct) |
| `App(mod_Int)`/`App(div_Int)` on Int | trunc encodings | unchanged |
| String/List/numeric specials (`StringBuiltinSpecial`, `ListBuiltinSpecial`, `to_real`/`to_int`) | same files, different code paths | unchanged |
| Float literal rendering (`encodeFloatLit`) | exact positional decimals, `.0` suffix | unchanged (M0-fixed, e2e-pinned) |

### Disambiguation strategy

The dispatch key is the **operand sort**, read per node: `DictRef.TypeName` (already normalized to `"Int"`/`"Float"`/`"String"` by `elaborate/dictionaries.go:105`), CoreTI head type from `VerifyOptions.CoreTI` (already plumbed to `EncodeFunctionOpts.CoreTI`), then `core.Lit` kind. No syntactic lookahead is involved; a sort that cannot be determined skips the function honestly rather than guessing Int (D3). This mirrors the parent M1 `resolveOperator(method, operandType)` contract.

### Programs that MUST still work

1. `examples/runnable/contracts/basic.ail` — `safeDivide` (Int `/` with `requires { divisor != 0 }`) — Int trunc encoding unchanged
2. `examples/runnable/contracts/insurance.ail` — Int arithmetic across 6 exported functions; its one pre-existing `applyDiscount` violation must stay byte-equivalent (it is an Int-overflow demo, unrelated)
3. `examples/runnable/contracts/empty_list_sort_verify.ail` + `cmd/ailang/verify_empty_list_sort_e2e_test.go` — the M0-era A4 fix, same encoder neighborhood
4. `cmd/ailang/verify_float_sqrt_e2e_test.go` fixtures — `tiny()`/`neg()` float literals and the `sqrt`-calling-helper skip path
5. `internal/smt/int_divmod_test.go` — all nine Int emitter shapes × sign pairs, binder-collision and nested-operand cases
6. External consumer: `sunholo/relativity` (R2-KERR) — float-heavy Kerr/celestial contracts; this bug is what blocks its publish

### What deliberately changes

- Float `/` and `%`, and String comparisons in contracts/bodies: Z3 hard errors and **bogus refutations** → faithful Real/`str.*` encodings. A float-division contract that v0.56.1 called `VIOLATION` or `ERROR` can now be `VERIFIED` — that is the fix, not a regression.
- Float division-by-zero under a permissive precondition (`requires { x >= 0.0 }` with body `1.0/x`): remains solver-unspecified (D1) — a refutation at divisor 0 is a model artifact, same as the shipped Int stance today; the `d ≠ 0` obligation hoist is Future Work.
- No syntax, typing, runtime, effect, or publishing-gate behavior changes.

## Testing Strategy

**Unit tests** (`internal/smt/float_divmod_test.go`):
- Emitter matrix mirroring `intDivModEmitters`: builtin `div_Float`/`mod_Float`, curried forms, `DictApp` (`Fractional`/`Float`/`div`, `Num`/`Int`/`div` regression), `Intrinsic` `OpDiv`/`OpMod`, `BinOp` `/`/`%` — each × sign-combination operand pairs, asserted **against the Go runtime value** (`float64` `/`, `math.Mod`) via Z3 `unsat`, skipping when `Z3Available()` is false (same pattern as `int_divmod_test.go`)
- Shape pins: Real division emits `(/ …)` and never `div`/`abs`; operands encoded once; internal `$ail_*` binder-collision test extended to the Real mod expansion
- Unknown-sort emitters (CoreTI miss, non-literal args) → `ErrUnencodableTypes`, never an Int encoding

**Integration/e2e tests** (`cmd/ailang/verify_float_div_e2e_test.go`):
- The reporter's `t.ail` verbatim → all statuses `verified`
- Float `%`, string ordering, and a divisor-0-under-permissive-pre case asserting the documented D1 stance
- Uses `buildAilang`/`writeTempAil`/`verifyStatuses` (which fails on any `unknown constant` in output)

**Regression-surface tests:**
- `int_divmod_test.go` untouched and green (Requirement: the fix must not alter Int encodings)
- `make verify-examples` (CI gate) and `make test-core` green

**Manual verification:**
- `ailang verify --verbose` on `t.ail`, `modf2.ail`, `a1a4.ail` repro files (kept in the e2e test as source strings)

## Non-Goals

- **Parent M2** (one name resolver / callee leak guards), **M3** (well-formedness pre-flight), **M4** (CI gate) — remain deferred in [m-smt-well-sorted-preflight.md](../../planned/m-smt-well-sorted-preflight.md)
- The `d ≠ 0` obligation hoist for division by zero (D1) — Future Work; needs script-level assertion hoisting
- IEEE-754 float modeling (Inf/NaN/rounding) — the encoded model stays `float=Real` (mathematical, exact); the scope line already disclaims runtime equivalence
- BitVec (`<<`/`>>`/`&`/`^`) sort dispatch — not reachable for Float (type checker rejects); no change
- Runtime float div-by-zero unification across engines (both already return `+Inf`; tree-walker `evalDiv`'s "division by zero" branch is dead for lowered code — noted, not fixed here)
- `ailang compile --emit-go` parity work (frozen per `design_docs/v1-mission.md`)

## Timeline

**Week 1** (3 days):
- Phase 1: sort resolution + Real `/` (4h)
- Phase 2: Real `%` + String comparisons (4h)
- Phase 3: e2e + regression corpus + changelog (4h)
- Buffer: Z3-version variance re-check on reporter's machine (their Z3 produced r=7/4; ours 3.000000002) + review fixes (4h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Z3 version variance: 4.8.12 silently coerces ill-formed `div`-on-Real (bogus sat) where other versions hard-error; a future solver could do worse | High | The fix removes the ill-formed text entirely; e2e asserts well-formed output; CI Z3 pinned; reporter re-runs on their build |
| Real mod expansion subtle wrongness (`to_int` floor vs trunc) | Medium | Explicit `ite` sign branch; unit test against `math.Mod` on sign-combination pairs; reviewer may downgrade D2 to honest skip without touching D1/D3 |
| `DictRef.TypeName` missing/odd for exotic dictionaries | Medium | `operandSortOf` falls back to CoreTI/Lit kind; unknown → skip (D3), never Int default |
| Int encoding regresses (it is correct and heavily pinned) | Medium | `int_divmod_test.go` must pass byte-identical; Int paths are the explicit "unchanged" column in the dispatch |
| Corpus flip: contracts previously ERROR (skipped downstream) now refute for real | Low | That is the verifier working; `verify --package` reports per-function reasons; PUB006 correctly gates real refutations |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One well-formed script per function; removes Z3-version-dependent elaboration of ill-formed div/mod terms (today the same source verifies, errors, or bogus-refutes depending on solver build) |
| A2: Replayability | 0 | No trace surface change |
| A3: Effect Legibility | 0 | No effect-row or capability change |
| A4: Explicit Authority | 0 | No authority/capability change |
| A5: Bounded Verification | +1 | The verifier becomes usable (sound *and* informative) on Float arithmetic — the language's `pure` float contracts become checkable locally |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Machine consumers of `verify --json` stop being told true float contracts are false (`VIOLATION`) or errored — the primary consumer of this verifier is AI-generated code |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No resource-cost surface change |
| A10: Composability | +1 | Reuses parent M0 encodings, `EncodeFunctionOpts.CoreTI`, and e2e helpers; dispatch composes with existing string/list/numeric tables |
| A11: Structured Failure | +1 | Unresolvable sorts skip with a reason instead of leaking raw symbols into Z3 ("unknown constant") |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — solver-unspecified `/0` matches the shipped Int stance and is documented
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Verification Log

Every load-bearing claim above, checked live on this workspace (v0.56.1, `d555a52a`, Z3 4.8.12) unless noted:

| # | Claim | Evidence |
|---|---|---|
| V1 | Repro: `VIOLATION tidalRadialOrbit` + `ERROR tidalTransverse` (sort mismatch, `unknown constant result`) | `ailang verify /tmp/repro/t.ail` — output quoted in Problem Statement |
| V2 | Generated SMT is `encodeTruncDivMod` output on Real operands | `ailang verify --verbose /tmp/repro/t.ail`; matches `codegen_operators.go:86` format string |
| V3 | Same ill-formed text → hard error on one script, silent `to_int`-coercion + bogus `sat` model on the other | `z3 -smt2` on both extracted scripts (radial: model contains `to_real (div 1 (to_int …))`, exit 0; transverse: sort mismatch, exit 1) |
| V4 | Z3 treats `(/ 1.0 0.0)` as uninterpreted `/0` (arbitrary Real) | `z3 -smt2` on `(assert (= q (/ 1.0 0.0)))(assert (= q 5.0))` → `sat`, model defines `/0` |
| V5 | Float `/` reaches the encoder as `DictApp{Fractional, Float, "div"}` | `ailang run --debug-types` on the repro prints `Constraint: Fractional → div`; `elaborate/dictionaries.go:105-110` builds `DictRef{TypeName}`; `op_lowering.go:293` passes `DictApp` through |
| V6 | Float `%` reaches the encoder as `App(mod_Float)` and errors | `ailang verify modf2.ail` → `! ERROR h`, `(mod 7.5 $p_x)` in `--verbose`, `BuiltinToSMTOp["mod_Float"] = "mod"` at `types.go:272` |
| V7 | String ordering in contracts errors today (A1) | `ailang verify a1a4.ail` → `! ERROR strLt`, `Sort mismatch … (declare-fun < (Int Int) Bool) supplied sort is String` |
| V8 | Int div/mod encodings are correct and must not change | `ailang verify modf2.ail` → `✓ VERIFIED g` (`7 % x`); `int_divmod_test.go` pins all nine emitter shapes × sign pairs |
| V9 | Runtime: float `1.0/0.0` → `+Inf` on BOTH engines; int `1/0` → RT001 on both | `ailang run --caps IO` and `ailang run --bytecode --caps IO` on `divzero.ail`/`intdiv0.ail` |
| V10 | Runtime: float `%` = Go `math.Mod` on both engines | `ailang run --caps IO modf.ail` → `0.0` (= `math.Mod(1.5, 0.5)`) both engines |
| V11 | Negative existence: no sort-aware dispatch exists in `internal/smt` today | `grep -rn "resolveOperator\|operandSort\|sortOf" internal/smt/*.go` → no hits (only unrelated `TypeName` uses in records/filter) |
| V12 | Negative existence: `methodToSMT` has no `mod`/`abs` entry | `grep -n '"mod"\|"abs"' internal/smt/codegen_operators.go` → empty |
| V13 | Negative existence: no float-division e2e fixture exists | `grep -rln "tidal\|div_Float" cmd/ailang/verify_*_test.go` → empty |
| V14 | Negative existence: no v0.56.2 changelog section exists yet | `grep -n "v0.56.2" changelogs/v0.32-current.md` → empty |
| V15 | `EncodeFunctionOpts.CoreTI` is already available in the encoder context | `codegen.go:86`; `verify.go:205` passes `opts.CoreTI`; `parametric_context.go:25-30` precedent for consuming it |
| V16 | `DictRef.TypeName` carries `"Float"`/`"Int"`/`"String"` | `elaborate/dictionaries.go:105` `types.NormalizeTypeName(rc.Type)`; parent doc M1 §probe states the same |
| V17 | PUB006 gates publish on refuted contracts | `internal/pkg/quality.go:294` `r.gate("PUB006", … Counterexample > 0)` |
| V18 | Parent doc covers this as A2/RC1/M1 (deferred) — not silently re-invented | [m-smt-well-sorted-preflight.md](../../planned/m-smt-well-sorted-preflight.md) lines 584 (A2, names `tidalRadialOrbit`), 608 (RC1), 828 (M1 `resolveOperator`, "never a default to Int") |
| V19 | `make verify-examples` exists as a CI gate on examples | `make/examples.mk:18` |
| V20 | Existing e2e helpers assert on verify statuses and fail on `unknown constant` | `cmd/ailang/verify_float_sqrt_e2e_test.go` (`buildAilang`, `writeTempAil`, `verifyStatuses`) |
| V21 | All repro `.ail` sources in this doc compile under the current binary | each was run through `ailang verify` (which compiles first); `modf.ail` also `ailang check`-clean |
| V22 | No new pipeline pass: fix sites are all inside `internal/smt` encoder dispatch | Files list; `pipeline_single.go`/`op_lowering.go` untouched |

## References

- **Parent roadmap (umbrella)**: [m-smt-well-sorted-preflight.md](../../planned/m-smt-well-sorted-preflight.md) — RC1/A1/A2, M1 `resolveOperator`, M0 merged v0.56.0 (PR #2019, issues #1960/#1961); M2–M4 deferred there
- **M0 implementation pinned by**: `internal/smt/int_divmod_test.go` (`#1960 (M-SMT-WELL-SORTED-PREFLIGHT M0, S1)`)
- **Prior sibling fix in the same layer**: `internal/smt/codegen_expr.go` `encodeFloatLit` (float literal rendering, stapledons report 2026-11-02), pinned by `cmd/ailang/verify_float_sqrt_e2e_test.go`
- **Reported by**: coordinator task (inbox_1793649570012_28ca24e0 + this report with repro and SMT), blocking sunholo/relativity R2-KERR, ailang-packages PR #173, PUB006
- **Gate**: `internal/pkg/quality.go` PUB006/PUB016 (publish-time contract verification)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **Division-by-zero obligation hoist** (D1 follow-up): lift divisor terms to script-level bindings so `(assert (distinct d 0))`-style side conditions can be emitted for BOTH Int and Float, eliminating refutation-at-divisor-0 model artifacts; likely a small `internal/smt` pass over the encoded result. Would also harden the shipped Int stance.
- **Parent M2–M4**: name resolver, well-formedness pre-flight, CI gate — see the parent doc.
- **IEEE-754 float model** (Inf/NaN/rounding) — explicitly out of scope; would be a new encoded-model decision, not an encoder fix.