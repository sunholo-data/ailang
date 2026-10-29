# M-EFFECT-LATENT-NESTED-CALLBACKS — Charge function values nested in data arguments, and close concrete-annotation tails wherever a function's row is read

**Status**: Implemented — unreleased
**Target**: v0.56.0 (next minor; original v0.55.0 target was released before integration). This is a soundness fix that rejects programs which check today, so it gets a minor bump, the same posture Mark ruled for M-EFFECT-LATENT-FUNCTION-VALUES on 2026-10-08.
**Priority**: P0 (High). A `pure` function can perform IO, `ailang check` reports it clean, and the side effect runs on both engines.
**Tracks**: #1718, #1826
**Refs**: #573, #1326 (L1/L2 rules this builds on), #616 (per-App publication record), #1443 / PR #1790 (ordering only, see "Ordering against PR #1790")
**Estimated**: 3–4 days (Phase 1 ≈ 1 day, Phase 2 ≈ 1 day, Phase 3 ≈ 0.5 day, Phase 4 corpus and docs ≈ 0.5–1 day)
**Dependencies**: None blocking. It builds on the shipped `ApplicationEffects` record (#1708 `3d4f49720`, #616 `accdfc4bf`).
**Base measured**: worktree at `origin/dev` `772703831`. The binary was built from that tree (`go build -o …/dd1/ailang ./cmd/ailang`) and checked behaviourally, not by its version stamp (known worktree stamping quirk). All checks ran with `AILANG_NO_CACHE=1`.
**Quorum trigger**: #1 fires. There are design-freeze items: the breaking-change posture and the intentional flips listed under "What deliberately changes". The requester is handling review, so no quorum was run in this session.

## Summary

The checker reads a function value's latent effects in five places. Each place recognises function types by **shape**: a top-level arrow parameter, a top-level arrow binder, or a direct callee. None of them uses a structural, provenance-aware rule. The places disagree, and both issues fall into the gaps:

1. **#1718 (unsound).** The callee body *erases* an unannotated callback's tail when the callback comes out of a tuple or list pattern. It treats the effect as the caller's to pay. The call site, though, *charges* only callbacks whose parameter type is a top-level arrow (`callbackMask`, `typechecker_latent_mask.go:15-27`). A function value inside a list, tuple, `Option`/generic ADT or record argument is therefore paid by nobody.
2. **#1826 (incomplete, crashes).** A concrete annotation `(string) -> bool ! {IO}` is elaborated as `{IO | ε_annotN}` (L1-open, `elaborate/file_funcs.go:414-417`). Only the direct-callee read closes that compatibility tail (`applicationCallRow`, `typechecker_app_effects.go:289-295`). The inferred App row (`typechecker_functions.go:656`) and the per-argument callback row (`typechecker_app_effects.go:204-206`) keep it. The tail then escapes into `mapE`'s `e`. In the parameter shape this is reported as `Unresolved effect tail`. In the record-field shape, where no binder owns the tail, it trips the internal invariant (`validate_effects_latent.go:56`).

**Fix:** add one type-directed helper that walks *data positions* only (list, array, map, tuple, type-application arguments, record fields; never arrow parameters or returns). For each arrow it reaches, it yields the latent row and closes a concrete contract's compatibility tail. All four sites use it: the callee mask, the per-argument callback rows, the implicit-ownership set, and the App constraint row. A scratch prototype of exactly this flips every leak and every #1826 shape to the expected answer. It has **zero status flips over 501 `examples/` and `std/` files and 44 `ailang-packages`**, and it makes `sunholo/mcp_oauth` 0.1.0 and 0.1.1 compile again (V20–V24).

## Why a new focused doc, not an addendum to M-EFFECT-LATENT-FUNCTION-VALUES

`design_docs/planned/v0_48_0/m-effect-latent-function-values.md` is the conceptual parent. Its L2 rule ("charge at polymorphic hand-off") is what this doc extends to nested positions. Even so, it is not the right home:

- **Its "Phase 3 deferred" is a different question.** That phase is upper-bound annotations (L1-closed, "open on use", the V13 width hole for #573/#1326). It does not cover nested callbacks; that doc never mentions containers. Adding #1718 there would bury a P0 under a human-gated spike that this fix does not need.
- **Its status is mixed** ("Phases 0–2 and 4 implemented for v0.53.0"), and its sprint plan and validation are closed. A new P0 needs its own acceptance criteria, corpus evidence and sprint handoff.
- **M-EFFECT-ROW-VAR-UNIFICATION (v1_0_0, line 648) excluded #1718 on purpose**, because it needs "new structural traversal". That traversal is exactly what this doc adds.

This doc cites L1-open and L2 from the parent and does not reopen any of the parent's frozen rulings.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change; checking stays deterministic |
| A2: Replayability | 0 | No trace or format change |
| A3: Effect Legibility | +1 | A declared row (incl. `pure`) becomes true for callbacks in data; today `pure` lies (V2–V8) |
| A4: Explicit Authority | +1 | IO performed under a `pure` signature is the ambient-authority hole this closes; capabilities stay the runtime backstop |
| A5: Bounded Verification | +1 | One local, type-directed rule at the App replaces shape-specific special cases; closes an internal-invariant crash (#1826) |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Agents get one consistent answer for `f`, `[f]`, `(f, x)`, `Some(f)`, `{f: f}`; #1826's false "propagate the shared row" advice disappears |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No cost surface |
| A10: Composability | +1 | `mapE` composes with annotated callbacks again (#1826); containers of callbacks compose with HOFs soundly |
| A11: Structured Failure | +1 | Replaces an `internal invariant` with the ordinary `Missing effects` diagnostic and a correct Suggested fix |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): removes a hidden side effect; adds none
- [x] A4 (Authority): removes ambient access; grants none
- [x] A7 (Machines First): the rule is type-directed and uniform; no human-convenience special case

## Problem Statement

### The repros (base `772703831`; every snippet below was run through `ailang check`)

Shared prelude for R1–R7:

```ailang
module r/leak_tuple
import std/io (println)

func logIt(x: int) -> int ! {IO} = { println("side effect"); x }
```

| # | Shape | Callee | Pure caller | Base `check` | Base `run --caps IO` (tree-walk and `--bytecode`) |
|---|---|---|---|---|---|
| R1 | tuple | `func applyFst(p: ((int) -> int, int)) -> int = match p { (f, x) => f(x) }` | `pure func sneaky(x: int) -> int = applyFst((logIt, x))` | ✓ clean | prints `side effect` |
| R2 | list | `func applyHead(fs: [(int) -> int], x: int) -> int = match fs { [] => x, [f, ..._] => f(x) }` | `pure func sneaky(x: int) -> int = applyHead([logIt], x)` | ✓ clean | prints `side effect` |
| R3 | Option, unannotated param | `func applyOpt(o, x: int) -> int = match o { Some(f) => f(x), None => x }` | `pure func sneaky(x: int) -> int = applyOpt(Some(logIt), x)` | ✓ clean | prints `side effect` |
| R4 | record param | `func applyRec(p: {f: (int) -> int, x: int}) -> int = p.f(p.x)` | `pure func sneaky(x: int) -> int = applyRec({f: logIt, x: x})` | ✓ clean | prints `side effect` |
| R5 | record alias, cross-module | `x/librec`: `export type R = {f: (int) -> int, x: int}`; `export func applyRec(p: R) -> int = p.f(p.x)` | `pure func sneaky(x: int) -> int = applyRec({f: logIt, x: x})` (in `x/mainrec`) | ✓ clean | prints `side effect` on both engines |
| R6 | let-bound tuple | R1's `applyFst` | `pure func sneaky(x: int) -> int = { let t = (logIt, x); applyFst(t) }` | ✓ clean | prints `side effect` |
| R7 | returned list | `func mk() -> [(int) -> int] = [logIt]` + R2's `applyHead` | `pure func sneaky(x: int) -> int = applyHead(mk(), x)` | ✓ clean | prints `side effect` |
| R8 | non-generic ADT holding a list | `type Reg = Reg([(int) -> int])`; `func applyReg(r: Reg, x: int) -> int = match r { Reg(fs) => match fs { [] => x, [f, ..._] => f(x) } }` | `pure func sneaky(x: int) -> int = applyReg(Reg([logIt]), x)` | ✓ clean | prints `side effect` |
| R9 | list of tuples | `func applyAll(fs: [((int) -> int, int)]) -> int = match fs { [] => 0, [(f, x), ..._] => f(x) }` | `pure func sneaky(x: int) -> int = applyAll([(logIt, x)])` | ✓ clean | prints `side effect` |

The issue's third shape, `applyOpt(Some(logIt), x)`, behaves differently when the parameter is **annotated**, and the result is a false *rejection* of the callee, not a leak:

| # | Callee | Base `check` |
|---|---|---|
| R10 | `func applyOpt(o: Option[(int) -> int], x: int) -> int = match o { Some(f) => f(x), None => x }` | ✗ `Effect checking failed for function 'applyOpt'` / `Unresolved effect tail: ε_annot1; propagate the shared row in the function signature` |
| R11 | `type Holder[a] = Hold(a)`; `func applyHold(h: Holder[(int) -> int], x: int) -> int = match h { Hold(f) => f(x) }` | ✗ same `Unresolved effect tail` on `applyHold` |

R10 is an ordinary, valid higher-order function. It fails with #1826's exact diagnostic text, but here the tail is an *implicit* (unannotated) `ε_annot`, not a concrete one. That shared symptom is the first clue that the two issues share a mechanism.

#1826 shapes (`H` from the issue):

```ailang
module v/viafield
import std/list (mapE)
export type H = { act: (string) -> bool ! {IO} }

export func viaField(h: H, xs: [string]) -> [bool] ! {IO} = mapE(\d. h.act(d), xs)
```

| # | Program | Base `check` |
|---|---|---|
| S1 | `viaField` above | ✗ `internal invariant: missing or malformed ApplicationEffects (LatentParamMask) for typed application 10` |
| S2 | `export func viaParam(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] ! {IO} = mapE(\d. g(d), xs)` | ✗ `Unresolved effect tail: ε_annot1` |
| S3 | the same with a local `func myMapE[a, b](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e}` | ✗ `Unresolved effect tail: ε_annot4`. So this is not specific to `std/list` or to imports |
| S4 | `func applyTo(f: (string) -> bool, x: string) -> bool = f(x)`; `func viaParamValue(g: (string) -> bool ! {IO}, x: string) -> bool ! {IO} = applyTo(g, x)` | ✗ `Unresolved effect tail: ε_annot2`. No lambda and no `mapE`: an `! {IO}` parameter handed to any open HOF |
| S5 | `sunholo/mcp_oauth` 0.1.0 and 0.1.1 (registry cache), `check --package` | ✗ invariant at `flow.ail:193` / `:195` (`revokeFamily`); 0.1.2 passes (workaround) |
| S6 | `~/dev/sunholo-data/docparse` (pins mcp_oauth 0.1.1) | 2 files fail on that invariant |

### Controls that are already correct on base

| Program | Base |
|---|---|
| `func applyTo(f: (int) -> int, x: int) -> int = f(x)`; `pure func sneaky(x: int) -> int = applyTo(logIt, x)` (#1326) | ✗ `Missing effects: IO` ✓ |
| `type Box = Box((int) -> int)`; `applyBox(Box(logIt), x)` from pure | ✗ `Missing effects: IO`. It is charged at *construction*, because `Box` is a constructor with a top-level arrow parameter |
| `[\n. { println("side effect"); n }]` passed to `applyHead` from pure | ✗ `Missing effects: IO`. A lambda literal is charged by its body |
| `x/lib` exports R1/R2's callees; a pure caller in `x/main` | ✗ `Missing effects: IO`. Imported callees use the published call row (see RC4) |
| `func mk() -> (int) -> int = logIt`; `pure … = applyTo(mk(), x)` | ✗ `Missing effects: IO` |
| S2 with a hand-written `func ap(k: (string) -> bool ! {IO}, x: string) -> bool ! {IO} = k(x)` | ✓ checks |

### Impact

- Any `pure` or narrower-row guarantee can be bypassed by putting the function value in a list, tuple, `Option`/`Result`/generic ADT payload, record, or a combination of these. The same holds after a `let`, through a returned container, or across modules for record aliases (R5). The runtime capability layer is the only backstop, so serve-api / MCP purity hints and motoko's `declared_vs_performed` classification inherit the lie.
- #1826 breaks the published `sunholo/mcp_oauth` 0.1.1 (live in AILANG Parse prod on v0.52.0) for any adopter on ≥ v0.53.1. It also rejects ordinary code (an `! {IO}` callback into `mapE` inside an `! {IO}` function, or S4) with advice that is wrong.

## Root Cause

### Where the checker reads a function value's latent row (base `772703831`)

| Site | File:line | What it recognises | Gap |
|---|---|---|---|
| **(a) Callee mask** (charge side) | `internal/types/typechecker_latent_mask.go:15-27` (`callbackMask`), called via `resolvedCallbackMask` `:74-85` from `inferApp` `typechecker_functions.go:561` | `param.(*TFunc2)` with an open, non-concrete row, **top-level only** (`:22`) | A param type `((int) -> int, int)`, `[(int) -> int]`, `Option[(int) -> int]`, `{f: (int) -> int}`, alias `R` → mask `false` → no charge |
| **(b) Per-argument callback row** | `internal/types/typechecker_app_effects.go:202-208` (publish); `:79-89` (re-zonk) | `effectAliasHead(argType).(*TFunc2)`, **head only** (`:204`, `:81`) | Nested arrows are never published. A concrete arrow's compatibility tail is cloned as-is (`:205`, `:82`) |
| **(c) Implicit-ownership set** (erase side) | `typechecker_app_effects.go:218-228` (`implicitRowsAbove`), `:306-324` (`resolveOwnedTypes`) | Binder whose type is a top-level `*TFunc2` with `ImplicitEffectContract` (`:222`), or a `TVar2` binder resolved later (`:309`) | A tuple/list pattern binds `f` directly to the element arrow, so it is recognised and erased. A record field or a param type that only *contains* the arrow is not. `Some(f)` against an annotated `Option[(int) -> int]` lands in neither branch → kept → `Unresolved effect tail` (R10, R11) |
| **(d) App constraint row** | `internal/types/typechecker_functions.go:580` (row), `:607` (`callRow`), **`:656`** (`appEffects := append(argEffects, effectRow)`) | Uses the unified `effectRow`. For a `ConcreteEffectContract` callee that is `{IO \| ε_annotN}` | The published `callRow` closes the tail (`applicationCallRow`, `typechecker_app_effects.go:289-295`), but the typed App's own row, which becomes an enclosing lambda's latent row and then `mapE`'s `e`, keeps it |
| **(e) Validator** | `internal/pipeline/validate_effects.go:363-370` (declared authority), `:387-392` (call-row override), `:394-412` (mask × `CallbackRows`) | Same-module callee with a declared row: the declared row wins, unless it has a tail | A same-module callee with no `!` (empty declared row) is never charged from its call row, so the mask (a) is the **only** bridge for its callbacks' effects |

The other places these rows come from:

- `internal/elaborate/file_funcs.go:393-426` gives every arrow annotation a fresh `ε_annotN` tail. An unannotated arrow becomes `ImplicitEffectContract`. A concrete `! {IO}` keeps the tail as an L1-open compatibility tail (`:414-417`, `ConcreteEffectContract = true`).
- `internal/pipeline/validate_effects_latent.go:56` raises the invariant when a published `CallRow` has an unowned tail.

### RC1 — #1718: the erase side is wider than the charge side

The latent-effect contract from #1326 says an unannotated callback's effects are the **caller's** to pay. The callee body erases the implicit tail (`ImplicitOwnedTail`, `validate_effects.go:389-391`; `ImplicitCallbacks`, `:407-409`), and the caller is charged through the mask. For soundness, *every tail erased in a body must be charged at every call site*. Site (c) erases tails reached through tuple and list pattern binders. Site (a) charges only top-level arrow params. Measured on R1 with `DEBUG_EFFECTS=1`:

```
=== Validating Let binding: applyFst ===
      Published App 12 call row: {...ε_annot1}     # f(x) inside the tuple match
      App total effects: []                         # implicit tail erased (site c)
=== Validating Let binding: sneaky ===
      Callee declared effects (from signature): []  # site e: declared authority
      Published App 22 call row: {IO}               # the checker KNOWS — but it is ignored
      Arg[0] effects: []                            # mask[0] == false (site a)
      App total effects: []
```

The type checker computes the right call row (`{IO}`). The validator discards it because the same-module callee has an empty declared row (RC4), and the mask, which is the only remaining bridge, is shape-blind.

### RC2 — #1826: a concrete annotation's compatibility tail is closed at one read and leaks at the others

`applicationCallRow` closes the `ε_annotN` of a `ConcreteEffectContract` callee. Sites (b) and (d) do not. Measured on S2:

```
=== Validating Let binding: viaParam ===
      Published App 7 call row: {IO, ...ε_annot1}   # mapE's e := the lambda's row (site d)
      Published App 4 call row: {IO}                # g(d) itself: tail closed (applicationCallRow)
Error: … Unresolved effect tail: ε_annot1
```

In S1 the tail comes from the *type declaration* `H`. No binder in `viaField`'s environment has a type in which `ε_annotK` is free: `h: H` is reached only through the alias. So `CallTailOwned` is false and the invariant at `validate_effects_latent.go:56` fires instead of the subsumption error. S4 is the same leak through site (b): the callback row for `g` is published as `{IO | ε_annot2}`.

### RC3 — does #1826 share #1718's root cause? **Yes, one mechanism with two symptoms**

Both are failures of the **per-argument latent-row publication**: the code that turns a function value's type into "what calling it costs". Today that answer is computed separately at five sites, each keyed on a local shape (top-level param, top-level binder, direct callee). #1718 is missing *positions* (nested data). #1826 is missing *tail classification* (concrete compatibility tail). R10 and R11 show the two crossing: an implicit tail reached through `Some(f)` produces #1826's exact diagnostic. The evidence that one fix covers both is the prototype, which changes only these sites (V20).

What #1826 does **not** share: it is not a `mapE` or stdlib defect (S3 uses a local combinator; S4 uses none). It is also not #573/#1326's open question of annotations as upper bounds (Phase 3 of the parent doc, see Non-Goals).

### RC4 — why imported callees are (mostly) charged and same-module ones are not

For a `*core.VarGlobal` callee, `usedDeclared` is false (`validate_effects.go:363-370` keys declared rows by same-module name only). The validator therefore uses the published, solved `CallRow`, which for R1/R2 is `{IO}`. That is why `x/main` is rejected on base. The call row is **not** a complete authority, though. A record alias's arrow tail (R5) never enters the callee's generalised row, so the cross-module record case still leaks. Alternative C below measures this.

### Adjacent defect found while measuring (U1): out of scope, must be filed separately

`RowUnifier.UnifyRows` (`internal/types/row_unification.go:97-121`) unifies an **open row that has labels** with a **closed** row without checking the open side's extra labels. A direct unit probe in a scratch copy shows `{IO | ε} ~ {}` (closed) succeeds with `ε := {}`, silently dropping `IO`. Record rows behave the same: `{x: int | r} ~ {y: int}` → `r := {y: int}`.

Observable leak on base: `std/list.map` takes a closed-pure callback, so `map(\d. println(d), xs)` is a type error. But `pure func viaMapPure(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] = map(g, xs)` checks clean, and calling it with `logIt` prints. The naive fix (reject a non-empty `only1`/`only2`) flips **17/501** corpus files (`std/ai.ail` at `_ai_step_with_stream`, plus 16 AI examples that import it), so it needs its own design. This doc neither depends on nor fixes U1. It does interact with it: closing concrete tails at site (d) moves some programs off the U1 path (see "What deliberately changes").

## Goals

**Primary Goal:** A function value's latent effects are charged to the caller wherever the callee's parameter type nests an effect-polymorphic arrow in a data position. A concrete annotation's compatibility tail never escapes into a published or inferred call row.

**Success Metrics:**
- R1–R9 are rejected with `Missing effects: IO` on `sneaky`, on both engines, before execution.
- R10, R11, S1–S6 check clean (S4/S2/S3 with their declared `! {IO}`). There is no `internal invariant` and no `Unresolved effect tail`.
- Zero unexplained status flips over `examples/**` + `std/**` (501 files at base), `ailang-packages` (45 package roots), `ailang-parse` (133 files), docparse, and motoko/world package roots.
- `sunholo/mcp_oauth` 0.1.0 and 0.1.1 check clean again. No package pin bump is needed.

## Non-Goals

- **Upper-bound annotations** (L1-closed, "open on use", V13 width hole). These are the parent doc's Phase 3, human-gated and tracked under #573/#1326. Concrete annotations stay L1-open. This doc only makes every *read* of a concrete row agree with the read that already exists (`applicationCallRow`).
- **U1 (row unifier label check).** File it as its own issue with the measurement above (17/501 flips) and design it separately.
- **Charging at construction or reference.** The parent doc's frozen ruling stays: latent effects are charged at application and at hand-off to an effect-polymorphic parameter, never at mere reference (V18 there).
- **Making `std/list` `map`/`filter`/`foldl` effect-transparent.** That is a separate human decision (parent doc, Deferred Decisions).
- **Traversing a non-generic ADT's constructor field types from a `TCon` parameter.** It is not needed for soundness. Every value of such a type is built by a constructor application, and the extended mask charges that application when the constructor's parameter nests an open arrow. On the prototype, `pure func mkReg() -> Reg = Reg([logIt])` is rejected on `mkReg` (V25). Recorded as a deferred check, not a gap.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Nested hand-off charges like top-level hand-off: passing a container that holds an effectful value to a callee param that nests an open arrow charges the caller, **even if the callee only stores it** | Consistent with the frozen top-level rule (`keep(f)` from pure is already rejected, V12). The alternative (charge only if invoked) needs C, which misses R5 | design (evidence V12, V25) | design | high |
| One data-position walker in `internal/types` feeds sites (a)–(d) | Prevents the next shape-specific gap; the five-site divergence *is* the bug | design | compile | med |
| Concrete compatibility tails are closed at every read of a function value's row, matching `applicationCallRow` | Fixes #1826 without changing L1-open; the alternative (L1-closed) is the human-gated Phase 3 | design | compile | high |
| Ship as a soundness break (minor bump, no opt-out), like v0.53.0 | Rejects R1–R9-shaped programs; flips `map(\d. g(d), xs)` (see below) | **human** | before merge | med |

### Design Freeze

- [x] Charge at hand-off, nested positions included. This extends the parent doc's frozen ruling; no new principle.
- [x] Concrete annotations stay L1-open; only reads are normalised.
- [x] **Human** (**Ruled 2026-10-26 (Mark, attended): accepted as recommended.**): confirm the breaking posture (minor bump to v0.55.0, CHANGELOG "Breaking — soundness", no opt-out). Confirm that three flips are intended: (i) the `map(\d. g(d), xs)` flip (a concrete `! {IO}` callback through closed-pure `std/list.map`, accepted today only via U1), with migration to `mapE`; (ii) store-only nested hand-off (`keepAll([logIt])`); (iii) a non-generic ADT constructor with a nested-arrow payload (`Reg([logIt])`) charging at construction, as `Box(logIt)` already does. Recommendation: accept all three.

## Solution Design

### Overview

Add one helper and four call sites. No syntax, no new diagnostics, no new error codes (the change only produces the existing `Missing effects` and type-mismatch messages).

```go
// internal/types/typechecker_latent_data.go (new, ~90 LOC)
// dataArrows returns the arrows reachable from t through DATA positions only:
// TList/TArray element, TMap key+value, TTuple elements, TApp args,
// TRecord/TRecordOpen fields, TRecord2 row labels. Alias heads are expanded
// at each level (effectAliasHead). It never descends into a TFunc2's params or
// return. Cycle-safe (visited set + depth cap, per internal/types/traverse's rule).
func (tc *CoreTypeChecker) dataArrows(t Type) []*TFunc2

// latentRow is the cost of invoking fn: its row, with a ConcreteEffectContract's
// L1-open compatibility tail closed (the rule applicationCallRow already applies).
func latentRow(fn *TFunc2) *Row

// nestedLatentRow unions latentRow over dataArrows(t), using
// CheckedUnionEffectRows. Distinct unresolved tails raise the existing #616
// explicit-conflict invariant; there is no silent choice.
func (tc *CoreTypeChecker) nestedLatentRow(t Type) *Row
```

`internal/types/traverse` cannot be reused from `internal/types`: it imports `internal/types` (`traverse.go:12`), so using it would create an import cycle. Its `children` also descends into arrow params and returns, which is wrong for data positions. The new walker follows the same cycle-safety rule.

### The four sites

1. **Mask (a)** — `resolvedCallbackMask` (`typechecker_latent_mask.go:74-85`): `mask[i]` is also true when `dataArrows(param)` contains an arrow with an open, non-concrete row. `callbackMask` stays the unit-tested top-level primitive.
2. **Callback rows (b)** — `publishApplicationWithCallee` (`typechecker_app_effects.go:202-208`) and the re-zonk in `substituteApplicationRows` (`:79-89`). For a top-level arrow, publish `latentRow(fn)`. Otherwise publish `nestedLatentRow(argType)`, recomputed from `callbackTypes[i]` after substitution, because nested rows are usually unresolved at publish time. `ImplicitCallbacks[i]` keeps today's meaning (the tail is implicit-owned by the caller's own binders).
3. **Implicit ownership (c)** — `implicitRowsAbove` (`:218-228`): a tail is implicit-owned if it belongs to an `ImplicitEffectContract` arrow anywhere in `dataArrows(bindingType)`, not only when the binding *is* that arrow. This makes R10/R11 erase exactly like R1. The soundness side condition is that whatever is erased here is charged by (1), and both use the same walker.
4. **App constraint row (d)** — `inferApp` (`typechecker_functions.go:656`): for a `ConcreteEffectContract` callee, contribute `callRow` (tail closed) instead of the unified `effectRow`. The typed App row, the enclosing lambda's latent row and any combinator's `e` then agree with the published call row.

Site (e), the validator, needs **no** change: it already unions `CallbackRows[i]` for every masked argument (`validate_effects.go:397-411`).

### Alternatives considered

| Id | Alternative | Measured / reasoned outcome | Verdict |
|---|---|---|---|
| A | **Charge at construction:** referencing a function value inside a list, tuple or record literal charges its row, as a lambda literal's body already is | Simple and sound, but it violates the frozen "charge at application or hand-off, never at reference" ruling. It would reject motoko's pure hook builders (`{ act: logS }` in a pure function, `c/ok_storage` style) and `length([logIt, logIt])` | Rejected |
| **B** | **Type-directed nested hand-off (this design):** extend the mask/callback rows to data positions, and normalise concrete tails at every read | Prototype: all of R1–R11 and S1–S6 correct; 0 flips over 501 + 45 + 133 files; mcp_oauth fixed (V20–V24) | **Chosen** |
| C | **Trust the published call row for same-module callees with an empty declared row** (one line at `validate_effects.go:387`) | Prototype: fixes R1–R4, R6, R7 and charges only when the callee actually invokes (`nested_store` stays accepted). But it **misses R5** (record alias, the tail never enters the callee's scheme) and does not touch R10/R11 or #1826. It also weakens the contamination-safe declared authority (`71b610d68`, `TestValidateEffects_PureFunctionNotContaminatedByContext`) | Rejected as primary; note as a possible later precision refinement |
| D | **L1-closed** (annotations closed, open on use) to remove the `ε_annot` tail entirely | It would dissolve RC2. But it is the human-gated Phase 3 of the parent doc, it changes inference of every function reference, and on its own it fixes nothing in #1718 | Out of scope |
| E | **Validator-side patch for #1826:** drop tails named `ε_annot*` from `CallRow` in `validate_effects.go` | Keys on a printed name (forbidden by #616's "do not equate tails by printed spelling"). It leaves the inferred row wrong for downstream consumers (iface, ceiling) and does not touch #1718 | Rejected |

### Implementation Plan

**Phase 1 — walker and concrete-tail normalisation, #1826 (~1 day)**
- [x] `typechecker_latent_data.go`: `dataArrows`, `latentRow`, `nestedLatentRow`, with unit tests: nesting depth, alias expansion, cycle (recursive alias), arrow params *not* traversed.
- [x] Site (d) at `typechecker_functions.go:656` and site (b)'s top-level branch use `latentRow`.
- [x] Regression tests S1, S2, S3, S4, plus `mcp_oauth` `revokeFamily` reduced to a fixture. Mutation check: revert (d) → S1 is back to the invariant and S2 to `Unresolved effect tail`.

**Phase 2 — nested hand-off, #1718 (~1 day)**
- [x] Site (a) in `resolvedCallbackMask`; site (b) nested branch plus the re-zonk in `substituteApplicationRows`.
- [x] Site (c) structural `implicitRowsAbove`.
- [x] Regression tests R1–R11 (reject or accept as specified in the Acceptance Criteria), plus every negative control. Mutation checks: revert (a) → R1–R9 accepted again; revert (c) → R10/R11 back to `Unresolved effect tail`.

**Phase 3 — invariant hardening (~0.5 day)**
- [x] Add a test-only assertion that every tail erased as implicit inside a callee body (`ImplicitOwnedTail`/`ImplicitCallbacks`) is reachable through `dataArrows` of some parameter type of that callee. This pins RC1's soundness condition so the next shape cannot reopen it. Agent may choose whether this is a debug-build invariant or a corpus test.

**Phase 4 — corpus, docs, examples (~0.5–1 day)**
- [x] Base-vs-fixed sweeps exactly as in "Risk of breaking". Record per-file status and diagnostics in the validation doc.
- [x] Remove the #1718 rows from `docs/LIMITATIONS.md:26` and `docs/docs/reference/limitations.md:69-78`. Add the `map` → `mapE` migration note.
- [x] Changelog fragment `changelogs/unreleased/<date>-effect-latent-nested-callbacks.md` (`### Fixed — …`, "Breaking — soundness").
- [x] `examples/runnable/effect_latent_nested_callbacks.ail` (accept arms only: the declared-`! {IO}` callers of R1/R2/R10, plus the S2 `mapE` form) with a manifest entry. Reject arms live in Go tests.
- [x] Teaching prompt: extend the existing sentence ("Passing a function to a higher-order function … requires the caller to declare the callback's effects") with "including a function stored in a list, tuple, Option or record argument". Verify the snippet with `ailang check`.

### Files to Modify/Create

- `internal/types/typechecker_latent_data.go` — new: walker, `latentRow`, `nestedLatentRow` (~90 LOC)
- `internal/types/typechecker_latent_data_test.go` — new: walker unit tests (~120 LOC)
- `internal/types/typechecker_latent_mask.go` — nested mask in `resolvedCallbackMask` (~+12 LOC; 85 → ~97)
- `internal/types/typechecker_app_effects.go` — callback rows (b), re-zonk, structural `implicitRowsAbove` (~+25 LOC; 374 → ~400)
- `internal/types/typechecker_functions.go` — site (d) (~+5 LOC; 756 → ~761, under the 800 cap). Do **not** touch `typechecker_core.go` (exactly 800 lines)
- `internal/pipeline/effect_latent_nested_test.go` — new: R1–R11, S1–S4, controls, both engines via the existing `TestLatentRunRejectsBeforeExecution` pattern (~250 LOC)
- `internal/pipeline/effect_latent_invariant_test.go` — Phase 3 erase⊆charge assertion (~+40 LOC)
- `docs/LIMITATIONS.md` — drop the #1718 row
- `docs/docs/reference/limitations.md` — drop the "Effectful Callbacks Nested in Data Arguments" section (`:69-78`)
- `examples/runnable/effect_latent_nested_callbacks.ail` — accept-arm example plus manifest entry
- `changelogs/unreleased/<date>-effect-latent-nested-callbacks.md` — fragment

## Conflict Surface

### Semantic positions touched

| Position | What else lives there | Why it stays correct | Evidence |
|---|---|---|---|
| Callee param mask (a) | Top-level arrow params (#1326 L2); concrete-annotated params (excluded by `!ConcreteEffectContract`); row-polymorphic `! {e}` params (charged through the call row) | Top-level branch unchanged; nested arrows reuse the same openness predicate | `control_direct`, `neg_concrete_nested`, `neg_rowpoly`, `ok_rowpoly` unchanged (V25) |
| Per-argument `CallbackRows` (b) | Lambdas (body-charged), imported names, aliases (`TestLatentStorageAlias`, `TestLatentImportedFunctionAlias`) | Top-level branch now closes a concrete tail, which only removes `ε_annot` noise; nested branch is new data | `go test ./internal/pipeline/...` green on the prototype (V22) |
| Implicit ownership (c) | Curried callback binders (`resolveOwnedTypes` comment `:313`), `TVar2` pattern binders | Strictly more tails recognised; each new one is also charged by (a), same walker | R10/R11 accept; `fwd_two` / `fwd_two_neg` unchanged (V25) |
| App constraint row (d) | #386 local solve and distinct-tail assertion (`assertSingleTailAfterAppSolve`); recursive multi-arm stdlib (std/sem); #1091 closure | Only `ConcreteEffectContract` callees change, and only by dropping their compatibility tail (identical to their published `callRow`). No row variable that generalisation depends on is touched | `go test ./internal/types/... ./internal/elaborate/...` green; std sweep 0 flips (V21–V22) |
| Package effect ceiling (`internal/pipeline/effect_ceiling.go`) | Reads the inferred row; `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` | The ceiling sees fewer phantom tails, never more labels | green on the prototype (V22) |
| PR #1790 `derivePurity` (`internal/iface/purity.go`, new in that PR) | Treats a callee's outer tail that is shared with a callback **nested in containers** (`callbackSharesTail` walks params) as latent and pure | This doc is what makes that assumption true at call sites. No shared code | See ordering |

### Programs that MUST still work (all verified on the prototype)

- `examples/runnable/effect_latent_function_values.ail`, `examples/runnable/effect_row_var_pure_caller.ail`, `examples/runnable/effect_row_var_noisy_twice.ail`, `examples/runnable/effectful_list_t1_mapE_basic.ail`, `examples/runnable/effectful_list_t4_flatMapE_expand.ail`: unchanged status in the 501-file sweep (V21).
- `std/**` (all 49+ modules in the sweep): unchanged.
- `TestLatentStorageAlias` (`internal/pipeline/effect_latent_function_values_test.go:196`), `TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback` (`effect_pure_row_overgeneralization_test.go:320`), `TestEffectCeiling_OwnTypesAndCallbacksNotCharged` (`effect_ceiling_reachable_test.go:85`), `TestValidateEffects_PureFunctionNotContaminatedByContext` (`validate_effects_test.go:125`): green (V22).
- Pure storage of effectful callbacks with no hand-off: `pure func build() -> H = { act: logS }`, `pure func mkList() -> [(int) -> int] = [logIt]`, `pure func count() -> int = length([logIt, logIt])`, `pure func pair(x: int) -> ((int) -> int, int) = (logIt, x)`: accepted (`c/ok_storage`).
- Pure callbacks in containers: `pure func ok(x: int) -> int = applyFst((inc, x)) + applyHead([inc, \n. n * 2], x)`: accepted.

### What deliberately changes

| Program shape | Base | After | Reason / migration |
|---|---|---|---|
| R1–R9 (effectful value nested in a data argument, pure or narrower caller) | accepted, effect runs | `Missing effects: IO` | The bug. Migration: declare the effect on the caller (the Suggested fix is printed) |
| Store-only nested hand-off: `func keepAll(fs: [(int) -> int]) -> [(int) -> int] = fs`; `pure func storeNested(x: int) -> int = { let _ = keepAll([logIt]); x }` | accepted | `Missing effects: IO` | Same as the existing top-level rule (`keep(logIt)` from pure is already rejected on base). Migration: declare the effect, or annotate the stored arrow concretely (`[(int) -> int ! {IO}]`), which makes it the callee's contract |
| Non-generic user ADT whose constructor payload *nests* an unannotated arrow, built in pure code: `type Reg = Reg([(int) -> int])`; `pure func mkReg() -> Reg = Reg([logIt])` | accepted | `Missing effects: IO` on `mkReg` | The constructor is a callee whose parameter nests an open arrow, so this is a hand-off. It matches base behaviour for a *top-level* payload (`Box(logIt)` from pure is already rejected). Tuple, list and record **literals** and generic constructors (`Some`, `Hold`) are not callees with arrow-typed params, so they stay storage-only. Migration: declare the effect, or annotate the payload concretely |
| R10, R11, S1–S4, mcp_oauth 0.1.0/0.1.1 | rejected (tail / invariant) | accepted | #1826 and its siblings |
| `func viaMap(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] ! {IO} = map(\d. g(d), xs)` with `std/list.map` | accepted (only via U1) | type error: `incompatible closed rows … [IO]` | `std/list.map` takes a closed-pure callback, and `map(\d. println(d), xs)` is already this error on base. Migration: `mapE`. Called out in the changelog |
| `pure func viaMapPure(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] = map(\d. g(d), xs)` | rejected (`Missing effects`) | rejected (type error) | Still rejected; message changes |

## Acceptance Criteria

Each criterion names its exact program. "Rejected" means `ailang check` exits non-zero with `Effect checking failed for function 'sneaky'` and `Missing effects: IO`, and `ailang run --caps IO --entry main` (tree-walk **and** `--bytecode`) fails before printing `side effect`. All runs use `AILANG_NO_CACHE=1`.

- [x] **AC1 — #1718 shapes rejected:** R1 (tuple), R2 (list), R3 (unannotated `Option` param), R4 (record param), R5 (record alias, cross-module `x/librec` + `x/mainrec`), R6 (let-bound tuple), R7 (returned list), R8 (`Reg([logIt])`), R9 (list of tuples).
- [x] **AC2 — annotated generic containers accepted at definition, charged at use:** R10 `applyOpt(o: Option[(int) -> int], …)` and R11 `applyHold(h: Holder[(int) -> int], …)` check clean as definitions. `pure func sneaky(x: int) -> int = applyOpt(Some(logIt), x)` (and the `Hold(logIt)` form) is rejected. The same callers declared `! {IO}` are accepted (`c/ok_declared`: `func ok(x: int) -> int ! {IO} = applyFst((logIt, x)) + applyHead([logIt], x) + applyOpt(Some(logIt), x)`).
- [x] **AC3 — #1826 fixed:** S1 (`viaField`) and S2 (`viaParam`) check clean. S3 (local `myMapE`) checks clean. S4 (`viaParamValue … ! {IO} = applyTo(g, x)`) checks clean, and its pure variant (`pure func viaParamValue(…) -> bool = applyTo(g, x)`) is rejected with `Missing effects: IO` and **no** `Unresolved effect tail` line. `sunholo/mcp_oauth` 0.1.0 and 0.1.1 (`check --package` on a copy of the registry cache) report "all passed". `grep -c "internal invariant"` over the whole sweep output = 0.
- [x] **AC4 — negative controls unchanged (accepted):** `ok_pure_cb` (pure callbacks in tuple/list incl. a lambda), `ok_storage` (record/list/tuple built and stored in pure code, `length([logIt, logIt])`), `ok_rowpoly` (`applyFstE[e](p: ((int) -> int ! {e}, int)) -> int ! {e}` called with `inc` from pure and with `logIt` from `! {IO}`), `fwd_two` (generic forwarding of two implicit callbacks through a tuple), `x/main_ok`, S2 with `ap`.
- [x] **AC5 — negative controls unchanged (rejected):** `control_direct` (#1326 `applyTo(logIt, x)`), `neg_rowpoly`, `neg_concrete_nested` (`applyFstIO(p: ((int) -> int ! {IO}, int)) -> int ! {IO}` from pure; blame stays on `sneaky`), `fwd_two_neg`, `leak_adt` (`Box(logIt)`), `ret_closure`, `leak_lambda_in_list`, `x/main`. Mixed rows: `func onlyIO(x: int) -> int ! {IO} = applyBoth((logIt, readIt), x)` is rejected naming **FS** (union of nested rows, not first-wins).
- [x] **AC6 — one walker:** sites (a)–(d) call the same `dataArrows`. A unit test proves that arrow params and returns are not traversed, that alias heads are expanded per level, and that a recursive alias terminates.
- [x] **AC7 — erase ⊆ charge:** the Phase 3 assertion holds over `internal/pipeline` tests and the corpus. Mutation check: reverting (a) while keeping (c) makes it fail.
- [x] **AC8 — corpus:** base-vs-fixed sweeps per "Risk of breaking" show zero unexplained flips. Every flip is listed by path in the changelog fragment with before/after and migration.
- [x] **AC9 — gates:** focused `go test ./internal/types/... ./internal/pipeline/... ./internal/elaborate/... ./internal/iface/... ./internal/eval/...`, `make test-core`, `make lint`, `make check-file-sizes`, `make check-boundaries`, `make check-changelog`, `make verify-examples` (run unpiped).
- [x] **AC10 — docs:** limitation entries removed; example plus manifest added; teaching-prompt sentence verified with `ailang check`.

### Mutation checks (revert the fix, watch the test fail)

| Component reverted | Must turn red | Must stay green |
|---|---|---|
| (a) nested mask | AC1 | AC3, AC4 |
| (b) nested callback rows (mask kept) | AC1 (with a fail-loud missing-callback-row invariant, not a silent pass) | AC4 |
| (c) structural ownership | AC2 (R10/R11 definitions) | AC1 |
| (d) closed concrete App row | AC3 (S1 invariant, S2/S3 tail) | AC1 |
| (b) `latentRow` on top-level branch | AC3 (S4 extra `Unresolved effect tail` line) | AC1 |

## Risk of breaking existing stdlib / packages — and how to measure it

**Measured on the prototype** (scratch copy of `772703831` with sites (a)–(d) as specified; nothing committed). The comparison binary was built from the same tree.

| Corpus | Command (from each root, `AILANG_NO_CACHE=1`) | Base | Prototype | Flips |
|---|---|---|---|---|
| `examples/**/*.ail` + `std/**/*.ail` (501 files) | `ailang check -timeout 90s <file>`, 8-way parallel | 423 pass / 78 fail | 423 / 78 | **0** |
| `ailang-packages/packages/*` (45 roots, local checkout `e998928`) | `ailang check --package` | 44 pass / 1 fail (invalid manifest, `motoko-ext-typefix-agent`) | 44 / 1 | **0** |
| `ailang-parse` (133 files) | `ailang check --package` | all passed | all passed | **0** |
| `docparse` (39 files) | `ailang check --package` | 15 / 24 | 15 / 24 | **0**. 2 files move from the mcp_oauth invariant to a pre-existing unrelated `STRICT_FALLBACK` in `sunholo/firestore` 0.7.2 |
| `ailang-motoko`, `ailang-world` package roots (13 + 17) | `ailang check --package` | per-root results | identical | **0** |
| `sunholo/mcp_oauth` 0.1.0 / 0.1.1 / 0.1.2 | `check --package` on copies | ✗ / ✗ / ✓ | ✓ / ✓ / ✓ | 2 intended fixes |
| Go suites | `go test ./internal/types/... ./internal/elaborate/... ./internal/pipeline/... ./internal/iface/... ./internal/eval/...` | — | all `ok` | — |

**Residual risks**

| Risk | Impact | Mitigation |
|---|---|---|
| Store-only nested hand-off flips in code outside the measured corpora (e.g. a pure function passing `[handler]` to a registry builder) | Med | This matches the existing top-level rule, the diagnostic prints the right fix, and a concrete annotation moves the contract to the callee. List it in the changelog |
| `map(\d. g(d), xs)` with a concrete-annotated `g` flips to a type error | Low–Med | Correct under `std/list.map`'s closed callback (same as the `println` lambda today); migration is `mapE`. Human freeze item |
| `nestedLatentRow` unions two distinct unresolved tails (two independent generic callbacks in one container, forwarded by a generic caller) | Med | Use `CheckedUnionEffectRows`. A conflict is an explicit invariant (#616), never first-wins. `fwd_two` exercises the implicit-tail case; add a declared `! {e1}`/`! {e2}` two-row case to the tests |
| A type-decl alias arrow (`type R = {f: (int) -> int}`) has one `ε_annotK` shared by every use | Med | The prototype is correct for R5. `c/alias_two_callers` (a pure caller with `inc` and an `! {IO}` caller with `logIt` of the same `applyRec(p: R)` in one module) checks clean on base and prototype. Keep it as a regression test and add a cross-module variant |
| Content cache hides a regression in sweeps | Med | Every arm uses `AILANG_NO_CACHE=1` / fresh copies |
| External package corpora are local checkouts, not registry HEAD | Low | Re-run the package sweep against current registry versions in the executor (`ailang` registry cache / fresh clones) and record SHAs |

**How the executor re-measures it:** build base (the merge-base, recorded by SHA) and fixed binaries from the same tree. Then run the five corpus rows above with identical commands, and diff per-file `status + first diagnostic`. Also run `make verify-examples` and `tools/verify-stdlib.sh`. Any flip not in "What deliberately changes" blocks completion.

## Ordering against PR #1790 (M-PURE-ROW-AND-IFACE-PURITY, #1443)

- **No code overlap.** #1790 (head `8236413d3`, currently `CONFLICTING`) changes `internal/pipeline/validate_effects.go` only in the declared-row collection loop (base `:121-129`, a `pure` + declared-row contradiction check), plus `internal/iface/*`, `internal/smt/*` and goldens. This doc touches `validate_effects.go` not at all, and `internal/types` App/latent files that #1790 does not touch (`gh pr list` shows no other open PR on these files).
- **Semantic dependency, one way:** #1790's `derivePurity` marks a function `pure` when its outer tail is shared with a callback **nested in its parameters** (`callbackSharesTail` walks containers). It assumes the caller is charged for nested callbacks, which is the #1718 invariant. Until this doc lands, the iface will publish `pure: true` for callers like R1's `sneaky` that really do perform IO.
- **Order:** land #1790 first (it is further along; the rebase is trivial since the hunks are disjoint). Then this doc in the **same release**, so no release advertises iface purity without nested charging. If #1790 slips, this doc does not wait for it.

## Testing Strategy

**Unit tests (`internal/types`):** the walker (positions, alias, cycle, no arrow descent); `latentRow` on concrete, implicit and row-polymorphic arrows; the nested mask on `resolvedCallbackMask` for each container type.

**Integration tests (`internal/pipeline`):** AC1–AC5 as `checkModules` fixtures with `assertLatentIO`-style blame checks (function name plus label), using the cross-module harness from `TestLatentCrossModule`. Also the two-engine run-rejects-before-execution harness from `TestLatentRunRejectsBeforeExecution`.

**Corpus:** the sweeps above, recorded in a validation doc next to the sprint plan.

## Deferred Decisions

- Exact representation of nested callback rows in `ApplicationEffects` (one union row per argument vs a per-path list). Agent may choose; one union row per argument is what the validator consumes today.
- Whether the Phase 3 erase⊆charge check is a debug-build invariant or a test-only sweep. Agent may choose.
- Whether `TCon` parameters of non-generic ADTs should also be traversed via constructor field types (belt-and-braces; not needed for soundness, see Non-Goals). Agent may add it if cheap and flip-free.
- Whether alternative C's "charge only if invoked" precision is worth a follow-up once U1 and Phase 3 settle. Human, later.

## Verification Log

| # | Claim | How verified (base `772703831` unless stated) | Result |
|---|---|---|---|
| V1 | Issue texts | `gh issue view 1718/1826/573/1326/1443 --json …` | #1718 and #1826 open; #573 and #1326 closed |
| V2 | R1, R2 leak | `ailang check` ✓; `run --caps IO` and `run --bytecode` print `side effect` | Confirmed |
| V3 | R3–R9 leak | same | Confirmed (R5 cross-module, both engines) |
| V4 | R10, R11 falsely rejected | `check` → `Unresolved effect tail: ε_annot1` on the callee | Confirmed |
| V5 | S1, S2, S3, S4 | `check` → invariant / tail as tabulated | Confirmed |
| V6 | mcp_oauth 0.1.0 and 0.1.1 fail, 0.1.2 passes | `check --package` on copies of `~/.ailang/cache/registry/sunholo/mcp_oauth/<v>` | Confirmed (`flow.ail:193/195`) |
| V7 | Controls already correct on base | `control_direct`, `leak_adt`, `leak_lambda_in_list`, `x/main`, `ret_closure`, `viaparam_hof` | Confirmed |
| V8 | The checker computes `{IO}` for R1's call but the validator ignores it | `DEBUG_EFFECTS=1` transcript (RC1) | Confirmed |
| V9 | `callbackMask` is top-level only | read `typechecker_latent_mask.go:15-27` (`:22` type assertion) | Confirmed |
| V10 | `CallbackRows` head-only; concrete tail cloned | read `typechecker_app_effects.go:79-89, 202-208` | Confirmed |
| V11 | `implicitRowsAbove` top-level binders only | read `:218-228` | Confirmed |
| V12 | Top-level store-only hand-off is already charged | `c/top_store`: `pure func storeTop(x: int) -> int = { let _ = keep(logIt); x }` → `Missing effects: IO` | Confirmed |
| V13 | App row uses the unified row, published row is closed | read `typechecker_functions.go:580, 607, 656`; `applicationCallRow` `:287-304` | Confirmed |
| V14 | Concrete annotation keeps an `ε_annot` tail | read `elaborate/file_funcs.go:398-425` | Confirmed (`:414-417`) |
| V15 | Invariant site | read `validate_effects_latent.go:51-60` | Confirmed (`:56`) |
| V16 | Same-module declared authority vs imported call row | read `validate_effects.go:358-392`; `x/main` rejected while R1 is accepted | Confirmed |
| V17 | U1 row unifier drops labels | scratch unit probe: `UnifyRows({IO\|ε}, {})` → `ε := {}`, no error; record `{x\|r} ~ {y}` → `r := {y}` | Confirmed |
| V18 | U1 observable leak | `pure func viaMapPure(g: (string) -> bool ! {IO}, …) = map(g, xs)` checks; run prints `side effect` | Confirmed |
| V19 | Naive U1 fix blast radius | scratch prototype rejecting open-side extra labels: 17/501 flips (`std/ai.ail:303` `_ai_step_with_stream` plus 16 dependants) | Measured; out of scope |
| V20 | Prototype B fixes everything tabulated | `matrix.sh ailang ailang-p2` over R1–R11, S1–S4, controls | Confirmed |
| V21 | 0 flips over 501 examples + std | `sweep.sh` base vs prototype, `diff` of status | 423/78 both; identical |
| V22 | Go suites green on prototype | `go test` over types, elaborate, pipeline, iface, eval | all `ok` |
| V23 | 0 flips over packages, ailang-parse, docparse, motoko, world | `check --package` per root, diff | identical (docparse: 2 first-errors move from the mcp_oauth invariant to a pre-existing unrelated error) |
| V24 | mcp_oauth fixed by prototype | `check --package` 0.1.0 / 0.1.1 → all passed | Confirmed |
| V25 | Negative controls on prototype | `ok_declared`, `ok_pure_cb`, `ok_storage`, `ok_rowpoly`, `neg_rowpoly`, `neg_concrete_nested`, `fwd_two`, `fwd_two_neg`, `mixed_rows` (rejects naming FS), `alias_two_callers` (accepted), `nested_store` and `reg_construct` (intended flips) | As tabulated |
| V26 | Alternative C misses R5 and #1826 | scratch prototype at `validate_effects.go:387` | R5 still accepted; R10/S2 unchanged |
| V27 | `traverse` cannot be imported from `internal/types` | `internal/types/traverse/traverse.go:12` imports `internal/types` | Confirmed (cycle) |
| V28 | No new error code needed | design emits only existing `Missing effects` and type-mismatch messages | n/a (nothing to allocate) |
| V29 | #1790 overlap | `gh pr diff 1790`: `validate_effects.go` hunk only in the declared-row loop; `internal/iface/purity.go` `callbackSharesTail` walks nested params | Confirmed; `gh pr list` shows no other open PR on the touched files |
| V30 | Cited tests and examples exist | `grep -n "^func Test…"`; `ls examples/runnable/…` | Confirmed (paths as cited) |

## Related Documents

- [M-EFFECT-LATENT-FUNCTION-VALUES](../../planned/v0_48_0/m-effect-latent-function-values.md): parent. Defines L1-open and L2; its Phase 3 (upper bounds) stays deferred and separate.
- [M-EFFECT-ROW-VAR-UNIFICATION](../../implemented/v1_0_0/m-effect-row-var-unification.md): per-App `ApplicationEffects` record; explicitly excluded #1718 (line 648).
- `design_docs/planned/v0_53_0/m-pure-row-and-iface-purity-sprint-plan.md` (on PR #1790's branch): iface purity that assumes nested callbacks are caller-charged.
- [stored-lambda-effects-charge-enclosing-fn](../../planned/ailang-core-triage/stored-lambda-effects-charge-enclosing-fn.md): the opposite over-approximation (lambda literal bodies charged at construction). Unchanged here; noted because Alternative A would generalise it.
- [M-EFFECT-HANDLERS](../../planned/v1_1_0/m-effect-handlers.md) (0.43 similarity): unrelated mechanism, no overlap.

## References

- [Design Axioms](/docs/references/axioms)
- Issues: [#1718](https://github.com/sunholo-data/ailang/issues/1718), [#1826](https://github.com/sunholo-data/ailang/issues/1826), [#573](https://github.com/sunholo-data/ailang/issues/573), [#1326](https://github.com/sunholo-data/ailang/issues/1326), [#1443](https://github.com/sunholo-data/ailang/issues/1443), [#616](https://github.com/sunholo-data/ailang/issues/616)
- PRs: [#1708](https://github.com/sunholo-data/ailang/pull/1708) (`3d4f49720`), [#1790](https://github.com/sunholo-data/ailang/pull/1790)
- Commit `accdfc4bf` (shared effect rows at App constraints, where #1826 starts)

## Future Work

- File **U1** (row unifier accepts an open row's labels against a closed row) as its own P0 issue with V17–V19 attached. It needs a std/ai signature decision before the label check can land.
- Parent doc Phase 3 (upper-bound annotations) would remove the `ε_annot` compatibility tail entirely. If it ships, `latentRow`'s tail-closing becomes a no-op but stays correct.

---

**Document created**: 2026-10-26
**Last updated**: 2026-10-29


## Implementation Report — 2026-10-29

Merged in [PR #1963](https://github.com/sunholo-data/ailang/pull/1963) at `87f846980c8b6fc601173be427b1364968559488`. This is implemented on dev and remains unreleased; the folder denotes the next-minor target, not a published tag.

The shared data-position walker now accounts for nested callback effects at all four read sites. It traverses finite deep structures, handles aliases and cycles, and turns only its defensive traversal-limit marker into an ordinary type error. The repair closes the demonstrated depth-17 erase-without-charge gap. Arrow arguments and results remain outside the approved traversal.

Independent evaluation accepted source `86d3a5f8e946e12967c38b446e5fb854a884a42e` at 93/100, with regression coverage 10/10 reported separately. The score preserves the original five-point archival deduction; this report does not rescore itself. See [evaluation](../../../.ailang/state/evaluations/eval_M-EFFECT-LATENT-NESTED-CALLBACKS_round_1.json) and [review report](m-effect-latent-nested-callbacks-evaluation.md).

The full native test suite passed with serial package scheduling after four unchanged-package startup timeouts; initial failures and isolated successful rechecks are preserved in the review report. Exact-head remote Linux/Windows tests, lint, structural/example checks, CodeQL, SonarCloud and launchd passed. Two hosted macOS build jobs were still queued at acceptance; no all-checks-success claim is made. Corpus comparisons found zero unexplained changes, and both engines reject effectful pure callers before performing output. Teaching prompt v0.16.9 and mirrors carry the new rule while prior prompt bytes remain frozen.


### Subsequent full-consumer compatibility evidence

The actual combined #1958/#1963 Motoko core gate later found five failures through a mixed pure/effectful callback registration fixture (55/60, compared with60/60 before nested charging). Independent four-arm review classifies this as the approved parameter-wide conservative hand-off rule, rather than a deviation from AC4's literal/all-concrete storage controls. Existing conversion also collapses written empty callback rows with omitted rows; the attempted four-field `! {}` migration therefore failed unchanged. See [independent follow-up](m-effect-latent-nested-callbacks-consumer-followup.md).

Mark approved option A of the focused explicit-empty callback design, then explicitly authorized its four-milestone sprint. That amendment is a new semantic decision: closed/concrete written empty callback contracts, preserved omitted-arrow behavior, import/cache/engine checks, and conditional four-field consumer delivery only after actual60-core/9-extension proof. That amendment is now independently accepted (95/100) and merged with #1958 in PR #2005 at `80793dbcf42423027be4bab1a0088da54e2323d8`. The exact four-field pinned-fork migration `083ef43726981099a0bad4fedda18fc72105d6fa` restores actual 60/60 core and 9/9 extension checks; fresh no-cache initial deadlines and passing serial retries remain recorded. See [accepted amendment](m-effect-explicit-empty-callback-contract.md) and [consumer follow-up](m-effect-latent-nested-callbacks-consumer-followup.md). The separate alias-ownership invariant follow-up was subsequently accepted at95/100 and merged in [PR2008](https://github.com/sunholo-data/ailang/pull/2008). Its [validation](m-effect-latent-alias-ownership-validation.md) and [independent evaluation](m-effect-latent-alias-ownership-evaluation.md) preserve the original PASS93 and strict-FS qualifications.
